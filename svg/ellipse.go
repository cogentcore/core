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

func (g *Ellipse) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Ellipse)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Pos = fr.Pos
	g.Radii = fr.Radii
}

func (g *Ellipse) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := g.Render()
	rs.Lock()
	rs.PushXForm(pc.XForm)
	pc.DrawEllipse(rs, g.Pos.X, g.Pos.Y, g.Radii.X, g.Radii.Y)
	pc.FillStrokeClear(rs)
	rs.Unlock()

	g.ComputeBBoxSVG()
	g.Render2DChildren()

	rs.PopXFormLock()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Ellipse) ApplyXForm(xf mat32.Mat2) {
	xf = g.MyXForm().Mul(xf)
	g.Pos = xf.MulVec2AsPt(g.Pos)
	g.Radii = xf.MulVec2AsVec(g.Radii)
}

// ApplyDeltaXForm applies the given 2D delta transform to the geometry of this node
// Changes position according to translation components ONLY
// and changes size according to scale components ONLY
func (g *Ellipse) ApplyDeltaXForm(xf mat32.Mat2) {
	mxf := g.MyXForm()
	scx, scy := mxf.ExtractScale()
	xf.X0 /= scx
	xf.Y0 /= scy
	g.Pos.X += xf.X0
	g.Pos.Y += xf.Y0
	g.Radii.X *= 1 + 0.8*(xf.XX-1)
	g.Radii.Y *= 1 + 0.8*(xf.YY-1)
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Ellipse) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 4)
	(*dat)[0] = g.Pos.X
	(*dat)[1] = g.Pos.Y
	(*dat)[2] = g.Radii.X
	(*dat)[3] = g.Radii.Y
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Ellipse) ReadGeom(dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Radii.X = dat[2]
	g.Radii.Y = dat[3]
}
