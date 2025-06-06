// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/sides"
)

// Rect is a SVG rectangle, optionally with rounded corners
type Rect struct {
	NodeBase

	// position of the top-left of the rectangle
	Pos math32.Vector2 `xml:"{x,y}"`

	// size of the rectangle
	Size math32.Vector2 `xml:"{width,height}"`

	// radii for curved corners. only rx is used for now.
	Radius math32.Vector2 `xml:"{rx,ry}"`
}

func (g *Rect) SVGName() string { return "rect" }

func (g *Rect) Init() {
	g.NodeBase.Init()
	g.Size.Set(1, 1)
}

func (g *Rect) SetNodePos(pos math32.Vector2) {
	g.Pos = pos
}

func (g *Rect) SetNodeSize(sz math32.Vector2) {
	g.Size = sz
}

func (g *Rect) LocalBBox(sv *SVG) math32.Box2 {
	bb := math32.Box2{}
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min = g.Pos.SubScalar(hlw)
	bb.Max = g.Pos.Add(g.Size).AddScalar(hlw)
	return bb
}

func (g *Rect) Render(sv *SVG) {
	if !g.IsVisible(sv) {
		return
	}
	pc := g.Painter(sv)
	if g.Radius.X == 0 && g.Radius.Y == 0 {
		pc.Rectangle(g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
	} else {
		// todo: only supports 1 radius right now -- easy to add another
		// the Painter also support different radii for each corner but not rx, ry at this point,
		// although that would be easy to add TODO:
		pc.RoundedRectangleSides(g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y, sides.NewFloats(g.Radius.X))
	}
	pc.Draw()
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Rect) ApplyTransform(sv *SVG, xf math32.Matrix2) {
	rot := xf.ExtractRot()
	if rot != 0 {
		g.Paint.Transform.SetMul(xf)
		g.SetTransformProperty()
	} else {
		g.Pos = xf.MulVector2AsPoint(g.Pos)
		g.Size = xf.MulVector2AsVector(g.Size)
		g.GradientApplyTransform(sv, xf)
	}
}
