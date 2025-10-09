// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// pagify organizes widget list into page-sized chunks.
func (p *pager) pagify(its []*item) [][]*item {
	widg := core.AsWidget
	size := func(it *item) float32 {
		wb := widg(it.w)
		ih := wb.Geom.Size.Actual.Total.Y // todo: something wrong with size!
		return ih + it.gap.Y
	}

	maxY := p.opts.BodyDots.Y

	var pgs [][]*item
	var cpg []*item
	ht := float32(0)
	n := len(its)
	for i, ci := range its {
		cw := widg(ci.w)
		// fmt.Println(cw)
		brk := cw.Property("paginate-break") != nil
		nobrk := cw.Property("paginate-no-break-after") != nil
		sz := size(ci)
		over := ht+sz > maxY
		if !over && nobrk {
			if i < n-1 {
				nsz := size(its[i+1]) // extra space to be sure
				if ht+sz+nsz > maxY {
					over = true // break now
					// } else {
					// 	fmt.Println("not !over, nobrk:", ht, sz, nsz, maxY, ht+sz+nsz, cw, its[i+1])
				}
			}
		}
		if brk || over {
			ht = 0
			if !brk && len(cpg) == 0 { // no blank pages!
				cpg = append(cpg, ci)
				pgs = append(pgs, cpg)
				cpg = nil
				continue
			}
			pgs = append(pgs, cpg)
			cpg = nil
		}
		ht += sz
		cpg = append(cpg, ci)
	}
	if len(cpg) > 0 {
		pgs = append(pgs, cpg)
	}
	return pgs
}

func (p *pager) outputPages(pgs [][]*item, newPage func(gap math32.Vector2, pageNo int) (page, body *core.Frame)) []*core.Frame {
	var outs []*core.Frame
	for pn, pg := range pgs {
		lastGap := math32.Vector2{}
		lastLeft := float32(0)
		if len(pg) > 0 {
			lastGap = pg[0].gap
		}
		page, body := newPage(lastGap, pn+1)
		cpar := body
		for _, it := range pg {
			gap := it.gap
			left := it.left
			if gap != lastGap || left != lastLeft {
				cpar = p.newOutFrame(body, gap, left)
				lastGap = gap
				lastLeft = left
			}
			tree.MoveToParent(it.w, cpar)
		}
		outs = append(outs, page)
	}
	return outs
}

func (p *pager) newOutFrame(par *core.Frame, gap math32.Vector2, left float32) *core.Frame {
	var fr *core.Frame
	if par != nil {
		fr = core.NewFrame(par)
	} else {
		fr = core.NewFrame()
	}
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.ZeroSpace()
		s.Min.X.Dot(p.opts.BodyDots.X)
		s.Max.X.Dot(p.opts.BodyDots.X)
		s.Gap.X.Dot(gap.X)
		s.Gap.Y.Dot(gap.Y)
		s.Padding.Left.Dot(core.ConstantSpacing(left))
	})
	return fr
}

// pre-render everything in the offscreen scene that will be used for final
// to get accurate element sizes.
func (p *pager) preRender(its []*item) {
	pg := [][]*item{its}
	op := p.outputPages(pg, func(gap math32.Vector2, pageNo int) (page, body *core.Frame) {
		fr := p.newOutFrame(nil, gap, 0)
		page, body = fr, fr
		return
	})
	sc := core.NewScene()
	sz := math32.Geom2DInt{}
	sz.Size = p.opts.SizeDots.ToPointCeil()
	sc.Resize(sz)
	sc.MakeTextShaper()

	tree.MoveToParent(op[0], sc)
	op[0].SetScene(sc)
	sc.StyleTree()
	sc.LayoutScene()
}

// layout reorders items within the pages and generates final output.
func (p *pager) layout(its []*item) {
	p.preRender(its)
	pgs := p.pagify(its)
	// note: could rearrange elements to put text at bottom and non-text at top?
	// but this is probably not necessary?
	p.outs = p.outputPages(pgs, p.newPage)
}
