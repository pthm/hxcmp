package components

import "time"

// Status represents the completion status of a todo.
type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
)

// Tag represents a category tag for todos.
type Tag string

const (
	TagWork     Tag = "work"
	TagPersonal Tag = "personal"
	TagUrgent   Tag = "urgent"
	TagLater    Tag = "later"
)

// AllTags returns all available tags.
func AllTags() []Tag {
	return []Tag{TagWork, TagPersonal, TagUrgent, TagLater}
}

// Todo represents a single todo item.
type Todo struct {
	ID          string
	Title       string
	Description string
	Status      Status
	Tags        []Tag
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsCompleted returns true if the todo is completed.
func (t *Todo) IsCompleted() bool {
	return t.Status == StatusCompleted
}

// HasTag returns true if the todo has the given tag.
func (t *Todo) HasTag(tag Tag) bool {
	for _, tg := range t.Tags {
		if tg == tag {
			return true
		}
	}
	return false
}

// TodoStats holds statistics about todos.
type TodoStats struct {
	Total     int
	Completed int
	Pending   int
	ByTag     map[Tag]int
}

// TodoStore is the interface for accessing todo data.
type TodoStore interface {
	Get(id string) *Todo
	List(status *Status, tags []Tag) []*Todo
	Add(title, description string, tags []Tag) string
	Update(id string, title, description string, tags []Tag) bool
	Toggle(id string) bool
	Delete(id string) bool
	Stats() TodoStats
}
