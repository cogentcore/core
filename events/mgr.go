// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"image"
	"time"

	"goki.dev/glop/nptime"
	"goki.dev/goosi/events/key"
)

// Mgr manages the event construction and sending process,
// for its parent window.  Caches state as needed
// to generate derived events such as MouseDrag.
type Mgr struct {
	Deque Dequer // must be set to parent window where events are sent

	// flag for ignoring mouse events when disabling mouse movement
	ResettingPos bool

	// MouseInStack is the stack of current elements under the mouse.
	// This is set when GUI generates a MouseEnter event based on
	// MouseMove or MouseDrag event sent to it.
	// MouseInStack ElStack

	// DragEl is the current element being dragged
	DragEl El

	// SlidingEl is the current slider element being manipulated with MouseDrag
	SlidingEl El

	// FocusEl is the current focus element
	FocusEl El

	// Last has the prior state for key variables
	Last MgrState
}

// MgrState tracks basic event state over time
// to enable recognition and full data for generating events.
type MgrState struct {
	// last mouse button event type (down or up)
	MouseButtonType Types

	// last mouse button
	MouseButton Buttons

	// time of MouseDown
	MouseDownTime nptime.Time

	// position at MouseDown
	MouseDownPos image.Point

	// button at MouseDown
	MouseDownButton Buttons

	// time of Click event -- for double-click
	ClickTime nptime.Time

	// position of mouse from move events
	MousePos image.Point

	// time of last move
	MouseMoveTime nptime.Time

	// keyboard modifiers (Shift, Alt, etc)
	Mods key.Modifiers

	// Key event code
	Key key.Codes
}

// SendKey processes a basic key event and sends it
func (em *Mgr) Key(typ Types, rn rune, code key.Codes, mods key.Modifiers) {
	ev := NewKey(typ, rn, code, mods)
	em.Last.Mods = mods
	em.Last.Key = code
	ev.Init()
	em.Deque.Send(ev)

	// don't include alt here.
	// missing: !mapped ||
	if typ == KeyDown && ev.Code < key.CodeLeftControl &&
		(ev.HasAnyModifier(key.Control, key.Meta) || ev.Code == key.CodeTab) {
		che := NewKey(KeyChord, rn, code, mods)
		che.Init()
		em.Deque.Send(che)
	}
}

// KeyChord processes a basic KeyChord event and sends it
func (em *Mgr) KeyChord(rn rune, code key.Codes, mods key.Modifiers) {
	ev := NewKey(KeyChord, rn, code, mods)
	// no further processing of these
	ev.Init()
	em.Deque.Send(ev)
}

// MouseButton creates and sends a mouse button event with given values
func (em *Mgr) MouseButton(typ Types, but Buttons, where image.Point, mods key.Modifiers) {
	ev := NewMouse(typ, but, where, mods)
	em.Last.Mods = mods
	em.Last.MouseButtonType = typ
	em.Last.MouseButton = but
	em.Last.MousePos = where
	if typ == MouseDown {
		em.Last.MouseDownPos = where
		interval := time.Now().Sub(em.Last.MouseDownTime.Time())
		if interval < DoubleClickInterval {
			ev.Typ = DoubleClick
		}
	}
	ev.Init()
	if typ == MouseDown {
		em.Last.MouseDownTime = ev.GenTime
		em.Last.MouseMoveTime = ev.GenTime
	}
	em.Deque.Send(ev)
}

// MouseMove creates and sends a mouse move or drag event with given values
func (em *Mgr) MouseMove(where image.Point) {
	lastPos := em.Last.MousePos
	var ev *Mouse
	if em.Last.MouseButtonType == MouseDown {
		ev = NewMouseDrag(em.Last.MouseButton, where, lastPos, em.Last.MouseDownPos, em.Last.Mods)
		ev.StTime = em.Last.MouseDownTime
		ev.PrvTime = em.Last.MouseMoveTime
	} else {
		ev = NewMouseMove(em.Last.MouseButton, where, lastPos, em.Last.Mods)
		ev.PrvTime = em.Last.MouseMoveTime
	}
	ev.Init()
	em.Last.MouseMoveTime = ev.GenTime
	// if em.Win.IsCursorEnabled() {
	// 	em.Last.MousePos = where
	// }
	em.Deque.Send(ev)
}

// Scroll creates and sends a scroll event with given values
func (em *Mgr) Scroll(where, delta image.Point) {
	ev := NewScroll(where, delta, em.Last.Mods)
	ev.Init()
	em.Deque.Send(ev)
}

//	func (em *Mgr) DND(act dnd.Actions, where image.Point, data mimedata.Mimes) {
//		ev := dnd.NewEvent(act, where, em.Last.Mods)
//		ev.Data = data
//		ev.Init()
//		em.Deque.Send(ev)
//	}

func (em *Mgr) Window(act WinActions) {
	ev := NewWindow(act)
	ev.Init()
	em.Deque.Send(ev)
}

func (em *Mgr) WindowPaint() {
	ev := NewWindowPaint()
	ev.Init()
	em.Deque.Send(ev)
}

func (em *Mgr) WindowResize() {
	ev := NewWindowResize()
	ev.Init()
	em.Deque.Send(ev)
}

func (em *Mgr) Custom(data any) {
	ce := &CustomEvent{}
	ce.Typ = Custom
	ce.Init()
	em.Deque.Send(ce)
}
