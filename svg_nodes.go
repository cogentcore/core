// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"strings"

	"github.com/chewxy/math32"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// svg_nodes.go contains the SVG specific rendering nodes, except Path which is in path.go

////////////////////////////////////////////////////////////////////////////////////////
// SVGNodeBase

// SVGNodeBase is an element within the SVG sub-scenegraph -- does not use
// layout logic -- just renders into parent SVG viewport
type SVGNodeBase struct {
	Node2DBase
	Pnt Paint `json:"-" xml:"-" desc:"full paint information for this node"`
}

var KiT_SVGNodeBase = kit.Types.AddType(&SVGNodeBase{}, SVGNodeBaseProps)

var SVGNodeBaseProps = ki.Props{
	"base-type": true, // excludes type from user selections
}

func (g *SVGNodeBase) AsSVGNode() *SVGNodeBase {
	return g
}

// Paint satisfies the painter interface
func (g *SVGNodeBase) Paint() *Paint {
	return &g.Pnt
}

// Init2DBase handles basic node initialization -- Init2D can then do special things
func (g *SVGNodeBase) Init2DBase() {
	g.Viewport = g.ParentViewport()
	g.Pnt.Defaults()
	g.ConnectToViewport()
}

func (g *SVGNodeBase) Init2D() {
	g.Init2DBase()
}

// Style2DSVG styles the Paint values directly from node properties -- for
// SVG-style nodes -- no relevant default styling here -- parents can just set
// props directly as needed
func Style2DSVG(gii Node2D) {
	g := gii.AsNode2D()
	if g.Viewport == nil { // robust
		gii.Init2D()
	}

	pntr, ok := gii.(Painter)
	if !ok {
		return
	}
	pc := pntr.Paint()

	SetCurStyleNode2D(gii)
	defer SetCurStyleNode2D(nil)

	pc.StyleSet = false // this is always first call, restart
	var pagg *ki.Props
	pgi, pg := KiToNode2D(gii.Parent())
	if pgi != nil {
		pagg = &pg.CSSAgg
		if pp, ok := pgi.(Painter); ok {
			pc.CopyStyleFrom(pp.Paint())
			pc.SetStyleProps(pp.Paint(), gii.Properties())
		} else {
			pc.SetStyleProps(nil, gii.Properties())
		}
	} else {
		pc.SetStyleProps(nil, gii.Properties())
	}
	// pc.SetUnitContext(g.Viewport, Vec2DZero)
	pc.ToDots() // we always inherit parent's unit context -- SVG sets it once-and-for-all
	if pagg != nil {
		AggCSS(&g.CSSAgg, *pagg)
	}
	AggCSS(&g.CSSAgg, g.CSS)
	StyleCSSSVG(gii, g.CSSAgg)
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	} else {
		pc.Off = false
	}
}

// ApplyCSSSVG applies css styles to given node, using key to select sub-props
// from overall properties list
func ApplyCSSSVG(node Node2D, key string, css ki.Props) bool {
	pntr, ok := node.(Painter)
	if !ok {
		return false
	}
	pp, got := css[key]
	if !got {
		return false
	}
	pmap, ok := pp.(ki.Props) // must be a props map
	if !ok {
		return false
	}

	pc := pntr.Paint()

	if pgi, _ := KiToNode2D(node.Parent()); pgi != nil {
		if pp, ok := pgi.(Painter); ok {
			pc.SetStyleProps(pp.Paint(), pmap)
		} else {
			pc.SetStyleProps(nil, pmap)
		}
	} else {
		pc.SetStyleProps(nil, pmap)
	}
	return true
}

// StyleCSSSVG applies css style properties to given SVG node, parsing
// out type, .class, and #name selectors
func StyleCSSSVG(node Node2D, css ki.Props) {
	tyn := strings.ToLower(node.Type().Name()) // type is most general, first
	ApplyCSSSVG(node, tyn, css)
	cln := "." + strings.ToLower(node.AsNode2D().Class) // then class
	ApplyCSSSVG(node, cln, css)
	idnm := "#" + strings.ToLower(node.Name()) // then name
	ApplyCSSSVG(node, idnm, css)
}

func (g *SVGNodeBase) Style2D() {
	Style2DSVG(g.This.(Node2D))
}

// ParentSVG returns the parent SVG viewport
func (g *SVGNodeBase) ParentSVG() *SVG {
	pvp := g.ParentViewport()
	for pvp != nil {
		if pvp.IsSVG() {
			return pvp.This.EmbeddedStruct(KiT_SVG).(*SVG)
		}
		pvp = pvp.ParentViewport()
	}
	return nil
}

func (g *SVGNodeBase) Size2D() {
}

func (g *SVGNodeBase) Layout2D(parBBox image.Rectangle) {
}

func (g *SVGNodeBase) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	fbox := rs.Raster.Scanner.GetPathExtent()
	bb := image.Rectangle{Min: image.Point{fbox.Min.X.Floor(), fbox.Min.Y.Floor()},
		Max: image.Point{fbox.Max.X.Ceil(), fbox.Max.Y.Ceil()}}
	return bb
}

func (g *SVGNodeBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
}

func (g *SVGNodeBase) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

// ComputeBBoxSVG is called by default in render to compute bounding boxes for
// gui interaction -- can only be done in rendering because that is when all
// the proper xforms are all in place -- VpBBox is intersected with parent SVG
func (g *SVGNodeBase) ComputeBBoxSVG() {
	g.BBox = g.This.(Node2D).BBox2D()
	g.ObjBBox = g.BBox // no diff
	g.VpBBox = g.Viewport.VpBBox.Intersect(g.ObjBBox)
	g.SetWinBBox()
	rs := &g.Viewport.Render
	rs.ObjBounds = g.ObjBBox
}

func (g *SVGNodeBase) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	// render path elements, then compute bbox, then fill / stroke
	g.ComputeBBoxSVG()
	g.Render2DChildren()
	rs.PopXForm()
}

func (g *SVGNodeBase) Move2D(delta image.Point, parBBox image.Rectangle) {
}

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

func (g *Rect) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	pu := &pc.UnContext
	rs.PushXForm(pc.XForm)
	if g.Radius.X == 0 && g.Radius.Y == 0 {
		pc.DrawRectangle(rs, pu.PxToDots(g.Pos.X), pu.PxToDots(g.Pos.Y), pu.PxToDots(g.Size.X), pu.PxToDots(g.Size.Y))
	} else {
		// todo: only supports 1 radius right now -- easy to add another
		pc.DrawRoundedRectangle(rs, pu.PxToDots(g.Pos.X), pu.PxToDots(g.Pos.Y), pu.PxToDots(g.Size.X), pu.PxToDots(g.Size.Y), pu.PxToDots(g.Radius.X))
	}
	g.ComputeBBoxSVG()
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

func (g *Circle) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	pu := &pc.UnContext
	rs.PushXForm(pc.XForm)
	pc.DrawCircle(rs, pu.PxToDots(g.Pos.X), pu.PxToDots(g.Pos.Y), pu.PxToDots(g.Radius))
	g.ComputeBBoxSVG()
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

func (g *Ellipse) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	pu := &pc.UnContext
	rs.PushXForm(pc.XForm)
	pc.DrawEllipse(rs, pu.PxToDots(g.Pos.X), pu.PxToDots(g.Pos.Y), pu.PxToDots(g.Radii.X), pu.PxToDots(g.Radii.Y))
	g.ComputeBBoxSVG()
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

func (g *Line) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	pu := &pc.UnContext
	rs.PushXForm(pc.XForm)
	pc.DrawLine(rs, pu.PxToDots(g.Start.X), pu.PxToDots(g.Start.Y), pu.PxToDots(g.End.X), pu.PxToDots(g.End.Y))
	g.ComputeBBoxSVG()
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

func (g *Polyline) Render2D() {
	if len(g.Points) < 2 {
		return
	}
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	pc.DrawPolylinePxToDots(rs, g.Points)
	g.ComputeBBoxSVG()
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

func (g *Polygon) Render2D() {
	if len(g.Points) < 2 {
		return
	}
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	pc.DrawPolygonPxToDots(rs, g.Points)
	g.ComputeBBoxSVG()
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
	// todo: fix
	return g.Pnt.BoundingBox(rs, g.Pos.X, g.Pos.Y, g.Pos.X+g.Render.Size.X, g.Pos.Y+g.Render.Size.Y)
}

func (g *SVGText) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	pu := &pc.UnContext
	rs.PushXForm(pc.XForm)
	if len(g.Text) > 0 {
		orgsz := pc.FontStyle.Size
		pos := rs.XForm.TransformPointVec2D(Vec2D{pu.PxToDots(g.Pos.X), pu.PxToDots(g.Pos.Y)})
		rot := math32.Atan2(-rs.XForm.XY, rs.XForm.XX)
		tx := rs.XForm.Rotate(-rot)
		scx, _ := tx.TransformVector(1, 0)
		_, scy := tx.TransformVector(0, 1)
		scalex := scx / scy
		if scalex == 1 {
			scalex = 0
		}
		pc.FontStyle.LoadFont(&pc.UnContext, "") // use original size font
		g.Render.SetString(g.Text, &pc.FontStyle, &pc.TextStyle, true, rot, scalex)
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
