// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/math32"
)

// Line is a SVG line
type Line struct {
	NodeBase

	// position of the start of the line
	Start math32.Vector2 `xml:"{x1,y1}"`

	// position of the end of the line
	End math32.Vector2 `xml:"{x2,y2}"`
}

func (g *Line) SVGName() string { return "line" }

func (g *Line) Init() {
	g.NodeBase.Init()
	g.End.Set(1, 1)
}

func (g *Line) SetPos(pos math32.Vector2) {
	g.Start = pos
}

func (g *Line) SetSize(sz math32.Vector2) {
	g.End = g.Start.Add(sz)
}

func (g *Line) LocalBBox(sv *SVG) math32.Box2 {
	bb := math32.B2Empty()
	bb.ExpandByPoint(g.Start)
	bb.ExpandByPoint(g.End)
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min.SetSubScalar(hlw)
	bb.Max.SetAddScalar(hlw)
	return bb
}

func (g *Line) Render(sv *SVG) {
	if !g.IsVisible(sv) {
		return
	}
	pc := g.Painter(sv)
	pc.Line(g.Start.X, g.Start.Y, g.End.X, g.End.Y)
	pc.Draw()

	g.PushContext(sv)
	if mrk := sv.MarkerByName(g, "marker-start"); mrk != nil {
		ang := math32.Atan2(g.End.Y-g.Start.Y, g.End.X-g.Start.X)
		mrk.RenderMarker(sv, g.Start, ang, g.Paint.Stroke.Width.Dots)
	}
	if mrk := sv.MarkerByName(g, "marker-end"); mrk != nil {
		ang := math32.Atan2(g.End.Y-g.Start.Y, g.End.X-g.Start.X)
		mrk.RenderMarker(sv, g.End, ang, g.Paint.Stroke.Width.Dots)
	}
	pc.PopContext()
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Line) ApplyTransform(sv *SVG, xf math32.Matrix2) {
	rot := xf.ExtractRot()
	if rot != 0 {
		g.Paint.Transform.SetMul(xf)
		g.SetTransformProperty()
	} else {
		g.Start = xf.MulVector2AsPoint(g.Start)
		g.End = xf.MulVector2AsPoint(g.End)
		g.GradientApplyTransform(sv, xf)
	}
}
