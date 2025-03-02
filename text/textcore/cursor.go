// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"
	"image"
	"image/draw"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/textpos"
)

var (
	// textcoreBlinker manages cursor blinking
	textcoreBlinker = core.Blinker{}

	// textcoreSpriteName is the name of the window sprite used for the cursor
	textcoreSpriteName = "textcore.Base.Cursor"
)

func init() {
	core.TheApp.AddQuitCleanFunc(textcoreBlinker.QuitClean)
	textcoreBlinker.Func = func() {
		w := textcoreBlinker.Widget
		textcoreBlinker.Unlock() // comes in locked
		if w == nil {
			return
		}
		ed := AsBase(w)
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
func (ed *Base) startCursor() {
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
	textcoreBlinker.SetWidget(ed.This.(core.Widget))
	textcoreBlinker.Blink(core.SystemSettings.CursorBlinkTime)
}

// clearCursor turns off cursor and stops it from blinking
func (ed *Base) clearCursor() {
	ed.stopCursor()
	ed.renderCursor(false)
}

// stopCursor stops the cursor from blinking
func (ed *Base) stopCursor() {
	if ed == nil || ed.This == nil {
		return
	}
	textcoreBlinker.ResetWidget(ed.This.(core.Widget))
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

// renderCursor renders the cursor on or off, as a sprite that is either on or off
func (ed *Base) renderCursor(on bool) {
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
	if !ed.posIsVisible(ed.CursorPos) {
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
func (ed *Base) cursorSpriteName() string {
	spnm := fmt.Sprintf("%v-%v", textcoreSpriteName, ed.charSize.Y)
	return spnm
}

// cursorSprite returns the sprite for the cursor, which is
// only rendered once with a vertical bar, and just activated and inactivated
// depending on render status.
func (ed *Base) cursorSprite(on bool) *core.Sprite {
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
		bbsz := image.Point{int(math32.Ceil(ed.CursorWidth.Dots)), int(math32.Ceil(ed.charSize.Y))}
		if bbsz.X < 2 { // at least 2
			bbsz.X = 2
		}
		sp = core.NewSprite(spnm, bbsz, image.Point{})
		if ed.CursorColor != nil {
			ibox := sp.Pixels.Bounds()
			draw.Draw(sp.Pixels, ibox, ed.CursorColor, image.Point{}, draw.Src)
			ms.Sprites.Add(sp)
		}
	}
	if on {
		ms.Sprites.ActivateSprite(sp.Name)
	} else {
		ms.Sprites.InactivateSprite(sp.Name)
	}
	return sp
}
