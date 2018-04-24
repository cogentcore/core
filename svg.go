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

// set a normalized 0-1 scaling transform so svg's use 0-1 coordinates that
// map to actual size of the viewport -- used e.g. for Icon
func (vp *Icon) SetNormXForm() {
	pc := &vp.Paint
	pc.Identity()
	vps := Vec2D{}
	vps.SetPoint(vp.ViewBox.Size)
	pc.Scale(vps.X, vps.Y)
}

func (vp *SVG) Init2D() {
	vp.Viewport2D.Init2D()
	bitflag.Set(&vp.Flag, int(VpFlagSVG)) // we are an svg type
}

func (vp *SVG) Style2D() {
	// we use both forms of styling -- need width, height, pos from widget..
	vp.Style2DSVG(nil)
	vp.Style2DWidget(nil)
}

func (vp *SVG) Layout2D(parBBox image.Rectangle) {
	pc := &vp.Paint
	rs := &vp.Render
	vp.Layout2DBase(parBBox, true)
	rs.PushXForm(pc.XForm) // need xforms to get proper bboxes during layout
	vp.Layout2DChildren()
	rs.PopXForm()
}

func (vp *SVG) Render2D() {
	if vp.PushBounds() {
		pc := &vp.Paint
		rs := &vp.Render
		if vp.Fill {
			vp.FillViewport()
		}
		rs.PushXForm(pc.XForm)
		vp.Render2DChildren() // we must do children first, then us!
		vp.PopBounds()
		rs.PopXForm()
		vp.RenderViewport2D() // update our parent image
	}
}

// check for interface implementation
var _ Node2D = &SVG{}

////////////////////////////////////////////////////////////////////////////////////////
//  todo parsing code etc
