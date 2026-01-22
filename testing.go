package hxcmp

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/a-h/templ"
)

// TestResult holds the result of rendering a component for testing.
//
// Provides convenience methods for asserting on HTML content, headers,
// status codes, events, flashes, and redirects.
type TestResult struct {
	HTML            string
	StatusCode      int
	Headers         http.Header
	TriggeredEvents []string
	Flashes         []Flash
	RedirectURL     string
}

// TestableComponent combines Hydrater and Renderer for testing.
//
// Most components will satisfy this interface automatically by implementing
// the required lifecycle methods.
type TestableComponent[P any] interface {
	Hydrater[P]
	Renderer[P]
}

// TestRender renders a component and returns testable output.
//
// Use this for pure unit tests of rendering logic when you control props directly
// and don't need HTTP mechanics. This bypasses URL encoding/decoding and runs
// only Hydrate + Render.
//
// For testing action handlers (including encoding, routing, and Result processing),
// use TestAction or the TestGet/TestPost convenience wrappers.
//
//	result, err := hxcmp.TestRender(comp, props)
//	if !result.HTMLContains("expected text") {
//	    t.Fatal("missing expected content")
//	}
func TestRender[P any](comp TestableComponent[P], props P) (*TestResult, error) {
	ctx := context.Background()

	// Run hydration
	if err := comp.Hydrate(ctx, &props); err != nil {
		return nil, err
	}

	// Render to buffer
	var buf bytes.Buffer
	component := comp.Render(ctx, props)
	if err := component.Render(ctx, &buf); err != nil {
		return nil, err
	}

	return &TestResult{
		HTML:       buf.String(),
		StatusCode: http.StatusOK,
		Headers:    make(http.Header),
	}, nil
}

// TestRenderWithContext renders a component with a custom context.
//
// Use this when testing components that read values from context
// (user authentication, request-scoped data):
//
//	ctx := context.WithValue(context.Background(), "user", testUser)
//	result, err := hxcmp.TestRenderWithContext(ctx, comp, props)
func TestRenderWithContext[P any](ctx context.Context, comp TestableComponent[P], props P) (*TestResult, error) {
	// Run hydration
	if err := comp.Hydrate(ctx, &props); err != nil {
		return nil, err
	}

	// Render to buffer
	var buf bytes.Buffer
	component := comp.Render(ctx, props)
	if err := component.Render(ctx, &buf); err != nil {
		return nil, err
	}

	return &TestResult{
		HTML:       buf.String(),
		StatusCode: http.StatusOK,
		Headers:    make(http.Header),
	}, nil
}

// TestAction simulates an action request against an HXComponent.
//
// This tests the full HTTP lifecycle including decoding, hydration,
// handler execution, and response rendering. Use this for integration
// tests of action handlers:
//
//	result, err := hxcmp.TestAction(comp, editURL, "POST", map[string]string{
//	    "name": "new name",
//	})
//	if !result.IsOK() {
//	    t.Fatal("expected success")
//	}
func TestAction(
	comp HXComponent,
	actionURL string,
	method string,
	formData map[string]string,
) (*TestResult, error) {
	// Build form body
	form := url.Values{}
	for k, v := range formData {
		form.Set(k, v)
	}

	var body *strings.Reader
	if len(formData) > 0 {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}

	req := httptest.NewRequest(method, actionURL, body)
	if len(formData) > 0 {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("HX-Request", "true")

	// Record response
	rec := httptest.NewRecorder()
	comp.HXServeHTTP(rec, req)

	result := &TestResult{
		HTML:       rec.Body.String(),
		StatusCode: rec.Code,
		Headers:    rec.Header(),
	}

	// Parse triggered events from HX-Trigger header
	if trigger := rec.Header().Get("HX-Trigger"); trigger != "" {
		result.TriggeredEvents = parseTriggerHeader(trigger)
	}

	// Parse redirect from HX-Redirect header
	if redirect := rec.Header().Get("HX-Redirect"); redirect != "" {
		result.RedirectURL = redirect
	}

	// Parse flashes from the HTML (they appear as OOB swaps)
	result.Flashes = parseFlashesFromHTML(result.HTML)

	return result, nil
}

// TestActionWithContext simulates an action request with a custom context.
//
// Use this for testing actions that require authenticated users or other
// context-scoped data:
//
//	ctx := context.WithValue(context.Background(), "userID", 123)
//	result, err := hxcmp.TestActionWithContext(ctx, comp, actionURL, "POST", formData)
func TestActionWithContext(
	ctx context.Context,
	comp HXComponent,
	actionURL string,
	method string,
	formData map[string]string,
) (*TestResult, error) {
	// Build form body
	form := url.Values{}
	for k, v := range formData {
		form.Set(k, v)
	}

	var body *strings.Reader
	if len(formData) > 0 {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}

	req := httptest.NewRequest(method, actionURL, body)
	req = req.WithContext(ctx)
	if len(formData) > 0 {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("HX-Request", "true")

	// Record response
	rec := httptest.NewRecorder()
	comp.HXServeHTTP(rec, req)

	result := &TestResult{
		HTML:       rec.Body.String(),
		StatusCode: rec.Code,
		Headers:    rec.Header(),
	}

	// Parse triggered events from HX-Trigger header
	if trigger := rec.Header().Get("HX-Trigger"); trigger != "" {
		result.TriggeredEvents = parseTriggerHeader(trigger)
	}

	// Parse redirect from HX-Redirect header
	if redirect := rec.Header().Get("HX-Redirect"); redirect != "" {
		result.RedirectURL = redirect
	}

	// Parse flashes from the HTML (they appear as OOB swaps)
	result.Flashes = parseFlashesFromHTML(result.HTML)

	return result, nil
}

// TestGet simulates a GET request (render) against an HXComponent.
//
// Convenience wrapper for TestAction with GET method:
//
//	result, err := hxcmp.TestGet(comp, renderURL)
func TestGet(comp HXComponent, url string) (*TestResult, error) {
	return TestAction(comp, url, http.MethodGet, nil)
}

// TestPost simulates a POST request against an HXComponent.
//
// Convenience wrapper for TestAction with POST method:
//
//	result, err := hxcmp.TestPost(comp, actionURL, map[string]string{
//	    "field": "value",
//	})
func TestPost(comp HXComponent, url string, formData map[string]string) (*TestResult, error) {
	return TestAction(comp, url, http.MethodPost, formData)
}

// HTMLContains checks if the HTML contains a substring.
func (r *TestResult) HTMLContains(substr string) bool {
	return strings.Contains(r.HTML, substr)
}

// HTMLContainsAll checks if the HTML contains all the given substrings.
func (r *TestResult) HTMLContainsAll(substrs ...string) bool {
	for _, s := range substrs {
		if !strings.Contains(r.HTML, s) {
			return false
		}
	}
	return true
}

// HTMLContainsAny checks if the HTML contains any of the given substrings.
func (r *TestResult) HTMLContainsAny(substrs ...string) bool {
	for _, s := range substrs {
		if strings.Contains(r.HTML, s) {
			return true
		}
	}
	return false
}

// HasEvent checks if an event was triggered.
func (r *TestResult) HasEvent(event string) bool {
	for _, e := range r.TriggeredEvents {
		if strings.Contains(e, event) {
			return true
		}
	}
	return false
}

// HasFlash checks if a flash message was set with the given level and message.
func (r *TestResult) HasFlash(level, message string) bool {
	for _, f := range r.Flashes {
		if f.Level == level && f.Message == message {
			return true
		}
	}
	return false
}

// HasFlashLevel checks if any flash message was set with the given level.
func (r *TestResult) HasFlashLevel(level string) bool {
	for _, f := range r.Flashes {
		if f.Level == level {
			return true
		}
	}
	return false
}

// WasRedirected checks if the response was a redirect.
func (r *TestResult) WasRedirected() bool {
	return r.RedirectURL != ""
}

// RedirectedTo checks if the response was redirected to a specific URL.
func (r *TestResult) RedirectedTo(url string) bool {
	return r.RedirectURL == url
}

// IsOK checks if the status code is 200.
func (r *TestResult) IsOK() bool {
	return r.StatusCode == http.StatusOK
}

// HasStatus checks if the status code matches.
func (r *TestResult) HasStatus(code int) bool {
	return r.StatusCode == code
}

// HasHeader checks if a header is set with the given value.
func (r *TestResult) HasHeader(key, value string) bool {
	return r.Headers.Get(key) == value
}

// GetHeader returns the value of a header.
func (r *TestResult) GetHeader(key string) string {
	return r.Headers.Get(key)
}

// parseTriggerHeader parses the HX-Trigger header value into event names.
// The header can be a simple event name or JSON.
func parseTriggerHeader(trigger string) []string {
	trigger = strings.TrimSpace(trigger)
	if trigger == "" {
		return nil
	}

	// If it starts with '{', it's JSON - parse event names from top-level keys
	if strings.HasPrefix(trigger, "{") {
		var events []string
		// Track depth to only extract top-level keys
		depth := 0
		inString := false
		stringStart := -1

		for i := 0; i < len(trigger); i++ {
			c := trigger[i]

			// Handle escape sequences in strings
			if inString && c == '\\' && i+1 < len(trigger) {
				i++ // Skip the escaped character
				continue
			}

			if c == '"' {
				if !inString {
					inString = true
					stringStart = i + 1
				} else {
					// End of string
					stringEnd := i
					inString = false

					// Only consider keys at depth 1 (top-level object)
					if depth == 1 {
						// Skip whitespace after the closing quote
						j := i + 1
						for j < len(trigger) && (trigger[j] == ' ' || trigger[j] == '\t') {
							j++
						}
						// Check if this is a key (followed by ':')
						if j < len(trigger) && trigger[j] == ':' {
							events = append(events, trigger[stringStart:stringEnd])
						}
					}
					stringStart = -1
				}
			} else if !inString {
				if c == '{' {
					depth++
				} else if c == '}' {
					depth--
				}
			}
		}
		return events
	}

	// Simple comma-separated list
	parts := strings.Split(trigger, ",")
	events := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			events = append(events, p)
		}
	}
	return events
}

// parseFlashesFromHTML extracts flash messages from OOB swap HTML.
// Looks for patterns like: <div class="toast toast-success" ...>message</div>
func parseFlashesFromHTML(html string) []Flash {
	var flashes []Flash

	// Find all toast divs
	const prefix = `<div class="toast toast-`
	idx := 0
	for {
		start := strings.Index(html[idx:], prefix)
		if start == -1 {
			break
		}
		start += idx + len(prefix)

		// Extract level (until the next quote)
		levelEnd := strings.Index(html[start:], `"`)
		if levelEnd == -1 {
			break
		}
		level := html[start : start+levelEnd]

		// Find the closing > of the opening tag
		tagEnd := strings.Index(html[start:], ">")
		if tagEnd == -1 {
			break
		}
		contentStart := start + tagEnd + 1

		// Find the closing </div>
		contentEnd := strings.Index(html[contentStart:], "</div>")
		if contentEnd == -1 {
			break
		}
		message := html[contentStart : contentStart+contentEnd]

		flashes = append(flashes, Flash{
			Level:   level,
			Message: message,
		})

		idx = contentStart + contentEnd
	}

	return flashes
}

// TestRequestBuilder provides a fluent interface for building test requests.
//
// Use this when you need fine-grained control over request construction:
//
//	result, err := hxcmp.NewTestRequest("POST", actionURL).
//	    WithFormData("name", "value").
//	    WithHeader("X-Custom", "header").
//	    WithContext(ctx).
//	    Execute(comp)
type TestRequestBuilder struct {
	method   string
	url      string
	formData map[string]string
	headers  map[string]string
	ctx      context.Context
}

// NewTestRequest creates a new test request builder.
func NewTestRequest(method, url string) *TestRequestBuilder {
	return &TestRequestBuilder{
		method:   method,
		url:      url,
		formData: make(map[string]string),
		headers:  make(map[string]string),
		ctx:      context.Background(),
	}
}

// WithFormData adds form data to the request.
func (b *TestRequestBuilder) WithFormData(key, value string) *TestRequestBuilder {
	b.formData[key] = value
	return b
}

// WithFormValues adds multiple form values to the request.
func (b *TestRequestBuilder) WithFormValues(data map[string]string) *TestRequestBuilder {
	for k, v := range data {
		b.formData[k] = v
	}
	return b
}

// WithHeader adds a header to the request.
func (b *TestRequestBuilder) WithHeader(key, value string) *TestRequestBuilder {
	b.headers[key] = value
	return b
}

// WithContext sets the context for the request.
func (b *TestRequestBuilder) WithContext(ctx context.Context) *TestRequestBuilder {
	b.ctx = ctx
	return b
}

// Execute executes the request against an HXComponent.
func (b *TestRequestBuilder) Execute(comp HXComponent) (*TestResult, error) {
	// Build form body
	form := url.Values{}
	for k, v := range b.formData {
		form.Set(k, v)
	}

	var body *strings.Reader
	if len(b.formData) > 0 {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}

	req := httptest.NewRequest(b.method, b.url, body)
	req = req.WithContext(b.ctx)

	// Set default HTMX header
	req.Header.Set("HX-Request", "true")

	// Set content type if form data present
	if len(b.formData) > 0 {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Set custom headers
	for k, v := range b.headers {
		req.Header.Set(k, v)
	}

	// Record response
	rec := httptest.NewRecorder()
	comp.HXServeHTTP(rec, req)

	result := &TestResult{
		HTML:       rec.Body.String(),
		StatusCode: rec.Code,
		Headers:    rec.Header(),
	}

	// Parse triggered events
	if trigger := rec.Header().Get("HX-Trigger"); trigger != "" {
		result.TriggeredEvents = parseTriggerHeader(trigger)
	}

	// Parse redirect
	if redirect := rec.Header().Get("HX-Redirect"); redirect != "" {
		result.RedirectURL = redirect
	}

	// Parse flashes
	result.Flashes = parseFlashesFromHTML(result.HTML)

	return result, nil
}

// MockHydrater wraps a component and provides a custom hydration function.
//
// Useful for injecting test data without needing real dependencies like
// databases or external services:
//
//	mockHydrate := func(ctx context.Context, props *Props) error {
//	    props.Repo = testRepo  // Inject test data
//	    return nil
//	}
//	mock := hxcmp.NewMockHydrater(comp, mockHydrate)
//	result, err := hxcmp.TestRender(mock, props)
type MockHydrater[P any] struct {
	Component    TestableComponent[P]
	HydrateFunc  func(ctx context.Context, props *P) error
	hydrateProps *P // Store last hydrated props for assertions
}

// NewMockHydrater creates a MockHydrater that wraps a component.
func NewMockHydrater[P any](comp TestableComponent[P], hydrateFn func(ctx context.Context, props *P) error) *MockHydrater[P] {
	return &MockHydrater[P]{
		Component:   comp,
		HydrateFunc: hydrateFn,
	}
}

// Hydrate calls the custom hydrate function.
func (m *MockHydrater[P]) Hydrate(ctx context.Context, props *P) error {
	m.hydrateProps = props
	return m.HydrateFunc(ctx, props)
}

// Render delegates to the underlying component.
func (m *MockHydrater[P]) Render(ctx context.Context, props P) templ.Component {
	return m.Component.Render(ctx, props)
}

// LastHydratedProps returns the props from the last Hydrate call.
//
// Useful for verifying that hydration was called with expected props:
//
//	mock.Hydrate(ctx, &props)
//	lastProps := mock.LastHydratedProps()
//	if lastProps.ID != expectedID { ... }
func (m *MockHydrater[P]) LastHydratedProps() *P {
	return m.hydrateProps
}
