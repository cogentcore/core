// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/abilities"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// positionTest makes a relative parent 300 x 200 dp with a 100 x 50 static
// child, and calls fun with the parent before laying it out.
func positionTest(t *testing.T, fun func(parent *Frame, static *Frame), check func(parent, static *Frame)) {
	b := NewBody()
	parent := NewFrame(b)
	parent.Styler(func(s *styles.Style) {
		s.Position = styles.PositionRelative
		s.Min.Set(units.Dp(300), units.Dp(200))
		s.Max.Set(units.Dp(300), units.Dp(200))
		s.Padding.Zero()
		s.Gap.Zero()
	})
	static := fixedFrame(parent, 100, 50)
	fun(parent, static)
	layoutGeom(b, func() { check(parent, static) })
}

func TestPositionAbsolute(t *testing.T) {
	var abs *Frame
	positionTest(t, func(parent, static *Frame) {
		abs = fixedFrame(parent, 40, 30)
		abs.Styler(func(s *styles.Style) {
			s.Position = styles.PositionAbsolute
			s.Inset.Top, s.Inset.Right = units.Dp(10), units.Dp(20)
			s.InsetSet.Top, s.InsetSet.Right = true, true
		})
		NewFrame(parent) // after the absolute one
	}, func(parent, static *Frame) {
		dp := parent.Styles.UnitContext.Dots(units.UnitDp)
		// it takes no space in the layout
		assert.InDelta(t, parent.Geom.Pos.Content.X+static.Geom.Size.Actual.Total.X,
			parent.Child(2).(Widget).AsWidget().Geom.Pos.Total.X, 1)
		// it is at the top right of its containing block
		pr := parent.Geom.Pos.Total.X + parent.Geom.Size.Actual.Total.X
		assert.InDelta(t, pr-20*dp, abs.Geom.Pos.Total.X+abs.Geom.Size.Actual.Total.X, 1.5)
		assert.InDelta(t, parent.Geom.Pos.Total.Y+10*dp, abs.Geom.Pos.Total.Y, 1.5)
	})
}

func TestPositionAbsoluteStretch(t *testing.T) {
	var abs *Frame
	positionTest(t, func(parent, static *Frame) {
		abs = NewFrame(parent)
		abs.Styler(func(s *styles.Style) {
			s.Position = styles.PositionAbsolute
			s.Inset.Set(units.Dp(10))
			s.InsetSet.Set(true)
		})
	}, func(parent, static *Frame) {
		dp := parent.Styles.UnitContext.Dots(units.UnitDp)
		assert.InDelta(t, 280*dp, abs.Geom.Size.Actual.Total.X, 2)
		assert.InDelta(t, 180*dp, abs.Geom.Size.Actual.Total.Y, 2)
	})
}

func TestPositionRelative(t *testing.T) {
	var rel, after *Frame
	positionTest(t, func(parent, static *Frame) {
		rel = fixedFrame(parent, 40, 30)
		rel.Styler(func(s *styles.Style) {
			s.Position = styles.PositionRelative
			s.Inset.Left, s.Inset.Top = units.Dp(5), units.Dp(7)
			s.InsetSet.Left, s.InsetSet.Top = true, true
		})
		after = fixedFrame(parent, 40, 30)
	}, func(parent, static *Frame) {
		dp := parent.Styles.UnitContext.Dots(units.UnitDp)
		// it is offset from its place in the layout, which the next one is after
		assert.InDelta(t, after.Geom.Pos.Total.X-rel.Geom.Size.Actual.Total.X+5*dp, rel.Geom.Pos.Total.X, 1.5)
		assert.InDelta(t, after.Geom.Pos.Total.Y+7*dp, rel.Geom.Pos.Total.Y, 1.5)
		// the positioned child is rendered last
		order := parent.renderOrder()
		assert.Equal(t, Widget(rel), order[len(order)-1])
	})
}

func TestPositionRenderOrder(t *testing.T) {
	b := NewBody()
	a := NewFrame(b)
	a.Styler(func(s *styles.Style) { s.Position = styles.PositionRelative; s.ZIndex = 2 })
	c := NewFrame(b)
	c.Styler(func(s *styles.Style) { s.Position = styles.PositionRelative; s.ZIndex = 1 })
	d := NewFrame(b)
	e := NewFrame(b)
	e.Styler(func(s *styles.Style) { s.Position = styles.PositionAbsolute; s.ZIndex = -1 })
	layoutGeom(b, func() {
		assert.Equal(t, []Widget{e, d, c, a}, b.renderOrder())
	})
}

// TestPositionEvents tests that a positioned widget that is rendered over
// a widget after it in the tree gets the events at their overlap.
func TestPositionEvents(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Position = styles.PositionRelative })
	over := fixedFrame(b, 100, 100)
	over.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.Inset.Top, s.Inset.Left = units.Dp(0), units.Dp(0)
		s.InsetSet.Top, s.InsetSet.Left = true, true
		s.SetAbilities(true, abilities.Clickable)
	})
	under := fixedFrame(b, 300, 300)
	under.Styler(func(s *styles.Style) { s.SetAbilities(true, abilities.Clickable) })
	var clicked Widget
	over.OnClick(func(e events.Event) { clicked = over })
	under.OnClick(func(e events.Event) { clicked = under })
	b.AssertRender(t, "position/events", func() {
		pos := over.Geom.TotalBBox.Min.Add(image.Pt(10, 10))
		b.Scene.RenderWindow().SystemWindow.Events().MouseButton(events.MouseDown, events.Left, pos, 0)
		b.Scene.RenderWindow().SystemWindow.Events().MouseButton(events.MouseUp, events.Left, pos, 0)
	}, func() {
		assert.Equal(t, Widget(over), clicked)
	})
}
