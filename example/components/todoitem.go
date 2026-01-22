package components

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pthm/hxcmp"
)

// TodoItemProps defines the props for the TodoItem component.
type TodoItemProps struct {
	TodoID   string         `hx:"id"`
	OnChange hxcmp.Callback `hx:"cb,omitempty"`
	// Hydrated data (not serialized)
	Todo *Todo `hx:"-"`
}

// TodoItem handles individual todo item actions.
type TodoItem struct {
	*hxcmp.Component[TodoItemProps]
	store TodoStore
}

// NewTodoItem creates a new TodoItem component.
func NewTodoItem(store TodoStore) *TodoItem {
	c := &TodoItem{
		Component: hxcmp.New[TodoItemProps]("todoitem"),
		store:     store,
	}
	c.Action("toggle", c.handleToggle)
	c.Action("delete", c.handleDelete).Method(http.MethodDelete)
	c.Action("edit", c.handleEdit)
	return c
}

// Hydrate loads the todo from the store.
func (c *TodoItem) Hydrate(ctx context.Context, props *TodoItemProps) error {
	props.Todo = c.store.Get(props.TodoID)
	return nil
}

// Render produces the HTML output.
func (c *TodoItem) Render(ctx context.Context, props TodoItemProps) templ.Component {
	return todoItemTemplate(c, props)
}

// handleToggle toggles the todo's completed status.
func (c *TodoItem) handleToggle(ctx context.Context, props TodoItemProps, r *http.Request) hxcmp.Result[TodoItemProps] {
	if !c.store.Toggle(props.TodoID) {
		return hxcmp.Err(props, hxcmp.ErrNotFound)
	}

	result := hxcmp.OK(props)
	if !props.OnChange.IsZero() {
		result = result.Callback(props.OnChange)
	}
	return result.Flash(hxcmp.FlashSuccess, "Todo updated!")
}

// handleDelete removes the todo.
func (c *TodoItem) handleDelete(ctx context.Context, props TodoItemProps, r *http.Request) hxcmp.Result[TodoItemProps] {
	if !c.store.Delete(props.TodoID) {
		return hxcmp.Err(props, hxcmp.ErrNotFound)
	}

	result := hxcmp.OK(props)
	if !props.OnChange.IsZero() {
		result = result.Callback(props.OnChange)
	}
	return result.Flash(hxcmp.FlashSuccess, "Todo deleted!")
}

// handleEdit updates the todo's title and description.
func (c *TodoItem) handleEdit(ctx context.Context, props TodoItemProps, r *http.Request) hxcmp.Result[TodoItemProps] {
	if err := r.ParseForm(); err != nil {
		return hxcmp.Err(props, err)
	}

	title := r.FormValue("title")
	description := r.FormValue("description")

	if !c.store.Update(props.TodoID, title, description, nil) {
		return hxcmp.Err(props, hxcmp.ErrNotFound)
	}

	result := hxcmp.OK(props)
	if !props.OnChange.IsZero() {
		result = result.Callback(props.OnChange)
	}
	return result.Flash(hxcmp.FlashSuccess, "Todo updated!")
}
