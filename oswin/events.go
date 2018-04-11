// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oswin

import (
	"image"
	// "fmt"
	"time"
)

/*
   much of this is copied directly from https://github.com/skelterjohn/go.wde

   Copyright 2012 the go.wde authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// EventType determines which type of GUI event is being sent -- need this for indexing into
// different event signalers based on event type, and sending event type in signals
type EventType int64

const (
	// mouse events
	MouseMovedEventType EventType = iota
	MouseDownEventType
	MouseUpEventType
	MouseDraggedEventType

	// gesture events
	MagnifyEventType
	RotateEventType
	ScrollEventType

	// key events
	KeyDownEventType
	KeyUpEventType
	KeyTypedEventType

	// window events -- todo: need iconfiy, etc events
	MouseEnteredEventType
	MouseExitedEventType
	ResizeEventType
	CloseEventType

	// number of event types
	EventTypeN
)

//go:generate stringer -type=EventType

// buttons that can be pressed
type MouseButton int

const (
	LeftButton MouseButton = 1 << iota
	MiddleButton
	RightButton
	WheelUpButton
	WheelDownButton
	WheelLeftButton  // only supported by xgb/win backends atm
	WheelRightButton // only supported by xgb/win backends atm
)

// interface for GUI events
type Event interface {
	// get the type of event associated with given event
	EventType() EventType
	// does the event have window position where it takes place?
	EventHasPos() bool
	// position where event took place -- needed for sending events to the right place
	EventPos() image.Point
	// does the event operate only on focus item (e.g., keyboard events)
	EventOnFocus() bool
	// time at which the event was generated
	EventTime() time.Time
	// has this event already been processed?
	IsProcessed() bool
	// mark as having been processed
	SetProcessed()
}

// base type for events -- records time and whether event has been processed by a receiver of the event -- in which case it is skipped
type EventBase struct {
	Processed bool
	Time      time.Time
}

func (ev EventBase) EventTime() time.Time {
	return ev.Time
}

func (ev EventBase) IsProcessed() bool {
	return ev.Processed
}

func (ev *EventBase) SetProcessed() {
	ev.Processed = true
}

////////////////////////////////////////////////////////////////////////////////////////
//   Mouse Events

// MouseEvent is used for data common to all mouse events, and should not appear as an event received by the caller program.
type MouseEvent struct {
	EventBase
	Where image.Point
}

func (ev MouseEvent) EventHasPos() bool {
	return true
}

func (ev MouseEvent) EventPos() image.Point {
	return ev.Where
}

func (ev MouseEvent) EventOnFocus() bool {
	return false
}

////////////////////////////////////////////

// MouseMovedEvent is for when the mouse moves within the window.
type MouseMovedEvent struct {
	MouseEvent
	From image.Point
}

func (ev MouseMovedEvent) EventType() EventType {
	return MouseMovedEventType
}

// check for interface implementation
var _ Event = &MouseMovedEvent{}

////////////////////////////////////////////

// MouseButtonEvent is used for data common to all mouse button events, and should not appear as an event received by the caller program.
type MouseButtonEvent struct {
	MouseEvent
	Which MouseButton
}

// MouseDownEvent is for when the mouse is clicked within the window.
type MouseDownEvent MouseButtonEvent

func (ev MouseDownEvent) EventType() EventType {
	return MouseDownEventType
}

// check for interface implementation
var _ Event = &MouseDownEvent{}

// MouseUpEvent is for when the mouse is unclicked within the window.
type MouseUpEvent MouseButtonEvent

func (ev MouseUpEvent) EventType() EventType {
	return MouseUpEventType
}

// check for interface implementation
var _ Event = &MouseUpEvent{}

////////////////////////////////////////////

// MouseDraggedEvent is for when the mouse is moved while a button is pressed.
type MouseDraggedEvent struct {
	MouseMovedEvent
	Which MouseButton
}

func (ev MouseDraggedEvent) EventType() EventType {
	return MouseDraggedEventType
}

// check for interface implementation
var _ Event = &MouseDraggedEvent{}

////////////////////////////////////////////////////////////////////////////////////////
//   Gesture Events

// GestureEvent is used to represents common elements of all gesture-based events
type GestureEvent struct {
	MouseEvent
}

////////////////////////////////////////////

// MagnifyEvent is used to represent a magnification gesture.
type MagnifyEvent struct {
	GestureEvent
	Magnification float64 // the multiplicative scale factor
}

func (ev MagnifyEvent) EventType() EventType {
	return MagnifyEventType
}

// check for interface implementation
var _ Event = &MagnifyEvent{}

////////////////////////////////////////////

// RotateEvent is used to represent a rotation gesture.
type RotateEvent struct {
	GestureEvent
	Rotation float64 // measured in degrees; positive == clockwise
}

func (ev RotateEvent) EventType() EventType {
	return RotateEventType
}

// check for interface implementation
var _ Event = &RotateEvent{}

////////////////////////////////////////////

// Scroll Event is used to represent a scrolling gesture.
type ScrollEvent struct {
	GestureEvent
	Delta image.Point
}

func (ev ScrollEvent) EventType() EventType {
	return ScrollEventType
}

// check for interface implementation
var _ Event = &ScrollEvent{}

////////////////////////////////////////////////////////////////////////////////////////
//   Key Events

// KeyEvent is used for data common to all key events, and should not appear as an event received by the caller program.
type KeyEvent struct {
	EventBase
	Key string
}

func (ev KeyEvent) EventHasPos() bool {
	return false
}

func (ev KeyEvent) EventPos() image.Point {
	return image.ZP
}

func (ev KeyEvent) EventOnFocus() bool {
	return true
}

////////////////////////////////////////////

// KeyDownEvent is for when a key is pressed.
type KeyDownEvent struct {
	KeyEvent
}

func (ev KeyDownEvent) EventType() EventType {
	return KeyDownEventType
}

// check for interface implementation
var _ Event = &KeyDownEvent{}

// KeyUpEvent is for when a key is unpressed.
type KeyUpEvent struct {
	KeyEvent
}

func (ev KeyUpEvent) EventType() EventType {
	return KeyUpEventType
}

// check for interface implementation
var _ Event = &KeyUpEvent{}

// KeyTypedEvent is for when a key is typed.
type KeyTypedEvent struct {
	KeyEvent
	// The glyph is the string corresponding to what the user wants to have typed in
	// whatever data entry is active.
	Glyph string
	// The "+" joined set of keys (not glyphs) participating in the chord completed
	// by this key event. The keys will be in a consistent order, no matter what
	// order they are pressed in.
	Chord string
}

func (ev KeyTypedEvent) EventType() EventType {
	return KeyTypedEventType
}

// check for interface implementation
var _ Event = &KeyTypedEvent{}

////////////////////////////////////////////////////////////////////////////////////////
//   Window Events

////////////////////////////////////////////

// MouseEnteredEvent is for when the mouse enters a window, or a widget (computed by window)
type MouseEnteredEvent struct {
	MouseMovedEvent
}

func (ev MouseEnteredEvent) EventType() EventType {
	return MouseEnteredEventType
}

// check for interface implementation
var _ Event = &MouseEnteredEvent{}

// MouseExitedEvent is for when the mouse exits a window, or a widget (computed by window)a
type MouseExitedEvent struct {
	MouseMovedEvent
}

func (ev MouseExitedEvent) EventType() EventType {
	return MouseExitedEventType
}

// check for interface implementation
var _ Event = &MouseExitedEvent{}

// ResizeEvent is for when the window changes size.
type ResizeEvent struct {
	EventBase
	Width, Height int
}

func (ev ResizeEvent) EventType() EventType {
	return ResizeEventType
}

func (ev ResizeEvent) EventHasPos() bool {
	return false
}

func (ev ResizeEvent) EventPos() image.Point {
	return image.ZP
}

func (ev ResizeEvent) EventOnFocus() bool {
	return false
}

// check for interface implementation
var _ Event = &ResizeEvent{}

// CloseEvent is for when the window is closed.
type CloseEvent struct {
	EventBase
}

func (ev CloseEvent) EventType() EventType {
	return CloseEventType
}

func (ev CloseEvent) EventHasPos() bool {
	return false
}

func (ev CloseEvent) EventPos() image.Point {
	return image.ZP
}

func (ev CloseEvent) EventOnFocus() bool {
	return false
}

// check for interface implementation
var _ Event = &CloseEvent{}
