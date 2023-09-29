// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/enums"
	"goki.dev/goosi"
)

// note: closures vs. independent functions that don't capture any surrounding variables
// are faster by roughly 10%: https://stackoverflow.com/questions/45937924/in-go-do-non-capturing-closures-harm-performance
// and result in more compact code.  Therefore, for high-use functions such as event funcs
// it does make sense to use RecvFunc and embedder logic.

// EventFunc contains priority and function for widget event processing.
type EventFunc struct {

	// type of event this is for -- also registered as EventFuncs map key
	Type goosi.EventTypes `desc:"type of event this is for -- also registered as EventFuncs map key"`

	// priority of event
	Pri EventPris `desc:"priority of event"`

	// function to process event
	Func func() `desc:"function to process event"`
}

// WidgetEvents contains the info for events processed by a Widget,
// including a map of functions per event type, and a bitmap Filter
// of events to process.
type WidgetEvents struct {

	// bitflag filter of all events that this widget processes.  If the flag is not set here, it won't receive the event.  Can selectively turn on or off event receipt while maintaining the full set of event functions.
	Filter goosi.EventTypes `desc:"bitflag filter of all events that this widget processes.  If the flag is not set here, it won't receive the event.  Can selectively turn on or off event receipt while maintaining the full set of event functions."`

	// bitflag for all events that have a function.  Filter is initialized to this by default, and can be used for restoring after turning off.
	All goosi.EventTypes `desc:"bitflag for all events that have a function.  Filter is initialized to this by default, and can be used for restoring after turning off."`

	// map of event functions -- shared by all widgets of the same type -- cannot modify after initial configuration!
	Funcs map[goosi.EventTypes]*EventFunc `desc:"map of event functions -- shared by all widgets of the same type -- cannot modify after initial configuration!"`
}

// Matches returns true if it matches given event type and priority
func (we *WidgetEvents) Matches(et goosi.EventTypes, pri EventPris) bool {
	if !we.Filter.HasFlag(et) {
		return false
	}
	ef := we.Funcs[et]
	return pri == ef.Pri
}

// HasFuncs returns true if funcs map already initialized.
// First step of AddEvents is to return if type's event funcs
// have already been added
func (we *WidgetEvents) HasFuncs() bool {
	return we.Funcs != nil
}

// AddFunc is the primary call for adding an event processing function.
// It sets the All flag in addition to adding to the map of funcs.
func (we *WidgetEvents) AddFunc(et goosi.EventTypes, pri EventPris, fun func()) *WidgetEvents {
	if we.Funcs == nil {
		we.Funcs = make(map[goosi.EventTypes]*EventFunc)
	}
	ef := &EventFunc{Type: et, Pri: pri, Func: fun}
	we.Funcs[et] = ef
	we.All.SetFlag(true, et)
	return we
}

// CopyFrom copies from other WidgetEvents -- typically the type general one.
// Also ensures that Filter = All (i.e., RecvAll)
// This is the first call in FilterEvents function per widget
func (we *WidgetEvents) CopyFrom(cp *WidgetEvents) *WidgetEvents {
	*we = *cp
	we.Filter = we.All
	return we
}

// RecvAll copies All to Filter so that all defined events are received
func (we *WidgetEvents) RecvAll() *WidgetEvents {
	we.Filter = we.All
	return we
}

// Ex excludes given event from Filter
func (we *WidgetEvents) Ex(et ...enums.BitFlag) *WidgetEvents {
	we.Filter.SetFlag(false, et...)
	return we
}
