package hxcmp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

// defaultRegistry is the global registry used by MustGet.
// Set via SetDefault during application initialization.
var defaultRegistry *Registry

// SetDefault sets the default registry for type-based component lookup.
// Call this once during application startup after creating your registry.
//
// Example:
//
//	reg := hxcmp.NewRegistry(key)
//	hxcmp.SetDefault(reg)
//	reg.Add(NewSidebar(store), NewTodoList(store))
func SetDefault(reg *Registry) {
	defaultRegistry = reg
}

// MustGet retrieves a component by type from the default registry.
// Panics if the component is not registered or SetDefault was not called.
//
// This is typically used by generated Cmp() getter functions:
//
//	func SidebarCmp() *Sidebar { return hxcmp.MustGet[*Sidebar]() }
//
// For manual usage in application code:
//
//	sidebar := hxcmp.MustGet[*Sidebar]()
//	sidebar.Render(ctx, props)
func MustGet[T any]() T {
	if defaultRegistry == nil {
		var zero T
		panic(fmt.Sprintf("hxcmp: no default registry set (call hxcmp.SetDefault first), looking for %T", zero))
	}
	return Get[T](defaultRegistry)
}

// Get retrieves a component by type from the given registry.
// Panics if the component is not registered.
//
// Example:
//
//	sidebar := hxcmp.Get[*Sidebar](reg)
func Get[T any](reg *Registry) T {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	for _, comp := range reg.components {
		if c, ok := comp.(T); ok {
			return c
		}
	}

	var zero T
	panic(fmt.Sprintf("hxcmp: component %T not registered", zero))
}

// Provide registers a component and returns it for optional chaining.
// This is a convenience wrapper around reg.Add that returns the component.
//
// Example:
//
//	sidebar := hxcmp.Provide(reg, NewSidebar(store))
//	// sidebar is now registered and can be used immediately
func Provide[T any](reg *Registry, comp T) T {
	reg.Add(comp)
	return comp
}

// Registry manages component registration and routing.
//
// The registry provides centralized component management with:
//   - Prefix collision detection at registration time (not runtime)
//   - CSRF protection via HX-Request header validation
//   - Shared encryption key for all components
//   - Customizable error handling via OnError callback
//
// Example:
//
//	reg := hxcmp.NewRegistry(encryptionKey)
//	reg.OnError = customErrorHandler
//	reg.Add(fileViewer, fileBrowser, commitList)
//	http.Handle("/_c/", reg.Handler())
//
// Components must implement Hydrater[P] and Renderer[P] interfaces.
// The registry verifies interfaces at registration time, panicking if
// requirements aren't met. This ensures errors are caught during startup,
// not during requests.
type Registry struct {
	mu         sync.RWMutex
	mux        *http.ServeMux
	encoder    *Encoder
	components map[string]any // map[prefix]component

	// OnError is called when a component returns an error or encounters
	// hydration/decryption failures.
	//
	// Customize this to handle errors appropriately for your application:
	//
	//	reg.OnError = func(w http.ResponseWriter, r *http.Request, err error) {
	//	    log.Printf("Component error: %v", err)
	//	    if hxcmp.IsNotFound(err) {
	//	        http.Error(w, "Not found", http.StatusNotFound)
	//	        return
	//	    }
	//	    http.Error(w, "Internal error", http.StatusInternalServerError)
	//	}
	//
	// The default handler returns 404 for IsNotFound, 400 for IsDecryptionError,
	// and 500 for all other errors.
	OnError func(http.ResponseWriter, *http.Request, error)
}

// NewRegistry creates a new component registry with the given encryption key.
//
// The encryption key is used for signing (all components) and encryption
// (components marked .Sensitive()). It should be at least 32 bytes of
// cryptographically random data.
//
// Panics if the encoder cannot be created (invalid key length).
func NewRegistry(encryptionKey []byte) *Registry {
	enc, err := NewEncoder(encryptionKey)
	if err != nil {
		panic(fmt.Sprintf("hxcmp: failed to create encoder: %v", err))
	}

	reg := &Registry{
		mux:        http.NewServeMux(),
		encoder:    enc,
		components: make(map[string]any),
	}

	// Default error handler - categorizes by error type
	reg.OnError = func(w http.ResponseWriter, r *http.Request, err error) {
		if IsNotFound(err) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if IsDecryptionError(err) {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}

	return reg
}

// Encoder returns the registry's encoder (used by components).
func (reg *Registry) Encoder() *Encoder {
	return reg.encoder
}

// Add registers components with the registry.
//
// Components must embed *hxcmp.Component[P] and implement Hydrater and Renderer.
// Panics if a component doesn't meet requirements or has a prefix collision.
//
// Prefix collisions indicate two component instances with the same name were
// created at the same source location, which shouldn't happen in normal use.
// If you need multiple instances of the same component type, create them in
// different locations or use different names.
//
// Example:
//
//	fileViewer := fileviewer.New(repo)
//	fileBrowser := filebrowser.New(repo)
//	reg.Add(fileViewer, fileBrowser)
func (reg *Registry) Add(components ...any) {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	for _, comp := range components {
		reg.registerComponent(comp)
	}
}

// registerComponent registers a single component by detecting whether it has
// generated code (HXComponent interface) or requires reflection fallback.
func (reg *Registry) registerComponent(comp any) {
	// Check if component implements HXComponent (generated code)
	if hxc, ok := comp.(HXComponent); ok {
		prefix := hxc.HXPrefix()
		if _, exists := reg.components[prefix]; exists {
			panic(fmt.Sprintf("hxcmp: prefix collision for %q", prefix))
		}
		reg.components[prefix] = comp

		// Set the encoder on the embedded Component via reflection.
		// Generated code accesses the encoder via c.Component.Encoder().
		reg.setEncoderOnComponent(comp)

		// Register the route pattern
		pattern := prefix + "/"
		reg.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			hxc.HXServeHTTP(w, r)
		})
		return
	}

	// Fallback: use reflection to find embedded Component and register manually.
	// This path is for components without generated code (shouldn't be used in production).
	reg.registerComponentReflection(comp)
}

// setEncoderOnComponent sets the encoder and error handler on a component's embedded Component field.
// This is necessary because generated code accesses the encoder via c.Component.Encoder()
// and the error handler via c.Component.OnError().
func (reg *Registry) setEncoderOnComponent(comp any) {
	val := reflect.ValueOf(comp)
	if val.Kind() != reflect.Ptr {
		return
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return
	}

	// Find embedded *Component[P] field
	compField, found := reg.findEmbeddedComponent(val)
	if !found {
		return
	}

	// Call SetEncoder on the embedded Component
	setEncoderMethod := compField.MethodByName("SetEncoder")
	if setEncoderMethod.IsValid() {
		setEncoderMethod.Call([]reflect.Value{reflect.ValueOf(reg.encoder)})
	}

	// Call SetOnError on the embedded Component to enable centralized error handling
	setOnErrorMethod := compField.MethodByName("SetOnError")
	if setOnErrorMethod.IsValid() {
		setOnErrorMethod.Call([]reflect.Value{reflect.ValueOf(reg.OnError)})
	}
}

// registerComponentReflection uses reflection to register a component without
// generated code. This fallback path enables rapid prototyping before running
// 'hxcmp generate', but has important limitations:
//
//   - No compile-time verification of action names or handler signatures
//   - Slower per-request dispatch due to reflection overhead
//   - Panics deferred to request time instead of registration time
//
// Production code should always use generated code for type safety and performance.
// If this path is taken, it means the component doesn't implement HXComponent
// (i.e., *_hx.go file is missing or out of date).
func (reg *Registry) registerComponentReflection(comp any) {
	val := reflect.ValueOf(comp)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		panic("hxcmp: component must be a pointer to a struct")
	}

	// Find embedded Component[P]
	compField, found := reg.findEmbeddedComponent(val.Elem())
	if !found {
		panic(fmt.Sprintf("hxcmp: %T does not embed *hxcmp.Component[P]", comp))
	}

	// Get prefix
	prefix := compField.MethodByName("Prefix").Call(nil)[0].String()
	if _, exists := reg.components[prefix]; exists {
		panic(fmt.Sprintf("hxcmp: prefix collision for %q", prefix))
	}

	// Set the encoder on the embedded component
	setEncoderMethod := compField.MethodByName("SetEncoder")
	if setEncoderMethod.IsValid() {
		setEncoderMethod.Call([]reflect.Value{reflect.ValueOf(reg.encoder)})
	}

	// Set the parent reference
	setParentMethod := compField.MethodByName("SetParent")
	if setParentMethod.IsValid() {
		setParentMethod.Call([]reflect.Value{val})
	}

	reg.components[prefix] = comp

	// Register a catch-all route for this component
	pattern := prefix + "/"
	reg.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		reg.handleRequest(comp, compField, w, r)
	})
}

// findEmbeddedComponent finds the embedded *Component[P] field via reflection.
func (reg *Registry) findEmbeddedComponent(val reflect.Value) (reflect.Value, bool) {
	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.Anonymous {
			continue
		}

		fieldVal := val.Field(i)
		fieldType := field.Type

		// Check if it's a pointer to Component[P]
		// Generic types have names like "Component[pkg.PropsType]"
		if fieldType.Kind() == reflect.Ptr {
			elemType := fieldType.Elem()
			typeName := elemType.Name()
			if typeName == "Component" || strings.HasPrefix(typeName, "Component[") {
				return fieldVal, true
			}
		}
	}
	return reflect.Value{}, false
}

// handleRequest handles a component request using reflection.
//
// This is the fallback path when generated code is not available. It provides
// basic functionality for prototyping without running the code generator.
// In production, generated code handles all of this more efficiently.
func (reg *Registry) handleRequest(comp any, compField reflect.Value, w http.ResponseWriter, r *http.Request) {
	// Get the prefix to determine the action path
	prefix := compField.MethodByName("Prefix").Call(nil)[0].String()
	path := strings.TrimPrefix(r.URL.Path, prefix)
	path = strings.TrimPrefix(path, "/")

	// Decode props if present
	encoded := r.URL.Query().Get("p")
	if encoded == "" && r.Method != http.MethodGet {
		if err := r.ParseForm(); err == nil {
			encoded = r.FormValue("p")
		}
	}

	// Create a new props value via reflection
	propsType := reg.getPropsType(compField)
	if propsType == nil {
		reg.OnError(w, r, fmt.Errorf("cannot determine props type"))
		return
	}
	propsPtr := reflect.New(propsType)

	// Decode props if encoded string present
	if encoded != "" {
		// Get the decoder method if it exists
		if decoder, ok := propsPtr.Interface().(Decodable); ok {
			if err := reg.encoder.Decode(encoded, reg.isSensitive(compField), decoder); err != nil {
				reg.OnError(w, r, WrapDecodeError(err))
				return
			}
		}
	}

	// Call Hydrate
	hydrateMethod := reflect.ValueOf(comp).MethodByName("Hydrate")
	if hydrateMethod.IsValid() {
		results := hydrateMethod.Call([]reflect.Value{
			reflect.ValueOf(r.Context()),
			propsPtr,
		})
		if len(results) > 0 && !results[0].IsNil() {
			reg.OnError(w, r, results[0].Interface().(error))
			return
		}
	}

	props := propsPtr.Elem()

	// Route based on method and path
	if r.Method == http.MethodGet && (path == "" || path == "/") {
		// GET / - render
		reg.reflectRender(comp, props, w, r)
		return
	}

	// Try to find and invoke an action handler
	actions := compField.MethodByName("Actions").Call(nil)[0]
	if actions.Kind() == reflect.Map {
		for _, key := range actions.MapKeys() {
			actionName := key.String()
			if path == actionName {
				actionDef := actions.MapIndex(key)
				if !actionDef.IsValid() {
					continue
				}

				// Check if method matches
				methodField := actionDef.Elem().FieldByName("method")
				if methodField.IsValid() {
					expectedMethod := methodField.String()
					if expectedMethod == "" {
						expectedMethod = "POST"
					}
					if r.Method != expectedMethod {
						continue
					}
				}

				// Get the handler
				handlerField := actionDef.Elem().FieldByName("handler")
				if !handlerField.IsValid() {
					continue
				}

				// Invoke the handler via reflection
				reg.reflectInvokeHandler(comp, handlerField.Interface(), props, w, r)
				return
			}
		}
	}

	http.NotFound(w, r)
}

// getPropsType extracts the props type from a Component[P] field.
func (reg *Registry) getPropsType(compField reflect.Value) reflect.Type {
	// The Component[P] has a method that uses P - we can extract it from Refresh's signature
	refreshMethod := compField.MethodByName("Refresh")
	if !refreshMethod.IsValid() {
		return nil
	}
	// Refresh takes (props P) and returns *Action
	// The first (and only) input parameter is the props type
	methodType := refreshMethod.Type()
	if methodType.NumIn() > 0 {
		return methodType.In(0)
	}
	return nil
}

// isSensitive checks if a component is marked as sensitive.
func (reg *Registry) isSensitive(compField reflect.Value) bool {
	method := compField.MethodByName("IsSensitive")
	if !method.IsValid() {
		return false
	}
	results := method.Call(nil)
	if len(results) > 0 {
		return results[0].Bool()
	}
	return false
}

// reflectRender renders a component via reflection.
func (reg *Registry) reflectRender(comp any, props reflect.Value, w http.ResponseWriter, r *http.Request) {
	renderMethod := reflect.ValueOf(comp).MethodByName("Render")
	if !renderMethod.IsValid() {
		reg.OnError(w, r, fmt.Errorf("component does not implement Render"))
		return
	}

	results := renderMethod.Call([]reflect.Value{
		reflect.ValueOf(r.Context()),
		props,
	})

	if len(results) == 0 {
		reg.OnError(w, r, fmt.Errorf("Render returned no value"))
		return
	}

	// The result should be a templ.Component
	templComp, ok := results[0].Interface().(interface {
		Render(context.Context, io.Writer) error
	})
	if !ok {
		reg.OnError(w, r, fmt.Errorf("Render did not return a templ.Component"))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templComp.Render(r.Context(), w); err != nil {
		// Already started writing, just log
		fmt.Printf("hxcmp: render error: %v\n", err)
	}
}

// reflectInvokeHandler invokes an action handler via reflection.
func (reg *Registry) reflectInvokeHandler(comp any, handler any, props reflect.Value, w http.ResponseWriter, r *http.Request) {
	handlerVal := reflect.ValueOf(handler)
	if !handlerVal.IsValid() || handlerVal.Kind() != reflect.Func {
		reg.OnError(w, r, fmt.Errorf("invalid handler"))
		return
	}

	// Determine handler signature and call appropriately
	handlerType := handlerVal.Type()
	numIn := handlerType.NumIn()

	var results []reflect.Value
	switch numIn {
	case 2:
		// func(ctx, props) Result[P]
		results = handlerVal.Call([]reflect.Value{
			reflect.ValueOf(r.Context()),
			props,
		})
	case 3:
		// func(ctx, props, request) or func(ctx, props, writer)
		thirdArgType := handlerType.In(2)
		if thirdArgType.Kind() == reflect.Ptr && thirdArgType.Elem().Name() == "Request" {
			results = handlerVal.Call([]reflect.Value{
				reflect.ValueOf(r.Context()),
				props,
				reflect.ValueOf(r),
			})
		} else {
			results = handlerVal.Call([]reflect.Value{
				reflect.ValueOf(r.Context()),
				props,
				reflect.ValueOf(w),
			})
		}
	default:
		reg.OnError(w, r, fmt.Errorf("unsupported handler signature"))
		return
	}

	if len(results) == 0 {
		return
	}

	// Process the Result[P]
	result := results[0]
	reg.processReflectResult(comp, result, props, w, r)
}

// processReflectResult handles a Result[P] value via reflection.
func (reg *Registry) processReflectResult(comp any, result reflect.Value, props reflect.Value, w http.ResponseWriter, r *http.Request) {
	// Check for error
	getErrMethod := result.MethodByName("GetErr")
	if getErrMethod.IsValid() {
		errResult := getErrMethod.Call(nil)
		if len(errResult) > 0 && !errResult[0].IsNil() {
			reg.OnError(w, r, errResult[0].Interface().(error))
			return
		}
	}

	// Check for redirect
	getRedirectMethod := result.MethodByName("GetRedirect")
	if getRedirectMethod.IsValid() {
		redirectResult := getRedirectMethod.Call(nil)
		if len(redirectResult) > 0 {
			redirect := redirectResult[0].String()
			if redirect != "" {
				w.Header().Set("HX-Redirect", redirect)
				return
			}
		}
	}

	// Check for skip
	shouldSkipMethod := result.MethodByName("ShouldSkip")
	if shouldSkipMethod.IsValid() {
		skipResult := shouldSkipMethod.Call(nil)
		if len(skipResult) > 0 && skipResult[0].Bool() {
			return
		}
	}

	// Get updated props and render
	getPropsMethod := result.MethodByName("GetProps")
	if getPropsMethod.IsValid() {
		propsResult := getPropsMethod.Call(nil)
		if len(propsResult) > 0 {
			props = propsResult[0]
		}
	}

	reg.reflectRender(comp, props, w, r)
}

// Handler returns the HTTP handler for component routes.
//
// Mount this at "/_c/" in your application:
//
//	http.Handle("/_c/", reg.Handler())
//
// The handler provides automatic CSRF protection - mutating methods
// (POST/PUT/DELETE/PATCH) require the HX-Request: true header that
// HTMX sends. Combined with SameSite cookies, this prevents cross-origin
// attacks without additional tokens.
func (reg *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CSRF protection: mutating methods require HX-Request header
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if r.Header.Get("HX-Request") != "true" {
				http.Error(w, "Forbidden: HTMX request required", http.StatusForbidden)
				return
			}
		}

		reg.mux.ServeHTTP(w, r)
	})
}
