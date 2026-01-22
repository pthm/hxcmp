package encoding

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/vmihailenco/msgpack/v5"
)

// Encoder handles encoding and decoding of component props.
// It supports two modes:
//   - Signed (default): Base64 + HMAC signature - visible but tamper-proof
//   - Encrypted: AES-256-GCM - fully opaque
type Encoder struct {
	key []byte
	gcm cipher.AEAD
}

// NewEncoder creates a new encoder with the given encryption key.
// The key should be 32 bytes for AES-256.
func NewEncoder(key []byte) (*Encoder, error) {
	if len(key) < 32 {
		// Derive a 32-byte key from the provided key
		h := sha256.Sum256(key)
		key = h[:]
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &Encoder{
		key: key,
		gcm: gcm,
	}, nil
}

// Encodable is implemented by types that can encode themselves efficiently.
// Generated code implements this interface.
type Encodable interface {
	HXEncode() map[string]any
}

// Decodable is implemented by types that can decode themselves efficiently.
// Generated code implements this interface.
type Decodable interface {
	HXDecode(map[string]any) error
}

// Encode serializes a value and returns an encoded string.
// If sensitive is true, the data is encrypted; otherwise it's signed.
func (e *Encoder) Encode(v any, sensitive bool) (string, error) {
	// Check if the value implements Encodable (generated code)
	var data map[string]any
	if enc, ok := v.(Encodable); ok {
		data = enc.HXEncode()
	} else {
		// Fallback: this shouldn't happen with generated code
		return "", errors.New("type does not implement Encodable")
	}

	// Marshal to msgpack
	packed, err := msgpack.Marshal(data)
	if err != nil {
		return "", err
	}

	if sensitive {
		return e.encrypt(packed)
	}
	return e.sign(packed)
}

// Decode deserializes an encoded string into a value.
// If sensitive is true, the data is decrypted; otherwise signature is verified.
func (e *Encoder) Decode(encoded string, sensitive bool, v any) error {
	var packed []byte
	var err error

	if sensitive {
		packed, err = e.decrypt(encoded)
	} else {
		packed, err = e.verify(encoded)
	}
	if err != nil {
		return err
	}

	// Unmarshal from msgpack
	var data map[string]any
	if err := msgpack.Unmarshal(packed, &data); err != nil {
		return err
	}

	// Check if the value implements Decodable (generated code)
	if dec, ok := v.(Decodable); ok {
		return dec.HXDecode(data)
	}

	return errors.New("type does not implement Decodable")
}

// sign creates a signed (but visible) encoding: base64.signature
func (e *Encoder) sign(data []byte) (string, error) {
	b64 := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, e.key)
	mac.Write(data)
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16]) // 16 bytes = 128 bits
	return b64 + "." + sig, nil
}

// verify verifies and decodes a signed string
func (e *Encoder) verify(encoded string) ([]byte, error) {
	parts := strings.SplitN(encoded, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid format: missing signature")
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, e.key)
	mac.Write(data)
	expected := mac.Sum(nil)[:16]

	if !hmac.Equal(sig, expected) {
		return nil, errors.New("signature verification failed")
	}

	return data, nil
}

// encrypt creates an encrypted encoding using AES-256-GCM
func (e *Encoder) encrypt(data []byte) (string, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := e.gcm.Seal(nonce, nonce, data, nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// decrypt decodes and decrypts an encrypted string
func (e *Encoder) decrypt(encoded string) ([]byte, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < e.gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:e.gcm.NonceSize()]
	ciphertext = ciphertext[e.gcm.NonceSize():]

	return e.gcm.Open(nil, nonce, ciphertext, nil)
}
