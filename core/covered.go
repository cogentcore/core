// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// IsInvisible returns whether the widget is not currently seen: it is not
// visible (see [WidgetBase.IsVisible]), or it is covered by a widget that
// is rendered over it (see [WidgetBase.isCovered]), as a dialog in the
// page over a video, or by an opaque widget in the layer of the scene (see
// [sceneLayer]), through which a translucent one is seen. It is more
// expensive than [WidgetBase.IsVisible], as it looks at the widgets of the
// scene for those over it, and is for a widget that renders often by
// itself (as a video does).
func (wb *WidgetBase) IsInvisible() bool {
	return !wb.IsVisible() || wb.covered(true)
}

// isCovered returns whether at least half of the visible box of the widget
// is covered by a box that paints over it (with a background, or an image),
// in a widget that is stacked (positioned or of a z-index, see
// [WidgetBase.stacked]) and that is rendered after it: of a higher z-index
// than the stacked widget that it is in (or itself), or of the same one but
// after it in the tree, as in the stacking of the layout of a [Frame]. It
// is covered also by a translucent box, which is the same as an opaque one
// here (rendering the widget would paint over it). The widgets in the layer
// of the scene (see [sceneLayer]) do not cover a widget that is not, which
// is rendered into the image of the scene, under the layer.
func (wb *WidgetBase) isCovered() bool {
	return wb.covered(false)
}

// covered returns whether the widget is covered (see
// [WidgetBase.isCovered]), also by the opaque widgets in the layer of the
// scene if seen is true.
func (wb *WidgetBase) covered(seen bool) bool {
	bb := wb.Geom.TotalBBox
	if bb.Empty() || wb.Scene == nil {
		return false
	}
	ancestors := map[*WidgetBase]bool{}
	var base *WidgetBase // the stacked widget that it is in, if any
	for w := wb; w != nil; w = w.parentWidget() {
		ancestors[w] = true
		if base == nil && w.stacked() {
			base = w
		}
	}
	baseZ := 0
	if base != nil {
		baseZ = base.Styles.ZIndex
	}
	half := bb.Dx() * bb.Dy() / 2
	passed := base == nil // after the base in the tree
	// the widgets in the layer of the scene are composed over the widget
	// (if it is not in the layer itself)
	layered := wb.Scene.layer.active && !wb.hasFlag(widgetInLayer)
	covered := false
	wb.Scene.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if cwb == base {
			passed = true
		}
		if covered || cwb == wb || cwb.StateIs(states.Invisible) {
			return tree.Break // (skips its children)
		}
		if ancestors[cwb] || !cwb.stacked() {
			return tree.Continue
		}
		z := cwb.Styles.ZIndex
		if z < baseZ || (z == baseZ && (!passed || z < 0)) {
			return tree.Break // under it
		}
		inLayer := layered && cwb.hasFlag(widgetInLayer)
		if inLayer && !seen {
			return tree.Break
		}
		if cwb.coversArea(bb, inLayer) >= half {
			covered = true
		}
		return tree.Break // (its children were looked at)
	})
	return covered
}

// coversArea returns the largest area of the given box that a box painted
// by the widget or a widget in it covers, only by an opaque box if opaque.
func (wb *WidgetBase) coversArea(bb image.Rectangle, opaque bool) int {
	area := 0
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if cwb.StateIs(states.Invisible) {
			return tree.Break
		}
		if !cwb.paints() || (opaque && !cwb.paintsOpaque()) {
			return tree.Continue
		}
		r := cwb.Geom.TotalBBox.Intersect(bb)
		area = max(area, r.Dx()*r.Dy())
		return tree.Continue
	})
	return area
}

// paints returns whether the widget paints its box: it has a background
// that is not transparent, or it is an image with an image.
func (wb *WidgetBase) paints() bool {
	if img, ok := wb.This.(*Image); ok && img.Image != nil {
		return true
	}
	bg := wb.Styles.Background
	if bg == nil {
		return false
	}
	if u, ok := bg.(*image.Uniform); ok {
		return !colors.IsNil(u.C) && colors.AsRGBA(u.C).A > 0 && wb.Styles.Opacity > 0
	}
	return true
}

// paintsOpaque returns whether the box that the widget paints (see
// [WidgetBase.paints]) is opaque: an image, or an opaque color.
func (wb *WidgetBase) paintsOpaque() bool {
	if img, ok := wb.This.(*Image); ok && img.Image != nil {
		return wb.Styles.Opacity >= 1
	}
	u, ok := wb.Styles.Background.(*image.Uniform)
	return ok && colors.AsRGBA(u.C).A == 255 && wb.Styles.Opacity >= 1
}
