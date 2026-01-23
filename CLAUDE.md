# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

hxcmp is an HTMX component system for Go that enables React-like component composition using Go, Templ templates, and HTMX. Components are strongly typed via generics with compile-time verification of action methods.

## Build Commands

```bash
# Required build order
hxcmp generate ./...   # Generate *_hx.go files (run first)
templ generate ./...   # Generate *_templ.go files (run second)
go build ./...         # Compile

# Run all tests
go test ./...

# Run specific test
go test -run TestActionBuilder ./...

# Clean generated files
hxcmp clean ./...

# Preview generation without writing
hxcmp generate --dry-run ./...
```

## Architecture

### Core Types

- `Component[P]` - Base type embedded by user components where P is the Props type
- `Result[P]` - Handler return type with fluent builder for flash, redirect, trigger
- `WireAttrs()` - Generates minimal HTMX attributes (hx-get/hx-post + hx-vals)
- `Registry` - Component manager that handles routing and CSRF protection

### Lifecycle Interfaces

Components must implement:
- `Hydrater[P]` - `Hydrate(ctx, *P) error` - Reconstructs rich objects from serialized IDs
- `Renderer[P]` - `Render(ctx, P) templ.Component` - Produces templ output

Hydrate runs automatically before any handler. Render is called after successful actions.

### Component Structure Pattern

```go
type FileViewer struct {
    *hxcmp.Component[Props]
    repo *ops.Repo
}

func New(repo *ops.Repo) *FileViewer {
    c := &FileViewer{
        Component: hxcmp.New[Props]("fileviewer"),
        repo: repo,
    }
    c.Action("edit", c.handleEdit)
    return c
}
```

### Security Model

- Props encoded in URLs via signed (default, HMAC) or encrypted (use `.Sensitive()`) modes
- CSRF protection: mutating methods require `HX-Request: true` header
- Registry verifies component interfaces at registration time (not runtime)

### Code Generation

The `hxcmp generate` command produces `*_hx.go` files containing:
- Fast encoder/decoder for Props (implements Encodable/Decodable)
- Wire methods (e.g., WireEdit, WireDelete) returning `templ.Attributes`
- `WireRender` method for the default GET endpoint
- `HXServeHTTP` dispatcher that routes requests to handlers

Generated code eliminates reflection in the hot path.

### Result Patterns

```go
return hxcmp.OK(props)                       // Auto-render with props
return hxcmp.OK(props).Flash("success", "Saved!")
return hxcmp.Err(props, err)                 // Error handling
return hxcmp.Redirect[Props]("/dashboard")   // Client redirect
return hxcmp.Skip[Props]()                   // Handler wrote own response
return hxcmp.OK(props).Trigger("item-updated") // Broadcast event
return hxcmp.OK(props).PushURL("/items?page=2") // Update browser URL
```

### File Layout

```
hxcmp/
├── component.go   # Component[P] generic type
├── registry.go    # Registry, routing, CSRF
├── action.go      # ActionBuilder, WireAttrs()
├── result.go      # Result[P] type
├── encoder.go     # Signed/encrypted encoding
├── flash.go       # Flash messages, OOB swaps
├── helpers.go     # IsHTMX(), BuildTriggerHeader(), etc.
├── errors.go      # Sentinel errors (IsNotFound, IsDecryptionError)
├── testing.go     # TestRender, TestAction, TestRequestBuilder
├── interfaces.go  # Hydrater, Renderer, HXComponent interfaces
├── cmd/hxcmp/     # CLI: hxcmp generate/clean
├── lib/generator/ # Code generation AST parser
└── ext/           # hxcmp-ext.js (event data injection, toast handling)
```

## Testing

The package provides testing utilities in `testing.go`:

```go
result, err := hxcmp.TestRender(comp, props)
result.HTMLContains("expected text")

result, err := hxcmp.TestAction(comp, url, "POST", formData)
result.IsOK()
result.HasFlash("success", "Saved!")
result.HasEvent("item-updated")
result.WasRedirected()
```

Use `MockHydrater` to inject test data without real dependencies.

## Dependencies

- `github.com/a-h/templ` - Templ templating
- `github.com/vmihailenco/msgpack/v5` - Props encoding
