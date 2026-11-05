// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
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
// the frame only overflows its min-content width; afterward, it is the
// actual width of its content, as given.
func (fr *Frame) overflowWidth(iter int, maxSize float32) float32 {
	if iter > 0 || !fr.HasChildren() || fr.Styles.Display == styles.Custom {
		return maxSize
	}
	return min(maxSize, fr.layout.minContent)
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
