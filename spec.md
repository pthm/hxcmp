# hxcmp: HTMX Component System for Go

**Status**: Draft v3 (Revised)
**Author**: Engineering Team
**Created**: 2026-01-21
**Last Updated**: 2026-01-21
**Go Version**: Go 1.22+ (generics required)

### Revision Summary (v3)

Key changes from v2:
- **Formalized lifecycle**: Required `Hydrate()` and `Render()` methods - hydration runs automatically before any handler
- **Semantic actions**: `c.Action("edit", handler)` replaces `c.POST("/edit", handler)` - **default POST**, explicit override supported
- **Fluent HTMX builder**: `c.Edit(props).Target("#x").Confirm("Sure?").Attrs()` - exposes HTMX, doesn't hide it
- **Result type**: `Result[P]` replaces `(Props, error)` - fluent builder for flash messages, redirects, callbacks
- **Flash messages**: `hxcmp.OK(props).Flash("success", "Saved!")` with OOB toast rendering
- **Signed vs Encrypted**: Default signed (visible, debuggable); `Sensitive()` for encrypted props
- **Callback props**: `hxcmp.Callback` type enables parent-child communication via `OnSave`, `OnSelect`, etc.
- **Event broadcasting**: `.Trigger("event")` for loose-coupled component communication
- **Lazy loading**: `c.Lazy(props, placeholder)` defers rendering until viewport intersection
- **Simplified handler signatures**: Auto-detected - `(ctx, Props)`, `(ctx, Props, *http.Request)`, or `(ctx, Props, http.ResponseWriter)`

### Previous Changes (v2)

- **Code generation required**: Generates fast encoder/decoder and typed action methods (no reflection in hot path)
- **Typed action methods**: `c.Edit(props)` with compile-time route verification
- **Built-in CSRF protection**: HX-Request header validation, no tokens needed
- **AST-based generation**: Generator parses source with `go/ast` - no compilation required
- **Error handling**: Centralized via `Registry.OnError` callback
- **Build workflow**: `hxcmp generate` → `templ generate` → `go build`

---

## Table of Contents

1. [Overview](#overview)
2. [Goals and Motivation](#goals-and-motivation)
3. [Core Concepts](#core-concepts)
   - [Components](#components)
   - [Props](#props)
   - [Actions and Routes](#actions-and-routes)
   - [Action Handlers](#action-handlers)
   - [Hydration](#hydration-required-lifecycle-method)
   - [Action Builders](#action-builders-fluent-htmx-api)
   - [Callbacks](#callbacks-parent-child-communication)
   - [Events](#events-loose-coupling)
   - [Flash Messages](#flash-messages-toasts)
   - [Lazy Loading](#lazy-loading)
   - [SPA-like Navigation](#spa-like-navigation-hx-boost)
4. [Architecture](#architecture)
5. [API Design](#api-design)
6. [Implementation Details](#implementation-details)
7. [Component Communication Patterns](#component-communication-patterns)
8. [Rationale and Design Decisions](#rationale-and-design-decisions)
9. [Invariants and Guarantees](#invariants-and-guarantees)
10. [Usage Patterns](#usage-patterns)
11. [Security Considerations](#security-considerations)
12. [Performance Characteristics](#performance-characteristics)
13. [Code Generation](#code-generation)
14. [Migration and Adoption](#migration-and-adoption)
15. [Future Directions](#future-directions)

---

## Overview

**hxcmp** is a component system for building server-rendered, interactive web applications using Go, Templ templates, and HTMX. It provides a React-like component model where components are self-contained units with their own templates, handlers, and routes.

The system solves the fundamental challenge of building reusable, composable UI components in a server-side rendering context while maintaining type safety, security, and developer ergonomics.

### Key Characteristics

- **Stdlib-first**: Built on `net/http` with no framework dependencies
- **Lifecycle-driven**: Required `Hydrate()` and `Render()` methods with automatic invocation
- **Semantic actions**: `c.Action("save", handler)` with default POST and fluent HTMX builders
- **Result type**: Fluent `Result[P]` for flash messages, redirects, and callbacks (not error abuse)
- **Code generation**: Fast encoder/decoder, typed action methods with compile-time verification
- **Explicit registration**: Clear dependency injection, no magic
- **Type-safe**: Props strongly typed via generics; action methods catch typos at compile time
- **Callback props**: Parent-child communication via `hxcmp.Callback` type
- **Event broadcasting**: Loose coupling via `.Trigger()` and `OnEvent()` listeners
- **Two security modes**: Signed (visible, debuggable) by default; `Sensitive()` for encrypted props
- **HTMX-native**: Exposes HTMX attributes, doesn't hide them; helpers for common patterns
- **Idiomatic Go**: Fluent builders, receiver methods, standard patterns

---

## Goals and Motivation

### Primary Goals

1. **React-like Composition**: Enable component-driven development similar to React, Vue, or Svelte, but with server-side rendering
2. **Idiomatic Go**: API should feel natural to Go developers, using familiar patterns from stdlib and popular frameworks
3. **Reusability**: Components should be usable across different pages and contexts without modification
4. **Type Safety**: Props should be strongly typed with compile-time verification via generics
5. **Security**: Component parameters in URLs should be tamper-proof and opaque
6. **Explicit over Implicit**: Clear component creation, registration, and dependency injection

### Non-Goals

1. **Client-side State Management**: This is a server-rendering library; client state is managed via HTMX
2. **Framework Lock-in**: While stdlib-first, we don't preclude framework adapters
3. **Virtual DOM**: We render on the server; HTMX handles DOM updates
4. **Streaming/Suspense**: Components render synchronously; async rendering is out of scope
5. **Auto-registration magic**: No init() functions, no global side effects

### Motivation

Traditional Go web frameworks couple UI rendering tightly to handlers. This creates several problems:

**Problem 1: Tight Coupling**
```go
// Traditional approach - page knows about all components
func (h *RepoHandler) Mount(mux *http.ServeMux) {
    mux.HandleFunc("/repo", h.handleRepo)
    h.fileViewer.Mount(mux)      // Handler must mount components
    h.fileBrowser.Mount(mux)
    h.commitList.Mount(mux)
}
```

**Problem 2: Route Collisions**
```go
// Components mounted on same path can collide
h.fileViewer.Mount(g, "/file")   // Registers /repo/file
h.fileBrowser.Mount(g, "/file")  // Collision!
```

**Problem 3: Non-Reusable Components**
```go
// Component tightly coupled to specific page structure
type FileViewer struct {
    basePath string  // Must know parent path
}
```

hxcmp solves these problems with:
1. **Explicit registration**: Components register with a central registry
2. **Template composition**: Templates receive component instances and render them
3. **Deterministic routing**: Components get unique prefixes automatically
4. **Context-aware URLs**: Components derive base paths from requests
5. **Dependency injection**: Components receive their dependencies explicitly

---

## Core Concepts

### Components

A **component** is a self-contained unit consisting of:
- **Props struct**: Defines what data the component needs (typed via generics)
- **Component struct**: Embeds `*hxcmp.Component[Props]` and holds dependencies
- **Hydrate method**: Reconstructs rich objects from serialized IDs (required)
- **Render method**: Returns a Templ template for display (required) and is used for both initial page renders and HTMX partials
- **Action handlers**: Named handlers for component interactions
- **Callbacks**: Optional props for parent notification

Components are created explicitly via constructors and registered with a registry.

**Example**:
```go
type FileViewer struct {
    *hxcmp.Component[Props]  // Embed typed component
    repo *ops.Repo            // Dependencies
}

func New(repo *ops.Repo) *FileViewer {
    c := &FileViewer{
        Component: hxcmp.New[Props]("fileviewer"),
        repo:      repo,
    }

    // Register actions - semantic names, default POST
    c.Action("edit", c.handleEdit)                                // POST (mutating)
    c.Action("delete", c.handleDelete).Method(http.MethodDelete)  // explicit method
    c.Action("raw", c.handleRaw).Method(http.MethodGet)           // explicit method

    return c
}

// Required: Hydrate reconstructs rich objects before any handler
func (c *FileViewer) Hydrate(ctx context.Context, props *Props) error {
    if props.Repo == nil && props.RepoID > 0 {
        repo, err := c.repo.GetByID(ctx, props.RepoID)
        if err != nil {
            return fmt.Errorf("hydrate repo: %w", err)
        }
        props.Repo = repo
    }
    return nil
}

// Required: Render returns the templ component
func (c *FileViewer) Render(ctx context.Context, props Props) templ.Component {
    return Template(c, props)
}
```

**Render behavior**: `Render` is a pure function that returns a `templ.Component`. On initial page loads, parents embed the returned component in larger templates. On HTMX requests, hxcmp renders just this component as the partial response. User code does not branch on request type.

### Props

**Props** are typed data structures passed to components. They contain:
- **Serializable fields**: Scalar values encoded into URLs for HTMX requests
- **Rich objects**: Complex types (structs, pointers) used during server-side rendering but not serialized
- **Callbacks**: Optional `hxcmp.Callback` fields for parent notification

Props use the `hx` tag to control serialization:
```go
type Props struct {
    // Serialized fields (sent in encrypted URL params)
    RepoID int64  `hx:"r"`        // Serialized with key "r"
    Path   string `hx:"p"`        // Serialized with key "p"

    // Rich objects (hydrated server-side, not serialized)
    Repo   *mdl.Repository `hx:"-"`  // Explicitly excluded

    // Callbacks (serialized, used for parent communication)
    OnSave hxcmp.Callback `hx:"cb,omitempty"` // Optional callback to parent
}
```

**Tag Rules**:
1. `hx:"key"` - Serialize with the specified key
2. `hx:"key,omitempty"` - Serialize only if non-zero
3. `hx:"-"` - Explicitly exclude from serialization
4. No tag + scalar type → Auto-include with lowercase field name
5. No tag + complex type (pointer, struct, interface) → Auto-exclude
6. `hxcmp.Callback` type → Always serialized (signed or encrypted reference to target action)

**Pragmatic guidance (v3)**:
- Prefer explicit `hx:"..."` tags for stable, long-lived props so field renames don’t break URLs.
- Keep serialized props small (IDs over large objects); hydrate the rest.

### Actions and Routes

Each component gets a **deterministic prefix** based on:
- Component name (for observability)
- Source location (file and line number of `New[Props]()` call)

Example: `/_c/fileviewer-a1b2c3d4`

**Uniqueness & collisions**:
- Prefix is derived from component name + source location hash.
- If two components resolve to the same prefix, registry registration fails with a clear error.
- To intentionally create multiple instances, construct them from different call sites or provide a distinct name.

Actions are registered using semantic names:
```go
c.Action("edit", c.handleEdit)                                // POST /_c/fileviewer-a1b2c3d4/edit
c.Action("delete", c.handleDelete).Method(http.MethodDelete)  // DELETE /_c/fileviewer-a1b2c3d4/delete
c.Action("raw", c.handleRaw).Method(http.MethodGet)           // GET /_c/fileviewer-a1b2c3d4/raw (non-mutating)
```

**Default method**: All actions are **POST** by default.

**Override with explicit method**:
```go
c.Action("archive", c.handleArchive).Method(http.MethodPatch)
```

### Action Handlers

Action handlers return `Result[P]` using a fluent builder pattern. The system auto-detects which signature you use:

```go
// Signature 1: Minimal - just context and props (most common)
func(ctx context.Context, props P) Result[P]

// Signature 2: With request - for form data, headers, etc.
func(ctx context.Context, props P, r *http.Request) Result[P]

// Signature 3: With writer - for custom responses (raw files, downloads)
func(ctx context.Context, props P, w http.ResponseWriter) Result[P]
```

**Result constructors**:
- `hxcmp.OK(props)` - Success, auto-render with props
- `hxcmp.Err(props, err)` - Error, handled by registry's OnError
- `hxcmp.Skip[Props]()` - Handler wrote response, no auto-render
- `hxcmp.Redirect[Props](url)` - Redirect via HX-Redirect header

**Result methods** (fluent chaining):
- `.Flash("success", "Saved!")` - Show toast notification
- `.Callback(cb)` - Trigger parent callback
- `.Trigger("event")` - Emit event via HX-Trigger
- `.Header("X-Custom", "value")` - Set response header
- `.Status(code)` - Set HTTP status code (e.g., 201, 422)

**Example** (no hydration boilerplate - `Hydrate()` runs automatically):
```go
func (c *FileViewer) handleEdit(ctx context.Context, props Props, r *http.Request) hxcmp.Result[Props] {
    // props.Repo is guaranteed non-nil (Hydrate ran first)
    content := r.FormValue("content")
    if err := props.Repo.WriteFile(ctx, props.Path, content); err != nil {
        return hxcmp.Err(props, err)
    }

    // Update props for re-render
    props.LastModified = time.Now()

    // Build result with flash and optional callback
    result := hxcmp.OK(props).Flash("success", "File saved!")
    if !props.OnSave.IsZero() {
        result = result.Callback(props.OnSave)
    }
    return result
}

func (c *FileViewer) handleDelete(ctx context.Context, props Props) hxcmp.Result[Props] {
    if err := props.Repo.DeleteFile(ctx, props.Path); err != nil {
        return hxcmp.Err(props, err)
    }

    return hxcmp.Redirect[Props]("/repos/" + props.Repo.Name).
        Flash("success", "File deleted")
}

func (c *FileViewer) handleRaw(ctx context.Context, props Props, w http.ResponseWriter) hxcmp.Result[Props] {
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte(props.File.Content))
    return hxcmp.Skip[Props]()
}
```

### Hydration (Required Lifecycle Method)

**Hydration** is the process of reconstructing rich objects from serialized IDs. It's a **required method** that runs automatically before `Render()` or any action handler.

```go
// Required interface method
func (c *FileViewer) Hydrate(ctx context.Context, props *Props) error {
    // Only hydrate if needed (HTMX request won't have Repo)
    if props.Repo == nil && props.RepoID > 0 {
        repo, err := c.repo.GetByID(ctx, props.RepoID)
        if err != nil {
            return fmt.Errorf("hydrate repo: %w", err)
        }
        props.Repo = repo
    }
    return nil
}
```

**Key points**:
- Takes `*Props` (pointer) to modify in place
- Called automatically before any handler - you never call it directly
- **Initial renders**: Hydration still runs; props can already be populated (no-op)
- **HTMX requests**: Props usually have only IDs; hydration reconstructs rich objects
- Errors propagate to registry's `OnError` handler

**Lifecycle flow**:
```
Request (initial render or HTMX)
    │
    ├─▶ Decode props from signed/encrypted params
    │
    ├─▶ Hydrate(ctx, &props)  ◀── Always runs first
    │       │
    │       └─▶ Error? → OnError handler, stop
    │
    ├─▶ Route to handler (Render or action)
    │
    └─▶ Return HTML (auto-render or custom)
```

### Action Builders (Fluent HTMX API)

Actions are invoked via **fluent builders** created by the code generator. Instead of raw HTMX attributes, use typed methods:

```go
// Generated methods - typos caught at compile time
<button {...c.Edit(props).Attrs()}>Edit</button>
<button {...c.Eidt(props).Attrs()}>Edit</button>  // Compile error!
```

Each action registration generates a corresponding builder method:
- `c.Action("edit", ...)` → `c.Edit(props)` returns `*Action`
- `c.Action("delete", ...)` → `c.Delete(props)` returns `*Action`

**Fluent configuration**:
```templ
// Simple: defaults to self-targeting with outerHTML swap
<button {...c.Edit(props).Attrs()}>Edit</button>

// With target and confirmation
<button {...c.Delete(props).Target("#file-list").Confirm("Delete file?").Attrs()}>
    Delete
</button>

// As a link (for GET actions)
<a {...c.Raw(props).AsLink()}>View Raw</a>

// Polling
<div {...c.Refresh(props).Every(5 * time.Second).Attrs()}>
    Auto-refreshing content
</div>
```

Templates receive the component instance to access these builders.

### Callbacks (Parent-Child Communication)

**Callbacks** enable child components to notify parents (or any known target) of events. This supports “bubble up” flows and sibling updates by passing callbacks down the tree. A callback is a **signed or encrypted reference** to a target component action. The callback payload **inherits the target component’s security mode** (signed vs encrypted).

```go
// Child component accepts optional callback
type Props struct {
    IssueID  int64          `hx:"i"`
    OnSubmit hxcmp.Callback `hx:"cb,omitempty"` // Optional: notify parent on submit
}

// Parent passes callback when rendering child
@commentForm.Render(ctx, commentform.Props{
    IssueID:  issue.ID,
    OnSubmit: commentList.Refresh(listProps).Target("#comment-list").AsCallback(),
})
```

**Triggering callbacks**:
```go
func (c *CommentForm) handleSubmit(ctx context.Context, props Props, r *http.Request) hxcmp.Result[Props] {
    // Create comment...

    // Notify parent if callback provided
    if !props.OnSubmit.IsZero() {
        return hxcmp.OK(props).Callback(props.OnSubmit)
    }
    return hxcmp.OK(props)
}
```

**What happens**:
1. The child returns `.Callback(cb)` and the response includes `HX-Trigger` with a structured payload.
2. A tiny built-in HTMX extension (shipped by hxcmp) listens for `hxcmp:callback` events and **issues the callback request** using HTMX semantics.
3. The callback request uses the target action URL and respects `Target` / `Swap` provided in the callback.
4. The extension is optional and keeps the client dependency surface to HTMX + a small plugin.

**Callback wiring (extension contract)**:
The server emits `HX-Trigger` with a JSON payload like:
```json
{
  "hxcmp:callback": {
    "url": "/_c/commentlist-abc123/?p=...",
    "target": "#comment-list",
    "swap": "outerHTML"
  }
}
```

The hxcmp HTMX extension listens for this event and issues the request:
```html
<script src="/static/htmx.min.js"></script>
<script src="/static/hxcmp-ext.js"></script>
```
```js
// hxcmp-ext.js (conceptual)
document.body.addEventListener("hxcmp:callback", function (evt) {
  var d = evt.detail || {};
  if (!d.url) return;
  htmx.ajax("GET", d.url, {
    target: d.target || "this",
    swap: d.swap || "outerHTML",
  });
});
```

**HTMX-only mode**: You can do callbacks without extra JS by explicitly wiring listeners in the DOM:
```templ
// Parent listens to "comment:submitted" and refreshes itself
<div {...commentList.Refresh(listProps).OnEvent("comment:submitted").Attrs()}>
    ...
</div>
```
In this mode the child uses `.Trigger("comment:submitted")` instead of `.Callback(...)`. Callback is for **directed** updates; events are for **broadcast** updates.

### Events (Loose Coupling)

**Events** enable components to communicate without explicit wiring. Any component can emit events; any component can listen.

```go
// Emitting an event
func (c *CommentForm) handleSubmit(ctx context.Context, props Props, r *http.Request) hxcmp.Result[Props] {
    comment, _ := c.comments.Create(ctx, props.IssueID, r.FormValue("body"))
    return hxcmp.OK(props).Trigger("comment:created")
}
```

```templ
// Listening for events
<div {...c.Refresh(props).OnEvent("comment:created").Attrs()}>
    // This refreshes when any component emits "comment:created"
</div>
```

**Events vs Callbacks**:
| Aspect | Callbacks | Events |
|--------|-----------|--------|
| Coupling | Tight (explicit prop) | Loose (by name) |
| Direction | Child → specific parent | Any → any listeners |
| Use case | "Notify my parent" | "Broadcast to whoever cares" |

### Flash Messages (Toasts)

**Flash messages** are one-time notifications displayed after actions. They use HTMX's out-of-band swap mechanism.

**Setup**: Add a toast container to your layout:
```templ
// layouts/base.templ
<body>
    { children... }

    // Toast container - flashes appear here via OOB swap
    <div id="toasts" class="toast-container"></div>
</body>
```

**Usage in handlers**:
```go
func (c *Editor) handleSave(ctx context.Context, props Props) hxcmp.Result[Props] {
    // ... save ...
    return hxcmp.OK(props).Flash("success", "Changes saved!")
}
```

**How it works**: When a handler returns a flash, the response includes an OOB swap:
```html
<!-- Main component response -->
<div id="file-viewer">...</div>

<!-- OOB: Injected into toast container -->
<div id="toasts" hx-swap-oob="beforeend">
    <div class="toast toast-success" data-auto-dismiss="3000">
        Changes saved!
    </div>
</div>
```

**Styling** (user provides CSS):
```css
.toast-container { position: fixed; top: 1rem; right: 1rem; }
.toast { padding: 1rem; border-radius: 0.5rem; margin-bottom: 0.5rem; }
.toast-success { background: var(--color-success); }
.toast-error { background: var(--color-error); }
```

### Lazy Loading

**Lazy loading** defers component rendering until it enters the viewport. Useful for expensive components below the fold.

```go
// Component method
func (c *Component[P]) Lazy(props P, placeholder templ.Component) templ.Component
```

**Usage**:
```templ
// Expensive chart loads only when scrolled into view
@analyticsChart.Lazy(chartProps, ChartSkeleton())

// Multiple lazy components
<div class="dashboard">
    @revenueWidget.Lazy(props, WidgetSkeleton())
    @userStats.Lazy(props, WidgetSkeleton())
    @activityFeed.Lazy(props, FeedSkeleton())
</div>
```

**Generated HTML**:
```html
<div hx-get="/_c/chart-abc123/?p=..." hx-trigger="intersect once" hx-swap="outerHTML">
    <!-- placeholder content -->
</div>
```

**Deferred loading** (load after page, don't wait for viewport):
```go
func (c *Component[P]) Defer(props P, placeholder templ.Component) templ.Component
// Generates: hx-trigger="load"
```

### SPA-like Navigation (hx-boost)

**hx-boost** converts standard links and forms to AJAX requests, providing faster navigation without full page reloads. This is a native HTMX feature - we just document the pattern.

**Enable on layout**:
```templ
// layouts/base.templ
<body hx-boost="true">
    // All descendant links become AJAX-powered
    { children... }
</body>
```

**How it works**:
1. User clicks a link
2. HTMX intercepts, fetches page via AJAX
3. Replaces `<body>` content (or configured target)
4. Pushes URL to browser history
5. No full page reload - 2x faster perceived navigation

**Selective boost**:
```templ
// Enable on navigation only
<nav hx-boost="true">
    <a href="/repos">Repos</a>      // Boosted
    <a href="/settings">Settings</a> // Boosted
</nav>

// Disable for specific links
<a href="https://external.com" hx-boost="false">External</a>
<a href="/download.pdf" hx-boost="false">Download</a>
```

**Preserve elements across navigation**:
```templ
// Audio player persists across page changes
<div id="audio-player" hx-preserve="true">
    <audio src="..." />
</div>

// Toast container persists
<div id="toasts" class="toast-container" hx-preserve="true"></div>
```

**Server detection**:
```go
// Detect boosted requests
func (h *Handler) handlePage(w http.ResponseWriter, r *http.Request) {
    if r.Header.Get("HX-Boosted") == "true" {
        // Could return partial page, but usually return full page
        // HTMX extracts just the body
    }
    // ... render page
}
```

**Note**: Unlike Livewire's `wire:navigate`, hx-boost is purely an HTMX feature. We don't wrap it - just document the pattern.

---

## Architecture

### System Diagram

```
┌─────────────────────────────────────────────────────────────┐
│  Application (main.go)                                      │
│                                                              │
│  ┌──────────────┐         ┌─────────────────────┐          │
│  │ Registry     │         │ Components          │          │
│  │              │◀────────│                     │          │
│  │ - Add()      │         │ fileviewer.New()    │          │
│  │ - Handler()  │         │ filebrowser.New()   │          │
│  └──────┬───────┘         └─────────────────────┘          │
│         │                                                    │
│         │ mux.Handle("/_c/", registry.Handler())            │
│         ▼                                                    │
│  ┌──────────────┐                                           │
│  │ HTTP Server  │                                           │
│  │              │                                           │
│  └──────────────┘                                           │
└─────────────────────────────────────────────────────────────┘
                 │
                 │ HTTP Request
                 ▼
┌─────────────────────────────────────────────────────────────┐
│  Page Request: /r/alice/repo                                │
│                                                              │
│  ┌──────────────┐                                           │
│  │ Page Handler │                                           │
│  │              │                                           │
│  │ - Fetch repo │                                           │
│  │ - Render()   │                                           │
│  └──────┬───────┘                                           │
│         │                                                    │
│         │ Pass component instance to template               │
│         ▼                                                    │
│  ┌──────────────────┐                                       │
│  │ Templ Template   │                                       │
│  │                  │                                       │
│  │ @fileViewer.Render(ctx, Props{...})                     │
│  └──────────────────┘                                       │
└─────────────────────────────────────────────────────────────┘
                 │
                 │ HTML with hx-post="/_c/fileviewer-abc123/edit?p=..."
                 ▼
┌─────────────────────────────────────────────────────────────┐
│  HTMX Request: /_c/fileviewer-abc123/edit?p=<encrypted>    │
│                                                              │
│  ┌──────────────┐                                           │
│  │ Registry     │                                           │
│  │              │                                           │
│  │ - Decrypt    │                                           │
│  │ - Route      │                                           │
│  └──────┬───────┘                                           │
│         │                                                    │
│         ▼                                                    │
│  ┌──────────────────┐                                       │
│  │ Component        │                                       │
│  │                  │                                       │
│  │ - Decode props   │                                       │
│  │ - Hydrate        │                                       │
│  │ - Handle action  │                                       │
│  │ - Render         │                                       │
│  └──────────────────┘                                       │
└─────────────────────────────────────────────────────────────┘
```

### Request Flow

**Initial Page Render**:
```
1. Browser requests /r/alice/myrepo
2. Page handler fetches repo from database
3. Page handler renders template, passes component instances
4. Template calls @fileViewer.Render(ctx, Props{RepoID: 123, Repo: repo})
5. Component sets itself in context, calls render function
6. Template uses generated URL methods: c.URLEdit(props)
7. HTML returned with HTMX attributes: hx-post="/_c/fileviewer-abc123/edit?p=..."
```

**HTMX Component Update**:
```
1. User clicks button with hx-post="/_c/fileviewer-abc123/edit?p=<encrypted>"
2. HTMX sends request with HX-Request header (required for CSRF protection)
3. Registry validates HX-Request header for mutating methods
4. Registry routes request to fileviewer component's /edit handler
5. Component decodes encrypted props: {RepoID: 123, Path: "src/main.go"}
6. Handler hydrates: fetches repo from database using RepoID
7. Handler processes action (e.g., saves edited file)
8. Handler returns (updatedProps, nil) → auto-render with returned props
9. Component renders and returns just the component HTML
10. HTMX swaps the updated HTML into the page
```

### Component Lifecycle

**Application Start**:
```
main()
  │
  ├─▶ Create dependencies (DB, ops, etc.)
  │
  ├─▶ Create components with dependencies
  │       fileViewer := fileviewer.New(repoOps)
  │       fileBrowser := filebrowser.New(repoOps)
  │
  ├─▶ Create registry
  │       registry := hxcmp.NewRegistry(encKey)
  │
  ├─▶ Register components
  │       registry.Add(fileViewer, fileBrowser)
  │
  ├─▶ Create pages with component instances
  │       repoPage := repo.New(repoOps, fileViewer, fileBrowser)
  │
  ├─▶ Mount pages
  │       repoPage.Mount(mux)
  │
  └─▶ Mount registry
          mux.Handle("/_c/", registry.Handler())
```

**Runtime - Page Request**:
```
Request: /r/alice/repo
  │
  ├─▶ Page handler fetches data
  │
  ├─▶ Template receives component instances + data
  │
  └─▶ @component.Render(ctx, props)
          │
          └─▶ Renders inline (props have rich objects)
```

**Runtime - Component Action**:
```
Request: /_c/fileviewer-abc123/edit?p=<encrypted>
  │
  ├─▶ Registry decrypts params
  │
  ├─▶ Routes to component's /edit handler
  │
  ├─▶ Handler receives typed props
  │
  ├─▶ Handler hydrates rich objects
  │
  ├─▶ Handler processes action
  │
  └─▶ Returns nil → Auto-render
          │
          └─▶ HTML fragment returned
```

---

## API Design

### Component Definition

```go
// components/fileviewer/fileviewer.go
package fileviewer

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/a-h/templ"
    "github.com/yourorg/hxcmp"
)

// Props - the component's data contract
type Props struct {
    // Serialized (sent in encrypted URL params)
    RepoID       int64     `hx:"r"`
    Path         string    `hx:"p"`
    LastModified time.Time `hx:"m,omitempty"`

    // Hydrated (reconstructed server-side)
    Repo *mdl.Repository `hx:"-"`
    File *mdl.File       `hx:"-"`

    // Callbacks (optional parent notifications)
    OnSave hxcmp.Callback `hx:"cb,omitempty"`
}

// FileViewer component struct
type FileViewer struct {
    *hxcmp.Component[Props]  // Embed typed component
    repo *ops.Repo           // Dependencies injected
}

// New creates and configures the component
func New(repo *ops.Repo) *FileViewer {
    c := &FileViewer{
        Component: hxcmp.New[Props]("fileviewer"),
        repo:      repo,
    }

    // Register actions - semantic names, default POST
    c.Action("edit", c.handleEdit)                                // POST (mutating action)
    c.Action("delete", c.handleDelete).Method(http.MethodDelete)  // explicit method
    c.Action("raw", c.handleRaw).Method(http.MethodGet)           // explicit method

    return c
}

// For sensitive components, use Sensitive() to encrypt props
func NewUserSettings(users *ops.User) *UserSettings {
    c := &UserSettings{
        Component: hxcmp.New[Props]("usersettings").Sensitive(), // Props encrypted
        users:     users,
    }
    // ...
    return c
}

// ═══════════════════════════════════════════════════════════════
// Required Lifecycle Methods
// ═══════════════════════════════════════════════════════════════

// Hydrate reconstructs rich objects from serialized IDs.
// Called automatically before Render or any action handler.
func (c *FileViewer) Hydrate(ctx context.Context, props *Props) error {
    if props.Repo == nil && props.RepoID > 0 {
        repo, err := c.repo.GetByID(ctx, props.RepoID)
        if err != nil {
            return fmt.Errorf("hydrate repo: %w", err)
        }
        props.Repo = repo
    }

    // Optionally hydrate file content for render
    if props.File == nil && props.Repo != nil && props.Path != "" {
        file, err := props.Repo.ReadFile(ctx, props.Path)
        if err != nil {
            return fmt.Errorf("hydrate file: %w", err)
        }
        props.File = file
    }

    return nil
}

// Render returns the templ component for display.
// Called for GET requests and after successful action handlers.
func (c *FileViewer) Render(ctx context.Context, props Props) templ.Component {
    // props.Repo and props.File are guaranteed non-nil (Hydrate ran first)
    return Template(c, props)
}

// ═══════════════════════════════════════════════════════════════
// Action Handlers (no hydration boilerplate needed)
// ═══════════════════════════════════════════════════════════════

// handleEdit processes file edits
// Signature: (ctx, Props, *http.Request) -> Result[Props]
func (c *FileViewer) handleEdit(ctx context.Context, props Props, r *http.Request) hxcmp.Result[Props] {
    // props.Repo is guaranteed non-nil (Hydrate ran first)
    content := r.FormValue("content")
    if err := props.Repo.WriteFile(ctx, props.Path, content); err != nil {
        return hxcmp.Err(props, err)
    }

    // Update props for re-render
    props.LastModified = time.Now()
    props.File = nil // Force re-hydration to get updated content

    // Build result with flash and optional callback
    result := hxcmp.OK(props).Flash("success", "File saved!")
    if !props.OnSave.IsZero() {
        result = result.Callback(props.OnSave)
    }
    return result
}

// handleDelete deletes the file
// Signature: (ctx, Props) -> Result[Props]
func (c *FileViewer) handleDelete(ctx context.Context, props Props) hxcmp.Result[Props] {
    if err := props.Repo.DeleteFile(ctx, props.Path); err != nil {
        return hxcmp.Err(props, err)
    }

    // Redirect with flash message
    return hxcmp.Redirect[Props]("/r/" + props.Repo.Owner + "/" + props.Repo.Name).
        Flash("success", "File deleted")
}

// handleRaw returns raw file content (custom response)
// Signature: (ctx, Props, http.ResponseWriter) -> Result[Props]
func (c *FileViewer) handleRaw(ctx context.Context, props Props, w http.ResponseWriter) hxcmp.Result[Props] {
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte(props.File.Content))
    return hxcmp.Skip[Props]() // No auto-render
}
```

### Template Usage

Templates receive the component instance to access fluent action builders:

```templ
// components/fileviewer/template.templ
package fileviewer

templ Template(c *FileViewer, props Props) {
    <div class="file-viewer" id="file-viewer">
        <header class="file-header">
            <span>{props.Path}</span>

            // Fluent action builders - typos caught at compile time
            <button {...c.Edit(props).Confirm("Save changes?").Attrs()}>
                Save
            </button>

            // GET action as link
            <a {...c.Raw(props).AsLink()} target="_blank">
                Raw
            </a>

            // DELETE with confirmation, targets parent container
            <button {...c.Delete(props).Target("#repo-content").Confirm("Delete this file?").Attrs()}>
                Delete
            </button>

            // Refresh just this component
            <button {...c.Refresh(props).Attrs()}>
                Refresh
            </button>
        </header>

        <pre class="file-content">{props.File.Content}</pre>

        // Edit form with explicit target
        <form {...c.Edit(props).Target("#file-viewer").Attrs()}>
            <textarea name="content">{props.File.Content}</textarea>
            <button type="submit">Save</button>
        </form>
    </div>
}
```

**Benefits of fluent builders**:
- `c.Eidt(props)` → compile error (typo caught)
- IDE autocomplete shows available actions and methods
- Type-safe swap modes: `.Swap(hxcmp.SwapOuter)` instead of `"outerHTML"` strings
- Refactoring action names updates all usages

**Available builder methods**:
```go
c.Edit(props)                    // Returns *Action
    .Target("#selector")         // hx-target
    .TargetThis()               // hx-target="this"
    .TargetClosest("form")      // hx-target="closest form"
    .Swap(hxcmp.SwapOuter)      // hx-swap="outerHTML"
    .SwapInner()                // hx-swap="innerHTML"
    .Confirm("Are you sure?")   // hx-confirm
    .Indicator("#spinner")      // hx-indicator
    .PushURL()                  // hx-push-url="true"
    .Vals(map[string]any{...})  // hx-vals
    .Every(5 * time.Second)     // hx-trigger="every 5s"
    .OnEvent("item:updated")    // hx-trigger="item:updated from:body"
    .Attrs()                    // Returns templ.Attributes for spreading
    .AsLink()                   // Returns templ.Attributes with href (for <a>)
    .URL()                      // Returns just the URL string
    .AsCallback()               // Returns hxcmp.Callback for parent props
```

### Page Handler

```go
// pages/repo/handler.go
package repo

import (
    "net/http"
    "github.com/yourorg/hxcmp"
    "github.com/yourorg/app/components/fileviewer"
    "github.com/yourorg/app/components/filebrowser"
)

type Handler struct {
    repo        *ops.Repo
    fileViewer  *fileviewer.FileViewer
    fileBrowser *filebrowser.FileBrowser
}

// New receives component instances via dependency injection
func New(repo *ops.Repo, fv *fileviewer.FileViewer, fb *filebrowser.FileBrowser) *Handler {
    return &Handler{
        repo:        repo,
        fileViewer:  fv,
        fileBrowser: fb,
    }
}

func (h *Handler) Mount(mux *http.ServeMux) {
    mux.HandleFunc("GET /r/{owner}/{name}", h.handleGet)
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
    owner := r.PathValue("owner")
    name := r.PathValue("name")

    // Fetch data
    repo, err := h.repo.GetByOwnerAndName(r.Context(), owner, name)
    if err != nil {
        http.Error(w, "Not found", http.StatusNotFound)
        return
    }

    // Pass components to template
    hxcmp.Render(w, r, RepoPage(repo, h.fileViewer, h.fileBrowser))
}
```

```templ
// pages/repo/page.templ
package repo

import "github.com/yourorg/app/components/fileviewer"
import "github.com/yourorg/app/components/filebrowser"

templ RepoPage(repo *mdl.Repository, fileViewer *fileviewer.FileViewer, fileBrowser *filebrowser.FileBrowser) {
    <html>
        <head>
            <script src="https://unpkg.com/htmx.org"></script>
        </head>
        <body>
            <h1>{repo.Owner}/{repo.Name}</h1>

            <div class="layout">
                <aside id="file-tree">
                    @fileBrowser.Render(ctx, filebrowser.Props{
                        RepoID: repo.ID,
                        Repo:   repo,
                        // Callback: when file selected, refresh the viewer
                        OnSelect: fileViewer.Refresh(fileviewer.Props{
                            RepoID: repo.ID,
                        }).Target("#file-content").AsCallback(),
                    })
                </aside>

                <main id="file-content">
                    @fileViewer.Render(ctx, fileviewer.Props{
                        RepoID: repo.ID,
                        Path:   "README.md",
                        Repo:   repo,
                        // Callback: when file saved, refresh the tree
                        OnSave: fileBrowser.Refresh(filebrowser.Props{
                            RepoID: repo.ID,
                        }).Target("#file-tree").AsCallback(),
                    })
                </main>
            </div>
        </body>
    </html>
}
```

**Callback flow**:
1. User edits file in `fileViewer` and clicks Save
2. `handleEdit` runs, saves file, returns `hxcmp.OK(props).Callback(props.OnSave)`
3. Response includes `HX-Trigger` header with structured callback payload
4. hxcmp's HTMX extension (optional) issues the callback request
5. File tree re-renders with updated modified indicators

### Main Application Setup

```go
// main.go
package main

import (
    "net/http"
    "github.com/yourorg/hxcmp"

    "github.com/yourorg/app/components/fileviewer"
    "github.com/yourorg/app/components/filebrowser"
    "github.com/yourorg/app/pages/repo"
)

func main() {
    // Setup dependencies
    db := setupDB()
    repoOps := ops.NewRepo(db)

    // Create registry with encryption key
    registry := hxcmp.NewRegistry(getEncryptionKey())

    // Create components with dependencies
    fileViewerComp := fileviewer.New(repoOps)
    fileBrowserComp := filebrowser.New(repoOps)

    // Register components explicitly
    registry.Add(
        fileViewerComp,
        fileBrowserComp,
    )

    // Create pages with component instances
    repoPage := repo.New(repoOps, fileViewerComp, fileBrowserComp)

    // Setup HTTP router
    mux := http.NewServeMux()

    // Mount pages
    repoPage.Mount(mux)

    // Mount component registry
    mux.Handle("/_c/", registry.Handler())

    // Start server
    http.ListenAndServe(":8080", mux)
}
```

### Core Types

```go
// ═══════════════════════════════════════════════════════════════
// Component Interface (implemented by user)
// ═══════════════════════════════════════════════════════════════

// Hydrater is implemented by components to reconstruct rich objects
type Hydrater[P any] interface {
    Hydrate(ctx context.Context, props *P) error
}

// Renderer is implemented by components to produce templ output
type Renderer[P any] interface {
    Render(ctx context.Context, props P) templ.Component
}

// ═══════════════════════════════════════════════════════════════
// Component Base Type (embedded by user)
// ═══════════════════════════════════════════════════════════════

// Component[P] is the base type embedded by user components
type Component[P any] struct {
    name      string
    prefix    string
    sensitive bool  // If true, props are encrypted; if false, props are signed
    encoder   *Encoder
    actions   map[string]*actionDef[P]
}

// New creates a new component with the given name and type parameter
// By default, props are signed (visible but tamper-proof)
func New[P any](name string) *Component[P]

// Sensitive marks the component as sensitive, enabling full encryption
// Use for components that handle user IDs, financial data, or anything
// where props should be opaque to clients
func (c *Component[P]) Sensitive() *Component[P]

// Action registers a named action handler
// Returns *ActionBuilder for optional configuration
func (c *Component[P]) Action(name string, handler any) *ActionBuilder

// Refresh returns an action builder for the default render
func (c *Component[P]) Refresh(props P) *Action

// Render renders the component with typed props
// Calls Hydrate (always) then user's Render method
func (c *Component[P]) Render(ctx context.Context, props P) templ.Component

// Prefix returns the component's URL prefix
func (c *Component[P]) Prefix() string

// ═══════════════════════════════════════════════════════════════
// Action Builder (fluent API for HTMX attributes)
// ═══════════════════════════════════════════════════════════════

// Action represents a component action with HTMX configuration
type Action struct {
    url       string
    method    string
    target    string
    swap      SwapMode
    trigger   string
    indicator string
    confirm   string
    pushURL   bool
    vals      map[string]any
}

// ActionBuilder configures action registration (e.g., HTTP method)
type ActionBuilder struct {
    action any
}

// Method overrides the default POST method for an action
func (ab *ActionBuilder) Method(m string) *ActionBuilder

// Targeting
func (a *Action) Target(selector string) *Action     // hx-target="#id"
func (a *Action) TargetThis() *Action                // hx-target="this"
func (a *Action) TargetClosest(sel string) *Action   // hx-target="closest .class"

// Swapping
func (a *Action) Swap(mode SwapMode) *Action         // hx-swap
func (a *Action) SwapOuter() *Action                 // hx-swap="outerHTML"
func (a *Action) SwapInner() *Action                 // hx-swap="innerHTML"

// Triggers
func (a *Action) Every(d time.Duration) *Action      // hx-trigger="every 5s"
func (a *Action) OnEvent(event string) *Action       // hx-trigger="event from:body"
func (a *Action) OnLoad() *Action                    // hx-trigger="load"
func (a *Action) OnIntersect() *Action               // hx-trigger="intersect once"

// UX
func (a *Action) Confirm(msg string) *Action         // hx-confirm
func (a *Action) Indicator(sel string) *Action       // hx-indicator
func (a *Action) PushURL() *Action                   // hx-push-url="true"
func (a *Action) Vals(v map[string]any) *Action      // hx-vals

// Terminal methods
func (a *Action) Attrs() templ.Attributes            // Returns spreadable attributes
func (a *Action) AsLink() templ.Attributes           // Returns href for <a> tags
func (a *Action) URL() string                        // Returns just the URL
func (a *Action) AsCallback() Callback               // Converts to Callback for props

// ═══════════════════════════════════════════════════════════════
// SwapMode (type-safe swap modes)
// ═══════════════════════════════════════════════════════════════

type SwapMode string

const (
    SwapOuter        SwapMode = "outerHTML"   // Replace entire element
    SwapInner        SwapMode = "innerHTML"   // Replace contents
    SwapBeforeEnd    SwapMode = "beforeend"   // Append to contents (inside, at end)
    SwapAfterEnd     SwapMode = "afterend"    // Insert after element (outside, after)
    SwapBeforeBegin  SwapMode = "beforebegin" // Insert before element (outside, before)
    SwapAfterBegin   SwapMode = "afterbegin"  // Prepend to contents (inside, at start)
    SwapDelete       SwapMode = "delete"      // Delete element
    SwapNone         SwapMode = "none"        // No swap
)

// ═══════════════════════════════════════════════════════════════
// Callback (parent-child communication)
// ═══════════════════════════════════════════════════════════════

// Callback is a signed/encrypted reference to a component action
// Used for child-to-parent (or directed) communication
// Security mode inherits the target component (signed vs encrypted)
type Callback struct {
    URL    string `json:"u"`           // Encrypted action URL
    Target string `json:"t,omitempty"` // Target selector
    Swap   string `json:"s,omitempty"` // Swap mode
}

func (cb Callback) IsZero() bool
func (cb Callback) TriggerJSON() string

// ═══════════════════════════════════════════════════════════════
// Result Type (Handler Return)
// ═══════════════════════════════════════════════════════════════

// Result[P] is returned from action handlers
// Replaces the (Props, error) pattern with a fluent builder
type Result[P any] struct {
    props    P
    err      error
    redirect string
    flashes  []Flash
    trigger  string
    callback *Callback
    headers  map[string]string
    status   int
    skip     bool
}

// Constructors
func OK[P any](props P) Result[P]                // Success, auto-render
func Err[P any](props P, err error) Result[P]    // Error, calls OnError
func Skip[P any]() Result[P]                     // Handler wrote response
func Redirect[P any](url string) Result[P]       // Redirect via HX-Redirect

// Fluent methods
func (r Result[P]) Flash(level, message string) Result[P]  // Add flash message
func (r Result[P]) Callback(cb Callback) Result[P]         // Trigger parent callback
func (r Result[P]) Trigger(event string) Result[P]         // Emit event (HX-Trigger)
func (r Result[P]) Header(key, value string) Result[P]     // Set response header
func (r Result[P]) Status(code int) Result[P]              // Set HTTP status code

// Accessors (used by generated code)
func (r Result[P]) GetProps() P
func (r Result[P]) GetErr() error
func (r Result[P]) GetRedirect() string
func (r Result[P]) GetTrigger() string
func (r Result[P]) GetCallback() *Callback
func (r Result[P]) GetHeaders() map[string]string
func (r Result[P]) GetStatus() int
func (r Result[P]) ShouldSkip() bool

// Flash levels
const (
    FlashSuccess = "success"
    FlashError   = "error"
    FlashWarning = "warning"
    FlashInfo    = "info"
)

// Flash represents a one-time notification
type Flash struct {
    Level   string // success, error, warning, info
    Message string
}
```

### Registry

```go
// Registry manages component registration and routing
type Registry struct {
    mux        *http.ServeMux
    encoder    *Encoder
    components map[string]any  // map[prefix]Component[T] of various T

    // Error handler - called when component returns error
    OnError func(http.ResponseWriter, *http.Request, error)
}

// NewRegistry creates a new component registry
func NewRegistry(encryptionKey []byte) *Registry

// Add registers components with the registry
// Validates that components implement Hydrater and Renderer interfaces
// Detects prefix collisions and errors on duplicates
func (reg *Registry) Add(components ...any)

// Handler returns the HTTP handler for component routes
// Includes CSRF protection: mutating methods require HX-Request header
func (reg *Registry) Handler() http.Handler
```

### Helper Functions

```go
// Render writes a Templ component to the HTTP response
func Render(w http.ResponseWriter, r *http.Request, component templ.Component) error

// IsHTMX returns true if the request originated from HTMX
func IsHTMX(r *http.Request) bool
```

Action methods are **generated** from action registrations (see [Code Generation](#code-generation)):
```go
c.Action("edit", c.handleEdit)   // Generates c.Edit(props) *Action
c.Action("delete", c.handleDelete) // Generates c.Delete(props) *Action
```

---

## Implementation Details

### Component Type with Generics

The core `Component[P]` type uses Go generics for type safety:

```go
type Component[P any] struct {
    name    string
    prefix  string
    encoder *Encoder
    actions map[string]*actionDef[P]
}

type actionDef[P any] struct {
    name     string
    method   string // GET, POST, DELETE, etc.
    handler  any    // One of the supported handler signatures
}

// Supported handler signatures (auto-detected):
// 1. func(context.Context, P) Result[P]                    - minimal
// 2. func(context.Context, P, *http.Request) Result[P]    - with request
// 3. func(context.Context, P, http.ResponseWriter) Result[P] - custom response
```

**Type flow**:
```go
// Component creation with type parameter
fileViewer := &FileViewer{
    Component: hxcmp.New[Props]("fileviewer"),  // P = Props
}

// Action registration - handler signature is auto-detected
c.Action("edit", c.handleEdit)  // Accepts multiple signatures

// Render - props are typed
c.Render(ctx, Props{...})  // Compile error if wrong type

// Action invocation via generated methods - typos caught at compile time
c.Edit(Props{...})   // Generated from c.Action("edit", ...)
c.Eidt(Props{...})   // Compile error - no such method!
```

### Action Registration (Semantic Names)

Actions are registered using semantic names with a default **POST** method. Override explicitly if needed.

```go
func (c *Component[P]) Action(name string, handler any) *ActionBuilder {
    // Validate handler signature at registration time
    sig := detectSignature(handler)
    if sig == signatureInvalid {
        panic(fmt.Sprintf("invalid handler signature for action %q", name))
    }

    c.actions[name] = &actionDef[P]{
        name:    name,
        method:  http.MethodPost,
        handler: handler,
    }

    return &ActionBuilder{action: c.actions[name]}
}

// Override method explicitly
func (ab *ActionBuilder) Method(m string) *ActionBuilder {
    ab.action.method = m
    return ab
}
```

### Lifecycle Implementation

The `Render` method orchestrates hydration and rendering:

```go
func (c *Component[P]) Render(ctx context.Context, props P) templ.Component {
    // Always run hydration before rendering
    if h, ok := c.parent.(Hydrater[P]); ok {
        if err := h.Hydrate(ctx, &props); err != nil {
            return errorComponent(err)
        }
    }

    // Get the concrete component (which implements Renderer)
    if r, ok := c.parent.(Renderer[P]); ok {
        return r.Render(ctx, props)
    }

    panic("component does not implement Renderer")
}
```

### Action Builder Implementation

The fluent `Action` type builds HTMX attributes:

```go
type Action struct {
    url       string
    method    string
    target    string
    swap      SwapMode
    trigger   string
    indicator string
    confirm   string
    pushURL   bool
    vals      map[string]any
}

func (a *Action) Target(sel string) *Action {
    a.target = sel
    return a
}

func (a *Action) Swap(mode SwapMode) *Action {
    a.swap = mode
    return a
}

func (a *Action) Confirm(msg string) *Action {
    a.confirm = msg
    return a
}

// Attrs returns HTMX attributes for spreading in templ
func (a *Action) Attrs() templ.Attributes {
    attrs := templ.Attributes{}

    // Set method-specific attribute
    switch a.method {
    case http.MethodGet:
        attrs["hx-get"] = a.url
    case http.MethodPost:
        attrs["hx-post"] = a.url
    case http.MethodDelete:
        attrs["hx-delete"] = a.url
    case http.MethodPut:
        attrs["hx-put"] = a.url
    case http.MethodPatch:
        attrs["hx-patch"] = a.url
    }

    if a.target != "" {
        attrs["hx-target"] = a.target
    }
    if a.swap != "" {
        attrs["hx-swap"] = string(a.swap)
    }
    if a.trigger != "" {
        attrs["hx-trigger"] = a.trigger
    }
    if a.indicator != "" {
        attrs["hx-indicator"] = a.indicator
    }
    if a.confirm != "" {
        attrs["hx-confirm"] = a.confirm
    }
    if a.pushURL {
        attrs["hx-push-url"] = "true"
    }
    if len(a.vals) > 0 {
        data, _ := json.Marshal(a.vals)
        attrs["hx-vals"] = string(data)
    }

    return attrs
}

// AsLink returns attributes suitable for <a> tags
func (a *Action) AsLink() templ.Attributes {
    return templ.Attributes{
        "href": a.url,
    }
}

// AsCallback converts the action to a Callback for passing to child components
func (a *Action) AsCallback() Callback {
    return Callback{
        URL:    a.url,
        Target: a.target,
        Swap:   string(a.swap),
    }
}
```

### Deterministic Prefix Generation

Components get unique prefixes based on name and source location:

```go
func componentHash(name string, skip int) string {
    _, file, line, ok := runtime.Caller(skip + 1)
    var input string
    if ok {
        // Use base filename only for portability
        input = fmt.Sprintf("%s:%d:%s", filepath.Base(file), line, name)
    } else {
        input = name
    }
    h := sha256.Sum256([]byte(input))
    return hex.EncodeToString(h[:4])  // 8 hex chars
}

func New[P any](name string) *Component[P] {
    prefix := "/_c/" + name + "-" + componentHash(name, 1)
    return &Component[P]{
        name:   name,
        prefix: prefix,
        routes: make(map[string]*route[P]),
    }
}
```

### Registry Implementation

The registry manages components and routes requests.

**Reflection usage note**: The registry uses reflection at **startup time** for component registration (finding embedded types, validating interfaces). This is acceptable because it happens once at application start, not per-request. The **hot path** (request handling, encoding/decoding, handler dispatch) uses generated code with zero reflection.

```go
type Registry struct {
    mu         sync.RWMutex
    mux        *http.ServeMux
    encoder    *Encoder
    components map[string]any  // map[prefix]Component[T]

    // Error handler - customizable
    OnError func(http.ResponseWriter, *http.Request, error)
}

func NewRegistry(encryptionKey []byte) *Registry {
    reg := &Registry{
        mux:        http.NewServeMux(),
        encoder:    newEncoder(encryptionKey),
        components: make(map[string]any),
    }

    // Default error handler
    reg.OnError = func(w http.ResponseWriter, r *http.Request, err error) {
        if errors.Is(err, ErrNotFound) {
            http.Error(w, "Not found", http.StatusNotFound)
            return
        }
        http.Error(w, "Internal error", http.StatusInternalServerError)
    }

    return reg
}

func (reg *Registry) Add(components ...any) {
    for _, comp := range components {
        reg.registerComponent(comp)
    }
}

func (reg *Registry) registerComponent(comp any) {
    val := reflect.ValueOf(comp).Elem()

    // Find embedded Component[P] by type, not position
    compField, ok := reg.findEmbeddedComponent(val)
    if !ok {
        panic("component must embed *hxcmp.Component[P]")
    }

    // Validate lifecycle interfaces
    if !implementsHydrater(comp) {
        panic(fmt.Sprintf("%T must implement Hydrate(ctx, *Props) error", comp))
    }
    if !implementsRenderer(comp) {
        panic(fmt.Sprintf("%T must implement Render(ctx, Props) templ.Component", comp))
    }

    // Extract prefix and actions
    prefix := compField.Elem().FieldByName("prefix").String()
    actions := compField.Elem().FieldByName("actions")

    // Register default GET route (for Refresh/Render)
    pattern := "GET " + prefix + "/"
    reg.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
        reg.handleRender(comp, compField, w, r)
    })

    // Register action routes
    for iter := actions.MapRange(); iter.Next(); {
        name := iter.Key().String()
        action := iter.Value()
        method := action.FieldByName("method").String()
        pattern := method + " " + prefix + "/" + name
        reg.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
            reg.handleAction(comp, compField, name, w, r)
        })
    }

    reg.components[prefix] = comp
}

func (reg *Registry) Handler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // CSRF protection: mutating methods require HX-Request header
        if r.Method != "GET" && r.Method != "HEAD" {
            if r.Header.Get("HX-Request") != "true" {
                http.Error(w, "Forbidden: HTMX request required", http.StatusForbidden)
                return
            }
        }

        reg.mux.ServeHTTP(w, r)
    })
}

func (reg *Registry) handleRender(comp any, compField reflect.Value, w http.ResponseWriter, r *http.Request) {
    // Decode props
    props, err := reg.decodeProps(compField, r.URL.Query().Get("p"))
    if err != nil {
        reg.OnError(w, r, err)
        return
    }

    // Hydrate
    if err := callHydrate(comp, r.Context(), props); err != nil {
        reg.OnError(w, r, err)
        return
    }

    // Render
    tmpl := callRender(comp, r.Context(), props)
    tmpl.Render(r.Context(), w)
}

func (reg *Registry) handleAction(comp any, compField reflect.Value, actionName string, w http.ResponseWriter, r *http.Request) {
    // Decode props
    props, err := reg.decodeProps(compField, r.URL.Query().Get("p"))
    if err != nil {
        reg.OnError(w, r, err)
        return
    }

    // Hydrate
    if err := callHydrate(comp, r.Context(), props); err != nil {
        reg.OnError(w, r, err)
        return
    }

    // Call action handler
    result := callAction(comp, compField, actionName, r.Context(), props, w, r)

    if result.GetErr() != nil {
        reg.OnError(w, r, result.GetErr())
        return
    }
    if result.GetStatus() != 0 {
        w.WriteHeader(result.GetStatus())
    }
    for k, v := range result.GetHeaders() {
        w.Header().Set(k, v)
    }
    if result.GetRedirect() != "" {
        w.Header().Set("HX-Redirect", result.GetRedirect())
        return
    }
    // Combine callback and trigger into single HX-Trigger header
    triggers := []string{}
    if cb := result.GetCallback(); cb != nil {
        triggers = append(triggers, cb.TriggerJSON())
    }
    if result.GetTrigger() != "" {
        triggers = append(triggers, result.GetTrigger())
    }
    if len(triggers) > 0 {
        w.Header().Set("HX-Trigger", strings.Join(triggers, ", "))
    }
    if result.ShouldSkip() {
        return
    }

    // Auto-render with updated props
    tmpl := callRender(comp, r.Context(), result.GetProps())
    tmpl.Render(r.Context(), w)
}
```

### Error Handling

Errors are handled centrally:

```go
// Sentinel errors
var (
    ErrNotFound        = errors.New("resource not found")
    ErrHydrationFailed = errors.New("hydration failed")
    ErrDecryptFailed   = errors.New("parameter decryption failed")
)

func (reg *Registry) handleRequest(comp any, compField reflect.Value, w http.ResponseWriter, r *http.Request) {
    // Decode props from encrypted parameter
    encoded := r.URL.Query().Get("p")
    props, err := reg.decodeProps(compField, encoded)
    if err != nil {
        reg.OnError(w, r, fmt.Errorf("%w: %v", ErrDecryptFailed, err))
        return
    }

    // Call handler (via reflection)
    result, err := reg.callHandler(comp, compField, w, r, props)
    if err != nil {
        // Check for SkipRender
        if errors.Is(err, SkipRender) {
            return
        }

        // Type-safe error checking
        var notFound *NotFoundError
        if errors.As(err, &notFound) {
            reg.OnError(w, r, fmt.Errorf("%w: %v", ErrNotFound, notFound))
            return
        }

        reg.OnError(w, r, err)
        return
    }

    // Auto-render with returned props
    reg.renderWithProps(comp, compField, w, r, result)
}
```

### Encoding: Signed vs Encrypted

Props are serialized using the `hx` tag system with two modes:

```go
func (e *Encoder) Encode(v any, sensitive bool) (string, error) {
    // Extract serializable fields via reflection
    serializable := e.extractSerializable(v)

    // Marshal to msgpack (compact binary format)
    data, _ := msgpack.Marshal(serializable)

    if sensitive {
        // Sensitive mode: AES-256-GCM encryption
        nonce := make([]byte, 12)
        rand.Read(nonce)
        ciphertext := e.gcm.Seal(nonce, nonce, data, nil)
        return base64.RawURLEncoding.EncodeToString(ciphertext), nil
    }

    // Default mode: Base64 + HMAC signature
    b64 := base64.RawURLEncoding.EncodeToString(data)
    mac := hmac.New(sha256.New, e.key)
    mac.Write(data)
    sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16]) // 16 bytes = 128 bits
    return b64 + "." + sig, nil
}

func (e *Encoder) Decode(encoded string, sensitive bool, v any) error {
    if sensitive {
        // Decrypt AES-256-GCM
        ciphertext, _ := base64.RawURLEncoding.DecodeString(encoded)
        nonce := ciphertext[:12]
        data, err := e.gcm.Open(nil, nonce, ciphertext[12:], nil)
        if err != nil {
            return fmt.Errorf("decryption failed: %w", err)
        }
        return msgpack.Unmarshal(data, v)
    }

    // Verify HMAC signature
    parts := strings.SplitN(encoded, ".", 2)
    if len(parts) != 2 {
        return errors.New("invalid format")
    }

    data, _ := base64.RawURLEncoding.DecodeString(parts[0])
    sig, _ := base64.RawURLEncoding.DecodeString(parts[1])

    mac := hmac.New(sha256.New, e.key)
    mac.Write(data)
    expected := mac.Sum(nil)[:16]

    if !hmac.Equal(sig, expected) {
        return errors.New("signature verification failed")
    }

    return msgpack.Unmarshal(data, v)
}

// Note: With code generation, extractSerializable is not used at runtime.
// The generator produces typed hxcmpEncode() methods instead.
// This shows the logic the generator implements:

func extractSerializableFields(v any) map[string]any {
    result := make(map[string]any)
    val := reflect.ValueOf(v)

    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }

    t := val.Type()
    for i := 0; i < t.NumField(); i++ {
        sf := t.Field(i)
        fv := val.Field(i)
        tag := sf.Tag.Get("hx")

        // Explicitly excluded
        if tag == "-" {
            continue
        }

        // Determine key and serializability
        var key string
        var serialize bool

        if tag != "" {
            // Explicit tag - always serialize
            key = tag
            serialize = true
        } else if isScalar(fv.Kind()) {
            // Scalar without tag - auto-include with lowercase name
            key = strings.ToLower(sf.Name)
            serialize = true
        } else {
            // Complex type with no tag - skip
            serialize = false
        }

        if serialize {
            result[key] = fv.Interface()
        }
    }

    return result
}

// isScalar determines if a type should be auto-serialized
func isScalar(k reflect.Kind) bool {
    switch k {
    case reflect.Bool,
         reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
         reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
         reflect.Float32, reflect.Float64,
         reflect.String:
        return true
    default:
        return false
    }
}
```

### Scalar Type Reference

The following types are automatically serialized when no `hx` tag is present:

| Category | Types |
|----------|-------|
| Integers | `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64` |
| Floats | `float32`, `float64` |
| Strings | `string` |
| Booleans | `bool` |
| Time | `time.Time` (serialized as RFC3339 string) |
| UUID | `uuid.UUID`, `[16]byte` (serialized as string) |

**Auto-excluded** (unless explicitly tagged):
- Pointers (`*T`)
- Slices (`[]T`)
- Maps (`map[K]V`)
- Structs (nested)
- Interfaces
- Channels, functions

Type aliases of scalar types (e.g., `type UserID int64`) are treated as their underlying type

---

## Rationale and Design Decisions

### Why Explicit Registration Instead of init()?

**Decision**: Components are created via constructors and registered explicitly with a registry.

**Rationale**:
1. **Testability**: Can create isolated component instances for testing
2. **Clarity**: Registration is visible in main.go, easy to understand
3. **Dependency injection**: Components receive dependencies explicitly
4. **No global state**: No hidden side effects from importing packages
5. **Idiomatic Go**: Follows stdlib patterns (http.Server, database/sql, etc.)

**Alternative considered**: Auto-registration via init()
```go
// Rejected: Magic, hard to test, global side effects
var Component *hxcmp.Component

func init() {
    Component = hxcmp.New("fileviewer", getKey(), render)
    hxcmp.GlobalRegistry.Register(Component)  // Ugh, global!
}
```

### Why Semantic Action Names?

**Decision**: Use `c.Action("edit", handler)` instead of `c.POST("/edit", handler)`.

**Rationale**:
1. **Semantic**: "edit" describes intent; POST is implementation detail
2. **Explicitness**: Default POST with explicit override when needed
3. **Generated API**: `c.Edit(props)` is cleaner than `c.URLEdit(props)`
4. **Consistency**: All actions use same registration pattern
5. **Self-documenting**: Action names show component's capabilities

**Comparison with alternatives**:

```go
// ❌ HTTP-method-centric - exposes implementation details
c.POST("/edit", c.handleEdit)
c.DELETE("/delete", c.handleDelete)
c.GET("/raw", c.handleRaw)

// ✅ Semantic names - describes intent
c.Action("edit", c.handleEdit)                        // POST by default
c.Action("delete", c.handleDelete).Method(http.MethodDelete)
c.Action("raw", c.handleRaw).Method(http.MethodGet)
```

### Why Embed `*hxcmp.Component[Props]`?

**Decision**: Component structs embed `*hxcmp.Component[Props]`.

**Rationale**:
1. **Method promotion**: GET, POST, etc. available directly
2. **Type safety**: Props type flows through
3. **Natural**: Feels like struct composition
4. **Clean**: No need to call `c.Component.GET(...)`

**Example**:
```go
type FileViewer struct {
    *hxcmp.Component[Props]  // Embedded
    repo *ops.Repo
}

func New(repo *ops.Repo) *FileViewer {
    c := &FileViewer{
        Component: hxcmp.New[Props]("fileviewer"),
        repo:      repo,
    }

    c.Action("edit", c.handleEdit)  // Action registered on embedded Component

    return c
}
```

### Why Formalized Hydrate/Render Lifecycle?

**Decision**: Components must implement `Hydrate(ctx, *Props)` and `Render(ctx, Props)` methods.

**Rationale**:
1. **DRY**: Hydration logic written once, not repeated in every handler
2. **Automatic**: Framework calls Hydrate before any handler
3. **Clear contract**: Interface makes expectations explicit
4. **Testable**: Can test Hydrate and Render independently

**Pattern**:
```go
// ❌ Before: Hydration duplicated in every handler
func (c *FileViewer) render(ctx context.Context, props Props) (templ.Component, error) {
    if props.Repo == nil { /* hydrate */ }  // Repeated
    return Template(props), nil
}

func (c *FileViewer) handleEdit(ctx context.Context, props Props) hxcmp.Result[Props] {
    if props.Repo == nil { /* hydrate */ }  // Same code again!
    // ... edit logic
    return hxcmp.OK(props)
}

// ✅ After: Hydration in one place, called automatically
func (c *FileViewer) Hydrate(ctx context.Context, props *Props) error {
    if props.Repo == nil && props.RepoID > 0 {
        repo, _ := c.repo.GetByID(ctx, props.RepoID)
        props.Repo = repo
    }
    return nil
}

func (c *FileViewer) handleEdit(ctx context.Context, props Props) hxcmp.Result[Props] {
    // props.Repo is guaranteed non-nil
    // ... edit logic
}
```

### Why Multiple Handler Signatures?

**Decision**: Support three handler signatures, auto-detected:

```go
func(ctx, Props) hxcmp.Result[Props]                 // Most handlers
func(ctx, Props, *http.Request) hxcmp.Result[Props]  // Need form data
func(ctx, Props, http.ResponseWriter) hxcmp.Result[Props] // Custom response
```

**Rationale**:
1. **Minimal surface**: Most handlers don't need w or r
2. **Progressive disclosure**: Add parameters only when needed
3. **Type safety**: Signature tells reader what handler uses
4. **Clean code**: No unused parameters

**Pattern**:
```go
// Simple mutation - no w/r needed
func (c *Counter) handleIncrement(ctx context.Context, props Props) hxcmp.Result[Props] {
    props.Count++
    return hxcmp.OK(props)
}

// Need form data - add request
func (c *Editor) handleSave(ctx context.Context, props Props, r *http.Request) hxcmp.Result[Props] {
    content := r.FormValue("content")
    // ...
    return hxcmp.OK(props)
}

// Custom response - add writer (no auto-render)
func (c *Download) handleDownload(ctx context.Context, props Props, w http.ResponseWriter) hxcmp.Result[Props] {
    w.Header().Set("Content-Type", "application/octet-stream")
    w.Write(props.FileData)
    return hxcmp.Skip[Props]()
}
```

### Why Fluent Action Builders?

**Decision**: Generate fluent builders like `c.Edit(props).Target("#x").Attrs()` instead of raw HTMX attributes.

**Rationale**:
1. **Compile-time safety**: Action and method names checked at compile time
2. **IDE support**: Autocomplete shows available actions and methods
3. **Type-safe constants**: `SwapOuter` instead of `"outerHTML"` strings
4. **Discoverability**: Builder methods document available options
5. **Less boilerplate**: Common patterns become one-liners

**Pattern**:
```templ
// ❌ Before: Raw HTMX attributes, typos not caught
<button
    hx-post={c.URLEdit(props)}
    hx-target="#file-viewer"
    hx-swap="outerHTML"
    hx-confirm="Save?"
>Save</button>

// ✅ After: Fluent builder, IDE-assisted
<button {...c.Edit(props).Target("#file-viewer").Confirm("Save?").Attrs()}>
    Save
</button>

// Typos caught at compile time
<button {...c.Eidt(props).Attrs()}>Save</button>  // Compile error!
```

**Trade-off**: Slightly more verbose for simple cases. Mitigated by sensible defaults:
```templ
// Just `.Attrs()` for default behavior (self-targeting, outerHTML swap)
<button {...c.Edit(props).Attrs()}>Save</button>
```

### Why Callbacks Instead of Just Events?

**Decision**: Support both `hxcmp.Callback` (explicit) and `.Trigger()` (broadcast).

**Rationale**:
1. **Type safety**: Callbacks are typed props, not magic strings
2. **Explicit wiring**: Parent passes callback to child, relationship is visible
3. **Security**: Callback payloads inherit the target component’s security mode
4. **Flexibility**: Events for loose coupling, callbacks for explicit contracts

**Pattern**:
```go
// Parent explicitly passes callback
@child.Render(ctx, child.Props{
    OnSave: parent.Refresh(parentProps).Target("#parent").AsCallback(),
})

// Child triggers it explicitly
if !props.OnSave.IsZero() {
    return hxcmp.OK(props).Callback(props.OnSave)
}
```

**Alternative considered**: Only events (like Livewire's $dispatch)
```go
// Rejected: Loose coupling makes relationships invisible
return hxcmp.OK(props).Trigger("item:saved")  // Who listens? Unknown.
```

**Our approach**: Use callbacks for parent-child, events for broadcasting to unknown listeners.

### Why `hx` Tag System?

**Decision**: Use custom `hx` struct tags with auto-detection.

**Rationale**:
1. **Abstraction**: Users don't need to know about msgpack
2. **Semantics**: `hx:"key"` expresses intent (HTMX param)
3. **Auto-detection**: Scalars auto-include, complex types auto-exclude
4. **Flexibility**: Can change serialization format later

**Tag rules**:
- `hx:"key"` → Serialize with key
- `hx:"-"` → Explicitly exclude
- No tag + scalar → Auto-include (lowercase field name)
- No tag + complex → Auto-exclude

**Example**:
```go
type Props struct {
    RepoID int64  `hx:"r"`        // Explicit key "r"
    Path   string `hx:"p"`        // Explicit key "p"
    Owner  string                 // Auto-include as "owner"
    Repo   *mdl.Repository        // Auto-exclude (pointer)
    Meta   FileMetadata           // Auto-exclude (struct)
    Secret string `hx:"-"`        // Explicitly exclude
}
```

### Why Hydration Pattern?

**Decision**: Rich objects are passed during initial render, fetched during HTMX requests.

**Rationale**:
1. **Efficiency**: Initial render already has objects (no extra fetch)
2. **Type safety**: Props can include rich types
3. **Flexibility**: Component decides what to hydrate and when
4. **Simplicity**: No complex serialization of domain objects

**Pattern**:
```go
func (c *FileViewer) render(ctx context.Context, props Props) (templ.Component, error) {
    // Hydrate if needed (HTMX request)
    if props.Repo == nil && props.RepoID > 0 {
        repo, err := c.repo.GetByID(ctx, props.RepoID)
        if err != nil {
            return nil, fmt.Errorf("hydrate repo: %w", err)
        }
        props.Repo = repo
    }

    return Template(props.Repo), nil
}
```

**Error handling**: Hydration errors propagate through the return value. The registry's `OnError` handler converts them to appropriate HTTP responses

### Why Stdlib-first?

**Decision**: Built on `net/http` with no framework dependencies.

**Rationale**:
1. **Portability**: Works with any Go HTTP framework
2. **Longevity**: stdlib is stable, frameworks change
3. **Simplicity**: Fewer dependencies
4. **Adoption**: Easier to adopt if no framework lock-in

Framework adapters can be added as separate packages.

---

## Invariants and Guarantees

### Lifecycle Invariants

1. **Hydrate always runs first**: `Hydrate()` is called before any handler
   ```go
   func (c *FileViewer) handleEdit(ctx context.Context, props Props) hxcmp.Result[Props] {
       // props.Repo is GUARANTEED non-nil (Hydrate ran first)
       props.Repo.WriteFile(...)  // Safe - no nil check needed
   }
   ```

2. **Render follows actions**: After successful action handlers, `Render()` is called automatically
   ```go
   return hxcmp.OK(props)  // Framework calls c.Render(ctx, props) automatically
   ```

3. **Interface enforcement**: Registry panics if component doesn't implement `Hydrate` and `Render`
   ```go
   // At registration time (not runtime)
   registry.Add(myComponent)  // Panics if interface not satisfied
   ```

### Type Safety Invariants

1. **Compile-time props validation**: Props type is checked at compile time
   ```go
   c.Render(ctx, Props{...})  // Compile error if wrong type
   c.Edit(Props{...})         // Compile error if wrong type
   ```

2. **Compile-time action validation**: Action typos caught at compile time
   ```go
   c.Edit(props)   // Valid - generated from c.Action("edit", ...)
   c.Eidt(props)   // Compile error - method doesn't exist
   ```

3. **Handler signature flexibility**: Multiple signatures supported, auto-detected
   ```go
   c.Action("edit", c.handleEdit)  // Accepts (ctx, P), (ctx, P, r), or (ctx, P, w)
   ```

4. **No type casting**: Props are typed throughout, no `any` casting needed
   ```go
   func (c *FileViewer) handleEdit(ctx context.Context, props Props) hxcmp.Result[Props] {
       // props is Props, not any!
       return hxcmp.OK(props)
   }
   ```

5. **Callback type safety**: Callbacks are typed props, not magic strings
   ```go
   props.OnSave.IsZero()          // Typed method
   hxcmp.OK(props).Callback(props.OnSave)  // Type-checked
   ```

### Security Invariants

1. **Tamper protection**: Encrypted params include GCM auth tag
   - Any modification causes decryption failure
   - Handler receives 400 Bad Request on tampered data

2. **Information hiding**: Params are opaque ciphertext
   - User cannot see IDs, paths, or structure
   - Cannot enumerate by modifying encrypted params

3. **Key per registry**: Encryption key is per-registry
   - Different apps can use different keys
   - Key rotation doesn't break other apps

4. **CSRF protection**: Mutating methods require HTMX
   - POST/PUT/DELETE/PATCH require `HX-Request: true` header
   - Combined with SameSite cookies, prevents cross-origin attacks
   - No additional tokens needed - built into the framework

### Routing Invariants

1. **Prefix uniqueness**: Each component instance gets unique prefix
   - Hash includes source location
   - Registry detects collisions and errors on duplicates

2. **Deterministic routes**: Same code produces same routes
   - URLs are stable across deploys
   - Safe to cache, bookmark, share

3. **No route collisions**: Components can't interfere
   - Each component in own namespace
   - Registry ensures uniqueness

### Concurrency Invariants

1. **Thread-safe registry**: After registration, registry is read-only
   - Safe concurrent access
   - All components registered before serving

2. **Stateless components**: Components don't have mutable state
   - All state is in props (request-scoped)
   - Safe to call concurrently

3. **Thread-safe encoding**: Encoder is safe for concurrent use
   - Each request gets its own nonce
   - GCM operations are thread-safe

---

## Usage Patterns

### Pattern 1: Basic Component

**Use case**: Simple component with default render.

```go
package usercard

type Props struct {
    UserID int64     `hx:"u"`
    User   *mdl.User `hx:"-"`
}

type UserCard struct {
    *hxcmp.Component[Props]
    users *ops.User
}

func New(users *ops.User) *UserCard {
    return &UserCard{
        Component: hxcmp.New[Props]("usercard"),
        users:     users,
    }
}

func (c *UserCard) Hydrate(ctx context.Context, props *Props) error {
    if props.User == nil && props.UserID > 0 {
        user, err := c.users.GetByID(ctx, props.UserID)
        if err != nil {
            return fmt.Errorf("hydrate user: %w", err)
        }
        props.User = user
    }
    return nil
}

func (c *UserCard) Render(ctx context.Context, props Props) templ.Component {
    return Template(props.User)
}
```

### Pattern 2: Component with Actions

**Use case**: Interactive component with form submission and callbacks.

```go
package commentform

type Props struct {
    IssueID  int64          `hx:"i"`
    Issue    *mdl.Issue     `hx:"-"`
    OnSubmit hxcmp.Callback `hx:"cb,omitempty"` // Notify parent on submit
}

type CommentForm struct {
    *hxcmp.Component[Props]
    comments *ops.Comment
    issues   *ops.Issue
}

func New(comments *ops.Comment, issues *ops.Issue) *CommentForm {
    c := &CommentForm{
        Component: hxcmp.New[Props]("commentform"),
        comments:  comments,
        issues:    issues,
    }
    c.Action("submit", c.handleSubmit)
    return c
}

func (c *CommentForm) Hydrate(ctx context.Context, props *Props) error {
    if props.Issue == nil && props.IssueID > 0 {
        issue, err := c.issues.GetByID(ctx, props.IssueID)
        if err != nil {
            return fmt.Errorf("hydrate issue: %w", err)
        }
        props.Issue = issue
    }
    return nil
}

func (c *CommentForm) Render(ctx context.Context, props Props) templ.Component {
    return Template(c, props)
}

func (c *CommentForm) handleSubmit(ctx context.Context, props Props, r *http.Request) hxcmp.Result[Props] {
    body := r.FormValue("body")
    if _, err := c.comments.Create(ctx, props.IssueID, body); err != nil {
        return hxcmp.Err(props, err)
    }

    // Build result with flash and optional callback
    result := hxcmp.OK(props).Flash("success", "Comment posted!")
    if !props.OnSubmit.IsZero() {
        result = result.Callback(props.OnSubmit)
    }
    return result
}
```

```templ
templ Template(c *CommentForm, props Props) {
    <form {...c.Submit(props).Attrs()}>
        <textarea name="body" placeholder="Add a comment..."></textarea>
        <button type="submit">Post</button>
    </form>
}
```

### Pattern 3: Component with Multiple Actions

**Use case**: CRUD operations on a resource.

```go
type FileViewer struct {
    *hxcmp.Component[Props]
    repo *ops.Repo
}

func New(repo *ops.Repo) *FileViewer {
    c := &FileViewer{
        Component: hxcmp.New[Props]("fileviewer"),
        repo:      repo,
    }
    c.Action("edit", c.handleEdit)
    c.Action("delete", c.handleDelete)
    c.Action("raw", c.handleRaw)        // GET (custom response)
    c.Action("editForm", c.showEditForm) // GET (returns different template)
    return c
}

func (c *FileViewer) handleEdit(ctx context.Context, props Props, r *http.Request) hxcmp.Result[Props] {
    content := r.FormValue("content")
    if err := props.Repo.WriteFile(ctx, props.Path, content); err != nil {
        return hxcmp.Err(props, err)
    }
    return hxcmp.OK(props).Flash("success", "Saved!")
}

func (c *FileViewer) handleDelete(ctx context.Context, props Props) hxcmp.Result[Props] {
    if err := props.Repo.DeleteFile(ctx, props.Path); err != nil {
        return hxcmp.Err(props, err)
    }
    return hxcmp.Redirect[Props]("/").Flash("success", "Deleted!")
}

func (c *FileViewer) handleRaw(ctx context.Context, props Props, w http.ResponseWriter) hxcmp.Result[Props] {
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte(props.File.Content))
    return hxcmp.Skip[Props]()
}

func (c *FileViewer) showEditForm(ctx context.Context, props Props) hxcmp.Result[Props] {
    props.EditMode = true
    return hxcmp.OK(props) // Re-render with edit form visible
}
```

### Pattern 4: Nested Components with Callbacks

**Use case**: Parent coordinates child components.

```go
type IssueDetail struct {
    *hxcmp.Component[Props]
    commentList *commentlist.CommentList
    commentForm *commentform.CommentForm
}

func (c *IssueDetail) Render(ctx context.Context, props Props) templ.Component {
    return Template(c, props)
}
```

```templ
templ Template(c *IssueDetail, props Props) {
    <div>
        <h1>{props.Issue.Title}</h1>

        <div id="comments">
            @c.commentList.Render(ctx, commentlist.Props{
                IssueID: props.Issue.ID,
            })
        </div>

        @c.commentForm.Render(ctx, commentform.Props{
            IssueID: props.Issue.ID,
            // When form submits, refresh the comment list
            OnSubmit: c.commentList.Refresh(commentlist.Props{
                IssueID: props.Issue.ID,
            }).Target("#comments").AsCallback(),
        })
    </div>
}
```

### Pattern 5: Event Broadcasting

**Use case**: Loose coupling between unrelated components.

```go
// Editor emits event
func (c *ItemEditor) handleSave(ctx context.Context, props Props) hxcmp.Result[Props] {
    if err := c.items.Save(ctx, props.Item); err != nil {
        return hxcmp.Err(props, err)
    }
    return hxcmp.OK(props).Trigger("item:updated").Flash("success", "Saved!")
}
```

```templ
// Sidebar listens for event (doesn't know about editor)
templ Sidebar(c *RecentItems, props Props) {
    <aside {...c.Refresh(props).OnEvent("item:updated").Attrs()}>
        // Auto-refreshes when any component emits "item:updated"
        for _, item := range props.RecentItems {
            <div>{item.Title}</div>
        }
    </aside>
}
```

### Pattern 6: Polling/Auto-refresh

**Use case**: Component that periodically refreshes.

```templ
templ Template(c *ActivityFeed, props Props) {
    <div {...c.Refresh(props).Every(5 * time.Second).Attrs()}>
        <h3>Recent Activity</h3>
        for _, activity := range props.Activities {
            <div>{activity.Message}</div>
        }
    </div>
}
```

### Pattern 7: Infinite Scroll

**Use case**: Load more items as user scrolls.

```templ
templ Template(c *ItemList, props Props) {
    <div>
        for _, item := range props.Items {
            <div>{item.Title}</div>
        }

        if len(props.Items) == props.Limit {
            <div {...c.Refresh(Props{
                Offset: props.Offset + props.Limit,
                Limit:  props.Limit,
            }).OnIntersect().Swap(hxcmp.SwapAfterEnd).Attrs()}>
                Loading...
            </div>
        }
    </div>
}
```

### Pattern 8: Confirmation Dialog

**Use case**: Dangerous action with confirmation.

```templ
templ Template(c *ItemCard, props Props) {
    <div class="item-card">
        <h3>{props.Item.Title}</h3>

        // Simple inline confirm
        <button {...c.Delete(props).Confirm("Delete this item?").Attrs()}>
            Delete
        </button>

        // Or target a modal
        <button {...c.ShowDeleteModal(props).Target("#modal").Swap(hxcmp.SwapInner).Attrs()}>
            Delete...
        </button>
    </div>
}
```

---

## Component Communication Patterns

hxcmp provides three communication mechanisms built on HTMX primitives:

### 1. Callbacks (Explicit Parent-Child)

**Use case**: Child needs to notify a specific parent after an action.

```go
// Parent passes callback when rendering child
@commentForm.Render(ctx, commentform.Props{
    IssueID:  issue.ID,
    OnSubmit: commentList.Refresh(listProps).Target("#comment-list").AsCallback(),
})
```

```go
// Child triggers callback after action
func (c *CommentForm) handleSubmit(ctx context.Context, props Props, r *http.Request) hxcmp.Result[Props] {
    // Save comment...
    if !props.OnSubmit.IsZero() {
        return hxcmp.OK(props).Callback(props.OnSubmit)
    }
    return hxcmp.OK(props)
}
```

**Result**: Response includes `HX-Trigger`; hxcmp's HTMX extension (optional) issues the callback request.

### 2. Events (Loose Coupling)

**Use case**: Broadcast that something happened; any interested component can react.

```go
// Emitter: broadcast event after action
func (c *ItemEditor) handleSave(ctx context.Context, props Props) hxcmp.Result[Props] {
    // Save item...
    return hxcmp.OK(props).Trigger("item:updated")
}
```

```templ
// Listener: refresh when event fires (using fluent builder)
<div {...c.Refresh(props).OnEvent("item:updated").Attrs()}>
    // Auto-refreshes when any component emits "item:updated"
</div>
```

**Generated HTML**:
```html
<div hx-get="/_c/itemlist-abc123/?p=..." hx-trigger="item:updated from:body" hx-swap="outerHTML">
```

### 3. Target Inheritance (Structural)

**Use case**: Child action should update a different part of the page.

```templ
// Child targets parent's container directly
<button {...c.Delete(props).Target("#file-list").Swap(hxcmp.SwapOuter).Attrs()}>
    Delete (updates file list)
</button>
```

### Comparison

| Pattern | Coupling | Direction | Use Case |
|---------|----------|-----------|----------|
| Callbacks | Tight | Child → specific parent | "Notify my parent when done" |
| Events | Loose | Any → any listeners | "Broadcast to whoever cares" |
| Target | Structural | Child → DOM element | "Update that specific element" |

### Out-of-Band Updates

For updating multiple elements in one response:

```go
func (c *CommentForm) handleSubmit(ctx context.Context, props Props, w http.ResponseWriter) hxcmp.Result[Props] {
    // Save comment...

    // Main response
    formHTML := renderForm(props)

    // OOB swap updates sibling
    listHTML := `<div id="comment-list" hx-swap-oob="true">` + renderCommentList(props.IssueID) + `</div>`

    w.Write([]byte(formHTML + listHTML))
    return hxcmp.Skip[Props]()
}
```

### Event Data

Events can include data. **Filtering is out of scope** for v3; use namespacing in event names (e.g., `comment:created:issue:123`) or use callbacks for directed updates.

### Summary

| Goal | Method |
|------|--------|
| Child notifies parent | `OnSubmit: parent.Action().AsCallback()` |
| Broadcast to listeners | `hxcmp.OK(props).Trigger("event")` |
| Listen for broadcasts | `.OnEvent("event")` |
| Update sibling | `.Target("#sibling-id")` |
| Update multiple | OOB swap with `hx-swap-oob` |
| Poll for updates | `.Every(5 * time.Second)` |

**Philosophy**: Provide typed wrappers around HTMX primitives. Events compile down to `HX-Trigger` headers and `hx-trigger` attributes. Callbacks compile down to `HX-Trigger` plus a tiny HTMX extension (optional) that issues the follow-up request.

---

## Security Considerations

### Threat Model

**Assumptions**:
- Attacker has access to URLs (can intercept HTTPS, inspect browser)
- Attacker cannot access signing/encryption keys (server-side only)
- Attacker can modify URLs, replay requests, send malicious data

**Out of scope**:
- XSS (mitigated by Templ auto-escaping)
- SQL injection (handled by ORM)

**Built-in protections**:
- CSRF (built-in via HX-Request header validation)
- Parameter tampering (HMAC signing or AES-256-GCM encryption)

### Two Security Modes

Components operate in one of two modes:

#### Default: Signed (Visible, Tamper-Proof)

```go
c := hxcmp.New[Props]("filebrowser")
```

- Props are Base64-encoded with HMAC signature
- **Visible**: Users can see props in URLs (good for debugging)
- **Tamper-proof**: HMAC fails if any byte is modified
- **Smaller URLs**: No encryption overhead
- **Bookmarkable**: Users can share component state

URL format: `/_c/filebrowser-abc123/?p=eyJyIjoxMjMsInAiOiJzcmMvIn0.HmacSig`

Use for: File browsers, navigation, search filters, pagination - anything where seeing the state is fine.

#### Sensitive: Encrypted (Hidden, Tamper-Proof)

```go
c := hxcmp.New[Props]("usersettings").Sensitive()
```

- Props are encrypted with AES-256-GCM
- **Hidden**: Props are opaque ciphertext
- **Tamper-proof**: GCM auth tag fails if modified
- **Larger URLs**: Encryption adds ~40 bytes overhead
- **Not enumerable**: Can't guess other users' IDs

URL format: `/_c/usersettings-abc123/?p=AES256GCMCiphertext`

Use for: User settings, billing, admin panels, anything with sensitive IDs.

**Callbacks**: The callback payload uses the **target component’s mode** (signed vs encrypted). A callback into a sensitive component produces an encrypted callback payload.

### Key Management

**Requirements**:
- 32-byte (256-bit) key from cryptographically secure RNG
- **Recommended**: derive separate keys for signing and encryption using HKDF
- Store in environment variables or secret management
- Rotate keys periodically (old URLs will fail gracefully)

```go
// Generate key
key := make([]byte, 32)
crypto_rand.Read(key)

// Use in registry
registry := hxcmp.NewRegistry(key)
```

### Parameter Tampering

Both modes detect tampering:

**Signed mode**:
```
Original: ?p=eyJyIjoxMjN9.validHmac
Modified: ?p=eyJyIjo5OTl9.validHmac  (changed payload)
Result: HMAC verification fails → HTTP 400
```

**Sensitive mode**:
```
Original: ?p=encryptedCiphertext
Modified: ?p=encryptedCiphertextX  (any change)
Result: GCM auth tag fails → HTTP 400
```

**Important**: Encryption doesn't replace authorization!

```go
// WRONG
func handleFile(w http.ResponseWriter, r *http.Request, props Props) error {
    // Missing authorization check!
    file := repo.ReadFile(props.Path)
}

// RIGHT
func handleFile(w http.ResponseWriter, r *http.Request, props Props) error {
    repo, _ := repoOps.GetByID(r.Context(), props.RepoID)

    user := auth.GetUser(r.Context())
    if !repo.CanRead(user) {
        return fmt.Errorf("forbidden")
    }

    file := repo.ReadFile(props.Path)
}
```

### CSRF Protection

**Built-in**: The registry automatically validates CSRF for mutating requests.

```go
func (reg *Registry) Handler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mutating methods require HX-Request header
        if r.Method != "GET" && r.Method != "HEAD" {
            if r.Header.Get("HX-Request") != "true" {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
        }
        reg.mux.ServeHTTP(w, r)
    })
}
```

**How it works**:
1. HTMX automatically adds `HX-Request: true` header to all requests
2. Browsers enforce SameSite cookie policies, preventing cross-origin requests with cookies
3. Combined, this prevents CSRF without tokens

**No additional configuration needed** - protection is automatic for all component actions.

### Replay Attacks

**Status**: Allowed - system is stateless.

URLs represent "render this component with these props" (idempotent).

For truly idempotent-sensitive operations (e.g., one-time tokens), implement additional checks in the handler:
```go
func (c *Component) handleRedeem(ctx context.Context, props Props, r *http.Request) hxcmp.Result[Props] {
    // Check if already redeemed in database
    if c.tokens.IsRedeemed(r.Context(), props.TokenID) {
        return hxcmp.Err(props, errors.New("token already redeemed"))
    }
    // ... redeem
    return hxcmp.OK(props)
}
```

### Rate Limiting

Apply rate limiting to component routes:
```go
componentHandler := hxcmp.Handler()
rateLimited := ratelimit.Middleware(componentHandler, 100, time.Minute)
mux.Handle("/_c/", rateLimited)
```

---

## Performance Characteristics

### Latency Breakdown

Typical request (Apple M1):
```
Total: ~450μs
├─ Route lookup: 5μs
├─ CSRF validation: 1μs
├─ Parameter decoding: 400μs
│   ├─ Base64 decode: 50μs
│   ├─ AES-GCM decrypt: 250μs
│   └─ Msgpack unmarshal: 100μs
└─ Template render: varies
```

### Memory Allocation

Per request:
```
No hydration: ~200 bytes, 3 allocs
With hydration: varies (DB queries)
```

### Throughput

Benchmarks:
```
No hydration:   200k req/s per core
With DB fetch:    2k req/s per core
```

---

## Migration and Adoption

### Incremental Adoption

Step 1: Add global router
```go
registry := hxcmp.NewRegistry(key)
mux.Handle("/_c/", registry.Handler())
```

Step 2: Create first component with lifecycle methods
```go
type UserCard struct {
    *hxcmp.Component[Props]
    users *ops.User
}

func (c *UserCard) Hydrate(ctx context.Context, props *Props) error {
    if props.User == nil && props.UserID > 0 {
        user, _ := c.users.GetByID(ctx, props.UserID)
        props.User = user
    }
    return nil
}

func (c *UserCard) Render(ctx context.Context, props Props) templ.Component {
    return Template(props.User)
}
```

Step 3: Register and use
```go
userCard := usercard.New(userOps)
registry.Add(userCard)
```

```templ
@userCard.Render(ctx, usercard.Props{UserID: 123})
```

### Migrating from Current Jorje System

**Current (manual mounting + hydration in handlers)**:
```go
func (h *Handler) Mount(e *echo.Echo) {
    h.fileViewer.Mount(g)
}

func (c *FileViewer) handleEdit(ctx context.Context, props Props) hxcmp.Result[Props] {
    if props.Repo == nil { /* hydrate */ }  // Repeated everywhere
    // ...
    return hxcmp.OK(props)
}
```

**New (registry + lifecycle methods)**:
```go
func main() {
    fileViewer := fileviewer.New(repoOps)
    registry.Add(fileViewer)  // Validates interface compliance
}

// Hydration written once
func (c *FileViewer) Hydrate(ctx context.Context, props *Props) error {
    if props.Repo == nil && props.RepoID > 0 { /* hydrate */ }
    return nil
}

// Handler assumes hydrated props
func (c *FileViewer) handleEdit(ctx context.Context, props Props) hxcmp.Result[Props] {
    // props.Repo is guaranteed non-nil
    return hxcmp.OK(props)
}
```

**Template migration**:
```templ
// Before: raw HTMX attributes
<button hx-post={c.URLEdit(props)} hx-target="#viewer" hx-confirm="Save?">

// After: fluent builder
<button {...c.Edit(props).Target("#viewer").Confirm("Save?").Attrs()}>
```

---

## Code Generation

hxcmp includes a code generator that provides performance improvements and enhanced type safety. Generation is **required** - there are no reflection-based fallbacks for the generated code.

**What's generated** (zero reflection in hot path):
- Props encoder/decoder methods
- Typed action builder methods (e.g., `c.Edit(props)`)
- Handler dispatch switch statements

**What uses reflection** (startup time only):
- Registry component registration (finding embedded types, validating interfaces)
- This happens once at application start, not per-request

### Build Workflow

```bash
# Required ordering: hxcmp before templ
hxcmp generate ./...
templ generate ./...
go build ./...
```

Enforce via build system:

```just
# justfile
generate:
    hxcmp generate ./...
    templ generate ./...

build: generate
    go build ./...

dev: generate
    # watch mode, etc.
```

### What Gets Generated

For each component, hxcmp generates a `*_hx.go` file:

```
components/
├── fileviewer/
│   ├── fileviewer.go      # User-written component
│   ├── fileviewer_hx.go   # Generated by hxcmp
│   └── template.templ     # User-written template
```

#### 1. Props Encoder/Decoder

Fast, reflection-free serialization:

```go
// fileviewer_hx.go (generated)

func (p Props) hxcmpEncode() map[string]any {
    m := make(map[string]any, 4)
    m["r"] = p.RepoID
    m["p"] = p.Path
    if !p.LastModified.IsZero() {
        m["m"] = p.LastModified.Format(time.RFC3339)
    }
    if !p.OnSave.IsZero() {
        m["cb"] = p.OnSave // Callback is serialized
    }
    return m
}

func (p *Props) hxcmpDecode(m map[string]any) error {
    if v, ok := m["r"]; ok {
        switch n := v.(type) {
        case float64:
            p.RepoID = int64(n)
        case int64:
            p.RepoID = n
        default:
            return fmt.Errorf("RepoID: expected number, got %T", v)
        }
    }
    if v, ok := m["p"].(string); ok {
        p.Path = v
    }
    if v, ok := m["m"].(string); ok {
        t, err := time.Parse(time.RFC3339, v)
        if err != nil {
            return fmt.Errorf("LastModified: %w", err)
        }
        p.LastModified = t
    }
    if v, ok := m["cb"].(map[string]any); ok {
        p.OnSave = hxcmp.CallbackFromMap(v)
    }
    return nil
}
```

**Performance**: ~10μs vs ~400μs with reflection (40x faster).

**Encoding format**:
- Default serialization is MessagePack + optional encryption/signing.
- Because hxcmp uses code generation, an even more efficient encoding (e.g., msgp) is feasible and may be offered as an opt-in later.

#### 2. Action Builder Methods

Typed action builders with fluent API:

```go
// fileviewer_hx.go (generated)

// Edit returns an action builder for the "edit" action
func (c *FileViewer) Edit(props Props) *hxcmp.Action {
    return &hxcmp.Action{
        URL:    c.Prefix() + "/edit?p=" + c.encodeProps(props),
        Method: http.MethodPost,
        Swap:   hxcmp.SwapOuter, // Default
    }
}

// Delete returns an action builder for the "delete" action
func (c *FileViewer) Delete(props Props) *hxcmp.Action {
    return &hxcmp.Action{
        URL:    c.Prefix() + "/delete?p=" + c.encodeProps(props),
        Method: http.MethodDelete,
        Swap:   hxcmp.SwapOuter,
    }
}

// Raw returns an action builder for the "raw" action
func (c *FileViewer) Raw(props Props) *hxcmp.Action {
    return &hxcmp.Action{
        URL:    c.Prefix() + "/raw?p=" + c.encodeProps(props),
        Method: http.MethodGet,
    }
}

// Refresh returns an action builder for re-rendering
func (c *FileViewer) Refresh(props Props) *hxcmp.Action {
    return &hxcmp.Action{
        URL:    c.Prefix() + "/?p=" + c.encodeProps(props),
        Method: http.MethodGet,
        Swap:   hxcmp.SwapOuter,
    }
}
```

**Usage in templates**:

```templ
// Typo caught at compile time
<button {...c.Edit(props).Attrs()}>Edit</button>
<button {...c.Eidt(props).Attrs()}>Edit</button>  // Compile error!

// Full fluent chain
<button {...c.Delete(props).Target("#file-list").Confirm("Delete?").Attrs()}>
    Delete
</button>
```

#### 3. Handler Dispatch Implementation

Static dispatch with lifecycle integration:

```go
// fileviewer_hx.go (generated)

// Compile-time interface compliance
var _ hxcmp.HXComponent = (*FileViewer)(nil)

func (c *FileViewer) HXPrefix() string {
    return "/_c/fileviewer-a1b2c3d4"
}

func (c *FileViewer) HXServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Decode props
    encoded := r.URL.Query().Get("p")
    var props Props
    if err := c.decodeProps(encoded, &props); err != nil {
        c.registry.OnError(w, r, err)
        return
    }

    // Run lifecycle: Hydrate
    if err := c.Hydrate(r.Context(), &props); err != nil {
        c.registry.OnError(w, r, err)
        return
    }

    // Route to handler
    path := strings.TrimPrefix(r.URL.Path, c.HXPrefix())
    switch r.Method + " " + path {
    case "GET /", "GET ":
        c.serveRender(w, r, props)
    case "POST /edit":
        c.serveEdit(w, r, props)
    case "GET /raw":
        c.serveRaw(w, r, props)
    case "DELETE /delete":
        c.serveDelete(w, r, props)
    default:
        http.NotFound(w, r)
    }
}

func (c *FileViewer) serveEdit(w http.ResponseWriter, r *http.Request, props Props) {
    // Call user's handler (auto-detected signature)
    result := c.handleEdit(r.Context(), props, r)

    // Handle result
    c.handleResult(w, r, result)
}

func (c *FileViewer) handleResult(w http.ResponseWriter, r *http.Request, result hxcmp.Result[Props]) {
    if result.GetErr() != nil {
        c.registry.OnError(w, r, result.GetErr())
        return
    }
    if result.GetStatus() != 0 {
        w.WriteHeader(result.GetStatus())
    }
    for k, v := range result.GetHeaders() {
        w.Header().Set(k, v)
    }
    if result.GetRedirect() != "" {
        w.Header().Set("HX-Redirect", result.GetRedirect())
        return
    }
    // Combine callback and trigger into single HX-Trigger header
    triggers := []string{}
    if cb := result.GetCallback(); cb != nil {
        triggers = append(triggers, cb.TriggerJSON())
    }
    if result.GetTrigger() != "" {
        triggers = append(triggers, result.GetTrigger())
    }
    if len(triggers) > 0 {
        w.Header().Set("HX-Trigger", strings.Join(triggers, ", "))
    }
    if result.ShouldSkip() {
        return
    }
    c.Render(r.Context(), result.GetProps()).Render(r.Context(), w)
}
```

### How Generation Works

The generator uses Go's AST parser - **no compilation required**:

```go
// Generator parses source files without compiling
fset := token.NewFileSet()
f, _ := parser.ParseFile(fset, "fileviewer.go", nil, parser.ParseComments)

// Walks AST to extract:
// 1. Structs embedding *hxcmp.Component[P]
// 2. Props struct with `hx` tags
// 3. Action registrations (c.Action("name", ...))
```

This means generation works even on a fresh clone with no existing generated files.

**Extraction process**:

1. Find struct types embedding `*hxcmp.Component[T]`
2. Resolve `T` to find the Props type
3. Parse Props struct fields for `hx` tags
4. Find action registrations by pattern matching `c.Action("name", ...)` calls
5. Generate encoder/decoder from Props fields
6. Generate URL methods from route registrations

### Generator CLI

```bash
# Install
go install github.com/yourorg/hxcmp/cmd/hxcmp@latest

# Generate for all packages
hxcmp generate ./...

# Generate for specific package
hxcmp generate ./components/fileviewer

# Check what would be generated (dry run)
hxcmp generate --dry-run ./...

# Clean generated files
hxcmp clean ./...
```

### Generated File Header

All generated files include a header preventing manual edits:

```go
// Code generated by hxcmp. DO NOT EDIT.
// Source: fileviewer.go

//go:build !hxcmp_ignore

package fileviewer
```

The `//go:build !hxcmp_ignore` tag allows excluding generated code if needed (e.g., for debugging).

### Component Dependencies

Components can depend on other components. Since each `*_hx.go` only adds methods to its own package's types, **generation order doesn't matter**:

```go
// fileviewer/fileviewer.go
type FileViewer struct {
    *hxcmp.Component[Props]
    browser *filebrowser.FileBrowser  // Dependency on another component
}
```

```templ
// fileviewer/template.templ
<a href={browser.URLExpand(fbProps)}>Expand</a>  // Uses FileBrowser's generated method
```

**Why this works**:
- `hxcmp generate` creates ALL `*_hx.go` files (any order)
- Then `templ generate` runs (all URL methods now exist)
- No circular dependencies between generated files

### CI Integration

```yaml
# .github/workflows/build.yml
jobs:
  build:
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install tools
        run: |
          go install github.com/yourorg/hxcmp/cmd/hxcmp@latest
          go install github.com/a-h/templ/cmd/templ@latest

      - name: Generate
        run: |
          hxcmp generate ./...
          templ generate ./...

      - name: Build
        run: go build ./...

      - name: Test
        run: go test ./...
```

### Checking Generated Code Into Git

**Recommended**: Don't check in generated files. Add to `.gitignore`:

```gitignore
# Generated files
*_hx.go
*_templ.go
```

**Alternative**: Check them in for faster CI / easier debugging. Ensure CI regenerates to catch staleness:

```yaml
- name: Check generated files are up to date
  run: |
    hxcmp generate ./...
    templ generate ./...
    git diff --exit-code
```

---

## Future Directions

### Near-term

1. **Testing utilities** - Component test helpers
   ```go
   result := hxcmp.TestRender(component, Props{...})
   assert.Contains(result.HTML, "expected content")
   assert.Equal("item:updated", result.TriggeredEvents[0])
   ```

2. **Framework adapters** - Echo, Chi, Gin middleware adapters
   ```go
   e.Use(hxcmp.EchoAdapter(registry))
   ```

3. **IDE plugin** - Jump to action definition, autocomplete builders

4. **Validation integration** - Props validation before hydration
   ```go
   func (p Props) Validate() error {
       if p.RepoID <= 0 { return errors.New("invalid repo id") }
       return nil
   }
   ```

### Medium-term

5. **Component middleware** - Per-component auth/rate limiting
   ```go
   c.Action("delete", c.handleDelete).
       Middleware(auth.RequireRole("admin")).
       RateLimit(10, time.Minute)
   ```

6. **Streaming components** - Server-Sent Events for real-time updates
   ```go
   c.Action("subscribe", c.handleSubscribe).Stream()
   ```

7. **Form validation** - Inline validation with error display
   ```go
   if errs := validate(r); len(errs) > 0 {
       return props, hxcmp.ValidationErrors(errs)
   }
   ```

### Long-term

8. **Hot reload** - Watch mode for development
9. **DevTools** - Browser extension for debugging component state
10. **Partial hydration** - Hydrate only changed props on subsequent renders

---

**End of Specification**
