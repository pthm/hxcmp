package main

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/pthm/hxcmp/example/components"
)

// Store is an in-memory todo store that implements components.TodoStore.
type Store struct {
	mu     sync.RWMutex
	todos  map[string]*components.Todo
	nextID int
}

// NewStore creates a new store with sample data.
func NewStore() *Store {
	s := &Store{
		todos:  make(map[string]*components.Todo),
		nextID: 1,
	}

	// Add sample todos
	s.Add("Buy groceries", "Milk, eggs, bread", []components.Tag{components.TagPersonal})
	s.Add("Review PR #123", "Check the authentication changes", []components.Tag{components.TagWork, components.TagUrgent})
	s.Add("Write documentation", "Update API docs for v2", []components.Tag{components.TagWork})
	s.Add("Call dentist", "Schedule annual checkup", []components.Tag{components.TagPersonal, components.TagLater})
	s.Add("Fix login bug", "Users can't reset passwords", []components.Tag{components.TagWork, components.TagUrgent})

	return s
}

// Add creates a new todo and returns its ID.
func (s *Store) Add(title, description string, tags []components.Tag) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("todo-%d", s.nextID)
	s.nextID++

	now := time.Now()
	s.todos[id] = &components.Todo{
		ID:          id,
		Title:       title,
		Description: description,
		Status:      components.StatusPending,
		Tags:        tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return id
}

// Get returns a todo by ID.
func (s *Store) Get(id string) *components.Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.todos[id]
}

// Update updates a todo's fields.
func (s *Store) Update(id string, title, description string, tags []components.Tag) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.todos[id]
	if !ok {
		return false
	}

	if title != "" {
		todo.Title = title
	}
	if description != "" {
		todo.Description = description
	}
	if tags != nil {
		todo.Tags = tags
	}
	todo.UpdatedAt = time.Now()
	return true
}

// Toggle toggles the completed status of a todo.
func (s *Store) Toggle(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	todo, ok := s.todos[id]
	if !ok {
		return false
	}

	if todo.Status == components.StatusCompleted {
		todo.Status = components.StatusPending
	} else {
		todo.Status = components.StatusCompleted
	}
	todo.UpdatedAt = time.Now()
	return true
}

// Delete removes a todo by ID.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.todos[id]; !ok {
		return false
	}
	delete(s.todos, id)
	return true
}

// List returns all todos, optionally filtered.
func (s *Store) List(status *components.Status, tags []components.Tag) []*components.Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*components.Todo
	for _, todo := range s.todos {
		// Filter by status
		if status != nil && todo.Status != *status {
			continue
		}

		// Filter by tags (must have ALL specified tags)
		if len(tags) > 0 {
			hasAllTags := true
			for _, tag := range tags {
				if !todo.HasTag(tag) {
					hasAllTags = false
					break
				}
			}
			if !hasAllTags {
				continue
			}
		}

		result = append(result, todo)
	}

	// Sort by CreatedAt descending (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result
}

// Stats returns statistics about the todos.
func (s *Store) Stats() components.TodoStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := components.TodoStats{
		ByTag: make(map[components.Tag]int),
	}

	for _, todo := range s.todos {
		stats.Total++
		if todo.Status == components.StatusCompleted {
			stats.Completed++
		} else {
			stats.Pending++
		}
		for _, tag := range todo.Tags {
			stats.ByTag[tag]++
		}
	}

	return stats
}
