// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/base/slicesx"
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
		g.SetProperty("transform", g.Paint.Transform.String())
	} else {
		g.Start = xf.MulVector2AsPoint(g.Start)
		g.End = xf.MulVector2AsPoint(g.End)
		g.GradientApplyTransform(sv, xf)
	}
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Line) WriteGeom(sv *SVG, dat *[]float32) {
	*dat = slicesx.SetLength(*dat, 4+6)
	(*dat)[0] = g.Start.X
	(*dat)[1] = g.Start.Y
	(*dat)[2] = g.End.X
	(*dat)[3] = g.End.Y
	g.WriteTransform(*dat, 4)
	g.GradientWritePts(sv, dat)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Line) ReadGeom(sv *SVG, dat []float32) {
	g.Start.X = dat[0]
	g.Start.Y = dat[1]
	g.End.X = dat[2]
	g.End.Y = dat[3]
	g.ReadTransform(dat, 4)
	g.GradientReadPts(sv, dat)
}
