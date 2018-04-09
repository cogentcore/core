// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
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

func (vp *SVG) Init2D() {
	vp.Viewport2D.Init2D()
	bitflag.Set(&vp.NodeFlags, int(VpFlagSVG)) // we are an svg type
}

func (vp *SVG) Style2D() {
	// we use both forms of styling -- need width, height, pos from widget..
	vp.Style2DSVG(nil)
	vp.Style2DWidget(nil)
}

func (vp *SVG) Render2D() {
	if vp.PushBounds() {
		pc := &vp.Paint
		rs := &vp.Render
		if vp.Fill {
			var tmp = Paint{}
			tmp = vp.Paint
			tmp.FillStyle.SetColor(&vp.Style.Background.Color)
			tmp.StrokeStyle.SetColor(nil)
			tmp.DrawRectangle(rs, 0.0, 0.0, float64(vp.ViewBox.Size.X), float64(vp.ViewBox.Size.Y))
			tmp.FillStrokeClear(rs)
		}
		rs.PushXForm(pc.XForm)
		vp.Render2DChildren() // we must do children first, then us!
		vp.RenderViewport2D() // update our parent image
		vp.PopBounds()
		rs.PopXForm()
	}
}

// check for interface implementation
var _ Node2D = &SVG{}

////////////////////////////////////////////////////////////////////////////////////////
//  todo parsing code etc
