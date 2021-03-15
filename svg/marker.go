// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Marker represents marker elements that can be drawn along paths (arrow heads, etc)
type Marker struct {
	NodeBase
	RefPos      mat32.Vec2  `xml:"{refX,refY}" desc:"reference position to align the vertex position with, specified in ViewBox coordinates"`
	Size        mat32.Vec2  `xml:"{markerWidth,markerHeight}" desc:"size of marker to render, in Units units"`
	Units       MarkerUnits `xml:"markerUnits" desc:"units to use"`
	ViewBox     ViewBox     `desc:"viewbox defines the internal coordinate system for the drawing elements within the marker"`
	Orient      string      `xml:"orient" desc:"orientation of the marker -- either 'auto' or an angle"`
	VertexPos   mat32.Vec2  `desc:"current vertex position"`
	VertexAngle float32     `desc:"current vertex angle in radians"`
	StrokeWidth float32     `desc:"current stroke width"`
	XForm       mat32.Mat2  `desc:"net transform computed from settings and current values -- applied prior to rendering"`
	EffSize     mat32.Vec2  `desc:"effective size for actual rendering"`
}

var KiT_Marker = kit.Types.AddType(&Marker{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewMarker adds a new marker to given parent node, with given name.
func AddNewMarker(parent ki.Ki, name string) *Marker {
	return parent.AddNewChild(KiT_Marker, name).(*Marker)
}

func (g *Marker) SVGName() string { return "marker" }

func (g *Marker) EnforceSVGName() bool { return false }

func (g *Marker) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Marker)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.RefPos = fr.RefPos
	g.Size = fr.Size
	g.Units = fr.Units
	g.ViewBox = fr.ViewBox
	g.Orient = fr.Orient
	g.VertexPos = fr.VertexPos
	g.VertexAngle = fr.VertexAngle
	g.StrokeWidth = fr.StrokeWidth
	g.XForm = fr.XForm
	g.EffSize = fr.EffSize
}

// MarkerUnits specifies units to use for svg marker elements
type MarkerUnits int32

const (
	StrokeWidth MarkerUnits = iota
	UserSpaceOnUse
	MarkerUnitsN
)

//go:generate stringer -type=MarkerUnits

var KiT_MarkerUnits = kit.Enums.AddEnumAltLower(MarkerUnitsN, kit.NotBitFlag, gist.StylePropProps, "")

func (ev MarkerUnits) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *MarkerUnits) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// RenderMarker renders the marker using given vertex position, angle (in
// radians), and stroke width
func (mrk *Marker) RenderMarker(vertexPos mat32.Vec2, vertexAng, strokeWidth float32) {
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
		ang, _ = mat32.ParseAngle32(mrk.Orient)
	}
	if mrk.ViewBox.Size.IsNil() {
		mrk.ViewBox.Size = mat32.Vec2{3, 3}
	}
	mrk.XForm = mat32.Rotate2D(ang).Scale(mrk.EffSize.X/mrk.ViewBox.Size.X, mrk.EffSize.Y/mrk.ViewBox.Size.Y).Translate(-mrk.RefPos.X, -mrk.RefPos.Y)
	mrk.XForm.X0 += vertexPos.X
	mrk.XForm.Y0 += vertexPos.Y

	mrk.Pnt.XForm = mrk.XForm

	mrk.Render2D()
}

func (g *Marker) Render2D() {
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
