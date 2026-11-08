// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/abilities"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
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

// TestPositionSticky tests a sticky widget at the top of a section of a
// frame that scrolls: it stays at its inset from the top of the visible
// part of the frame while the section is scrolled past it, and then goes
// with the end of the section.
func TestPositionSticky(t *testing.T) {
	b := NewBody()
	scroller := NewFrame(b)
	scroller.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Overflow.Set(styles.OverflowAuto)
		s.Min.Set(units.Dp(200), units.Dp(200))
		s.Max.Set(units.Dp(200), units.Dp(200))
		s.Padding.Zero()
		s.Gap.Zero()
	})
	section := NewFrame(scroller)
	section.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Padding.Zero()
		s.Gap.Zero()
	})
	sticky := fixedFrame(section, 100, 30)
	sticky.Styler(func(s *styles.Style) {
		s.Position = styles.PositionSticky
		s.Inset.Top = units.Dp(5)
		s.InsetSet.Top = true
	})
	fixedFrame(section, 100, 500)
	fixedFrame(scroller, 100, 1000) // after the section
	b.RenderImage(func() {
		dp := scroller.Styles.UnitContext.Dots(units.UnitDp)
		top := scroller.Geom.Pos.Content.Y
		assert.InDelta(t, top+5*dp, sticky.Geom.Pos.Total.Y, 1.5, "at its inset, below its place in the layout")
		scroller.ScrollDimToStart(math32.Y, int(top+200*dp))
		scroller.ApplyScenePos()
		assert.InDelta(t, top+5*dp, sticky.Geom.Pos.Total.Y, 1.5, "at its inset from the top")
		scroller.ScrollDimToStart(math32.Y, int(top+(530-10)*dp))
		scroller.ApplyScenePos()
		sectionEnd := section.Geom.Pos.Total.Y + section.Geom.Size.Actual.Total.Y
		assert.InDelta(t, sectionEnd-30*dp, sticky.Geom.Pos.Total.Y, 1.5, "within its section")
	})
}

// TestPositionRaised tests an absolute widget with a positive z-index in a
// widget that is not a stacking context (as the dropdown menus in the
// header of Wikipedia): it is rendered over the widgets after the one that
// it is in, as in CSS, and it gets the events at their overlap.
func TestPositionRaised(t *testing.T) {
	b := NewBody()
	header := NewFrame(b)
	header.Styler(func(s *styles.Style) { s.Padding.Zero(); s.NoClip = true })
	wrap := fixedFrame(header, 50, 20)
	wrap.Styler(func(s *styles.Style) { s.Position = styles.PositionRelative; s.NoClip = true })
	menu := fixedFrame(wrap, 100, 100)
	menu.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.ZIndex = 5
		s.Inset.Top, s.Inset.Left = units.Ph(100), units.Dp(0)
		s.InsetSet.Top, s.InsetSet.Left = true, true
		s.Background = colors.Uniform(colors.Red)
		s.SetAbilities(true, abilities.Clickable)
	})
	after := fixedFrame(b, 300, 300)
	after.Styler(func(s *styles.Style) {
		s.Background = colors.Uniform(colors.Blue)
		s.SetAbilities(true, abilities.Clickable)
	})
	var clicked Widget
	menu.OnClick(func(e events.Event) { clicked = menu })
	after.OnClick(func(e events.Event) { clicked = after })
	var pos image.Point
	img := b.RenderImage(func() {
		assert.Equal(t, b.Scene.AsWidget(), menu.raisedTo(), "raised to the scene")
		pos = menu.Geom.TotalBBox.Max.Sub(image.Pt(5, 5))
		assert.True(t, pos.In(after.Geom.TotalBBox), "over the widget after")
		b.Scene.RenderWindow().SystemWindow.Events().MouseButton(events.MouseDown, events.Left, pos, 0)
		b.Scene.RenderWindow().SystemWindow.Events().MouseButton(events.MouseUp, events.Left, pos, 0)
	}, func() {
		assert.Equal(t, Widget(menu), clicked)
	})
	r, g, bl, _ := img.At(pos.X, pos.Y).RGBA()
	assert.Equal(t, [3]uint32{0xffff, 0, 0}, [3]uint32{r, g, bl}, "rendered over the widget after")
}

// TestPositionHiddenEvents tests that a raised widget that is not rendered
// (opacity 0, with no abilities, as for visibility: hidden in CSS) does
// not get the events at its overlap with a widget after it, nor do the
// clickable widgets in it, as the items of the closed menus of Wikipedia.
func TestPositionHiddenEvents(t *testing.T) {
	b := NewBody()
	header := NewFrame(b)
	header.Styler(func(s *styles.Style) { s.Padding.Zero(); s.NoClip = true })
	wrap := fixedFrame(header, 50, 20)
	wrap.Styler(func(s *styles.Style) { s.Position = styles.PositionRelative; s.NoClip = true })
	menu := fixedFrame(wrap, 100, 100)
	menu.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.ZIndex = 5
		s.Inset.Top, s.Inset.Left = units.Ph(100), units.Dp(0)
		s.InsetSet.Top, s.InsetSet.Left = true, true
		s.Opacity = 0
		s.NoClip = true
	})
	menu.FinalStyler(func(s *styles.Style) { s.Abilities = 0 })
	item := fixedFrame(menu, 100, 100)
	item.Styler(func(s *styles.Style) { s.SetAbilities(true, abilities.Clickable) })
	after := fixedFrame(b, 300, 300)
	after.Styler(func(s *styles.Style) { s.SetAbilities(true, abilities.Clickable) })
	var clicked Widget
	item.OnClick(func(e events.Event) { clicked = item })
	after.OnClick(func(e events.Event) { clicked = after })
	b.RenderImage(func() {
		pos := menu.Geom.TotalBBox.Max.Sub(image.Pt(5, 5))
		assert.True(t, pos.In(after.Geom.TotalBBox))
		b.Scene.RenderWindow().SystemWindow.Events().MouseButton(events.MouseDown, events.Left, pos, 0)
		b.Scene.RenderWindow().SystemWindow.Events().MouseButton(events.MouseUp, events.Left, pos, 0)
	}, func() {
		assert.Equal(t, Widget(after), clicked)
	})
}

// TestPositionCoveredEvents tests that a widget under a raised widget that
// has no abilities (as the background of an open menu of Wikipedia, over a
// button of the page) does not get the hover nor the clicks over it, as in
// CSS, where only the element on top and those that it is in get them.
func TestPositionCoveredEvents(t *testing.T) {
	b := NewBody()
	header := NewFrame(b)
	header.Styler(func(s *styles.Style) { s.Padding.Zero(); s.NoClip = true })
	wrap := fixedFrame(header, 50, 20)
	wrap.Styler(func(s *styles.Style) { s.Position = styles.PositionRelative; s.NoClip = true })
	menu := fixedFrame(wrap, 100, 100)
	menu.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.ZIndex = 5
		s.Inset.Top, s.Inset.Left = units.Ph(100), units.Dp(0)
		s.InsetSet.Top, s.InsetSet.Left = true, true
		s.Background = colors.Uniform(colors.White)
	})
	menu.FinalStyler(func(s *styles.Style) { s.Abilities = 0 })
	under := fixedFrame(b, 300, 300)
	under.Styler(func(s *styles.Style) { s.SetAbilities(true, abilities.Clickable, abilities.Hoverable) })
	clicked := false
	under.OnClick(func(e events.Event) { clicked = true })
	var pos image.Point
	b.RenderImage(func() {
		pos = menu.Geom.TotalBBox.Max.Sub(image.Pt(5, 5))
		assert.True(t, pos.In(under.Geom.TotalBBox), "over the widget under it")
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseMove(pos.Sub(image.Pt(2, 2)))
		ev.MouseMove(pos)
		ev.MouseButton(events.MouseDown, events.Left, pos, 0)
		ev.MouseButton(events.MouseUp, events.Left, pos, 0)
	}, func() {
		assert.False(t, under.StateIs(states.Hovered), "not hovered under the menu")
		assert.False(t, clicked, "not clicked under the menu")
		// and outside of the menu, it is
		out := under.Geom.TotalBBox.Max.Sub(image.Pt(5, 5))
		b.Scene.RenderWindow().SystemWindow.Events().MouseMove(out)
	}, func() {
		assert.True(t, under.StateIs(states.Hovered), "hovered outside of the menu")
	})
}

// TestPositionTransparentContent tests that the widgets in an absolute
// widget without a background do not paint the background of their
// parents over the widgets that they overlap (as the text over the green
// fill of the percentage bars of Wikipedia).
func TestPositionTransparentContent(t *testing.T) {
	b := NewBody()
	b.Styler(func(s *styles.Style) { s.Position = styles.PositionRelative })
	fill := fixedFrame(b, 200, 40)
	fill.Styler(func(s *styles.Style) { s.Background = colors.Uniform(colors.Green) })
	over := NewFrame(b)
	over.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.Inset.Top, s.Inset.Left = units.Dp(0), units.Dp(0)
		s.InsetSet.Top, s.InsetSet.Left = true, true
		s.Min.Set(units.Dp(200), units.Dp(40))
		s.Padding.Zero()
	})
	tx := NewText(over).SetText("  ")
	tx.Styler(func(s *styles.Style) { s.Min.Set(units.Dp(200), units.Dp(40)) })
	var pos image.Point
	img := b.RenderImage(func() { pos = tx.Geom.TotalBBox.Min.Add(image.Pt(5, 5)) })
	r, g, bl, _ := img.At(pos.X, pos.Y).RGBA()
	assert.Equal(t, [3]uint32{0, 0x80, 0}, [3]uint32{r >> 8, g >> 8, bl >> 8}, "the fill under the text")
}

// TestPositionPercentOfContainingBlock tests that the percentages of an
// absolute widget (as of its margins) are of its containing block, as in
// CSS, not of its parent (as the dot under the bookmark icon of
// Wikipedia, with a margin of 50% of its item, which was a long bar).
func TestPositionPercentOfContainingBlock(t *testing.T) {
	b := NewBody()
	item := fixedFrame(b, 40, 40)
	item.Styler(func(s *styles.Style) { s.Position = styles.PositionRelative; s.NoClip = true })
	wide := NewFrame(item) // a static parent, wider than the item
	wide.Styler(func(s *styles.Style) { s.Min.X = units.Dp(400); s.NoClip = true })
	dot := fixedFrame(wide, 12, 12)
	dot.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.Inset.Top, s.Inset.Left = units.Dp(0), units.Dp(0)
		s.InsetSet.Top, s.InsetSet.Left = true, true
		s.Margin.Left = units.Pw(50)
	})
	layoutGeom(b, func() {
		dp := dot.Styles.UnitContext.Dots(units.UnitDp)
		assert.InDelta(t, 20*dp, dot.Styles.Margin.Left.Dots, 1.5, "50% of the item")
		assert.Less(t, dot.Geom.Size.Actual.Total.X, 40*dp)
	})
}

// TestPositionNegativeZIndex tests that an absolute widget with a negative
// z-index is rendered under the other children of its parent (as the
// offset shadows of the screenshots of github.com, under their surface).
func TestPositionNegativeZIndex(t *testing.T) {
	b := NewBody()
	box := NewFrame(b)
	box.Styler(func(s *styles.Style) {
		s.Position = styles.PositionRelative
		s.Padding.Zero()
		s.NoClip = true
	})
	shadow := NewFrame(box)
	shadow.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.ZIndex = -1
		s.Inset.Top, s.Inset.Left = units.Dp(4), units.Dp(-4)
		s.InsetSet.Top, s.InsetSet.Left = true, true
		s.Min.Set(units.Dp(100), units.Dp(100))
		s.Background = colors.Uniform(colors.Green)
	})
	surface := fixedFrame(box, 100, 100)
	surface.Styler(func(s *styles.Style) { s.Background = colors.Uniform(colors.White) })
	b.Styler(func(s *styles.Style) { s.NoClip = true })
	fixedFrame(b, 200, 50) // room below the box for its shadow
	img := b.RenderImage()
	c := surface.Geom.TotalBBox.Min.Add(surface.Geom.TotalBBox.Size().Div(2))
	r, g, bl, _ := img.At(c.X, c.Y).RGBA()
	assert.Equal(t, [3]uint32{0xffff, 0xffff, 0xffff}, [3]uint32{r, g, bl}, "the surface over the shadow")
	r, g, _, _ = img.At(c.X, surface.Geom.TotalBBox.Max.Y+1).RGBA()
	assert.Equal(t, uint32(0), r, "the shadow below the surface")
	assert.Greater(t, g, uint32(0))
}

// TestPositionStatic tests that an absolute widget without a top (or
// left) inset is at its static position in those dimensions: below the
// widget before it in a column (as the dropdown panels of the menus of
// MDN, below their buttons, which they covered).
func TestPositionStatic(t *testing.T) {
	b := NewBody()
	col := NewFrame(b)
	col.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Position = styles.PositionRelative
		s.NoClip = true
	})
	button := fixedFrame(col, 80, 40)
	panel := fixedFrame(col, 200, 100)
	panel.Styler(func(s *styles.Style) {
		s.Position = styles.PositionAbsolute
		s.Inset.Left = units.Dp(0)
		s.InsetSet.Left = true
	})
	b.RenderImage(func() {
		bb, pb := button.Geom.TotalBBox, panel.Geom.Pos.Total
		assert.InDelta(t, float32(bb.Max.Y), pb.Y, 1, "below the button")
		assert.InDelta(t, col.Geom.Pos.Content.X, pb.X, 1, "at the left of its containing block")
	})
}
