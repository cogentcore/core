// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi"
	"github.com/goki/ki/kit"
)

// Line is a SVG line
type Line struct {
	SVGNodeBase
	Start gi.Vec2D `xml:"{x1,y1}" desc:"position of the start of the line"`
	End   gi.Vec2D `xml:"{x2,y2}" desc:"position of the end of the line"`
}

var KiT_Line = kit.Types.AddType(&Line{}, nil)

func (g *Line) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	pc.DrawLine(rs, g.Start.X, g.Start.Y, g.End.X, g.End.Y)
	pc.Stroke(rs)
	g.ComputeBBoxSVG()
	g.Render2DChildren()
	rs.PopXForm()
}
