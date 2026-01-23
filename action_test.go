package hxcmp

import (
	"net/http"
	"testing"
)

func TestWireAttrs_GET(t *testing.T) {
	attrs := WireAttrs("/test/url", http.MethodGet, "encoded123")

	if attrs["hx-get"] != "/test/url?p=encoded123" {
		t.Errorf("hx-get = %q, want %q", attrs["hx-get"], "/test/url?p=encoded123")
	}
	if _, ok := attrs["hx-vals"]; ok {
		t.Error("GET should not set hx-vals")
	}
}

func TestWireAttrs_GET_NoProps(t *testing.T) {
	attrs := WireAttrs("/test/url", http.MethodGet, "")

	if attrs["hx-get"] != "/test/url" {
		t.Errorf("hx-get = %q, want %q", attrs["hx-get"], "/test/url")
	}
}

func TestWireAttrs_POST(t *testing.T) {
	attrs := WireAttrs("/test/url", http.MethodPost, "encoded123")

	if attrs["hx-post"] != "/test/url" {
		t.Errorf("hx-post = %q, want %q", attrs["hx-post"], "/test/url")
	}
	if attrs["hx-vals"] == "" {
		t.Error("POST with encoded props should set hx-vals")
	}
}

func TestWireAttrs_POST_NoProps(t *testing.T) {
	attrs := WireAttrs("/test/url", http.MethodPost, "")

	if attrs["hx-post"] != "/test/url" {
		t.Errorf("hx-post = %q, want %q", attrs["hx-post"], "/test/url")
	}
	if _, ok := attrs["hx-vals"]; ok {
		t.Error("POST without props should not set hx-vals")
	}
}

func TestWireAttrs_Methods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		wantAttr string
	}{
		{"GET", http.MethodGet, "hx-get"},
		{"POST", http.MethodPost, "hx-post"},
		{"PUT", http.MethodPut, "hx-put"},
		{"PATCH", http.MethodPatch, "hx-patch"},
		{"DELETE", http.MethodDelete, "hx-delete"},
		{"empty defaults to GET", "", "hx-get"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := WireAttrs("/url", tt.method, "")
			if _, ok := attrs[tt.wantAttr]; !ok {
				t.Errorf("Expected attribute %q not found", tt.wantAttr)
			}
		})
	}
}

func TestActionBuilderMethod(t *testing.T) {
	action := &actionDef{name: "test", method: "POST"}
	ab := &ActionBuilder{action: action}

	ab.Method(http.MethodDelete)

	if action.method != http.MethodDelete {
		t.Errorf("method = %q, want %q", action.method, http.MethodDelete)
	}
}
