// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/chewxy/math32"
	"github.com/goki/gi/units"
	"github.com/goki/ki/kit"
)

// svg_nodes.go contains the SVG specific rendering nodes, except Path which is in path.go

////////////////////////////////////////////////////////////////////////////////////////
// SVGGroup

// SVGGroup groups together SVG elements -- doesn't do much but provide a
// locus for properties etc
type SVGGroup struct {
	SVGNodeBase
}

var KiT_SVGGroup = kit.Types.AddType(&SVGGroup{}, nil)

// BBoxFromChildren sets the Group BBox from children
func (g *SVGGroup) BBoxFromChildren() image.Rectangle {
	bb := image.ZR
	for i, kid := range g.Kids {
		_, gi := KiToNode2D(kid)
		if gi != nil {
			if i == 0 {
				bb = gi.BBox
			} else {
				bb = bb.Union(gi.BBox)
			}
		}
	}
	return bb
}

func (g *SVGGroup) BBox2D() image.Rectangle {
	bb := g.BBoxFromChildren()
	return bb
}

func (g *SVGGroup) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	g.Render2DChildren()
	g.ComputeBBoxSVG()
	rs.PopXForm()
}

////////////////////////////////////////////////////////////////////////////////////////
// Rect

// 2D rectangle, optionally with rounded corners
type Rect struct {
	SVGNodeBase
	Pos    Vec2D `xml:"{x,y}" desc:"position of the top-left of the rectangle"`
	Size   Vec2D `xml:"{width,height}" desc:"size of the rectangle"`
	Radius Vec2D `xml:"{rx,ry}" desc:"radii for curved corners, as a proportion of width, height"`
}

var KiT_Rect = kit.Types.AddType(&Rect{}, nil)

func (g *Rect) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	bb := g.Pnt.BoundingBox(rs, g.Pos.X, g.Pos.Y, g.Pos.X+g.Size.X, g.Pos.Y+g.Size.Y)
	return bb
}

func (g *Rect) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	g.ComputeBBoxSVG()
	if g.Radius.X == 0 && g.Radius.Y == 0 {
		pc.DrawRectangle(rs, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
	} else {
		// todo: only supports 1 radius right now -- easy to add another
		pc.DrawRoundedRectangle(rs, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y, g.Radius.X)
	}
	pc.FillStrokeClear(rs)
	g.Render2DChildren()
	rs.PopXForm()
}

////////////////////////////////////////////////////////////////////////////////////////
// Circle

// 2D circle
type Circle struct {
	SVGNodeBase
	Pos    Vec2D   `xml:"{cx,cy}" desc:"position of the center of the circle"`
	Radius float32 `xml:"r" desc:"radius of the circle"`
}

var KiT_Circle = kit.Types.AddType(&Circle{}, nil)

func (g *Circle) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	bb := g.Pnt.BoundingBox(rs, g.Pos.X-g.Radius, g.Pos.Y-g.Radius, g.Pos.X+g.Radius, g.Pos.Y+g.Radius)
	return bb
}

func (g *Circle) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	g.ComputeBBoxSVG()
	pc.DrawCircle(rs, g.Pos.X, g.Pos.Y, g.Radius)
	pc.FillStrokeClear(rs)
	g.Render2DChildren()
	rs.PopXForm()
}

////////////////////////////////////////////////////////////////////////////////////////
// Ellipse

// 2D ellipse
type Ellipse struct {
	SVGNodeBase
	Pos   Vec2D `xml:"{cx,cy}" desc:"position of the center of the ellipse"`
	Radii Vec2D `xml:"{rx,ry}" desc:"radii of the ellipse in the horizontal, vertical axes"`
}

var KiT_Ellipse = kit.Types.AddType(&Ellipse{}, nil)

func (g *Ellipse) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	bb := g.Pnt.BoundingBox(rs, g.Pos.X-g.Radii.X, g.Pos.Y-g.Radii.Y, g.Pos.X+g.Radii.X, g.Pos.Y+g.Radii.Y)
	return bb
}

func (g *Ellipse) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	g.ComputeBBoxSVG()
	pc.DrawEllipse(rs, g.Pos.X, g.Pos.Y, g.Radii.X, g.Radii.Y)
	pc.FillStrokeClear(rs)
	g.Render2DChildren()
	rs.PopXForm()
}

////////////////////////////////////////////////////////////////////////////////////////
// Line

// a 2D line
type Line struct {
	SVGNodeBase
	Start Vec2D `xml:"{x1,y1}" desc:"position of the start of the line"`
	End   Vec2D `xml:"{x2,y2}" desc:"position of the end of the line"`
}

var KiT_Line = kit.Types.AddType(&Line{}, nil)

func (g *Line) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	bb := g.Pnt.BoundingBox(rs, g.Start.X, g.Start.Y, g.End.X, g.End.Y).Canon()
	return bb
}

func (g *Line) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	g.ComputeBBoxSVG()
	pc.DrawLine(rs, g.Start.X, g.Start.Y, g.End.X, g.End.Y)
	pc.Stroke(rs)
	g.Render2DChildren()
	rs.PopXForm()
}

////////////////////////////////////////////////////////////////////////////////////////
// Polyline

// 2D Polyline
type Polyline struct {
	SVGNodeBase
	Points []Vec2D `xml:"points" desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest"`
}

var KiT_Polyline = kit.Types.AddType(&Polyline{}, nil)

func (g *Polyline) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	bb := g.Pnt.BoundingBoxFromPoints(rs, g.Points)
	return bb
}

func (g *Polyline) Render2D() {
	if len(g.Points) < 2 {
		return
	}
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	g.ComputeBBoxSVG()
	pc.DrawPolyline(rs, g.Points)
	pc.FillStrokeClear(rs)
	g.Render2DChildren()
	rs.PopXForm()
}

////////////////////////////////////////////////////////////////////////////////////////
// Polygon

// 2D Polygon
type Polygon struct {
	SVGNodeBase
	Points []Vec2D `xml:"points" desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest, then does a closepath at the end"`
}

var KiT_Polygon = kit.Types.AddType(&Polygon{}, nil)

func (g *Polygon) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	bb := g.Pnt.BoundingBoxFromPoints(rs, g.Points)
	return bb
}

func (g *Polygon) Render2D() {
	if len(g.Points) < 2 {
		return
	}
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	g.ComputeBBoxSVG()
	pc.DrawPolygon(rs, g.Points)
	pc.FillStrokeClear(rs)
	g.Render2DChildren()
	rs.PopXForm()
}

////////////////////////////////////////////////////////////////////////////////////////
// SVGText Node

// todo: lots of work likely needed on laying-out text in proper way
// https://www.w3.org/TR/SVG2/text.html#GlyphsMetrics
// todo: tspan element

// SVGText renders 2D text within an SVG -- it handles both text and tspan
// elements (a tspan is just nested under a parent text)
type SVGText struct {
	SVGNodeBase
	Pos          Vec2D      `xml:"{x,y}" desc:"position of the left, baseline of the text"`
	Width        float32    `xml:"width" desc:"width of text to render if using word-wrapping"`
	Text         string     `xml:"text" desc:"text string to render"`
	Render       TextRender `xml:"-" json:"-" desc:"render version of text"`
	CharPosX     []float32  `desc:"character positions along X axis, if specified"`
	CharPosY     []float32  `desc:"character positions along Y axis, if specified"`
	CharPosDX    []float32  `desc:"character delta-positions along X axis, if specified"`
	CharPosDY    []float32  `desc:"character delta-positions along Y axis, if specified"`
	CharRots     []float32  `desc:"character rotations, if specified"`
	TextLength   float32    `desc:"author's computed text length, if specified -- we attempt to match"`
	AdjustGlyphs bool       `desc:"in attempting to match TextLength, should we adjust glyphs in addition to spacing?"`
}

var KiT_SVGText = kit.Types.AddType(&SVGText{}, nil)

func (g *SVGText) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	return g.Pnt.BoundingBox(rs, g.Pos.X, g.Pos.Y, g.Pos.X+g.Render.Size.X, g.Pos.Y+g.Render.Size.Y)
}

func (g *SVGText) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	if len(g.Text) > 0 {
		orgsz := pc.FontStyle.Size
		pos := rs.XForm.TransformPointVec2D(g.Pos)
		rot := math32.Atan2(-rs.XForm.XY, rs.XForm.XX)
		tx := rs.XForm.Rotate(-rot)
		scx, _ := tx.TransformVector(1, 0)
		_, scy := tx.TransformVector(0, 1)
		scalex := scx / scy
		if scalex == 1 {
			scalex = 0
		}
		pc.FontStyle.LoadFont(&pc.UnContext, "") // use original size font
		g.Render.SetString(g.Text, &pc.FontStyle, false, rot, scalex)
		pc.FontStyle.Size = units.Value{orgsz.Val * scy, orgsz.Un, orgsz.Dots * scy} // rescale by y
		pc.FontStyle.LoadFont(&pc.UnContext, "")
		sr := &(g.Render.Spans[0])
		sr.Render[0].Face = pc.FontStyle.Face // upscale
		for i := range sr.Text {
			sr.Render[i].RelPos = rs.XForm.TransformVectorVec2D(sr.Render[i].RelPos)
			sr.Render[i].Size.Y *= scy
			sr.Render[i].Size.X *= scx
		}
		pc.FontStyle.Size = orgsz
		g.Render.Render(rs, pos)
		g.ComputeBBoxSVG()
	}
	g.Render2DChildren()
	rs.PopXForm()
}

/////////////////////////////////////////////////////////////////////////////////
// Misc Nodes

// Gradient is used for holding a specified color gradient -- name is id for
// lookup in url
type Gradient struct {
	SVGNodeBase
	Grad ColorSpec `desc:"the color gradient"`
}

var KiT_Gradient = kit.Types.AddType(&Gradient{}, nil)

// ClipPath is used for holding a path that renders as a clip path
type ClipPath struct {
	SVGNodeBase
}

var KiT_ClipPath = kit.Types.AddType(&ClipPath{}, nil)

// Marker represents marker elements that can be drawn along paths (arrow heads, etc)
type Marker struct {
	SVGNodeBase
}

var KiT_Marker = kit.Types.AddType(&Marker{}, nil)

// SVGFlow represents SVG flow* elements
type SVGFlow struct {
	SVGNodeBase
	FlowType string
}

var KiT_SVGFlow = kit.Types.AddType(&SVGFlow{}, nil)

// SVGFilter represents SVG filter* elements
type SVGFilter struct {
	SVGNodeBase
	FilterType string
}

var KiT_SVGFilter = kit.Types.AddType(&SVGFilter{}, nil)
