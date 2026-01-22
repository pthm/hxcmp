package hxcmp

import "encoding/json"

// Callback is a signed/encrypted reference to a component action for
// child-to-parent communication.
//
// Callbacks enable directed communication where a parent component passes
// an action reference in props, and the child invokes it after completing
// an action. This is safer and more explicit than event broadcasting.
//
// Example:
//
//	// Parent creates callback and passes to child:
//	childProps.OnSave = c.RefreshList(props).Target("#list").AsCallback()
//
//	// Child invokes callback after saving:
//	func (c *Child) handleSave(ctx context.Context, props Props) Result[Props] {
//	    // ... save logic ...
//	    return hxcmp.OK(props).Callback(props.OnSave)
//	}
//
// Security mode (signed vs encrypted) inherits from the target component.
// The callback URL contains encoded props, so it's as secure as any other
// component action.
//
// Callbacks are transmitted via HX-Trigger header and processed client-side
// by the hxcmp JavaScript extension, which triggers the callback's URL with
// the specified target and swap mode.
type Callback struct {
	URL    string         `json:"u"`           // Action URL with encoded props
	Target string         `json:"t,omitempty"` // Target selector
	Swap   string         `json:"s,omitempty"` // Swap mode
	Vals   map[string]any `json:"v,omitempty"` // Dynamic values to append as query params
}

// IsZero returns true if the callback is empty/unset.
//
// Use this to check if a callback was provided before invoking it:
//
//	if !props.OnSave.IsZero() {
//	    return hxcmp.OK(props).Callback(props.OnSave)
//	}
func (cb Callback) IsZero() bool {
	return cb.URL == ""
}

// TriggerJSON returns the JSON payload for HX-Trigger header.
//
// The hxcmp JavaScript extension listens for "hxcmp:callback" events
// and triggers the callback URL. This is called by generated code when
// processing Result.Callback().
func (cb Callback) TriggerJSON() string {
	payload := map[string]any{
		"hxcmp:callback": map[string]any{
			"url": cb.URL,
		},
	}
	if cb.Target != "" {
		payload["hxcmp:callback"].(map[string]any)["target"] = cb.Target
	}
	if cb.Swap != "" {
		payload["hxcmp:callback"].(map[string]any)["swap"] = cb.Swap
	}
	if len(cb.Vals) > 0 {
		payload["hxcmp:callback"].(map[string]any)["vals"] = cb.Vals
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

// CallbackFromMap reconstructs a Callback from a decoded map.
//
// Used when callbacks are embedded in props and need to be deserialized
// from the generic decoder output.
func CallbackFromMap(m map[string]any) Callback {
	cb := Callback{}
	if v, ok := m["u"].(string); ok {
		cb.URL = v
	}
	if v, ok := m["t"].(string); ok {
		cb.Target = v
	}
	if v, ok := m["s"].(string); ok {
		cb.Swap = v
	}
	if v, ok := m["v"].(map[string]any); ok {
		cb.Vals = v
	}
	return cb
}

// WithVals returns a copy of the callback with the specified values.
// These values will be appended as query parameters when the callback is triggered.
//
// Example:
//
//	return hxcmp.OK(props).Callback(props.OnFilter.WithVals(map[string]any{"status": "pending"}))
func (cb Callback) WithVals(vals map[string]any) Callback {
	cb.Vals = vals
	return cb
}
