// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/colors"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
)

// TODO: rich tooltips

// NewTooltipFromScene returns a new Tooltip stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewTooltipFromScene(sc *Scene, ctx Widget) *PopupStage {
	return NewPopupStage(TooltipStage, sc, ctx)
}

// NewTooltip returns a new tooltip stage displaying the tooltip text
// for the given widget at the given position.
func NewTooltip(w Widget, pos image.Point) *PopupStage {
	return NewTooltipText(w, w.AsWidget().Tooltip, pos)
}

// NewTooltipText returns a new tooltip stage displaying the given tooltip text
// for the given widget at the given position.
func NewTooltipText(w Widget, tooltip string, pos image.Point) *PopupStage {
	return NewTooltipFromScene(NewTooltipScene(w, tooltip, pos), w)
}

// NewTooltipScene returns a new tooltip scene for the given widget with the
// given tooltip based on the given position.
func NewTooltipScene(w Widget, tooltip string, pos image.Point) *Scene {
	sc := NewScene(w.Name() + "-tooltip")
	sc.SceneGeom.Pos.X = pos.X
	sc.SceneGeom.Pos.Y = pos.Y
	sc.Style(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Grow.Set(1, 1)
		s.Padding.Set(units.Dp(8))
		s.BackgroundColor.SetSolid(colors.Scheme.InverseSurface)
		s.Color = colors.Scheme.InverseOnSurface
		s.BoxShadow = styles.BoxShadow1()
	})
	NewLabel(sc, "text").SetType(LabelBodyMedium).SetText(tooltip).
		Style(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Text.WhiteSpace = styles.WhiteSpaceNormal
			if s.Is(states.Selected) {
				s.Color = colors.Scheme.Select.OnContainer
			}
		})
	return sc
}
