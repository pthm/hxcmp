package components

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/pthm/hxcmp"
)

// CounterProps holds the counter state.
type CounterProps struct {
	Count int
}

// Counter is a simple stateless counter component.
// State is stored entirely in the encoded props.
type Counter struct {
	*hxcmp.Component[CounterProps]
}

// NewCounter creates a new Counter component.
func NewCounter() *Counter {
	c := &Counter{
		Component: hxcmp.New[CounterProps]("counter"),
	}
	c.Action("increment", c.handleIncrement)
	c.Action("decrement", c.handleDecrement)
	c.Action("reset", c.handleReset)
	return c
}

// Hydrate is a no-op since the counter has no external dependencies.
func (c *Counter) Hydrate(ctx context.Context, props *CounterProps) error {
	return nil
}

// Render produces the HTML output.
func (c *Counter) Render(ctx context.Context, props CounterProps) templ.Component {
	return counterTemplate(c, props)
}

func (c *Counter) handleIncrement(ctx context.Context, props CounterProps, r *http.Request) hxcmp.Result[CounterProps] {
	props.Count++
	return hxcmp.OK(props)
}

func (c *Counter) handleDecrement(ctx context.Context, props CounterProps, r *http.Request) hxcmp.Result[CounterProps] {
	props.Count--
	return hxcmp.OK(props)
}

func (c *Counter) handleReset(ctx context.Context, props CounterProps, r *http.Request) hxcmp.Result[CounterProps] {
	props.Count = 0
	return hxcmp.OK(props).Flash(hxcmp.FlashSuccess, "Counter reset!")
}
