// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package eventmgr

import (
	"image"
	"time"

	"goki.dev/goosi"
	"goki.dev/goosi/key"
	"goki.dev/goosi/mouse"
)

// Mgr manages the event construction and sending process,
// for its parent window.  Caches state as needed,
// to generate derived events such as MouseDragEvent.
type Mgr struct {
	Win goosi.Window // must be set to parent window

	LastMouseClickTime time.Time
	LastMousePos       image.Point
	LastMouseButton    mouse.Buttons
	LastMouseButtonPos image.Point
	LastMouseAction    mouse.Actions
	LastMouseMoveTime  time.Time
	LastMods           goosi.Modifiers
	LastKey            key.Codes

	// flag for ignoring mouse events when disabling mouse movement
	ResettingPos bool
}

// Key processes a basic key event and sends it to given window
func (em *Mgr) Key(rn rune, code key.Codes, act key.Actions, mods goosi.Modifiers) {
	ev := key.NewEvent(rn, code, act, mods)
	em.LastMods = mods
	em.LastKey = code
	ev.Init()
	em.Win.Send(ev)

	// don't include alt here.
	// missing: !mapped ||
	if ev.Action == key.Press && ev.Code < key.CodeLeftControl &&
		(ev.HasAnyModifier(goosi.Control, goosi.Meta) || ev.Code == key.CodeTab) {
		che := key.NewChordEvent(rn, code, act, mods)
		che.Init()
		em.Win.Send(che)
	}
}

// KeyChord processes a basic KeyChord event and sends it to given window
func (em *Mgr) KeyChord(rn rune, code key.Codes, act key.Actions, mods goosi.Modifiers) {
	ev := key.NewChordEvent(rn, code, act, mods)
	// no further processing of these
	ev.Init()
	em.Win.Send(ev)
}

// MouseButton creates and sends a mouse button event with given values
func (em *Mgr) MouseButton(but mouse.Buttons, act mouse.Actions, where image.Point, mods goosi.Modifiers) {
	ev := mouse.NewEvent(but, act, where, mods)
	em.LastMods = mods
	em.LastMouseButton = but
	em.LastMouseAction = act
	em.LastMousePos = where
	if ev.Action == mouse.Press {
		em.LastMouseButtonPos = where
	}
	if act == mouse.Press {
		interval := time.Now().Sub(em.LastMouseClickTime)
		if interval < mouse.DoubleClickInterval {
			ev.Action = mouse.DoubleClick
		}
	}
	ev.Init()
	if act == mouse.Press {
		em.LastMouseClickTime = ev.Time()
		em.LastMouseMoveTime = ev.Time()
	}
	em.Win.Send(ev)
}

// MouseScroll creates and sends a mouse scroll event with given values
func (em *Mgr) MouseScroll(where, delta image.Point) {
	ev := mouse.NewScrollEvent(where, delta, em.LastMods)
	ev.Init()
	em.Win.Send(ev)
}

// MouseMove creates and sends a mouse move or drag event with given values
func (em *Mgr) MouseMove(where image.Point) {
	lastPos := em.LastMousePos
	var ev *mouse.Event
	if em.LastMouseAction == mouse.Press {
		ev = mouse.NewDragEvent(em.LastMouseButton, where, lastPos, em.LastMouseButtonPos, em.LastMods)
		ev.StTime.SetTime(em.LastMouseClickTime)
		ev.PrvTime.SetTime(em.LastMouseMoveTime)
	} else {
		ev = mouse.NewMoveEvent(em.LastMouseButton, where, lastPos, em.LastMods)
		ev.PrvTime.SetTime(em.LastMouseMoveTime)
	}
	ev.Init()
	em.LastMouseMoveTime = ev.Time()
	if em.Win.IsCursorEnabled() {
		em.LastMousePos = where
	}
	em.Win.Send(ev)
}
