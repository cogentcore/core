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

func (g *Circle) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Circle)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Pos = fr.Pos
	g.Radius = fr.Radius
}

func (g *Circle) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := g.Render()
	rs.Lock()
	rs.PushXForm(pc.XForm)
	pc.DrawCircle(rs, g.Pos.X, g.Pos.Y, g.Radius)
	pc.FillStrokeClear(rs)
	rs.Unlock()

	g.ComputeBBoxSVG()
	g.Render2DChildren()

	rs.PopXFormLock()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Circle) ApplyXForm(xf mat32.Mat2) {
	xf = g.MyXForm().Mul(xf)
	g.Pos = xf.MulVec2AsPt(g.Pos)
	scx, scy := xf.ExtractScale()
	g.Radius *= 0.5 * (scx + scy)
}

// ApplyDeltaXForm applies the given 2D delta transform to the geometry of this node
// Changes position according to translation components ONLY
// and changes size according to scale components ONLY
func (g *Circle) ApplyDeltaXForm(xf mat32.Mat2) {
	mxf := g.MyXForm()
	scx, scy := mxf.ExtractScale()
	xf.X0 /= scx
	xf.Y0 /= scy
	g.Pos.X += xf.X0
	g.Pos.Y += xf.Y0
	g.Radius *= 1 + 0.5*(0.5*(xf.XX+xf.YY)-1)
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Circle) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 3)
	(*dat)[0] = g.Pos.X
	(*dat)[1] = g.Pos.Y
	(*dat)[2] = g.Radius
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Circle) ReadGeom(dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Radius = dat[2]
}
