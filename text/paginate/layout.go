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
		sz := ih + it.gap.Y
		return sz
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
				nsz := size(its[i+1]) + 100 // extra space to be sure
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

func (p *pager) outputPages(pgs [][]*item) {
	for _, pg := range pgs {
		lastGap := math32.Vector2{}
		lastLeft := float32(0)
		if len(pg) > 0 {
			lastGap = pg[0].gap
		}
		page, body := p.newPage(lastGap)
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
		p.outs = append(p.outs, page)
	}
}

func (p *pager) newOutFrame(par *core.Frame, gap math32.Vector2, left float32) *core.Frame {
	fr := core.NewFrame(par)
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

// layout reorders items within the pages and generates final output.
func (p *pager) layout(its []*item) {
	pgs := p.pagify(its)
	// note: could rearrange elements to put text at bottom and non-text at top?
	// but this is probably not necessary?
	p.outputPages(pgs)
}
