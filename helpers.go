package hxcmp

import (
	"encoding/json"
	"net/http"

	"github.com/a-h/templ"
)

// Render writes a templ component to the HTTP response.
//
// Sets Content-Type to text/html and renders the component using the
// request's context. Use this for non-component pages or when manually
// rendering component output.
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    hxcmp.Render(w, r, myTemplate())
//	}
//
// Component handlers don't need this - the framework auto-renders via
// the Renderer interface.
func Render(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(r.Context(), w)
}

// IsHTMX returns true if the request originated from HTMX.
//
// HTMX sends HX-Request: true on all requests. Use this to conditionally
// render partial content for HTMX vs full page for direct browser requests:
//
//	if hxcmp.IsHTMX(r) {
//	    return partialView()
//	}
//	return fullPageView()
func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// IsBoosted returns true if the request is a boosted navigation (hx-boost).
//
// hx-boost converts regular links/forms to HTMX requests. Use this to
// detect boosted requests and return only the main content area instead
// of the full layout:
//
//	if hxcmp.IsBoosted(r) {
//	    return contentOnly()
//	}
//	return fullLayout()
func IsBoosted(r *http.Request) bool {
	return r.Header.Get("HX-Boosted") == "true"
}

// CurrentURL returns the current URL from the HX-Current-URL header.
//
// This is the URL the browser is currently on (not the request URL).
// Useful for context-aware rendering or analytics:
//
//	currentPage := hxcmp.CurrentURL(r)
//
// Returns empty string if header not present (non-HTMX request).
func CurrentURL(r *http.Request) string {
	return r.Header.Get("HX-Current-URL")
}

// TriggerURL returns the URL that triggered this request (if HTMX).
//
// This is an alias for CurrentURL for semantic clarity.
func TriggerURL(r *http.Request) string {
	return r.Header.Get("HX-Current-URL")
}

// TriggerName returns the name attribute of the element that triggered the request.
//
// Useful for form handlers that need to know which submit button was clicked:
//
//	if hxcmp.TriggerName(r) == "save-draft" {
//	    // Handle draft save
//	}
//
// Returns empty string if not present.
func TriggerName(r *http.Request) string {
	return r.Header.Get("HX-Trigger-Name")
}

// TriggerID returns the id attribute of the element that triggered the request.
//
// Returns empty string if not present.
func TriggerID(r *http.Request) string {
	return r.Header.Get("HX-Trigger")
}

// TargetID returns the id attribute of the target element.
//
// This is the element that will receive the response (hx-target).
// Returns empty string if not present.
func TargetID(r *http.Request) string {
	return r.Header.Get("HX-Target")
}

// BuildTriggerHeader builds a properly formatted HX-Trigger header value.
//
// When both callback and trigger are present, they must be merged into a
// single JSON object. HTMX expects either bare event names or a JSON object,
// but not both mixed with commas.
//
// Used by generated code in handleResult.
func BuildTriggerHeader(cb *Callback, trigger string) string {
	if cb == nil && trigger == "" {
		return ""
	}

	// If only trigger (and no callback), return as simple event name
	if cb == nil {
		return trigger
	}

	// If callback exists, we need JSON format
	// Build a merged JSON object with callback and trigger
	merged := make(map[string]any)

	// Add callback event
	cbData := map[string]any{"url": cb.URL}
	if cb.Target != "" {
		cbData["target"] = cb.Target
	}
	if cb.Swap != "" {
		cbData["swap"] = cb.Swap
	}
	if len(cb.Vals) > 0 {
		cbData["vals"] = cb.Vals
	}
	merged["hxcmp:callback"] = cbData

	// Add trigger event (if present)
	if trigger != "" {
		merged[trigger] = true
	}

	data, _ := json.Marshal(merged)
	return string(data)
}
