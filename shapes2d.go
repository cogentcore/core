// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"strconv"
	"strings"
	"unicode"

	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// shapes2d contains all the SVG-based objects for drawing shapes, paths, etc

////////////////////////////////////////////////////////////////////////////////////////
// SVGNodeBase

// SVGNodeBase is an element within the SVG sub-scenegraph -- does not use
// layout logic -- just renders into parent SVG viewport
type SVGNodeBase struct {
	Node2DBase
	Paint Paint `json:"-" xml:"-" desc:"full paint information for this node"`
}

var KiT_SVGNodeBase = kit.Types.AddType(&SVGNodeBase{}, SVGNodeBaseProps)

var SVGNodeBaseProps = ki.Props{
	"base-type": true, // excludes type from user selections
}

func (g *SVGNodeBase) AsSVGNode() *SVGNodeBase {
	return g
}

// Init2DBase handles basic node initialization -- Init2D can then do special things
func (g *SVGNodeBase) Init2DBase() {
	g.Viewport = g.ParentViewport()
	g.Paint.Defaults()
	g.ConnectToViewport()
}

func (g *SVGNodeBase) Init2D() {
	g.Init2DBase()
}

// Style2DSVG styles the Paint values directly from node properties -- for
// SVG-style nodes -- no relevant default styling here -- parents can just set
// props directly as needed
func (g *SVGNodeBase) Style2DSVG() {
	gii, _ := g.This.(Node2D)
	if g.Viewport == nil { // robust
		gii.Init2D()
	}
	SetCurStyleNode2D(gii)
	defer SetCurStyleNode2D(nil)
	var pagg *ki.Props
	pg := g.CopyParentPaint() // svg always inherits all paint settings from parent
	g.Paint.StyleSet = false  // this is always first call, restart
	if pg != nil {
		g.Paint.SetStyleProps(&pg.Paint, g.Properties())
		pagg = &pg.CSSAgg
	} else {
		g.Paint.SetStyleProps(nil, g.Properties())
	}
	g.Paint.SetUnitContext(g.Viewport, Vec2DZero)

	if pagg != nil {
		AggCSS(&g.CSSAgg, *pagg)
	}
	AggCSS(&g.CSSAgg, g.CSS)
	StyleCSSSVG(gii, g.CSSAgg)
}

// ApplyCSSSVG applies css styles to given node, using key to select sub-props
// from overall properties list
func ApplyCSSSVG(node Node2D, key string, css ki.Props) bool {
	pp, got := css[key]
	if !got {
		return false
	}
	pmap, ok := pp.(ki.Props) // must be a props map
	if !ok {
		return false
	}

	nb := node.AsSVGNode()
	if nb == nil {
		return false
	}
	if psvgi, _ := KiToNode2D(node.Parent()); psvgi != nil {
		if psvg := psvgi.AsSVGNode(); psvg != nil {
			nb.Paint.SetStyleProps(&psvg.Paint, pmap)
		} else {
			nb.Paint.SetStyleProps(nil, pmap)
		}
	} else {
		nb.Paint.SetStyleProps(nil, pmap)
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
	g.Style2DSVG()
	pc := &g.Paint
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	}
}

// CopyParentPaint copy our paint from our parents -- called during Style for
// SVG
func (g *SVGNodeBase) CopyParentPaint() *SVGNodeBase {
	pgi, _ := KiToNode2D(g.Par)
	if pgi == nil {
		return nil
	}
	psvg := pgi.AsSVGNode()
	if psvg == nil {
		return nil
	}
	g.Paint.CopyFrom(&psvg.Paint)
	return psvg
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
	return g.BBox
}

func (g *SVGNodeBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
}

func (g *SVGNodeBase) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

func (g *SVGNodeBase) Render2D() {
	g.Render2DChildren()
}

func (g *SVGNodeBase) ReRender2D() (node Node2D, layout bool) {
	svg := g.ParentSVG()
	if svg != nil {
		node = svg
	} else {
		node = g.This.(Node2D) // no other option..
	}
	layout = false
	return
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
	rs.PushXForm(g.Paint.XForm)
	bb := g.Paint.BoundingBox(rs, g.Pos.X, g.Pos.Y, g.Pos.X+g.Size.X, g.Pos.Y+g.Size.Y)
	rs.PopXForm()
	return bb
}

func (g *Rect) Render2D() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
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
	rs.PushXForm(g.Paint.XForm)
	bb := g.Paint.BoundingBox(rs, g.Pos.X-g.Radius, g.Pos.Y-g.Radius, g.Pos.X+g.Radius, g.Pos.Y+g.Radius)
	rs.PopXForm()
	return bb
}

func (g *Circle) Render2D() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
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
	rs.PushXForm(g.Paint.XForm)
	bb := g.Paint.BoundingBox(rs, g.Pos.X-g.Radii.X, g.Pos.Y-g.Radii.Y, g.Pos.X+g.Radii.X, g.Pos.Y+g.Radii.Y)
	rs.PopXForm()
	return bb
}

func (g *Ellipse) Render2D() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
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
	rs.PushXForm(g.Paint.XForm)
	bb := g.Paint.BoundingBox(rs, g.Start.X, g.Start.Y, g.End.X, g.End.Y).Canon()
	rs.PopXForm()
	return bb
}

func (g *Line) Render2D() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
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
	rs.PushXForm(g.Paint.XForm)
	bb := g.Paint.BoundingBoxFromPoints(rs, g.Points)
	rs.PopXForm()
	return bb
}

func (g *Polyline) Render2D() {
	if len(g.Points) < 2 {
		return
	}
	pc := &g.Paint
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
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
	rs.PushXForm(g.Paint.XForm)
	bb := g.Paint.BoundingBoxFromPoints(rs, g.Points)
	rs.PopXForm()
	return bb
}

func (g *Polygon) Render2D() {
	if len(g.Points) < 2 {
		return
	}
	pc := &g.Paint
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	pc.DrawPolygon(rs, g.Points)
	pc.FillStrokeClear(rs)
	g.Render2DChildren()
	rs.PopXForm()
}

////////////////////////////////////////////////////////////////////////////////////////
// Path

// 2D Path, using SVG-style data that can render just about anything
type Path struct {
	SVGNodeBase
	Data     []PathData `xml:"-" desc:"the path data to render -- path commands and numbers are serialized, with each command specifying the number of floating-point coord data points that follow"`
	DataStr  string     `xml:"d" desc:"string version of the path data"`
	MinCoord Vec2D      `desc:"minimum coord in path -- computed in BBox2D"`
	MaxCoord Vec2D      `desc:"maximum coord in path -- computed in BBox2D"`
}

var KiT_Path = kit.Types.AddType(&Path{}, nil)

// SetData sets the path data to given string, parsing it into an optimized
// form used for rendering
func (g *Path) SetData(data string) error {
	g.DataStr = data
	var err error
	g.Data, err = PathDataParse(data)
	return err
}

func (g *Path) BBox2D() image.Rectangle {
	// todo: cache values, only update when path is updated..
	rs := &g.Viewport.Render
	rs.PushXForm(g.Paint.XForm)
	g.MinCoord, g.MaxCoord = PathDataMinMax(g.Data)
	bb := g.Paint.BoundingBox(rs, g.MinCoord.X, g.MinCoord.Y, g.MaxCoord.X, g.MaxCoord.Y)
	fmt.Printf("xform: %v min: %v max: %v bb: %v\n", g.Paint.XForm, g.MinCoord, g.MaxCoord, bb)
	rs.PopXForm()
	return bb
	// return vp.Viewport.VpBBox
}

// PathCmds are the commands within the path SVG drawing data type
type PathCmds byte

const (
	// move pen, abs coords
	PcM PathCmds = iota
	// move pen, rel coords
	Pcm
	// lineto, abs
	PcL
	// lineto, rel
	Pcl
	// horizontal lineto, abs
	PcH
	// relative lineto, rel
	Pch
	// vertical lineto, abs
	PcV
	// vertical lineto, rel
	Pcv
	// Bezier curveto, abs
	PcC
	// Bezier curveto, rel
	Pcc
	// smooth Bezier curveto, abs
	PcS
	// smooth Bezier curveto, rel
	Pcs
	// quadratic Bezier curveto, abs
	PcQ
	// quadratic Bezier curveto, rel
	Pcq
	// smooth quadratic Bezier curveto, abs
	PcT
	// smooth quadratic Bezier curveto, rel
	Pct
	// elliptical arc, abs
	PcA
	// elliptical arc, rel
	Pca
	// close path
	PcZ
	// close path
	Pcz
	// errror -- invalid command
	PcErr
)

// PathData encodes the svg path data, using 32-bit floats which are converted
// into int32 for path commands, which contain the number of data points
// following the path command to interpret as numbers, in the lower and upper
// 2 bytes of the converted int32 number.  We don't need that many bits, but
// keeping 32-bit alignment is probably good and really these things don't
// need to be crazy compact as it is unlikely to make a relevant diff in size
// or perf to pack down further
type PathData float32

// decode path data as a command and a number of subsequent values for that command
func (pd PathData) Cmd() (PathCmds, int) {
	iv := int32(pd)
	cmd := PathCmds(iv & 0xFF)   // only the lowest byte for cmd
	n := int((iv & 0xFF00) >> 8) // extract the n from next highest byte
	return cmd, n
}

// encode command and n into PathData
func (pc PathCmds) EncCmd(n int) PathData {
	nb := int32(n << 8) // n up-shifted
	pd := PathData(int32(pc) | nb)
	return pd
}

// PathDataNext gets the next path data point, incrementing the index -- ++
// not an expression so its clunky
func PathDataNext(data []PathData, i *int) float32 {
	pd := data[*i]
	(*i)++
	return float32(pd)
}

// PathDataNextCmd gets the next path data command, incrementing the index -- ++
// not an expression so its clunky
func PathDataNextCmd(data []PathData, i *int) (PathCmds, int) {
	pd := data[*i]
	(*i)++
	return pd.Cmd()
}

// PathDataRender traverses the path data and renders it using paint and render state --
// we assume all the data has been validated and that n's are sufficient, etc
func PathDataRender(data []PathData, pc *Paint, rs *RenderState) {
	sz := len(data)
	if sz == 0 {
		return
	}
	var cx, cy, x1, y1, x2, y2 float32
	for i := 0; i < sz; {
		cmd, n := PathDataNextCmd(data, &i)
		switch cmd {
		case PcM:
			cx = PathDataNext(data, &i)
			cy = PathDataNext(data, &i)
			pc.MoveTo(rs, cx, cy)
			for np := 1; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case Pcm:
			cx += PathDataNext(data, &i)
			cy += PathDataNext(data, &i)
			pc.MoveTo(rs, cx, cy)
			for np := 1; np < n/2; np++ {
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case PcL:
			for np := 0; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case Pcl:
			for np := 0; np < n/2; np++ {
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case PcH:
			for np := 0; np < n; np++ {
				cx = PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case Pch:
			for np := 0; np < n; np++ {
				cx += PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case PcV:
			for np := 0; np < n; np++ {
				cy = PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case Pcv:
			for np := 0; np < n; np++ {
				cy += PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case PcC:
			for np := 0; np < n/6; np++ {
				x1 = PathDataNext(data, &i)
				y1 = PathDataNext(data, &i)
				x2 = PathDataNext(data, &i)
				y2 = PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.CubicTo(rs, x1, y1, x2, y2, cx, cy)
			}
		case Pcc:
			for np := 0; np < n/6; np++ {
				x1 = cx + PathDataNext(data, &i)
				y1 = cy + PathDataNext(data, &i)
				x2 = cx + PathDataNext(data, &i)
				y2 = cy + PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.CubicTo(rs, x1, y1, x2, y2, cx, cy)
			}
		case PcS:
			for np := 0; np < n/4; np++ {
				x1 = 2*cx - x2 // this is a reflection -- todo: need special case where x2 no existe
				y1 = 2*cy - y2
				x2 = PathDataNext(data, &i)
				y2 = PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.CubicTo(rs, x1, y1, x2, y2, cx, cy)
			}
		case Pcs:
			for np := 0; np < n/4; np++ {
				x1 = 2*cx - x2 // this is a reflection -- todo: need special case where x2 no existe
				y1 = 2*cy - y2
				x2 = cx + PathDataNext(data, &i)
				y2 = cy + PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.CubicTo(rs, x1, y1, x2, y2, cx, cy)
			}
		case PcQ:
			for np := 0; np < n/4; np++ {
				x1 = PathDataNext(data, &i)
				y1 = PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.QuadraticTo(rs, x1, y1, cx, cy)
			}
		case Pcq:
			for np := 0; np < n/4; np++ {
				x1 = cx + PathDataNext(data, &i)
				y1 = cy + PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.QuadraticTo(rs, x1, y1, cx, cy)
			}
		case PcT:
			for np := 0; np < n/2; np++ {
				x1 = 2*cx - x1 // this is a reflection
				y1 = 2*cy - y1
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.QuadraticTo(rs, x1, y1, cx, cy)
			}
		case Pct:
			for np := 0; np < n/2; np++ {
				x1 = 2*cx - x1 // this is a reflection
				y1 = 2*cy - y1
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.QuadraticTo(rs, x1, y1, cx, cy)
			}
		case PcA:
			for np := 0; np < n/7; np++ {
				rx := PathDataNext(data, &i)
				ry := PathDataNext(data, &i)
				ang := PathDataNext(data, &i)
				_ = PathDataNext(data, &i) // large-arc-flag
				_ = PathDataNext(data, &i) // sweep-flag
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				/// https://www.w3.org/TR/SVG/paths.html#PathDataEllipticalArcCommands
				// todo: paint expresses in terms of 2 angles, SVG has these flags.. how to map?
				pc.DrawEllipticalArc(rs, cx, cy, rx, ry, ang, 0)
			}
		case Pca:
			for np := 0; np < n/7; np++ {
				rx := PathDataNext(data, &i)
				ry := PathDataNext(data, &i)
				ang := PathDataNext(data, &i)
				_ = PathDataNext(data, &i) // large-arc-flag
				_ = PathDataNext(data, &i) // sweep-flag
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				/// https://www.w3.org/TR/SVG/paths.html#PathDataEllipticalArcCommands
				// todo: paint expresses in terms of 2 angles, SVG has these flags.. how to map?
				pc.DrawEllipticalArc(rs, cx, cy, rx, ry, ang, 0)
			}
		case PcZ:
			pc.ClosePath(rs)
		case Pcz:
			pc.ClosePath(rs)
		}
	}
}

// update min max for given coord index and coords
func minMaxUpdate(cx, cy float32, min, max *Vec2D) {
	c := Vec2D{cx, cy}
	if *min == Vec2DZero && *max == Vec2DZero {
		*min = c
		*max = c
	} else {
		min.SetMin(c)
		max.SetMax(c)
	}
}

// PathDataMinMax traverses the path data and extracts the min and max point coords
func PathDataMinMax(data []PathData) (min, max Vec2D) {
	sz := len(data)
	if sz == 0 {
		return
	}
	var cx, cy float32
	for i := 0; i < sz; {
		cmd, n := PathDataNextCmd(data, &i)
		switch cmd {
		case PcM:
			cx = PathDataNext(data, &i)
			cy = PathDataNext(data, &i)
			minMaxUpdate(cx, cy, &min, &max)
			for np := 1; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case Pcm:
			cx += PathDataNext(data, &i)
			cy += PathDataNext(data, &i)
			minMaxUpdate(cx, cy, &min, &max)
			for np := 1; np < n/2; np++ {
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case PcL:
			for np := 0; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case Pcl:
			for np := 0; np < n/2; np++ {
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case PcH:
			for np := 0; np < n; np++ {
				cx = PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case Pch:
			for np := 0; np < n; np++ {
				cx += PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case PcV:
			for np := 0; np < n; np++ {
				cy = PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case Pcv:
			for np := 0; np < n; np++ {
				cy += PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case PcC:
			for np := 0; np < n/6; np++ {
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case Pcc:
			for np := 0; np < n/6; np++ {
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case PcS:
			for np := 0; np < n/4; np++ {
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case Pcs:
			for np := 0; np < n/4; np++ {
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case PcQ:
			for np := 0; np < n/4; np++ {
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case Pcq:
			for np := 0; np < n/4; np++ {
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case PcT:
			for np := 0; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case Pct:
			for np := 0; np < n/2; np++ {
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max)
			}
		case PcA:
			for np := 0; np < n/7; np++ {
				PathDataNext(data, &i) // rx
				PathDataNext(data, &i) // ry
				PathDataNext(data, &i) // ang
				PathDataNext(data, &i) // large-arc-flag
				PathDataNext(data, &i) // sweep-flag
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				/// https://www.w3.org/TR/SVG/paths.html#PathDataEllipticalArcCommands
				// todo: paint expresses in terms of 2 angles, SVG has these flags.. how to map?
				minMaxUpdate(cx, cy, &min, &max) // todo: not accurate
			}
		case Pca:
			for np := 0; np < n/7; np++ {
				PathDataNext(data, &i) // rx
				PathDataNext(data, &i) // ry
				PathDataNext(data, &i) // ang
				PathDataNext(data, &i) // large-arc-flag
				PathDataNext(data, &i) // sweep-flag
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				minMaxUpdate(cx, cy, &min, &max) // todo: not accurate
			}
		case PcZ:
		case Pcz:
		}
	}
	return
}

// PathDecodeCmd decodes rune into corresponding command
func PathDecodeCmd(r rune) PathCmds {
	cmd := PcErr
	switch r {
	case 'M':
		cmd = PcM
	case 'm':
		cmd = Pcm
	case 'L':
		cmd = PcL
	case 'l':
		cmd = Pcl
	case 'H':
		cmd = PcH
	case 'h':
		cmd = Pch
	case 'V':
		cmd = PcV
	case 'v':
		cmd = Pcv
	case 'C':
		cmd = PcC
	case 'c':
		cmd = Pcc
	case 'S':
		cmd = PcS
	case 's':
		cmd = Pcs
	case 'Q':
		cmd = PcQ
	case 'q':
		cmd = Pcq
	case 'T':
		cmd = PcT
	case 't':
		cmd = Pct
	case 'A':
		cmd = PcA
	case 'a':
		cmd = Pca
	case 'Z':
		cmd = PcZ
	case 'z':
		cmd = Pcz
	}
	return cmd
}

// PathDataParse parses a string representation of the path data into compiled path data
func PathDataParse(d string) ([]PathData, error) {
	var pd []PathData
	endi := len(d) - 1
	numSt := -1
	numGotDec := false // did last number already get a decimal point -- if so, then an additional decimal point now acts as a delimiter -- some crazy paths actually leverage that!
	lr := ' '
	lstCmd := -1
	// first pass: just do the raw parse into commands and numbers
	for i, r := range d {
		num := unicode.IsNumber(r) || (r == '.' && !numGotDec) || (r == '-' && lr == 'e') || r == 'e'
		notn := !num
		if i == endi || notn {
			if numSt != -1 || (i == endi && !notn) {
				if numSt == -1 {
					numSt = i
				}
				nstr := d[numSt:i]
				if i == endi && !notn {
					nstr = d[numSt : i+1]
				}
				p, err := strconv.ParseFloat(nstr, 32)
				if err != nil {
					log.Printf("gi.PathDataParse could not parse string: %v into float\n", nstr)
					IconAutoLoad = false
					return nil, err
				}
				pd = append(pd, PathData(p))
			}
			if r == '-' || r == '.' {
				numSt = i
				if r == '.' {
					numGotDec = true
				} else {
					numGotDec = false
				}
			} else {
				numSt = -1
				numGotDec = false
				if lstCmd != -1 { // update number of args for previous command
					lcm, _ := pd[lstCmd].Cmd()
					n := (len(pd) - lstCmd) - 1
					pd[lstCmd] = lcm.EncCmd(n)
				}
				if !unicode.IsSpace(r) && r != ',' {
					cmd := PathDecodeCmd(r)
					if cmd == PcErr {
						if i != endi {
							err := fmt.Errorf("gi.PathDataParse invalid command rune: %v\n", r)
							log.Println(err)
							return nil, err
						}
					} else {
						pc := cmd.EncCmd(0) // encode with 0 length to start
						lstCmd = len(pd)
						pd = append(pd, pc) // push on
					}
				}
			}
		} else if numSt == -1 { // got start of a number
			numSt = i
			if r == '.' {
				numGotDec = true
			} else {
				numGotDec = false
			}
		} else { // inside a number
			if r == '.' {
				numGotDec = true
			}
		}
		lr = r
	}
	return pd, nil
	// todo: add some error checking..
}

func (g *Path) Render2D() {
	if len(g.Data) < 2 {
		return
	}
	pc := &g.Paint
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	PathDataRender(g.Data, pc, rs)
	// fmt.Printf("PathRender: %v Bg: %v Fill: %v Clr: %v Stroke: %v\n",
	// 	g.PathUnique(), g.Style.Background.Color, g.Paint.FillStyle.Color, g.Style.Color, g.Paint.StrokeStyle.Color)
	pc.FillStrokeClear(rs)
	g.Render2DChildren()
	rs.PopXForm()
}

// todo: new in SVG2: mesh
