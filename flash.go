package hxcmp

import (
	"context"
	"html"
	"io"
	"strings"

	"github.com/a-h/templ"
)

// Flash levels for toast notifications.
const (
	FlashSuccess = "success"
	FlashError   = "error"
	FlashWarning = "warning"
	FlashInfo    = "info"
)

// Flash represents a one-time notification.
type Flash struct {
	Level   string // success, error, warning, info
	Message string
}

// RenderFlashesOOB renders flashes as OOB swap HTML.
// Appends to #toasts container using beforeend swap.
func RenderFlashesOOB(flashes []Flash) string {
	if len(flashes) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`<div id="toasts" hx-swap-oob="beforeend">`)

	for _, f := range flashes {
		sb.WriteString(`<div class="toast toast-`)
		sb.WriteString(html.EscapeString(f.Level))
		sb.WriteString(`" data-auto-dismiss="3000">`)
		sb.WriteString(html.EscapeString(f.Message))
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

// ToastContainer returns a templ component for the toast container.
// Add this to your layout template.
func ToastContainer() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<div id="toasts" class="toast-container"></div>`)
		return err
	})
}
