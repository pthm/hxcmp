package components

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pthm/hxcmp"
)

// SidebarProps defines the props for the Sidebar component.
type SidebarProps struct {
	CurrentStatus string `hx:"status,omitempty"`
	// Note: CurrentTags would need special handling for arrays
}

// Sidebar displays filters for todos.
type Sidebar struct {
	*hxcmp.Component[SidebarProps]
	store TodoStore
}

// NewSidebar creates a new Sidebar component.
func NewSidebar(store TodoStore) *Sidebar {
	c := &Sidebar{
		Component: hxcmp.New[SidebarProps]("sidebar"),
		store:     store,
	}
	c.Action("filter", c.handleFilter)
	c.Action("clear", c.handleClear)
	return c
}

// Hydrate prepares the component (no-op for sidebar).
func (c *Sidebar) Hydrate(ctx context.Context, props *SidebarProps) error {
	return nil
}

// Render produces the HTML output.
func (c *Sidebar) Render(ctx context.Context, props SidebarProps) templ.Component {
	return sidebarTemplate(c, props)
}

// handleFilter applies filters and emits filter:changed event.
func (c *Sidebar) handleFilter(ctx context.Context, props SidebarProps, r *http.Request) hxcmp.Result[SidebarProps] {
	if err := r.ParseForm(); err != nil {
		return hxcmp.Err(props, err)
	}

	// Update props with selected filter
	props.CurrentStatus = r.FormValue("status")

	// Emit event so listeners (e.g., TodoList) can refresh with new filter
	return hxcmp.OK(props).Trigger("filter:changed", map[string]any{
		"status": props.CurrentStatus,
	})
}

// handleClear clears all filters.
func (c *Sidebar) handleClear(ctx context.Context, props SidebarProps, r *http.Request) hxcmp.Result[SidebarProps] {
	props.CurrentStatus = ""

	// Emit event with empty status to clear the filter
	return hxcmp.OK(props).
		Trigger("filter:changed", map[string]any{"status": ""}).
		Flash(hxcmp.FlashInfo, "Filters cleared")
}
