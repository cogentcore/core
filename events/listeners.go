// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import "slices"

// Listeners registers lists of event listener functions
// to receive different event types.
// Listeners are closure methods with all context captured.
// Functions are called in *reverse* order of when they are added:
// First In, Last Called, so that "base" functions are typically
// added first, and then can be overridden by later-added ones.
// Call SetHandled() on the event to stop further propagation.
type Listeners map[Types][]func(ev Event)

// Init ensures that map is constructed
func (ls *Listeners) Init() {
	if *ls != nil {
		return
	}
	*ls = make(map[Types][]func(Event))
}

// Add adds a function for given type, to the end of the current stack,
// so that it will be called before everything else already on the stack.
func (ls *Listeners) Add(typ Types, fun func(Event)) {
	ls.Init()
	ets := (*ls)[typ]
	ets = append(ets, fun)
	(*ls)[typ] = ets
}

// AddLastCall adds a function for given type, to the start of the current
// stack, so that it will be called *after* everything else on the stack.
func (ls *Listeners) AddLastCall(typ Types, fun func(Event)) {
	ls.Init()
	ets := (*ls)[typ]
	ets = slices.Insert(ets, 0, fun)
	(*ls)[typ] = ets
}

// Call calls all functions for given event.
// It goes in _reverse_ order to the last functions added are the first called
// and it stops when the event is marked as Handled.  This allows for a natural
// and optional override behavior, as compared to requiring more complex
// priority-based mechanisms.
func (ls *Listeners) Call(ev Event) {
	if ev.IsHandled() {
		return
	}
	typ := ev.Type()
	ets := (*ls)[typ]
	n := len(ets)
	for i := n - 1; i >= 0; i-- {
		fun := ets[i]
		fun(ev)
		if ev.IsHandled() {
			break
		}
	}
}
