// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/mobile/event:
//
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package window defines events associated with windows -- including changes
// in the dimensions, physical resolution and orientation of the app's window,
// and iconify, open and close events.
package window

//go:generate enumgen

import (
	"fmt"

	"goki.dev/goosi"
)

// window.Event reports on actions taken on a window.
// The goosi.Window Flags and other state information
// will always be updated prior to this event being sent,
// so those should be consulted directly for the new current state.
type Event struct {
	goosi.EventBase

	// Action taken on the window -- what has changed.  Window state fields
	// have current values.
	Action Actions
}

func NewEvent(act Actions) *Event {
	ev := &Event{}
	ev.Action = act
	ev.Typ = goosi.WindowEvent
	return ev
}

func NewResizeEvent() *Event {
	ev := &Event{}
	ev.Action = Resize
	ev.Typ = goosi.WindowResizeEvent
	return ev
}

func NewPaintEvent() *Event {
	ev := &Event{}
	ev.Action = Paint
	ev.Typ = goosi.WindowPaintEvent
	return ev
}

func NewShowEvent() *Event {
	ev := &Event{}
	ev.Action = Show
	ev.Typ = goosi.WindowShowEvent
	return ev
}

func NewFocusEvent(act Actions) *Event {
	ev := &Event{}
	ev.Action = act
	ev.Typ = goosi.WindowFocusEvent
	return ev
}

func (ev *Event) HasPos() bool {
	return false
}

func (ev *Event) OnFocus() bool {
	return false
}

func (ev *Event) String() string {
	return fmt.Sprintf("Type: %v Action: %v  Time: %v", ev.Type(), ev.Action, ev.Time())
}

// Actions is the action taken on the window by the user.
type Actions int32 //enums:enum

const (
	// Close means that the window is about to close, but has not yet closed.
	Close Actions = iota

	// Minimize means that the window has been iconified / miniaturized / is no
	// longer visible.
	Minimize

	// Resize means that the window was resized, including changes in DPI
	// associated with moving to a new screen.  Position may have also changed
	// too.  Requires a redraw.
	Resize

	// Move means that the window was moved but NOT resized or changed in any
	// other way -- does not require a redraw, but anything tracking positions
	// will want to update.
	Move

	// Focus indicates that the window has been activated for receiving user
	// input.
	Focus

	// DeFocus indicates that the window is no longer activated for
	// receiving input.
	DeFocus

	// Paint events are sent to drive updating of the window at
	// regular FPS frames per second intervals.
	Paint

	// Show is for the WindowShow event -- sent by the system 1 second
	// after the window has opened, to ensure that full rendering
	// is completed with the proper size, and to trigger one-time actions such as
	// configuring the main menu after the window has opened.
	Show

	// ScreenUpdate occurs when any of the screen information is updated
	// This event is sent to the first window on the list of active windows
	// and it should then perform any necessary updating
	ScreenUpdate
)
