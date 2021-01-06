// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/mobile/event:
//
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package touch defines an event for touch input, for the GoGi GUI system.
package touch

// The best source on android input events is the NDK: include/android/input.h
//
// iOS event handling guide:
// https://developer.apple.com/library/ios/documentation/EventHandling/Conceptual/EventHandlingiPhoneOS

import (
	"image"

	"github.com/goki/gi/oswin"
	"github.com/goki/ki/kit"
)

// touch.Event is a touch event.
type Event struct {
	oswin.EventBase

	// Where is the touch location, in raw display dots (raw, actual pixels)
	Where image.Point

	// Sequence is the sequence number. The same number is shared by all events
	// in a sequence. A sequence begins with a single Begin, is followed by
	// zero or more Moves, and ends with a single End. A Sequence
	// distinguishes concurrent sequences but its value is subsequently reused.
	Sequence Sequence

	// Action is the touch action
	Action Actions
}

// Sequence identifies a sequence of touch events.
type Sequence int64

// Actions describes the action taken for a touch event.
type Actions int32

const (
	// Begin is a user first touching the device.
	//
	// On Android, this is a AMOTION_EVENT_ACTION_DOWN.
	// On iOS, this is a call to touchesBegan.
	Begin Actions = iota

	// Move is a user dragging across the device.
	//
	// A TypeMove is delivered between a TypeBegin and TypeEnd.
	//
	// On Android, this is a AMOTION_EVENT_ACTION_MOVE.
	// On iOS, this is a call to touchesMoved.
	Move

	// End is a user no longer touching the device.
	//
	// On Android, this is a AMOTION_EVENT_ACTION_UP.
	// On iOS, this is a call to touchesEnded.
	End

	ActionsN
)

//go:generate stringer -type=Actions

var KiT_Actions = kit.Enums.AddEnum(ActionsN, kit.NotBitFlag, nil)

/////////////////////////////
// oswin.Event interface

func (ev *Event) Type() oswin.EventType {
	return oswin.TouchEvent
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

// check for interface implementation
var _ oswin.Event = &Event{}

// todo: what about these higher-level abstractions of touch-like events?

// // MagnifyEvent is used to represent a magnification gesture.
// type MagnifyEvent struct {
// 	GestureEvent
// 	Magnification float64 // the multiplicative scale factor
// }

// func (ev *MagnifyEvent) EventType() EventType {
// 	return MagnifyEventType
// }

// // check for interface implementation
// var _ Event = &MagnifyEvent{}

// ////////////////////////////////////////////

// // RotateEvent is used to represent a rotation gesture.
// type RotateEvent struct {
// 	GestureEvent
// 	Rotation float64 // measured in degrees; positive == clockwise
// }

// func (ev *RotateEvent) EventType() EventType {
// 	return RotateEventType
// }

// // check for interface implementation
// var _ Event = &RotateEvent{}

// // Scroll Event is used to represent a scrolling gesture.
// type ScrollEvent struct {
// 	GestureEvent
// 	Delta image.Point
// }

// func (ev *ScrollEvent) EventType() EventType {
// 	return ScrollEventType
// }

// // check for interface implementation
// var _ Event = &ScrollEvent{}
