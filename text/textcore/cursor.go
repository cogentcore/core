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
)

// cursorSpriteName is the name of the window sprite used for the cursor
var cursorSpriteName = "textcore.Base.Cursor"

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
	ed.cursorOn = on
	core.TextCursor(on, ed.AsWidget(), cursorSpriteName, ed.CursorWidth.Dots, ed.charSize.Y, ed.CursorColor, func() image.Point {
		return core.TextCursorPoint(ed.charStartPos(ed.CursorPos))
	})
}

// updateCursorPosition updates the position of the cursor.
func (ed *Base) updateCursorPosition() {
	// An editor that is not focused updates its cursor only if it is on
	// (as with [Base.SetCursorShow]) and no other widget has it now.
	if !ed.StateIs(states.Focused) {
		if !ed.cursorOn {
			return
		}
		if owner := core.TextCursorOwner(ed.AsWidget(), cursorSpriteName); owner != nil && owner != ed.This {
			return
		}
	}
	if ed.IsReadOnly() { // no cursor, but there can be an InputPos
		ed.updateTextInput()
		return
	}
	sc := ed.Scene
	if sc == nil || sc.Stage == nil || sc.Stage.Main == nil {
		return
	}
	ms := sc.Stage.Main
	ms.Sprites.Lock()
	sp, ok := ms.Sprites.SpriteByNameNoLock(cursorSpriteName)
	if ok {
		sp.EventBBox.Min = core.TextCursorPoint(ed.charStartPos(ed.CursorPos))
	}
	ms.Sprites.Unlock()
	// The sprites are removed when the scene is rebuilt or resized,
	// so the cursor is started again.
	if !ok {
		ed.startCursor()
	}
	ed.updateTextInput()
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
