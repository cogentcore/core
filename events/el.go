// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"goki.dev/girl/states"
	"goki.dev/glop/nptime"
)

// El represents a GUI element for purposes of event management.
// The GUI system sets the element based on events, and the
// event manager uses this info to generate appropriate events.
type El struct {
	// IsSet determines if this element has been set or not
	IsSet bool

	// Name of the element -- for debugging / trace methods only
	Name string

	// Ptr is the pointer to the element (e.g., a Widget)
	// which can be useful for the GUI system.
	Ptr any

	// State accumulates GiRl states.States corresponding to the
	// history of events associated with the current element.
	State states.States

	// Abilities has the GiRl states.Abilities for what the element
	// is capable of doing in response to events.
	Abilities states.Abilities

	// Time when element was first entered
	EnterStart nptime.Time

	// LastMove is last time an above threshold mouse move event
	// happened after EnterStart -- used for computing LongHoverStart
	LastMove nptime.Time
}

func (el *El) SetEl(nm string, ptr any, state states.States, able states.Abilities) *El {
	el.IsSet = true
	el.Name = nm
	el.Ptr = ptr
	el.State = state
	el.Abilities = able
	el.EnterStart = nptime.Time{}
	el.LastMove = nptime.Time{}

	return el
}
