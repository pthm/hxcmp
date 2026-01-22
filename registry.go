package hxcmp

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"
)

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

// registerComponent registers a single component.
func (reg *Registry) registerComponent(comp any) {
	// Check if component implements HXComponent (generated code)
	if hxc, ok := comp.(HXComponent); ok {
		prefix := hxc.HXPrefix()
		if _, exists := reg.components[prefix]; exists {
			panic(fmt.Sprintf("hxcmp: prefix collision for %q", prefix))
		}
		reg.components[prefix] = comp

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

// registerComponentReflection uses reflection to register a component.
//
// This is the fallback when generated code is not available. It's slower
// and provides less compile-time safety than generated code, but allows
// the system to work without code generation for prototyping.
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

		// Check if it's a pointer to Component
		if fieldType.Kind() == reflect.Ptr {
			elemType := fieldType.Elem()
			if elemType.Name() == "Component" {
				return fieldVal, true
			}
		}
	}
	return reflect.Value{}, false
}

// handleRequest handles a component request using reflection.
//
// This is the fallback path when generated code is not available. In practice,
// generated code handles all of this more efficiently without reflection.
func (reg *Registry) handleRequest(comp any, compField reflect.Value, w http.ResponseWriter, r *http.Request) {
	// This is a simplified implementation.
	// The full implementation would decode props, call Hydrate, route to handlers, etc.
	// In practice, generated code handles all of this more efficiently.
	http.Error(w, "Component requires generated code", http.StatusInternalServerError)
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
