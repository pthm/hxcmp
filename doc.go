// Package hxcmp provides a component system for building server-rendered,
// interactive web applications using Go, Templ templates, and HTMX.
//
// hxcmp enables React-like component composition where components are
// self-contained units with their own templates, handlers, and routes.
// Components are strongly typed via Go generics, eliminating runtime type
// assertions and enabling compile-time verification of action methods.
//
// # Core Concepts
//
// Components embed *Component[P] where P is the Props type. Props must be
// serializable and should contain only IDs or minimal data - rich objects
// are reconstructed during hydration.
//
//	type FileViewer struct {
//	    *hxcmp.Component[Props]
//	    repo *ops.Repo
//	}
//
// The lifecycle is formalized through two required interfaces:
//   - Hydrater[P]: Hydrate(ctx, *P) reconstructs rich objects from IDs
//   - Renderer[P]: Render(ctx, P) produces the templ.Component output
//
// Hydrate runs automatically before any handler, ensuring props are always
// fully populated. Render is called automatically after successful actions.
//
// # Actions and Routing
//
// Actions are registered with semantic names using c.Action():
//
//	c.Action("edit", c.handleEdit)
//	c.Action("delete", c.handleDelete).Method(http.MethodDelete)
//
// Code generation produces typed Wire methods that return templ.Attributes
// with the minimal HTMX attributes (hx-get/hx-post + hx-vals):
//
//	<button { c.WireEdit(props)... } hx-target="#editor" hx-confirm="Save?">
//
// All other HTMX attributes (hx-target, hx-swap, hx-trigger, etc.) are
// written directly in templates, keeping component code HTMX-native.
//
// Each component receives a unique URL prefix based on its name and source
// location hash. The registry prevents prefix collisions at registration time.
//
// # Security Model
//
// Props are encoded in URLs using one of two modes:
//   - Signed (default): HMAC-authenticated JSON, visible but tamper-proof
//   - Encrypted: AES-GCM encrypted, opaque to clients (use .Sensitive())
//
// CSRF protection is automatic - mutating methods (POST/PUT/DELETE/PATCH)
// require the HX-Request: true header that HTMX sends, preventing cross-origin
// attacks without additional tokens.
//
// # Component Communication
//
// Components communicate through events and flash messages:
//
// 1. Events: Broadcast events via HX-Trigger for loose coupling
//
//	// Emitter sends event with data:
//	return hxcmp.OK(props).Trigger("item:saved", map[string]any{"id": item.ID})
//
//	// Listener responds to event in template (raw HTMX):
//	<div { c.WireRender(props)... } hx-trigger="item:saved from:body">
//
// 2. Flash messages: One-time notifications rendered as OOB swaps
//
//	return hxcmp.OK(props).Flash("success", "Saved!")
//
// Events decouple components - the emitter doesn't know who's listening.
//
// # Registration and Routing
//
// Components are registered explicitly with a Registry:
//
//	reg := hxcmp.NewRegistry(encryptionKey)
//	reg.Add(fileViewer, fileBrowser, commitList)
//	http.Handle("/_c/", reg.Handler())
//
// The registry provides centralized error handling via OnError callback
// and ensures components meet interface requirements at registration time,
// not during requests.
//
// # Code Generation
//
// Run 'hxcmp generate' to produce:
//   - Fast encoder/decoder for Props (implements Encodable/Decodable)
//   - Wire methods (e.g., WireEdit, WireDelete) returning templ.Attributes
//   - HXServeHTTP dispatcher that routes requests to handlers
//
// Generated code eliminates reflection in the hot path and enables
// compile-time verification of action names and props types.
//
// # Design Rationale
//
// The system favors explicitness over magic:
//   - Explicit registration (no init() side effects)
//   - Explicit lifecycle (Hydrate/Render interfaces)
//   - Explicit communication (Events, not global state)
//   - Explicit security (Signed vs Encrypted via .Sensitive())
//
// This enables testability, clarity, and strong static guarantees while
// maintaining the flexibility of server-side rendering.
package hxcmp
