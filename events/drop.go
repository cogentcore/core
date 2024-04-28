// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
	"image"

	"cogentcore.org/core/events/key"
)

// DragDrop represents the drag-and-drop Drop event
type DragDrop struct {
	Base

	// When event is received by target, DropMod indicates the suggested modifier
	// action associated with the drop (affected by holding down modifier
	// keys), suggesting what to do with the dropped item, where appropriate.
	// Receivers can ignore or process in their own relevant way as needed,
	// BUT it is essential to update the event with the actual type of Mod
	// action taken, because the event will be sent back to the source with
	// this Mod as set by the receiver.  The main consequence is that a
	// DropMove requires the drop source to delete itself once the event has
	// been received, otherwise it (typically) doesn't do anything, so just
	// be careful about that particular case.
	DropMod DropMods

	// Data contains the data from the Source of the drag,
	// typically a mimedata encoded representation.
	Data any

	// Source of the drop, only available for internal DND actions.
	// If it is an external drop, this will be nil.
	Source any

	// Target of the drop -- receiver of an accepted drop should set this to
	// itself, so Source (if internal) can see who got it
	Target any
}

func NewDragDrop(typ Types, mdrag *Mouse) *DragDrop {
	ev := &DragDrop{}
	ev.Base = mdrag.Base
	ev.Flags.SetFlag(false, Handled)
	ev.Typ = typ
	ev.DefaultMod()
	return ev
}

func NewExternalDrop(typ Types, but Buttons, where image.Point, mods key.Modifiers, data any) *DragDrop {
	ev := &DragDrop{}
	ev.Typ = typ
	ev.SetUnique()
	ev.Button = but
	ev.Where = where
	ev.Mods = mods
	ev.Data = data
	return ev
}

func (ev *DragDrop) String() string {
	return fmt.Sprintf("%v{Button: %v, Pos: %v, Mods: %v, Time: %v}", ev.Type(), ev.Button, ev.Where, ev.Mods.ModifiersString(), ev.Time().Format("04:05"))
}

func (ev *DragDrop) HasPos() bool {
	return true
}

// DropMods indicates the modifier associated with the drop action (affected by
// holding down modifier keys), suggesting what to do with the dropped item,
// where appropriate
type DropMods int32 //enums:enum -trim-prefix Drop

const (
	NoDropMod DropMods = iota

	// Copy is the default and implies data is just copied -- receiver can do
	// with it as they please and source does not need to take any further
	// action
	DropCopy

	// Move is signaled with a Shift or Meta key (by default) and implies that
	// the source should delete itself when it receives the DropFromSource event
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
func DefaultModBits(mods key.Modifiers) DropMods {
	switch {
	case key.HasAnyModifier(mods, key.Control):
		return DropCopy
	case key.HasAnyModifier(mods, key.Shift, key.Meta):
		return DropMove
	case key.HasAnyModifier(mods, key.Alt):
		return DropLink
	default:
		return DropCopy
	}
}

// DefaultMod sets the default DropMod modifier action based on modifier keys
func (e *DragDrop) DefaultMod() {
	e.DropMod = DefaultModBits(e.Mods)
}
