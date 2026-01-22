package generator

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestDetectHandlerSignature(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected HandlerSignature
	}{
		{
			name: "ctx and props only",
			code: `
package test
import "context"
type Props struct{}
type Result struct{}
func (c *Comp) handler(ctx context.Context, props Props) Result { return Result{} }
`,
			expected: HandlerSigCtxProps,
		},
		{
			name: "ctx, props, and request",
			code: `
package test
import (
	"context"
	"net/http"
)
type Props struct{}
type Result struct{}
func (c *Comp) handler(ctx context.Context, props Props, r *http.Request) Result { return Result{} }
`,
			expected: HandlerSigCtxPropsRequest,
		},
		{
			name: "ctx, props, and response writer",
			code: `
package test
import (
	"context"
	"net/http"
)
type Props struct{}
type Result struct{}
func (c *Comp) handler(ctx context.Context, props Props, w http.ResponseWriter) Result { return Result{} }
`,
			expected: HandlerSigCtxPropsWriter,
		},
	}

	g := New(Options{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}

			// Find the function declaration
			for _, decl := range file.Decls {
				funcDecl, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if funcDecl.Name.Name != "handler" {
					continue
				}

				sig := g.detectHandlerSignature(funcDecl.Type.Params.List)
				if sig != tt.expected {
					t.Errorf("detectHandlerSignature() = %v, want %v", sig, tt.expected)
				}
				return
			}
			t.Fatal("handler function not found")
		})
	}
}

func TestFindHandlerSignatures(t *testing.T) {
	code := `
package test

import (
	"context"
	"net/http"
)

type Props struct{}
type Result struct{}

type Comp struct{}

func (c *Comp) simpleHandler(ctx context.Context, props Props) Result {
	return Result{}
}

func (c *Comp) requestHandler(ctx context.Context, props Props, r *http.Request) Result {
	return Result{}
}

func (c *Comp) writerHandler(ctx context.Context, props Props, w http.ResponseWriter) Result {
	return Result{}
}
`

	g := New(Options{})
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, 0)
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	sigs := g.findHandlerSignatures(file, "Comp")

	tests := []struct {
		name     string
		expected HandlerSignature
	}{
		{"simpleHandler", HandlerSigCtxProps},
		{"requestHandler", HandlerSigCtxPropsRequest},
		{"writerHandler", HandlerSigCtxPropsWriter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, ok := sigs[tt.name]
			if !ok {
				t.Fatalf("Handler %s not found", tt.name)
			}
			if sig != tt.expected {
				t.Errorf("Signature for %s = %v, want %v", tt.name, sig, tt.expected)
			}
		})
	}
}
