// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"image"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/tree"
)

var (
	// cursorSpriteName is the name of the window sprite used for the cursor
	cursorSpriteName = "textcore.Base.Cursor"

	// lastCursor is the *Base that last created a cursor sprite.
	lastCursor tree.Node
)

// startCursor starts the cursor blinking and renders it.
// This must be called to update the cursor position -- is called in render.
func (ed *Base) startCursor() {
	if ed == nil || ed.This == nil || !ed.IsVisible() {
		return
	}
	if ed.IsReadOnly() || !ed.AbilityIs(abilities.Focusable) {
		return
	}
	ed.toggleCursor(true)
}

// stopCursor stops the cursor from blinking
func (ed *Base) stopCursor() {
	ed.toggleCursor(false)
}

// toggleSprite turns on or off the cursor sprite.
func (ed *Base) toggleCursor(on bool) {
	core.TextCursor(on, ed.AsWidget(), &lastCursor, cursorSpriteName, ed.CursorWidth.Dots, ed.charSize.Y, ed.CursorColor, func() image.Point {
		return ed.charStartPos(ed.CursorPos).ToPointFloor()
	})
}

// updateCursorPosition updates the position of the cursor.
func (ed *Base) updateCursorPosition() {
	if ed.IsReadOnly() || !ed.StateIs(states.Focused) {
		return
	}
	sc := ed.Scene
	if sc == nil || sc.Stage == nil || sc.Stage.Main == nil {
		return
	}
	ms := sc.Stage.Main
	ms.Sprites.Lock()
	defer ms.Sprites.Unlock()
	if sp, ok := ms.Sprites.SpriteByNameNoLock(cursorSpriteName); ok {
		sp.EventBBox.Min = ed.charStartPos(ed.CursorPos).ToPointFloor()
	}
}

// cursorBBox returns a bounding-box for a cursor at given position
func (ed *Base) cursorBBox(pos textpos.Pos) image.Rectangle {
	cpos := ed.charStartPos(pos)
	cbmin := cpos.SubScalar(ed.CursorWidth.Dots)
	cbmax := cpos.AddScalar(ed.CursorWidth.Dots)
	cbmax.Y += ed.charSize.Y
	curBBox := image.Rectangle{cbmin.ToPointFloor(), cbmax.ToPointCeil()}
	return curBBox
}
