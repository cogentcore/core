// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi"
	"github.com/goki/ki/kit"
)

// Marker represents marker elements that can be drawn along paths (arrow heads, etc)
type Marker struct {
	SVGNodeBase
	RefPos      gi.Vec2D    `xml:"{refX,refY}" desc:"reference position to align the vertex position with, specified in ViewBox coordinates"`
	Size        gi.Vec2D    `xml:"{markerWidth,markerHeight}" desc:"size of marker to render, in Units units"`
	Units       MarkerUnits `xml:"markerUnits" desc:"units to use"`
	ViewBox     ViewBox     `desc:"viewbox defines the internal coordinate system for the drawing elements within the marker"`
	Orient      string      `xml:"orient" desc:"orientation of the marker -- either 'auto' or an angle"`
	VertexPos   gi.Vec2D    `desc:"current vertex position"`
	VertexAngle float32     `desc:"current vertex angle in radians"`
	StrokeWidth float32     `desc:"current stroke width"`
	XForm       gi.Matrix2D `desc:"net transform computed from settings and current values -- applied prior to rendering"`
	EffSize     gi.Vec2D    `desc:"effective size for actual rendering"`
}

var KiT_Marker = kit.Types.AddType(&Marker{}, nil)

// MarkerUnits specifies units to use for svg marker elements
type MarkerUnits int32

const (
	StrokeWidth MarkerUnits = iota
	UserSpaceOnUse
	MarkerUnitsN
)

//go:generate stringer -type=MarkerUnits

var KiT_MarkerUnits = kit.Enums.AddEnumAltLower(MarkerUnitsN, false, gi.StylePropProps, "")

func (ev MarkerUnits) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *MarkerUnits) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// RenderMarker renders the marker using given vertex postion, angle (in
// radians), and stroke width
func (mrk *Marker) RenderMarker(vertexPos gi.Vec2D, vertexAng, strokeWidth float32) {
	mrk.VertexPos = vertexPos
	mrk.VertexAngle = vertexAng
	mrk.StrokeWidth = strokeWidth
	if mrk.Units == StrokeWidth {
		mrk.EffSize = mrk.Size.MulVal(strokeWidth)
	} else {
		mrk.EffSize = mrk.Size
	}

	ang := vertexAng
	if mrk.Orient != "auto" {
		ang, _ = gi.ParseAngle32(mrk.Orient)
	}

	sc := gi.Scale2D(mrk.EffSize.X/mrk.ViewBox.Size.X, mrk.EffSize.Y/mrk.ViewBox.Size.Y)
	mrk.XForm = sc.Multiply(gi.Rotate2D(ang))
	roff := sc.TransformPointVec2D(mrk.RefPos)
	mrk.XForm.X0 = vertexPos.X - roff.X
	mrk.XForm.Y0 = vertexPos.Y - roff.Y

	mrk.Pnt.XForm = mrk.XForm

	mrk.Render2D()
}
