// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/math32"
)

// Polyline is a SVG multi-line shape
type Polyline struct {
	NodeBase

	// the coordinates to draw -- does a moveto on the first, then lineto for all the rest
	Points []math32.Vector2 `xml:"points"`
}

func (g *Polyline) SVGName() string { return "polyline" }

func (g *Polyline) SetPos(pos math32.Vector2) {
	// todo: set offset relative to bbox
}

func (g *Polyline) SetSize(sz math32.Vector2) {
	// todo: scale bbox
}

func (g *Polyline) LocalBBox(sv *SVG) math32.Box2 {
	bb := math32.B2Empty()
	for _, pt := range g.Points {
		bb.ExpandByPoint(pt)
	}
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min.SetSubScalar(hlw)
	bb.Max.SetAddScalar(hlw)
	return bb
}

func (g *Polyline) Render(sv *SVG) {
	sz := len(g.Points)
	if sz < 2 || !g.IsVisible(sv) {
		return
	}
	pc := g.Painter(sv)
	pc.Polyline(g.Points...)
	pc.Draw()

	g.PushContext(sv)
	if mrk := sv.MarkerByName(g, "marker-start"); mrk != nil {
		pt := g.Points[0]
		ptn := g.Points[1]
		ang := math32.Atan2(ptn.Y-pt.Y, ptn.X-pt.X)
		mrk.RenderMarker(sv, pt, ang, g.Paint.Stroke.Width.Dots)
	}
	if mrk := sv.MarkerByName(g, "marker-end"); mrk != nil {
		pt := g.Points[sz-1]
		ptp := g.Points[sz-2]
		ang := math32.Atan2(pt.Y-ptp.Y, pt.X-ptp.X)
		mrk.RenderMarker(sv, pt, ang, g.Paint.Stroke.Width.Dots)
	}
	if mrk := sv.MarkerByName(g, "marker-mid"); mrk != nil {
		for i := 1; i < sz-1; i++ {
			pt := g.Points[i]
			ptp := g.Points[i-1]
			ptn := g.Points[i+1]
			ang := 0.5 * (math32.Atan2(pt.Y-ptp.Y, pt.X-ptp.X) + math32.Atan2(ptn.Y-pt.Y, ptn.X-pt.X))
			mrk.RenderMarker(sv, pt, ang, g.Paint.Stroke.Width.Dots)
		}
	}
	pc.PopContext()
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Polyline) ApplyTransform(sv *SVG, xf math32.Matrix2) {
	rot := xf.ExtractRot()
	if rot != 0 {
		g.Paint.Transform.SetMul(xf)
		g.SetTransformProperty()
	} else {
		for i, p := range g.Points {
			p = xf.MulVector2AsPoint(p)
			g.Points[i] = p
		}
		g.GradientApplyTransform(sv, xf)
	}
}
