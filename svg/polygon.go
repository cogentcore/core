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

// Polygon is a SVG polygon
type Polygon struct {
	NodeBase
	Points []mat32.Vec2 `xml:"points" desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest, then does a closepath at the end"`
}

var KiT_Polygon = kit.Types.AddType(&Polygon{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewPolygon adds a new polygon to given parent node, with given name and points.
func AddNewPolygon(parent ki.Ki, name string, points []mat32.Vec2) *Polygon {
	g := parent.AddNewChild(KiT_Polygon, name).(*Polygon)
	g.Points = points
	return g
}

func (g *Polygon) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Polygon)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Points = make([]mat32.Vec2, len(fr.Points))
	copy(g.Points, fr.Points)
}

func (g *Polygon) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	sz := len(g.Points)
	if sz < 2 {
		return
	}
	pc := &g.Pnt
	rs := g.Render()
	rs.PushXForm(pc.XForm)
	pc.DrawPolygon(rs, g.Points)
	pc.FillStrokeClear(rs)
	g.ComputeBBoxSVG()

	if mrk := g.Marker("marker-start"); mrk != nil {
		pt := g.Points[0]
		ptn := g.Points[1]
		ang := math32.Atan2(ptn.Y-pt.Y, ptn.X-pt.X)
		mrk.RenderMarker(pt, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := g.Marker("marker-end"); mrk != nil {
		pt := g.Points[sz-1]
		ptp := g.Points[sz-2]
		ang := math32.Atan2(pt.Y-ptp.Y, pt.X-ptp.X)
		mrk.RenderMarker(pt, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := g.Marker("marker-mid"); mrk != nil {
		for i := 1; i < sz-1; i++ {
			pt := g.Points[i]
			ptp := g.Points[i-1]
			ptn := g.Points[i+1]
			ang := 0.5 * (math32.Atan2(pt.Y-ptp.Y, pt.X-ptp.X) + math32.Atan2(ptn.Y-pt.Y, ptn.X-pt.X))
			mrk.RenderMarker(pt, ang, g.Pnt.StrokeStyle.Width.Dots)
		}
	}

	g.Render2DChildren()
	rs.PopXForm()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Polygon) ApplyXForm(xf mat32.Mat2) {
	xf = g.MyXForm().Mul(xf)
	for i, p := range g.Points {
		p = xf.MulVec2AsPt(p)
		g.Points[i] = p
	}
}

// ApplyDeltaXForm applies the given 2D delta transform to the geometry of this node
// Changes position according to translation components ONLY
// and changes size according to scale components ONLY
func (g *Polygon) ApplyDeltaXForm(xf mat32.Mat2) {
	mxf := g.MyXForm()
	scx, scy := mxf.ExtractScale()
	off := mat32.Vec2{xf.X0 / scx, xf.Y0 / scy}
	sc := mat32.Vec2{xf.XX, xf.YY}
	ost := mat32.Vec2{}
	nst := mat32.Vec2{}
	for i, p := range g.Points {
		if i == 0 {
			ost = p
			p.SetAdd(off)
			nst = p
		} else {
			p = nst.Add(p.Sub(ost).Mul(sc))
		}
		g.Points[i] = p
	}
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Polygon) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, len(g.Points)*2)
	for i, p := range g.Points {
		(*dat)[i*2] = p.X
		(*dat)[i*2+1] = p.Y
	}
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Polygon) ReadGeom(dat []float32) {
	for i, p := range g.Points {
		p.X = dat[i*2]
		p.Y = dat[i*2+1]
		g.Points[i] = p
	}
}
