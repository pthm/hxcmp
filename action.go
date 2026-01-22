package hxcmp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

// Action represents a component action with HTMX configuration.
//
// Provides a fluent builder for constructing HTMX attributes. Generated
// code produces typed action methods that return *Action:
//
//	c.Edit(props).Target("#editor").Confirm("Save?").Attrs()
//
// The builder exposes HTMX functionality directly rather than hiding it,
// enabling full control over swap strategies, targets, and triggers.
type Action struct {
	url          string // Full URL (for backwards compat) or base path
	method       string
	target       string
	swap         SwapMode
	trigger      string
	indicator    string
	confirm      string
	pushURL      bool
	vals         map[string]any
	encodedProps string // Encoded props to send separately for POST/PUT/DELETE
}

// NewAction creates a new action with URL and method.
// Called by generated code and Component.Refresh().
// Deprecated: Use NewActionWithProps for proper body encoding on POST.
func NewAction(url, method string) *Action {
	return &Action{
		url:    url,
		method: method,
		swap:   SwapOuter, // Default to replacing the entire element
	}
}

// NewActionWithProps creates an action with path and encoded props.
// For GET requests, props are added as query parameter.
// For POST/PUT/DELETE, props are sent in the request body via hx-vals.
func NewActionWithProps(path, method, encodedProps string) *Action {
	return &Action{
		url:          path,
		method:       method,
		swap:         SwapOuter,
		encodedProps: encodedProps,
	}
}

// ═══════════════════════════════════════════════════════════════
// Targeting
// ═══════════════════════════════════════════════════════════════

// Target sets the CSS selector for the target element.
//
// This determines which element receives the response HTML:
//
//	c.Edit(props).Target("#editor")     // Replace element with id="editor"
//	c.Edit(props).Target(".form-area")  // Replace first matching class
func (a *Action) Target(selector string) *Action {
	a.target = selector
	return a
}

// TargetThis sets the target to "this" (the triggering element).
//
// Useful for buttons that update themselves:
//
//	<button {...c.Toggle(props).TargetThis().Attrs()}>Toggle</button>
func (a *Action) TargetThis() *Action {
	a.target = "this"
	return a
}

// TargetClosest sets the target to the closest matching ancestor.
//
// Searches up the DOM tree from the trigger element:
//
//	c.Delete(props).TargetClosest(".item")  // Replace parent .item
func (a *Action) TargetClosest(selector string) *Action {
	a.target = "closest " + selector
	return a
}

// TargetFind sets the target to a descendant of the triggering element.
//
// Searches down the DOM tree from the trigger element:
//
//	c.LoadDetails(props).TargetFind(".details")
func (a *Action) TargetFind(selector string) *Action {
	a.target = "find " + selector
	return a
}

// TargetNext sets the target to the next sibling matching selector.
//
//	c.Expand(props).TargetNext(".content")
func (a *Action) TargetNext(selector string) *Action {
	a.target = "next " + selector
	return a
}

// TargetPrevious sets the target to the previous sibling matching selector.
//
//	c.Collapse(props).TargetPrevious(".header")
func (a *Action) TargetPrevious(selector string) *Action {
	a.target = "previous " + selector
	return a
}

// ═══════════════════════════════════════════════════════════════
// Swapping
// ═══════════════════════════════════════════════════════════════

// Swap sets the swap mode (how the response HTML replaces the target).
//
// See SwapMode constants for available strategies.
func (a *Action) Swap(mode SwapMode) *Action {
	a.swap = mode
	return a
}

// SwapOuter sets swap to outerHTML (replace entire element including tag).
// This is the default swap mode.
func (a *Action) SwapOuter() *Action {
	a.swap = SwapOuter
	return a
}

// SwapInner sets swap to innerHTML (replace contents, keep outer tag).
//
// Useful for updating containers without replacing the container itself:
//
//	<div id="list">{...c.RefreshList(props).Target("#list").SwapInner().Attrs()}</div>
func (a *Action) SwapInner() *Action {
	a.swap = SwapInner
	return a
}

// SwapBeforeEnd appends to end of target's contents.
//
// Adds new content as the last child:
//
//	c.AddItem(props).Target("#list").SwapBeforeEnd()  // Append to list
func (a *Action) SwapBeforeEnd() *Action {
	a.swap = SwapBeforeEnd
	return a
}

// SwapAfterEnd inserts after the target element (as next sibling).
func (a *Action) SwapAfterEnd() *Action {
	a.swap = SwapAfterEnd
	return a
}

// SwapBeforeBegin inserts before the target element (as previous sibling).
func (a *Action) SwapBeforeBegin() *Action {
	a.swap = SwapBeforeBegin
	return a
}

// SwapAfterBegin prepends to start of target's contents.
//
// Adds new content as the first child:
//
//	c.AddItem(props).Target("#list").SwapAfterBegin()  // Prepend to list
func (a *Action) SwapAfterBegin() *Action {
	a.swap = SwapAfterBegin
	return a
}

// SwapDelete removes the target element.
//
// Use with empty response to delete elements:
//
//	c.Delete(props).TargetClosest(".item").SwapDelete()
func (a *Action) SwapDelete() *Action {
	a.swap = SwapDelete
	return a
}

// SwapNone performs no swap (useful for side-effects only).
//
// The action executes but the response is discarded. Use when you only
// care about server-side effects (logging, analytics) or when using
// events/callbacks to notify other components:
//
//	c.Track(props).SwapNone()
func (a *Action) SwapNone() *Action {
	a.swap = SwapNone
	return a
}

// ═══════════════════════════════════════════════════════════════
// Triggers
// ═══════════════════════════════════════════════════════════════

// Every sets polling interval for periodic updates.
//
//	c.RefreshStats(props).Every(5 * time.Second)
func (a *Action) Every(d time.Duration) *Action {
	a.trigger = "every " + formatDuration(d)
	return a
}

// OnEvent listens for a custom event from body.
//
// Use for loose coupling between components. Can be chained for multiple events:
//
//	c.RefreshList(props).OnEvent("item-updated").OnEvent("filter-changed")
//	// Another component: return hxcmp.OK(props).Trigger("item-updated")
//
// Events with data are supported - the hxcmp JS extension automatically
// injects event data into request parameters.
func (a *Action) OnEvent(event string) *Action {
	eventTrigger := event + " from:body"
	if a.trigger == "" {
		a.trigger = eventTrigger
	} else {
		a.trigger = a.trigger + ", " + eventTrigger
	}
	return a
}

// OnLoad triggers on element load.
//
// Fires once when the element appears in the DOM:
//
//	c.LoadDetails(props).OnLoad()
func (a *Action) OnLoad() *Action {
	a.trigger = "load"
	return a
}

// OnIntersect triggers when element enters viewport (once).
//
// Useful for lazy-loading content below the fold:
//
//	c.LoadComments(props).OnIntersect()
func (a *Action) OnIntersect() *Action {
	a.trigger = "intersect once"
	return a
}

// OnRevealed triggers when element is scrolled into view.
//
// Similar to OnIntersect but fires on every scroll into view:
//
//	c.TrackView(props).OnRevealed()
func (a *Action) OnRevealed() *Action {
	a.trigger = "revealed"
	return a
}

// ═══════════════════════════════════════════════════════════════
// UX Enhancements
// ═══════════════════════════════════════════════════════════════

// Confirm shows a confirmation dialog before the action.
//
//	c.Delete(props).Confirm("Are you sure?")
func (a *Action) Confirm(message string) *Action {
	a.confirm = message
	return a
}

// Indicator sets the CSS selector for the loading indicator.
//
// The matched element receives the "htmx-request" class during the request:
//
//	c.Submit(props).Indicator("#spinner")
func (a *Action) Indicator(selector string) *Action {
	a.indicator = selector
	return a
}

// PushURL enables URL push to browser history.
//
// Updates the browser address bar with the request URL, enabling
// back/forward navigation:
//
//	c.ShowDetails(props).PushURL()
func (a *Action) PushURL() *Action {
	a.pushURL = true
	return a
}

// Vals sets additional values to include with the request.
//
// Sends extra data as JSON in the hx-vals attribute:
//
//	c.Filter(props).Vals(map[string]any{"page": 2})
func (a *Action) Vals(v map[string]any) *Action {
	a.vals = v
	return a
}

// ═══════════════════════════════════════════════════════════════
// Terminal Methods
// ═══════════════════════════════════════════════════════════════

// Attrs returns HTMX attributes for spreading in templ.
//
// This is the terminal method that produces the final attribute map:
//
//	<button {...c.Edit(props).Target("#editor").Confirm("Save?").Attrs()}>
//	    Save
//	</button>
//
// For GET requests, encoded props are added to the URL as query param.
// For POST/PUT/DELETE/PATCH, props are sent in the request body via hx-vals.
func (a *Action) Attrs() templ.Attributes {
	attrs := templ.Attributes{}

	// Build the URL - for GET include props in query string, for others use base path
	url := a.url
	if a.encodedProps != "" && (a.method == http.MethodGet || a.method == "") {
		url = a.url + "?p=" + a.encodedProps
	}

	// Set method-specific attribute
	switch a.method {
	case http.MethodGet, "":
		attrs["hx-get"] = url
	case http.MethodPost:
		attrs["hx-post"] = url
	case http.MethodPut:
		attrs["hx-put"] = url
	case http.MethodPatch:
		attrs["hx-patch"] = url
	case http.MethodDelete:
		attrs["hx-delete"] = url
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

	// Build hx-vals - merge user vals with encoded props for non-GET methods
	if a.encodedProps != "" && a.method != http.MethodGet && a.method != "" {
		mergedVals := make(map[string]any)
		for k, v := range a.vals {
			mergedVals[k] = v
		}
		mergedVals["p"] = a.encodedProps
		data, _ := json.Marshal(mergedVals)
		attrs["hx-vals"] = string(data)
	} else if len(a.vals) > 0 {
		data, _ := json.Marshal(a.vals)
		attrs["hx-vals"] = string(data)
	}

	return attrs
}

// AsLink returns attributes for an <a> tag.
//
// Produces a plain href attribute without HTMX. Use for actions that
// should work without JavaScript:
//
//	<a {...c.ViewRaw(props).AsLink()}>View Raw</a>
func (a *Action) AsLink() templ.Attributes {
	return templ.Attributes{
		"href": a.url,
	}
}

// URL returns the action URL with encoded props (for GET-style access).
//
// Useful for manual attribute construction or passing to JavaScript:
//
//	data-action-url={c.Submit(props).URL()}
func (a *Action) URL() string {
	if a.encodedProps != "" {
		return a.url + "?p=" + a.encodedProps
	}
	return a.url
}

// AsCallback converts the action to a Callback for parent-child communication.
//
// Deprecated: Use event-based communication instead. Have the child emit an
// event with Trigger, and have the parent listen with OnEvent:
//
//	// Child emits event:
//	return hxcmp.OK(props).Trigger("item:saved", map[string]any{"id": item.ID})
//
//	// Parent listens in template:
//	c.RefreshList(props).OnEvent("item:saved").Attrs()
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
