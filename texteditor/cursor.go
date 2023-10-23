// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"
	"image"
	"image/draw"
	"sync"
	"time"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/states"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/lex"
)

// ViewBlinkMu is mutex protecting ViewBlink updating and access
var ViewBlinkMu sync.Mutex

// ViewBlinker is the time.Ticker for blinking cursors for text fields,
// only one of which can be active at at a time
var ViewBlinker *time.Ticker

// BlinkingView is the text field that is blinking
var BlinkingView *Editor

// ViewSpriteName is the name of the window sprite used for the cursor
var ViewSpriteName = "texteditor.Editor.Cursor"

// ViewBlink is function that blinks text field cursor
func ViewBlink() {
	for {
		ViewBlinkMu.Lock()
		if ViewBlinker == nil {
			ViewBlinkMu.Unlock()
			return // shutdown..
		}
		ViewBlinkMu.Unlock()
		<-ViewBlinker.C
		ViewBlinkMu.Lock()
		if BlinkingView == nil || BlinkingView.This() == nil || BlinkingView.Is(ki.Deleted) {
			ViewBlinkMu.Unlock()
			continue
		}
		ed := BlinkingView
		if ed.Sc == nil || !ed.StateIs(states.Focused) || !ed.This().(gi.Widget).IsVisible() {
			ed.RenderCursor(false)
			BlinkingView = nil
			ViewBlinkMu.Unlock()
			continue
		}
		ed.BlinkOn = !ed.BlinkOn
		ed.RenderCursor(ed.BlinkOn)
		ViewBlinkMu.Unlock()
	}
}

// StartCursor starts the cursor blinking and renders it
func (ed *Editor) StartCursor() {
	if ed == nil || ed.This() == nil {
		return
	}
	if !ed.This().(gi.Widget).IsVisible() {
		return
	}
	ed.BlinkOn = true
	if gi.CursorBlinkTime == 0 {
		ed.RenderCursor(true)
		return
	}
	ViewBlinkMu.Lock()
	if ViewBlinker == nil {
		ViewBlinker = time.NewTicker(gi.CursorBlinkTime)
		go ViewBlink()
	}
	ed.BlinkOn = true
	ed.RenderCursor(true)
	BlinkingView = ed
	ViewBlinkMu.Unlock()
}

// StopCursor stops the cursor from blinking
func (ed *Editor) StopCursor() {
	if ed == nil || ed.This() == nil {
		return
	}
	if !ed.This().(gi.Widget).IsVisible() {
		return
	}
	ed.RenderCursor(false)
	ViewBlinkMu.Lock()
	if BlinkingView == ed {
		BlinkingView = nil
	}
	ViewBlinkMu.Unlock()
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
	spnm := fmt.Sprintf("%v-%v", ViewSpriteName, ed.FontHeight)
	return spnm
}

// CursorSprite returns the sprite for the cursor, which is
// only rendered once with a vertical bar, and just activated and inactivated
// depending on render status.
func (ed *Editor) CursorSprite(on bool) *gi.Sprite {
	sc := ed.Sc
	if sc == nil {
		return nil
	}
	ms := sc.MainStage()
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
		draw.Draw(sp.Pixels, ibox, &image.Uniform{ed.CursorColor.Solid}, image.Point{}, draw.Src)
		ms.Sprites.Add(sp)
	}
	if on {
		ms.Sprites.ActivateSprite(sp.Name)
	} else {
		ms.Sprites.InactivateSprite(sp.Name)
	}
	return sp
}
