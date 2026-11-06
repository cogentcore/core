// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cmp"
	"slices"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// Widgets with a [styles.Style.Position] of [styles.PositionAbsolute] or
// [styles.PositionFixed] are out of the flow of the layout of their parent:
// they are sized and positioned like the other children, but they take no
// space in the layout of the parent, and they are positioned by their
// [styles.Style.Inset] relative to their containing block.

// outOfFlow returns whether the widget is out of the flow of the
// layout of its parent (absolute or fixed position).
func (wb *WidgetBase) outOfFlow() bool {
	p := wb.Styles.Position
	return p == styles.PositionAbsolute || p == styles.PositionFixed
}

// forFlowChildren calls the given function for the visible children
// that are in the flow of the layout (not [WidgetBase.outOfFlow],
// and not floating, [WidgetBase.isFloat]).
func (wb *WidgetBase) forFlowChildren(fun func(i int, cw Widget, cwb *WidgetBase) bool) {
	wb.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if cwb.outOfFlow() || cwb.isFloat() {
			return tree.Continue
		}
		return fun(i, cw, cwb)
	})
}

// containingBlock returns the widget that the given out of flow
// widget is positioned relative to: for absolute, the nearest widget
// that it is in that is positioned (not static), or otherwise that
// scrolls (as the document does in a browser); for fixed, the scene.
func (wb *WidgetBase) containingBlock() *WidgetBase {
	if wb.Styles.Position == styles.PositionAbsolute {
		var scroller *WidgetBase
		for p := wb.parentWidget(); p != nil; p = p.parentWidget() {
			if p.Styles.Position != styles.PositionStatic {
				return p
			}
			if scroller == nil && (p.Styles.Overflow.X >= styles.OverflowAuto || p.Styles.Overflow.Y >= styles.OverflowAuto) {
				scroller = p
			}
		}
		if scroller != nil {
			return scroller
		}
	}
	if wb.Scene != nil {
		return wb.Scene.AsWidget()
	}
	return wb.parentWidget()
}

// paddingBox returns the position offset (from Pos.Total) and the size
// of the padding box of the widget (inside its border).
func (wb *WidgetBase) paddingBox() (offset, size math32.Vector2) {
	m := wb.Styles.TotalMargin() // margin and border
	offset = math32.Vec2(m.Left, m.Top)
	size = wb.Geom.Size.Alloc.Total.Sub(math32.Vec2(m.Left+m.Right, m.Top+m.Bottom))
	return
}

// sizeDownOutOfFlow allocates the out of flow children of the frame their
// actual size, or, in a dimension in which both of the opposite sides of
// their inset are set, the space between them in their containing block.
func (fr *Frame) sizeDownOutOfFlow() {
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if !cwb.outOfFlow() {
			return tree.Continue
		}
		sz := &cwb.Geom.Size
		in := cwb.Styles.Inset.Dots()
		set := cwb.Styles.InsetSet
		_, cbsz := cwb.containingBlock().paddingBox()
		// sizes that are relative to the parent are of the containing block
		space := sz.Space
		for d := math32.X; d <= math32.Y; d++ {
			for _, v := range []units.Value{cwb.Styles.Min.Dim(d), cwb.Styles.Max.Dim(d)} {
				if v.Value > 0 && (v.Unit == units.UnitPw || v.Unit == units.UnitPh) {
					ref := cbsz.X
					if v.Unit == units.UnitPh {
						ref = cbsz.Y
					}
					sz.Actual.Total.SetDim(d, ref*v.Value/100+space.Dim(d))
					break
				}
			}
		}
		if set.Left && set.Right {
			sz.Actual.Total.X = max(cbsz.X-in.Left-in.Right, 0)
		}
		if set.Top && set.Bottom {
			sz.Actual.Total.Y = max(cbsz.Y-in.Top-in.Bottom, 0)
		}
		sz.SetContentFromTotal(&sz.Actual)
		sz.Alloc = sz.Actual
		return tree.Continue
	})
}

// setPosOutOfFlow sets the position of the widget, which is out of the flow
// of its parent, from its inset and its containing block, using the given
// position in the parent for the dimensions in which no inset is set.
func (wb *WidgetBase) setPosOutOfFlow(parPos math32.Vector2) {
	cb := wb.containingBlock()
	off, cbsz := cb.paddingBox()
	cbpos := cb.Geom.Pos.Total.Add(off).Add(cb.Geom.Scroll.Round()) // with its content
	if wb.Styles.Position == styles.PositionFixed {
		cbpos = off // relative to the scene, which does not scroll
	}
	in := wb.Styles.Inset.Dots()
	set := wb.Styles.InsetSet
	sz := wb.Geom.Size.Actual.Total
	pos := wb.Geom.RelPos.Add(parPos) // the static position
	switch {
	case set.Left:
		pos.X = cbpos.X + in.Left
	case set.Right:
		pos.X = cbpos.X + cbsz.X - in.Right - sz.X
	}
	switch {
	case set.Top:
		pos.Y = cbpos.Y + in.Top
	case set.Bottom:
		pos.Y = cbpos.Y + cbsz.Y - in.Bottom - sz.Y
	}
	wb.Geom.Pos.Total = pos
}

// relativeOffset returns the offset of a widget with a relative position
// from its position in the layout.
func (wb *WidgetBase) relativeOffset() math32.Vector2 {
	var off math32.Vector2
	in := wb.Styles.Inset.Dots()
	set := wb.Styles.InsetSet
	switch {
	case set.Left:
		off.X = in.Left
	case set.Right:
		off.X = -in.Right
	}
	switch {
	case set.Top:
		off.Y = in.Top
	case set.Bottom:
		off.Y = -in.Bottom
	}
	return off
}

// translateOffset returns the offset of the [styles.Style.Translate] of the
// widget in dots, with percentages of the element width and height of its
// own size.
func (wb *WidgetBase) translateOffset() math32.Vector2 {
	tr := &wb.Styles.Translate
	if tr.X.Value == 0 && tr.Y.Value == 0 {
		return math32.Vector2{}
	}
	sz := wb.Geom.Size.Actual.Total
	dots := func(v units.Value, size float32) float32 {
		switch v.Unit {
		case units.UnitEw, units.UnitEh:
			return v.Value / 100 * size
		}
		return wb.Styles.UnitContext.ToDots(v.Value, v.Unit)
	}
	return math32.Vec2(dots(tr.X, sz.X), dots(tr.Y, sz.Y))
}

// renderOrder returns the children of the widget in the order that they are
// rendered: those that float after the others (over their backgrounds),
// and those that are positioned (not static) after them,
// in the increasing order of their ZIndex, except that those with a negative
// ZIndex are first. It returns nil if they are rendered in order.
func (wb *WidgetBase) renderOrder() []Widget {
	positioned := false
	wb.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if cwb.Styles.Position != styles.PositionStatic || cwb.isFloat() {
			positioned = true
			return tree.Break
		}
		return tree.Continue
	})
	if !positioned {
		return nil
	}
	type item struct {
		w     Widget
		layer int // -1: negative z-index; 0: static; 1: float; 2: positioned
		z     int
	}
	var items []item
	wb.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		it := item{w: cw}
		switch {
		case cwb.Styles.Position != styles.PositionStatic:
			it.layer, it.z = 2, cwb.Styles.ZIndex
			if it.z < 0 {
				it.layer = -1
			}
		case cwb.isFloat():
			it.layer = 1
		}
		items = append(items, it)
		return tree.Continue
	})
	slices.SortStableFunc(items, func(a, b item) int {
		if a.layer != b.layer {
			return cmp.Compare(a.layer, b.layer)
		}
		return cmp.Compare(a.z, b.z)
	})
	ws := make([]Widget, len(items))
	for i, it := range items {
		ws[i] = it.w
	}
	return ws
}
