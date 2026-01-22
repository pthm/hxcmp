package hxcmp

import (
	"context"
	"errors"
	"html"
	"io"

	"github.com/a-h/templ"
)

// Sentinel errors for component operations.
var (
	// ErrNotFound indicates a requested resource doesn't exist.
	// Return this from Hydrate when an ID doesn't resolve to an object.
	ErrNotFound = errors.New("hxcmp: resource not found")

	// ErrDecryptFailed indicates props decryption failed (for sensitive components).
	// This typically means the ciphertext was corrupted or the key changed.
	ErrDecryptFailed = errors.New("hxcmp: parameter decryption failed")

	// ErrSignatureInvalid indicates props signature verification failed.
	// This means the props were tampered with or the key changed.
	ErrSignatureInvalid = errors.New("hxcmp: signature verification failed")

	// ErrInvalidFormat indicates props encoding is malformed.
	// This means the URL parameter is not valid base64 or JSON.
	ErrInvalidFormat = errors.New("hxcmp: invalid parameter format")

	// ErrHydrationFailed wraps errors from Hydrate implementations.
	//
	// Generated code wraps Hydrate errors with this sentinel so OnError handlers
	// can distinguish hydration failures (bad IDs, missing data) from action
	// handler errors (business logic failures). User code should return raw
	// errors from Hydrate - the framework handles wrapping automatically.
	ErrHydrationFailed = errors.New("hxcmp: hydration failed")
)

// IsNotFound checks if err is a not-found error.
//
// Use this to detect resource not found errors and return 404:
//
//	if hxcmp.IsNotFound(err) {
//	    http.Error(w, "Not found", http.StatusNotFound)
//	    return
//	}
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsDecryptionError checks if err is a decryption or signature error.
//
// Use this to detect tampered/corrupted props and return 400:
//
//	if hxcmp.IsDecryptionError(err) {
//	    http.Error(w, "Bad request", http.StatusBadRequest)
//	    return
//	}
func IsDecryptionError(err error) bool {
	return errors.Is(err, ErrDecryptFailed) || errors.Is(err, ErrSignatureInvalid)
}

// ErrorComponent returns a templ.Component that renders an error message.
//
// This is used by generated RenderHydrated code to display hydration errors
// visually rather than failing silently. The error is rendered as an HTML
// div with a class for styling.
func ErrorComponent(err error) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		msg := html.EscapeString(err.Error())
		_, writeErr := w.Write([]byte(`<div class="hxcmp-error" style="color: red; padding: 1rem; border: 1px solid red; margin: 0.5rem 0;">Hydration error: ` + msg + `</div>`))
		return writeErr
	})
}
