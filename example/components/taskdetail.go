package components

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pthm/hxcmp"
)

// TaskDetailProps defines the props for the TaskDetail component.
type TaskDetailProps struct {
	TodoID       string `hx:"id"`
	EditingTitle bool   `hx:"et"` // Track if we're in title editing mode
	// Hydrated data (not serialized)
	Todo *Todo `hx:"-"`
}

// TaskDetail displays full details for a single todo.
type TaskDetail struct {
	*hxcmp.Component[TaskDetailProps]
	store TodoStore
}

// NewTaskDetail creates a new TaskDetail component.
func NewTaskDetail(store TodoStore) *TaskDetail {
	c := &TaskDetail{
		Component: hxcmp.New[TaskDetailProps]("taskdetail"),
		store:     store,
	}
	c.Action("update", c.handleUpdate)
	c.Action("toggle", c.handleToggle)
	c.Action("delete", c.handleDelete).Method(http.MethodDelete)
	// Inline title editing actions
	c.Action("editTitle", c.handleEditTitle).Method(http.MethodGet)
	c.Action("saveTitle", c.handleSaveTitle)
	c.Action("cancelEdit", c.handleCancelEdit).Method(http.MethodGet)
	return c
}

// Hydrate loads the todo from the store.
func (c *TaskDetail) Hydrate(ctx context.Context, props *TaskDetailProps) error {
	props.Todo = c.store.Get(props.TodoID)
	return nil
}

// Render produces the HTML output.
func (c *TaskDetail) Render(ctx context.Context, props TaskDetailProps) templ.Component {
	return taskDetailTemplate(c, props)
}

// handleUpdate updates the todo.
func (c *TaskDetail) handleUpdate(ctx context.Context, props TaskDetailProps, r *http.Request) hxcmp.Result[TaskDetailProps] {
	if err := r.ParseForm(); err != nil {
		return hxcmp.Err(props, err)
	}

	title := r.FormValue("title")
	description := r.FormValue("description")

	// Parse tags
	var tags []Tag
	for _, t := range r.Form["tags"] {
		tags = append(tags, Tag(t))
	}

	if !c.store.Update(props.TodoID, title, description, tags) {
		return hxcmp.Err(props, hxcmp.ErrNotFound)
	}

	return hxcmp.OK(props).Flash(hxcmp.FlashSuccess, "Todo updated!").Trigger("todo-updated")
}

// handleToggle toggles the todo's status.
func (c *TaskDetail) handleToggle(ctx context.Context, props TaskDetailProps, r *http.Request) hxcmp.Result[TaskDetailProps] {
	if !c.store.Toggle(props.TodoID) {
		return hxcmp.Err(props, hxcmp.ErrNotFound)
	}
	return hxcmp.OK(props).Flash(hxcmp.FlashSuccess, "Status updated!")
}

// handleDelete deletes the todo and redirects to list.
func (c *TaskDetail) handleDelete(ctx context.Context, props TaskDetailProps, r *http.Request) hxcmp.Result[TaskDetailProps] {
	if !c.store.Delete(props.TodoID) {
		return hxcmp.Err(props, hxcmp.ErrNotFound)
	}
	return hxcmp.Redirect[TaskDetailProps]("/").Flash(hxcmp.FlashSuccess, "Todo deleted!")
}

// handleEditTitle switches to title editing mode.
func (c *TaskDetail) handleEditTitle(ctx context.Context, props TaskDetailProps, r *http.Request) hxcmp.Result[TaskDetailProps] {
	props.EditingTitle = true
	return hxcmp.OK(props)
}

// handleSaveTitle saves the title and exits editing mode.
func (c *TaskDetail) handleSaveTitle(ctx context.Context, props TaskDetailProps, r *http.Request) hxcmp.Result[TaskDetailProps] {
	if err := r.ParseForm(); err != nil {
		return hxcmp.Err(props, err)
	}

	title := r.FormValue("title")
	if title == "" {
		// Keep in edit mode on validation error
		props.EditingTitle = true
		return hxcmp.OK(props).Flash(hxcmp.FlashError, "Title cannot be empty")
	}

	// Update only the title (preserve description and tags)
	if !c.store.Update(props.TodoID, title, props.Todo.Description, props.Todo.Tags) {
		return hxcmp.Err(props, hxcmp.ErrNotFound)
	}

	// Exit edit mode - Hydrate runs automatically before render to refresh Todo
	props.EditingTitle = false

	return hxcmp.OK(props).Flash(hxcmp.FlashSuccess, "Title updated!").Trigger("todo-updated")
}

// handleCancelEdit exits title editing mode without saving.
func (c *TaskDetail) handleCancelEdit(ctx context.Context, props TaskDetailProps, r *http.Request) hxcmp.Result[TaskDetailProps] {
	props.EditingTitle = false
	return hxcmp.OK(props)
}
