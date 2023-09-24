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

// Mgr manages raw events to generate derived events
// such as MouseDragEvent
type Mgr struct {
	LastMouseClickTime time.Time
	LastMousePos       image.Point
	LastMouseButton    mouse.Buttons
	LastMouseButtonPos image.Point
	LastMouseAction    mouse.Actions
	LastMods           goosi.Modifiers
	LastKey            key.Codes
}

// Key processes a basic key event and sends it to given window
func (em *Mgr) Key(win goosi.Window, ev *key.Event) {
	em.LastMods = ev.Mods
	em.LastKey = ev.Code
	ev.Init()
	win.Send(ev)

	// don't include alt here
	// !mapped ||
	if ev.Action == key.Press && ev.Code < key.CodeLeftControl &&
		(ev.HasAnyModifier(goosi.Control, goosi.Meta) || ev.Code == key.CodeTab) {
		che := key.NewChordEvent(ev.Rune, ev.Code, ev.Action, ev.Mods)
		che.Init()
		win.Send(che)
	}
}

// KeyChord processes a basic KeyChord event and sends it to given window
func (em *Mgr) KeyChord(win goosi.Window, ev *key.Event) {
	// no further processing of these
	ev.Init()
	win.Send(ev)
}

// MouseButton processes a mouse button event before sending
func (em *Mgr) MouseButton(ev *mouse.Event) {
	em.LastMods = ev.Mods
	em.LastMouseButton = ev.Button
	em.LastMouseAction = ev.Action
	if ev.Action == mouse.Press {
		em.LastMouseButtonPos = where
	}
	if action == glfw.Press {
		interval := time.Now().Sub(w.EventMgr.LastMouseClickTime)
		// fmt.Printf("interval: %v\n", interval)
		if (interval / time.Millisecond) < time.Duration(mouse.DoubleClickMSec) {
			act = mouse.DoubleClick
		}
	}
	if act == mouse.Press {
		w.EventMgr.LastMouseClickTime = event.Time()
	}

}
