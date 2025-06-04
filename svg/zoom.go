// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image"

	"cogentcore.org/core/math32"
)

// ContentBounds returns the bounding box of the contents
// in its natural units, without any Viewbox transformations, etc.
// Can set the Viewbox to this to have the contents fully occupy the space.
func (sv *SVG) ContentBounds() math32.Box2 {
	tr := sv.Root.Paint.Transform
	sv.Root.Paint.Transform = math32.Identity2()
	sv.Root.BBoxes(sv, math32.Identity2())
	sv.Root.Paint.Transform = tr
	return sv.Root.BBox
}

// UpdateSize ensures that the size is valid, using existing ViewBox values
// to set proportions if size is not valid.
func (sv *SVG) UpdateSize() {
	vb := &sv.Root.ViewBox
	if vb.Size.X == 0 {
		if sv.PhysicalWidth.Dots > 0 {
			vb.Size.X = sv.PhysicalWidth.Dots
		} else {
			vb.Size.X = 640
		}
	}
	if vb.Size.Y == 0 {
		if sv.PhysicalHeight.Dots > 0 {
			vb.Size.Y = sv.PhysicalHeight.Dots
		} else {
			vb.Size.Y = 480
		}
	}
	if sv.Geom.Size.X > 0 && sv.Geom.Size.Y == 0 {
		sv.Geom.Size.Y = int(float32(sv.Geom.Size.X) * (float32(vb.Size.Y) / float32(vb.Size.X)))
	} else if sv.Geom.Size.Y > 0 && sv.Geom.Size.X == 0 {
		sv.Geom.Size.X = int(float32(sv.Geom.Size.Y) * (float32(vb.Size.X) / float32(vb.Size.Y)))
	}
}

// setRootTransform sets the Root node transform based on ViewBox, Translate, Scale
// parameters set on the SVG object.
func (sv *SVG) setRootTransform() {
	sv.UpdateSize()
	vb := &sv.Root.ViewBox
	box := math32.FromPoint(sv.Geom.Size)
	tr := math32.Translate2D(float32(sv.Geom.Pos.X)+sv.Translate.X, float32(sv.Geom.Pos.Y)+sv.Translate.Y)
	_, trans, scale := vb.Transform(box)
	if sv.InvertY {
		scale.Y *= -1
	}
	trans.SetSub(vb.Min)
	scale.SetMulScalar(sv.Scale)
	rt := math32.Scale2D(scale.X, scale.Y).Translate(trans.X, trans.Y)
	if sv.InvertY {
		rt.Y0 = -rt.Y0
	}
	sv.Root.Paint.Transform = tr.Mul(rt)
}

// SetDPITransform sets a scaling transform to compensate for
// a given LogicalDPI factor.
// svg rendering is done within a 96 DPI context.
func (sv *SVG) SetDPITransform(logicalDPI float32) {
	pc := &sv.Root.Paint
	dpisc := logicalDPI / 96.0
	pc.Transform = math32.Scale2D(dpisc, dpisc)
}

// ZoomAtScroll calls ZoomAt using the Delta.Y and Pos() parameters
// from an [events.MouseScroll] event, to produce well-behaved zooming behavior,
// for elements of any size.
func (sv *SVG) ZoomAtScroll(deltaY float32, pos image.Point) {
	del := 0.01 * deltaY * max(sv.Root.ViewBox.Size.X/1280, 0.1)
	del /= max(1, sv.Scale)
	del = math32.Clamp(del, -0.1, 0.1)
	sv.ZoomAt(pos, del)
}

// ZoomAt updates the global Scale by given delta value,
// by multiplying the current Scale by 1+delta
// (+ means zoom in; - means zoom out).
// Delta should be < 1 in magnitude, and resulting scale is clamped
// in range 0.01..100.
// The global Translate is updated so that the given render
// coordinate point (dots) corresponds to the same
// underlying svg viewbox coordinate point.
func (sv *SVG) ZoomAt(pt image.Point, delta float32) {
	sc := float32(1)
	if delta > 1 {
		sc += delta
	} else {
		sc *= (1 - math32.Min(-delta, .5))
	}
	nsc := math32.Clamp(sv.Scale*sc, 0.01, 100)
	sv.ScaleAt(pt, nsc)
}

// ScaleAt sets the global Scale parameter and updates the
// global Translate parameter so that the given render coordinate
// point (dots) corresponds to the same underlying svg viewbox
// coordinate point.
func (sv *SVG) ScaleAt(pt image.Point, sc float32) {
	sv.setRootTransform()
	rxf := sv.Root.Paint.Transform.Inverse()
	mpt := math32.FromPoint(pt)
	xpt := rxf.MulVector2AsPoint(mpt)
	sv.Scale = sc

	sv.setRootTransform()
	rxf = sv.Root.Paint.Transform
	npt := rxf.MulVector2AsPoint(xpt) // original point back to screen
	dpt := mpt.Sub(npt)
	sv.Translate.SetAdd(dpt)
}

func (sv *SVG) ZoomReset() {
	sv.Translate.Set(0, 0)
	sv.Scale = 1
}

// ZoomToContents sets the scale to fit the current contents
// into a display of given size.
func (sv *SVG) ZoomToContents(size math32.Vector2) {
	sv.ZoomReset()
	bb := sv.ContentBounds()
	bsz := bb.Size().Max(math32.Vec2(1, 1))
	if bsz == (math32.Vector2{}) {
		return
	}
	sc := size.Div(bsz)
	sv.Translate = bb.Min.Negate()
	sv.Scale = math32.Min(sc.X, sc.Y)
}

// ResizeToContents resizes the drawing to just fit the current contents,
// including moving everything to start at upper-left corner.
// The given grid spacing parameter ensures sizes are in units of the grid
// spacing: pass a 1 to just use actual sizes.
func (sv *SVG) ResizeToContents(grid float32) {
	sv.ZoomReset()
	bb := sv.ContentBounds()
	bsz := bb.Size()
	if bsz.X <= 0 || bsz.Y <= 0 {
		return
	}
	trans := bb.Min
	treff := trans
	if grid > 1 {
		incr := grid * sv.Scale
		treff.X = math32.Floor(trans.X/incr) * incr
		treff.Y = math32.Floor(trans.Y/incr) * incr
		bsz.SetAdd(trans.Sub(treff))
		bsz.X = math32.Ceil(bsz.X/incr) * incr
		bsz.Y = math32.Ceil(bsz.Y/incr) * incr
	}
	root := sv.Root
	root.ViewBox.Min = treff
	root.ViewBox.Size = bsz
	// sv.PhysicalWidth.Value = bsz.X
	// sv.PhysicalHeight.Value = bsz.Y
}
