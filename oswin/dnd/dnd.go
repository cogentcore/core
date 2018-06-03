// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dnd defines the system drag-and-drop events for the GoGi GUI
// system.  These are generated internally for dnd's within a given window,
// and are automatically converted into external dnd events when the mouse
// leaves the window
package dnd

import (
	"image"
	"time"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/ki/kit"
)

// dnd.Event represents the drag-and-drop event, specifically the drop
type Event struct {
	oswin.EventBase

	// Where is the mouse location, in raw display dots (raw, actual pixels)
	Where image.Point

	// Action associated with the specific drag event: Drop, Move, Enter, Exit
	Action Action

	// Mod indicates the modifier associated with the drop action
	// (affected by holding down modifier keys), suggesting what to do with
	// the dropped item, where appropriate
	Mod DropMod

	// Data contains the MIME-typed data -- multiple different types are
	// possible (and encouraged)
	Data mimedata.Mimes
}

/////////////////////////////////////////////////////////////////

// dnd.MoveEvent is emitted when dnd is moved
type MoveEvent struct {
	Event

	// From is the previous location of the mouse
	From image.Point

	// LastTime is the time of the previous event
	LastTime time.Time
}

/////////////////////////////////////////////////////////////////

// dnd.FocusEvent records actions of Enter and Exit of DND into a given widget
// bounding box -- generated in gi.Window, which knows about widget bounding
// boxes
type FocusEvent struct {
	Event
}

// Action associated with the DND event -- this is the nature of the event
type Action int32

const (
	NoAction Action = iota
	Drop
	Move
	Enter
	Exit

	ActionN
)

//go:generate stringer -type=Action

var KiT_Action = kit.Enums.AddEnum(ActionN, false, nil)

// DropMod indicates the modifier associated with the drop action (affected by
// holding down modifier keys), suggesting what to do with the dropped item,
// where appropriate
type DropMod int32

const (
	NoDropMod DropMod = iota
	DropCopy
	DropMove
	DropLink
	DropIgnore

	DropModN
)

//go:generate stringer -type=DropMod

var KiT_DropMod = kit.Enums.AddEnum(DropModN, false, nil)
