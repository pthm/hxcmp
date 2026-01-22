package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

// generateComponent generates the *_hx.go file for a component.
func (g *Generator) generateComponent(pkgPath, pkgName string, comp *ComponentInfo) error {
	// Determine output filename
	baseName := strings.TrimSuffix(filepath.Base(comp.SourceFile), ".go")
	outputFile := filepath.Join(pkgPath, baseName+"_hx.go")

	fmt.Printf("generating %s\n", outputFile)

	if g.opts.DryRun {
		return nil
	}

	// Generate the code
	code, err := g.renderTemplate(pkgName, comp)
	if err != nil {
		return fmt.Errorf("render template: %w", err)
	}

	// Format the code
	formatted, err := format.Source(code)
	if err != nil {
		// Write unformatted for debugging
		if writeErr := os.WriteFile(outputFile+".unformatted", code, 0644); writeErr == nil {
			fmt.Printf("  wrote unformatted code to %s.unformatted for debugging\n", outputFile)
		}
		return fmt.Errorf("format source: %w", err)
	}

	// Write the file
	return os.WriteFile(outputFile, formatted, 0644)
}

// renderTemplate renders the generated code template.
func (g *Generator) renderTemplate(pkgName string, comp *ComponentInfo) ([]byte, error) {
	tmpl, err := template.New("hx").Funcs(template.FuncMap{
		"title":       strings.Title,
		"lower":       strings.ToLower,
		"upper":       strings.ToUpper,
		"camelToTitle": camelToTitle,
		"encodeField": encodeFieldCode,
		"decodeField": decodeFieldCode,
	}).Parse(hxTemplate)
	if err != nil {
		return nil, err
	}

	data := struct {
		Package   string
		Component *ComponentInfo
	}{
		Package:   pkgName,
		Component: comp,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// camelToTitle converts "handleEdit" to "Edit".
func camelToTitle(s string) string {
	s = strings.TrimPrefix(s, "handle")
	if len(s) == 0 {
		return s
	}
	return string(unicode.ToUpper(rune(s[0]))) + s[1:]
}

// encodeFieldCode generates the code to encode a field.
func encodeFieldCode(f PropField) string {
	if f.Exclude {
		return ""
	}

	key := f.Tag
	if key == "" {
		key = strings.ToLower(f.Name)
	}

	// Handle omitempty
	if f.OmitEmpty {
		switch f.Type {
		case "string":
			return fmt.Sprintf(`if p.%s != "" { m["%s"] = p.%s }`, f.Name, key, f.Name)
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64":
			return fmt.Sprintf(`if p.%s != 0 { m["%s"] = p.%s }`, f.Name, key, f.Name)
		case "bool":
			return fmt.Sprintf(`if p.%s { m["%s"] = p.%s }`, f.Name, key, f.Name)
		case "float32", "float64":
			return fmt.Sprintf(`if p.%s != 0 { m["%s"] = p.%s }`, f.Name, key, f.Name)
		case "time.Time":
			return fmt.Sprintf(`if !p.%s.IsZero() { m["%s"] = p.%s.Format(time.RFC3339) }`, f.Name, key, f.Name)
		case "hxcmp.Callback":
			return fmt.Sprintf(`if !p.%s.IsZero() { m["%s"] = map[string]any{"u": p.%s.URL, "t": p.%s.Target, "s": p.%s.Swap} }`, f.Name, key, f.Name, f.Name, f.Name)
		default:
			return fmt.Sprintf(`m["%s"] = p.%s`, key, f.Name)
		}
	}

	// Non-omitempty
	switch f.Type {
	case "time.Time":
		return fmt.Sprintf(`m["%s"] = p.%s.Format(time.RFC3339)`, key, f.Name)
	case "hxcmp.Callback":
		return fmt.Sprintf(`m["%s"] = map[string]any{"u": p.%s.URL, "t": p.%s.Target, "s": p.%s.Swap}`, key, f.Name, f.Name, f.Name)
	default:
		return fmt.Sprintf(`m["%s"] = p.%s`, key, f.Name)
	}
}

// decodeFieldCode generates the code to decode a field.
func decodeFieldCode(f PropField) string {
	if f.Exclude {
		return ""
	}

	key := f.Tag
	if key == "" {
		key = strings.ToLower(f.Name)
	}

	switch f.Type {
	case "string":
		return fmt.Sprintf(`if v, ok := m["%s"].(string); ok { p.%s = v }`, key, f.Name)
	case "int":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = toInt(v) }`, key, f.Name)
	case "int8":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = int8(toInt(v)) }`, key, f.Name)
	case "int16":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = int16(toInt(v)) }`, key, f.Name)
	case "int32":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = int32(toInt(v)) }`, key, f.Name)
	case "int64":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = toInt64(v) }`, key, f.Name)
	case "uint":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = uint(toInt64(v)) }`, key, f.Name)
	case "uint8":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = uint8(toInt64(v)) }`, key, f.Name)
	case "uint16":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = uint16(toInt64(v)) }`, key, f.Name)
	case "uint32":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = uint32(toInt64(v)) }`, key, f.Name)
	case "uint64":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = uint64(toInt64(v)) }`, key, f.Name)
	case "float32":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = float32(toFloat64(v)) }`, key, f.Name)
	case "float64":
		return fmt.Sprintf(`if v, ok := m["%s"]; ok { p.%s = toFloat64(v) }`, key, f.Name)
	case "bool":
		return fmt.Sprintf(`if v, ok := m["%s"].(bool); ok { p.%s = v }`, key, f.Name)
	case "time.Time":
		return fmt.Sprintf(`if v, ok := m["%s"].(string); ok { if t, err := time.Parse(time.RFC3339, v); err == nil { p.%s = t } }`, key, f.Name)
	case "hxcmp.Callback":
		return fmt.Sprintf(`if v, ok := m["%s"].(map[string]any); ok { p.%s = hxcmp.CallbackFromMap(v) }`, key, f.Name)
	default:
		return fmt.Sprintf(`// TODO: decode %s of type %s`, f.Name, f.Type)
	}
}

const hxTemplate = `// Code generated by hxcmp. DO NOT EDIT.
// Source: {{.Component.SourceFile}}

//go:build !hxcmp_ignore

package {{.Package}}

import (
	"net/http"
	"strings"
	"time"

	"github.com/pthm/hxcmp"
)

// Compile-time interface compliance
var _ hxcmp.HXComponent = (*{{.Component.TypeName}})(nil)
var _ hxcmp.Encodable = (*{{.Component.PropsType}})(nil)
var _ hxcmp.Decodable = (*{{.Component.PropsType}})(nil)

// HXEncode encodes props to a map for serialization.
func (p {{.Component.PropsType}}) HXEncode() map[string]any {
	m := make(map[string]any)
	{{- range .Component.Props}}
	{{encodeField .}}
	{{- end}}
	return m
}

// HXDecode decodes props from a map.
func (p *{{.Component.PropsType}}) HXDecode(m map[string]any) error {
	{{- range .Component.Props}}
	{{decodeField .}}
	{{- end}}
	return nil
}

// HXPrefix returns the component's URL prefix.
func (c *{{.Component.TypeName}}) HXPrefix() string {
	return c.Prefix()
}

// HXServeHTTP handles HTTP requests for this component.
func (c *{{.Component.TypeName}}) HXServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Decode props
	encoded := r.URL.Query().Get("p")
	var props {{.Component.PropsType}}
	if encoded != "" {
		if err := c.Component.Encoder().Decode(encoded, c.IsSensitive(), &props); err != nil {
			http.Error(w, "Invalid parameters", http.StatusBadRequest)
			return
		}
	}

	// Run lifecycle: Hydrate
	if err := c.Hydrate(r.Context(), &props); err != nil {
		http.Error(w, "Hydration failed", http.StatusInternalServerError)
		return
	}

	// Route to handler
	path := strings.TrimPrefix(r.URL.Path, c.HXPrefix())
	switch r.Method + " " + path {
	case "GET /", "GET ":
		c.serveRender(w, r, props)
	{{- range .Component.Actions}}
	case "{{if eq .Method ""}}POST{{else}}{{.Method}}{{end}} /{{.Name}}":
		c.serve{{camelToTitle .Handler}}(w, r, props)
	{{- end}}
	default:
		http.NotFound(w, r)
	}
}

func (c *{{.Component.TypeName}}) serveRender(w http.ResponseWriter, r *http.Request, props {{.Component.PropsType}}) {
	tmpl := c.Render(r.Context(), props)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Render(r.Context(), w)
}

{{range .Component.Actions}}
func (c *{{$.Component.TypeName}}) serve{{camelToTitle .Handler}}(w http.ResponseWriter, r *http.Request, props {{$.Component.PropsType}}) {
	result := c.{{.Handler}}(r.Context(), props, r)
	c.handleResult(w, r, result)
}
{{end}}

func (c *{{.Component.TypeName}}) handleResult(w http.ResponseWriter, r *http.Request, result hxcmp.Result[{{.Component.PropsType}}]) {
	if err := result.GetErr(); err != nil {
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}
	if status := result.GetStatus(); status != 0 {
		w.WriteHeader(status)
	}
	for k, v := range result.GetHeaders() {
		w.Header().Set(k, v)
	}
	if redirect := result.GetRedirect(); redirect != "" {
		w.Header().Set("HX-Redirect", redirect)
		return
	}
	// Handle triggers
	var triggers []string
	if cb := result.GetCallback(); cb != nil {
		triggers = append(triggers, cb.TriggerJSON())
	}
	if t := result.GetTrigger(); t != "" {
		triggers = append(triggers, t)
	}
	if len(triggers) > 0 {
		w.Header().Set("HX-Trigger", strings.Join(triggers, ", "))
	}
	if result.ShouldSkip() {
		return
	}
	// Auto-render with updated props
	tmpl := c.Render(r.Context(), result.GetProps())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Render(r.Context(), w)
}

{{range .Component.Actions}}
// {{camelToTitle .Name}} returns an action builder for the "{{.Name}}" action.
func (c *{{$.Component.TypeName}}) {{camelToTitle .Name}}(props {{$.Component.PropsType}}) *hxcmp.Action {
	return &hxcmp.Action{
		URL:    c.buildURL("{{.Name}}", props),
		Method: "{{if eq .Method ""}}POST{{else}}{{.Method}}{{end}}",
		Swap:   hxcmp.SwapOuter,
	}
}
{{end}}

func (c *{{.Component.TypeName}}) buildURL(action string, props {{.Component.PropsType}}) string {
	path := c.Prefix() + "/"
	if action != "" {
		path = c.Prefix() + "/" + action
	}

	enc := c.Component.Encoder()
	if enc == nil {
		return path
	}

	encoded, err := enc.Encode(props, c.IsSensitive())
	if err != nil {
		return path
	}

	return path + "?p=" + encoded
}

// Helper functions for type conversion
func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	default:
		return 0
	}
}

// Ensure time import is used
var _ = time.RFC3339
`
