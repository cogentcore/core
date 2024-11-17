// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "cogentcore.org/core/math32"

// SizeClasses are the different size classes that a window can have.
type SizeClasses int32 //enums:enum -trim-prefix Size

const (
	// SizeCompact is the size class for windows with a width less than
	// 600dp, which typically happens on phones.
	SizeCompact SizeClasses = iota

	// SizeMedium is the size class for windows with a width between 600dp
	// and 840dp inclusive, which typically happens on tablets.
	SizeMedium

	// SizeExpanded is the size class for windows with a width greater than
	// 840dp, which typically happens on desktop and laptop computers.
	SizeExpanded
)

// SceneSize returns the effective size of the scene in which the widget is contained
// in terms of dp (density-independent pixels).
func (wb *WidgetBase) SceneSize() math32.Vector2 {
	dots := math32.FromPoint(wb.Scene.SceneGeom.Size)
	if wb.Scene.hasFlag(sceneContentSizing) {
		if currentRenderWindow != nil {
			rg := currentRenderWindow.SystemWindow.RenderGeom()
			dots = math32.FromPoint(rg.Size)
		}
	}
	dpd := wb.Scene.Styles.UnitContext.Dp(1) // dots per dp
	dp := dots.DivScalar(dpd)                // dots / (dots / dp) = dots * (dp / dots) = dp
	return dp
}

// SizeClass returns the size class of the scene in which the widget is contained
// based on [WidgetBase.SceneSize].
func (wb *WidgetBase) SizeClass() SizeClasses {
	dp := wb.SceneSize().X
	switch {
	case dp < 600:
		return SizeCompact
	case dp > 840:
		return SizeExpanded
	default:
		return SizeMedium
	}
}
