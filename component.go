package hxcmp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"runtime"

	"github.com/a-h/templ"
)

// actionDef holds metadata about a registered action.
type actionDef struct {
	name    string
	method  string
	handler any
}

// Component[P] is the base type embedded by user components.
// P is the Props type for this component.
//
// Components embed *Component[P] to gain action registration, URL generation,
// and HTMX builders. The embedding pattern promotes methods directly onto the
// user's component type.
//
// Example:
//
//	type FileViewer struct {
//	    *hxcmp.Component[Props]
//	    repo *ops.Repo
//	}
//
//	func New(repo *ops.Repo) *FileViewer {
//	    c := &FileViewer{
//	        Component: hxcmp.New[Props]("fileviewer"),
//	        repo: repo,
//	    }
//	    c.Action("edit", c.handleEdit)
//	    return c
//	}
//
// Each component instance receives a deterministic URL prefix based on its
// name and source location (file:line), ensuring uniqueness without manual
// coordination.
type Component[P any] struct {
	name      string
	prefix    string
	sensitive bool
	actions   map[string]*actionDef
	encoder   *Encoder
	parent    any // The concrete component that embeds this
}

// New creates a new component with the given name.
//
// By default, props are signed (visible in URLs but tamper-proof via HMAC).
// Call .Sensitive() to enable full encryption for props containing sensitive
// data like user IDs or financial information.
//
// The component's URL prefix is derived from the name and source location
// (file:line where New is called), ensuring different instances get unique
// routes even with the same name.
func New[P any](name string) *Component[P] {
	prefix := "/_c/" + name + "-" + componentHash(name, 1)
	return &Component[P]{
		name:    name,
		prefix:  prefix,
		actions: make(map[string]*actionDef),
	}
}

// Sensitive marks the component as sensitive, enabling full encryption.
//
// Use for components that handle user IDs, financial data, or anything
// where props should be completely opaque to clients (not just tamper-proof).
//
// Signed mode (default) is debuggable - props are visible in URLs as base64
// JSON. Encrypted mode (via Sensitive) makes props completely opaque.
func (c *Component[P]) Sensitive() *Component[P] {
	c.sensitive = true
	return c
}

// Name returns the component's name.
func (c *Component[P]) Name() string {
	return c.name
}

// Prefix returns the component's URL prefix.
// All actions for this component are mounted under this prefix.
func (c *Component[P]) Prefix() string {
	return c.prefix
}

// IsSensitive returns whether the component uses encrypted props.
func (c *Component[P]) IsSensitive() bool {
	return c.sensitive
}

// Action registers a named action handler with default POST method.
//
// Actions use semantic names that describe intent (edit, delete, approve)
// rather than HTTP methods. The generated code produces typed methods
// (c.Edit, c.Delete) that catch typos at compile time.
//
// Returns *ActionBuilder to optionally override the HTTP method:
//
//	c.Action("edit", c.handleEdit)  // POST by default
//	c.Action("raw", c.handleRaw).Method(http.MethodGet)
//
// Handler signatures are auto-detected and can be:
//   - func(ctx, P) Result[P]
//   - func(ctx, P, *http.Request) Result[P]
//   - func(ctx, P, http.ResponseWriter) Result[P]
//
// The framework calls Hydrate before invoking the handler and Render
// after the handler returns OK or Err results.
func (c *Component[P]) Action(name string, handler any) *ActionBuilder {
	c.actions[name] = &actionDef{
		name:    name,
		method:  "POST", // Default to POST for mutations
		handler: handler,
	}
	return &ActionBuilder{action: c.actions[name]}
}

// Actions returns the registered actions (used by registry).
func (c *Component[P]) Actions() map[string]*actionDef {
	return c.actions
}

// SetEncoder sets the encoder for this component (called by registry).
func (c *Component[P]) SetEncoder(enc *Encoder) {
	c.encoder = enc
}

// Encoder returns the encoder for this component.
func (c *Component[P]) Encoder() *Encoder {
	return c.encoder
}

// SetParent sets the parent component (the concrete type embedding this).
// Called by generated code to enable method dispatch.
func (c *Component[P]) SetParent(parent any) {
	c.parent = parent
}

// Refresh returns an action builder for the default render (GET).
//
// Use this to create refresh/reload actions that re-render the component
// with updated props:
//
//	c.Refresh(props).Target("#content").Attrs()
func (c *Component[P]) Refresh(props P) *Action {
	return NewAction(c.buildURL("", props), "GET")
}

// Lazy returns a templ component that defers rendering until viewport intersection.
//
// The placeholder renders immediately; the actual component loads when scrolled
// into view. This optimizes initial page load by deferring below-the-fold content.
//
//	c.Lazy(props, loadingSpinner())
//
// Uses HTMX's "intersect once" trigger - loads once when entering viewport.
func (c *Component[P]) Lazy(props P, placeholder templ.Component) templ.Component {
	return lazyComponent(c.buildURL("", props), placeholder, "intersect once")
}

// Defer returns a templ component that loads after page load (not on intersection).
//
// The placeholder renders immediately; the actual component loads after the page
// finishes loading. Use for non-critical content that shouldn't block initial render.
//
//	c.Defer(props, placeholder())
//
// Uses HTMX's "load" trigger - fires once after page load completes.
func (c *Component[P]) Defer(props P, placeholder templ.Component) templ.Component {
	return lazyComponent(c.buildURL("", props), placeholder, "load")
}

// buildURL constructs the URL for an action with encoded props.
// Empty action string means default render (GET).
func (c *Component[P]) buildURL(action string, props P) string {
	path := c.prefix + "/"
	if action != "" {
		path = c.prefix + "/" + action
	}

	if c.encoder == nil {
		// Fallback if encoder not set (shouldn't happen in normal use)
		return path
	}

	encoded, err := c.encoder.Encode(props, c.sensitive)
	if err != nil {
		// In production, this should be logged
		return path
	}

	return path + "?p=" + encoded
}

// componentHash generates a deterministic hash based on component name and source location.
// This ensures each component instance gets a unique prefix without manual coordination.
func componentHash(name string, skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	var input string
	if ok {
		// Use base filename only for portability across environments
		input = fmt.Sprintf("%s:%d:%s", filepath.Base(file), line, name)
	} else {
		input = name
	}
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:4]) // 8 hex chars
}

// lazyComponent creates a placeholder that loads content on trigger.
func lazyComponent(url string, placeholder templ.Component, trigger string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, fmt.Sprintf(`<div hx-get="%s" hx-trigger="%s" hx-swap="outerHTML">`, url, trigger))
		if err != nil {
			return err
		}
		if placeholder != nil {
			if err := placeholder.Render(ctx, w); err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, `</div>`)
		return err
	})
}
