package components

import "github.com/pthm/hxcmp"

// Init initializes all components with their dependencies and registers them.
// Call this once at application startup before handling requests.
//
// Usage:
//
//	reg := hxcmp.NewRegistry(key)
//	hxcmp.SetDefault(reg)
//	components.Init(store, reg)
func Init(store TodoStore, reg *hxcmp.Registry) {
	reg.Add(
		NewTodoList(store),
		NewTodoItem(store),
		NewSidebar(store),
		NewAddTodo(store),
		NewTaskDetail(store),
		NewStats(store),
	)
}
