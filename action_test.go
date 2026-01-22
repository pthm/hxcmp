package hxcmp

import (
	"net/http"
	"testing"
	"time"
)

func TestNewAction(t *testing.T) {
	a := NewAction("/test/url", http.MethodPost)

	if a.URL() != "/test/url" {
		t.Errorf("URL() = %q, want %q", a.URL(), "/test/url")
	}

	attrs := a.Attrs()
	if attrs["hx-post"] != "/test/url" {
		t.Errorf("hx-post = %q, want %q", attrs["hx-post"], "/test/url")
	}

	// Default swap should be outerHTML
	if attrs["hx-swap"] != "outerHTML" {
		t.Errorf("hx-swap = %q, want %q", attrs["hx-swap"], "outerHTML")
	}
}

func TestActionMethods(t *testing.T) {
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
			a := NewAction("/url", tt.method)
			attrs := a.Attrs()

			if _, ok := attrs[tt.wantAttr]; !ok {
				t.Errorf("Expected attribute %q not found", tt.wantAttr)
			}
		})
	}
}

func TestActionTarget(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*Action) *Action
		expect string
	}{
		{
			name:   "Target",
			setup:  func(a *Action) *Action { return a.Target("#my-element") },
			expect: "#my-element",
		},
		{
			name:   "TargetThis",
			setup:  func(a *Action) *Action { return a.TargetThis() },
			expect: "this",
		},
		{
			name:   "TargetClosest",
			setup:  func(a *Action) *Action { return a.TargetClosest(".card") },
			expect: "closest .card",
		},
		{
			name:   "TargetFind",
			setup:  func(a *Action) *Action { return a.TargetFind(".content") },
			expect: "find .content",
		},
		{
			name:   "TargetNext",
			setup:  func(a *Action) *Action { return a.TargetNext(".sibling") },
			expect: "next .sibling",
		},
		{
			name:   "TargetPrevious",
			setup:  func(a *Action) *Action { return a.TargetPrevious(".sibling") },
			expect: "previous .sibling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.setup(NewAction("/url", http.MethodPost))
			attrs := a.Attrs()

			if attrs["hx-target"] != tt.expect {
				t.Errorf("hx-target = %q, want %q", attrs["hx-target"], tt.expect)
			}
		})
	}
}

func TestActionSwap(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*Action) *Action
		expect SwapMode
	}{
		{"Swap", func(a *Action) *Action { return a.Swap(SwapInner) }, SwapInner},
		{"SwapOuter", func(a *Action) *Action { return a.SwapOuter() }, SwapOuter},
		{"SwapInner", func(a *Action) *Action { return a.SwapInner() }, SwapInner},
		{"SwapBeforeEnd", func(a *Action) *Action { return a.SwapBeforeEnd() }, SwapBeforeEnd},
		{"SwapAfterEnd", func(a *Action) *Action { return a.SwapAfterEnd() }, SwapAfterEnd},
		{"SwapBeforeBegin", func(a *Action) *Action { return a.SwapBeforeBegin() }, SwapBeforeBegin},
		{"SwapAfterBegin", func(a *Action) *Action { return a.SwapAfterBegin() }, SwapAfterBegin},
		{"SwapDelete", func(a *Action) *Action { return a.SwapDelete() }, SwapDelete},
		{"SwapNone", func(a *Action) *Action { return a.SwapNone() }, SwapNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.setup(NewAction("/url", http.MethodPost))
			attrs := a.Attrs()

			if attrs["hx-swap"] != string(tt.expect) {
				t.Errorf("hx-swap = %q, want %q", attrs["hx-swap"], tt.expect)
			}
		})
	}
}

func TestActionTriggers(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*Action) *Action
		expect string
	}{
		{
			name:   "Every",
			setup:  func(a *Action) *Action { return a.Every(5 * time.Second) },
			expect: "every 5s",
		},
		{
			name:   "Every milliseconds",
			setup:  func(a *Action) *Action { return a.Every(500 * time.Millisecond) },
			expect: "every 500ms",
		},
		{
			name:   "OnEvent",
			setup:  func(a *Action) *Action { return a.OnEvent("itemAdded") },
			expect: "itemAdded from:body",
		},
		{
			name:   "OnLoad",
			setup:  func(a *Action) *Action { return a.OnLoad() },
			expect: "load",
		},
		{
			name:   "OnIntersect",
			setup:  func(a *Action) *Action { return a.OnIntersect() },
			expect: "intersect once",
		},
		{
			name:   "OnRevealed",
			setup:  func(a *Action) *Action { return a.OnRevealed() },
			expect: "revealed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.setup(NewAction("/url", http.MethodPost))
			attrs := a.Attrs()

			if attrs["hx-trigger"] != tt.expect {
				t.Errorf("hx-trigger = %q, want %q", attrs["hx-trigger"], tt.expect)
			}
		})
	}
}

func TestActionUX(t *testing.T) {
	t.Run("Confirm", func(t *testing.T) {
		a := NewAction("/url", http.MethodDelete).Confirm("Are you sure?")
		attrs := a.Attrs()

		if attrs["hx-confirm"] != "Are you sure?" {
			t.Errorf("hx-confirm = %q, want %q", attrs["hx-confirm"], "Are you sure?")
		}
	})

	t.Run("Indicator", func(t *testing.T) {
		a := NewAction("/url", http.MethodPost).Indicator("#spinner")
		attrs := a.Attrs()

		if attrs["hx-indicator"] != "#spinner" {
			t.Errorf("hx-indicator = %q, want %q", attrs["hx-indicator"], "#spinner")
		}
	})

	t.Run("PushURL", func(t *testing.T) {
		a := NewAction("/url", http.MethodGet).PushURL()
		attrs := a.Attrs()

		if attrs["hx-push-url"] != "true" {
			t.Errorf("hx-push-url = %q, want %q", attrs["hx-push-url"], "true")
		}
	})

	t.Run("Vals", func(t *testing.T) {
		a := NewAction("/url", http.MethodPost).Vals(map[string]any{
			"extra": "value",
			"count": 42,
		})
		attrs := a.Attrs()

		// Should contain JSON
		if attrs["hx-vals"] == "" {
			t.Error("hx-vals should not be empty")
		}
	})
}

func TestActionAsLink(t *testing.T) {
	a := NewAction("/download/file.pdf", http.MethodGet)
	attrs := a.AsLink()

	if attrs["href"] != "/download/file.pdf" {
		t.Errorf("href = %q, want %q", attrs["href"], "/download/file.pdf")
	}

	// Should not have hx-* attributes
	if _, ok := attrs["hx-get"]; ok {
		t.Error("AsLink should not include hx-get")
	}
}

func TestActionAsCallback(t *testing.T) {
	a := NewAction("/refresh", http.MethodGet).
		Target("#list").
		Swap(SwapInner)

	cb := a.AsCallback()

	if cb.URL != "/refresh" {
		t.Errorf("URL = %q, want %q", cb.URL, "/refresh")
	}
	if cb.Target != "#list" {
		t.Errorf("Target = %q, want %q", cb.Target, "#list")
	}
	if cb.Swap != "innerHTML" {
		t.Errorf("Swap = %q, want %q", cb.Swap, "innerHTML")
	}
}

func TestActionChaining(t *testing.T) {
	// Test that fluent chaining works correctly
	a := NewAction("/api/items", http.MethodPost).
		Target("#items-list").
		SwapBeforeEnd().
		Confirm("Add item?").
		Indicator("#loading").
		Vals(map[string]any{"source": "ui"})

	attrs := a.Attrs()

	if attrs["hx-post"] != "/api/items" {
		t.Error("hx-post not set correctly")
	}
	if attrs["hx-target"] != "#items-list" {
		t.Error("hx-target not set correctly")
	}
	if attrs["hx-swap"] != "beforeend" {
		t.Error("hx-swap not set correctly")
	}
	if attrs["hx-confirm"] != "Add item?" {
		t.Error("hx-confirm not set correctly")
	}
	if attrs["hx-indicator"] != "#loading" {
		t.Error("hx-indicator not set correctly")
	}
	if attrs["hx-vals"] == "" {
		t.Error("hx-vals not set")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d      time.Duration
		expect string
	}{
		{5 * time.Second, "5s"},
		{30 * time.Second, "30s"},
		{500 * time.Millisecond, "500ms"},
		{100 * time.Millisecond, "100ms"},
		{1 * time.Second, "1s"},
		{1500 * time.Millisecond, "1s"}, // Rounds down to seconds
	}

	for _, tt := range tests {
		t.Run(tt.d.String(), func(t *testing.T) {
			result := formatDuration(tt.d)
			if result != tt.expect {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, result, tt.expect)
			}
		})
	}
}
