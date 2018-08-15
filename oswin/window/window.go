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

import (
	"image"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/kit"
)

// window.Event reports on actions taken on a window.  The oswin.Window Flags
// and other state information will always be updated prior to this event
// being sent, so those should be consulted directly for the new current
// state.
type Event struct {
	oswin.EventBase

	// Action taken on the window -- what has changed.  Window state fields
	// have current values.
	Action Actions
}

// Actions is the action taken on the window by the user.
type Actions int32

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

	// Paint indicates a request to repaint the window.
	Paint

	ActionsN
)

//go:generate stringer -type=Actions

var KiT_Actions = kit.Enums.AddEnum(ActionsN, false, nil)

/////////////////////////////
// oswin.Event interface

func (ev Event) Type() oswin.EventType {
	if ev.Action == Resize {
		return oswin.WindowResizeEvent
	} else {
		return oswin.WindowEvent
	}
}

func (ev Event) HasPos() bool {
	return false
}

func (ev Event) Pos() image.Point {
	return image.ZP
}

func (ev Event) OnFocus() bool {
	return false
}

// check for interface implementation
var _ oswin.Event = &Event{}
