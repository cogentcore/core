// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"log"

	"cogentcore.org/core/math32"
)

// Marker represents marker elements that can be drawn along paths (arrow heads, etc)
type Marker struct {
	NodeBase

	// reference position to align the vertex position with, specified in ViewBox coordinates
	RefPos math32.Vec2 `xml:"{refX,refY}"`

	// size of marker to render, in Units units
	Size math32.Vec2 `xml:"{markerWidth,markerHeight}"`

	// units to use
	Units MarkerUnits `xml:"markerUnits"`

	// viewbox defines the internal coordinate system for the drawing elements within the marker
	ViewBox ViewBox

	// orientation of the marker -- either 'auto' or an angle
	Orient string `xml:"orient"`

	// current vertex position
	VertexPos math32.Vec2

	// current vertex angle in radians
	VertexAngle float32

	// current stroke width
	StrokeWidth float32

	// net transform computed from settings and current values -- applied prior to rendering
	Transform math32.Mat2

	// effective size for actual rendering
	EffSize math32.Vec2
}

func (g *Marker) SVGName() string { return "marker" }

func (g *Marker) EnforceSVGName() bool { return false }

// MarkerUnits specifies units to use for svg marker elements
type MarkerUnits int32 //enum: enum

const (
	StrokeWidth MarkerUnits = iota
	UserSpaceOnUse
	MarkerUnitsN
)

// RenderMarker renders the marker using given vertex position, angle (in
// radians), and stroke width
func (mrk *Marker) RenderMarker(sv *SVG, vertexPos math32.Vec2, vertexAng, strokeWidth float32) {
	mrk.VertexPos = vertexPos
	mrk.VertexAngle = vertexAng
	mrk.StrokeWidth = strokeWidth
	if mrk.Units == StrokeWidth {
		mrk.EffSize = mrk.Size.MulScalar(strokeWidth)
	} else {
		mrk.EffSize = mrk.Size
	}

	ang := vertexAng
	if mrk.Orient != "auto" {
		ang, _ = math32.ParseAngle32(mrk.Orient)
	}
	if mrk.ViewBox.Size == (math32.Vec2{}) {
		mrk.ViewBox.Size = math32.V2(3, 3)
	}
	mrk.Transform = math32.Rotate2D(ang).Scale(mrk.EffSize.X/mrk.ViewBox.Size.X, mrk.EffSize.Y/mrk.ViewBox.Size.Y).Translate(-mrk.RefPos.X, -mrk.RefPos.Y)
	mrk.Transform.X0 += vertexPos.X
	mrk.Transform.Y0 += vertexPos.Y

	mrk.Paint.Transform = mrk.Transform

	// fmt.Println("render marker:", mrk.Name(), strokeWidth)
	mrk.Render(sv)
}

func (g *Marker) Render(sv *SVG) {
	pc := &g.Paint
	rs := &sv.RenderState
	rs.PushTransform(pc.Transform)

	g.RenderChildren(sv)
	g.BBoxes(sv) // must come after render

	rs.PopTransform()
}

func (g *Marker) BBoxes(sv *SVG) {
	if g.This() == nil {
		return
	}
	ni := g.This().(Node)
	g.BBox = ni.NodeBBox(sv)
	g.BBox.Canon()
	g.VisBBox = sv.Geom.SizeRect().Intersect(g.BBox)
}

//////////////////////////////////////////////////////////
// 	SVG marker management

// MarkerByName finds marker property of given name, or generic "marker"
// type, and if set, attempts to find that marker and return it
func (sv *SVG) MarkerByName(n Node, marker string) *Marker {
	url := NodePropURL(n, marker)
	if url == "" {
		url = NodePropURL(n, "marker")
	}
	if url == "" {
		return nil
	}
	mrkn := sv.NodeFindURL(n, url)
	if mrkn == nil {
		return nil
	}
	mrk, ok := mrkn.(*Marker)
	if !ok {
		log.Printf("SVG Found element named: %v but isn't a Marker type, instead is: %T", url, mrkn)
		return nil
	}
	return mrk
}
