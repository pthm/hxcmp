package hxcmp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/a-h/templ"
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
// Uses fluent builder pattern for configuration.
type Action struct {
	url       string
	method    string
	target    string
	swap      SwapMode
	trigger   string
	indicator string
	confirm   string
	pushURL   bool
	vals      map[string]any
}

// NewAction creates a new action with URL and method.
// This is called by generated code.
func NewAction(url, method string) *Action {
	return &Action{
		url:    url,
		method: method,
		swap:   SwapOuter, // Default
	}
}

// ═══════════════════════════════════════════════════════════════
// Targeting
// ═══════════════════════════════════════════════════════════════

// Target sets the CSS selector for the target element.
func (a *Action) Target(selector string) *Action {
	a.target = selector
	return a
}

// TargetThis sets the target to "this" (the triggering element).
func (a *Action) TargetThis() *Action {
	a.target = "this"
	return a
}

// TargetClosest sets the target to the closest matching ancestor.
func (a *Action) TargetClosest(selector string) *Action {
	a.target = "closest " + selector
	return a
}

// TargetFind sets the target to a descendant of the triggering element.
func (a *Action) TargetFind(selector string) *Action {
	a.target = "find " + selector
	return a
}

// TargetNext sets the target to the next sibling matching selector.
func (a *Action) TargetNext(selector string) *Action {
	a.target = "next " + selector
	return a
}

// TargetPrevious sets the target to the previous sibling matching selector.
func (a *Action) TargetPrevious(selector string) *Action {
	a.target = "previous " + selector
	return a
}

// ═══════════════════════════════════════════════════════════════
// Swapping
// ═══════════════════════════════════════════════════════════════

// Swap sets the swap mode.
func (a *Action) Swap(mode SwapMode) *Action {
	a.swap = mode
	return a
}

// SwapOuter sets swap to outerHTML (replace entire element).
func (a *Action) SwapOuter() *Action {
	a.swap = SwapOuter
	return a
}

// SwapInner sets swap to innerHTML (replace contents).
func (a *Action) SwapInner() *Action {
	a.swap = SwapInner
	return a
}

// SwapBeforeEnd appends to end of target's contents.
func (a *Action) SwapBeforeEnd() *Action {
	a.swap = SwapBeforeEnd
	return a
}

// SwapAfterEnd inserts after the target element.
func (a *Action) SwapAfterEnd() *Action {
	a.swap = SwapAfterEnd
	return a
}

// SwapBeforeBegin inserts before the target element.
func (a *Action) SwapBeforeBegin() *Action {
	a.swap = SwapBeforeBegin
	return a
}

// SwapAfterBegin prepends to start of target's contents.
func (a *Action) SwapAfterBegin() *Action {
	a.swap = SwapAfterBegin
	return a
}

// SwapDelete removes the target element.
func (a *Action) SwapDelete() *Action {
	a.swap = SwapDelete
	return a
}

// SwapNone performs no swap (useful for side-effects only).
func (a *Action) SwapNone() *Action {
	a.swap = SwapNone
	return a
}

// ═══════════════════════════════════════════════════════════════
// Triggers
// ═══════════════════════════════════════════════════════════════

// Every sets polling interval.
func (a *Action) Every(d time.Duration) *Action {
	a.trigger = "every " + formatDuration(d)
	return a
}

// OnEvent listens for a custom event from body.
func (a *Action) OnEvent(event string) *Action {
	a.trigger = event + " from:body"
	return a
}

// OnLoad triggers on element load.
func (a *Action) OnLoad() *Action {
	a.trigger = "load"
	return a
}

// OnIntersect triggers when element enters viewport (once).
func (a *Action) OnIntersect() *Action {
	a.trigger = "intersect once"
	return a
}

// OnRevealed triggers when element is scrolled into view.
func (a *Action) OnRevealed() *Action {
	a.trigger = "revealed"
	return a
}

// ═══════════════════════════════════════════════════════════════
// UX Enhancements
// ═══════════════════════════════════════════════════════════════

// Confirm shows a confirmation dialog before the action.
func (a *Action) Confirm(message string) *Action {
	a.confirm = message
	return a
}

// Indicator sets the CSS selector for the loading indicator.
func (a *Action) Indicator(selector string) *Action {
	a.indicator = selector
	return a
}

// PushURL enables URL push to browser history.
func (a *Action) PushURL() *Action {
	a.pushURL = true
	return a
}

// Vals sets additional values to include with the request.
func (a *Action) Vals(v map[string]any) *Action {
	a.vals = v
	return a
}

// ═══════════════════════════════════════════════════════════════
// Terminal Methods
// ═══════════════════════════════════════════════════════════════

// Attrs returns HTMX attributes for spreading in templ.
// Usage: <button {...c.Edit(props).Attrs()}>Edit</button>
func (a *Action) Attrs() templ.Attributes {
	attrs := templ.Attributes{}

	// Set method-specific attribute
	switch a.method {
	case http.MethodGet, "":
		attrs["hx-get"] = a.url
	case http.MethodPost:
		attrs["hx-post"] = a.url
	case http.MethodPut:
		attrs["hx-put"] = a.url
	case http.MethodPatch:
		attrs["hx-patch"] = a.url
	case http.MethodDelete:
		attrs["hx-delete"] = a.url
	}

	if a.target != "" {
		attrs["hx-target"] = a.target
	}
	if a.swap != "" {
		attrs["hx-swap"] = string(a.swap)
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

// AsLink returns attributes for an <a> tag.
// Usage: <a {...c.Raw(props).AsLink()}>View Raw</a>
func (a *Action) AsLink() templ.Attributes {
	return templ.Attributes{
		"href": a.url,
	}
}

// URL returns just the action URL.
// Useful for manual attribute construction.
func (a *Action) URL() string {
	return a.url
}

// AsCallback converts the action to a Callback.
// Usage: OnSubmit: c.Refresh(props).Target("#list").AsCallback()
func (a *Action) AsCallback() Callback {
	return Callback{
		URL:    a.url,
		Target: a.target,
		Swap:   string(a.swap),
	}
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

// formatDuration converts duration to HTMX format (e.g., "5s", "500ms").
func formatDuration(d time.Duration) string {
	if d >= time.Second {
		secs := int(d.Seconds())
		return fmt.Sprintf("%ds", secs)
	}
	ms := int(d.Milliseconds())
	return fmt.Sprintf("%dms", ms)
}
