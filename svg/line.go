// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/chewxy/math32"
	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Line is a SVG line
type Line struct {
	NodeBase
	Start mat32.Vec2 `xml:"{x1,y1}" desc:"position of the start of the line"`
	End   mat32.Vec2 `xml:"{x2,y2}" desc:"position of the end of the line"`
}

var KiT_Line = kit.Types.AddType(&Line{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewLine adds a new line to given parent node, with given name, st and end.
func AddNewLine(parent ki.Ki, name string, sx, sy, ex, ey float32) *Line {
	g := parent.AddNewChild(KiT_Line, name).(*Line)
	g.Start.Set(sx, sy)
	g.End.Set(ex, ey)
	return g
}

func (g *Line) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Line)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Start = fr.Start
	g.End = fr.End
}

func (g *Line) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := g.Render()
	rs.Lock()
	rs.PushXForm(pc.XForm)
	pc.DrawLine(rs, g.Start.X, g.Start.Y, g.End.X, g.End.Y)
	pc.Stroke(rs)
	g.ComputeBBoxSVG()

	if mrk := g.Marker("marker-start"); mrk != nil {
		ang := math32.Atan2(g.End.Y-g.Start.Y, g.End.X-g.Start.X)
		mrk.RenderMarker(g.Start, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := g.Marker("marker-end"); mrk != nil {
		ang := math32.Atan2(g.End.Y-g.Start.Y, g.End.X-g.Start.X)
		mrk.RenderMarker(g.End, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	rs.Unlock()

	g.Render2DChildren()
	rs.PopXFormLock()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Line) ApplyXForm(xf mat32.Mat2) {
	g.Start = xf.MulVec2AsPt(g.Start)
	g.End = xf.MulVec2AsPt(g.End)
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Line) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	xf, lpt := g.DeltaXForm(trans, scale, rot, pt)
	g.Start = xf.MulVec2AsPtCtr(g.Start, lpt)
	g.End = xf.MulVec2AsPtCtr(g.End, lpt)
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Line) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 4)
	(*dat)[0] = g.Start.X
	(*dat)[1] = g.Start.Y
	(*dat)[2] = g.End.X
	(*dat)[3] = g.End.Y
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Line) ReadGeom(dat []float32) {
	g.Start.X = dat[0]
	g.Start.Y = dat[1]
	g.End.X = dat[2]
	g.End.Y = dat[3]
}
