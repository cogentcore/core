// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// Separator draws a separator line. It goes in the direction
// specified by [style.Style.Direction].
type Separator struct {
	Box
}

func (sp *Separator) OnInit() {
	sp.WidgetBase.OnInit()
	sp.SetStyles()
}

func (sp *Separator) SetStyles() {
	sp.Style(func(s *styles.Style) {
		s.Align.Self = styles.Center
		s.Justify.Self = styles.Center
		s.Background = colors.C(colors.Scheme.OutlineVariant)
	})
	sp.StyleFinal(func(s *styles.Style) {
		if s.Direction == styles.Row {
			s.Grow.Set(1, 0)
			s.Min.Y.Dp(1)
			s.Margin.SetHorizontal(units.Dp(6))
		} else {
			s.Grow.Set(0, 1)
			s.Min.X.Dp(1)
			s.Margin.SetVertical(units.Dp(6))
		}
	})
}
