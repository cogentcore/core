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

// Ellipse is a SVG ellipse
type Ellipse struct {
	NodeBase
	Pos   mat32.Vec2 `xml:"{cx,cy}" desc:"position of the center of the ellipse"`
	Radii mat32.Vec2 `xml:"{rx,ry}" desc:"radii of the ellipse in the horizontal, vertical axes"`
}

var KiT_Ellipse = kit.Types.AddType(&Ellipse{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewEllipse adds a new button to given parent node, with given name, pos and radii.
func AddNewEllipse(parent ki.Ki, name string, x, y, rx, ry float32) *Ellipse {
	g := parent.AddNewChild(KiT_Ellipse, name).(*Ellipse)
	g.Pos.Set(x, y)
	g.Radii.Set(rx, ry)
	return g
}

func (g *Ellipse) SVGName() string { return "ellipse" }

func (g *Ellipse) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Ellipse)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Pos = fr.Pos
	g.Radii = fr.Radii
}

func (g *Ellipse) SetPos(pos mat32.Vec2) {
	g.Pos = pos.Sub(g.Radii)
}

func (g *Ellipse) SetSize(sz mat32.Vec2) {
	g.Radii = sz.MulScalar(0.5)
}

func (g *Ellipse) SVGLocalBBox() mat32.Box2 {
	bb := mat32.Box2{}
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min = g.Pos.Sub(g.Radii.AddScalar(hlw))
	bb.Max = g.Pos.Add(g.Radii.AddScalar(hlw))
	return bb
}

func (g *Ellipse) Render2D() {
	vis, rs := g.PushXForm()
	if !vis {
		return
	}
	pc := &g.Pnt
	pc.DrawEllipse(rs, g.Pos.X, g.Pos.Y, g.Radii.X, g.Radii.Y)
	pc.FillStrokeClear(rs)

	g.ComputeBBoxSVG()
	g.Render2DChildren()

	rs.PopXFormLock()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Ellipse) ApplyXForm(xf mat32.Mat2) {
	rot := xf.ExtractRot()
	if rot != 0 || !g.Pnt.XForm.IsIdentity() {
		g.Pnt.XForm = g.Pnt.XForm.Mul(xf)
		g.SetProp("transform", g.Pnt.XForm.String())
		g.GradientApplyXForm(xf)
	} else {
		g.Pos = xf.MulVec2AsPt(g.Pos)
		g.Radii = xf.MulVec2AsVec(g.Radii)
		g.GradientApplyXForm(xf)
	}
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Ellipse) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	if rot != 0 {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, false) // exclude self
		mat := g.Pnt.XForm.MulCtr(xf, lpt)
		g.Pnt.XForm = mat
		g.SetProp("transform", g.Pnt.XForm.String())
	} else {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, true) // include self
		g.Pos = xf.MulVec2AsPtCtr(g.Pos, lpt)
		g.Radii = xf.MulVec2AsVec(g.Radii)
		g.GradientApplyXFormPt(xf, lpt)
	}
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Ellipse) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 4+6)
	(*dat)[0] = g.Pos.X
	(*dat)[1] = g.Pos.Y
	(*dat)[2] = g.Radii.X
	(*dat)[3] = g.Radii.Y
	g.WriteXForm(*dat, 4)
	g.GradientWritePts(dat)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Ellipse) ReadGeom(dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Radii.X = dat[2]
	g.Radii.Y = dat[3]
	g.ReadXForm(dat, 4)
	g.GradientReadPts(dat)
}
