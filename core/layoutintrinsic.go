// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"slices"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// This file has the intrinsic widths of the layout, as in CSS: the
// min-content width ([GeomSize.minWidth]) and the max-content (preferred)
// width ([GeomSize.prefWidth]) of each widget, computed in SizeUp. Widths
// are intrinsic and heights follow: a widget starts at its preferred width,
// within the width allocated to it, is narrowed toward its minimum only
// when there is too little width, and its height is that of its content
// at the width that it is finally given.

// sizeUpIntrinsicFrame sets the intrinsic widths of the frame from those
// of its children, as they combine for its layout: a row sums them, a
// column takes the largest, a wrapped row takes the largest minimum and
// the sum of the preferred widths, and a grid does each per column.
func (fr *Frame) sizeUpIntrinsicFrame() {
	minw, prefw := fr.childrenIntrinsicWidths()
	fr.layout.minContent = minw
	sz := &fr.Geom.Size
	if fr.Styles.Overflow.X >= styles.OverflowAuto {
		minw = 0 // its content scrolls
	}
	if fr.Styles.ContentMinWidth && sz.Max.X > 0 && minw > sz.Max.X {
		// as wide as its content needs to be (as a table in CSS)
		sz.Max.X = math32.Ceil(minw)
		sz.Actual.Content.X = max(sz.Actual.Content.X, sz.Max.X)
		sz.SetTotalFromContent(&sz.Actual)
	}
	sz.setIntrinsicWidths(minw, prefw)
}

// childrenIntrinsicWidths returns the min-content and max-content widths
// of the content of the frame, from its children in the flow of its layout.
func (fr *Frame) childrenIntrinsicWidths() (minw, prefw float32) {
	s := &fr.Styles
	gap := fr.layout.Gap.X
	switch {
	case s.Display == styles.Stacked:
		if kwb := fr.StackTopWidget(); kwb != nil {
			return kwb.Geom.Size.minWidth, kwb.Geom.Size.prefWidth
		}
		return 0, 0
	case s.Display == styles.Grid:
		return fr.gridIntrinsicWidths()
	case s.Display != styles.Flex:
		w := fr.Geom.Size.Actual.Content.X
		return w, w
	}
	row := s.Direction == styles.Row
	n := 0
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ksz := &cwb.Geom.Size
		switch {
		case row && !s.Wrap:
			minw += ksz.minWidth
			prefw += ksz.prefWidth
		case row: // wrapped: one item per line at its narrowest
			minw = max(minw, ksz.minWidth)
			prefw += ksz.prefWidth
		default:
			minw = max(minw, ksz.minWidth)
			prefw = max(prefw, ksz.prefWidth)
		}
		n++
		return tree.Continue
	})
	if row && n > 1 {
		prefw += float32(n-1) * gap
		if !s.Wrap {
			minw += float32(n-1) * gap
		}
	}
	if fl := fr.layout.floats; fl != nil {
		minw = max(minw, fl.size.X)
		prefw = max(prefw, fl.size.X)
	}
	if fr.onlyFloats() {
		// A column that only has floats is as wide as the widest one at
		// least, and has them side by side when there is room for all of
		// them (as in CSS, in which the max-content width of a block has
		// its floats on one line), as for a float of floats.
		var fmin, fpref float32
		fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
			if cwb.isFloat() {
				fmin = max(fmin, cwb.Geom.Size.minWidth)
				fpref += cwb.Geom.Size.prefWidth
			}
			return tree.Continue
		})
		minw = max(minw, fmin)
		prefw = max(prefw, fpref)
	}
	return minw, prefw
}

// gridIntrinsicWidths returns the intrinsic widths of the content of a grid
// frame, as the sums of those of its columns: the largest of the cells in
// each, with the cells that span columns widening them evenly as needed.
func (fr *Frame) gridIntrinsicWidths() (minw, prefw float32) {
	li := &fr.layout
	n := li.Shape.X
	if n <= 0 {
		return 0, 0
	}
	mins := make([]float32, n)
	prefs := make([]float32, n)
	type spanning struct {
		col, span int
		minw      float32
		prefw     float32
	}
	var spans []spanning
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		c := cwb.Geom.Cell.X
		if c >= n {
			return tree.Continue
		}
		ksz := &cwb.Geom.Size
		if span := fr.colSpan(cwb); span > 1 {
			spans = append(spans, spanning{c, span, ksz.minWidth, ksz.prefWidth})
			return tree.Continue
		}
		mins[c] = max(mins[c], ksz.minWidth)
		prefs[c] = max(prefs[c], ksz.prefWidth)
		return tree.Continue
	})
	widen := func(ws []float32, col, span int, w float32) {
		sum := li.Gap.X * float32(span-1)
		for c := col; c < col+span; c++ {
			sum += ws[c]
		}
		if extra := w - sum; extra > 0 {
			for c := col; c < col+span; c++ {
				ws[c] += extra / float32(span)
			}
		}
	}
	for _, sp := range spans {
		widen(mins, sp.col, sp.span, sp.minw)
		widen(prefs, sp.col, sp.span, sp.prefw)
	}
	gaps := float32(n-1) * li.Gap.X
	minw, prefw = gaps, gaps
	for c := range n {
		minw += mins[c]
		prefw += max(prefs[c], mins[c])
	}
	return minw, prefw
}

// narrowTo returns the given allocated width of the given child, narrowed
// to the given width if that is less. In a column (block is true), a frame
// with children is narrowed to the width, as a block is in CSS, so that its
// content then overflows it, while other widgets are not narrowed below
// their min-content width, which keeps the words of text whole (the shaper
// would otherwise break them). The items of a wrapped row (block is false)
// are not narrowed below their min-content width, as flex items in CSS.
func narrowTo(cwb *WidgetBase, alloc, width float32, block bool) float32 {
	if alloc <= width {
		return alloc
	}
	if block && cwb.HasChildren() {
		if _, ok := cwb.This.(Layouter); ok {
			return width
		}
	}
	return max(width, min(alloc, cwb.Geom.Size.minWidth))
}

// overflowWidth returns the width of the content of the frame that it
// would overflow in, for deciding whether it needs to scroll in width:
// in the first SizeDown iteration, the children still have their
// preferred widths from SizeUp, which they are narrowed from to fit, so
// the frame only overflows its min-content width, plus any content that
// overflows its children visibly beyond their Max (see [Frame.sizeOverflow]);
// afterward, it is the actual width of its content, as given.
func (fr *Frame) overflowWidth(iter int, maxSize float32) float32 {
	if iter > 0 || !fr.HasChildren() || fr.Styles.Display == styles.Custom {
		return maxSize
	}
	excess := max(fr.layout.overflow.X-fr.Geom.Size.Internal.X, 0)
	return min(maxSize, fr.layout.minContent+excess)
}

// growMaxWidth returns the total width of the Max of the given child
// (with its Space), as it is (or will be) for the content width of the
// frame, which a parent-relative Max is relative to, or 0 if it has none.
func (fr *Frame) growMaxWidth(cwb *WidgetBase) float32 {
	ksz := &cwb.Geom.Size
	mx := cwb.Styles.Max.X
	w := ksz.Max.X
	switch mx.Unit {
	case units.UnitPw:
		w = (fr.Geom.Size.Alloc.Content.X - fr.Geom.Size.InnerSpace.X) * 0.01 * mx.Value
		w = max(w-cwb.boxSizingSpace().X, 1)
	case units.UnitPh:
		w = (fr.Geom.Size.Alloc.Content.Y - fr.Geom.Size.InnerSpace.Y) * 0.01 * mx.Value
		w = max(w-cwb.boxSizingSpace().X, 1)
	}
	if w <= 0 {
		return 0
	}
	return w + ksz.Space.X
}

// sizeOverflow sets the overflow size of the content of the frame
// (layoutState.overflow), and its own extent ([GeomSize.extent]), from the
// visible overflow of its children: the content that overflows a child
// (which is beyond its Actual size, as for a frame with a Max narrower
// than its content) is still shown, and the frame that scrolls around it
// scrolls over it, as for the scrollable overflow of CSS. The overflow is
// the content size plus the largest overflow of a child, in each
// dimension. A frame that clips its content (not visible overflow) has no
// overflow of its own beyond its size, and the content of a frame only
// overflows it beyond a Max, as its actual size otherwise fits it (the
// content size of wrapped layouts can be ahead of their actual size).
// The overflow of a child only counts beyond the room that it has in the
// content of the frame (see [Frame.overflowRoom]).
func (fr *Frame) sizeOverflow() {
	sz := &fr.Geom.Size
	var excess math32.Vector2
	room := fr.overflowRoom()
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		ksz := &cwb.Geom.Size
		over := ksz.extent.Sub(ksz.Actual.Total)
		if r := room(cwb); over.X > 0 || over.Y > 0 {
			over = over.Sub(r)
		}
		excess.SetMax(over)
		return tree.Continue
	})
	fr.layout.overflow = sz.Internal.Add(excess)
	sz.extent = sz.Actual.Total
	for d := math32.X; d <= math32.Y; d++ {
		// Its own content only overflows it beyond a Max (otherwise its
		// actual size fits its content), while that of its children
		// overflows it in any case.
		ov := sz.Actual.Content.Dim(d) + excess.Dim(d)
		if sz.Max.Dim(d) > 0 {
			ov = max(ov, fr.layout.overflow.Dim(d))
		}
		if fr.Styles.Overflow.Dim(d) != styles.OverflowVisible {
			continue
		}
		sz.extent.SetDim(d, max(sz.extent.Dim(d), ov+sz.Space.Dim(d)))
	}
}

// overflowRoom returns a function that returns the room that each child
// of the frame, called in order, has after it in the content of the frame,
// in each dimension, which content that overflows the child visibly can
// take without overflowing the frame. In a flex layout on one line, the
// position of a child along its axis follows from the sizes of the
// children before it, and that across it from its alignment. For a grid,
// wrapped lines, and distributed justification, the positions are not known
// when it is sized, so there is no room.
func (fr *Frame) overflowRoom() func(cwb *WidgetBase) math32.Vector2 {
	sz := &fr.Geom.Size
	jc := fr.Styles.Justify.Content
	if fr.Styles.Display != styles.Flex || fr.Styles.Wrap ||
		jc == styles.SpaceBetween || jc == styles.SpaceAround || jc == styles.SpaceEvenly {
		return func(cwb *WidgetBase) math32.Vector2 { return math32.Vector2{} }
	}
	content := sz.Actual.Content
	ma := fr.Styles.Direction.Dim()
	ca := ma.Other()
	gap := fr.layout.Gap.Dim(ma)
	pos := styles.AlignPos(jc, sz.Internal.Dim(ma), content.Dim(ma))
	return func(cwb *WidgetBase) math32.Vector2 {
		size := cwb.Geom.Size.Actual.Total
		var room math32.Vector2
		room.SetDim(ma, max(content.Dim(ma)-pos-size.Dim(ma), 0))
		align := styles.ItemAlign(fr.Styles.Align.Items, cwb.Styles.Align.Self)
		cpos := styles.AlignPos(align, size.Dim(ca), content.Dim(ca))
		room.SetDim(ca, max(content.Dim(ca)-cpos-size.Dim(ca), 0))
		pos += size.Dim(ma) + gap
		return room
	}
}

// growWidths returns the given widths of the columns of the frame (from
// [Frame.shrinkWidths], when they fit), grown into the rest of the given
// width by the columns that grow, in proportion to their Grow, up to their
// Max: the width that a column cannot take beyond its Max goes to the others.
func (fr *Frame) growWidths(widths []float32, width float32) []float32 {
	li := &fr.layout
	n := len(widths)
	grow := make([]float32, n)
	maxw := make([]float32, n) // 0 for none
	hasMax := make([]bool, n)
	for c := range n {
		grow[c] = li.cell(math32.X, c, 0).Grow.X
		hasMax[c] = true
	}
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		c := cwb.Geom.Cell.X
		if c >= n || fr.colSpan(cwb) > 1 {
			return tree.Continue
		}
		if mx := cwb.Geom.Size.maxTotal().X; mx > 0 && cwb.Geom.Size.Max.X > 0 {
			maxw[c] = max(maxw[c], mx)
		} else {
			hasMax[c] = false
		}
		return tree.Continue
	})
	left := width
	for _, w := range widths {
		left -= w
	}
	grown := slices.Clone(widths)
	for range n { // again when some reach their Max
		if left < 0.5 {
			break
		}
		gsum := float32(0)
		for c := range n {
			gsum += grow[c]
		}
		if gsum <= 0 {
			break
		}
		used := float32(0)
		for c := range n {
			if grow[c] == 0 {
				continue
			}
			nw := grown[c] + left*grow[c]/gsum
			if hasMax[c] && maxw[c] > 0 && nw >= maxw[c] {
				nw = max(grown[c], maxw[c])
				grow[c] = 0 // at its Max
			}
			used += nw - grown[c]
			grown[c] = nw
		}
		left -= used
	}
	for c := range n {
		grown[c] = math32.Floor(grown[c])
	}
	return grown
}
