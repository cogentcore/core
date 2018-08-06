// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cursor defines the oswin cursor interface and standard system
// cursors that are supported across platforms
package cursor

import (
	"github.com/goki/ki/kit"
)

// Shapes are the standard cursor shapes available on all platforms
type Shapes int32

const (
	// Arrow is the standard arrow pointer
	Arrow Shapes = iota

	// Cross is a crosshair plus-like cursor -- typically used for precise actions.
	Cross

	// DragCopy indicates that the current drag operation will copy the dragged items
	DragCopy

	// DragMove indicates that the current drag operation will move the dragged items
	DragMove

	// DragLink indicates that the current drag operation will link the dragged items
	DragLink

	// HandPointing is a hand with a pointing index finger -- typically used
	// to indicate a link is clickable.
	HandPointing

	// HandOpen is an open hand -- typically used to indicate ability to click
	// and drag to move something.
	HandOpen

	// HandClosed is a closed hand -- typically used to indicate a dragging
	// operation involving scrolling.
	HandClosed

	// Help is an arrow and question mark indicating help is available.
	Help

	// IBeam is the standard text-entry symbol like a capital I.
	IBeam

	// Not is a slashed circle indicating operation not allowed (NO).
	Not

	// UpDown is Double-pointed arrow pointing up and down (SIZENS).
	UpDown

	// LeftRight is a Double-pointed arrow pointing west and east (SIZEWE).
	LeftRight

	// UpRight is a Double-pointed arrow pointing up-right and down-left (SIZEWE).
	UpRight

	// UpLeft is a Double-pointed arrow pointing up-left and down-right (SIZEWE).
	UpLeft

	// AllArrows is all four directions of arrow pointing.
	AllArrows

	// Wait is a system-dependent busy / wait cursor (typically an hourglass).
	Wait

	// ShapesN is number of standard cursor shapes
	ShapesN
)

//go:generate stringer -type=Shapes

var KiT_Shapes = kit.Enums.AddEnum(ShapesN, false, nil)

// Cursor manages the mouse cursor / pointer appearance.  Currently only a
// fixed set of standard cursors are supported, but in the future it will be
// possible to set the cursor from an image / svg.
type Cursor interface {

	// Push pushes a new active cursor.
	Push(sh Shapes)

	// Pop pops cursor off the stack and restores the previous cursor -- an
	// error message is emitted if no more cursors on the stack (programming
	// error).
	Pop()

	// Set sets the active cursor, without reference to the cursor stack --
	// generally not recommended for direct use -- prefer Push / Pop.
	Set(sh Shapes)

	// Hide hides the cursor
	Hide()

	// Show shows the cursor after a hide -- must always be balanced with Hide
	Show()
}
