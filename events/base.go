// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
	"image"
	"time"

	"goki.dev/enums"
	"goki.dev/glop/nptime"
	"goki.dev/goosi/events/key"
)

// Base is the base type for events.
// It is designed to support most event types so no further subtypes
// are needed.
type Base struct {
	// Typ is the type of event, returned as Type()
	Typ Types

	// Flags records event boolean state, using atomic flag operations
	Flags EventFlags

	// GenTime records the time when the event was first generated, using more
	// efficient nptime struct
	GenTime nptime.Time

	// Key Modifiers present when event occurred: for Key, Mouse, Touch events
	Mods key.Modifiers

	// Where is the window-based position in raw display dots
	// (pixels) where event took place.
	Where image.Point

	// Start is the window-based starting position in raw display dots
	// (pixels) where event started.
	Start image.Point

	// Prev is the window-based previous position in raw display dots
	// (pixels) -- e.g., for mouse dragging.
	Prev image.Point

	// StTime is the starting time, using more efficient nptime struct
	StTime nptime.Time

	// PrvTime is the time of the previous event, using more efficient nptime struct
	PrvTime nptime.Time

	// LocalOffset is the offset subtracted from original window coordinates
	// to compute the local coordinates.
	LocalOffset image.Point

	// WhereLocal is the local position, which can be adjusted from the window pos
	// via SubLocalOffset based on a local top-left coordinate for a region within
	// the window.
	WhereLocal image.Point

	// StartLocal is the local starting position
	StartLocal image.Point

	// PrevLocal is the local previous position
	PrevLocal image.Point

	// Button is the mouse button being pressed or released, for relevant events.
	Button Buttons

	// Rune is the meaning of the key event as determined by the
	// operating system. The mapping is determined by system-dependent
	// current layout, modifiers, lock-states, etc.
	Rune rune

	// Code is the identity of the physical key relative to a notional
	// "standard" keyboard, independent of current layout, modifiers,
	// lock-states, etc
	Code key.Codes

	// todo: add El info
	Data any
}

// SetTime sets the event time to Now
func (ev *Base) SetTime() {
	ev.GenTime.Now()
}

func (ev *Base) Init() {
	ev.SetTime()
	ev.SetLocalOff(image.Point{}) // ensure local is copied
}

func (ev Base) Type() Types {
	return ev.Typ
}

func (ev *Base) AsBase() *Base {
	return ev
}

func (ev Base) IsSame(oth Event) bool {
	return ev.Typ == oth.Type() // basic check.  redefine in subtypes
}

func (ev Base) IsUnique() bool {
	return ev.Flags.HasFlag(Unique)
}

func (ev Base) SetUnique() {
	ev.Flags.SetFlag(true, Unique)
}

func (ev Base) Time() time.Time {
	return ev.GenTime.Time()
}

func (ev Base) StartTime() time.Time {
	return ev.StTime.Time()
}

func (ev Base) PrevTime() time.Time {
	return ev.PrvTime.Time()
}

func (ev Base) IsHandled() bool {
	return ev.Flags.HasFlag(Handled)
}

func (ev *Base) SetHandled() {
	ev.Flags.SetFlag(true, Handled)
}

func (ev *Base) ClearHandled() {
	ev.Flags.SetFlag(false, Handled)
}

func (ev Base) String() string {
	return fmt.Sprintf("%v{Time: %v}", ev.Typ, ev.Time())
}

func (ev Base) OnWinFocus() bool {
	return true
}

// SetModifiers sets the bitflags based on a list of key.Modifiers
func (ev *Base) SetModifiers(mods ...enums.BitFlag) {
	ev.Mods.SetFlag(true, mods...)
}

// HasAllModifiers tests whether all of given modifier(s) were set
func (ev Base) HasAllModifiers(mods ...enums.BitFlag) bool {
	return key.HasAnyModifier(ev.Mods, mods...)
}

func (ev Base) HasAnyModifier(mods ...enums.BitFlag) bool {
	return key.HasAnyModifier(ev.Mods, mods...)
}

func (ev Base) HasPos() bool {
	return false
}

func (ev Base) Pos() image.Point {
	return ev.Where
}

func (ev Base) StartPos() image.Point {
	return ev.Start
}

func (ev Base) PrevPos() image.Point {
	return ev.Prev
}

// Delta returns the amount of mouse movement (Where - Prev)
func (ev Base) Delta() image.Point {
	return ev.Where.Sub(ev.Prev)
}

func (ev *Base) SetLocalOff(off image.Point) {
	ev.LocalOffset = off
	ev.WhereLocal = ev.Where.Sub(off)
	ev.StartLocal = ev.Start.Sub(off)
	ev.PrevLocal = ev.Prev.Sub(off)
}

func (ev Base) LocalOff() image.Point {
	return ev.LocalOffset
}

func (ev Base) LocalPos() image.Point {
	return ev.WhereLocal
}

func (ev Base) LocalStartPos() image.Point {
	return ev.StartLocal
}

func (ev Base) LocalPrevPos() image.Point {
	return ev.PrevLocal
}

// SelectMode returns the selection mode based on given modifiers on event
func (ev Base) SelectMode() SelectModes {
	return SelectModeBits(ev.Mods)
}

// MouseButton is the mouse button being pressed or released, for relevant events.
func (ev Base) MouseButton() Buttons {
	return ev.Button
}

// Modifiers returns the modifier keys present at time of event
func (ev Base) Modifiers() key.Modifiers {
	return ev.Mods
}

func (ev Base) KeyRune() rune {
	return ev.Rune
}

func (ev Base) KeyCode() key.Codes {
	return ev.Code
}

// KeyChord returns a string representation of the keyboard event suitable for
// keyboard function maps, etc. Printable runes are sent directly, and
// non-printable ones are converted to their corresponding code names without
// the "Code" prefix.
func (ev Base) KeyChord() key.Chord {
	return key.NewChord(ev.Rune, ev.Code, ev.Mods)
}

func (ev Base) Clone() Event {
	nb := &Base{}
	*nb = ev
	nb.Flags.SetFlag(false, Handled)
	return nb
}
