// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"io"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/renderers/pdfrender"
	"cogentcore.org/core/tree"
)

// PDF generates PDF pages from given input content using given options,
// writing to the given writer.
func PDF(w io.Writer, opts Options, ins ...core.Widget) {
	if len(ins) == 0 {
		return
	}
	p := pager{opts: &opts, ins: ins}
	p.paginate()
	sc := core.NewScene()
	sz := math32.Geom2DInt{}
	sz.Size = opts.sizeDots.ToPointCeil()
	sc.Resize(sz)
	sc.MakeTextShaper()
	pdr := paint.NewPDFRenderer(opts.sizeDots, &p.ctx).(*pdfrender.Renderer)
	np := len(p.outs)
	for i, p := range p.outs {
		tree.MoveToParent(p, sc)
		p.SetScene(sc)
		sc.StyleTree()
		sc.LayoutRenderScene()

		rend := sc.Painter.RenderDone()
		pdr.RenderPage(w, rend)
		break
		if i < np-1 {
			pdr.PDF.NewPage(opts.sizeDots.X, opts.sizeDots.Y)
		}
		sc.DeleteChildren()
	}
	pdr.PDF.Close()
}
