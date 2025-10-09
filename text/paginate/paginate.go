// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles/units"
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
	sc  *core.Scene
}

// optsUpdate updates the option sizes based on unit context in first input.
func (p *pager) optsUpdate() {
	p.opts.Update()
	in0 := p.ins[0].AsWidget()
	p.ctx = in0.Styles.UnitContext
	p.opts.ToDots(&p.ctx)
}

func (p *pager) paginate() {
	its := p.extract()
	p.layout(its)
}

// offScene returns a scene suitable for offline rendering,
// sized to the full page size.
func (p *pager) offScene() *core.Scene {
	if p.sc != nil {
		p.sc.DeleteChildren()
		return p.sc
	}
	sz := p.opts.SizeDots
	sc := core.NewScene()
	// critical to be image not htmlcanvas, and not to be the correct size yet..
	sc.Renderer = paint.NewImageRenderer(sz.SubScalar(1))
	scsz := math32.Geom2DInt{}
	scsz.Size = sz.ToPointCeil()
	sc.Resize(scsz)
	sc.MakeTextShaper()
	p.sc = sc
	return sc
}
