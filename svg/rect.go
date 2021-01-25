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

// Rect is a SVG rectangle, optionally with rounded corners
type Rect struct {
	NodeBase
	Pos    mat32.Vec2 `xml:"{x,y}" desc:"position of the top-left of the rectangle"`
	Size   mat32.Vec2 `xml:"{width,height}" desc:"size of the rectangle"`
	Radius mat32.Vec2 `xml:"{rx,ry}" desc:"radii for curved corners, as a proportion of width, height"`
}

var KiT_Rect = kit.Types.AddType(&Rect{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewRect adds a new rectangle to given parent node, with given name, pos, and size.
func AddNewRect(parent ki.Ki, name string, x, y, sx, sy float32) *Rect {
	g := parent.AddNewChild(KiT_Rect, name).(*Rect)
	g.Pos.Set(x, y)
	g.Size.Set(sx, sy)
	return g
}

func (g *Rect) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Rect)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Pos = fr.Pos
	g.Size = fr.Size
	g.Radius = fr.Radius
}

func (g *Rect) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := g.Render()
	rs.PushXForm(pc.XForm)
	if g.Radius.X == 0 && g.Radius.Y == 0 {
		pc.DrawRectangle(rs, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
	} else {
		// todo: only supports 1 radius right now -- easy to add another
		pc.DrawRoundedRectangle(rs, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y, g.Radius.X)
	}
	pc.FillStrokeClear(rs)
	g.ComputeBBoxSVG()
	g.Render2DChildren()
	rs.PopXForm()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Rect) ApplyXForm(xf mat32.Mat2) {
	xf = g.MyXForm().Mul(xf)
	g.Pos = xf.MulVec2AsPt(g.Pos)
	g.Size = xf.MulVec2AsVec(g.Size)
}

// ApplyDeltaXForm applies the given 2D delta transform to the geometry of this node
// Changes position according to translation components ONLY
// and changes size according to scale components ONLY
func (g *Rect) ApplyDeltaXForm(xf mat32.Mat2) {
	mxf := g.MyXForm()
	scx, scy := mxf.ExtractScale()
	xf.X0 /= scx
	xf.Y0 /= scy
	g.Pos.X += xf.X0
	g.Pos.Y += xf.Y0
	g.Size.X *= xf.XX
	g.Size.Y *= xf.YY
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Rect) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 6)
	(*dat)[0] = g.Pos.X
	(*dat)[1] = g.Pos.Y
	(*dat)[2] = g.Size.X
	(*dat)[3] = g.Size.Y
	(*dat)[4] = g.Radius.X
	(*dat)[5] = g.Radius.Y
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Rect) ReadGeom(dat []float32) {
	g.Pos.X = dat[0]
	g.Pos.Y = dat[1]
	g.Size.X = dat[2]
	g.Size.Y = dat[3]
	g.Radius.X = dat[4]
	g.Radius.Y = dat[5]
}
