// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi"
	"github.com/goki/ki/kit"
)

// Circle is a SVG circle
type Circle struct {
	NodeBase
	Pos    gi.Vec2D `xml:"{cx,cy}" desc:"position of the center of the circle"`
	Radius float32  `xml:"r" desc:"radius of the circle"`
}

var KiT_Circle = kit.Types.AddType(&Circle{}, nil)

func (g *Circle) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	pc.DrawCircle(rs, g.Pos.X, g.Pos.Y, g.Radius)
	pc.FillStrokeClear(rs)
	g.ComputeBBoxSVG()
	g.Render2DChildren()
	rs.PopXForm()
}
