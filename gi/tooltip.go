// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
)

// TooltipConfigStyles configures the default styles
// for the given tooltip frame with the given parent.
// It should be called on tooltips when they are created.
func TooltipConfigStyles(tooltip *Scene) {
	tooltip.AddStyles(func(s *styles.Style) {
		s.Border.Style.Set(styles.BorderNone)
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Padding.Set(units.Dp(8 * Prefs.DensityMul()))
		s.BackgroundColor.SetSolid(colors.Scheme.InverseSurface)
		s.Color = colors.Scheme.InverseOnSurface
		s.BoxShadow = styles.BoxShadow1() // STYTODO: not sure whether we should have this
	})
}

// NewTooltipScene returns a new Tooltip stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewTooltipScene(sc *Scene, ctx Widget) *PopupStage {
	return NewPopupStage(Tooltip, sc, ctx)
}

// NewTooltip pops up a scene displaying the tooltip text for the given widget
// at the given position.
func NewTooltip(w Widget, pos image.Point) *PopupStage {
	sc := StageScene(w.Name() + "-tooltip")
	sc.Geom.Pos = pos
	sc.AddStyles(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.BackgroundColor.SetSolid(colors.Scheme.InverseSurface)
	})
	NewLabel(sc, "tooltip").SetText("Hello, World!").
		SetType(LabelBodyMedium).
		AddStyles(func(s *styles.Style) {
			s.Color = colors.Scheme.InverseOnSurface
		})
	return NewTooltipScene(sc, w)
}
