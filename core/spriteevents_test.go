// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// TestSpriteMouseEvents tests the events of a sprite from the mouse:
// entering and leaving it, clicking it, and sliding it.
func TestSpriteMouseEvents(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(200)) })
	got := map[events.Types]int{}
	count := func(e events.Event) { got[e.Type()]++ }
	sp := NewSprite("handle", nil)
	sp.EventBBox = image.Rect(20, 20, 40, 40)
	sp.OnClick(count).OnSlideStart(count).OnSlideMove(count).OnSlideStop(count)
	sp.On(events.MouseEnter, count).On(events.MouseLeave, count)
	in, out := image.Pt(30, 30), image.Pt(100, 100)
	mouse := func(typ events.Types, p image.Point) {
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		if typ == events.MouseMove {
			ev.MouseMove(p)
		} else {
			ev.MouseButton(typ, events.Left, p, 0)
		}
		time.Sleep(20 * time.Millisecond)
	}
	RenderImage(b, Locked(b, func() {
		ss := &b.Scene.Stage.Main.Sprites
		sp.Active = true
		ss.Add(sp)
	}), func() {
		mouse(events.MouseMove, in)
		mouse(events.MouseMove, out)
		mouse(events.MouseDown, in)
		mouse(events.MouseUp, in)
		time.Sleep(TimingSettings.DoubleClickInterval)
		DragMouse(b, in, out)
	}, Locked(b, func() {
		assert.Positive(t, got[events.MouseEnter], "entered")
		assert.Positive(t, got[events.MouseLeave], "left")
		assert.Positive(t, got[events.Click], "clicked")
		// a sprite slides from the press, for the click as well as the drag
		assert.Equal(t, 2, got[events.SlideStart])
		assert.Positive(t, got[events.SlideMove])
		assert.Equal(t, 2, got[events.SlideStop])
	}))
}
