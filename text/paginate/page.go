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

func (p *pager) newPage(gap math32.Vector2) (page, body *core.Frame) {
	styMinMax := func(s *styles.Style, x, y float32) {
		s.ZeroSpace()
		s.Min.X.Dot(x)
		s.Min.Y.Dot(y)
		s.Max.X.Dot(x)
		s.Max.Y.Dot(y)
	}
	pn := fmt.Sprintf("page-%d", len(p.outs)+1)
	page = core.NewFrame()
	page.SetName(pn)
	page.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		styMinMax(s, p.opts.sizeDots.X, p.opts.sizeDots.Y)
	})
	hdr := core.NewFrame(page)
	hdr.SetName("header")
	hdr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		styMinMax(s, p.opts.sizeDots.X, p.opts.margDots.Top)
	})
	bodRow := core.NewFrame(page)
	bodRow.SetName("body-row")
	bodRow.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		styMinMax(s, p.opts.sizeDots.X, p.opts.bodyDots.Y)
	})
	ftr := core.NewFrame(page)
	ftr.SetName("footer")
	ftr.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		styMinMax(s, p.opts.sizeDots.X, p.opts.margDots.Bottom)
	})
	lmar := core.NewFrame(bodRow)
	lmar.SetName("left-margin")
	lmar.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		styMinMax(s, p.opts.margDots.Left, p.opts.bodyDots.Y)
	})
	body = core.NewFrame(bodRow)
	body.SetName("body")
	body.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		styMinMax(s, p.opts.bodyDots.X, p.opts.bodyDots.Y)
		s.Gap.X.Dot(gap.X)
		s.Gap.Y.Dot(gap.Y)
	})
	return
}
