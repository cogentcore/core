// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
	"image"
)

// The best source on android input events is the NDK: include/android/input.h
//
// iOS event handling guide:
// https://developer.apple.com/library/ios/documentation/EventHandling/Conceptual/EventHandlingiPhoneOS

// Touch is a touch event.
type Touch struct {
	Base

	// Sequence is the sequence number. The same number is shared by all events
	// in a sequence. A sequence begins with a single Begin, is followed by
	// zero or more Moves, and ends with a single End. A Sequence
	// distinguishes concurrent sequences but its value is subsequently reused.
	Sequence Sequence
}

// Sequence identifies a sequence of touch events.
type Sequence int64

// NewTouch creates a new touch event from the given values.
func NewTouch(typ Types, seq Sequence, where image.Point) *Touch {
	ev := &Touch{}
	ev.Typ = typ
	ev.SetUnique()
	ev.Sequence = seq
	ev.Where = where
	return ev
}

func (ev *Touch) HasPos() bool {
	return true
}

func (ev *Touch) String() string {
	return fmt.Sprintf("%v{Pos: %v, Sequence: %v, Time: %v}", ev.Type(), ev.Where, ev.Sequence, ev.Time().Format("04:05"))
}

// todo: what about these higher-level abstractions of touch-like events?

// TouchMagnify is a touch magnification (scaling) gesture event.
// It is the event struct corresponding to events of type [Magnify].
type TouchMagnify struct {
	Touch
	// the multiplicative scale factor relative to the previous
	// zoom of the screen
	ScaleFactor float32
}

// NewMagnify creates a new [TouchMagnify] event based on
// the given multiplicative scale factor.
func NewMagnify(scaleFactor float32, where image.Point) *TouchMagnify {
	ev := &TouchMagnify{}
	ev.Typ = Magnify
	ev.ScaleFactor = scaleFactor
	ev.Where = where
	return ev
}

// // check for interface implementation
// var _ Event = &MagnifyEvent{}

// ////////////////////////////////////////////

// // RotateEvent is used to represent a rotation gesture.
// type RotateEvent struct {
// 	GestureEvent
// 	Rotation float64 // measured in degrees; positive == clockwise
// }

// func (ev *RotateEvent) EventTypes() EventTypes {
// 	return RotateEventTypes
// }

// // check for interface implementation
// var _ Event = &RotateEvent{}

// // Scroll Event is used to represent a scrolling gesture.
// type ScrollEvent struct {
// 	GestureEvent
// 	Delta image.Point
// }

// func (ev *ScrollEvent) EventTypes() EventTypes {
// 	return ScrollEventTypes
// }

// // check for interface implementation
// var _ Event = &ScrollEvent{}
