// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// DefaultTooltipPos returns the default position for the tooltip
// for this widget in window coordinates using the window bounding box.
func (wb *WidgetBase) DefaultTooltipPos() image.Point {
	bb := wb.winBBox()
	pos := bb.Min
	pos.X += (bb.Max.X - bb.Min.X) / 2 // center on X
	// top of Y
	return pos
}

// newTooltipFromScene returns a new Tooltip stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func newTooltipFromScene(sc *Scene, ctx Widget) *Stage {
	return NewPopupStage(TooltipStage, sc, ctx)
}

// newTooltip returns a new tooltip stage displaying the given tooltip text
// for the given widget based at the given window-level position, with the size
// defaulting to the size of the widget.
func newTooltip(w Widget, tooltip string, pos image.Point) *Stage {
	return newTooltipTextSize(w, tooltip, pos, w.AsWidget().winBBox().Size())
}

// newTooltipTextSize returns a new tooltip stage displaying the given tooltip text
// for the given widget at the given window-level position with the given size.
func newTooltipTextSize(w Widget, tooltip string, pos, sz image.Point) *Stage {
	return newTooltipFromScene(newTooltipScene(w, tooltip, pos, sz), w)
}

// newTooltipScene returns a new tooltip scene for the given widget with the
// given tooltip based on the given context position and context size.
func newTooltipScene(w Widget, tooltip string, pos, sz image.Point) *Scene {
	sc := NewScene(w.AsTree().Name + "-tooltip")
	// tooltip positioning uses the original scene geom as the context values
	sc.SceneGeom.Pos = pos
	sc.SceneGeom.Size = sz // used for positioning if needed
	sc.Styler(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Grow.Set(1, 1)
		s.Overflow.Set(styles.OverflowVisible) // key for avoiding sizing errors when re-rendering with small pref size
		s.Padding.Set(units.Dp(8))
		s.Background = colors.Scheme.InverseSurface
		s.Color = colors.Scheme.InverseOnSurface
		s.BoxShadow = styles.BoxShadow1()
	})
	NewText(sc).SetType(TextBodyMedium).SetText(tooltip).
		Styler(func(s *styles.Style) {
			s.SetTextWrap(true)
			s.Max.X.Em(20)
		})
	return sc
}
