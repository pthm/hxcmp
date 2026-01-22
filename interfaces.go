package hxcmp

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
)

// Hydrater is implemented by components to reconstruct rich objects from
// serialized IDs in props. Called automatically before any handler (including
// GET/render).
//
// Hydration transforms lean, serializable props into fully-populated objects
// by fetching from databases, caches, or other sources. This pattern keeps
// URL-encoded props minimal while ensuring handlers always work with complete
// data.
//
// Example:
//
//	func (c *FileViewer) Hydrate(ctx context.Context, props *Props) error {
//	    // props.RepoID is set from URL
//	    props.Repo = c.repo.Get(props.RepoID)  // Fetch rich object
//	    return nil
//	}
//
// Hydrate runs exactly once per request, before any handler is invoked.
// Handlers can safely assume hydrated props are complete.
type Hydrater[P any] interface {
	Hydrate(ctx context.Context, props *P) error
}

// Renderer is implemented by components to produce templ output.
// Called for GET requests and automatically after successful action handlers
// that return OK or Err results.
//
// Render receives fully-hydrated props and should be pure - it reads props
// and produces HTML without side effects.
//
// Example:
//
//	func (c *FileViewer) Render(ctx context.Context, props Props) templ.Component {
//	    return fileViewerTemplate(props)
//	}
//
// If an action handler returns Skip(), Render is not called (the handler
// wrote its own response). If it returns Redirect(), no rendering occurs.
type Renderer[P any] interface {
	Render(ctx context.Context, props P) templ.Component
}

// HXComponent is implemented by generated code to enable the registry to
// dispatch requests without reflection.
//
// User components should not implement this directly - the hxcmp generator
// produces the implementation by generating HXServeHTTP, which decodes props,
// calls Hydrate, routes to the appropriate handler, and calls Render.
//
// HXPrefix returns the unique URL prefix for this component instance.
// HXServeHTTP handles all HTTP requests for the component's routes.
type HXComponent interface {
	HXPrefix() string
	HXServeHTTP(w http.ResponseWriter, r *http.Request)
}
