// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"image/draw"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/parse/lexer"
)

var (
	// editorBlinker manages cursor blinking
	editorBlinker = core.Blinker{}

	// editorSpriteName is the name of the window sprite used for the cursor
	editorSpriteName = "texteditor.Editor.Cursor"
)

func init() {
	core.TheApp.AddQuitCleanFunc(editorBlinker.QuitClean)
	editorBlinker.Func = func() {
		w := editorBlinker.Widget
		editorBlinker.Unlock() // comes in locked
		if w == nil {
			return
		}
		ed := AsEditor(w)
		ed.AsyncLock()
		if !w.AsWidget().StateIs(states.Focused) || !w.AsWidget().IsVisible() {
			ed.blinkOn = false
			ed.renderCursor(false)
		} else {
			ed.blinkOn = !ed.blinkOn
			ed.renderCursor(ed.blinkOn)
		}
		ed.AsyncUnlock()
	}
}

// startCursor starts the cursor blinking and renders it
func (ed *Editor) startCursor() {
	if ed == nil || ed.This == nil {
		return
	}
	if !ed.IsVisible() {
		return
	}
	ed.blinkOn = true
	ed.renderCursor(true)
	if core.SystemSettings.CursorBlinkTime == 0 {
		return
	}
	editorBlinker.SetWidget(ed.This.(core.Widget))
	editorBlinker.Blink(core.SystemSettings.CursorBlinkTime)
}

// clearCursor turns off cursor and stops it from blinking
func (ed *Editor) clearCursor() {
	ed.stopCursor()
	ed.renderCursor(false)
}

// stopCursor stops the cursor from blinking
func (ed *Editor) stopCursor() {
	if ed == nil || ed.This == nil {
		return
	}
	editorBlinker.ResetWidget(ed.This.(core.Widget))
}

// cursorBBox returns a bounding-box for a cursor at given position
func (ed *Editor) cursorBBox(pos textpos.Pos) image.Rectangle {
	cpos := ed.charStartPos(pos)
	cbmin := cpos.SubScalar(ed.CursorWidth.Dots)
	cbmax := cpos.AddScalar(ed.CursorWidth.Dots)
	cbmax.Y += ed.fontHeight
	curBBox := image.Rectangle{cbmin.ToPointFloor(), cbmax.ToPointCeil()}
	return curBBox
}

// renderCursor renders the cursor on or off, as a sprite that is either on or off
func (ed *Editor) renderCursor(on bool) {
	if ed == nil || ed.This == nil {
		return
	}
	if !on {
		if ed.Scene == nil {
			return
		}
		ms := ed.Scene.Stage.Main
		if ms == nil {
			return
		}
		spnm := ed.cursorSpriteName()
		ms.Sprites.InactivateSprite(spnm)
		return
	}
	if !ed.IsVisible() {
		return
	}
	if ed.renders == nil {
		return
	}
	ed.cursorMu.Lock()
	defer ed.cursorMu.Unlock()

	sp := ed.cursorSprite(on)
	if sp == nil {
		return
	}
	sp.Geom.Pos = ed.charStartPos(ed.CursorPos).ToPointFloor()
}

// cursorSpriteName returns the name of the cursor sprite
func (ed *Editor) cursorSpriteName() string {
	spnm := fmt.Sprintf("%v-%v", editorSpriteName, ed.fontHeight)
	return spnm
}

// cursorSprite returns the sprite for the cursor, which is
// only rendered once with a vertical bar, and just activated and inactivated
// depending on render status.
func (ed *Editor) cursorSprite(on bool) *core.Sprite {
	sc := ed.Scene
	if sc == nil {
		return nil
	}
	ms := sc.Stage.Main
	if ms == nil {
		return nil // only MainStage has sprites
	}
	spnm := ed.cursorSpriteName()
	sp, ok := ms.Sprites.SpriteByName(spnm)
	if !ok {
		bbsz := image.Point{int(math32.Ceil(ed.CursorWidth.Dots)), int(math32.Ceil(ed.fontHeight))}
		if bbsz.X < 2 { // at least 2
			bbsz.X = 2
		}
		sp = core.NewSprite(spnm, bbsz, image.Point{})
		ibox := sp.Pixels.Bounds()
		draw.Draw(sp.Pixels, ibox, ed.CursorColor, image.Point{}, draw.Src)
		ms.Sprites.Add(sp)
	}
	if on {
		ms.Sprites.ActivateSprite(sp.Name)
	} else {
		ms.Sprites.InactivateSprite(sp.Name)
	}
	return sp
}
