// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  SVG

// SVG is a viewport for containing SVG drawing objects, correponding to the
// svg tag in html -- it provides its own bitmap for drawing into
type SVG struct {
	Viewport2D
}

var KiT_SVG = kit.Types.AddType(&SVG{}, nil)

func (vp *SVG) AsNode2D() *Node2DBase {
	return &vp.Node2DBase
}

func (vp *SVG) AsViewport2D() *Viewport2D {
	return &vp.Viewport2D
}

func (vp *SVG) AsLayout2D() *Layout {
	return nil
}

func (vp *SVG) Init2D() {
	vp.Init2DBase()
	bitflag.Set(&vp.NodeFlags, int(VpFlagSVG)) // we are an svg type
}

func (vp *SVG) Style2D() {
	// we use both forms of styling -- need width, height, pos from widget..
	vp.Style2DSVG()
	vp.Style2DWidget()
}

func (vp *SVG) Size2D() {
	vp.Viewport2D.Size2D()
}

func (vp *SVG) Layout2D(parBBox image.Rectangle) {
	vp.Viewport2D.Layout2D(parBBox)
}

func (vp *SVG) BBox2D() image.Rectangle {
	return vp.Viewport2D.BBox2D()
}

func (vp *SVG) ComputeBBox2D(parBBox image.Rectangle) Vec2D {
	return vp.Viewport2D.ComputeBBox2D(parBBox)
}

func (vp *SVG) ChildrenBBox2D() image.Rectangle {
	return vp.VpBBox // no margin, padding, etc
}

func (vp *SVG) Render2D() {
	if vp.PushBounds() {
		pc := &vp.Paint
		rs := &vp.Render
		if vp.Fill {
			pc.FillStyle.SetColor(&vp.Style.Background.Color)
			pc.StrokeStyle.SetColor(nil)
			pc.DrawRectangle(rs, 0.0, 0.0, float64(vp.ViewBox.Size.X), float64(vp.ViewBox.Size.Y))
			pc.FillStrokeClear(rs)
		}
		rs.PushXForm(pc.XForm)
		vp.Render2DChildren() // we must do children first, then us!
		vp.RenderViewport2D() // update our parent image
		vp.PopBounds()
		rs.PopXForm()
	}
}

func (vp *SVG) CanReRender2D() bool {
	return true
}

func (vp *SVG) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &SVG{}

////////////////////////////////////////////////////////////////////////////////////////
//  todo parsing code etc
