// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"io"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/core"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/pdf"
	"cogentcore.org/core/paint/renderers/pdfrender"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/printer"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// PDF generates PDF pages from given input content using given options,
// writing to the given writer. It re-renders the input widgets with
// the default PDF fonts in place (Helvetica, Times, Courier),
// and the size set to the target size as configured in the options.
// This will produce accurate PDF layout.
func PDF(w io.Writer, opts Options, ins ...core.Widget) {
	if len(ins) == 0 {
		return
	}
	cmode := matcolor.SchemeIsDark
	colors.SetScheme(false)
	cset := pdf.UseStandardFonts()

	p := pager{opts: &opts, ins: ins}
	p.ctx = *units.NewContext() // generic, invariant of actual context
	p.opts.ToDots(&p.ctx)
	p.assemble()
	p.paginate()

	sc := p.offScene()

	pdr := paint.NewPDFRenderer(opts.SizeDots, &p.ctx).(*pdfrender.Renderer)
	pdr.StartRender(w)
	np := len(p.outs)
	for i, p := range p.outs {
		tree.MoveToParent(p, sc)
		p.SetScene(sc)
		sc.StyleTree()
		sc.LayoutScene()

		sc.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if tx, ok := cwb.This.(*core.Text); ok {
				lns := tx.PaintText()
				if len(lns.Lines) > 0 {
					ln := &lns.Lines[0]
					ln.Offset.Y += 3 // todo: seriously, this fixes an otherwise inexplicable offset
				}
			}
			return true
		})

		p.RenderWidget()

		rend := sc.Painter.RenderDone()
		pdr.RenderPage(rend)
		if i < np-1 {
			pdr.AddPage()
		}
		sc.DeleteChildren()
	}
	pdr.EndRender()
	colors.SetScheme(cmode)
	pdf.RestorePreviousFonts(cset)
}

// assemble collects everything to be rendered into one big list,
// and sets the font family and size for text elements.
// only for full format rendering (e.g., PDF)
func (p *pager) assemble() {
	sc := core.AsWidget(p.ins[0]).Scene
	if p.opts.Title != nil {
		tf := core.NewFrame()
		tf.Scene = sc
		tf.FinalStyler(func(s *styles.Style) {
			s.Min.X.Dot(p.opts.BodyDots.X)
			s.Min.Y.Dot(p.opts.BodyDots.Y)
		})
		p.opts.Title(tf, p.opts)
		p.ins = append([]core.Widget{tf.This.(core.Widget)}, p.ins...)
		tf.StyleTree()
	}
	fsc := printer.Settings.FontScale() * p.opts.FontScale
	for ii, in := range p.ins {
		iw := core.AsWidget(in)

		iw.FinalStyler(func(s *styles.Style) {
			s.Min.X.Dot(p.opts.BodyDots.X)
			s.Min.Y.Dot(p.opts.BodyDots.Y)
		})
		if p.opts.Title != nil && ii == 0 { // don't restyle the title
			continue
		}
		iw.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			if tx, ok := cwb.This.(*core.Text); ok {
				if _, ok := cwb.Parent.(*core.Frame); ok { // not inside buttons etc
					cwb.Styler(func(s *styles.Style) {
						if tx.Styles.Font.Family == rich.SansSerif {
							s.Font.Family = printer.Settings.FontFamily
						}
						if tx.Type == core.TextBodyLarge {
							s.Text.LineHeight = printer.Settings.LineHeight
						}
						s.Font.Size.Value *= fsc
						s.Color = colors.Uniform(colors.Black) // in case dark mode
						if p.opts.TextStyler != nil {
							p.opts.TextStyler(tx)
						}
					})
				}
			}
			return true
		})
	}
}
