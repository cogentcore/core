// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
)

// Separator defines a string to indicate a menu separator item
var MenuTextSeparator = "-------------"

// Separator draws a vertical or horizontal line
type Separator struct {
	WidgetBase

	// whether this is a horizontal separator; if false, it is vertical
	Horiz bool
}

func (sp *Separator) OnInit() {
	// TODO: fix disappearing separator in menu
	sp.Style(func(s *styles.Style) {
		s.Margin.Zero()
		s.Padding.Set(units.Dp(4), units.Zero())
		s.Align.Y = styles.AlignCenter
		s.Align.X = styles.AlignCenter
		s.Border.Style.Top = styles.BorderSolid
		s.Border.Color.Top = colors.Scheme.OutlineVariant
		s.Border.Width.Top.Dp(1)
		if sp.Horiz {
			s.Grow.Set(1, 0)
			s.Min.Y.Dp(1)
		} else {
			s.Grow.Set(0, 1)
			s.Min.X.Dp(1)
		}
	})
}

func (sp *Separator) CopyFieldsFrom(frm any) {
	fr := frm.(*Separator)
	sp.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sp.Horiz = fr.Horiz
}

func (sp *Separator) RenderSeparator(sc *Scene) {
	rs, pc, st := sp.RenderLock(sc)
	defer sp.RenderUnlock(rs)

	pos := sp.Alloc.Pos.Add(st.TotalMargin().Pos())
	sz := sp.Alloc.Size.Total.Sub(st.TotalMargin().Size())

	if !st.BackgroundColor.IsNil() {
		pc.FillBox(rs, pos, sz, &st.BackgroundColor)
	}
	// border-top is standard property for separators in CSS (see https://www.w3schools.com/howto/howto_css_dividers.asp)
	pc.StrokeStyle.Width = st.Border.Width.Top
	pc.StrokeStyle.SetColor(&st.Border.Color.Top)
	if sp.Horiz {
		pc.DrawLine(rs, pos.X, pos.Y+0.5*sz.Y, pos.X+sz.X, pos.Y+0.5*sz.Y)
	} else {
		pc.DrawLine(rs, pos.X+0.5*sz.X, pos.Y, pos.X+0.5*sz.X, pos.Y+sz.Y)
	}
	pc.FillStrokeClear(rs)
}

func (sp *Separator) Render(sc *Scene) {
	if sp.PushBounds(sc) {
		sp.RenderSeparator(sc)
		sp.RenderChildren(sc)
		sp.PopBounds(sc)
	}
}
