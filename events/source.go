// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
	"image"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/nptime"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
)

// TraceWindowPaint prints out a . for each WindowPaint event,
// - for other window events, and * for mouse move events.
// Makes it easier to see what is going on in the overall flow.
var TraceWindowPaint = false

// Source is a source of events that manages the event
// construction and sending process for its parent window.
// It caches state as needed to generate derived events such
// as [MouseDrag].
type Source struct {
	// Deque is the event queue
	Deque Deque

	// flag for ignoring mouse events when disabling mouse movement
	ResettingPos bool

	// Last has the prior state for key variables
	Last SourceState

	// PaintCount is used for printing paint events as .
	PaintCount int
}

// SourceState tracks basic event state over time
// to enable recognition and full data for generating events.
type SourceState struct {
	// last mouse button event type (down or up)
	MouseButtonType Types

	// last mouse button
	MouseButton Buttons

	// time of MouseDown
	MouseDownTime nptime.Time

	// position at MouseDown
	MouseDownPos image.Point

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
func (es *Source) Key(typ Types, rn rune, code key.Codes, mods key.Modifiers) {
	ev := NewKey(typ, rn, code, mods)
	es.Last.Mods = mods
	es.Last.Key = code
	ev.Init()
	es.Deque.Send(ev)

	_, mapped := key.CodeRuneMap[code]

	if typ == KeyDown && ev.Code < key.CodeLeftControl &&
		(ev.HasAnyModifier(key.Control, key.Meta) || !mapped || ev.Code == key.CodeTab) {
		che := NewKey(KeyChord, rn, code, mods)
		che.Init()
		es.Deque.Send(che)
	}
}

// KeyChord processes a basic KeyChord event and sends it
func (es *Source) KeyChord(rn rune, code key.Codes, mods key.Modifiers) {
	ev := NewKey(KeyChord, rn, code, mods)
	// no further processing of these
	ev.Init()
	es.Deque.Send(ev)
}

// MouseButton creates and sends a mouse button event with given values
func (es *Source) MouseButton(typ Types, but Buttons, where image.Point, mods key.Modifiers) {
	ev := NewMouse(typ, but, where, mods)
	if typ != MouseDown && es.Last.MouseButtonType == MouseDown {
		ev.StTime = es.Last.MouseDownTime
		ev.PrvTime = es.Last.MouseMoveTime
		ev.Start = es.Last.MouseDownPos
		ev.Prev = es.Last.MousePos
	}
	es.Last.Mods = mods
	es.Last.MouseButtonType = typ
	es.Last.MouseButton = but
	es.Last.MousePos = where
	ev.Init()
	if typ == MouseDown {
		es.Last.MouseDownPos = where
		es.Last.MouseDownTime = ev.GenTime
		es.Last.MouseMoveTime = ev.GenTime
	}
	es.Deque.Send(ev)
}

// MouseMove creates and sends a mouse move or drag event with given values
func (es *Source) MouseMove(where image.Point) {
	lastPos := es.Last.MousePos
	var ev *Mouse
	if es.Last.MouseButtonType == MouseDown {
		ev = NewMouseDrag(es.Last.MouseButton, where, lastPos, es.Last.MouseDownPos, es.Last.Mods)
		ev.StTime = es.Last.MouseDownTime
		ev.PrvTime = es.Last.MouseMoveTime
	} else {
		ev = NewMouseMove(es.Last.MouseButton, where, lastPos, es.Last.Mods)
		ev.PrvTime = es.Last.MouseMoveTime
	}
	ev.Init()
	es.Last.MouseMoveTime = ev.GenTime
	// if em.Win.IsCursorEnabled() {
	es.Last.MousePos = where
	// }
	if TraceWindowPaint {
		fmt.Printf("*")
	}
	es.Deque.Send(ev)
}

// Scroll creates and sends a scroll event with given values
func (es *Source) Scroll(where image.Point, delta math32.Vector2) {
	ev := NewScroll(where, delta, es.Last.Mods)
	ev.Init()
	es.Deque.Send(ev)
}

// DropExternal creates and sends a Drop event with given values
func (es *Source) DropExternal(where image.Point, md mimedata.Mimes) {
	ev := NewExternalDrop(Drop, es.Last.MouseButton, where, es.Last.Mods, md)
	es.Last.MousePos = where
	ev.Init()
	es.Deque.Send(ev)
}

// Touch creates and sends a touch event with the given values.
// It also creates and sends a corresponding mouse event.
func (es *Source) Touch(typ Types, seq Sequence, where image.Point) {
	ev := NewTouch(typ, seq, where)
	ev.Init()
	es.Deque.Send(ev)

	if typ == TouchStart {
		es.MouseButton(MouseDown, Left, where, 0) // TODO: modifiers
	} else if typ == TouchEnd {
		es.MouseButton(MouseUp, Left, where, 0) // TODO: modifiers
	} else {
		es.MouseMove(where)
	}
}

// Magnify creates and sends a [TouchMagnify] event with the given values.
func (es *Source) Magnify(scaleFactor float32, where image.Point) {
	ev := NewMagnify(scaleFactor, where)
	ev.Init()
	es.Deque.Send(ev)
}

//	func (es *Source) DND(act dnd.Actions, where image.Point, data mimedata.Mimes) {
//		ev := dnd.NewEvent(act, where, em.Last.Mods)
//		ev.Data = data
//		ev.Init()
//		es.Deque.Send(ev)
//	}

func (es *Source) Window(act WinActions) {
	ev := NewWindow(act)
	ev.Init()
	if TraceWindowPaint {
		fmt.Printf("-")
	}
	es.Deque.SendFirst(ev)
}

func (es *Source) WindowPaint() {
	ev := NewWindowPaint()
	ev.Init()
	if TraceWindowPaint {
		fmt.Printf(".")
		es.PaintCount++
		if es.PaintCount > 60 {
			fmt.Println("")
			es.PaintCount = 0
		}
	}
	es.Deque.SendFirst(ev) // separate channel for window!
}

func (es *Source) WindowResize() {
	ev := NewWindowResize()
	ev.Init()
	if TraceWindowPaint {
		fmt.Printf("r")
	}
	es.Deque.SendFirst(ev)
}

func (es *Source) Custom(data any) {
	ce := &CustomEvent{}
	ce.Typ = Custom
	ce.Data = data
	ce.Init()
	es.Deque.Send(ce)
}
