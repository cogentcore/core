// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"image/draw"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/pi/lex"
	"cogentcore.org/core/states"
)

func init() {
	gi.AddQuitCleanFunc(EditorBlinker.QuitClean)
}

var (
	// EditorBlinker manages cursor blinking
	EditorBlinker = gi.Blinker{}

	// EditorSpriteName is the name of the window sprite used for the cursor
	EditorSpriteName = "texteditor.Editor.Cursor"
)

// StartCursor starts the cursor blinking and renders it
func (ed *Editor) StartCursor() {
	if ed == nil || ed.This() == nil {
		return
	}
	if !ed.This().(gi.Widget).IsVisible() {
		return
	}
	ed.BlinkOn = true
	ed.RenderCursor(true)
	if gi.SystemSettings.CursorBlinkTime == 0 {
		return
	}
	EditorBlinker.Blink(gi.SystemSettings.CursorBlinkTime, func(w gi.Widget) {
		eed := AsEditor(w)
		if !eed.StateIs(states.Focused) || !w.IsVisible() {
			eed.BlinkOn = false
			eed.RenderCursor(false)
			EditorBlinker.Widget = nil
		} else {
			eed.BlinkOn = !eed.BlinkOn
			eed.RenderCursor(eed.BlinkOn)
		}
	})
	EditorBlinker.SetWidget(ed.This().(gi.Widget))
}

// ClearCursor turns off cursor and stops it from blinking
func (ed *Editor) ClearCursor() {
	// if tf.IsReadOnly() {
	// 	return
	// }
	ed.StopCursor()
	ed.RenderCursor(false)
}

// StopCursor stops the cursor from blinking
func (ed *Editor) StopCursor() {
	if ed == nil || ed.This() == nil {
		return
	}
	EditorBlinker.ResetWidget(ed.This().(gi.Widget))
}

// CursorBBox returns a bounding-box for a cursor at given position
func (ed *Editor) CursorBBox(pos lex.Pos) image.Rectangle {
	cpos := ed.CharStartPos(pos)
	cbmin := cpos.SubScalar(ed.CursorWidth.Dots)
	cbmax := cpos.AddScalar(ed.CursorWidth.Dots)
	cbmax.Y += ed.FontHeight
	curBBox := image.Rectangle{cbmin.ToPointFloor(), cbmax.ToPointCeil()}
	return curBBox
}

// RenderCursor renders the cursor on or off, as a sprite that is either on or off
func (ed *Editor) RenderCursor(on bool) {
	if ed == nil || ed.This() == nil {
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
	if !ed.This().(gi.Widget).IsVisible() {
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
func (ed *Editor) CursorSprite(on bool) *gi.Sprite {
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
		bbsz := image.Point{int(mat32.Ceil(ed.CursorWidth.Dots)), int(mat32.Ceil(ed.FontHeight))}
		if bbsz.X < 2 { // at least 2
			bbsz.X = 2
		}
		sp = gi.NewSprite(spnm, bbsz, image.Point{})
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
