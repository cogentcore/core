// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/mobile/event:
//
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package window defines an event associated with windows -- including
// changes in the dimensions, physical resolution and orientation of the app's
// window, and iconify, open and close events (see also lifecycle events,
// which pertain to the main app window -- these window events are for all
// windows including dialogs and popups)
package window

import (
	"image"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/kit"
)

// window.Event holds the dimensions, physical resolution, orientation,
// visibility status of a given window, and actions taken on the window
type Event struct {
	oswin.EventBase

	// Size is the window's new dimensions in raw physical pixels.
	Size image.Point

	// DPI is the window's current logical dots-per-inch -- can change if
	// moved to different screen
	LogicalDPI float32

	// ScreenNumber contains index of the current physical or logical screen
	// on which the window is being displayed -- see App.Screen() for
	// the accessing the data associated with this screen
	ScreenNumber int

	// Visibility is the visibility status of the window: Closed, NotVisible,
	// Visible, Iconified
	Visibility Visibilities

	// Action taken on the window -- what has changed
	Action Actions
}

// Bounds returns the window's bounds in raw display dots (pixels), at the
// time this size event was sent.
//
// The top-left pixel is always (0, 0).
func (e Event) Bounds() image.Rectangle {
	return image.Rectangle{Max: e.Size}
}

// Screen returns the screen associated with the screen on which this
// window is displayed
func (e Event) Screen() *oswin.Screen {
	return oswin.TheApp.Screen(e.ScreenNumber)
}

// Visibilities is the visibility status of the window
type Visibilities int32

const (
	// Closed means the window is closed -- has not been opened yet or is now closed
	Closed Visibilities = iota

	// NotVisible means the window is not visible for various reasons (occluded by other windows, off the screen), but not specifically Iconfied
	NotVisible

	// Visible means the window has been opened and is visible
	Visible

	// Iconified means the window has been iconified
	Iconified

	VisibilitiesN
)

//go:generate stringer -type=Visibilities

var KiT_Visibilities = kit.Enums.AddEnum(VisibilitiesN, false, nil)

// Actions is the action taken on the window by the user
type Actions int32

const (
	Open Actions = iota
	Close
	Iconify

	// Resize includes changes in DPI
	Resize

	Maximize
	Minimize

	Move

	// Enter indicates that the user focus / mouse has entered the window
	Enter

	// Leave indicates that the user focus / mouse has left the window
	Leave

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

// todo: above should subsume function of paint event, but can revisit
