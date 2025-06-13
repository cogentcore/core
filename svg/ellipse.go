// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/math32"
)

// Ellipse is a SVG ellipse
type Ellipse struct {
	NodeBase

	// position of the center of the ellipse
	Pos math32.Vector2 `xml:"{cx,cy}"`

	// radii of the ellipse in the horizontal, vertical axes
	Radii math32.Vector2 `xml:"{rx,ry}"`
}

func (g *Ellipse) SVGName() string { return "ellipse" }

func (g *Ellipse) Init() {
	g.NodeBase.Init()
	g.Radii.Set(1, 1)
}

func (g *Ellipse) SetNodePos(pos math32.Vector2) {
	g.Pos = pos.Sub(g.Radii)
}

func (g *Ellipse) SetNodeSize(sz math32.Vector2) {
	g.Radii = sz.MulScalar(0.5)
}

func (g *Ellipse) LocalBBox(sv *SVG) math32.Box2 {
	bb := math32.Box2{}
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min = g.Pos.Sub(g.Radii.AddScalar(hlw))
	bb.Max = g.Pos.Add(g.Radii.AddScalar(hlw))
	return bb
}

func (g *Ellipse) Render(sv *SVG) {
	if !g.IsVisible(sv) {
		return
	}
	pc := g.Painter(sv)
	pc.Ellipse(g.Pos.X, g.Pos.Y, g.Radii.X, g.Radii.Y)
	pc.Draw()
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Ellipse) ApplyTransform(sv *SVG, xf math32.Matrix2) {
	rot := xf.ExtractRot()
	if rot != 0 || !g.Paint.Transform.IsIdentity() {
		g.Paint.Transform.SetMul(xf)
		g.SetTransformProperty()
	} else {
		// todo: this is not the correct transform:
		g.Pos = xf.MulVector2AsPoint(g.Pos)
		g.Radii = xf.MulVector2AsVector(g.Radii)
		g.GradientApplyTransform(sv, xf)
	}
}
