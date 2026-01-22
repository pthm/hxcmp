package components

import (
	"context"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/pthm/hxcmp"
)

// AddTodoProps defines the props for the AddTodo component.
type AddTodoProps struct {
	OnAdd hxcmp.Callback `hx:"cb,omitempty"`
}

// AddTodo handles adding new todos.
type AddTodo struct {
	*hxcmp.Component[AddTodoProps]
	store TodoStore
}

// NewAddTodo creates a new AddTodo component.
func NewAddTodo(store TodoStore) *AddTodo {
	c := &AddTodo{
		Component: hxcmp.New[AddTodoProps]("addtodo"),
		store:     store,
	}
	c.Action("add", c.handleAdd)
	return c
}

// Hydrate prepares the component (no-op for form).
func (c *AddTodo) Hydrate(ctx context.Context, props *AddTodoProps) error {
	return nil
}

// Render produces the HTML output.
func (c *AddTodo) Render(ctx context.Context, props AddTodoProps) templ.Component {
	return addTodoTemplate(c, props)
}

// handleAdd creates a new todo.
func (c *AddTodo) handleAdd(ctx context.Context, props AddTodoProps, r *http.Request) hxcmp.Result[AddTodoProps] {
	if err := r.ParseForm(); err != nil {
		return hxcmp.Err(props, err)
	}

	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))

	if title == "" {
		return hxcmp.OK(props).Flash(hxcmp.FlashError, "Title is required")
	}

	// Parse tags
	var tags []Tag
	for _, t := range r.Form["tags"] {
		tags = append(tags, Tag(t))
	}

	c.store.Add(title, description, tags)

	result := hxcmp.OK(props)
	if !props.OnAdd.IsZero() {
		result = result.Callback(props.OnAdd)
	}
	// Trigger event for loosely-coupled listeners (e.g., Stats component)
	return result.Flash(hxcmp.FlashSuccess, "Todo added!").Trigger("todo:added")
}
