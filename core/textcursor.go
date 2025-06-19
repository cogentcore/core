// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"time"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

// TextCursor implements a blinking text cursor using a [Sprite].
//   - on determines whether the cursor should be turned on or off,
//   - w is the text widget the cursor belongs to
//   - lastW is a pointer to a [tree.Node] holding the last text object to create
//     a sprite.
//   - name is the unique name of the cursor (shared among all of same type)
//   - width, height are size of cursor line to render
//   - color is the color to render.
//   - pos is a function that returns the current top-left position of cursor
func TextCursor(on bool, w *WidgetBase, lastW *tree.Node, name string, width, height float32, color image.Image, pos func() image.Point) {
	sc := w.Scene
	if sc == nil || sc.Stage == nil || sc.Stage.Main == nil {
		return
	}
	ms := sc.Stage.Main
	ms.Sprites.Lock()
	defer ms.Sprites.Unlock()

	activate := func(sp *Sprite) {
		sp.EventBBox.Min = pos()
		sp.Active = true
		sp.Properties["turnOn"] = true
		sp.Properties["on"] = true
		sp.Properties["lastSwitch"] = time.Now()
	}

	if !on || *lastW == w.This {
		if sp, ok := ms.Sprites.SpriteByNameNoLock(name); ok {
			if on {
				activate(sp)
			} else {
				sp.Active = false
			}
			return
		}
		if !on {
			*lastW = nil
			return
		}
	}
	var sp *Sprite
	sp = NewSprite(name, func(pc *paint.Painter) {
		if !sp.Active || w == nil || w.This == nil || !w.IsVisible() {
			return
		}
		turnOn := sp.Properties["turnOn"].(bool) // force on
		if !turnOn {
			isOn := sp.Properties["on"].(bool)
			lastSwitch := sp.Properties["lastSwitch"].(time.Time)
			if TheApp.Platform() != system.Offscreen && SystemSettings.CursorBlinkTime > 0 && time.Since(lastSwitch) > SystemSettings.CursorBlinkTime {
				isOn = !isOn
				sp.Properties["on"] = isOn
				sp.Properties["lastSwitch"] = time.Now()
			}
			if !isOn {
				return
			}
		}
		bbsz := math32.Vec2(math32.Ceil(width), math32.Ceil(height))
		if bbsz.X < 2 { // at least 2
			bbsz.X = 2
		}
		sp.Properties["turnOn"] = false
		pos := math32.FromPoint(sp.EventBBox.Min)
		if !pos.AddScalar(3).ToPointCeil().Sub(sc.SceneGeom.Pos).In(w.Geom.ContentBBox) {
			return
		}
		pc.Fill.Color = nil
		pc.Stroke.Color = color
		pc.Stroke.Width.Dot(bbsz.X)
		pc.Line(pos.X, pos.Y, pos.X, pos.Y+bbsz.Y)
		pc.Draw()
	})
	sp.InitProperties()
	activate(sp)
	ms.Sprites.AddNoLock(sp)
	*lastW = w.This
}
