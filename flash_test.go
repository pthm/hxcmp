package hxcmp

import (
	"strings"
	"testing"
)

func TestFlashLevelConstants(t *testing.T) {
	// Ensure constants have expected values
	if FlashSuccess != "success" {
		t.Errorf("FlashSuccess = %q, want %q", FlashSuccess, "success")
	}
	if FlashError != "error" {
		t.Errorf("FlashError = %q, want %q", FlashError, "error")
	}
	if FlashWarning != "warning" {
		t.Errorf("FlashWarning = %q, want %q", FlashWarning, "warning")
	}
	if FlashInfo != "info" {
		t.Errorf("FlashInfo = %q, want %q", FlashInfo, "info")
	}
}

func TestRenderFlashesOOBEmpty(t *testing.T) {
	result := RenderFlashesOOB(nil)
	if result != "" {
		t.Errorf("RenderFlashesOOB(nil) = %q, want empty string", result)
	}

	result = RenderFlashesOOB([]Flash{})
	if result != "" {
		t.Errorf("RenderFlashesOOB([]) = %q, want empty string", result)
	}
}

func TestRenderFlashesOOBSingle(t *testing.T) {
	flashes := []Flash{
		{Level: FlashSuccess, Message: "Item saved successfully"},
	}

	result := RenderFlashesOOB(flashes)

	// Check for OOB swap wrapper
	if !strings.Contains(result, `id="toasts"`) {
		t.Error("Missing id=\"toasts\"")
	}
	if !strings.Contains(result, `hx-swap-oob="beforeend"`) {
		t.Error("Missing hx-swap-oob=\"beforeend\"")
	}

	// Check for toast content
	if !strings.Contains(result, `class="toast toast-success"`) {
		t.Error("Missing toast-success class")
	}
	if !strings.Contains(result, `data-auto-dismiss="3000"`) {
		t.Error("Missing data-auto-dismiss")
	}
	if !strings.Contains(result, "Item saved successfully") {
		t.Error("Missing flash message")
	}
}

func TestRenderFlashesOOBMultiple(t *testing.T) {
	flashes := []Flash{
		{Level: FlashSuccess, Message: "First message"},
		{Level: FlashError, Message: "Second message"},
		{Level: FlashWarning, Message: "Third message"},
	}

	result := RenderFlashesOOB(flashes)

	// Should have one container
	if strings.Count(result, `id="toasts"`) != 1 {
		t.Error("Should have exactly one toasts container")
	}

	// Should have three toast divs
	if strings.Count(result, `class="toast`) != 3 {
		t.Error("Should have three toast elements")
	}

	// Check each level
	if !strings.Contains(result, "toast-success") {
		t.Error("Missing toast-success")
	}
	if !strings.Contains(result, "toast-error") {
		t.Error("Missing toast-error")
	}
	if !strings.Contains(result, "toast-warning") {
		t.Error("Missing toast-warning")
	}
}

func TestRenderFlashesOOBHTMLEscaping(t *testing.T) {
	flashes := []Flash{
		{Level: FlashError, Message: "<script>alert('xss')</script>"},
	}

	result := RenderFlashesOOB(flashes)

	// Should NOT contain raw script tag
	if strings.Contains(result, "<script>") {
		t.Error("HTML should be escaped - found raw <script> tag")
	}

	// Should contain escaped version
	if !strings.Contains(result, "&lt;script&gt;") {
		t.Error("HTML should be escaped - missing &lt;script&gt;")
	}
}

func TestRenderFlashesOOBLevelEscaping(t *testing.T) {
	// Even the level should be escaped (though it shouldn't contain HTML in practice)
	flashes := []Flash{
		{Level: "<bad>", Message: "test"},
	}

	result := RenderFlashesOOB(flashes)

	if strings.Contains(result, `class="toast toast-<bad>"`) {
		t.Error("Level should be escaped")
	}
}

func TestRenderFlashesOOBStructure(t *testing.T) {
	flashes := []Flash{
		{Level: FlashInfo, Message: "Test"},
	}

	result := RenderFlashesOOB(flashes)

	// Verify basic structure
	if !strings.HasPrefix(result, "<div") {
		t.Error("Should start with <div")
	}
	if !strings.HasSuffix(result, "</div>") {
		t.Error("Should end with </div>")
	}

	// Verify nesting - outer div contains inner div
	openCount := strings.Count(result, "<div")
	closeCount := strings.Count(result, "</div>")
	if openCount != closeCount {
		t.Errorf("Mismatched div tags: %d opens, %d closes", openCount, closeCount)
	}
}

func TestToastContainer(t *testing.T) {
	// ToastContainer returns a templ.Component, test that it renders
	tc := ToastContainer()
	if tc == nil {
		t.Fatal("ToastContainer() returned nil")
	}

	// We can't easily test the render output without a writer,
	// but we can verify it's not nil
}
