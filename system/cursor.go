// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"cogentcore.org/core/enums"
	"cogentcore.org/core/styles/units"
)

// Cursor manages the mouse cursor / pointer appearance.
type Cursor interface {

	// Current returns the current cursor as an enum, which is a
	// [cogentcore.org/core/cursors.Cursor]
	// by default, but could be something else if you are extending
	// the default cursor set.
	Current() enums.Enum

	// Set sets the active cursor to the given cursor as an enum, which is typically
	// a [cursors.Cursor], unless you are extending the default cursor set, in
	// which case it should be a type you defined. The string version of the
	// enum value must correspond to a filename of the form "name.svg" in
	// [cogentcore.org/core/cursors.Cursors]; this will be satisfied automatically by all
	// [cursor.Cursor] values.
	Set(cursor enums.Enum) error

	// IsVisible returns whether cursor is currently visible (according to [Cursor.Hide] and [Cursor.Show] actions)
	IsVisible() bool

	// Hide hides the cursor if it is not already hidden.
	Hide()

	// Show shows the cursor after a hide if it is hidden.
	Show()

	// SetSize sets the size that cursors are rendered at.
	// The standard size is [units.Dp](32).
	SetSize(size units.Value)
}

// CursorBase provides the common infrastructure for the [Cursor] interface,
// to be extended on desktop platforms. It can also be used as an empty
// implementation of the [Cursor] interface on mobile platforms, as they
// do not have cursors.
type CursorBase struct {
	// Cur is the current cursor, which is maintained by the standard methods.
	Cur enums.Enum

	// Vis is whether the cursor is visible; be sure to initialize to true!
	Vis bool

	// Size is the size that cursors are rendered at.
	// The standard size is [units.Dp](32).
	Size units.Value
}

// CursorBase should be a valid cursor so that it can be used directly in mobile
var _ Cursor = (*CursorBase)(nil)

func (c *CursorBase) Current() enums.Enum {
	return c.Cur
}

func (c *CursorBase) Set(cursor enums.Enum) error {
	c.Cur = cursor
	return nil
}

func (c *CursorBase) IsVisible() bool {
	return c.Vis
}

func (c *CursorBase) Hide() {
	c.Vis = false
}

func (c *CursorBase) Show() {
	c.Vis = true
}

func (c *CursorBase) SetSize(size units.Value) {
	c.Size = size
}
