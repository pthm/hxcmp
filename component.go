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

// Hydrater is implemented by components to reconstruct rich objects from serialized IDs.
// Called automatically before Render or any action handler.
type Hydrater[P any] interface {
	Hydrate(ctx context.Context, props *P) error
}

// Renderer is implemented by components to produce templ output.
type Renderer[P any] interface {
	Render(ctx context.Context, props P) templ.Component
}

// actionDef holds metadata about a registered action.
type actionDef struct {
	name    string
	method  string
	handler any
}

// Component[P] is the base type embedded by user components.
// P is the Props type for this component.
type Component[P any] struct {
	name      string
	prefix    string
	sensitive bool
	actions   map[string]*actionDef
	encoder   *Encoder
	parent    any // The concrete component that embeds this
}

// New creates a new component with the given name.
// By default, props are signed (visible but tamper-proof).
func New[P any](name string) *Component[P] {
	prefix := "/_c/" + name + "-" + componentHash(name, 1)
	return &Component[P]{
		name:    name,
		prefix:  prefix,
		actions: make(map[string]*actionDef),
	}
}

// Sensitive marks the component as sensitive, enabling full encryption.
// Use for components that handle user IDs, financial data, or anything
// where props should be opaque to clients.
func (c *Component[P]) Sensitive() *Component[P] {
	c.sensitive = true
	return c
}

// Name returns the component's name.
func (c *Component[P]) Name() string {
	return c.name
}

// Prefix returns the component's URL prefix.
func (c *Component[P]) Prefix() string {
	return c.prefix
}

// IsSensitive returns whether the component uses encrypted props.
func (c *Component[P]) IsSensitive() bool {
	return c.sensitive
}

// Action registers a named action handler.
// Returns *ActionBuilder for optional configuration (e.g., Method override).
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
func (c *Component[P]) SetParent(parent any) {
	c.parent = parent
}

// Refresh returns an action builder for the default render (GET).
func (c *Component[P]) Refresh(props P) *Action {
	return &Action{
		URL:    c.buildURL("", props),
		Method: "GET",
		Swap:   SwapOuter,
	}
}

// Lazy returns a templ component that defers rendering until viewport intersection.
func (c *Component[P]) Lazy(props P, placeholder templ.Component) templ.Component {
	return lazyComponent(c.buildURL("", props), placeholder, "intersect once")
}

// Defer returns a templ component that loads after page load (not on intersection).
func (c *Component[P]) Defer(props P, placeholder templ.Component) templ.Component {
	return lazyComponent(c.buildURL("", props), placeholder, "load")
}

// buildURL constructs the URL for an action with encoded props.
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
func componentHash(name string, skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	var input string
	if ok {
		// Use base filename only for portability
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
