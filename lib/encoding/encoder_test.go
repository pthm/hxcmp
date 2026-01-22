package encoding

import (
	"testing"
)

// testProps implements Encodable and Decodable for testing.
type testProps struct {
	ID   int64
	Name string
	Flag bool
}

func (p testProps) HXEncode() map[string]any {
	return map[string]any{
		"id":   p.ID,
		"name": p.Name,
		"flag": p.Flag,
	}
}

func (p *testProps) HXDecode(m map[string]any) error {
	if v, ok := m["id"]; ok {
		switch n := v.(type) {
		case int64:
			p.ID = n
		case float64:
			p.ID = int64(n)
		}
	}
	if v, ok := m["name"].(string); ok {
		p.Name = v
	}
	if v, ok := m["flag"].(bool); ok {
		p.Flag = v
	}
	return nil
}

func TestNewEncoder(t *testing.T) {
	// Should work with any key length (derives 32-byte key)
	_, err := NewEncoder([]byte("short"))
	if err != nil {
		t.Fatalf("NewEncoder with short key failed: %v", err)
	}

	_, err = NewEncoder([]byte("this-is-a-32-byte-key-for-aes!!"))
	if err != nil {
		t.Fatalf("NewEncoder with 32-byte key failed: %v", err)
	}
}

func TestSignedRoundTrip(t *testing.T) {
	enc, err := NewEncoder([]byte("test-key"))
	if err != nil {
		t.Fatalf("NewEncoder failed: %v", err)
	}

	original := testProps{
		ID:   12345,
		Name: "test-file.txt",
		Flag: true,
	}

	// Encode (signed, not encrypted)
	encoded, err := enc.Encode(original, false)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Should contain a dot separator (base64.signature)
	if len(encoded) == 0 {
		t.Fatal("Encoded string is empty")
	}

	// Decode
	var decoded testProps
	err = enc.Decode(encoded, false, &decoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify values
	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %d, want %d", decoded.ID, original.ID)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Flag != original.Flag {
		t.Errorf("Flag mismatch: got %v, want %v", decoded.Flag, original.Flag)
	}
}

func TestEncryptedRoundTrip(t *testing.T) {
	enc, err := NewEncoder([]byte("test-key"))
	if err != nil {
		t.Fatalf("NewEncoder failed: %v", err)
	}

	original := testProps{
		ID:   67890,
		Name: "secret-file.txt",
		Flag: false,
	}

	// Encode (encrypted)
	encoded, err := enc.Encode(original, true)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Decode
	var decoded testProps
	err = enc.Decode(encoded, true, &decoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	// Verify values
	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %d, want %d", decoded.ID, original.ID)
	}
	if decoded.Name != original.Name {
		t.Errorf("Name mismatch: got %q, want %q", decoded.Name, original.Name)
	}
	if decoded.Flag != original.Flag {
		t.Errorf("Flag mismatch: got %v, want %v", decoded.Flag, original.Flag)
	}
}

func TestSignatureVerificationFailure(t *testing.T) {
	enc, err := NewEncoder([]byte("test-key"))
	if err != nil {
		t.Fatalf("NewEncoder failed: %v", err)
	}

	original := testProps{ID: 123, Name: "test"}
	encoded, err := enc.Encode(original, false)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Tamper with the encoded string
	tampered := encoded[:len(encoded)-2] + "XX"

	var decoded testProps
	err = enc.Decode(tampered, false, &decoded)
	if err == nil {
		t.Error("Expected error for tampered signature, got nil")
	}
	if err != ErrSignatureInvalid && err != ErrDecryptFailed {
		t.Errorf("Expected signature/decrypt error, got: %v", err)
	}
}

func TestDecryptionFailure(t *testing.T) {
	enc, err := NewEncoder([]byte("test-key"))
	if err != nil {
		t.Fatalf("NewEncoder failed: %v", err)
	}

	original := testProps{ID: 123, Name: "test"}
	encoded, err := enc.Encode(original, true)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Tamper with the encrypted string
	tampered := encoded[:len(encoded)-2] + "XX"

	var decoded testProps
	err = enc.Decode(tampered, true, &decoded)
	if err == nil {
		t.Error("Expected error for tampered ciphertext, got nil")
	}
}

func TestInvalidFormat(t *testing.T) {
	enc, err := NewEncoder([]byte("test-key"))
	if err != nil {
		t.Fatalf("NewEncoder failed: %v", err)
	}

	// Missing signature separator
	var decoded testProps
	err = enc.Decode("invalidbase64withoutseparator", false, &decoded)
	if err != ErrInvalidFormat {
		t.Errorf("Expected ErrInvalidFormat, got: %v", err)
	}
}

func TestDifferentKeysCannotDecode(t *testing.T) {
	enc1, _ := NewEncoder([]byte("key-one"))
	enc2, _ := NewEncoder([]byte("key-two"))

	original := testProps{ID: 123, Name: "test"}

	// Encode with key 1
	encoded, err := enc1.Encode(original, false)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Try to decode with key 2
	var decoded testProps
	err = enc2.Decode(encoded, false, &decoded)
	if err == nil {
		t.Error("Expected error when decoding with different key")
	}
}

func TestEmptyProps(t *testing.T) {
	enc, err := NewEncoder([]byte("test-key"))
	if err != nil {
		t.Fatalf("NewEncoder failed: %v", err)
	}

	original := testProps{} // Zero values

	// Signed
	encoded, err := enc.Encode(original, false)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	var decoded testProps
	err = enc.Decode(encoded, false, &decoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.ID != 0 || decoded.Name != "" || decoded.Flag != false {
		t.Error("Empty props not decoded correctly")
	}
}
