package hxcmp

import (
	"errors"

	"github.com/pthm/hxcmp/lib/encoding"
)

// Encoder is an alias for encoding.Encoder for convenience.
type Encoder = encoding.Encoder

// Encodable is implemented by types that can encode themselves efficiently.
type Encodable = encoding.Encodable

// Decodable is implemented by types that can decode themselves efficiently.
type Decodable = encoding.Decodable

// NewEncoder creates a new encoder with the given encryption key.
func NewEncoder(key []byte) (*Encoder, error) {
	return encoding.NewEncoder(key)
}

// wrapEncodingError wraps encoding package errors with hxcmp sentinel errors.
func wrapEncodingError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, encoding.ErrInvalidFormat) {
		return ErrInvalidFormat
	}
	if errors.Is(err, encoding.ErrSignatureInvalid) {
		return ErrSignatureInvalid
	}
	if errors.Is(err, encoding.ErrDecryptFailed) {
		return ErrDecryptFailed
	}
	return err
}
