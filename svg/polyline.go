// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Polyline is a SVG multi-line shape
type Polyline struct {
	NodeBase
	Points []mat32.Vec2 `xml:"points" desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest"`
}

var KiT_Polyline = kit.Types.AddType(&Polyline{}, nil)

// AddNewPolyline adds a new polyline to given parent node, with given name and points.
func AddNewPolyline(parent ki.Ki, name string, points []mat32.Vec2) *Polyline {
	g := parent.AddNewChild(KiT_Polyline, name).(*Polyline)
	g.Points = points
	return g
}

func (g *Polyline) SVGName() string { return "polyline" }

func (g *Polyline) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Polyline)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Points = make([]mat32.Vec2, len(fr.Points))
	copy(g.Points, fr.Points)
}

func (g *Polyline) SetPos(pos mat32.Vec2) {
	// todo: set offset relative to bbox
}

func (g *Polyline) SetSize(sz mat32.Vec2) {
	// todo: scale bbox
}

func (g *Polyline) SVGLocalBBox() mat32.Box2 {
	bb := mat32.NewEmptyBox2()
	for _, pt := range g.Points {
		bb.ExpandByPoint(pt)
	}
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min.SetSubScalar(hlw)
	bb.Max.SetAddScalar(hlw)
	return bb
}

func (g *Polyline) Render2D() {
	sz := len(g.Points)
	if sz < 2 {
		return
	}
	vis, rs := g.PushXForm()
	if !vis {
		return
	}
	pc := &g.Pnt
	pc.DrawPolyline(rs, g.Points)
	pc.FillStrokeClear(rs)
	g.ComputeBBoxSVG()

	if mrk := MarkerByName(g, "marker-start"); mrk != nil {
		pt := g.Points[0]
		ptn := g.Points[1]
		ang := mat32.Atan2(ptn.Y-pt.Y, ptn.X-pt.X)
		mrk.RenderMarker(pt, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := MarkerByName(g, "marker-end"); mrk != nil {
		pt := g.Points[sz-1]
		ptp := g.Points[sz-2]
		ang := mat32.Atan2(pt.Y-ptp.Y, pt.X-ptp.X)
		mrk.RenderMarker(pt, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := MarkerByName(g, "marker-mid"); mrk != nil {
		for i := 1; i < sz-1; i++ {
			pt := g.Points[i]
			ptp := g.Points[i-1]
			ptn := g.Points[i+1]
			ang := 0.5 * (mat32.Atan2(pt.Y-ptp.Y, pt.X-ptp.X) + mat32.Atan2(ptn.Y-pt.Y, ptn.X-pt.X))
			mrk.RenderMarker(pt, ang, g.Pnt.StrokeStyle.Width.Dots)
		}
	}

	g.Render2DChildren()
	rs.PopXFormLock()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Polyline) ApplyXForm(xf mat32.Mat2) {
	rot := xf.ExtractRot()
	if rot != 0 || !g.Pnt.XForm.IsIdentity() {
		g.Pnt.XForm = g.Pnt.XForm.Mul(xf)
		g.SetProp("transform", g.Pnt.XForm.String())
		g.GradientApplyXForm(xf)
	} else {
		for i, p := range g.Points {
			p = xf.MulVec2AsPt(p)
			g.Points[i] = p
		}
		g.GradientApplyXForm(xf)
	}
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Polyline) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	if rot != 0 {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, false) // exclude self
		mat := g.Pnt.XForm.MulCtr(xf, lpt)
		g.Pnt.XForm = mat
		g.SetProp("transform", g.Pnt.XForm.String())
	} else {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, true) // include self
		for i, p := range g.Points {
			p = xf.MulVec2AsPtCtr(p, lpt)
			g.Points[i] = p
		}
		g.GradientApplyXFormPt(xf, lpt)
	}
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Polyline) WriteGeom(dat *[]float32) {
	sz := len(g.Points) * 2
	SetFloat32SliceLen(dat, sz+6)
	for i, p := range g.Points {
		(*dat)[i*2] = p.X
		(*dat)[i*2+1] = p.Y
	}
	g.WriteXForm(*dat, sz)
	g.GradientWritePts(dat)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Polyline) ReadGeom(dat []float32) {
	sz := len(g.Points) * 2
	for i, p := range g.Points {
		p.X = dat[i*2]
		p.Y = dat[i*2+1]
		g.Points[i] = p
	}
	g.ReadXForm(dat, sz)
	g.GradientReadPts(dat)
}
