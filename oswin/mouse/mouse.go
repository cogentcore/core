// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/mobile/event:
//
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mouse defines mouse events, for the GoGi GUI system.  The most
// important distinction for mouse events is moving versus static -- many GI
// elements don't need to know about motion, and they are high-volume events,
// so it is important to split those out.
package mouse

import (
	"image"
	"time"

	"github.com/goki/goki/gi/oswin"
	"github.com/goki/goki/gi/oswin/key"
	"github.com/goki/goki/ki/kit"
)

// DoubleClickMSec is the maximum time interval in msec between button press
// events to count as a double-click -- can set this global variable to change.
// This is also in gi.Prefs and updated from there
var DoubleClickMSec = 500

// TODO: implement DoubleClickWait

// DoubleClickWait causes the event system to wait for a possible double-click
// event before sending single clicks.  This causes a delay, but avoids many
// sources of potential difficulty in dealing with double-clicking, as
// described here:
// https://blogs.msdn.microsoft.com/oldnewthing/20041015-00/?p=37553
var DoubleClickWait = false

// ScrollWheelRate controls how fast the scroll wheel moves (typically
// interpreted as pixels per wheel step) -- only relevant for some OS's which
// do not have a native preference for this setting, e.g., X11
var ScrollWheelRate = 2

// mouse.Event is a basic mouse event for button presses, but not motion or scrolling
type Event struct {
	oswin.EventBase

	// Where is the mouse location, in raw display dots (raw, actual pixels)
	Where image.Point

	// Button is the mouse button being pressed or released. Its value may be
	// ButtonNone (zero), for a mouse move with no button
	Button Button

	// Action taken on the mouse button: Press, Release, DoubleClick, Drag or Move
	Action Action

	// TODO: have a field to hold what other buttons are down, for detecting
	// drags or button-chords.

	// Modifiers is a bitmask representing a set of modifier keys:
	// key.ModShift, key.ModAlt, etc. -- bit positions are key.Modifiers
	Modifiers int32

	// TODO: add a Device ID, for multiple input devices?
}

// SetModifiers sets the bitflags based on a list of key.Modifiers
func (e *Event) SetModifiers(mods ...key.Modifiers) {
	for _, m := range mods {
		e.Modifiers |= 1 << uint32(m)
	}
}

/////////////////////////////////////////////////////////////////

// mouse.MoveEvent is for mouse movement, without button down -- action is Move
type MoveEvent struct {
	Event

	// From is the previous location of the mouse
	From image.Point

	// LastTime is the time of the previous event
	LastTime time.Time
}

/////////////////////////////////////////////////////////////////

// mouse.DragEvent is for mouse movement, with button down -- action is Drag
// -- many receivers will be interested in Drag events but not Move events,
// which is why these are separate
type DragEvent struct {
	MoveEvent
}

// Delta returns the amount of mouse movement (Where - From)
func (e MoveEvent) Delta() image.Point {
	return e.Where.Sub(e.From)
}

/////////////////////////////////////////////////////////////////

// mouse.ScrollEvent is for mouse scrolling, recording the delta of the scroll
type ScrollEvent struct {
	Event

	// Delta is the amount of scrolling in each axis
	Delta image.Point
}

// NonZeroDelta attempts to find a non-zero delta -- often only get Y
// dimension scrolling and want to use that for X if prefX is true
func (e ScrollEvent) NonZeroDelta(prefX bool) int {
	if prefX {
		if e.Delta.X == 0 {
			return e.Delta.Y
		}
		return e.Delta.X
	}
	return e.Delta.Y
}

/////////////////////////////////////////////////////////////////

// mouse.FocusEvent records actions of Enter and Exit of mouse into a given
// widget bounding box -- generated from mouse.MoveEvents in gi.Window, which
// knows about widget bounding boxes
type FocusEvent struct {
	Event
}

// Button is a mouse button.
type Button int32

// TODO: have a separate axis concept for wheel up/down? How does that relate
// to joystick events?

const (
	NoButton Button = iota
	Left
	Middle
	Right

	ButtonN
)

//go:generate stringer -type=Button

var KiT_Button = kit.Enums.AddEnum(ButtonN, false, nil)

// Action taken with the mouse button -- different ones are applicable to
// different mouse event types
type Action int32

const (
	NoAction Action = iota
	Press
	Release
	DoubleClick
	Move
	Drag
	Scroll
	Enter
	Exit

	ActionN
)

//go:generate stringer -type=Action

var KiT_Action = kit.Enums.AddEnum(ActionN, false, nil)

/////////////////////////////
// oswin.Event interface

func (ev Event) Type() oswin.EventType {
	return oswin.MouseEvent
}

func (ev Event) HasPos() bool {
	return true
}

func (ev Event) Pos() image.Point {
	return ev.Where
}

func (ev Event) OnFocus() bool {
	return false
}

// check for interface implementation
var _ oswin.Event = &Event{}

func (ev MoveEvent) Type() oswin.EventType {
	return oswin.MouseMoveEvent
}

func (ev DragEvent) Type() oswin.EventType {
	return oswin.MouseDragEvent
}

func (ev ScrollEvent) Type() oswin.EventType {
	return oswin.MouseScrollEvent
}

func (ev FocusEvent) Type() oswin.EventType {
	return oswin.MouseFocusEvent
}
