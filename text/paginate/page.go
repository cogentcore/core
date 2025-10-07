// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"fmt"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
)

func styMinMax(s *styles.Style, x, y float32) {
	s.ZeroSpace()
	s.Min.X.Dot(x)
	s.Min.Y.Dot(y)
	s.Max.X.Dot(x)
	s.Max.Y.Dot(y)
}

func (p *pager) newPage(gap math32.Vector2) (page, body *core.Frame) {

	curPage := len(p.outs) + 1
	pn := fmt.Sprintf("page-%d", curPage)

	page = core.NewFrame()
	page.SetName(pn)
	page.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		styMinMax(s, p.opts.SizeDots.X, p.opts.SizeDots.Y)
	})
	lmar := core.NewFrame(page)
	lmar.SetName("left-margin")
	lmar.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		styMinMax(s, p.opts.MargDots.Left, p.opts.SizeDots.Y)
	})
	bfr := core.NewFrame(page)
	bfr.SetName("body-frame")
	bfr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		styMinMax(s, p.opts.BodyDots.X, p.opts.SizeDots.Y)
	})

	hdr := core.NewFrame(bfr)
	hdr.SetName("header")
	hdr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		styMinMax(s, p.opts.BodyDots.X, p.opts.MargDots.Top)
	})
	if p.opts.Header != nil {
		p.opts.Header(hdr, p.opts, curPage)
	}

	body = core.NewFrame(bfr)
	body.SetName("body")
	body.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		styMinMax(s, p.opts.BodyDots.X, p.opts.BodyDots.Y)
		s.Gap.X.Dot(gap.X)
		s.Gap.Y.Dot(gap.Y)
	})

	ftr := core.NewFrame(bfr)
	ftr.SetName("footer")
	ftr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		styMinMax(s, p.opts.BodyDots.X, p.opts.MargDots.Bottom)
	})
	if p.opts.Footer != nil {
		p.opts.Footer(ftr, p.opts, curPage)
	}

	return
}
