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
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/styles/states"
)

var (
	// EditorBlinker manages cursor blinking
	EditorBlinker = core.Blinker{}

	// EditorSpriteName is the name of the window sprite used for the cursor
	EditorSpriteName = "texteditor.Editor.Cursor"
)

func init() {
	core.TheApp.AddQuitCleanFunc(EditorBlinker.QuitClean)
	EditorBlinker.Func = func() {
		w := EditorBlinker.Widget
		if w == nil {
			return
		}
		ed := AsEditor(w)
		if !w.StateIs(states.Focused) || !w.IsVisible() {
			ed.BlinkOn = false
			ed.RenderCursor(false)
		} else {
			ed.BlinkOn = !ed.BlinkOn
			ed.RenderCursor(ed.BlinkOn)
		}
	}
}

// StartCursor starts the cursor blinking and renders it
func (ed *Editor) StartCursor() {
	if ed == nil || ed.This == nil {
		return
	}
	if !ed.This.(core.Widget).IsVisible() {
		return
	}
	ed.BlinkOn = true
	ed.RenderCursor(true)
	if core.SystemSettings.CursorBlinkTime == 0 {
		return
	}
	EditorBlinker.SetWidget(ed.This.(core.Widget))
	EditorBlinker.Blink(core.SystemSettings.CursorBlinkTime)
}

// ClearCursor turns off cursor and stops it from blinking
func (ed *Editor) ClearCursor() {
	ed.StopCursor()
	ed.RenderCursor(false)
}

// StopCursor stops the cursor from blinking
func (ed *Editor) StopCursor() {
	if ed == nil || ed.This == nil {
		return
	}
	EditorBlinker.ResetWidget(ed.This.(core.Widget))
}

// CursorBBox returns a bounding-box for a cursor at given position
func (ed *Editor) CursorBBox(pos lexer.Pos) image.Rectangle {
	cpos := ed.CharStartPos(pos)
	cbmin := cpos.SubScalar(ed.CursorWidth.Dots)
	cbmax := cpos.AddScalar(ed.CursorWidth.Dots)
	cbmax.Y += ed.FontHeight
	curBBox := image.Rectangle{cbmin.ToPointFloor(), cbmax.ToPointCeil()}
	return curBBox
}

// RenderCursor renders the cursor on or off, as a sprite that is either on or off
func (ed *Editor) RenderCursor(on bool) {
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
		spnm := ed.CursorSpriteName()
		ms.Sprites.InactivateSprite(spnm)
		return
	}
	if !ed.This.(core.Widget).IsVisible() {
		return
	}
	if ed.Renders == nil {
		return
	}
	ed.CursorMu.Lock()
	defer ed.CursorMu.Unlock()

	sp := ed.CursorSprite(on)
	if sp == nil {
		return
	}
	sp.Geom.Pos = ed.CharStartPos(ed.CursorPos).ToPointFloor()
}

// CursorSpriteName returns the name of the cursor sprite
func (ed *Editor) CursorSpriteName() string {
	spnm := fmt.Sprintf("%v-%v", EditorSpriteName, ed.FontHeight)
	return spnm
}

// CursorSprite returns the sprite for the cursor, which is
// only rendered once with a vertical bar, and just activated and inactivated
// depending on render status.
func (ed *Editor) CursorSprite(on bool) *core.Sprite {
	sc := ed.Scene
	if sc == nil {
		return nil
	}
	ms := sc.Stage.Main
	if ms == nil {
		return nil // only MainStage has sprites
	}
	spnm := ed.CursorSpriteName()
	sp, ok := ms.Sprites.SpriteByName(spnm)
	if !ok {
		bbsz := image.Point{int(math32.Ceil(ed.CursorWidth.Dots)), int(math32.Ceil(ed.FontHeight))}
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
