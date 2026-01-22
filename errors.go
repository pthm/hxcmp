package hxcmp

import "errors"

// Sentinel errors for component operations.
var (
	ErrNotFound         = errors.New("hxcmp: resource not found")
	ErrDecryptFailed    = errors.New("hxcmp: parameter decryption failed")
	ErrSignatureInvalid = errors.New("hxcmp: signature verification failed")
	ErrInvalidFormat    = errors.New("hxcmp: invalid parameter format")
	ErrHydrationFailed  = errors.New("hxcmp: hydration failed")
)

// IsNotFound checks if err is a not-found error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsDecryptionError checks if err is a decryption or signature error.
func IsDecryptionError(err error) bool {
	return errors.Is(err, ErrDecryptFailed) || errors.Is(err, ErrSignatureInvalid)
}
