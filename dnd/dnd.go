// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dnd defines the system drag-and-drop events for the GoGi GUI
// system.  These are generated internally for dnd's within a given window,
// and are automatically converted into external dnd events when the mouse
// leaves the window
package dnd

//go:generate enumgen

import (
	"image"

	"goki.dev/goosi"
	"goki.dev/goosi/mimedata"
)

// dnd.Event represents the drag-and-drop event, specifically the drop
type Event struct {
	goosi.EventBase

	// Action associated with the specific drag event: Start, Drop*, Move, Enter, Exit
	Action Actions

	// When event is received by target, Mod indicates the suggested modifier
	// action associated with the drop (affected by holding down modifier
	// keys), suggesting what to do with the dropped item, where appropriate
	// -- receivers can ignore or process in their own relevant way as needed,
	// BUT it is essential to update the event with the actual type of Mod
	// action taken, because the event will be sent back to the source with
	// this Mod as set by the receiver.  The main consequence is that a
	// DropMove requires the drop source to delete itself once the event has
	// been received -- otherwise it (typically) doesn't do anything, so just
	// be careful about that particular case.
	Mod DropMods

	// Data contains the MIME-typed data -- multiple different types are
	// possible (and encouraged)
	Data mimedata.Mimes

	// Source of the drop -- only available for internal DND actions
	Source any

	// Target of the drop -- receiver of an accepted drop should set this to
	// itself, so Source (if internal) can see who got it
	Target any
}

func NewEvent(act Actions, where image.Point, mods goosi.Modifiers) *Event {
	ev := &Event{}
	ev.Typ = goosi.DNDEvent
	ev.Action = act
	ev.Where = where
	ev.Mods = mods
	ev.DefaultMod()
	return ev
}

func NewMoveEvent() *Event {
	ev := &Event{}
	ev.Typ = goosi.DNDMoveEvent
	return ev
}

func NewFocusEvent() *Event {
	ev := &Event{}
	ev.Typ = goosi.DNDFocusEvent
	return ev
}

func (ev *Event) HasPos() bool {
	return true
}

func (ev *Event) OnFocus() bool {
	return false
}

/////////////////////////////////////////////////////////////////

// Actions associated with the DND event -- this is the nature of the event.
type Actions int32 //enums:enum

const (
	NoAction Actions = iota

	// Start is triggered when criteria for DND starting have been met -- it
	// is the chance for potential sources to start a DND event.
	Start

	// DropOnTarget is set when event is sent to the target where the item is dropped.
	DropOnTarget

	// DropFmSource is set when event is sent back to the source after the
	// target has been dropped on a valid target that did not ignore the event
	// -- the source should check if Mod = DropMove, and typically delete
	// itself in this case.
	DropFmSource

	// External is triggered from an external drop event
	External

	// Move is sent whenever mouse is moving while dragging -- usually not needed.
	Move

	// Enter is sent when drag enters a given widget, in a FocusEvent.
	Enter

	// Exit is sent when drag exits a given widget, in a FocusEvent.  Exit
	// from one widget always happens before entering another (so you can
	// reset cursor to Not).
	Exit

	// Hover is sent when drag is hovering over a widget without moving -- can
	// use this for spring-loaded opening of items to drag into, for example.
	Hover
)

// DropMods indicates the modifier associated with the drop action (affected by
// holding down modifier keys), suggesting what to do with the dropped item,
// where appropriate
type DropMods int32 //enums:enum

const (
	NoDropMod DropMods = iota

	// Copy is the default and implies data is just copied -- receiver can do
	// with it as they please and source does not need to take any further
	// action
	DropCopy

	// Move is signaled with a Shift or Meta key (by default) and implies that
	// the source should delete itself when it receives the DropFmSource event
	// action with this Mod value set -- receiver must update the Mod to
	// reflect actual action taken, and be particularly careful with this one
	DropMove

	// Link can be any other kind of alternative action -- link is applicable
	// to files (symbolic link)
	DropLink

	// Ignore means that the receiver chose to not process this drop
	DropIgnore
)

// DefaultModBits returns the default DropMod modifier action based on modifier keys
func DefaultModBits(modBits goosi.Modifiers) DropMods {
	switch {
	case goosi.HasAnyModifier(modBits, goosi.Control):
		return DropCopy
	case goosi.HasAnyModifier(modBits, goosi.Shift, goosi.Meta):
		return DropMove
	case goosi.HasAnyModifier(modBits, goosi.Alt):
		return DropLink
	default:
		return DropCopy
	}
}

// DefaultMod sets the default DropMod modifier action based on modifier keys
func (e *Event) DefaultMod() {
	e.Mod = DefaultModBits(e.Mods)
}
