package hxcmp

// Result[P] is returned from action handlers.
// It provides a fluent builder for specifying flash messages, redirects, callbacks, etc.
type Result[P any] struct {
	props    P
	err      error
	redirect string
	flashes  []Flash
	trigger  string
	callback *Callback
	headers  map[string]string
	status   int
	skip     bool
}

// OK creates a success result that will auto-render with the given props.
func OK[P any](props P) Result[P] {
	return Result[P]{props: props}
}

// Err creates an error result. The error will be passed to the registry's OnError handler.
func Err[P any](props P, err error) Result[P] {
	return Result[P]{props: props, err: err}
}

// Skip creates a result indicating the handler wrote its own response.
// No auto-render will occur.
func Skip[P any]() Result[P] {
	return Result[P]{skip: true}
}

// Redirect creates a result that will redirect via HX-Redirect header.
func Redirect[P any](url string) Result[P] {
	var zero P
	return Result[P]{props: zero, redirect: url}
}

// Flash adds a flash message (toast notification) to the result.
func (r Result[P]) Flash(level, message string) Result[P] {
	r.flashes = append(r.flashes, Flash{Level: level, Message: message})
	return r
}

// Callback triggers a parent callback.
func (r Result[P]) Callback(cb Callback) Result[P] {
	r.callback = &cb
	return r
}

// Trigger emits an event via HX-Trigger header.
func (r Result[P]) Trigger(event string) Result[P] {
	r.trigger = event
	return r
}

// Header sets a response header.
func (r Result[P]) Header(key, value string) Result[P] {
	if r.headers == nil {
		r.headers = make(map[string]string)
	}
	r.headers[key] = value
	return r
}

// Status sets the HTTP status code.
func (r Result[P]) Status(code int) Result[P] {
	r.status = code
	return r
}

// GetProps returns the props from the result.
func (r Result[P]) GetProps() P {
	return r.props
}

// GetErr returns the error from the result.
func (r Result[P]) GetErr() error {
	return r.err
}

// GetRedirect returns the redirect URL.
func (r Result[P]) GetRedirect() string {
	return r.redirect
}

// GetFlashes returns the flash messages.
func (r Result[P]) GetFlashes() []Flash {
	return r.flashes
}

// GetTrigger returns the trigger event.
func (r Result[P]) GetTrigger() string {
	return r.trigger
}

// GetCallback returns the callback.
func (r Result[P]) GetCallback() *Callback {
	return r.callback
}

// GetHeaders returns the response headers.
func (r Result[P]) GetHeaders() map[string]string {
	return r.headers
}

// GetStatus returns the HTTP status code.
func (r Result[P]) GetStatus() int {
	return r.status
}

// ShouldSkip returns whether the handler wrote its own response.
func (r Result[P]) ShouldSkip() bool {
	return r.skip
}
