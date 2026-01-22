package hxcmp

import (
	"errors"

	"github.com/pthm/hxcmp/lib/encoding"
)

// Encoder handles signing and encryption of component props for URLs.
//
// Props are encoded as URL parameters using one of two modes:
//   - Signed (default): HMAC-SHA256 authenticated JSON, visible but tamper-proof
//   - Encrypted (.Sensitive()): AES-256-GCM encrypted, completely opaque
//
// The encoder is shared across all components in a Registry and uses a
// single encryption key. This key should be at least 32 bytes of
// cryptographically random data.
//
// This is an alias for lib/encoding.Encoder for convenience.
type Encoder = encoding.Encoder

// Encodable is implemented by types that provide custom efficient encoding.
//
// Generated code implements this for Props types, producing fast,
// reflection-free serialization:
//
//	func (p Props) HXEncode(enc *encoding.EncodeBuffer) error {
//	    enc.WriteInt(p.ID)
//	    enc.WriteString(p.Name)
//	    return nil
//	}
//
// This is an alias for lib/encoding.Encodable.
type Encodable = encoding.Encodable

// Decodable is implemented by types that provide custom efficient decoding.
//
// Generated code implements this for Props types:
//
//	func (p *Props) HXDecode(dec *encoding.DecodeBuffer) error {
//	    p.ID = dec.ReadInt()
//	    p.Name = dec.ReadString()
//	    return nil
//	}
//
// This is an alias for lib/encoding.Decodable.
type Decodable = encoding.Decodable

// NewEncoder creates a new encoder with the given encryption key.
//
// The key must be at least 16 bytes, but 32 bytes is recommended for
// AES-256. Returns an error if the key is too short.
func NewEncoder(key []byte) (*Encoder, error) {
	return encoding.NewEncoder(key)
}

// wrapEncodingError wraps encoding package errors with hxcmp sentinel errors.
//
// This provides a stable error API at the hxcmp package level while allowing
// the encoding package to remain independent.
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
