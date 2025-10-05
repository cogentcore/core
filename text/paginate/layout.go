// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tree"
)

// pagify organizes widget list into page-sized chunks.
func (p *pager) pagify(its []*item) [][]*item {
	widg := core.AsWidget
	size := func(it *item) float32 {
		return widg(it.w).Geom.Size.Actual.Total.Y + it.gap.Y
	}

	maxY := p.opts.bodyDots.Y

	var pgs [][]*item
	var cpg []*item
	ht := float32(0)
	n := len(its)
	for i, ci := range its {
		cw := widg(ci.w)
		brk := cw.Property("paginate-break") != nil
		nobrk := cw.Property("paginate-no-break-after") != nil
		sz := size(ci)
		over := ht+sz > maxY
		if !over && nobrk {
			if i < n-1 {
				nsz := size(its[i+1])
				if ht+sz+nsz > maxY {
					over = true // break now
				}
			}
		}
		if brk || over {
			if !brk && len(cpg) == 0 { // no blank pages!
				cpg = append(cpg, ci)
			}
			pgs = append(pgs, cpg)
			cpg = nil
			ht = 0
		}
		ht += sz
		cpg = append(cpg, ci)
	}
	return pgs
}

// layout reorders items within the pages and generates final output.
func (p *pager) layout(its []*item) {
	pgs := p.pagify(its)
	for _, pg := range pgs {
		// todo: rearrange elements to put text at bottom and non-text at top

		gap := math32.Vector2{}
		if len(pg) > 0 {
			gap = pg[0].gap // todo: better
		}
		// now transfer over to frame
		page, body := p.newPage(gap)
		for _, it := range pg {
			tree.MoveToParent(it.w, body)
		}
		p.outs = append(p.outs, page)
	}
}
