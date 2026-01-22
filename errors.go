package hxcmp

import "errors"

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
	// Used by generated code to distinguish hydration errors from action errors.
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
