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

//go:generate enumgen

import (
	"fmt"
	"image"
	"time"

	"goki.dev/goosi"
)

// DoubleClickInterval is the maximum time interval between button press
// events to count as a double-click.
// This is also in gi.Prefs and updated from there.
var DoubleClickInterval = 500 * time.Millisecond

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

// mouse.Event is a basic mouse event for all mouse events except Scroll
type Event struct {
	goosi.EventBase

	// Button is the mouse button being pressed or released. Its value may be
	// ButtonNone (zero), for a mouse move with no button
	Button Buttons

	// Action taken on the mouse button: Press, Release, DoubleClick, Drag or Move
	Action Actions

	// TODO: have a field to hold what other buttons are down, for detecting
	// drags or button-chords.

	// TODO: add a Device ID, for multiple input devices?
}

func NewEvent(but Buttons, act Actions, where image.Point, mods goosi.Modifiers) *Event {
	ev := &Event{}
	ev.Typ = goosi.MouseEvent
	ev.Button = but
	ev.Action = act
	ev.Where = where
	ev.Mods = mods
	return ev
}

func NewEventCopy(typ goosi.EventTypes, cp *Event) *Event {
	ev := &Event{}
	*ev = *cp
	ev.Typ = typ
	return ev
}

func (ev *Event) String() string {
	return fmt.Sprintf("Type: %v Button: %v Action: %v  Pos: %v  Mods: %v Time: %v", ev.Type(), ev.Button, ev.Action, ev.Where, goosi.ModsString(ev.Mods), ev.Time())
}

func (ev *Event) OnFocus() bool {
	return false
}

func (ev Event) HasPos() bool {
	return true
}

func NewMoveEvent(but Buttons, where, prev image.Point, mods goosi.Modifiers) *Event {
	ev := &Event{}
	ev.Typ = goosi.MouseMoveEvent
	ev.Button = but
	ev.Action = Move
	ev.Where = where
	ev.Prev = prev
	ev.Mods = mods
	return ev
}

func NewDragEvent(but Buttons, where, prev, start image.Point, mods goosi.Modifiers) *Event {
	ev := &Event{}
	ev.Typ = goosi.MouseDragEvent
	ev.Button = but
	ev.Action = Drag
	ev.Where = where
	ev.Prev = prev
	ev.Start = start
	ev.Mods = mods
	return ev
}

func NewScrollEvent(where, delta image.Point, mods goosi.Modifiers) *ScrollEvent {
	ev := &ScrollEvent{}
	ev.Typ = goosi.MouseScrollEvent
	ev.Action = Scroll
	ev.Where = where
	ev.Delta = delta
	ev.Mods = mods
	return ev
}

// Focus event must be generated inside higher-level GUI.  Actions are Enter / Leave
func NewFocusEvent(act Actions, where image.Point, mods goosi.Modifiers) *Event {
	ev := &Event{}
	ev.Typ = goosi.MouseFocusEvent
	ev.Action = act
	ev.Where = where
	ev.Mods = mods
	return ev
}

// Hover event must be generated inside higher-level GUI.
func NewHoverEvent(where image.Point, mods goosi.Modifiers) *Event {
	ev := &Event{}
	ev.Typ = goosi.MouseHoverEvent
	ev.Action = Hover
	ev.Where = where
	ev.Mods = mods
	return ev
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
func (ev ScrollEvent) NonZeroDelta(prefX bool) int {
	if prefX {
		if ev.Delta.X == 0 {
			return ev.Delta.Y
		}
		return ev.Delta.X
	}
	return ev.Delta.Y
}

// SelectModeBits returns the selection mode based on given modifiers bitflags
func SelectModeBits(modBits goosi.Modifiers) SelectModes {
	if goosi.HasAnyModifier(modBits, goosi.Shift) {
		return ExtendContinuous
	}
	if goosi.HasAnyModifier(modBits, goosi.Meta) {
		return ExtendOne
	}
	return SelectOne
}

// SelectMode returns the selection mode based on given modifiers on event
func (ev *Event) SelectMode() SelectModes {
	return SelectModeBits(ev.Mods)
}

// Buttons is a mouse button.
type Buttons int32 //enums:enum

// TODO: have a separate axis concept for wheel up/down? How does that relate
// to joystick events?

const (
	NoButton Buttons = iota
	Left
	Middle
	Right
)

// Actions taken with the mouse button -- different ones are applicable to
// different mouse event types
type Actions int32 //enums:enum

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
)

// SelectModes interprets the modifier keys to determine what type of selection mode to use
// This is also used for selection actions and has modes not directly activated by
// modifier keys
type SelectModes int32 //enums:enum

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
)
