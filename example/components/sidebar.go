package components

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pthm/hxcmp"
)

// SidebarProps defines the props for the Sidebar component.
type SidebarProps struct {
	CurrentStatus string         `hx:"status,omitempty"`
	OnFilter      hxcmp.Callback `hx:"cb,omitempty"`
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

// handleFilter applies filters and triggers callback.
func (c *Sidebar) handleFilter(ctx context.Context, props SidebarProps, r *http.Request) hxcmp.Result[SidebarProps] {
	if err := r.ParseForm(); err != nil {
		return hxcmp.Err(props, err)
	}

	// Update props with selected filter
	props.CurrentStatus = r.FormValue("status")

	result := hxcmp.OK(props)
	if !props.OnFilter.IsZero() {
		// Pass the current status to the callback so the todolist can filter
		result = result.Callback(props.OnFilter.WithVals(map[string]any{
			"status": props.CurrentStatus,
		}))
	}
	return result
}

// handleClear clears all filters.
func (c *Sidebar) handleClear(ctx context.Context, props SidebarProps, r *http.Request) hxcmp.Result[SidebarProps] {
	props.CurrentStatus = ""

	result := hxcmp.OK(props)
	if !props.OnFilter.IsZero() {
		// Pass empty status to clear the filter
		result = result.Callback(props.OnFilter.WithVals(map[string]any{
			"status": "",
		}))
	}
	return result.Flash(hxcmp.FlashInfo, "Filters cleared")
}
