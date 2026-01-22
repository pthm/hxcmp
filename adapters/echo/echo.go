// Package hxcmpecho provides Echo framework integration for hxcmp components.
//
// Mount components onto an Echo instance or group:
//
//	e := echo.New()
//	reg := hxcmpecho.Mount(e)
//	reg.Add(myComponent)
//
// Or mount on a group with middleware:
//
//	g := e.Group("/app", authMiddleware)
//	reg := hxcmpecho.MountGroup(g)
//	reg.Add(myComponent)
package hxcmpecho

import (
	"crypto/rand"
	"fmt"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/pthm/hxcmp"
)

// Option configures the Mount and MountGroup functions.
type Option func(*options)

type options struct {
	key  []byte
	path string
}

// WithKey sets the encryption key for the registry.
// The key should be at least 32 bytes of cryptographically random data.
// If not provided, a random key is generated (suitable for development only).
func WithKey(key []byte) Option {
	return func(o *options) {
		o.key = key
	}
}

// WithPath sets the URL path prefix for component routes.
// Defaults to "/_c/".
func WithPath(path string) Option {
	return func(o *options) {
		o.path = path
	}
}

// Mount creates a registry and mounts the component handler on an Echo instance.
//
//	e := echo.New()
//	reg := hxcmpecho.Mount(e)
//	reg.Add(myComponent)
//
//	// With options:
//	reg := hxcmpecho.Mount(e, hxcmpecho.WithKey(key))
func Mount(e *echo.Echo, opts ...Option) *hxcmp.Registry {
	reg := newRegistry(opts)
	e.Any(reg.path+"*", echo.WrapHandler(reg.Handler()))
	return reg.Registry
}

// MountGroup creates a registry and mounts the component handler on an Echo group.
// This allows components to share middleware with the group (auth, logging, etc.).
//
//	g := e.Group("/app", authMiddleware)
//	reg := hxcmpecho.MountGroup(g)
//	reg.Add(myComponent)
func MountGroup(g *echo.Group, opts ...Option) *hxcmp.Registry {
	reg := newRegistry(opts)
	g.Any(reg.path+"*", echo.WrapHandler(reg.Handler()))
	return reg.Registry
}

type mountedRegistry struct {
	*hxcmp.Registry
	path string
}

func newRegistry(opts []Option) *mountedRegistry {
	o := &options{path: "/_c/"}
	for _, opt := range opts {
		opt(o)
	}

	key := o.key
	if key == nil {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			panic(fmt.Sprintf("hxcmpecho: failed to generate random key: %v", err))
		}
	}

	reg := hxcmp.NewRegistry(key)
	hxcmp.SetDefault(reg)

	return &mountedRegistry{Registry: reg, path: o.path}
}

// Render writes a templ component to the Echo response.
//
//	func handler(c echo.Context) error {
//	    return hxcmpecho.Render(c, myTemplate())
//	}
func Render(c echo.Context, component templ.Component) error {
	c.Response().Header().Set("Content-Type", "text/html; charset=utf-8")
	return component.Render(c.Request().Context(), c.Response())
}
