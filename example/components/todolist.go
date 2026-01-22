package components

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pthm/hxcmp"
)

// TodoListProps defines the props for the TodoList component.
type TodoListProps struct {
	FilterStatus string   `hx:"status,omitempty"`
	FilterTags   []string `hx:"-"` // Complex types excluded, handled via form
	// Hydrated data (not serialized)
	Todos []*Todo `hx:"-"`
}

// TodoList displays a list of todo items.
type TodoList struct {
	*hxcmp.Component[TodoListProps]
	store TodoStore
}

// NewTodoList creates a new TodoList component.
func NewTodoList(store TodoStore) *TodoList {
	c := &TodoList{
		Component: hxcmp.New[TodoListProps]("todolist"),
		store:     store,
	}
	c.Action("refresh", c.handleRefresh).Method("GET")
	return c
}

// Hydrate loads todos from the store based on filter props.
func (c *TodoList) Hydrate(ctx context.Context, props *TodoListProps) error {
	var status *Status
	if props.FilterStatus != "" {
		s := Status(props.FilterStatus)
		status = &s
	}

	var tags []Tag
	for _, t := range props.FilterTags {
		tags = append(tags, Tag(t))
	}

	props.Todos = c.store.List(status, tags)
	return nil
}

// Render produces the HTML output.
func (c *TodoList) Render(ctx context.Context, props TodoListProps) templ.Component {
	return todoListTemplate(props)
}

// handleRefresh re-renders the list (used for callbacks).
func (c *TodoList) handleRefresh(ctx context.Context, props TodoListProps, r *http.Request) hxcmp.Result[TodoListProps] {
	// Read filter status from query params (passed via callback vals)
	if status := r.URL.Query().Get("status"); status != "" {
		props.FilterStatus = status
	}
	// Hydrate runs automatically before this handler, but we've updated
	// FilterStatus so return OK to trigger re-render with new filter
	return hxcmp.OK(props)
}
