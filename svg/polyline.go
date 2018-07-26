// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi"
	"github.com/goki/ki/kit"
)

// Polyline is a SVG multi-line shape
type Polyline struct {
	SVGNodeBase
	Points []gi.Vec2D `xml:"points" desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest"`
}

var KiT_Polyline = kit.Types.AddType(&Polyline{}, nil)

func (g *Polyline) Render2D() {
	if len(g.Points) < 2 {
		return
	}
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	pc.DrawPolyline(rs, g.Points)
	pc.FillStrokeClear(rs)
	g.ComputeBBoxSVG()
	g.Render2DChildren()
	rs.PopXForm()
}
