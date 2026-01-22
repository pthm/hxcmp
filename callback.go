package hxcmp

import "encoding/json"

// Callback is a signed/encrypted reference to a component action.
// Used for child-to-parent (or directed) communication.
// Security mode inherits from the target component (signed vs encrypted).
type Callback struct {
	URL    string `json:"u"`           // Action URL
	Target string `json:"t,omitempty"` // Target selector
	Swap   string `json:"s,omitempty"` // Swap mode
}

// IsZero returns true if the callback is empty/unset.
func (cb Callback) IsZero() bool {
	return cb.URL == ""
}

// TriggerJSON returns the JSON payload for HX-Trigger header.
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
	data, _ := json.Marshal(payload)
	return string(data)
}

// CallbackFromMap reconstructs a Callback from a decoded map.
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
	return cb
}
