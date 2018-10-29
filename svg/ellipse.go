// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi"
	"github.com/goki/ki/kit"
)

// Ellipse is a SVG ellipse
type Ellipse struct {
	NodeBase
	Pos   gi.Vec2D `xml:"{cx,cy}" desc:"position of the center of the ellipse"`
	Radii gi.Vec2D `xml:"{rx,ry}" desc:"radii of the ellipse in the horizontal, vertical axes"`
}

var KiT_Ellipse = kit.Types.AddType(&Ellipse{}, nil)

func (g *Ellipse) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.Lock()
	rs.PushXForm(pc.XForm)
	pc.DrawEllipse(rs, g.Pos.X, g.Pos.Y, g.Radii.X, g.Radii.Y)
	pc.FillStrokeClear(rs)
	rs.Unlock()

	g.ComputeBBoxSVG()
	g.Render2DChildren()

	rs.PopXFormLock()
}
