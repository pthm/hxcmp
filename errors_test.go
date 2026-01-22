package hxcmp

import (
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
