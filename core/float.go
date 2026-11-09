// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// Widgets with a [styles.Style.Float] float to the left or right side
// of their parent, which is a column ([styles.Flex] with [styles.Column]
// and no wrap): they are out of the flow of its layout, at the position
// in the column where they are, and each widget after them that starts
// beside them is narrowed and moved to be beside them. (This is a
// simplification of CSS floats, in which each line of text beside a
// float is shortened instead.) [styles.Style.Clear] moves a widget below
// the floats on the given sides. The column is as tall as its floats.

// isFloat returns whether the widget floats (see [styles.Style.Float]).
func (wb *WidgetBase) isFloat() bool {
	return wb.Styles.Float != styles.FloatNone && !wb.outOfFlow()
}

// floatPlace is the place of a child in a column with floats.
type floatPlace struct {
	// pos is the position of the child relative to the content of the column.
	pos math32.Vector2

	// width is the width allocated to the child.
	width float32
}

// floatLayout is the layout of a column that has floats.
type floatLayout struct {
	// places has the place of each child, by its index.
	places map[int]floatPlace

	// size is the size of the content of the column,
	// with the floats and the space that clearing them adds.
	size math32.Vector2

	// floatsWidth is the width that the floats take, on the left and the
	// right side, which is the width of a column that only has floats
	// (see [Frame.onlyFloats]).
	floatsWidth float32
}

// hasFloats returns whether the frame is a column with floating children.
func (fr *Frame) hasFloats() bool {
	if fr.Styles.Display != styles.Flex || fr.Styles.Direction != styles.Column || fr.Styles.Wrap {
		return false
	}
	has := false
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if cwb.isFloat() {
			has = true
			return tree.Break
		}
		return tree.Continue
	})
	return has
}

// onlyFloats returns whether the frame is a column with floating children
// and no other children in the flow of its layout.
func (fr *Frame) onlyFloats() bool {
	if !fr.hasFloats() {
		return false
	}
	flow := false
	fr.forFlowChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		flow = true
		return tree.Break
	})
	return !flow
}

// layoutFloats computes the [floatLayout] of the frame from the current
// sizes of its children, sets the allocations of the children from it,
// and returns whether any of them changed.
func (fr *Frame) layoutFloats() bool {
	if !fr.hasFloats() {
		fr.layout.floats = nil
		return false
	}
	type rect struct {
		left       bool
		x, y, w, h float32
	}
	var floats []rect
	width := fr.Geom.Size.Alloc.Content.X - fr.Geom.Size.InnerSpace.X
	gap := fr.layout.Gap.Y
	fl := &floatLayout{places: map[int]floatPlace{}}
	// edges returns the edges of the space beside the floats at y.
	edges := func(y float32) (left, right float32) {
		left, right = 0, width
		for _, f := range floats {
			if y < f.y || y >= f.y+f.h {
				continue
			}
			if f.left {
				left = max(left, f.x+f.w)
			} else {
				right = min(right, f.x)
			}
		}
		return
	}
	// below returns the bottom of the floats on the given sides.
	below := func(left, right bool) float32 {
		b := float32(0)
		for _, f := range floats {
			if (f.left && left) || (!f.left && right) {
				b = max(b, f.y+f.h)
			}
		}
		return b
	}
	changed := false
	var maxLeft, maxRight float32
	y := float32(0)
	floatTop := float32(0) // the top of the last float
	// the last child in the flow, if it is inline content: its top, and its
	// preferred width (on one line)
	lastFlow, lastFlowY, lastFlowPref := false, float32(0), float32(0)
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if cwb.outOfFlow() {
			return tree.Continue
		}
		ksz := &cwb.Geom.Size
		cy := y // below the floats that it clears
		switch cwb.Styles.Clear {
		case styles.ClearLeft:
			cy = max(y, below(true, false))
		case styles.ClearRight:
			cy = max(y, below(false, true))
		case styles.ClearBoth:
			cy = max(y, below(true, true))
		}
		if cwb.isFloat() {
			// a float that clears is moved down, but not the flow after it
			w := min(ksz.Actual.Total.X, width)
			if cwb.Styles.Grow.X > 0 {
				// a growing float (as for a percentage width in CSS) takes
				// the width of the column, up to its Max
				// (a percentage Max is of the column, which it may not
				// have been resolved for yet)
				w = width
				if mx := cwb.Styles.Max.X; mx.Unit == units.UnitPw && mx.Value > 0 {
					w = min(w, width*mx.Value/100+ksz.Space.X-cwb.boxSizingSpace().X)
				} else if mt := ksz.maxTotal().X; mt > 0 {
					w = min(w, mt)
				}
			} else if AsFrame(cw) != nil {
				// A float is as wide as its content (shrink-to-fit, as in
				// CSS): its preferred width, within the column, also when
				// its content is floats (which have no size in its flow), or
				// a block of floats (as the breadcrumbs of the header of
				// craigslist.org, a list of floats in a floating nav, which
				// had no width, so its crumbs were each on a line).
				w = min(max(w, ksz.prefWidth), width)
			}
			h := ksz.Actual.Total.Y
			left := cwb.Styles.Float == styles.FloatLeft
			// a float is not higher than the floats before it, as in CSS
			fy := max(cy, floatTop)
			if !left && lastFlow && cwb.Styles.Clear == styles.ClearNone && lastFlowY >= floatTop {
				// a right float after content that is on one line is on that
				// line, beside it, if it fits there (as the theme and search
				// box of the header of the Python docs, after its links), as
				// in CSS, where it is on the line of the content before it
				if l, r := edges(lastFlowY); r-w >= l+lastFlowPref {
					fy = lastFlowY
				}
			}
			// the widths of the floats are rounded up to whole dots, by less
			// than a dot each: a float fits in a space that is narrower by
			// less than that for the floats before it (as the nine slides of
			// the photos of the postings of craigslist.org, of 442px each in
			// a row of 3978px, the last of which wrapped)
			slack := float32(len(floats))
			for range len(floats) + 1 { // down below floats until it fits
				l, r := edges(fy)
				if r-l+slack >= w || l == 0 && r == width {
					break
				}
				next := float32(-1) // the next bottom of a float
				for _, f := range floats {
					if b := f.y + f.h; b > fy && (next < 0 || b < next) {
						next = b
					}
				}
				if next < 0 {
					break
				}
				fy = next
			}
			l, r := edges(fy)
			x := l
			if !left {
				x = r - w
			}
			floats = append(floats, rect{left, x, fy, w, h})
			floatTop = fy
			if ksz.Alloc.Total.X != w {
				changed = true
			}
			ksz.Alloc.Total = math32.Vec2(w, h)
			ksz.SetContentFromTotal(&ksz.Alloc)
			if AsFrame(cw) != nil && ksz.Actual.Total.X < w {
				// the box of a float is its width (shrink-to-fit), also when
				// it is finally sized before its width is set here (as the
				// breadcrumbs of craigslist.org, a nav of a list of floats,
				// which stayed as wide as nothing)
				ksz.Actual.Total.X = w
				ksz.SetContentFromTotal(&ksz.Actual)
			}
			fl.places[i] = floatPlace{pos: math32.Vec2(x, fy), width: w}
			fl.size.Y = max(fl.size.Y, fy+h)
			if left { // the widths that the floats take on each side
				maxLeft = max(maxLeft, x+w)
			} else {
				maxRight = max(maxRight, width-x)
			}
			return tree.Continue
		}
		y = cy
		l, r := edges(y)
		w := ksz.Alloc.Total.X
		if cwb.Styles.Grow.X > 0 || w > r-l {
			w = r - l
		}
		if w != ksz.Alloc.Total.X {
			changed = true
			ksz.Alloc.Total.X = w
			ksz.SetContentFromTotal(&ksz.Alloc)
		}
		fl.places[i] = floatPlace{pos: math32.Vec2(l, y), width: w}
		// only inline content (a text, or a row of inline content that
		// wraps), not a block, which a float after it is below in CSS
		_, isText := cw.(*Text)
		inline := isText || cwb.Styles.Direction == styles.Row && cwb.Styles.Wrap
		lastFlow, lastFlowY, lastFlowPref = inline, y, ksz.prefWidth
		y += ksz.Actual.Total.Y + gap
		return tree.Continue
	})
	fl.size.Y = max(fl.size.Y, y-gap)
	fl.size.X = width
	fl.floatsWidth = maxLeft + maxRight
	fr.layout.floats = fl
	return changed
}

// positionFloats positions the children of the frame from its [floatLayout],
// returning false if it has none.
func (fr *Frame) positionFloats() bool {
	fl := fr.layout.floats
	if fl == nil {
		return false
	}
	sz := &fr.Geom.Size
	var stPos math32.Vector2
	stPos.Y = styles.AlignPos(fr.Styles.Justify.Content, fl.size.Y, sz.Actual.Content.Y)
	fr.forVisibleChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		pl, ok := fl.places[i]
		if !ok {
			return tree.Continue
		}
		pos := stPos.Add(pl.pos)
		ksz := &cwb.Geom.Size
		pos.X += styles.AlignPos(styles.ItemAlign(fr.Styles.Align.Items, cwb.Styles.Align.Self), ksz.Actual.Total.X, pl.width)
		cwb.Geom.RelPos = pos
		return tree.Continue
	})
	return true
}
