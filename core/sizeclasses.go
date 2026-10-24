// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"

	"cogentcore.org/core/math32"
)

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
	sc := wb.Scene
	dots := sc.SceneGeom.Size
	// While the scene is sized to its content, the size of the window that it
	// is shown in is used, if it has one. A scene of a new window does not
	// have one yet, so the size it is expected to have is used instead, once
	// that is known (see [Stage.runWindow]). It is
	// never the current window, which a new window is not shown in, and which
	// changes as other windows are opened and closed.
	if sc.hasFlag(sceneContentSizing) {
		if rw := sc.RenderWindow(); rw != nil {
			dots = rw.SystemWindow.RenderGeom().Size
		} else if sc.sizingSize != (image.Point{}) {
			dots = sc.sizingSize
		}
	}
	return sc.dotsToDp(dots)
}

// dotsToDp converts the given size in dots to dp (density-independent pixels).
func (sc *Scene) dotsToDp(dots image.Point) math32.Vector2 {
	dpd := sc.Styles.UnitContext.Dp(1)           // dots per dp
	return math32.FromPoint(dots).DivScalar(dpd) // dots / (dots / dp) = dots * (dp / dots) = dp
}

// SizeClass returns the size class of the scene in which the widget is contained
// based on [WidgetBase.SceneSize].
func (wb *WidgetBase) SizeClass() SizeClasses {
	return sizeClassOf(wb.SceneSize().X)
}

// sizeClassOf returns the size class for the given width in dp.
func sizeClassOf(dp float32) SizeClasses {
	switch {
	case dp < 600:
		return SizeCompact
	case dp > 840:
		return SizeExpanded
	default:
		return SizeMedium
	}
}
