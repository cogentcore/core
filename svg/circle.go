// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/math32"
)

// Circle is a SVG circle
type Circle struct {
	NodeBase

	// position of the center of the circle
	Pos math32.Vector2 `xml:"{cx,cy}"`

	// radius of the circle
	Radius float32 `xml:"r"`
}

func (g *Circle) SVGName() string { return "circle" }

func (g *Circle) Init() {
	g.Radius = 1
}

func (g *Circle) SetNodePos(pos math32.Vector2) {
	g.Pos = pos.SubScalar(g.Radius)
}

func (g *Circle) SetNodeSize(sz math32.Vector2) {
	g.Radius = 0.25 * (sz.X + sz.Y)
}

func (g *Circle) LocalBBox(sv *SVG) math32.Box2 {
	bb := math32.Box2{}
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min = g.Pos.SubScalar(g.Radius + hlw)
	bb.Max = g.Pos.AddScalar(g.Radius + hlw)
	return bb
}

func (g *Circle) Render(sv *SVG) {
	if !g.IsVisible(sv) {
		return
	}
	pc := g.Painter(sv)
	pc.Circle(g.Pos.X, g.Pos.Y, g.Radius)
	pc.Draw()
	g.RenderChildren(sv)
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Circle) ApplyTransform(sv *SVG, xf math32.Matrix2) {
	rot := xf.ExtractRot()
	if rot != 0 {
		g.Paint.Transform.SetMul(xf)
		g.SetProperty("transform", g.Paint.Transform.String())
	} else {
		g.Pos = xf.MulVector2AsPoint(g.Pos)
		scx, scy := xf.ExtractScale()
		g.Radius *= 0.5 * (scx + scy)
		g.GradientApplyTransform(sv, xf)
	}
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Circle) WriteGeom(sv *SVG, dat *[]float32) {
	*dat = slicesx.SetLength(*dat, 3+6)
	(*dat)[0] = g.Pos.X
	(*dat)[1] = g.Pos.Y
	(*dat)[2] = g.Radius
	g.WriteTransform(*dat, 3)
	g.GradientWritePts(sv, dat)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Circle) ReadGeom(sv *SVG, dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Radius = dat[2]
	g.ReadTransform(dat, 3)
	g.GradientReadPts(sv, dat)
}
