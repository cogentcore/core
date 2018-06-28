// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"math"
	"strconv"
	"unicode"

	"github.com/goki/ki/kit"
)

// svg_nodes contains all the SVG-based nodes for drawing shapes, paths, etc
// see svg.go for the base class

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
	g.MinCoord, g.MaxCoord = PathDataMinMax(g.Data)
	bb := g.Pnt.BoundingBox(rs, g.MinCoord.X, g.MinCoord.Y, g.MaxCoord.X, g.MaxCoord.Y)
	return bb
}

func (g *Path) Render2D() {
	if len(g.Data) < 2 {
		return
	}
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	g.ComputeBBoxSVG()
	PathDataRender(g.Data, pc, rs)
	pc.FillStrokeClear(rs)
	g.Render2DChildren()
	rs.PopXForm()
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

func reflectPt(px, py, rx, ry float32) (x, y float32) {
	return px*2 - rx, py*2 - ry
}

// PathDataRender traverses the path data and renders it using paint and render state --
// we assume all the data has been validated and that n's are sufficient, etc
func PathDataRender(data []PathData, pc *Paint, rs *RenderState) {
	sz := len(data)
	if sz == 0 {
		return
	}
	lastCmd := PcErr
	var stx, sty, cx, cy, x1, y1, ctrlx, ctrly float32
	for i := 0; i < sz; {
		cmd, n := PathDataNextCmd(data, &i)
		rel := false
		switch cmd {
		case PcM:
			cx = PathDataNext(data, &i)
			cy = PathDataNext(data, &i)
			pc.MoveTo(rs, cx, cy)
			stx, sty = cx, cy
			for np := 1; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case Pcm:
			cx += PathDataNext(data, &i)
			cy += PathDataNext(data, &i)
			pc.MoveTo(rs, cx, cy)
			stx, sty = cx, cy
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
				ctrlx = PathDataNext(data, &i)
				ctrly = PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.CubicTo(rs, x1, y1, ctrlx, ctrly, cx, cy)
			}
		case Pcc:
			for np := 0; np < n/6; np++ {
				x1 = cx + PathDataNext(data, &i)
				y1 = cy + PathDataNext(data, &i)
				ctrlx = cx + PathDataNext(data, &i)
				ctrly = cy + PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.CubicTo(rs, x1, y1, ctrlx, ctrly, cx, cy)
			}
		case Pcs:
			rel = true
			fallthrough
		case PcS:
			for np := 0; np < n/4; np++ {
				switch lastCmd {
				case Pcc, PcC, Pcs, PcS:
					ctrlx, ctrly = reflectPt(cx, cy, ctrlx, ctrly)
				default:
					ctrlx, ctrly = cx, cy
				}
				if rel {
					x1 = cx + PathDataNext(data, &i)
					y1 = cy + PathDataNext(data, &i)
					cx += PathDataNext(data, &i)
					cy += PathDataNext(data, &i)
				} else {
					x1 = PathDataNext(data, &i)
					y1 = PathDataNext(data, &i)
					cx = PathDataNext(data, &i)
					cy = PathDataNext(data, &i)
				}
				pc.CubicTo(rs, ctrlx, ctrly, x1, y1, cx, cy)
				lastCmd = cmd
				ctrlx = x1
				ctrly = y1
			}
		case PcQ:
			for np := 0; np < n/4; np++ {
				ctrlx = PathDataNext(data, &i)
				ctrly = PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.QuadraticTo(rs, ctrlx, ctrly, cx, cy)
			}
		case Pcq:
			for np := 0; np < n/4; np++ {
				ctrlx = cx + PathDataNext(data, &i)
				ctrly = cy + PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.QuadraticTo(rs, ctrlx, ctrly, cx, cy)
			}
		case Pct:
			rel = true
			fallthrough
		case PcT:
			for np := 0; np < n/2; np++ {
				switch lastCmd {
				case Pcq, PcQ, PcT, Pct:
					ctrlx, ctrly = reflectPt(cx, cy, ctrlx, ctrly)
				default:
					ctrlx, ctrly = cx, cy
				}
				if rel {
					cx += PathDataNext(data, &i)
					cy += PathDataNext(data, &i)
				} else {
					cx = PathDataNext(data, &i)
					cy = PathDataNext(data, &i)
				}
				pc.QuadraticTo(rs, ctrlx, ctrly, cx, cy)
				lastCmd = cmd
			}
		case Pca:
			rel = true
			fallthrough
		case PcA:
			for np := 0; np < n/7; np++ {
				rx := PathDataNext(data, &i)
				ry := PathDataNext(data, &i)
				ang := PathDataNext(data, &i)
				largeArc := (PathDataNext(data, &i) != 0)
				sweep := (PathDataNext(data, &i) != 0)
				pcx := cx
				pcy := cy
				if rel {
					cx += PathDataNext(data, &i)
					cy += PathDataNext(data, &i)
				} else {
					cx = PathDataNext(data, &i)
					cy = PathDataNext(data, &i)
				}
				ncx, ncy := FindEllipseCenter(&rx, &ry, ang*math.Pi/180, pcx, pcy, cx, cy, sweep, largeArc)
				cx, cy = pc.DrawEllipticalArcPath(rs, ncx, ncy, cx, cy, pcx, pcy, rx, ry, ang, largeArc, sweep)
			}
		case PcZ:
			pc.ClosePath(rs)
			cx, cy = stx, sty
		case Pcz:
			pc.ClosePath(rs)
			cx, cy = stx, sty
		}
		lastCmd = cmd
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

////////////////////////////////////////////////////////////////////////////////////////
// SVGText Node

// todo: lots of work likely needed on laying-out text in proper way
// https://www.w3.org/TR/SVG2/text.html#GlyphsMetrics
// todo: tspan element

// SVGText renders 2D text within an SVG -- it handles both text and tspan
// elements (a tspan is just nested under a parent text)
type SVGText struct {
	SVGNodeBase
	Pos          Vec2D     `xml:"{x,y}" desc:"position of the left, baseline of the text"`
	Width        float32   `xml:"width" desc:"width of text to render if using word-wrapping"`
	Text         string    `xml:"text" desc:"text string to render"`
	WrappedText  []string  `json:"-" xml:"-" desc:"word-wrapped version of the string"`
	CharPosX     []float32 `desc:"character positions along X axis, if specified"`
	CharPosY     []float32 `desc:"character positions along Y axis, if specified"`
	CharPosDX    []float32 `desc:"character delta-positions along X axis, if specified"`
	CharPosDY    []float32 `desc:"character delta-positions along Y axis, if specified"`
	CharRots     []float32 `desc:"character rotations, if specified"`
	TextLength   float32   `desc:"author's computed text length, if specified -- we attempt to match"`
	AdjustGlyphs bool      `desc:"in attempting to match TextLength, should we adjust glyphs in addition to spacing?"`
}

var KiT_SVGText = kit.Types.AddType(&SVGText{}, nil)

// func (g *SVGText) Size2D() {
// 	g.InitLayout2D()
// 	pc := &g.Pnt
// 	var w, h float32
// 	// pre-wrap the text
// 	if pc.TextStyle.WordWrap {
// 		g.WrappedText, h = pc.MeasureStringWrapped(g.Text, g.Width, pc.TextStyle.EffLineHeight())
// 	} else {
// 		w, h = pc.MeasureString(g.Text)
// 	}
// 	g.LayData.AllocSize = Vec2D{w, h}
// }

func (g *SVGText) BBox2D() image.Rectangle {
	rs := &g.Viewport.Render
	// todo: this is not right -- update
	return g.Pnt.BoundingBox(rs, g.Pos.X, g.Pos.Y, g.Pos.X+20, g.Pos.Y+20)
}

func (g *SVGText) Render2D() {
	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.PushXForm(pc.XForm)
	g.ComputeBBoxSVG()
	// fmt.Printf("rendering text %v\n", g.Text) todo: important: need a
	// current text position from parent text -- these coords are relative to
	// that!
	if pc.TextStyle.WordWrap {
		pc.DrawStringLines(rs, g.WrappedText, g.Pos.X, g.Pos.Y, g.Width, 0)
		// g.LayData.AllocSize.X, g.LayData.AllocSize.Y)
	} else {
		pc.DrawString(rs, g.Text, g.Pos.X, g.Pos.Y, 0) // g.LayData.AllocSize.X)
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
