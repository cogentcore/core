// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"fmt"
	"image"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/text/textpos"
)

var (
	// blinkerSpriteName is the name of the window sprite used for the cursor
	blinkerSpriteName = "textcore.Base.Cursor"
)

// startCursor starts the cursor blinking and renders it
func (ed *Base) startCursor() {
	if ed == nil || ed.This == nil {
		return
	}
	if !ed.IsVisible() {
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
	sc := ed.Scene
	if sc == nil {
		return
	}
	ms := sc.Stage.Main
	if ms == nil {
		return // only MainStage has sprites
	}
	spnm := fmt.Sprintf("%v-%v", blinkerSpriteName, ed.charSize.Y)
	ms.Sprites.Lock()
	defer ms.Sprites.Unlock()

	sp, ok := ms.Sprites.SpriteByNameLocked(spnm)
	if ok {
		if on {
			sp.EventBBox.Min = ed.charStartPos(ed.CursorPos).ToPointFloor()
			sp.Active = true
			sp.Properties["turnOn"] = true
			sp.Properties["on"] = true
			sp.Properties["lastSwitch"] = time.Now()
		} else {
			sp.Active = false
		}
		return
	}
	if !on {
		return
	}
	bbsz := math32.Vec2(math32.Ceil(ed.CursorWidth.Dots), math32.Ceil(ed.charSize.Y))
	if bbsz.X < 2 { // at least 2
		bbsz.X = 2
	}
	sp = core.NewSprite(spnm, func(pc *paint.Painter) {
		if !sp.Active {
			return
		}
		turnOn := sp.Properties["turnOn"].(bool) // force on
		if !turnOn {
			isOn := sp.Properties["on"].(bool)
			lastSwitch := sp.Properties["lastSwitch"].(time.Time)
			if time.Since(lastSwitch) > core.SystemSettings.CursorBlinkTime {
				isOn = !isOn
				sp.Properties["on"] = isOn
				sp.Properties["lastSwitch"] = time.Now()
			}
			if !isOn {
				return
			}
		}
		sp.Properties["turnOn"] = false
		pc.Fill.Color = nil
		pc.Stroke.Color = ed.CursorColor
		pc.Stroke.Width.Dot(bbsz.X)
		pos := math32.FromPoint(sp.EventBBox.Min)
		pc.Line(pos.X, pos.Y, pos.X, pos.Y+bbsz.Y)
		pc.Draw()
	})
	sp.EventBBox.Min = ed.charStartPos(ed.CursorPos).ToPointFloor()
	sp.Active = true
	sp.InitProperties()
	sp.Properties["turnOn"] = false
	sp.Properties["on"] = true
	sp.Properties["lastSwitch"] = time.Now()
	ms.Sprites.AddLocked(sp)
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
