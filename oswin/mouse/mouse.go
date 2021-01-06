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
	"fmt"
	"image"
	"time"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/ki/kit"
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

// ScrollWheelSpeed controls how fast the scroll wheel moves (typically
// interpreted as pixels per wheel step) -- only relevant for some OS's which
// do not have a native preference for this setting, e.g., X11
// This is also in gi.Prefs and updated from there
var ScrollWheelSpeed = float32(20)

// mouse.Event is a basic mouse event for button presses, but not motion or scrolling
type Event struct {
	oswin.EventBase

	// Where is the mouse location, in raw display dots (raw, actual pixels)
	Where image.Point

	// Button is the mouse button being pressed or released. Its value may be
	// ButtonNone (zero), for a mouse move with no button
	Button Buttons

	// Action taken on the mouse button: Press, Release, DoubleClick, Drag or Move
	Action Actions

	// TODO: have a field to hold what other buttons are down, for detecting
	// drags or button-chords.

	// Modifiers is a bitmask representing a set of modifier keys:
	// key.ModShift, key.ModAlt, etc. -- bit positions are key.Modifiers
	Modifiers int32

	// TODO: add a Device ID, for multiple input devices?
}

// SetModifiers sets the bitflags based on a list of key.Modifiers
func (e *Event) SetModifiers(mods ...key.Modifiers) {
	key.SetModifierBits(&e.Modifiers, mods...)
}

// HasAnyModifier tests whether any of given modifier(s) were set
func (e *Event) HasAnyModifier(mods ...key.Modifiers) bool {
	return key.HasAnyModifierBits(e.Modifiers, mods...)
}

// HasAllModifiers tests whether all of given modifier(s) were set
func (e *Event) HasAllModifier(mods ...key.Modifiers) bool {
	return key.HasAllModifierBits(e.Modifiers, mods...)
}

// SelectModeBits returns the selection mode based on given modifiers bitflags
func SelectModeBits(modBits int32) SelectModes {
	if key.HasAnyModifierBits(modBits, key.Shift) {
		return ExtendContinuous
	}
	if key.HasAnyModifierBits(modBits, key.Meta) {
		return ExtendOne
	}
	return SelectOne
}

// SelectMode returns the selection mode based on given modifiers on event
func (e *Event) SelectMode() SelectModes {
	return SelectModeBits(e.Modifiers)
}

func (ev *Event) String() string {
	return fmt.Sprintf("Type: %v Button: %v Action: %v  Pos: %v  Mods: %v Time: %v", ev.Type(), ev.Button, ev.Action, ev.Where, key.ModsString(ev.Modifiers), ev.Time())
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

// mouse.HoverEvent is generated by gi.Window based on lack of motion after
// entering a widget
type HoverEvent struct {
	Event
}

// Buttons is a mouse button.
type Buttons int32

// TODO: have a separate axis concept for wheel up/down? How does that relate
// to joystick events?

const (
	NoButton Buttons = iota
	Left
	Middle
	Right

	ButtonsN
)

//go:generate stringer -type=Buttons

var KiT_Buttons = kit.Enums.AddEnum(ButtonsN, kit.NotBitFlag, nil)

// Actions taken with the mouse button -- different ones are applicable to
// different mouse event types
type Actions int32

const (
	NoAction Actions = iota
	Press
	Release
	DoubleClick
	Move
	Drag
	Scroll
	Enter
	Exit
	Hover

	ActionsN
)

//go:generate stringer -type=Actions

var KiT_Actions = kit.Enums.AddEnum(ActionsN, kit.NotBitFlag, nil)

// SelectModes interprets the modifier keys to determine what type of selection mode to use
// This is also used for selection actions and has modes not directly activated by
// modifier keys
type SelectModes int32

const (
	// SelectOne selects a single item, and is the default when no modifier key
	// is pressed
	SelectOne SelectModes = iota

	// ExtendContinuous, activated by Shift key, extends the selection to
	// select a continuous region of selected items, with no gaps
	ExtendContinuous

	// ExtendOne, activated by Control or Meta / Command, extends the
	// selection by adding the one additional item just clicked on, creating a
	// potentially discontinuous set of selected items
	ExtendOne

	// NoSelect means do not update selection -- this is used programmatically
	// and not available via modifier key
	NoSelect

	// Unselect means unselect items -- this is used programmatically
	// and not available via modifier key -- typically ExtendOne will
	// unselect if already selected
	Unselect

	// SelectQuiet means select without doing other updates or signals -- for
	// bulk updates with a final update at the end -- used programmatically
	SelectQuiet

	// UnselectQuiet means unselect without doing other updates or signals -- for
	// bulk updates with a final update at the end -- used programmatically
	UnselectQuiet

	SelectModesN
)

//go:generate stringer -type=SelectModes

var KiT_SelectModes = kit.Enums.AddEnum(SelectModesN, kit.NotBitFlag, nil)

/////////////////////////////
// oswin.Event interface

func (ev *Event) Type() oswin.EventType {
	return oswin.MouseEvent
}

func (ev *Event) HasPos() bool {
	return true
}

func (ev *Event) Pos() image.Point {
	return ev.Where
}

func (ev *Event) OnFocus() bool {
	return false
}

func (ev *MoveEvent) Type() oswin.EventType {
	return oswin.MouseMoveEvent
}

func (ev *DragEvent) Type() oswin.EventType {
	return oswin.MouseDragEvent
}

func (ev *ScrollEvent) Type() oswin.EventType {
	return oswin.MouseScrollEvent
}

func (ev *FocusEvent) Type() oswin.EventType {
	return oswin.MouseFocusEvent
}

func (ev *HoverEvent) Type() oswin.EventType {
	return oswin.MouseHoverEvent
}
