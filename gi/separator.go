// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// TODO(kai): this seems bad

// Separator defines a string to indicate a menu separator item
var MenuTextSeparator = "-------------"

// Separator draws a vertical or horizontal line
type Separator struct {
	Box

	// TODO(kai): remove Dim

	// Dim is the dimension the separator goes along (X means it goes longer horizontally, etc)
	Dim mat32.Dims
}

func (sp *Separator) OnInit() {
	sp.WidgetBase.OnInit()
	sp.SetStyles()
}

func (sp *Separator) SetStyles() {
	// TODO: fix disappearing separator in menu
	sp.Style(func(s *styles.Style) {
		s.Align.Self = styles.Center
		s.Justify.Self = styles.Center
		s.Background = colors.C(colors.Scheme.OutlineVariant)
		if sp.Dim == mat32.X {
			s.Grow.Set(1, 0)
			s.Min.Y.Dp(1)
			s.Margin.SetHoriz(units.Dp(6))
		} else {
			s.Grow.Set(0, 1)
			s.Min.X.Dp(1)
			s.Margin.SetVert(units.Dp(6))
		}
	})
}
