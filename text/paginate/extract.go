// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"cogentcore.org/core/base/stack"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
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
		// todo: left margin
		if cp.i == 0 {
			gap.Y = 0
		}
		cw := widg(cpw.Child(cp.i))
		if fr, ok := cw.This.(*core.Frame); ok {
			if fr.Styles.Direction == styles.Column {
				if fr.Property("paginate-block") == nil {
					pars.Push(&posn{fr.This.(core.Widget), 0})
					continue
				}
			}
		}
		its = append(its, &item{w: cw.This.(core.Widget), gap: gap})
		next()
		if atEnd {
			break
		}
	}
	// fmt.Println("its:", its)
	return its
}
