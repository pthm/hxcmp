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

// handleRefresh re-renders the list with current URL state.
func (c *TodoList) handleRefresh(ctx context.Context, props TodoListProps, r *http.Request) hxcmp.Result[TodoListProps] {
	// SyncURL injects "status" from browser URL into request params
	props.FilterStatus = r.URL.Query().Get("status")

	// Re-hydrate with updated filter to fetch correct todos
	if err := c.Hydrate(r.Context(), &props); err != nil {
		return hxcmp.Err(props, err)
	}

	return hxcmp.OK(props)
}
