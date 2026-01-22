package hxcmp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/a-h/templ"
)

// SwapMode represents HTMX swap modes.
type SwapMode string

const (
	SwapOuter       SwapMode = "outerHTML"   // Replace entire element
	SwapInner       SwapMode = "innerHTML"   // Replace contents
	SwapBeforeEnd   SwapMode = "beforeend"   // Append to contents (inside, at end)
	SwapAfterEnd    SwapMode = "afterend"    // Insert after element (outside, after)
	SwapBeforeBegin SwapMode = "beforebegin" // Insert before element (outside, before)
	SwapAfterBegin  SwapMode = "afterbegin"  // Prepend to contents (inside, at start)
	SwapDelete      SwapMode = "delete"      // Delete element
	SwapNone        SwapMode = "none"        // No swap
)

// ActionBuilder configures action registration (e.g., HTTP method).
type ActionBuilder struct {
	action *actionDef
}

// Method overrides the default POST method for an action.
func (ab *ActionBuilder) Method(m string) *ActionBuilder {
	ab.action.method = m
	return ab
}

// Action represents a component action with HTMX configuration.
// It provides a fluent API for building HTMX attributes.
// Fields are exported for use in generated struct literals.
type Action struct {
	URL       string
	Method    string
	Swap      SwapMode
	target    string
	trigger   string
	indicator string
	confirm   string
	pushURL   bool
	vals      map[string]any
}

// Target sets the hx-target selector.
func (a *Action) Target(selector string) *Action {
	a.target = selector
	return a
}

// TargetThis sets hx-target="this".
func (a *Action) TargetThis() *Action {
	a.target = "this"
	return a
}

// TargetClosest sets hx-target="closest <selector>".
func (a *Action) TargetClosest(selector string) *Action {
	a.target = "closest " + selector
	return a
}

// SwapMode sets the hx-swap mode.
func (a *Action) SwapMode(mode SwapMode) *Action {
	a.Swap = mode
	return a
}

// SwapOuter sets hx-swap="outerHTML".
func (a *Action) SwapOuter() *Action {
	a.Swap = SwapOuter
	return a
}

// SwapInner sets hx-swap="innerHTML".
func (a *Action) SwapInner() *Action {
	a.Swap = SwapInner
	return a
}

// Every sets hx-trigger="every <duration>".
func (a *Action) Every(d time.Duration) *Action {
	a.trigger = fmt.Sprintf("every %s", d.String())
	return a
}

// OnEvent sets hx-trigger="<event> from:body".
func (a *Action) OnEvent(event string) *Action {
	a.trigger = event + " from:body"
	return a
}

// OnLoad sets hx-trigger="load".
func (a *Action) OnLoad() *Action {
	a.trigger = "load"
	return a
}

// OnIntersect sets hx-trigger="intersect once".
func (a *Action) OnIntersect() *Action {
	a.trigger = "intersect once"
	return a
}

// Confirm sets hx-confirm.
func (a *Action) Confirm(msg string) *Action {
	a.confirm = msg
	return a
}

// Indicator sets hx-indicator.
func (a *Action) Indicator(selector string) *Action {
	a.indicator = selector
	return a
}

// PushURL sets hx-push-url="true".
func (a *Action) PushURL() *Action {
	a.pushURL = true
	return a
}

// Vals sets hx-vals with the given map.
func (a *Action) Vals(v map[string]any) *Action {
	a.vals = v
	return a
}

// GetURL returns the URL string.
func (a *Action) GetURL() string {
	return a.URL
}

// Attrs returns templ.Attributes for spreading in templates.
func (a *Action) Attrs() templ.Attributes {
	attrs := templ.Attributes{}

	// Set method-specific attribute
	switch a.Method {
	case http.MethodGet:
		attrs["hx-get"] = a.URL
	case http.MethodPost:
		attrs["hx-post"] = a.URL
	case http.MethodDelete:
		attrs["hx-delete"] = a.URL
	case http.MethodPut:
		attrs["hx-put"] = a.URL
	case http.MethodPatch:
		attrs["hx-patch"] = a.URL
	default:
		attrs["hx-post"] = a.URL // Default to POST
	}

	if a.target != "" {
		attrs["hx-target"] = a.target
	}
	if a.Swap != "" {
		attrs["hx-swap"] = string(a.Swap)
	}
	if a.trigger != "" {
		attrs["hx-trigger"] = a.trigger
	}
	if a.indicator != "" {
		attrs["hx-indicator"] = a.indicator
	}
	if a.confirm != "" {
		attrs["hx-confirm"] = a.confirm
	}
	if a.pushURL {
		attrs["hx-push-url"] = "true"
	}
	if len(a.vals) > 0 {
		data, _ := json.Marshal(a.vals)
		attrs["hx-vals"] = string(data)
	}

	return attrs
}

// AsLink returns attributes suitable for <a> tags (href instead of hx-*).
func (a *Action) AsLink() templ.Attributes {
	return templ.Attributes{
		"href": a.URL,
	}
}

// AsCallback converts the action to a Callback for passing to child components.
func (a *Action) AsCallback() Callback {
	return Callback{
		URL:    a.URL,
		Target: a.target,
		Swap:   string(a.Swap),
	}
}
