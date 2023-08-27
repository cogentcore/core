// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"

	"goki.dev/gi/gi"
	"goki.dev/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Marker represents marker elements that can be drawn along paths (arrow heads, etc)
type Marker struct {
	NodeBase

	// reference position to align the vertex position with, specified in ViewBox coordinates
	RefPos mat32.Vec2 `xml:"{refX,refY}" desc:"reference position to align the vertex position with, specified in ViewBox coordinates"`

	// size of marker to render, in Units units
	Size mat32.Vec2 `xml:"{markerWidth,markerHeight}" desc:"size of marker to render, in Units units"`

	// units to use
	Units MarkerUnits `xml:"markerUnits" desc:"units to use"`

	// viewbox defines the internal coordinate system for the drawing elements within the marker
	ViewBox ViewBox `desc:"viewbox defines the internal coordinate system for the drawing elements within the marker"`

	// orientation of the marker -- either 'auto' or an angle
	Orient string `xml:"orient" desc:"orientation of the marker -- either 'auto' or an angle"`

	// current vertex position
	VertexPos mat32.Vec2 `desc:"current vertex position"`

	// current vertex angle in radians
	VertexAngle float32 `desc:"current vertex angle in radians"`

	// current stroke width
	StrokeWidth float32 `desc:"current stroke width"`

	// net transform computed from settings and current values -- applied prior to rendering
	XForm mat32.Mat2 `desc:"net transform computed from settings and current values -- applied prior to rendering"`

	// effective size for actual rendering
	EffSize mat32.Vec2 `desc:"effective size for actual rendering"`
}

var TypeMarker = kit.Types.AddType(&Marker{}, ki.Props{ki.EnumTypeFlag: gi.TypeNodeFlags})

// AddNewMarker adds a new marker to given parent node, with given name.
func AddNewMarker(parent ki.Ki, name string) *Marker {
	return parent.AddNewChild(TypeMarker, name).(*Marker)
}

func (g *Marker) SVGName() string { return "marker" }

func (g *Marker) EnforceSVGName() bool { return false }

func (g *Marker) CopyFieldsFrom(frm any) {
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

var TypeMarkerUnits = kit.Enums.AddEnumAltLower(MarkerUnitsN, kit.NotBitFlag, gist.StylePropProps, "")

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

func (g *Marker) ComputeBBoxSVG() {
	if g.This() == nil {
		return
	}
	g.BBoxMu.Lock()
	ni := g.This().(NodeSVG)
	g.ObjBBox = ni.BBox2D()
	g.ObjBBox.Canon()
	pbbox := g.Viewport.This().(gi.Node2D).ChildrenBBox2D()
	g.VpBBox = pbbox.Intersect(g.ObjBBox)
	g.BBoxMu.Unlock()
	g.SetWinBBox()

	if gi.Render2DTrace {
		fmt.Printf("Render: %v at %v\n", g.Path(), g.VpBBox)
	}
}
