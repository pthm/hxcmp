<img width="434" height="83" alt="hxcmp" src="https://github.com/user-attachments/assets/11a72728-2030-4977-ac59-c2976b9f13f4" />

# hxcmp

A component system for Go that brings React-like composition to server-rendered applications using [Templ](https://templ.guide) templates and [HTMX](https://htmx.org). Components are strongly typed via generics with compile-time verified actions and zero reflection in the request path.

```go
type Counter struct {
    *hxcmp.Component[Props]
}

type Props struct {
    Count int `hx:"count"`
}

func New() *Counter {
    c := &Counter{Component: hxcmp.New[Props]("counter")}
    c.Action("increment", c.handleIncrement)
    return c
}

func (c *Counter) Hydrate(ctx context.Context, props *Props) error { return nil }

func (c *Counter) Render(ctx context.Context, props Props) templ.Component {
    return counterTemplate(c, props)
}

func (c *Counter) handleIncrement(ctx context.Context, props Props) hxcmp.Result[Props] {
    props.Count++
    return hxcmp.OK(props)
}
```

Each component is a self-contained unit with its own props, lifecycle methods, action handlers, and routes. The `hxcmp generate` tool produces type-safe dispatchers and serializers so the runtime never uses reflection.

> **Work in progress** -- contributions welcome.

## Installation

### Runtime library

```bash
go get github.com/pthm/hxcmp
```

### CLI tool

```bash
go install github.com/pthm/hxcmp/cmd/hxcmp@latest
```

### Client extension

Include the provided `hxcmp-ext.js` after HTMX in your layout. It handles event data injection and toast auto-dismiss.

```html
<script src="https://unpkg.com/htmx.org"></script>
<script src="/static/hxcmp-ext.js"></script>
```

## Code Generation

`hxcmp generate` parses your component source files and produces `*_hx.go` files containing fast prop encoders/decoders, Wire methods, and HTTP dispatch logic. Generation must run before `templ generate`:

```bash
hxcmp generate ./...   # produces *_hx.go files
templ generate ./...   # produces *_templ.go files (may reference generated actions)
go build ./...
```

Other commands:

```bash
hxcmp generate --dry-run ./...   # preview without writing
hxcmp clean ./...                # remove generated files
```

## Quick Start

Mount the component system onto your mux and register components:

```go
func main() {
    mux := http.NewServeMux()

    // Mount creates a registry, sets it as default, and attaches the handler.
    // In production pass hxcmp.WithKey(key) for stable prop signing across restarts.
    reg := hxcmp.Mount(mux)

    // Register components
    reg.Add(
        counter.New(),
        todolist.New(store),
    )

    http.ListenAndServe(":8080", mux)
}
```

## Core Concepts

### Component Lifecycle

Every component embeds `*hxcmp.Component[P]` where `P` is a props struct, and implements two interfaces:

- **`Hydrate(ctx, *P) error`** -- Runs before every request. Reconstructs rich objects (DB lookups, service calls) from the serialized prop IDs.
- **`Render(ctx, P) templ.Component`** -- Produces the Templ output after hydration and handler execution.

### Props

Props are the component's serializable state. Scalar fields are encoded into signed URLs by default; complex fields marked `hx:"-"` are excluded and populated during hydration.

```go
type Props struct {
    ItemID   string `hx:"id"`
    Page     int    `hx:"page,omitempty"`
    Item     *Item  `hx:"-"` // hydrated, not serialized
}
```

Call `.Sensitive()` on a component to encrypt props instead of signing them.

### Actions

Actions register named handlers on a component. They default to POST; override with `.Method()`:

```go
c.Action("save", c.handleSave)
c.Action("delete", c.handleDelete).Method(http.MethodDelete)
```

Code generation produces Wire methods (e.g. `c.WireSave(props)`, `c.WireDelete(props)`) that return `templ.Attributes` with the minimal HTMX attributes. All other HTMX attributes are written directly in templates:

```html
<!-- In a templ template -->
<button { c.WireSave(props)... } hx-target="#form" hx-confirm="Save changes?">
    Save
</button>
```

Handler signatures are auto-detected -- use whichever you need:

```go
func (c *Comp) handle(ctx context.Context, props Props) Result[Props]
func (c *Comp) handle(ctx context.Context, props Props, r *http.Request) Result[Props]
func (c *Comp) handle(ctx context.Context, props Props, w http.ResponseWriter) Result[Props]
```

### Result

Handlers return `Result[P]`, a fluent builder for the response:

```go
return hxcmp.OK(props)                                  // re-render with updated props
return hxcmp.OK(props).Flash("success", "Saved!")       // with toast notification
return hxcmp.OK(props).Trigger("item:changed")          // broadcast event
return hxcmp.OK(props).PushURL("/items/42")             // update browser URL
return hxcmp.Err(props, err)                            // error response
return hxcmp.Redirect[Props]("/dashboard")              // client redirect
return hxcmp.Skip[Props]()                              // handler wrote its own response
```

### Wire Methods

Generated Wire methods return minimal `templ.Attributes` containing only the HTTP
method attribute and encoded props. All other HTMX attributes are written directly
in templates:

```html
<button
    { c.WireIncrement(props)... }
    hx-target="#counter"
    hx-swap="outerHTML"
>+</button>
```

This keeps templates HTMX-native — you write standard HTMX attributes for targeting,
swapping, triggers, confirms, etc.

### Component Communication

Components communicate through events, not direct references:

```go
// Sender: broadcast after mutation
return hxcmp.OK(props).Trigger("todo:changed")
```

```html
<!-- Receiver: re-render when event fires -->
<div { c.WireRender(props)... }
     hx-target="#stats"
     hx-swap="outerHTML"
     hx-trigger="todo:changed from:body">
</div>
```

### Lazy Loading

Defer rendering until the element enters the viewport:

```go
c.Lazy(props, placeholder)  // loads on intersection
c.Defer(props, placeholder) // loads immediately after page
```

### Flash Messages

Toast notifications rendered via HTMX out-of-band swaps:

```go
return hxcmp.OK(props).Flash(hxcmp.FlashSuccess, "Item saved!")
```

Add `hxcmp.ToastContainer()` to your layout to receive them.

## Security

- **Prop integrity**: Props are HMAC-signed by default. Use `.Sensitive()` for AES encryption.
- **CSRF protection**: Mutating actions require the `HX-Request: true` header (sent automatically by HTMX).
- **No direct prop access**: Users cannot forge or tamper with component state.

## Testing

The `hxcmp` package provides testing utilities for unit-testing components without a running server:

```go
// Test rendering
result, err := hxcmp.TestRender(comp, props)
result.HTMLContains("expected text")

// Test action handlers
result, err := hxcmp.TestAction(comp, actionURL, "POST", formData)
result.IsOK()
result.HasFlash("success", "Saved!")
result.HasEvent("item:changed")
result.WasRedirected()
```

Use `MockHydrater` to inject test data without real dependencies.

## Examples

A complete working example with multiple interacting components is available in the [`examples/todo`](./examples/todo) directory of this repository.

## Dependencies

- [templ](https://github.com/a-h/templ) -- Go HTML templating
- [msgpack](https://github.com/vmihailenco/msgpack) -- Efficient prop serialization

## License

See [LICENSE](./LICENSE).
