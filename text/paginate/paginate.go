// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"cogentcore.org/core/base/stack"
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
				if _, ok := cwb.Parent.(*core.Frame); ok { // not inside buttons etc
					cwb.Styler(func(s *styles.Style) {
						s.Font.Family = p.opts.FontFamily
					})
				}
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
	type posn struct {
		w core.Widget
		i int
	}

	pars := stack.Stack[*posn]{}
	pars.Push(&posn{p.ins[ii], 0})
	atEnd := false
	gap := p.ins[0].AsWidget().Styles.Gap.Dots().Floor()

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

	maxY := p.opts.bodyDots.Y
	// per page, list of widgets -- must accumulate all, to not change structure
	var ws [][]core.Widget
	for {
		var cws []core.Widget
		ht := float32(0)
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
			cw := widg(cpw.Child(cp.i))
			sz := cw.Geom.Size.Actual.Total.Y
			if ht+sz > maxY {
				if fr, ok := cw.This.(*core.Frame); ok {
					pars.Push(&posn{fr.This.(core.Widget), 0})
					continue
				}
				if len(cws) == 0 {
					cws = append(cws, cw.This.(core.Widget))
					next()
				}
				break
			}
			ht += sz + gap.Y
			cws = append(cws, cw.This.(core.Widget))
			next()
			if atEnd {
				break
			}
		}
		ws = append(ws, cws)
		if atEnd {
			break
		}
	}

	for _, cws := range ws {
		// todo: rearrange elements to put text at bottom and non-text at top
		// now transfer over to frame
		page, body := p.newPage(gap)
		for _, w := range cws {
			tree.MoveToParent(w, body)
		}
		p.outs = append(p.outs, page)
	}
}
