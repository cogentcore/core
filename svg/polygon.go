// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/mat32"
)

// Polygon is a SVG polygon
type Polygon struct {
	Polyline
}

func (g *Polygon) SVGName() string { return "polygon" }

func (g *Polygon) Render(sv *SVG) {
	sz := len(g.Points)
	if sz < 2 {
		return
	}
	vis, pc := g.PushTransform(sv)
	if !vis {
		return
	}
	pc.Lock()
	pc.DrawPolygon(g.Points)
	pc.FillStrokeClear()
	pc.Unlock()
	g.BBoxes(sv)

	if mrk := sv.MarkerByName(g, "marker-start"); mrk != nil {
		pt := g.Points[0]
		ptn := g.Points[1]
		ang := mat32.Atan2(ptn.Y-pt.Y, ptn.X-pt.X)
		mrk.RenderMarker(sv, pt, ang, g.Paint.StrokeStyle.Width.Dots)
	}
	if mrk := sv.MarkerByName(g, "marker-end"); mrk != nil {
		pt := g.Points[sz-1]
		ptp := g.Points[sz-2]
		ang := mat32.Atan2(pt.Y-ptp.Y, pt.X-ptp.X)
		mrk.RenderMarker(sv, pt, ang, g.Paint.StrokeStyle.Width.Dots)
	}
	if mrk := sv.MarkerByName(g, "marker-mid"); mrk != nil {
		for i := 1; i < sz-1; i++ {
			pt := g.Points[i]
			ptp := g.Points[i-1]
			ptn := g.Points[i+1]
			ang := 0.5 * (mat32.Atan2(pt.Y-ptp.Y, pt.X-ptp.X) + mat32.Atan2(ptn.Y-pt.Y, ptn.X-pt.X))
			mrk.RenderMarker(sv, pt, ang, g.Paint.StrokeStyle.Width.Dots)
		}
	}

	g.RenderChildren(sv)
	pc.PopTransformLock()
}
