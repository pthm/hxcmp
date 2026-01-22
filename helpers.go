package hxcmp

import (
	"net/http"

	"github.com/a-h/templ"
)

// Render writes a templ component to the HTTP response.
func Render(w http.ResponseWriter, r *http.Request, component templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(r.Context(), w)
}

// IsHTMX returns true if the request originated from HTMX.
func IsHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// IsBoosted returns true if the request is a boosted navigation (hx-boost).
func IsBoosted(r *http.Request) bool {
	return r.Header.Get("HX-Boosted") == "true"
}

// CurrentURL returns the current URL from the HX-Current-URL header.
// Returns empty string if not present.
func CurrentURL(r *http.Request) string {
	return r.Header.Get("HX-Current-URL")
}

// TriggerName returns the name of the element that triggered the request.
// Returns empty string if not present.
func TriggerName(r *http.Request) string {
	return r.Header.Get("HX-Trigger-Name")
}

// TriggerID returns the ID of the element that triggered the request.
// Returns empty string if not present.
func TriggerID(r *http.Request) string {
	return r.Header.Get("HX-Trigger")
}

// TargetID returns the ID of the target element.
// Returns empty string if not present.
func TargetID(r *http.Request) string {
	return r.Header.Get("HX-Target")
}
