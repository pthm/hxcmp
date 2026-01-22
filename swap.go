package hxcmp

// SwapMode defines HTMX swap strategies.
type SwapMode string

const (
	SwapOuter       SwapMode = "outerHTML"   // Replace entire element (default)
	SwapInner       SwapMode = "innerHTML"   // Replace contents
	SwapBeforeEnd   SwapMode = "beforeend"   // Append to contents
	SwapAfterEnd    SwapMode = "afterend"    // Insert after element
	SwapBeforeBegin SwapMode = "beforebegin" // Insert before element
	SwapAfterBegin  SwapMode = "afterbegin"  // Prepend to contents
	SwapDelete      SwapMode = "delete"      // Delete element
	SwapNone        SwapMode = "none"        // No swap
)
