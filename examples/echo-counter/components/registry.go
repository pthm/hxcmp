package components

import "github.com/pthm/hxcmp"

// Init registers all components with the registry.
func Init(reg *hxcmp.Registry) {
	reg.Add(NewCounter())
}
