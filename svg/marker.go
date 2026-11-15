// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"log"
	"strings"

	"cogentcore.org/core/math32"
)

// Marker represents marker elements that can be drawn along paths (arrow heads, etc)
type Marker struct {
	NodeBase

	// reference position to align the vertex position with, specified in ViewBox coordinates
	RefPos math32.Vector2 `svg:"refX,refY"`

	// size of marker to render, in Units units
	Size math32.Vector2 `svg:"markerWidth,markerHeight"`

	// units to use
	Units MarkerUnits `xml:"markerUnits"`

	// viewbox defines the internal coordinate system for the drawing elements within the marker
	ViewBox ViewBox

	// orientation of the marker -- either 'auto' or an angle
	Orient string `xml:"orient"`

	// start is whether the marker is rendered as a marker-start.
	start bool

	// current vertex position
	VertexPos math32.Vector2

	// current vertex angle in radians
	VertexAngle float32

	// current stroke width
	StrokeWidth float32

	// net transform computed from settings and current values -- applied prior to rendering
	Transform math32.Matrix2

	// effective size for actual rendering
	EffSize math32.Vector2
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

// RenderMarkerStart renders the marker as a marker-start, with the given
// vertex position, angle (in radians), and stroke width: as
// [Marker.RenderMarker], reversed for an orient of auto-start-reverse.
func (g *Marker) RenderMarkerStart(sv *SVG, vertexPos math32.Vector2, vertexAng, strokeWidth float32) {
	g.start = true
	g.RenderMarker(sv, vertexPos, vertexAng, strokeWidth)
	g.start = false
}

// RenderMarker renders the marker using given vertex position, angle (in
// radians), and stroke width
func (g *Marker) RenderMarker(sv *SVG, vertexPos math32.Vector2, vertexAng, strokeWidth float32) {
	g.VertexPos = vertexPos
	g.VertexAngle = vertexAng
	g.StrokeWidth = strokeWidth
	if g.Units == StrokeWidth {
		g.EffSize = g.Size.MulScalar(strokeWidth)
	} else {
		g.EffSize = g.Size
	}

	ang := vertexAng
	switch strings.TrimSpace(g.Orient) {
	case "auto":
	case "auto-start-reverse": // the start marker points backward
		if g.start {
			ang += math32.Pi
		}
	default:
		ang, _ = math32.ParseAngle32(g.Orient)
	}
	if g.ViewBox.Size == (math32.Vector2{}) {
		g.ViewBox.Size = math32.Vec2(3, 3)
	}
	g.Transform = math32.Rotate2D(ang).Scale(g.EffSize.X/g.ViewBox.Size.X, g.EffSize.Y/g.ViewBox.Size.Y).Translate(-g.RefPos.X, -g.RefPos.Y)
	g.Transform.X0 += vertexPos.X
	g.Transform.Y0 += vertexPos.Y

	g.Paint.Transform = g.Transform

	// fmt.Println("render marker:", mrk.Name, strokeWidth, mrk.EffSize, mrk.Transform)
	g.Render(sv)
}

func (g *Marker) BBoxes(sv *SVG, parTransform math32.Matrix2) {
	g.BBoxesFromChildren(sv, parTransform)
}

func (g *Marker) Render(sv *SVG) {
	pc := g.Painter(sv)
	pc.PushContext(&g.Paint, nil)
	g.RenderChildren(sv)
	pc.PopContext()
}

////////  SVG marker management

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
