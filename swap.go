package hxcmp

// SwapMode defines HTMX swap strategies for how response HTML replaces the target.
//
// Each mode corresponds to an HTMX hx-swap value. The default is SwapOuter.
//
// See https://htmx.org/attributes/hx-swap/ for visual examples.
type SwapMode string

const (
	// SwapOuter replaces the entire element including its tag (outerHTML).
	// This is the default swap mode.
	SwapOuter SwapMode = "outerHTML"

	// SwapInner replaces only the element's contents, preserving the outer tag (innerHTML).
	SwapInner SwapMode = "innerHTML"

	// SwapBeforeEnd appends the response to the end of the target's contents (before closing tag).
	// Useful for adding items to lists.
	SwapBeforeEnd SwapMode = "beforeend"

	// SwapAfterEnd inserts the response after the target element (as next sibling).
	SwapAfterEnd SwapMode = "afterend"

	// SwapBeforeBegin inserts the response before the target element (as previous sibling).
	SwapBeforeBegin SwapMode = "beforebegin"

	// SwapAfterBegin prepends the response to the start of the target's contents (after opening tag).
	// Useful for prepending items to lists.
	SwapAfterBegin SwapMode = "afterbegin"

	// SwapDelete removes the target element entirely.
	// Response content is ignored.
	SwapDelete SwapMode = "delete"

	// SwapNone performs no swap - response is discarded.
	// Useful for actions with only side effects or when using events/callbacks.
	SwapNone SwapMode = "none"
)
