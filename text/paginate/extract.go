// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"strings"

	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/printer"
	"cogentcore.org/core/tree"
)

// item is one layout item
type item struct {
	w    core.Widget
	gap  math32.Vector2 // gap to add before this element
	left float32        // left-side margin from parent frame
}

func (it *item) String() string {
	return core.AsWidget(it.w).String()
}

// extract returns the widget chunks to actually paginate.
func (p *pager) extract() []*item {
	widg := core.AsWidget

	ii := 0
	type posn struct {
		w core.Widget
		i int
	}

	pars := stack.Stack[*posn]{} // stack of parents that we are iterating
	pars.Push(&posn{p.ins[ii], 0})
	atEnd := false

	next := func() {
	start:
		cp := pars.Peek()
		cp.i++
		if cp.i >= cp.w.AsWidget().NumChildren() {
			pars.Pop()
			if len(pars) == 0 {
				ii++
				if ii >= len(p.ins) {
					atEnd = true
					return
				}
				pars.Push(&posn{p.ins[ii], 0})
				return
			} else {
				goto start
			}
		}
	}

	// The blocks are spaced by half of the font size of the text,
	// as by the default gap of frames, which is not in the browser
	// style of htmlcore, whose block margins are dropped for this
	// (see below).
	fs := printer.Settings.FontSize
	blockGap := math32.Floor(0.5 * fs.ToDots(&p.ctx))

	var its []*item
	for {
		cp := pars.Peek()
		cpw := cp.w.AsWidget()
		if cp.i >= cpw.NumChildren() {
			next()
			if atEnd {
				break
			}
			continue
		}
		gap := cpw.Styles.Gap.Dots().Floor()
		left := cpw.Styles.Padding.Left.Dots
		if cp.i == 0 {
			gap.Y = 0
		}
		gap.Y = max(gap.Y, blockGap)
		cw := widg(cpw.Child(cp.i))
		if fr, ok := cw.This.(*core.Frame); ok {
			if fr.Styles.Direction == styles.Column && hasBlocks(fr) {
				if fr.Property("paginate-block") == nil {
					pars.Push(&posn{fr.This.(core.Widget), 0})
					continue
				}
			}
			// a block kept whole is spaced by the gap
			fr.Styler(func(s *styles.Style) {
				s.Margin.Top.Zero()
				s.Margin.Bottom.Zero()
			})
		}
		its = append(its, &item{w: cw.This.(core.Widget), gap: gap, left: left})
		next()
		if atEnd {
			break
		}
	}
	// fmt.Println("its:", its)
	return its
}

// hasBlocks returns whether the frame has other frames in it, other than
// the inline containers of its text, into which it is divided for the
// pages. A frame with just text in it, as of a paragraph or heading in
// the browser style of htmlcore, is kept whole, with its margins (the
// spacing of the blocks), and the styles that its text inherits from it.
func hasBlocks(fr *core.Frame) bool {
	blocks := false
	fr.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
		if _, ok := cw.(*core.Frame); ok && !strings.HasPrefix(cwb.Name, "inline-container") {
			blocks = true
			return tree.Break
		}
		return tree.Continue
	})
	return blocks
}
