// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"goki.dev/mat32/v2"
)

// Circle is a SVG circle
type Circle struct {
	NodeBase

	// position of the center of the circle
	Pos mat32.Vec2 `xml:"{cx,cy}" set:"-"`

	// radius of the circle
	Radius float32 `xml:"r"`
}

func (g *Circle) SVGName() string { return "circle" }

func (g *Circle) OnInit() {
	g.Radius = 1
}

func (g *Circle) CopyFieldsFrom(frm any) {
	fr := frm.(*Circle)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Pos = fr.Pos
	g.Radius = fr.Radius
}

func (g *Circle) SetPos(pos mat32.Vec2) *Circle {
	g.Pos = pos.SubScalar(g.Radius)
	return g
}

func (g *Circle) SetSize(sz mat32.Vec2) *Circle {
	g.Radius = 0.25 * (sz.X + sz.Y)
	return g
}

func (g *Circle) LocalBBox() mat32.Box2 {
	bb := mat32.Box2{}
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min = g.Pos.SubScalar(g.Radius + hlw)
	bb.Max = g.Pos.AddScalar(g.Radius + hlw)
	return bb
}

func (g *Circle) Render(sv *SVG) {
	vis, pc := g.PushTransform(sv)
	if !vis {
		return
	}
	pc.Lock()
	pc.DrawCircle(g.Pos.X, g.Pos.Y, g.Radius)
	pc.FillStrokeClear()
	pc.Unlock()

	g.BBoxes(sv)
	g.RenderChildren(sv)

	pc.PopTransformLock()
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Circle) ApplyTransform(sv *SVG, xf mat32.Mat2) {
	rot := xf.ExtractRot()
	if rot != 0 || !g.Paint.Transform.IsIdentity() {
		g.Paint.Transform = g.Paint.Transform.Mul(xf)
		g.SetProp("transform", g.Paint.Transform.String())
	} else {
		g.Pos = xf.MulVec2AsPt(g.Pos)
		scx, scy := xf.ExtractScale()
		g.Radius *= 0.5 * (scx + scy)
		g.GradientApplyTransform(sv, xf)
	}
}

// ApplyDeltaTransform applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Circle) ApplyDeltaTransform(sv *SVG, trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	crot := g.Paint.Transform.ExtractRot()
	if rot != 0 || crot != 0 {
		xf, lpt := g.DeltaTransform(trans, scale, rot, pt, false) // exclude self
		mat := g.Paint.Transform.MulCtr(xf, lpt)
		g.Paint.Transform = mat
		g.SetProp("transform", g.Paint.Transform.String())
	} else {
		xf, lpt := g.DeltaTransform(trans, scale, rot, pt, true) // include self
		g.Pos = xf.MulVec2AsPtCtr(g.Pos, lpt)
		scx, scy := xf.ExtractScale()
		g.Radius *= 0.5 * (scx + scy)
		g.GradientApplyTransformPt(sv, xf, lpt)
	}
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Circle) WriteGeom(sv *SVG, dat *[]float32) {
	SetFloat32SliceLen(dat, 3+6)
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
