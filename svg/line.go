// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/chewxy/math32"
	"github.com/goki/gi"
	"github.com/goki/ki/kit"
)

// Line is a SVG line
type Line struct {
	NodeBase
	Start gi.Vec2D `xml:"{x1,y1}" desc:"position of the start of the line"`
	End   gi.Vec2D `xml:"{x2,y2}" desc:"position of the end of the line"`
}

var KiT_Line = kit.Types.AddType(&Line{}, nil)

func (g *Line) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.Lock()
	rs.PushXForm(pc.XForm)
	pc.DrawLine(rs, g.Start.X, g.Start.Y, g.End.X, g.End.Y)
	pc.Stroke(rs)
	g.ComputeBBoxSVG()

	if mrk := g.Marker("marker-start"); mrk != nil {
		ang := math32.Atan2(g.End.Y-g.Start.Y, g.End.X-g.Start.X)
		mrk.RenderMarker(g.Start, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := g.Marker("marker-end"); mrk != nil {
		ang := math32.Atan2(g.End.Y-g.Start.Y, g.End.X-g.Start.X)
		mrk.RenderMarker(g.End, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	rs.Unlock()

	g.Render2DChildren()
	rs.PopXFormLock()
}
