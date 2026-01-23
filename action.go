package hxcmp

import (
	"encoding/json"
	"net/http"

	"github.com/a-h/templ"
)

// ActionBuilder configures action registration (e.g., HTTP method override).
//
// Returned by Component.Action() to allow optional method override:
//
//	c.Action("edit", handler)  // POST by default
//	c.Action("raw", handler).Method(http.MethodGet)
type ActionBuilder struct {
	action *actionDef
}

// Method overrides the default POST method for an action.
//
// Use for idempotent actions that should use GET, or for semantic deletions
// that should use DELETE:
//
//	c.Action("raw", c.handleRaw).Method(http.MethodGet)
//	c.Action("delete", c.handleDelete).Method(http.MethodDelete)
func (ab *ActionBuilder) Method(m string) *ActionBuilder {
	ab.action.method = m
	return ab
}

// WireAttrs builds the minimal HTMX attributes for a component action.
//
// For GET actions, returns hx-get with props encoded in the URL query string.
// For POST/PUT/DELETE/PATCH, returns hx-post (etc.) with props in hx-vals.
//
// This is the only thing hxcmp needs to provide — the URL and prop encoding.
// All other HTMX attributes (hx-target, hx-swap, hx-trigger, etc.) are
// written directly by the user in their templates.
//
// Used by generated Wire methods:
//
//	func (c *Counter) WireIncrement(props CounterProps) templ.Attributes {
//	    path, encoded := c.buildActionURL("increment", props)
//	    return hxcmp.WireAttrs(path, "POST", encoded)
//	}
func WireAttrs(path, method, encoded string) templ.Attributes {
	attrs := templ.Attributes{}

	if method == http.MethodGet || method == "" {
		url := path
		if encoded != "" {
			url = path + "?p=" + encoded
		}
		attrs["hx-get"] = url
	} else {
		switch method {
		case http.MethodPost:
			attrs["hx-post"] = path
		case http.MethodPut:
			attrs["hx-put"] = path
		case http.MethodPatch:
			attrs["hx-patch"] = path
		case http.MethodDelete:
			attrs["hx-delete"] = path
		}
		if encoded != "" {
			data, _ := json.Marshal(map[string]string{"p": encoded})
			attrs["hx-vals"] = string(data)
		}
	}

	return attrs
}
