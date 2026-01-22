package components

import "github.com/pthm/hxcmp"

// C holds all component instances, accessed via generated getter functions.
// Usage: components.TodoListCmp().Render(ctx, props)
var C struct {
	TodoList   *TodoList
	TodoItem   *TodoItem
	Sidebar    *Sidebar
	AddTodo    *AddTodo
	TaskDetail *TaskDetail
	Stats      *Stats
}

// Init initializes all components with their dependencies and registers them.
// Call this once at application startup before handling requests.
func Init(store TodoStore, reg *hxcmp.Registry) {
	// Create components
	C.TodoList = NewTodoList(store)
	C.TodoItem = NewTodoItem(store)
	C.Sidebar = NewSidebar(store)
	C.AddTodo = NewAddTodo(store)
	C.TaskDetail = NewTaskDetail(store)
	C.Stats = NewStats(store)

	// Register with hxcmp registry
	reg.Add(C.TodoList, C.TodoItem, C.Sidebar, C.AddTodo, C.TaskDetail, C.Stats)
}
