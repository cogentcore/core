// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Polygon is a SVG polygon
type Polygon struct {
	Polyline
}

var KiT_Polygon = kit.Types.AddType(&Polygon{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewPolygon adds a new polygon to given parent node, with given name and points.
func AddNewPolygon(parent ki.Ki, name string, points []mat32.Vec2) *Polygon {
	g := parent.AddNewChild(KiT_Polygon, name).(*Polygon)
	g.Points = points
	return g
}

func (g *Polygon) SVGName() string { return "polygon" }

func (g *Polygon) Render2D() {
	sz := len(g.Points)
	if sz < 2 {
		return
	}
	vis, rs := g.PushXForm()
	if !vis {
		return
	}
	pc := &g.Pnt
	pc.DrawPolygon(rs, g.Points)
	pc.FillStrokeClear(rs)
	g.ComputeBBoxSVG()

	if mrk := MarkerByName(g, "marker-start"); mrk != nil {
		pt := g.Points[0]
		ptn := g.Points[1]
		ang := mat32.Atan2(ptn.Y-pt.Y, ptn.X-pt.X)
		mrk.RenderMarker(pt, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := MarkerByName(g, "marker-end"); mrk != nil {
		pt := g.Points[sz-1]
		ptp := g.Points[sz-2]
		ang := mat32.Atan2(pt.Y-ptp.Y, pt.X-ptp.X)
		mrk.RenderMarker(pt, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := MarkerByName(g, "marker-mid"); mrk != nil {
		for i := 1; i < sz-1; i++ {
			pt := g.Points[i]
			ptp := g.Points[i-1]
			ptn := g.Points[i+1]
			ang := 0.5 * (mat32.Atan2(pt.Y-ptp.Y, pt.X-ptp.X) + mat32.Atan2(ptn.Y-pt.Y, ptn.X-pt.X))
			mrk.RenderMarker(pt, ang, g.Pnt.StrokeStyle.Width.Dots)
		}
	}

	g.Render2DChildren()
	rs.PopXFormLock()
}
