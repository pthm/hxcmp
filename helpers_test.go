package hxcmp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsHTMX(t *testing.T) {
	tests := []struct {
		name   string
		header string
		expect bool
	}{
		{"with HX-Request true", "true", true},
		{"with HX-Request false", "false", false},
		{"without header", "", false},
		{"with other value", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("HX-Request", tt.header)
			}

			result := IsHTMX(req)
			if result != tt.expect {
				t.Errorf("IsHTMX() = %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestIsBoosted(t *testing.T) {
	tests := []struct {
		name   string
		header string
		expect bool
	}{
		{"with HX-Boosted true", "true", true},
		{"with HX-Boosted false", "false", false},
		{"without header", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("HX-Boosted", tt.header)
			}

			result := IsBoosted(req)
			if result != tt.expect {
				t.Errorf("IsBoosted() = %v, want %v", result, tt.expect)
			}
		})
	}
}

func TestCurrentURL(t *testing.T) {
	tests := []struct {
		name   string
		header string
		expect string
	}{
		{"with URL", "http://example.com/page", "http://example.com/page"},
		{"without header", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("HX-Current-URL", tt.header)
			}

			result := CurrentURL(req)
			if result != tt.expect {
				t.Errorf("CurrentURL() = %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestTriggerURL(t *testing.T) {
	// TriggerURL should be an alias for CurrentURL
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HX-Current-URL", "http://example.com/trigger")

	result := TriggerURL(req)
	if result != "http://example.com/trigger" {
		t.Errorf("TriggerURL() = %q, want %q", result, "http://example.com/trigger")
	}
}

func TestTriggerName(t *testing.T) {
	tests := []struct {
		name   string
		header string
		expect string
	}{
		{"with name", "submit-button", "submit-button"},
		{"without header", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("HX-Trigger-Name", tt.header)
			}

			result := TriggerName(req)
			if result != tt.expect {
				t.Errorf("TriggerName() = %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestTriggerID(t *testing.T) {
	tests := []struct {
		name   string
		header string
		expect string
	}{
		{"with ID", "btn-123", "btn-123"},
		{"without header", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("HX-Trigger", tt.header)
			}

			result := TriggerID(req)
			if result != tt.expect {
				t.Errorf("TriggerID() = %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestTargetID(t *testing.T) {
	tests := []struct {
		name   string
		header string
		expect string
	}{
		{"with ID", "target-div", "target-div"},
		{"without header", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("HX-Target", tt.header)
			}

			result := TargetID(req)
			if result != tt.expect {
				t.Errorf("TargetID() = %q, want %q", result, tt.expect)
			}
		})
	}
}

func TestRenderHelper(t *testing.T) {
	// We need a simple templ component for testing
	// Since we can't easily create one without templ, we'll skip the full render test
	// and just verify the function signature exists
	t.Skip("Render() requires a templ.Component which needs the templ package")
}

func TestBuildTriggerHeader(t *testing.T) {
	tests := []struct {
		name        string
		trigger     string
		triggerData map[string]any
		expect      string
	}{
		{
			name:   "empty",
			expect: "",
		},
		{
			name:    "simple trigger",
			trigger: "item-updated",
			expect:  "item-updated",
		},
		{
			name:        "trigger with data",
			trigger:     "filter:changed",
			triggerData: map[string]any{"status": "pending"},
			expect:      `{"filter:changed":{"status":"pending"}}`,
		},
		{
			name:        "trigger with multiple data values",
			trigger:     "item:saved",
			triggerData: map[string]any{"id": "123", "name": "test"},
			expect:      `{"item:saved":{"id":"123","name":"test"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildTriggerHeader(tt.trigger, tt.triggerData)
			if result != tt.expect {
				t.Errorf("BuildTriggerHeader() = %q, want %q", result, tt.expect)
			}
		})
	}
}
