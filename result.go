package hxcmp

// Result[P] is returned from action handlers to control rendering and side effects.
//
// Result is a fluent builder that enables handlers to specify flash messages,
// redirects, events, and custom headers without writing directly to the
// ResponseWriter. The framework processes the Result after the handler
// returns, applying headers and calling Render as appropriate.
//
// Example patterns:
//
//	// Success - auto-render with updated props
//	return hxcmp.OK(props)
//
//	// Success with flash message
//	return hxcmp.OK(props).Flash("success", "Saved!")
//
//	// Error with fallback render
//	return hxcmp.Err(props, err)
//
//	// Redirect via HX-Redirect header
//	return hxcmp.Redirect[Props]("/dashboard")
//
//	// Broadcast event for loose coupling
//	return hxcmp.OK(props).Trigger("item-updated")
//
//	// Broadcast event with data (listeners receive as request params)
//	return hxcmp.OK(props).Trigger("filter:changed", map[string]any{"status": "active"})
//
// Result is not error-as-value pattern abuse - it's a structured way to
// communicate rendering intent to the framework. Errors are still errors
// (via Err()), not control flow.
type Result[P any] struct {
	props              P
	err                error
	redirect           string
	flashes            []Flash
	trigger            string
	triggerData        map[string]any
	triggerAfterSettle string // Event fired after swap settles (for URL sync)
	callback           *Callback // Deprecated: use Trigger with data instead
	headers            map[string]string
	status             int
	skip               bool
}

// OK creates a success result that will auto-render with the given props.
//
// The framework calls component.Render(ctx, props) to produce the response.
// Use this for the typical success case where the action updates props
// and you want the updated view rendered.
func OK[P any](props P) Result[P] {
	return Result[P]{props: props}
}

// Err creates an error result that passes the error to OnError handler.
//
// The registry's OnError callback determines the response (typically 500).
// Props are included so OnError can render a fallback view if desired.
//
// Hydration errors and decryption errors are automatically wrapped and
// sent through OnError - handlers typically only return Err for domain
// logic failures (validation, not found, permission denied).
func Err[P any](props P, err error) Result[P] {
	return Result[P]{props: props, err: err}
}

// Skip creates a result indicating the handler wrote its own response.
//
// No auto-render will occur. Use when the handler needs full control,
// such as streaming responses, file downloads, or custom content types:
//
//	func (c *Component) handleDownload(ctx context.Context, props Props) Result[Props] {
//	    w.Header().Set("Content-Type", "application/pdf")
//	    io.Copy(w, file)
//	    return hxcmp.Skip[Props]()
//	}
func Skip[P any]() Result[P] {
	return Result[P]{skip: true}
}

// Redirect creates a result that will redirect via HX-Redirect header.
//
// HTMX intercepts this header and performs a client-side redirect.
// Use for post-action navigation:
//
//	return hxcmp.Redirect[Props]("/dashboard")
func Redirect[P any](url string) Result[P] {
	var zero P
	return Result[P]{props: zero, redirect: url}
}

// Flash adds a flash message (toast notification) to the result.
//
// Flash messages are rendered as out-of-band (OOB) swaps that append
// to the #toasts container. Levels typically include "success", "error",
// "warning", "info" (see FlashSuccess, FlashError constants).
//
//	return hxcmp.OK(props).Flash("success", "Item saved!")
//
// Multiple flashes can be chained:
//
//	return hxcmp.OK(props).
//	    Flash("success", "Primary action completed").
//	    Flash("info", "Notification sent")
func (r Result[P]) Flash(level, message string) Result[P] {
	r.flashes = append(r.flashes, Flash{Level: level, Message: message})
	return r
}

// Callback triggers a parent callback to enable child-to-parent communication.
//
// Deprecated: Use Trigger with data instead. The event-based approach is more
// HTMX-native and decouples components. Callbacks will be removed in a future version.
//
//	// Old callback pattern:
//	return hxcmp.OK(props).Callback(props.OnSave)
//
//	// New event pattern:
//	return hxcmp.OK(props).Trigger("item:saved", map[string]any{"id": item.ID})
func (r Result[P]) Callback(cb Callback) Result[P] {
	r.callback = &cb
	return r
}

// Trigger emits an event via HX-Trigger header for component communication.
//
// Other components can listen for this event using OnEvent():
//
//	// Emitter (no data):
//	return hxcmp.OK(props).Trigger("item-updated")
//
//	// Emitter (with data - listeners receive as request params):
//	return hxcmp.OK(props).Trigger("filter:changed", map[string]any{"status": "active"})
//
//	// Listener (in template):
//	c.Refresh(props).OnEvent("filter:changed").Attrs()
//
// When data is provided, it's sent as part of the HX-Trigger header and
// automatically injected into listener requests as parameters by the
// hxcmp JavaScript extension.
//
// This pattern decouples components - the emitter doesn't know who's listening.
func (r Result[P]) Trigger(event string, data ...map[string]any) Result[P] {
	r.trigger = event
	if len(data) > 0 {
		r.triggerData = data[0]
	}
	return r
}

// PushURL updates the browser URL via HX-Push-Url header.
//
// Use this when an action changes shared URL state. Combined with TriggerURLSync,
// this enables React-like reactivity where URL is the shared state:
//
//	return hxcmp.OK(props).
//	    PushURL("/todos?status=pending").
//	    TriggerURLSync()
//
// Components using SyncURL() will automatically refresh and read the new URL params.
func (r Result[P]) PushURL(url string) Result[P] {
	return r.Header("HX-Push-Url", url)
}

// TriggerURLSync emits the "url:sync" event to refresh all URL-bound components.
//
// Components with SyncURL() listen for this event and re-render, reading their
// state from the browser's current URL. Use after PushURL to notify components:
//
//	return hxcmp.OK(props).
//	    PushURL("/todos?status=pending").
//	    TriggerURLSync()
//
// This uses HX-Trigger-After-Settle to ensure the URL is updated before
// the event fires, preventing race conditions where components read stale URLs.
func (r Result[P]) TriggerURLSync() Result[P] {
	r.triggerAfterSettle = "url:sync"
	return r
}

// Header sets a custom response header.
//
// Use for cache control, rate limiting metadata, or other HTTP semantics:
//
//	return hxcmp.OK(props).Header("Cache-Control", "no-store")
func (r Result[P]) Header(key, value string) Result[P] {
	if r.headers == nil {
		r.headers = make(map[string]string)
	}
	r.headers[key] = value
	return r
}

// Status sets the HTTP status code.
//
// The default is 200 for OK results. Use this to signal other success
// codes (201 Created, 204 No Content) or client errors (400, 404, 422):
//
//	return hxcmp.OK(props).Status(http.StatusCreated)
func (r Result[P]) Status(code int) Result[P] {
	r.status = code
	return r
}

// GetProps returns the props from the result.
func (r Result[P]) GetProps() P {
	return r.props
}

// GetErr returns the error from the result.
func (r Result[P]) GetErr() error {
	return r.err
}

// GetRedirect returns the redirect URL.
func (r Result[P]) GetRedirect() string {
	return r.redirect
}

// GetFlashes returns the flash messages.
func (r Result[P]) GetFlashes() []Flash {
	return r.flashes
}

// GetTrigger returns the trigger event name.
func (r Result[P]) GetTrigger() string {
	return r.trigger
}

// GetTriggerData returns the trigger event data.
func (r Result[P]) GetTriggerData() map[string]any {
	return r.triggerData
}

// GetTriggerAfterSettle returns the after-settle trigger event name.
func (r Result[P]) GetTriggerAfterSettle() string {
	return r.triggerAfterSettle
}

// GetCallback returns the callback.
//
// Deprecated: Callbacks are deprecated in favor of Trigger with data.
func (r Result[P]) GetCallback() *Callback {
	return r.callback
}

// GetHeaders returns the response headers.
func (r Result[P]) GetHeaders() map[string]string {
	return r.headers
}

// GetStatus returns the HTTP status code (0 means not set, use default 200).
func (r Result[P]) GetStatus() int {
	return r.status
}

// ShouldSkip returns whether the handler wrote its own response.
func (r Result[P]) ShouldSkip() bool {
	return r.skip
}
