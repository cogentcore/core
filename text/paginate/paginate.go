// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	_ "cogentcore.org/core/text/tex"
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
	sc := core.AsWidget(p.ins[0]).Scene
	// sc.AsyncLock()
	if p.opts.Title != nil {
		tf := core.NewFrame()
		tf.Scene = sc
		tf.FinalStyler(func(s *styles.Style) {
			s.Min.X.Dot(p.opts.BodyDots.X)
			s.Min.Y.Dot(p.opts.BodyDots.Y)
		})
		p.opts.Title(tf, p.opts)
		p.ins = append([]core.Widget{tf.This.(core.Widget)}, p.ins...)
	}
	for _, in := range p.ins {
		iw := core.AsWidget(in)

		iw.FinalStyler(func(s *styles.Style) {
			s.Min.X.Dot(p.opts.BodyDots.X)
			s.Min.Y.Dot(p.opts.BodyDots.Y)
		})
		iw.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if tx, ok := cwb.This.(*core.Text); ok {
				if tx.Styles.Font.Family == rich.SansSerif {
					if _, ok := cwb.Parent.(*core.Frame); ok { // not inside buttons etc
						cwb.Styler(func(s *styles.Style) {
							s.Font.Family = p.opts.FontFamily
						})
					}
				}
			}
			return true
		})

		iw.Scene.StyleTree()
		iw.Scene.LayoutRenderScene()
	}
	// sc.AsyncUnlock()
}

func (p *pager) paginate() {
	its := p.extract()
	p.layout(its)
}
