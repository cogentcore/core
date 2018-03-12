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

func (g *Rect) InitNode2D() {
}

func (g *Rect) PaintProps2D() {
	pc := &g.MyPaint
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	}
}

func (g *Rect) Layout2D() {
	g.Layout.AllocSize.SetFromPoint(g.Node2DBBox().Size())
}

func (g *Rect) Node2DBBox() image.Rectangle {
	return g.MyPaint.BoundingBox(g.Pos.X, g.Pos.Y, g.Pos.X+g.Size.X, g.Pos.Y+g.Size.Y)
}

func (g *Rect) Render2D() {
	g.SetWinBBox(g.Node2DBBox())
	pc := &g.MyPaint
	rs := &g.Viewport.Render
	if g.Radius.X == 0 && g.Radius.Y == 0 {
		pc.DrawRectangle(rs, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
	} else {
		// todo: only supports 1 radius right now -- easy to add another
		pc.DrawRoundedRectangle(rs, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y, g.Radius.X)
	}
	pc.FillStrokeClear(rs)
}

func (g *Rect) CanReRender2D() bool {
	// todo: could optimize by checking for an opaque fill, and same bbox
	return false
}

// check for interface implementation
var _ Node2D = &Rect{}

////////////////////////////////////////////////////////////////////////////////////////

// todo: for ViewportFill support an option to insert a HiDPI correction scaling factor at the top!

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

func (g *Viewport2DFill) InitNode2D() {
	vp := g.Viewport
	g.Pos = Point2D{0, 0}
	g.Size = Size2D{float64(vp.ViewBox.Size.X), float64(vp.ViewBox.Size.Y)} // assuming no transforms..
}

func (g *Viewport2DFill) PaintProps2D() {
	g.CopyParentPaint()
	g.MyPaint.SetFromNode(&g.Node2DBase)
}

func (g *Viewport2DFill) Layout2D() {
	g.Layout.AllocSize.SetFromPoint(g.Node2DBBox().Size())
}

func (g *Viewport2DFill) Node2DBBox() image.Rectangle {
	g.InitNode2D() // keep up-to-date -- cheap
	return g.MyPaint.BoundingBox(g.Pos.X, g.Pos.Y, g.Pos.X+g.Size.X, g.Pos.Y+g.Size.Y)
}

func (g *Viewport2DFill) Render2D() {
	g.Rect.Render2D()
}

func (g *Viewport2DFill) CanReRender2D() bool {
	return false
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

func (g *Circle) InitNode2D() {
}

func (g *Circle) PaintProps2D() {
	pc := &g.MyPaint
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	}
}

func (g *Circle) Layout2D() {
	g.Layout.AllocSize.SetFromPoint(g.Node2DBBox().Size())
}

func (g *Circle) Node2DBBox() image.Rectangle {
	return g.MyPaint.BoundingBox(g.Pos.X-g.Radius, g.Pos.Y-g.Radius, g.Pos.X+g.Radius, g.Pos.Y+g.Radius)
}

func (g *Circle) Render2D() {
	g.SetWinBBox(g.Node2DBBox())
	pc := &g.MyPaint
	rs := &g.Viewport.Render
	pc.DrawCircle(rs, g.Pos.X, g.Pos.Y, g.Radius)
	pc.FillStrokeClear(rs)
}

func (g *Circle) CanReRender2D() bool {
	// todo: could optimize by checking for an opaque fill, and same bbox
	return false
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

func (g *Ellipse) InitNode2D() {
}

func (g *Ellipse) PaintProps2D() {
	pc := &g.MyPaint
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	}
}

func (g *Ellipse) Layout2D() {
	g.Layout.AllocSize.SetFromPoint(g.Node2DBBox().Size())
}

func (g *Ellipse) Node2DBBox() image.Rectangle {
	return g.MyPaint.BoundingBox(g.Pos.X-g.Radii.X, g.Pos.Y-g.Radii.Y, g.Pos.X+g.Radii.X, g.Pos.Y+g.Radii.Y)
}

func (g *Ellipse) Render2D() {
	g.SetWinBBox(g.Node2DBBox())
	pc := &g.MyPaint
	rs := &g.Viewport.Render
	pc.DrawEllipse(rs, g.Pos.X, g.Pos.Y, g.Radii.X, g.Radii.Y)
	pc.FillStrokeClear(rs)
}

func (g *Ellipse) CanReRender2D() bool {
	return false
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

func (g *Line) InitNode2D() {
}

func (g *Line) PaintProps2D() {
	pc := &g.MyPaint
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	}
}

func (g *Line) Layout2D() {
	g.Layout.AllocSize.SetFromPoint(g.Node2DBBox().Size())
}

func (g *Line) Node2DBBox() image.Rectangle {
	return g.MyPaint.BoundingBox(g.Start.X, g.Start.Y, g.End.X, g.End.Y).Canon()
}

func (g *Line) Render2D() {
	pc := &g.MyPaint
	rs := &g.Viewport.Render
	pc.DrawLine(rs, g.Start.X, g.Start.Y, g.End.X, g.End.Y)
	pc.FillStrokeClear(rs)
}

func (g *Line) CanReRender2D() bool {
	return false
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

func (g *Polyline) InitNode2D() {
}

func (g *Polyline) PaintProps2D() {
	pc := &g.MyPaint
	if pc.HasNoStrokeOrFill() || len(g.Points) < 2 {
		pc.Off = true
	}
}

func (g *Polyline) Layout2D() {
	g.Layout.AllocSize.SetFromPoint(g.Node2DBBox().Size())
}

func (g *Polyline) Node2DBBox() image.Rectangle {
	return g.MyPaint.BoundingBoxFromPoints(g.Points)
}

func (g *Polyline) Render2D() {
	g.SetWinBBox(g.Node2DBBox())
	pc := &g.MyPaint
	rs := &g.Viewport.Render
	if len(g.Points) < 2 {
		return
	}
	pc.DrawPolyline(rs, g.Points)
	pc.FillStrokeClear(rs)
}

func (g *Polyline) CanReRender2D() bool {
	return false
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

func (g *Polygon) InitNode2D() {
}

func (g *Polygon) PaintProps2D() {
	pc := &g.MyPaint
	if pc.HasNoStrokeOrFill() || len(g.Points) < 2 {
		pc.Off = true
	}
}

func (g *Polygon) Layout2D() {
	g.Layout.AllocSize.SetFromPoint(g.Node2DBBox().Size())
}

func (g *Polygon) Node2DBBox() image.Rectangle {
	return g.MyPaint.BoundingBoxFromPoints(g.Points)
}

func (g *Polygon) Render2D() {
	g.SetWinBBox(g.Node2DBBox())
	pc := &g.MyPaint
	rs := &g.Viewport.Render
	if len(g.Points) < 2 {
		return
	}
	pc.DrawPolygon(rs, g.Points)
	pc.FillStrokeClear(rs)
}

func (g *Polygon) CanReRender2D() bool {
	return false
}

////////////////////////////////////////////////////////////////////////////////////////

// todo: new in SVG2: mesh
