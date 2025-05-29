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

// setRootTransform sets the Root node transform based on ViewBox, Translate, Scale
// parameters set on the SVG object.
func (sv *SVG) setRootTransform() {
	vb := &sv.Root.ViewBox
	box := math32.FromPoint(sv.Geom.Size)
	if vb.Size.X == 0 {
		vb.Size.X = sv.PhysicalWidth.Dots
	}
	if vb.Size.Y == 0 {
		vb.Size.Y = sv.PhysicalHeight.Dots
	}
	tr := math32.Translate2D(float32(sv.Geom.Pos.X), float32(sv.Geom.Pos.Y))
	_, trans, scale := vb.Transform(box)
	if sv.InvertY {
		scale.Y *= -1
	}
	trans.SetSub(vb.Min)
	trans.SetAdd(sv.Translate)
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

// ZoomAt updates the scale and translate parameters at given point
// by given delta: + means zoom in, - means zoom out,
// delta should always be < 1 in magnitude.
func (sv *SVG) ZoomAt(pt image.Point, delta float32) {
	sc := float32(1)
	if delta > 1 {
		sc += delta
	} else {
		sc *= (1 - math32.Min(-delta, .5))
	}

	osc := sv.Scale
	nsc := osc * sc
	nsc = math32.Clamp(nsc, 0.01, 100)

	rxf := sv.Root.Paint.Transform
	xf := rxf.Inverse()

	mpt := math32.FromPoint(pt)
	xpt := xf.MulVector2AsPoint(mpt)
	xpt.SetSub(sv.Root.ViewBox.Min)
	dt := xpt.DivScalar(nsc).Sub(xpt.DivScalar(osc))

	sv.Translate.SetAdd(dt)
	sv.Scale = nsc
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
