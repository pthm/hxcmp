package hxcmp

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
)

// Hydrater is implemented by components to reconstruct rich objects
// from serialized IDs. Called automatically before any handler.
type Hydrater[P any] interface {
	Hydrate(ctx context.Context, props *P) error
}

// Renderer is implemented by components to produce templ output.
// Called for GET requests and after successful action handlers.
type Renderer[P any] interface {
	Render(ctx context.Context, props P) templ.Component
}

// HXComponent is implemented by generated code.
// This interface allows the registry to dispatch requests.
type HXComponent interface {
	HXPrefix() string
	HXServeHTTP(w http.ResponseWriter, r *http.Request)
}
