package hxcmp

import "github.com/pthm/hxcmp/lib/encoding"

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
