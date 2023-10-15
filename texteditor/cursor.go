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
var BlinkingView *View

// ViewSpriteName is the name of the window sprite used for the cursor
var ViewSpriteName = "textview.View.Cursor"

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
		if BlinkingView == nil || BlinkingView.This() == nil {
			ViewBlinkMu.Unlock()
			continue
		}
		if BlinkingView.Is(ki.Destroyed) || BlinkingView.Is(ki.Deleted) {
			BlinkingView = nil
			ViewBlinkMu.Unlock()
			continue
		}
		tv := BlinkingView
		if tv.Sc == nil || !tv.StateIs(states.Focused) || !tv.This().(gi.Widget).IsVisible() {
			tv.RenderCursor(false)
			BlinkingView = nil
			ViewBlinkMu.Unlock()
			continue
		}
		tv.BlinkOn = !tv.BlinkOn
		tv.RenderCursor(tv.BlinkOn)
		ViewBlinkMu.Unlock()
	}
}

// StartCursor starts the cursor blinking and renders it
func (tv *View) StartCursor() {
	if tv == nil || tv.This() == nil {
		return
	}
	if !tv.This().(gi.Widget).IsVisible() {
		return
	}
	tv.BlinkOn = true
	if gi.CursorBlinkTime == 0 {
		tv.RenderCursor(true)
		return
	}
	ViewBlinkMu.Lock()
	if ViewBlinker == nil {
		ViewBlinker = time.NewTicker(gi.CursorBlinkTime)
		go ViewBlink()
	}
	tv.BlinkOn = true
	tv.RenderCursor(true)
	BlinkingView = tv
	ViewBlinkMu.Unlock()
}

// StopCursor stops the cursor from blinking
func (tv *View) StopCursor() {
	if tv == nil || tv.This() == nil {
		return
	}
	if !tv.This().(gi.Widget).IsVisible() {
		return
	}
	tv.RenderCursor(false)
	ViewBlinkMu.Lock()
	if BlinkingView == tv {
		BlinkingView = nil
	}
	ViewBlinkMu.Unlock()
}

// CursorBBox returns a bounding-box for a cursor at given position
func (tv *View) CursorBBox(pos lex.Pos) image.Rectangle {
	cpos := tv.CharStartPos(pos)
	cbmin := cpos.SubScalar(tv.CursorWidth.Dots)
	cbmax := cpos.AddScalar(tv.CursorWidth.Dots)
	cbmax.Y += tv.FontHeight
	curBBox := image.Rectangle{cbmin.ToPointFloor(), cbmax.ToPointCeil()}
	return curBBox
}

// RenderCursor renders the cursor on or off, as a sprite that is either on or off
func (tv *View) RenderCursor(on bool) {
	if tv == nil || tv.This() == nil {
		return
	}
	if !tv.This().(gi.Widget).IsVisible() {
		return
	}
	if tv.Renders == nil {
		return
	}
	tv.CursorMu.Lock()
	defer tv.CursorMu.Unlock()

	sp := tv.CursorSprite(on)
	if sp == nil {
		return
	}
	sp.Geom.Pos = tv.CharStartPos(tv.CursorPos).ToPointFloor()
}

// CursorSpriteName returns the name of the cursor sprite
func (tv *View) CursorSpriteName() string {
	spnm := fmt.Sprintf("%v-%v", ViewSpriteName, tv.FontHeight)
	return spnm
}

// CursorSprite returns the sprite for the cursor, which is
// only rendered once with a vertical bar, and just activated and inactivated
// depending on render status.
func (tv *View) CursorSprite(on bool) *gi.Sprite {
	sc := tv.Sc
	if sc == nil {
		return nil
	}
	ms := sc.MainStage()
	if ms == nil {
		return nil // only MainStage has sprites
	}
	spnm := tv.CursorSpriteName()
	sp, ok := ms.Sprites.SpriteByName(spnm)
	if !ok {
		bbsz := image.Point{int(mat32.Ceil(tv.CursorWidth.Dots)), int(mat32.Ceil(tv.FontHeight))}
		if bbsz.X < 2 { // at least 2
			bbsz.X = 2
		}
		sp = gi.NewSprite(spnm, bbsz, image.Point{})
		ibox := sp.Pixels.Bounds()
		draw.Draw(sp.Pixels, ibox, &image.Uniform{tv.CursorColor.Solid}, image.Point{}, draw.Src)
		ms.Sprites.Add(sp)
	}
	if on {
		ms.Sprites.ActivateSprite(sp.Name)
	} else {
		ms.Sprites.InactivateSprite(sp.Name)
	}
	return sp
}
