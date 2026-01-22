package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Options configures the generator.
type Options struct {
	DryRun bool
}

// Generator generates hxcmp code.
type Generator struct {
	opts Options
	fset *token.FileSet
}

// New creates a new generator.
func New(opts Options) *Generator {
	return &Generator{
		opts: opts,
		fset: token.NewFileSet(),
	}
}

// Generate generates code for the given package patterns.
func (g *Generator) Generate(patterns ...string) error {
	packages, err := g.findPackages(patterns)
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		if err := g.generatePackage(pkg); err != nil {
			return fmt.Errorf("package %s: %w", pkg, err)
		}
	}

	return nil
}

// Clean removes generated files for the given package patterns.
func (g *Generator) Clean(patterns ...string) error {
	packages, err := g.findPackages(patterns)
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		if err := g.cleanPackage(pkg); err != nil {
			return fmt.Errorf("package %s: %w", pkg, err)
		}
	}

	return nil
}

// findPackages resolves package patterns to directory paths.
func (g *Generator) findPackages(patterns []string) ([]string, error) {
	var packages []string

	for _, pattern := range patterns {
		// Handle ./... pattern
		if strings.HasSuffix(pattern, "/...") {
			root := strings.TrimSuffix(pattern, "/...")
			if root == "." || root == "" {
				root = "."
			}

			err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					return nil
				}
				// Skip hidden directories and vendor
				base := filepath.Base(path)
				if strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata" {
					return filepath.SkipDir
				}

				// Check if directory contains Go files
				entries, err := os.ReadDir(path)
				if err != nil {
					return nil
				}
				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
						packages = append(packages, path)
						break
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			// Direct path
			packages = append(packages, pattern)
		}
	}

	return packages, nil
}

// generatePackage generates code for a single package.
func (g *Generator) generatePackage(pkgPath string) error {
	// Parse all Go files in the package
	pkgs, err := parser.ParseDir(g.fset, pkgPath, func(info os.FileInfo) bool {
		name := info.Name()
		// Skip test files and generated files
		return !strings.HasSuffix(name, "_test.go") && !strings.HasSuffix(name, "_hx.go")
	}, parser.ParseComments)
	if err != nil {
		return err
	}

	for pkgName, pkg := range pkgs {
		components := g.findComponents(pkg)
		if len(components) == 0 {
			continue
		}

		for _, comp := range components {
			if err := g.generateComponent(pkgPath, pkgName, comp); err != nil {
				return err
			}
		}
	}

	return nil
}

// cleanPackage removes generated files from a package.
func (g *Generator) cleanPackage(pkgPath string) error {
	entries, err := os.ReadDir(pkgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), "_hx.go") {
			path := filepath.Join(pkgPath, entry.Name())
			fmt.Printf("removing %s\n", path)
			if !g.opts.DryRun {
				if err := os.Remove(path); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ComponentInfo holds information about a discovered component.
type ComponentInfo struct {
	SourceFile   string
	TypeName     string       // e.g., "FileViewer"
	PropsType    string       // e.g., "Props"
	Props        []PropField  // Parsed props fields
	Actions      []ActionInfo // Registered actions
	ComponentNew string       // The name passed to hxcmp.New[P]("name")
}

// PropField represents a field in the Props struct.
type PropField struct {
	Name      string
	Type      string
	Tag       string // The hx tag value
	OmitEmpty bool
	Exclude   bool // hx:"-"
}

// ActionInfo represents a registered action.
type ActionInfo struct {
	Name    string // Action name (e.g., "edit")
	Method  string // HTTP method (defaults to POST)
	Handler string // Handler method name
}

// findComponents finds all component types in a package.
func (g *Generator) findComponents(pkg *ast.Package) []*ComponentInfo {
	var components []*ComponentInfo

	for filename, file := range pkg.Files {
		for _, decl := range file.Decls {
			// Look for type declarations
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				// Check if it embeds *hxcmp.Component[P]
				propsType, componentNew := g.findEmbeddedComponent(structType)
				if propsType == "" {
					continue
				}

				comp := &ComponentInfo{
					SourceFile:   filename,
					TypeName:     typeSpec.Name.Name,
					PropsType:    propsType,
					ComponentNew: componentNew,
				}

				// Find the Props struct in the same file
				comp.Props = g.findPropsFields(file, propsType)

				// Find action registrations
				comp.Actions = g.findActions(file, typeSpec.Name.Name)

				components = append(components, comp)
			}
		}
	}

	return components
}

// findEmbeddedComponent checks if a struct embeds *hxcmp.Component[P].
// Returns the props type name and the component name from New().
func (g *Generator) findEmbeddedComponent(structType *ast.StructType) (propsType string, componentNew string) {
	for _, field := range structType.Fields.List {
		// Check for anonymous field (embedding)
		if len(field.Names) != 0 {
			continue
		}

		// Check if it's a pointer type
		starExpr, ok := field.Type.(*ast.StarExpr)
		if !ok {
			continue
		}

		// Check if it's an indexed expression (generic type)
		indexExpr, ok := starExpr.X.(*ast.IndexExpr)
		if !ok {
			continue
		}

		// Check if it's hxcmp.Component or just Component
		var typeName string
		switch x := indexExpr.X.(type) {
		case *ast.SelectorExpr:
			// hxcmp.Component
			if ident, ok := x.X.(*ast.Ident); ok && ident.Name == "hxcmp" {
				typeName = x.Sel.Name
			}
		case *ast.Ident:
			// Component (imported without qualifier)
			typeName = x.Name
		}

		if typeName != "Component" {
			continue
		}

		// Get the type parameter
		if ident, ok := indexExpr.Index.(*ast.Ident); ok {
			propsType = ident.Name
			return propsType, ""
		}
	}

	return "", ""
}

// findPropsFields finds the fields of a Props struct.
func (g *Generator) findPropsFields(file *ast.File, propsTypeName string) []PropField {
	var fields []PropField

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != propsTypeName {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue // Skip embedded fields
				}

				for _, name := range field.Names {
					pf := PropField{
						Name: name.Name,
						Type: g.typeToString(field.Type),
					}

					// Parse hx tag
					if field.Tag != nil {
						tag := strings.Trim(field.Tag.Value, "`")
						pf.Tag, pf.OmitEmpty, pf.Exclude = parseHXTag(tag)
					}

					// Auto-detection for untagged fields
					if pf.Tag == "" && !pf.Exclude {
						if isScalarType(pf.Type) {
							pf.Tag = strings.ToLower(name.Name)
						} else {
							pf.Exclude = true
						}
					}

					fields = append(fields, pf)
				}
			}
		}
	}

	return fields
}

// findActions finds action registrations in the component's New function.
func (g *Generator) findActions(file *ast.File, typeName string) []ActionInfo {
	var actions []ActionInfo

	// Look for function declarations
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Look for c.Action(...) calls
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Check if it's a method call on 'c'
			selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if selExpr.Sel.Name != "Action" {
				return true
			}

			// Extract action name
			if len(callExpr.Args) < 2 {
				return true
			}

			nameLit, ok := callExpr.Args[0].(*ast.BasicLit)
			if !ok || nameLit.Kind != token.STRING {
				return true
			}

			actionName := strings.Trim(nameLit.Value, `"`)
			action := ActionInfo{
				Name:   actionName,
				Method: "POST", // Default
			}

			// Try to extract handler name
			if len(callExpr.Args) >= 2 {
				if sel, ok := callExpr.Args[1].(*ast.SelectorExpr); ok {
					action.Handler = sel.Sel.Name
				}
			}

			actions = append(actions, action)
			return true
		})
	}

	return actions
}

// typeToString converts an AST type to a string representation.
func (g *Generator) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + g.typeToString(t.X)
	case *ast.SelectorExpr:
		return g.typeToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + g.typeToString(t.Elt)
		}
		return "[...]" + g.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + g.typeToString(t.Key) + "]" + g.typeToString(t.Value)
	case *ast.IndexExpr:
		return g.typeToString(t.X) + "[" + g.typeToString(t.Index) + "]"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// parseHXTag parses an hx struct tag.
func parseHXTag(tagStr string) (key string, omitEmpty bool, exclude bool) {
	// Find hx:"..." in the tag string
	for _, part := range strings.Split(tagStr, " ") {
		if strings.HasPrefix(part, `hx:"`) {
			value := strings.TrimPrefix(part, `hx:"`)
			value = strings.TrimSuffix(value, `"`)

			if value == "-" {
				return "", false, true
			}

			parts := strings.Split(value, ",")
			key = parts[0]
			for _, p := range parts[1:] {
				if p == "omitempty" {
					omitEmpty = true
				}
			}
			return key, omitEmpty, false
		}
	}
	return "", false, false
}

// isScalarType checks if a type should be auto-serialized.
func isScalarType(typeName string) bool {
	switch typeName {
	case "bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"string",
		"time.Time":
		return true
	default:
		return false
	}
}
