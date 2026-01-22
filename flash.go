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

// Flash represents a one-time notification message.
//
// Flash messages are rendered as out-of-band (OOB) swaps that append to
// the #toasts container. The hxcmp JavaScript extension automatically
// dismisses toasts after a configurable delay (via data-auto-dismiss).
//
// Typical usage:
//
//	return hxcmp.OK(props).Flash("success", "Item saved!")
//	return hxcmp.Err(props, err).Flash("error", "Save failed")
//
// Multiple flashes can be returned from a single action - each appears
// as a separate toast notification.
type Flash struct {
	Level   string // success, error, warning, info
	Message string
}

// RenderFlashesOOB renders flashes as OOB swap HTML.
//
// Generates HTML that appends to the #toasts container using the
// hx-swap-oob="beforeend" attribute. Called by generated code when
// processing Result with flashes.
//
// The data-auto-dismiss attribute is read by the hxcmp JavaScript
// extension, which removes the toast after the specified delay (milliseconds).
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
//
// Add this to your layout template (typically near the end of <body>):
//
//	@hxcmp.ToastContainer()
//
// The container is targeted by OOB swaps from flash messages. It should
// be styled with CSS to position toasts (typically fixed top-right or
// bottom-right).
func ToastContainer() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, `<div id="toasts" class="toast-container"></div>`)
		return err
	})
}
