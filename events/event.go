// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
	"image"
	"time"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/events/key"
)

// Cogent Core event structure was originally derived from go.wde and golang/x/mobile/event
// but has diverged significantly from there.  Also informed by JavaScript event type.
//
// Cogent Core requires event type enum for widgets to request what events to
// receive, and we add an overall interface with base support for time and
// marking events as processed, which is critical for simplifying logic and
// preventing unintended multiple effects
//
// System deals exclusively in raw "dot" pixel integer coordinates (as in
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

// Event is the interface for system GUI events.  also includes Stringer
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

	// NeedsFocus this event goes to current focus widget
	NeedsFocus() bool

	// HasPos returns true if the event has a window position where it takes place
	HasPos() bool

	// WindowPos returns the original window-based position in raw display dots
	// (pixels) where event took place.
	WindowPos() image.Point

	// SetLocalOff sets the offset subtracted from window-based positions
	// to compute Local versions of positions, which are updated.
	SetLocalOff(off image.Point)

	// LocalOff returns the offset subtracted from window-based positions
	// to compute Local versions of positions.
	LocalOff() image.Point

	// Pos returns the local position, which is adjusted from the WindowPos
	// via SubLocalOffset based on a local top-left coordinate for a region
	// within the window.
	Pos() image.Point

	// WindowStartPos returns the starting (MouseDown) window-based position.
	WindowStartPos() image.Point

	// StartPos returns the starting (MouseDown) local position
	StartPos() image.Point

	// StartDelta returns Pos - Start
	StartDelta() image.Point

	// WindowPrevPos returns the previous (MouseMove/Drag) window-based position.
	WindowPrevPos() image.Point

	// PrevPos returns the previous (MouseMove/Drag) local position
	PrevPos() image.Point

	// PrevDelta returns Pos - Prev
	PrevDelta() image.Point

	// Time returns the time at which the event was generated, in UnixNano nanosecond units
	Time() time.Time

	// StartTime returns time of StartPos (MouseDown),
	// or other starting time of relevance to the event.
	StartTime() time.Time

	// SinceStart returns Time().Sub(StartTime()) -- duration since Start
	SinceStart() time.Duration

	// PrevTime returns time of PrevPos (MouseMove),
	// or other earlier time of relevance to the event.
	PrevTime() time.Time

	// SincePrev returns Time().Sub(PrevTime()) -- duration since Prev
	SincePrev() time.Duration

	// IsHandled returns whether this event has already been processed
	// Event handling checks this and terminates processing if
	// SetHandled has been called.
	IsHandled() bool

	// SetHandled marks the event as having been processed,
	// so no further processing occurs.  This can accomplish
	// the same effect as PreventDefault function in JavaScript.
	SetHandled()

	// ClearHandled marks the event as no longer having been processed,
	// meaning that it will be processed by future event handlers.
	// This reverses the effects of [Event.SetHandled].
	ClearHandled()

	// Init sets the time to now, and any other initialization.
	// Done just prior to event Send.
	Init()

	// Clone returns a duplicate of this event with the basic event parameters
	// copied (specialized Event types have their own CloneX methods)
	// and the Handled flag is reset.  This is suitable for repurposing.
	Clone() Event

	// NewFromClone returns a duplicate of this event with the basic event parameters
	// copied, and type set to given type.  The resulting type is ready for sending.
	NewFromClone(typ Types) Event

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

	// HasAllModifiers tests whether all of given modifier(s) were set
	HasAllModifiers(mods ...enums.BitFlag) bool

	// HasAnyModifier testes whether any of the given modifiers were set
	HasAnyModifier(mods ...enums.BitFlag) bool

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
