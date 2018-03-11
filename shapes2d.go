// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/rcoreilly/goki/ki"
	// "fmt"
	"image"
)

// NOTE: for all render2D calls, viewport render has already handled the SetPaintFromNode call,
// and also managed disabled, visible status

////////////////////////////////////////////////////////////////////////////////////////

// 2D rectangle, optionally with rounded corners
type Rect struct {
	Node2DBase
	Pos    Point2D `svg:"{x,y}",desc:"position of top-left corner"`
	Size   Size2D  `svg:"{width,height}",desc:"size of rectangle"`
	Radius Point2D `svg:"{rx,ry}",desc:"radii for curved corners, as a proportion of width, height"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Rect = ki.KiTypes.AddType(&Rect{})

func (g *Rect) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Rect) GiViewport2D() *Viewport2D {
	return nil
}

func (g *Rect) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *Rect) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBox(g.Pos.X, g.Pos.Y, g.Pos.X+g.Size.X, g.Pos.Y+g.Size.Y)
}

func (g *Rect) Render2D(vp *Viewport2D) bool {
	if vp.HasNoStrokeOrFill() {
		return true
	}
	if g.Radius.X == 0 && g.Radius.Y == 0 {
		vp.DrawRectangle(g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
	} else {
		// todo: only supports 1 radius right now -- easy to add another
		vp.DrawRoundedRectangle(g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y, g.Radius.X)
	}
	if vp.HasFill() {
		vp.FillPreserve()
	}
	if vp.HasStroke() {
		vp.StrokePreserve()
	}
	vp.ClearPath()
	return true
}

////////////////////////////////////////////////////////////////////////////////////////

// viewport fill fills entire viewport -- just a rect that automatically sets size to viewport
type Viewport2DFill struct {
	Rect
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Viewport2DFill = ki.KiTypes.AddType(&Viewport2DFill{})

func (g *Viewport2DFill) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Viewport2DFill) GiViewport2D() *Viewport2D {
	return nil
}

func (g *Viewport2DFill) InitNode2D(vp *Viewport2D) bool {
	g.Pos = Point2D{0, 0}
	g.Size = Size2D{float64(vp.ViewBox.Size.X), float64(vp.ViewBox.Size.Y)} // assuming no transforms..
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *Viewport2DFill) Node2DBBox(vp *Viewport2D) image.Rectangle {
	g.Pos = Point2D{0, 0}
	g.Size = Size2D{float64(vp.ViewBox.Size.X), float64(vp.ViewBox.Size.Y)} // assuming no transforms..
	return vp.BoundingBox(g.Pos.X, g.Pos.Y, g.Pos.X+g.Size.X, g.Pos.Y+g.Size.Y)
}

func (g *Viewport2DFill) Render2D(vp *Viewport2D) bool {
	return g.Rect.Render2D(vp)
}

////////////////////////////////////////////////////////////////////////////////////////

// 2D circle
type Circle struct {
	Node2DBase
	Pos    Point2D `svg:"{cx,cy}",desc:"position of the center of the circle"`
	Radius float64 `svg:"r",desc:"radius of the circle"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Circle = ki.KiTypes.AddType(&Circle{})

func (g *Circle) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Circle) GiViewport2D() *Viewport2D {
	return nil
}

func (g *Circle) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *Circle) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBox(g.Pos.X-g.Radius, g.Pos.Y-g.Radius, g.Pos.X+g.Radius, g.Pos.Y+g.Radius)
}

func (g *Circle) Render2D(vp *Viewport2D) bool {
	if vp.HasNoStrokeOrFill() {
		return true
	}
	vp.DrawCircle(g.Pos.X, g.Pos.Y, g.Radius)
	if vp.HasFill() {
		vp.FillPreserve()
	}
	if vp.HasStroke() {
		vp.StrokePreserve()
	}
	vp.ClearPath()
	return true
}

////////////////////////////////////////////////////////////////////////////////////////

// 2D ellipse
type Ellipse struct {
	Node2DBase
	Pos   Point2D `svg:"{cx,cy}",desc:"position of the center of the ellipse"`
	Radii Size2D  `svg:"{rx, ry}",desc:"radii of the ellipse in the horizontal, vertical axes"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Ellipse = ki.KiTypes.AddType(&Ellipse{})

func (g *Ellipse) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Ellipse) GiViewport2D() *Viewport2D {
	return nil
}

func (g *Ellipse) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *Ellipse) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBox(g.Pos.X-g.Radii.X, g.Pos.Y-g.Radii.Y, g.Pos.X+g.Radii.X, g.Pos.Y+g.Radii.Y)
}

func (g *Ellipse) Render2D(vp *Viewport2D) bool {
	if vp.HasNoStrokeOrFill() {
		return true
	}
	vp.DrawEllipse(g.Pos.X, g.Pos.Y, g.Radii.X, g.Radii.Y)
	if vp.HasFill() {
		vp.FillPreserve()
	}
	if vp.HasStroke() {
		vp.StrokePreserve()
	}
	vp.ClearPath()
	return true
}

////////////////////////////////////////////////////////////////////////////////////////

// a 2D line
type Line struct {
	Node2DBase
	Start Point2D `svg:"{x1,y1}",desc:"position of the start of the line"`
	End   Point2D `svg:"{x2, y2}",desc:"position of the end of the line"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Line = ki.KiTypes.AddType(&Line{})

func (g *Line) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Line) GiViewport2D() *Viewport2D {
	return nil
}

func (g *Line) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *Line) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBox(g.Start.X, g.Start.Y, g.End.X, g.End.Y).Canon()
}

func (g *Line) Render2D(vp *Viewport2D) bool {
	if vp.HasNoStrokeOrFill() {
		return true
	}
	vp.DrawLine(g.Start.X, g.Start.Y, g.End.X, g.End.Y)
	if vp.HasFill() {
		vp.FillPreserve()
	}
	if vp.HasStroke() {
		vp.StrokePreserve()
	}
	vp.ClearPath()
	return true
}

////////////////////////////////////////////////////////////////////////////////////////

// 2D Polyline
type Polyline struct {
	Node2DBase
	Points []Point2D `svg:"points",desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Polyline = ki.KiTypes.AddType(&Polyline{})

func (g *Polyline) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Polyline) GiViewport2D() *Viewport2D {
	return nil
}

func (g *Polyline) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *Polyline) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBoxFromPoints(g.Points)
}

func (g *Polyline) Render2D(vp *Viewport2D) bool {
	if vp.HasNoStrokeOrFill() {
		return true
	}
	if len(g.Points) < 2 {
		return true // todo: could issue warning but..
	}
	vp.DrawPolyline(g.Points)
	if vp.HasFill() {
		vp.FillPreserve()
	}
	if vp.HasStroke() {
		vp.StrokePreserve()
	}
	vp.ClearPath()
	return true
}

////////////////////////////////////////////////////////////////////////////////////////

// 2D Polygon
type Polygon struct {
	Node2DBase
	Points []Point2D `svg:"points",desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest, then does a closepath at the end"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Polygon = ki.KiTypes.AddType(&Polygon{})

func (g *Polygon) GiNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Polygon) GiViewport2D() *Viewport2D {
	return nil
}

func (g *Polygon) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *Polygon) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBoxFromPoints(g.Points)
}

func (g *Polygon) Render2D(vp *Viewport2D) bool {
	if vp.HasNoStrokeOrFill() {
		return true
	}
	if len(g.Points) < 2 {
		return true // todo: could issue warning but..
	}
	vp.DrawPolygon(g.Points)
	if vp.HasFill() {
		vp.FillPreserve()
	}
	if vp.HasStroke() {
		vp.StrokePreserve()
	}
	vp.ClearPath()
	return true
}

////////////////////////////////////////////////////////////////////////////////////////

// todo: new in SVG2: mesh
