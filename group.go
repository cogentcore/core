// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"image"

	"goki.dev/mat32/v2"
)

// Group groups together SVG elements.
// Provides a common transform for all group elements
// and shared style properties.
type Group struct {
	NodeBase
}

func (g *Group) SVGName() string { return "g" }

func (g *Group) EnforceSVGName() bool { return false }

// BBoxFromChildren sets the Group BBox from children
func BBoxFromChildren(gi Node) image.Rectangle {
	bb := image.Rectangle{}
	for i, kid := range *gi.Children() {
		kgi := kid.(Node)
		kg := kgi.AsNodeBase()
		if i == 0 {
			bb = kg.BBox
		} else {
			bb = bb.Union(kg.BBox)
		}
	}
	return bb
}

func (g *Group) NodeBBox(sv *SVG) image.Rectangle {
	bb := BBoxFromChildren(g)
	return bb
}

func (g *Group) Render(sv *SVG) {
	pc := &g.Paint
	rs := &sv.RenderState
	if pc.Off || rs == nil {
		return
	}
	rs.PushTransformLock(pc.Transform)

	g.RenderChildren(sv)
	g.BBoxes(sv) // must come after render

	rs.PopTransformLock()
}

// ApplyTransform applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Group) ApplyTransform(sv *SVG, xf mat32.Mat2) {
	g.Paint.Transform = xf.Mul(g.Paint.Transform)
	g.SetProp("transform", g.Paint.Transform.String())
}

// ApplyDeltaTransform applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Group) ApplyDeltaTransform(sv *SVG, trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	xf, lpt := g.DeltaTransform(trans, scale, rot, pt, false) // group does NOT include self
	mat := g.Paint.Transform.MulCtr(xf, lpt)
	g.Paint.Transform = mat
	g.SetProp("transform", g.Paint.Transform.String())
}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Group) WriteGeom(sv *SVG, dat *[]float32) {
	SetFloat32SliceLen(dat, 6)
	g.WriteTransform(*dat, 0)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Group) ReadGeom(sv *SVG, dat []float32) {
	g.ReadTransform(dat, 0)
}
