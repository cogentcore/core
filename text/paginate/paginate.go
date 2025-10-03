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
// See PDF for function that generates paginated PDFs suitable
// for printing: it ensures that the content layout matches
// the page sizes, for example, which is not done in this version.
func Paginate(opts Options, ins ...core.Widget) []*core.Frame {
	if len(ins) == 0 {
		return nil
	}
	p := pager{opts: &opts, ins: ins}
	p.optsUpdate()
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

// optsUpdate updates the option sizes based on unit context in first input.
func (p *pager) optsUpdate() {
	p.opts.Update()
	in0 := p.ins[0].AsWidget()
	p.ctx = in0.Styles.UnitContext
	p.opts.ToDots(&p.ctx)
}

// preRender re-renders inputs with styles enforced to fit page size,
// and setting the font family and size for text elements.
func (p *pager) preRender() {
	for _, in := range p.ins {
		iw := core.AsWidget(in)

		iw.FinalStyler(func(s *styles.Style) {
			s.Min.X.Dot(p.opts.bodyDots.X)
			s.Min.Y.Dot(p.opts.bodyDots.Y)
		})
		iw.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if _, ok := cwb.This.(*core.Text); ok {
				cwb.Styler(func(s *styles.Style) {
					s.Font.Family = p.opts.FontFamily
				})
			}
			return true
		})

		iw.Scene.StyleTree()
		iw.Scene.LayoutRenderScene()
	}
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
	gap := cIn.Styles.Gap.Dots().Floor()
	for {
		// find height
		ht := float32(0)
		var ws []core.Widget
		for {
			if cw == nil {
				atEnd = true
				break
			}
			ht += cw.Geom.Size.Actual.Total.Y
			if ht >= p.opts.bodyDots.Y {
				break
			}
			ht += gap.Y
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
				gap = cIn.Styles.Gap.Dots().Floor() // todo: need to track this per parent input
			}
			cw = widg(cIn.Child(ci))
		}
		// todo: rearrange elements to put text at bottom and non-text at top

		// now transfer over to frame
		page, body := p.newPage(gap)
		for _, w := range ws {
			tree.MoveToParent(w, body)
		}
		p.outs = append(p.outs, page)
		if atEnd {
			break
		}
	}
}
