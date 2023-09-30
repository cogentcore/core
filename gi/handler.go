// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

// Handlers registers lists of handler functions
// to receive different event types.
// Handlers are methods on a given receiver object,
// which is passed in as an argument.
// Handlers are registered _once for each type_, whereas
// Listeners are closure methods with all context captured,
// registered on specific objects.
type Handlers map[Types][]func(recv Widget, ev Event)

// Init ensures that map is constructed
func (ls *Handlers) Init() {
	if *ls != nil {
		return
	}
	*ls = make(map[Types][]func(Widget, Event))
}

// Add adds a function for given type
func (ls *Handlers) Add(typ Types, fun func(Widget, Event)) {
	ls.Init()
	ets := (*ls)[typ]
	ets = append(ets, fun)
	(*ls)[typ] = ets
}

// Call calls all functions for given event.
// It goes in _reverse_ order to the last functions added are the first called
// and it stops when the event is marked as Handled.  This allows for a natural
// and optional override behavior, as compared to requiring more complex
// priority-based mechanisms.
func (ls *Handlers) Call(recv Widget, ev Event) {
	if ev.IsHandled() {
		return
	}
	typ := ev.Type()
	ets := (*ls)[typ]
	n := len(ets)
	if n == 0 {
		return
	}
	for i := n - 1; i >= 0; i-- {
		fun := ets[i]
		fun(recv, ev)
		if ev.IsHandled() {
			break
		}
	}
}
