package components

import (
	"context"
	"time"

	"github.com/a-h/templ"
	"github.com/pthm/hxcmp"
)

// StatsProps defines the props for the Stats component.
type StatsProps struct {
	// Hydrated data
	Stats TodoStats `hx:"-"`
}

// Stats displays statistics about todos.
// Demonstrates lazy loading and polling.
type Stats struct {
	*hxcmp.Component[StatsProps]
	store TodoStore
}

// NewStats creates a new Stats component.
func NewStats(store TodoStore) *Stats {
	c := &Stats{
		Component: hxcmp.New[StatsProps]("stats"),
		store:     store,
	}
	return c
}

// Hydrate loads stats from the store.
func (c *Stats) Hydrate(ctx context.Context, props *StatsProps) error {
	props.Stats = c.store.Stats()
	return nil
}

// Render produces the HTML output.
func (c *Stats) Render(ctx context.Context, props StatsProps) templ.Component {
	return statsTemplate(c, props)
}

// RefreshAction returns an action for polling refresh.
// Also listens for todo:added events for immediate updates.
func (c *Stats) RefreshAction(props StatsProps) *hxcmp.Action {
	return c.Refresh(props).Every(5 * time.Second).OnEvent("todo:added")
}
