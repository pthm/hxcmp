package hxcmp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	// Verify sentinel errors are distinct
	errs := []error{
		ErrNotFound,
		ErrDecryptFailed,
		ErrSignatureInvalid,
		ErrInvalidFormat,
		ErrHydrationFailed,
	}

	for i, err1 := range errs {
		for j, err2 := range errs {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("Sentinel errors should be distinct: %v and %v", err1, err2)
			}
		}
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil error", nil, false},
		{"ErrNotFound", ErrNotFound, true},
		{"wrapped ErrNotFound", fmt.Errorf("wrapped: %w", ErrNotFound), true},
		{"other error", errors.New("other error"), false},
		{"ErrDecryptFailed", ErrDecryptFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			if result != tt.expect {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, result, tt.expect)
			}
		})
	}
}

func TestIsDecryptionError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil error", nil, false},
		{"ErrDecryptFailed", ErrDecryptFailed, true},
		{"ErrSignatureInvalid", ErrSignatureInvalid, true},
		{"wrapped ErrDecryptFailed", fmt.Errorf("wrapped: %w", ErrDecryptFailed), true},
		{"wrapped ErrSignatureInvalid", fmt.Errorf("wrapped: %w", ErrSignatureInvalid), true},
		{"ErrNotFound", ErrNotFound, false},
		{"ErrInvalidFormat", ErrInvalidFormat, false},
		{"other error", errors.New("other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDecryptionError(tt.err)
			if result != tt.expect {
				t.Errorf("IsDecryptionError(%v) = %v, want %v", tt.err, result, tt.expect)
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	// Ensure error messages contain "hxcmp:" prefix
	errs := []error{
		ErrNotFound,
		ErrDecryptFailed,
		ErrSignatureInvalid,
		ErrInvalidFormat,
		ErrHydrationFailed,
	}

	for _, err := range errs {
		if err.Error()[:6] != "hxcmp:" {
			t.Errorf("Error %q should start with 'hxcmp:'", err.Error())
		}
	}
}

func TestErrorComponent(t *testing.T) {
	testErr := errors.New("test hydration error")
	comp := ErrorComponent(testErr)

	var buf bytes.Buffer
	err := comp.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("ErrorComponent.Render() error = %v", err)
	}

	html := buf.String()

	// Should contain the error class for styling
	if !bytes.Contains([]byte(html), []byte(`class="hxcmp-error"`)) {
		t.Errorf("ErrorComponent output should contain hxcmp-error class: %s", html)
	}

	// Should contain the error message
	if !bytes.Contains([]byte(html), []byte("test hydration error")) {
		t.Errorf("ErrorComponent output should contain error message: %s", html)
	}

	// Should contain "Hydration error:" prefix
	if !bytes.Contains([]byte(html), []byte("Hydration error:")) {
		t.Errorf("ErrorComponent output should contain 'Hydration error:' prefix: %s", html)
	}
}

func TestErrorComponent_HTMLEscaping(t *testing.T) {
	// Test that error messages are HTML-escaped to prevent XSS
	maliciousErr := errors.New(`<script>alert("xss")</script>`)
	comp := ErrorComponent(maliciousErr)

	var buf bytes.Buffer
	err := comp.Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("ErrorComponent.Render() error = %v", err)
	}

	html := buf.String()

	// Should NOT contain unescaped script tag
	if bytes.Contains([]byte(html), []byte("<script>")) {
		t.Errorf("ErrorComponent should escape HTML: %s", html)
	}

	// Should contain escaped version
	if !bytes.Contains([]byte(html), []byte("&lt;script&gt;")) {
		t.Errorf("ErrorComponent should contain HTML-escaped message: %s", html)
	}
}
