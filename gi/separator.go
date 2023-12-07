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
		s.Align.Self = styles.Center
		s.Justify.Self = styles.Center
		if sp.Horiz {
			s.Border.Style.Top = styles.BorderSolid
			s.Border.Color.Top = colors.Scheme.OutlineVariant
			s.Border.Width.Top.Dp(2)
			s.Grow.Set(1, 0)
			s.Padding.Set(units.Dp(4), units.Zero())
			s.Min.Y.Dp(2)
			s.Min.X.Em(1)
		} else {
			s.Border.Style.Left = styles.BorderSolid
			s.Border.Color.Left = colors.Scheme.OutlineVariant
			s.Border.Width.Left.Dp(2)
			s.Padding.Set(units.Zero(), units.Dp(4))
			s.Grow.Set(0, 1)
			s.Min.X.Dp(2)
			s.Min.Y.Em(1)
		}
	})
}

func (sp *Separator) CopyFieldsFrom(frm any) {
	fr := frm.(*Separator)
	sp.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sp.Horiz = fr.Horiz
}

func (sp *Separator) RenderSeparator() {
	pc, st := sp.RenderLock()
	defer sp.RenderUnlock()

	pos := sp.Geom.Pos.Total.Add(st.TotalMargin().Pos())
	sz := sp.Geom.Size.Actual.Total.Sub(st.TotalMargin().Size())

	if !st.BackgroundColor.IsNil() {
		pc.FillBox(pos, sz, &st.BackgroundColor)
	}
	// border-top is standard property for separators in CSS (see https://www.w3schools.com/howto/howto_css_dividers.asp)
	pc.StrokeStyle.Width = st.Border.Width.Top
	pc.StrokeStyle.SetColor(&st.Border.Color.Top)
	if sp.Horiz {
		pc.DrawLine(pos.X, pos.Y+0.5*sz.Y, pos.X+sz.X, pos.Y+0.5*sz.Y)
	} else {
		pc.DrawLine(pos.X+0.5*sz.X, pos.Y, pos.X+0.5*sz.X, pos.Y+sz.Y)
	}
	pc.FillStrokeClear()
}

func (sp *Separator) Render() {
	if sp.PushBounds() {
		sp.RenderSeparator()
		sp.RenderChildren()
		sp.PopBounds()
	}
}
