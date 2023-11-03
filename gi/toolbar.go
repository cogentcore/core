// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
)

// ToolbarStyles can be applied to any layout (e.g., Frame) to achieve
// standard toolbar styling.
// It is used in the Toolbar and TopAppBar widgets.
func ToolbarStyles(ly Layouter) {
	lb := ly.AsLayout()
	ly.Style(func(s *styles.Style) {
		s.SetStretchMaxWidth()
		s.Border.Radius = styles.BorderRadiusFull
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
		s.Padding.SetHoriz(units.Dp(16))
		s.Spacing.Zero()
	})
	ly.OnWidgetAdded(func(w Widget) {
		if bt := AsButton(w); bt != nil {
			bt.Type = ButtonAction
			return
		}
		if sp, ok := w.(*Separator); ok {
			sp.Horiz = lb.Lay != LayoutHoriz
		}
	})
}

// Toolbar is just a styled Frame layout for holding buttons
// and other widgets.  Use this for any toolbar embedded within
// a window.  See TopAppBar for the main app-level toolbar,
// with considerable additional functionality.
type Toolbar struct {
	Frame
}

func (tb *Toolbar) OnInit() {
	tb.ToolbarStyles()
	tb.HandleLayoutEvents()
}

func (tb *Toolbar) ToolbarStyles() {
	ToolbarStyles(tb)
}

// UpdateButtons calls UpdateFunc on all buttons in toolbar.
// individual menus are automatically generated at popup time.
func (tb *Toolbar) UpdateButtons() {
	if tb == nil {
		return
	}
	updt := tb.UpdateStart()
	defer tb.UpdateEndRender(updt)

	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			ac.UpdateButtons()
		}
	}
}
