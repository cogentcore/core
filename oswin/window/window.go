// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
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

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/ki/kit"
)

// window.Event holds the dimensions, physical resolution, orientation,
// visibility status of a given window, and actions taken on the window
type Event struct {
	oswin.EventBase

	// Size is the window's dimensions in raw physical pixels.
	Size image.Point

	// ScreenNumber contains index of the current physical or logical screen
	// on which the window is being displayed -- see Screen.ScreenData() for
	// the accessing the data associated with this screen
	ScreenNumber int

	// Visibility is the visibility status of the window: Closed, NotVisible, Visible, Iconified
	Visibility Visibility

	// Action taken on the window -- what has changed
	Action Action
}

// Bounds returns the window's bounds in raw display dots (pixels), at the
// time this size event was sent.
//
// The top-left pixel is always (0, 0).
func (e Event) Bounds() image.Rectangle {
	return image.Rectangle{Max: e.Size}
}

// ScreenData returns the screen data associated with the screen on which this
// window is displayed
func (e Event) ScreenData() *oswin.ScreenData {
	return oswin.CurScreen.ScreenData(e.ScreenNumber)
}

// Visibility is the visibility status of the window
type Visibility int32

const (
	// Closed means the window is closed -- has not been opened yet or is now closed
	Closed Visibility = iota

	// NotVisible means the window is not visible for various reasons (occluded by other windows, off the screen), but not specifically Iconfied
	NotVisible

	// Visible means the window has been opened and is visible
	Visible

	// Iconified means the window has been iconified
	Iconified

	VisibilityN
)

//go:generate stringer -type=Visibility

var KiT_Visibility = kit.Enums.AddEnum(VisibilityN, false, nil)

// Action is the action taken on the window by the user
type Action int32

const (
	Open Action = iota
	Close
	Iconify
	Resize
	Maximize
	Minimize
	// Move of window across physical screens can trigger changes in DPI..
	Move
	// Enter indicates that the user focus / mouse has entered the window
	Enter
	// Leave indicates that the user focus / mouse has left the window
	Leave

	ActionN
)

//go:generate stringer -type=Action

var KiT_Action = kit.Enums.AddEnum(ActionN, false, nil)

/////////////////////////////
// oswin.Event interface

func (ev Event) Type() oswin.EventType {
	return oswin.WindowEvent
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
