// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image"

	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Group groups together SVG elements -- doesn't do much but provide a
// locus for properties etc
type Group struct {
	NodeBase
}

var KiT_Group = kit.Types.AddType(&Group{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewGroup adds a new group to given parent node, with given name.
func AddNewGroup(parent ki.Ki, name string) *Group {
	return parent.AddNewChild(KiT_Group, name).(*Group)
}

func (g *Group) SVGName() string { return "g" }

func (g *Group) EnforceSVGName() bool { return false }

func (g *Group) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Group)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
}

// BBoxFromChildren sets the Group BBox from children
func BBoxFromChildren(gii gi.Node2D) image.Rectangle {
	bb := image.ZR
	for i, kid := range *gii.Children() {
		_, gi := gi.KiToNode2D(kid)
		if gi != nil {
			if i == 0 {
				bb = gi.ObjBBox
			} else {
				bb = bb.Union(gi.ObjBBox)
			}
		}
	}
	return bb
}

func (g *Group) BBox2D() image.Rectangle {
	bb := BBoxFromChildren(g)
	return bb
}

func (g *Group) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	pc := &g.Pnt
	rs := g.Render()
	if rs == nil {
		return
	}
	rs.PushXFormLock(pc.XForm)

	g.Render2DChildren()
	g.ComputeBBoxSVG() // must come after render

	rs.PopXFormLock()
}

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Group) ApplyXForm(xf mat32.Mat2) {
	g.Pnt.XForm = xf.Mul(g.Pnt.XForm)
	g.SetProp("transform", g.Pnt.XForm.String())
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Group) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	xf, lpt := g.DeltaXForm(trans, scale, rot, pt, false) // group does NOT include self
	mat := g.Pnt.XForm.MulCtr(xf, lpt)
	g.Pnt.XForm = mat
	g.SetProp("transform", g.Pnt.XForm.String())
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Group) WriteGeom(dat *[]float32) {
	SetFloat32SliceLen(dat, 6)
	g.WriteXForm(*dat, 0)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Group) ReadGeom(dat []float32) {
	g.ReadXForm(dat, 0)
}
