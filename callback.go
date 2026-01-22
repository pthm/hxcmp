package hxcmp

import "encoding/json"

// Callback is a signed/encrypted reference to a component action for
// child-to-parent communication.
//
// Deprecated: Use event-based communication with Trigger instead. Events are
// more HTMX-native and decouple components. This type will be removed in a
// future version.
//
// Old callback pattern:
//
//	// Parent creates callback and passes to child:
//	childProps.OnSave = c.RefreshList(props).Target("#list").AsCallback()
//
//	// Child invokes callback:
//	return hxcmp.OK(props).Callback(props.OnSave)
//
// New event pattern:
//
//	// Child emits event with data:
//	return hxcmp.OK(props).Trigger("item:saved", map[string]any{"id": item.ID})
//
//	// Parent listens in template:
//	c.RefreshList(props).OnEvent("item:saved").Attrs()
type Callback struct {
	URL    string         `json:"u"`           // Action URL with encoded props
	Target string         `json:"t,omitempty"` // Target selector
	Swap   string         `json:"s,omitempty"` // Swap mode
	Vals   map[string]any `json:"v,omitempty"` // Dynamic values to append as query params
}

// IsZero returns true if the callback is empty/unset.
//
// Deprecated: Callbacks are deprecated. Use Trigger with data instead.
func (cb Callback) IsZero() bool {
	return cb.URL == ""
}

// TriggerJSON returns the JSON payload for HX-Trigger header.
//
// Deprecated: Callbacks are deprecated. Use Trigger with data instead.
func (cb Callback) TriggerJSON() string {
	data, _ := json.Marshal(map[string]any{"hxcmp:callback": cb.toMap()})
	return string(data)
}

// toMap converts the callback to a map for JSON serialization.
func (cb Callback) toMap() map[string]any {
	m := map[string]any{"url": cb.URL}
	if cb.Target != "" {
		m["target"] = cb.Target
	}
	if cb.Swap != "" {
		m["swap"] = cb.Swap
	}
	if len(cb.Vals) > 0 {
		m["vals"] = cb.Vals
	}
	return m
}

// CallbackFromMap reconstructs a Callback from a decoded map.
//
// Deprecated: Callbacks are deprecated. Use Trigger with data instead.
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
//
// Deprecated: Callbacks are deprecated. Use Trigger with data instead:
//
//	return hxcmp.OK(props).Trigger("filter:changed", map[string]any{"status": "pending"})
func (cb Callback) WithVals(vals map[string]any) Callback {
	cb.Vals = vals
	return cb
}
