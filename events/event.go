// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
	"image"
	"time"

	"goki.dev/goosi/events/key"
)

// GoGi event structure was originally derived from go.wde and golang/x/mobile/event
// but has diverged significantly from there.  Also informed by JavaScript event type.
//
// GoGi requires event type enum for widgets to request what events to
// receive, and we add an overall interface with base support for time and
// marking events as processed, which is critical for simplifying logic and
// preventing unintended multiple effects
//
// Goosi deals exclusively in raw "dot" pixel integer coordinates (as in
// go.wde) -- abstraction to different DPI etc takes place higher up in the
// system

// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
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

// Event is the interface for goosi GUI events.  also includes Stringer
// to get a string description of the event
type Event interface {
	fmt.Stringer

	// Type returns the type of event associated with given event
	Type() Types

	// AsBase returns this event as a Base event type,
	// which is used for most Event types.
	AsBase() *Base

	// IsUnique returns true if this event must always be sent,
	// even if the last event sent is of the same type.
	// This is true for Key, MouseButton,
	// Otherwise, events of the same type are compressed
	// such that if the last one written to the queue is of
	// the same type, it is replaced instead of adding a new one.
	IsUnique() bool

	// HasPos returns true if the event has a window position where it takes place
	HasPos() bool

	// Pos returns the original window-based position in raw display dots
	// (pixels) where event took place.
	Pos() image.Point

	// SetLocalOff sets the offset subtracted from window-based positions
	// to compute Local versions of positions, which are updated.
	SetLocalOff(off image.Point)

	// LocalOff returns the offset subtracted from window-based positions
	// to compute Local versions of positions.
	LocalOff() image.Point

	// LocalPos returns the local position, which can be adjusted from the window pos
	// via SubLocalOffset based on a local top-left coordinate for a region within
	// the window.
	LocalPos() image.Point

	// StartPos returns the original starting window-based position.
	StartPos() image.Point

	// LocalStartPos returns the local starting position
	LocalStartPos() image.Point

	// PrevPos returns the original previous window-based position.
	PrevPos() image.Point

	// LocalPrevPos returns the local previous position
	LocalPrevPos() image.Point

	// Time returns the time at which the event was generated, in UnixNano nanosecond units
	Time() time.Time

	// StartTime returns time of StartPos, or other starting time of relevance to the event,
	// in UnixNano nanosecond units.
	StartTime() time.Time

	// PrevTime returns time of PrevPos, or other earlier time of relevance to the event,
	// in UnixNano nanosecond units.
	PrevTime() time.Time

	// IsHandled returns whether this event has already been processed
	IsHandled() bool

	// SetHandled marks the event as having been processed
	SetHandled()

	// Init sets the time to now, and any other init -- done just prior to event delivery
	Init()

	// Clone returns a duplicate of this event with the basic event parameters
	// copied (specialized Event types have their own CloneX methods)
	// and the Handled flag is reset.  This is suitable for repurposing.
	Clone() Event

	// SetTime sets the event time to Now
	SetTime()

	// IsSame returns true if the current event is the same as other.
	// Checks Type and, where relevant, Action.
	IsSame(oth Event) bool

	// MouseButton is the mouse button being pressed or released, for relevant events.
	MouseButton() Buttons

	// SelectMode returns the selection mode based on given modifiers on event
	SelectMode() SelectModes

	// Modifiers returns the modifier keys present at time of event
	Modifiers() key.Modifiers

	// Rune is the meaning of the key event as determined by the
	// operating system. The mapping is determined by system-dependent
	// current layout, modifiers, lock-states, etc.
	KeyRune() rune

	// Code is the identity of the physical key relative to a notional
	// "standard" keyboard, independent of current layout, modifiers,
	// lock-states, etc
	//
	// For standard key codes, its value matches USB HID key codes.
	// Compare its value to uint32-typed constants in this package, such
	// as CodeLeftShift and CodeEscape.
	//
	// Pressing the regular '2' key and number-pad '2' key (with Num-Lock)
	// generate different Codes (but the same Rune).
	KeyCode() key.Codes

	// KeyChord returns a string representation of the keyboard event suitable for
	// keyboard function maps, etc. Printable runes are sent directly, and
	// non-printable ones are converted to their corresponding code names without
	// the "Code" prefix.
	KeyChord() key.Chord
}
