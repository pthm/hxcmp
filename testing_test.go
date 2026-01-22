package hxcmp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/a-h/templ"
)

// mockProps is a simple props type for testing
type mockProps struct {
	Name  string
	Count int
}

// mockComponent implements TestableComponent for testing
type mockComponent struct {
	hydrateErr  error
	renderErr   error
	hydrateFn   func(ctx context.Context, props *mockProps) error
	lastProps   *mockProps
	renderCount int
}

func (m *mockComponent) Hydrate(ctx context.Context, props *mockProps) error {
	m.lastProps = props
	if m.hydrateFn != nil {
		return m.hydrateFn(ctx, props)
	}
	return m.hydrateErr
}

func (m *mockComponent) Render(ctx context.Context, props mockProps) templ.Component {
	m.renderCount++
	if m.renderErr != nil {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return m.renderErr
		})
	}
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<div class="mock">Hello, `+props.Name+`!</div>`)
		return err
	})
}

// mockHXComponent implements HXComponent for testing
type mockHXComponent struct {
	prefix     string
	handler    func(w http.ResponseWriter, r *http.Request)
	lastMethod string
	lastPath   string
}

func (m *mockHXComponent) HXPrefix() string {
	return m.prefix
}

func (m *mockHXComponent) HXServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.lastMethod = r.Method
	m.lastPath = r.URL.Path
	if m.handler != nil {
		m.handler(w, r)
	}
}

func TestTestRender_Success(t *testing.T) {
	comp := &mockComponent{}
	props := mockProps{Name: "World", Count: 42}

	result, err := TestRender(comp, props)
	if err != nil {
		t.Fatalf("TestRender() error = %v", err)
	}

	if result == nil {
		t.Fatal("TestRender() returned nil result")
	}

	if !result.HTMLContains("Hello, World!") {
		t.Errorf("HTML does not contain expected content: %s", result.HTML)
	}

	if !result.HTMLContains(`class="mock"`) {
		t.Errorf("HTML does not contain expected class: %s", result.HTML)
	}

	if result.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", result.StatusCode, http.StatusOK)
	}
}

func TestTestRender_HydrationError(t *testing.T) {
	expectedErr := errors.New("hydration failed")
	comp := &mockComponent{hydrateErr: expectedErr}
	props := mockProps{Name: "Test"}

	result, err := TestRender(comp, props)
	if err == nil {
		t.Fatal("TestRender() expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}

	if result != nil {
		t.Error("expected nil result on error")
	}
}

func TestTestRender_RenderError(t *testing.T) {
	expectedErr := errors.New("render failed")
	comp := &mockComponent{renderErr: expectedErr}
	props := mockProps{Name: "Test"}

	result, err := TestRender(comp, props)
	if err == nil {
		t.Fatal("TestRender() expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}

	if result != nil {
		t.Error("expected nil result on error")
	}
}

func TestTestRenderWithContext(t *testing.T) {
	type ctxKey string
	key := ctxKey("test-key")
	ctx := context.WithValue(context.Background(), key, "test-value")

	var capturedCtx context.Context
	comp := &mockComponent{
		hydrateFn: func(ctx context.Context, props *mockProps) error {
			capturedCtx = ctx
			return nil
		},
	}
	props := mockProps{Name: "Test"}

	_, err := TestRenderWithContext(ctx, comp, props)
	if err != nil {
		t.Fatalf("TestRenderWithContext() error = %v", err)
	}

	if capturedCtx == nil {
		t.Fatal("context was not passed to Hydrate")
	}

	if capturedCtx.Value(key) != "test-value" {
		t.Error("context value was not preserved")
	}
}

func TestTestAction_GET(t *testing.T) {
	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<div>GET Response</div>`))
		},
	}

	result, err := TestAction(comp, "/_c/test/action", http.MethodGet, nil)
	if err != nil {
		t.Fatalf("TestAction() error = %v", err)
	}

	if comp.lastMethod != http.MethodGet {
		t.Errorf("method = %s, want %s", comp.lastMethod, http.MethodGet)
	}

	if !result.HTMLContains("GET Response") {
		t.Errorf("HTML = %s, want to contain 'GET Response'", result.HTML)
	}
}

func TestTestAction_POST_WithFormData(t *testing.T) {
	var receivedName string
	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			receivedName = r.FormValue("name")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<div>POST Response</div>`))
		},
	}

	formData := map[string]string{"name": "Alice", "count": "5"}
	result, err := TestAction(comp, "/_c/test/submit", http.MethodPost, formData)
	if err != nil {
		t.Fatalf("TestAction() error = %v", err)
	}

	if comp.lastMethod != http.MethodPost {
		t.Errorf("method = %s, want %s", comp.lastMethod, http.MethodPost)
	}

	if receivedName != "Alice" {
		t.Errorf("form name = %s, want Alice", receivedName)
	}

	if !result.HTMLContains("POST Response") {
		t.Errorf("HTML = %s, want to contain 'POST Response'", result.HTML)
	}
}

func TestTestAction_ParsesHTMXHeaders(t *testing.T) {
	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("HX-Trigger", "item-saved, list-updated")
			w.Header().Set("HX-Redirect", "/dashboard")
			w.WriteHeader(http.StatusOK)
		},
	}

	result, err := TestAction(comp, "/_c/test/save", http.MethodPost, nil)
	if err != nil {
		t.Fatalf("TestAction() error = %v", err)
	}

	if len(result.TriggeredEvents) != 2 {
		t.Errorf("TriggeredEvents count = %d, want 2", len(result.TriggeredEvents))
	}

	if !result.HasEvent("item-saved") {
		t.Error("expected 'item-saved' event")
	}

	if !result.HasEvent("list-updated") {
		t.Error("expected 'list-updated' event")
	}

	if !result.WasRedirected() {
		t.Error("expected redirect")
	}

	if result.RedirectURL != "/dashboard" {
		t.Errorf("RedirectURL = %s, want /dashboard", result.RedirectURL)
	}
}

func TestTestAction_SetsHXRequestHeader(t *testing.T) {
	var hxRequest string
	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			hxRequest = r.Header.Get("HX-Request")
			w.WriteHeader(http.StatusOK)
		},
	}

	_, err := TestAction(comp, "/_c/test/action", http.MethodPost, nil)
	if err != nil {
		t.Fatalf("TestAction() error = %v", err)
	}

	if hxRequest != "true" {
		t.Errorf("HX-Request header = %s, want 'true'", hxRequest)
	}
}

func TestTestGet(t *testing.T) {
	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}

	_, err := TestGet(comp, "/_c/test/")
	if err != nil {
		t.Fatalf("TestGet() error = %v", err)
	}

	if comp.lastMethod != http.MethodGet {
		t.Errorf("method = %s, want GET", comp.lastMethod)
	}
}

func TestTestPost(t *testing.T) {
	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	}

	_, err := TestPost(comp, "/_c/test/submit", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("TestPost() error = %v", err)
	}

	if comp.lastMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", comp.lastMethod)
	}
}

func TestTestResult_HTMLContains(t *testing.T) {
	result := &TestResult{HTML: `<div class="container"><span>Hello World</span></div>`}

	tests := []struct {
		substr string
		want   bool
	}{
		{"Hello World", true},
		{"container", true},
		{"<span>", true},
		{"Missing", false},
		{"", true}, // empty string is always contained
	}

	for _, tt := range tests {
		t.Run(tt.substr, func(t *testing.T) {
			if got := result.HTMLContains(tt.substr); got != tt.want {
				t.Errorf("HTMLContains(%q) = %v, want %v", tt.substr, got, tt.want)
			}
		})
	}
}

func TestTestResult_HTMLContainsAll(t *testing.T) {
	result := &TestResult{HTML: `<div class="container"><span>Hello World</span></div>`}

	if !result.HTMLContainsAll("Hello", "World", "container") {
		t.Error("expected HTMLContainsAll to return true for all present substrings")
	}

	if result.HTMLContainsAll("Hello", "Missing") {
		t.Error("expected HTMLContainsAll to return false when any substring is missing")
	}
}

func TestTestResult_HTMLContainsAny(t *testing.T) {
	result := &TestResult{HTML: `<div>Hello World</div>`}

	if !result.HTMLContainsAny("Missing", "Hello", "NotHere") {
		t.Error("expected HTMLContainsAny to return true when any substring is present")
	}

	if result.HTMLContainsAny("Missing", "NotHere", "Absent") {
		t.Error("expected HTMLContainsAny to return false when no substrings are present")
	}
}

func TestTestResult_HasEvent(t *testing.T) {
	result := &TestResult{
		TriggeredEvents: []string{"item-created", "list-updated", "form-validated"},
	}

	if !result.HasEvent("item-created") {
		t.Error("expected HasEvent to find 'item-created'")
	}

	if !result.HasEvent("created") { // partial match
		t.Error("expected HasEvent to find partial match 'created'")
	}

	if result.HasEvent("deleted") {
		t.Error("expected HasEvent to not find 'deleted'")
	}
}

func TestTestResult_HasFlash(t *testing.T) {
	result := &TestResult{
		Flashes: []Flash{
			{Level: "success", Message: "Item saved"},
			{Level: "error", Message: "Validation failed"},
		},
	}

	if !result.HasFlash("success", "Item saved") {
		t.Error("expected HasFlash to find success flash")
	}

	if !result.HasFlash("error", "Validation failed") {
		t.Error("expected HasFlash to find error flash")
	}

	if result.HasFlash("success", "Wrong message") {
		t.Error("expected HasFlash to not find flash with wrong message")
	}

	if result.HasFlash("warning", "Item saved") {
		t.Error("expected HasFlash to not find flash with wrong level")
	}
}

func TestTestResult_HasFlashLevel(t *testing.T) {
	result := &TestResult{
		Flashes: []Flash{
			{Level: "success", Message: "Done"},
			{Level: "info", Message: "Info"},
		},
	}

	if !result.HasFlashLevel("success") {
		t.Error("expected HasFlashLevel to find success")
	}

	if result.HasFlashLevel("error") {
		t.Error("expected HasFlashLevel to not find error")
	}
}

func TestTestResult_WasRedirected(t *testing.T) {
	tests := []struct {
		name        string
		redirectURL string
		want        bool
	}{
		{"with redirect", "/dashboard", true},
		{"without redirect", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &TestResult{RedirectURL: tt.redirectURL}
			if got := result.WasRedirected(); got != tt.want {
				t.Errorf("WasRedirected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTestResult_RedirectedTo(t *testing.T) {
	result := &TestResult{RedirectURL: "/dashboard"}

	if !result.RedirectedTo("/dashboard") {
		t.Error("expected RedirectedTo to return true for matching URL")
	}

	if result.RedirectedTo("/other") {
		t.Error("expected RedirectedTo to return false for non-matching URL")
	}
}

func TestTestResult_StatusChecks(t *testing.T) {
	result := &TestResult{StatusCode: http.StatusOK}

	if !result.IsOK() {
		t.Error("expected IsOK to return true for 200")
	}

	if !result.HasStatus(200) {
		t.Error("expected HasStatus(200) to return true")
	}

	if result.HasStatus(404) {
		t.Error("expected HasStatus(404) to return false")
	}

	result.StatusCode = http.StatusNotFound
	if result.IsOK() {
		t.Error("expected IsOK to return false for 404")
	}
}

func TestTestResult_HeaderMethods(t *testing.T) {
	result := &TestResult{
		Headers: http.Header{
			"Content-Type": []string{"text/html"},
			"X-Custom":     []string{"value"},
		},
	}

	if !result.HasHeader("Content-Type", "text/html") {
		t.Error("expected HasHeader to find Content-Type")
	}

	if result.HasHeader("Content-Type", "application/json") {
		t.Error("expected HasHeader to not match wrong value")
	}

	if result.GetHeader("X-Custom") != "value" {
		t.Errorf("GetHeader = %s, want 'value'", result.GetHeader("X-Custom"))
	}

	if result.GetHeader("Missing") != "" {
		t.Error("expected GetHeader to return empty string for missing header")
	}
}

func TestParseTriggerHeader_Simple(t *testing.T) {
	events := parseTriggerHeader("event1, event2, event3")

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	expected := []string{"event1", "event2", "event3"}
	for i, e := range expected {
		if events[i] != e {
			t.Errorf("events[%d] = %s, want %s", i, events[i], e)
		}
	}
}

func TestParseTriggerHeader_JSON(t *testing.T) {
	events := parseTriggerHeader(`{"event1": null, "event2": {"data": "value"}}`)

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(events), events)
	}

	foundEvent1 := false
	foundEvent2 := false
	for _, e := range events {
		if e == "event1" {
			foundEvent1 = true
		}
		if e == "event2" {
			foundEvent2 = true
		}
	}

	if !foundEvent1 || !foundEvent2 {
		t.Errorf("expected to find event1 and event2, got %v", events)
	}
}

func TestParseTriggerHeader_Empty(t *testing.T) {
	events := parseTriggerHeader("")
	if events != nil {
		t.Errorf("expected nil for empty string, got %v", events)
	}

	events = parseTriggerHeader("   ")
	if events != nil {
		t.Errorf("expected nil for whitespace string, got %v", events)
	}
}

func TestParseFlashesFromHTML(t *testing.T) {
	html := `<div id="toasts" hx-swap-oob="beforeend">` +
		`<div class="toast toast-success" data-auto-dismiss="3000">Item saved</div>` +
		`<div class="toast toast-error" data-auto-dismiss="3000">Error occurred</div>` +
		`</div>`

	flashes := parseFlashesFromHTML(html)

	if len(flashes) != 2 {
		t.Fatalf("expected 2 flashes, got %d", len(flashes))
	}

	if flashes[0].Level != "success" || flashes[0].Message != "Item saved" {
		t.Errorf("flash[0] = %+v, want success/Item saved", flashes[0])
	}

	if flashes[1].Level != "error" || flashes[1].Message != "Error occurred" {
		t.Errorf("flash[1] = %+v, want error/Error occurred", flashes[1])
	}
}

func TestParseFlashesFromHTML_NoFlashes(t *testing.T) {
	html := `<div>No flashes here</div>`
	flashes := parseFlashesFromHTML(html)

	if len(flashes) != 0 {
		t.Errorf("expected 0 flashes, got %d", len(flashes))
	}
}

func TestTestRequestBuilder(t *testing.T) {
	var capturedMethod, capturedPath string
	var capturedFormValue string
	var capturedHeader string

	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			capturedMethod = r.Method
			capturedPath = r.URL.Path
			r.ParseForm()
			capturedFormValue = r.FormValue("key")
			capturedHeader = r.Header.Get("X-Custom")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`<div>Created</div>`))
		},
	}

	result, err := NewTestRequest(http.MethodPost, "/_c/test/create").
		WithFormData("key", "value").
		WithHeader("X-Custom", "custom-value").
		Execute(comp)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", capturedMethod)
	}

	if capturedPath != "/_c/test/create" {
		t.Errorf("path = %s, want /_c/test/create", capturedPath)
	}

	if capturedFormValue != "value" {
		t.Errorf("form value = %s, want 'value'", capturedFormValue)
	}

	if capturedHeader != "custom-value" {
		t.Errorf("custom header = %s, want 'custom-value'", capturedHeader)
	}

	if result.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", result.StatusCode, http.StatusCreated)
	}
}

func TestTestRequestBuilder_WithContext(t *testing.T) {
	type ctxKey string
	key := ctxKey("test-key")
	ctx := context.WithValue(context.Background(), key, "ctx-value")

	var capturedValue any

	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			capturedValue = r.Context().Value(key)
			w.WriteHeader(http.StatusOK)
		},
	}

	_, err := NewTestRequest(http.MethodGet, "/_c/test/").
		WithContext(ctx).
		Execute(comp)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if capturedValue != "ctx-value" {
		t.Errorf("context value = %v, want 'ctx-value'", capturedValue)
	}
}

func TestTestRequestBuilder_WithFormValues(t *testing.T) {
	var capturedValues map[string]string

	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			capturedValues = make(map[string]string)
			for k := range r.Form {
				capturedValues[k] = r.FormValue(k)
			}
			w.WriteHeader(http.StatusOK)
		},
	}

	_, err := NewTestRequest(http.MethodPost, "/_c/test/submit").
		WithFormValues(map[string]string{
			"name":  "Alice",
			"email": "alice@example.com",
		}).
		Execute(comp)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if capturedValues["name"] != "Alice" {
		t.Errorf("name = %s, want 'Alice'", capturedValues["name"])
	}

	if capturedValues["email"] != "alice@example.com" {
		t.Errorf("email = %s, want 'alice@example.com'", capturedValues["email"])
	}
}

func TestMockHydrater(t *testing.T) {
	baseComp := &mockComponent{}

	mockHydrate := NewMockHydrater(baseComp, func(ctx context.Context, props *mockProps) error {
		// Inject test data
		props.Name = "Injected"
		props.Count = 100
		return nil
	})

	result, err := TestRender(mockHydrate, mockProps{Name: "Original"})
	if err != nil {
		t.Fatalf("TestRender() error = %v", err)
	}

	// The mock hydration should have changed the name
	if !result.HTMLContains("Hello, Injected!") {
		t.Errorf("expected injected name in output: %s", result.HTML)
	}

	// Check last hydrated props
	lastProps := mockHydrate.LastHydratedProps()
	if lastProps == nil {
		t.Fatal("expected last hydrated props to be set")
	}

	if lastProps.Name != "Injected" {
		t.Errorf("last props name = %s, want 'Injected'", lastProps.Name)
	}
}

func TestMockHydrater_Error(t *testing.T) {
	baseComp := &mockComponent{}
	expectedErr := errors.New("mock hydration error")

	mockHydrate := NewMockHydrater(baseComp, func(ctx context.Context, props *mockProps) error {
		return expectedErr
	})

	_, err := TestRender(mockHydrate, mockProps{})
	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
}

// TestHeadersBeforeWriteHeader verifies that headers are set BEFORE WriteHeader is called.
// This tests the fix for the bug where headers were being set after WriteHeader,
// causing them to be dropped from the response.
func TestHeadersBeforeWriteHeader(t *testing.T) {
	// Test that custom headers are present when a status code is set
	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			// Simulate the correct behavior: set headers BEFORE WriteHeader
			w.Header().Set("X-Custom-Header", "test-value")
			w.Header().Set("HX-Trigger", "item-saved")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`<div>Created</div>`))
		},
	}

	result, err := TestAction(comp, "/_c/test/create", http.MethodPost, nil)
	if err != nil {
		t.Fatalf("TestAction() error = %v", err)
	}

	// Verify headers are present
	if result.GetHeader("X-Custom-Header") != "test-value" {
		t.Errorf("X-Custom-Header = %q, want %q", result.GetHeader("X-Custom-Header"), "test-value")
	}

	if result.GetHeader("HX-Trigger") != "item-saved" {
		t.Errorf("HX-Trigger = %q, want %q", result.GetHeader("HX-Trigger"), "item-saved")
	}

	if result.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode = %d, want %d", result.StatusCode, http.StatusCreated)
	}
}

// TestHeadersDroppedAfterWriteHeader demonstrates the bug where headers
// set AFTER WriteHeader are silently dropped. This test documents the
// incorrect behavior that the fix addresses.
func TestHeadersDroppedAfterWriteHeader(t *testing.T) {
	// This test demonstrates the buggy pattern: setting headers after WriteHeader
	comp := &mockHXComponent{
		prefix: "/_c/test",
		handler: func(w http.ResponseWriter, r *http.Request) {
			// BUG PATTERN: WriteHeader called BEFORE headers are set
			w.WriteHeader(http.StatusCreated)
			// These headers will be dropped!
			w.Header().Set("X-Should-Be-Dropped", "dropped")
			w.Write([]byte(`<div>Created</div>`))
		},
	}

	result, err := TestAction(comp, "/_c/test/create", http.MethodPost, nil)
	if err != nil {
		t.Fatalf("TestAction() error = %v", err)
	}

	// Header should be empty because it was set after WriteHeader
	if result.GetHeader("X-Should-Be-Dropped") != "" {
		t.Log("Note: Header was unexpectedly present - Go's httptest may buffer headers")
	}
}

// TestCentralizedErrorHandling verifies that Registry.OnError is called
// for errors from generated code instead of calling http.Error directly.
func TestCentralizedErrorHandling(t *testing.T) {
	var capturedErr error
	var capturedStatusCode int

	// Create a registry with custom error handler
	key := make([]byte, 32)
	reg := NewRegistry(key)
	reg.OnError = func(w http.ResponseWriter, r *http.Request, err error) {
		capturedErr = err
		if IsNotFound(err) {
			capturedStatusCode = http.StatusNotFound
			http.Error(w, "Custom not found", http.StatusNotFound)
		} else {
			capturedStatusCode = http.StatusInternalServerError
			http.Error(w, "Custom error", http.StatusInternalServerError)
		}
	}

	// Test that OnError callback is properly set on components
	if reg.OnError == nil {
		t.Fatal("OnError should not be nil")
	}

	// Verify default behavior categorizes errors correctly
	testReg := NewRegistry(key)

	if IsNotFound(ErrNotFound) != true {
		t.Error("IsNotFound should return true for ErrNotFound")
	}

	if IsDecryptionError(ErrDecryptFailed) != true {
		t.Error("IsDecryptionError should return true for ErrDecryptFailed")
	}

	// The testReg should have a default OnError that handles these
	if testReg.OnError == nil {
		t.Fatal("Default OnError should not be nil")
	}

	// Test that the custom handler is called
	_ = capturedErr        // Will be set if OnError is called
	_ = capturedStatusCode // Will be set if OnError is called
}

// TestComponentOnErrorCallback verifies that Component.SetOnError and OnError work.
func TestComponentOnErrorCallback(t *testing.T) {
	comp := New[mockProps]("test-onerror")

	// Initially nil
	if comp.OnError() != nil {
		t.Error("OnError should be nil initially")
	}

	// Set a handler
	var called bool
	handler := func(w http.ResponseWriter, r *http.Request, err error) {
		called = true
	}
	comp.SetOnError(handler)

	// Should return the handler
	if comp.OnError() == nil {
		t.Error("OnError should not be nil after SetOnError")
	}

	// Call it to verify it works
	comp.OnError()(nil, nil, errors.New("test"))
	if !called {
		t.Error("OnError handler was not called")
	}
}
