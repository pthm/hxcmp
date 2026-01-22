package hxcmp

import (
	"errors"
	"testing"
)

type testResultProps struct {
	ID   int
	Name string
}

func TestResultOK(t *testing.T) {
	props := testResultProps{ID: 1, Name: "test"}
	r := OK(props)

	if r.GetProps().ID != 1 {
		t.Errorf("GetProps().ID = %d, want %d", r.GetProps().ID, 1)
	}
	if r.GetProps().Name != "test" {
		t.Errorf("GetProps().Name = %q, want %q", r.GetProps().Name, "test")
	}
	if r.GetErr() != nil {
		t.Errorf("GetErr() = %v, want nil", r.GetErr())
	}
	if r.ShouldSkip() {
		t.Error("ShouldSkip() = true, want false")
	}
	if r.GetRedirect() != "" {
		t.Errorf("GetRedirect() = %q, want empty", r.GetRedirect())
	}
}

func TestResultErr(t *testing.T) {
	props := testResultProps{ID: 1}
	testErr := errors.New("test error")
	r := Err(props, testErr)

	if r.GetErr() != testErr {
		t.Errorf("GetErr() = %v, want %v", r.GetErr(), testErr)
	}
	if r.GetProps().ID != 1 {
		t.Errorf("GetProps().ID = %d, want %d", r.GetProps().ID, 1)
	}
}

func TestResultSkip(t *testing.T) {
	r := Skip[testResultProps]()

	if !r.ShouldSkip() {
		t.Error("ShouldSkip() = false, want true")
	}
}

func TestResultRedirect(t *testing.T) {
	r := Redirect[testResultProps]("/new-location")

	if r.GetRedirect() != "/new-location" {
		t.Errorf("GetRedirect() = %q, want %q", r.GetRedirect(), "/new-location")
	}
}

func TestResultFlash(t *testing.T) {
	props := testResultProps{ID: 1}
	r := OK(props).
		Flash(FlashSuccess, "Item saved").
		Flash(FlashError, "But something else failed")

	flashes := r.GetFlashes()
	if len(flashes) != 2 {
		t.Fatalf("len(GetFlashes()) = %d, want 2", len(flashes))
	}

	if flashes[0].Level != FlashSuccess {
		t.Errorf("flashes[0].Level = %q, want %q", flashes[0].Level, FlashSuccess)
	}
	if flashes[0].Message != "Item saved" {
		t.Errorf("flashes[0].Message = %q, want %q", flashes[0].Message, "Item saved")
	}

	if flashes[1].Level != FlashError {
		t.Errorf("flashes[1].Level = %q, want %q", flashes[1].Level, FlashError)
	}
	if flashes[1].Message != "But something else failed" {
		t.Errorf("flashes[1].Message = %q, want %q", flashes[1].Message, "But something else failed")
	}
}

func TestResultCallback(t *testing.T) {
	props := testResultProps{ID: 1}
	cb := Callback{
		URL:    "/parent/refresh",
		Target: "#parent",
		Swap:   "outerHTML",
	}
	r := OK(props).Callback(cb)

	got := r.GetCallback()
	if got == nil {
		t.Fatal("GetCallback() = nil, want non-nil")
	}
	if got.URL != "/parent/refresh" {
		t.Errorf("GetCallback().URL = %q, want %q", got.URL, "/parent/refresh")
	}
	if got.Target != "#parent" {
		t.Errorf("GetCallback().Target = %q, want %q", got.Target, "#parent")
	}
}

func TestResultTrigger(t *testing.T) {
	props := testResultProps{ID: 1}
	r := OK(props).Trigger("itemUpdated")

	if r.GetTrigger() != "itemUpdated" {
		t.Errorf("GetTrigger() = %q, want %q", r.GetTrigger(), "itemUpdated")
	}
}

func TestResultHeader(t *testing.T) {
	props := testResultProps{ID: 1}
	r := OK(props).
		Header("X-Custom-Header", "custom-value").
		Header("X-Another", "another-value")

	headers := r.GetHeaders()
	if headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("Header X-Custom-Header = %q, want %q", headers["X-Custom-Header"], "custom-value")
	}
	if headers["X-Another"] != "another-value" {
		t.Errorf("Header X-Another = %q, want %q", headers["X-Another"], "another-value")
	}
}

func TestResultStatus(t *testing.T) {
	props := testResultProps{ID: 1}
	r := OK(props).Status(201)

	if r.GetStatus() != 201 {
		t.Errorf("GetStatus() = %d, want %d", r.GetStatus(), 201)
	}
}

func TestResultChaining(t *testing.T) {
	props := testResultProps{ID: 1, Name: "test"}
	r := OK(props).
		Flash(FlashSuccess, "Saved!").
		Trigger("itemSaved").
		Header("X-Item-ID", "1").
		Status(201)

	if len(r.GetFlashes()) != 1 {
		t.Error("Flash not set")
	}
	if r.GetTrigger() != "itemSaved" {
		t.Error("Trigger not set")
	}
	if r.GetHeaders()["X-Item-ID"] != "1" {
		t.Error("Header not set")
	}
	if r.GetStatus() != 201 {
		t.Error("Status not set")
	}
	if r.GetProps().ID != 1 {
		t.Error("Props lost during chaining")
	}
}

func TestResultDefaultValues(t *testing.T) {
	props := testResultProps{ID: 1}
	r := OK(props)

	if r.GetErr() != nil {
		t.Error("Default error should be nil")
	}
	if r.GetRedirect() != "" {
		t.Error("Default redirect should be empty")
	}
	if len(r.GetFlashes()) != 0 {
		t.Error("Default flashes should be empty")
	}
	if r.GetTrigger() != "" {
		t.Error("Default trigger should be empty")
	}
	if r.GetCallback() != nil {
		t.Error("Default callback should be nil")
	}
	if len(r.GetHeaders()) != 0 {
		t.Error("Default headers should be empty")
	}
	if r.GetStatus() != 0 {
		t.Error("Default status should be 0")
	}
	if r.ShouldSkip() {
		t.Error("Default skip should be false")
	}
}
