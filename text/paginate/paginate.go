// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

// Paginate organizes the given input widget content into frames
// that each fit within the page size specified in the options.
func Paginate(opts Options, ins ...core.Widget) []*core.Frame {
	if len(ins) == 0 {
		return nil
	}
	p := pager{opts: &opts, ins: ins}
	p.paginate()
	return p.outs
}

// pager implements the pagination.
type pager struct {
	opts *Options
	ins  []core.Widget
	outs []*core.Frame

	ctx units.Context
}

func (p *pager) paginate() {
	p.opts.Update()
	p.ctx = p.ins[0].(core.Widget).AsWidget().Styles.UnitContext
	p.opts.ToDots(&p.ctx)
	widg := core.AsWidget

	ii := 0
	ci := 0
	cIn := widg(p.ins[ii])
	cw := widg(cIn.Child(ci))
	atEnd := false
	for {
		// find height
		ht := float32(0)
		var ws []core.Widget
		for {
			if cw == nil {
				atEnd = true
				break
			}
			gp := cIn.Styles.Gap.Dots().Floor()
			ht += cw.Geom.Size.Actual.Total.Y + gp.Y
			if ht >= p.opts.bodyDots.Y {
				break
			}
			ws = append(ws, cw.This.(core.Widget))
			ci++
			if ci >= cIn.NumChildren() {
				ci = 0
				ii++
				if ii >= len(p.ins) {
					atEnd = true
					break
				}
				cIn = widg(p.ins[ii])
			}
			cw = widg(cIn.Child(ci))
		}
		// todo: rearrange elements to put text at bottom and non-text at top
		// now transfer over to frame
		cOut := core.NewFrame()
		cOut.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Min.X.Dot(p.opts.sizeDots.X)
			s.Min.Y.Dot(p.opts.sizeDots.Y)
			s.Max.X.Dot(p.opts.sizeDots.X)
			s.Max.Y.Dot(p.opts.sizeDots.Y)
		})
		for _, w := range ws {
			tree.MoveToParent(w, cOut)
		}
		// fmt.Println("ht:", ht, "n:", len(ws))
		p.outs = append(p.outs, cOut)
		if atEnd {
			break
		}
	}
}
