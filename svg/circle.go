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

// Circle is a SVG circle
type Circle struct {
	NodeBase
	Pos    mat32.Vec2 `xml:"{cx,cy}" desc:"position of the center of the circle"`
	Radius float32    `xml:"r" desc:"radius of the circle"`
}

var KiT_Circle = kit.Types.AddType(&Circle{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewCircle adds a new button to given parent node, with given name, x,y pos, and radius.
func AddNewCircle(parent ki.Ki, name string, x, y, radius float32) *Circle {
	g := parent.AddNewChild(KiT_Circle, name).(*Circle)
	g.Pos.Set(x, y)
	g.Radius = radius
	return g
}

func (g *Circle) SVGName() string { return "circle" }

func (g *Circle) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Circle)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Pos = fr.Pos
	g.Radius = fr.Radius
}

func (g *Circle) SetPos(pos mat32.Vec2) {
	g.Pos = pos.SubScalar(g.Radius)
}

func (g *Circle) SetSize(sz mat32.Vec2) {
	g.Radius = 0.25 * (sz.X + sz.Y)
}

func (g *Circle) SVGLocalBBox() mat32.Box2 {
	bb := mat32.Box2{}
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min = g.Pos.SubScalar(g.Radius + hlw)
	bb.Max = g.Pos.AddScalar(g.Radius + hlw)
	return bb
}

func (g *Circle) Render2D() {
	vis, rs := g.PushXForm()
	if !vis {
		return
	}
	pc := &g.Pnt
	pc.DrawCircle(rs, g.Pos.X, g.Pos.Y, g.Radius)
	pc.FillStrokeClear(rs)

	g.ComputeBBoxSVG()
	g.Render2DChildren()

	rs.PopXFormLock()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Circle) ApplyXForm(xf mat32.Mat2) {
	rot := xf.ExtractRot()
	if rot != 0 || !g.Pnt.XForm.IsIdentity() {
		g.Pnt.XForm = g.Pnt.XForm.Mul(xf)
		g.SetProp("transform", g.Pnt.XForm.String())
		g.GradientApplyXForm(xf)
	} else {
		g.Pos = xf.MulVec2AsPt(g.Pos)
		scx, scy := xf.ExtractScale()
		g.Radius *= 0.5 * (scx + scy)
		g.GradientApplyXForm(xf)
	}
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Circle) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	if rot != 0 {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, false) // exclude self
		mat := g.Pnt.XForm.MulCtr(xf, lpt)
		g.Pnt.XForm = mat
		g.SetProp("transform", g.Pnt.XForm.String())
	} else {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, true) // include self
		g.Pos = xf.MulVec2AsPtCtr(g.Pos, lpt)
		scx, scy := xf.ExtractScale()
		g.Radius *= 0.5 * (scx + scy)
		g.GradientApplyXFormPt(xf, lpt)
	}
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Circle) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 3+6)
	(*dat)[0] = g.Pos.X
	(*dat)[1] = g.Pos.Y
	(*dat)[2] = g.Radius
	g.WriteXForm(*dat, 3)
	g.GradientWritePts(dat)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Circle) ReadGeom(dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Radius = dat[2]
	g.ReadXForm(dat, 3)
	g.GradientReadPts(dat)
}
