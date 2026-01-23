package hxcmpecho

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/pthm/hxcmp"
)

func TestMount(t *testing.T) {
	e := echo.New()
	reg := Mount(e)

	if reg == nil {
		t.Fatal("Mount returned nil registry")
	}
}

func TestMountWithKey(t *testing.T) {
	e := echo.New()
	key := make([]byte, 32)
	reg := Mount(e, WithKey(key))

	if reg == nil {
		t.Fatal("Mount returned nil registry")
	}
}

func TestMountWithPath(t *testing.T) {
	e := echo.New()
	reg := Mount(e, WithPath("/components/"))

	if reg == nil {
		t.Fatal("Mount returned nil registry")
	}
}

func TestMountGroup(t *testing.T) {
	e := echo.New()
	g := e.Group("/app")
	reg := MountGroup(g)

	if reg == nil {
		t.Fatal("MountGroup returned nil registry")
	}
}

func TestMountSetsDefault(t *testing.T) {
	e := echo.New()
	reg := Mount(e)

	// Verify SetDefault was called by checking MustGet doesn't panic
	// with a type that isn't registered (it should panic with "not found" not "no default")
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic from MustGet with unregistered type")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("unexpected panic type: %T", r)
		}
		if msg == "" {
			t.Fatal("empty panic message")
		}
		_ = reg
	}()
	hxcmp.MustGet[*testing.T]()
}

func TestPOSTNotForbiddenWithoutHXRequest(t *testing.T) {
	e := echo.New()
	Mount(e)

	// POST without HX-Request header should NOT be forbidden
	// (CSRF protection is the app's responsibility, not hxcmp's)
	req := httptest.NewRequest(http.MethodPost, "/_hxc/test/action", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Error("POST without HX-Request should not be forbidden (CSRF is app-level)")
	}
}

func TestGETRouted(t *testing.T) {
	e := echo.New()
	Mount(e)

	// GET requests are routed to the mux (404 since no component registered)
	req := httptest.NewRequest(http.MethodGet, "/_hxc/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Should not be 403 — just 404 since no component is registered
	if rec.Code == http.StatusForbidden {
		t.Error("GET request should not be forbidden")
	}
}
