// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cursor defines the oswin cursor interface and standard system
// cursors that are supported across platforms
package cursor

import (
	"fmt"
	"log"

	"github.com/goki/ki/kit"
)

// todo: apps can add new named shapes starting at ShapesN

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

var KiT_Shapes = kit.Enums.AddEnum(ShapesN, kit.NotBitFlag, nil)

// Drags is a map-set of cursors used for signaling dragging events.
var Drags = map[Shapes]struct{}{
	DragCopy: {},
	DragMove: {},
	DragLink: {},
}

// Cursor manages the mouse cursor / pointer appearance.  Currently only a
// fixed set of standard cursors are supported, but in the future it will be
// possible to set the cursor from an image / svg.
type Cursor interface {

	// Current returns the current shape of the cursor.
	Current() Shapes

	// Push pushes a new active cursor.
	Push(sh Shapes)

	// PushIfNot pushes a new active cursor if it is not already set to given shape.
	PushIfNot(sh Shapes) bool

	// Pop pops cursor off the stack and restores the previous cursor -- an
	// error message is emitted if no more cursors on the stack (programming
	// error).
	Pop()

	// PopIf pops cursor off the stack and restores the previous cursor if the
	// current cursor is the given shape.
	PopIf(sh Shapes) bool

	// Set sets the active cursor, without reference to the cursor stack --
	// generally not recommended for direct use -- prefer Push / Pop.
	Set(sh Shapes)

	// IsVisible returns whether cursor is currently visible (according to Hide / show actions)
	IsVisible() bool

	// Hide hides the cursor if it is not already hidden.
	Hide()

	// Show shows the cursor after a hide if it is hidden.
	Show()

	// IsDrag returns true if the current cursor is used for signaling dragging events.
	IsDrag() bool
}

// CursorBase provides the common infrastructure for Cursor interface.
type CursorBase struct {

	// Stack is the stack of shapes from push / pop actions.
	Stack []Shapes

	// Cur is current shape -- maintained by std methods.
	Cur Shapes

	// Vis is visibility: be sure to initialize to true!
	Vis bool
}

func (c *CursorBase) Current() Shapes {
	return c.Cur
}

func (c *CursorBase) IsVisible() bool {
	return c.Vis
}

func (c *CursorBase) IsDrag() bool {
	_, has := Drags[c.Cur]
	return has
}

// PushStack pushes item on the stack
func (c *CursorBase) PushStack(sh Shapes) {
	c.Cur = sh
	c.Stack = append(c.Stack, sh)
}

// PopStack pops item off the stack, returning 2nd-to-last item on stack
func (c *CursorBase) PopStack() (Shapes, error) {
	sz := len(c.Stack)
	if len(c.Stack) == 0 {
		err := fmt.Errorf("gi.oswin.cursor PopStack: stack is empty -- programmer error\n")
		log.Print(err)
		return Arrow, err
	}
	c.Stack = c.Stack[:sz-1]
	c.Cur = c.PeekStack()
	return c.Cur, nil
}

// PeekStack returns top item on the stack (default Arrow if nothing on stack)
func (c *CursorBase) PeekStack() Shapes {
	sz := len(c.Stack)
	if len(c.Stack) == 0 {
		return Arrow
	}
	return c.Stack[sz-1]
}
