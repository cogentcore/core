// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"github.com/rcoreilly/goki/ki/kit"
	// "fmt"
	"image"
)

// shapes2d contains all the SVG-based objects for drawing shapes, paths, etc

////////////////////////////////////////////////////////////////////////////////////////
// Rect

// 2D rectangle, optionally with rounded corners
type Rect struct {
	Node2DBase
	Pos    Vec2D `xml:"{x,y}" desc:"position of the top-left of the rectangle"`
	Size   Vec2D `xml:"{width,height}" desc:"size of the rectangle"`
	Radius Vec2D `xml:"{rx,ry}" desc:"radii for curved corners, as a proportion of width, height"`
}

var KiT_Rect = kit.Types.AddType(&Rect{}, nil)

func (g *Rect) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Rect) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Rect) AsLayout2D() *Layout {
	return nil
}

func (g *Rect) Init2D() {
	g.Init2DBase()
}

func (g *Rect) Style2D() {
	g.Style2DSVG()
	pc := &g.Paint
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	}
}

func (g *Rect) Size2D() {
	g.InitLayout2D()
	g.LayData.AllocSize.SetFromPoint(g.BBox2D().Size())
}

func (g *Rect) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, false) // no style
	g.Layout2DChildren()
}

func (g *Rect) BBox2D() image.Rectangle {
	return g.Paint.BoundingBox(g.Pos.X, g.Pos.Y, g.Pos.X+g.Size.X, g.Pos.Y+g.Size.Y)
}

func (g *Rect) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

func (g *Rect) Render2D() {
	if g.PushBounds() {
		pc := &g.Paint
		rs := &g.Viewport.Render
		if g.Radius.X == 0 && g.Radius.Y == 0 {
			pc.DrawRectangle(rs, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y)
		} else {
			// todo: only supports 1 radius right now -- easy to add another
			pc.DrawRoundedRectangle(rs, g.Pos.X, g.Pos.Y, g.Size.X, g.Size.Y, g.Radius.X)
		}
		pc.FillStrokeClear(rs)
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *Rect) CanReRender2D() bool {
	// todo: could optimize by checking for an opaque fill, and same bbox
	return false
}

func (g *Rect) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Rect{}

////////////////////////////////////////////////////////////////////////////////////////
// Viewport2DFill

// todo: for ViewportFill support an option to insert a HiDPI correction scaling factor at the top!

// viewport fill fills entire viewport -- just a rect that automatically sets size to viewport
type Viewport2DFill struct {
	Rect
}

var KiT_Viewport2DFill = kit.Types.AddType(&Viewport2DFill{}, nil)

func (g *Viewport2DFill) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Viewport2DFill) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Viewport2DFill) AsLayout2D() *Layout {
	return nil
}

func (g *Viewport2DFill) Init2D() {
	g.Init2DBase()
	vp := g.Viewport
	g.Pos = Vec2DZero
	g.Size = Vec2D{float64(vp.ViewBox.Size.X), float64(vp.ViewBox.Size.Y)} // assuming no transforms..
}

func (g *Viewport2DFill) Style2D() {
	g.Style2DSVG()
}

func (g *Viewport2DFill) Size2D() {
	g.InitLayout2D()
	g.LayData.AllocSize.SetFromPoint(g.BBox2D().Size())
}

func (g *Viewport2DFill) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, false) // no style
	g.Layout2DChildren()
}

func (g *Viewport2DFill) BBox2D() image.Rectangle {
	g.Init2D() // keep up-to-date -- cheap
	return g.Paint.BoundingBox(g.Pos.X, g.Pos.Y, g.Pos.X+g.Size.X, g.Pos.Y+g.Size.Y)
}

func (g *Viewport2DFill) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

func (g *Viewport2DFill) Render2D() {
	g.Rect.Render2D()
}

func (g *Viewport2DFill) CanReRender2D() bool {
	return false // why bother
}

func (g *Viewport2DFill) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Viewport2DFill{}

////////////////////////////////////////////////////////////////////////////////////////
// Circle

// 2D circle
type Circle struct {
	Node2DBase
	Pos    Vec2D   `xml:"{cx,cy}" desc:"position of the center of the circle"`
	Radius float64 `xml:"r" desc:"radius of the circle"`
}

var KiT_Circle = kit.Types.AddType(&Circle{}, nil)

func (g *Circle) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Circle) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Circle) AsLayout2D() *Layout {
	return nil
}

func (g *Circle) Init2D() {
	g.Init2DBase()
}

func (g *Circle) Style2D() {
	g.Style2DSVG()
	pc := &g.Paint
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	}
}

func (g *Circle) Size2D() {
	g.InitLayout2D()
	g.LayData.AllocSize.SetFromPoint(g.BBox2D().Size())
}

func (g *Circle) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, false) // no style
	g.Layout2DChildren()
}

func (g *Circle) BBox2D() image.Rectangle {
	return g.Paint.BoundingBox(g.Pos.X-g.Radius, g.Pos.Y-g.Radius, g.Pos.X+g.Radius, g.Pos.Y+g.Radius)
}

func (g *Circle) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

func (g *Circle) Render2D() {
	if g.PushBounds() {
		pc := &g.Paint
		rs := &g.Viewport.Render
		pc.DrawCircle(rs, g.Pos.X, g.Pos.Y, g.Radius)
		pc.FillStrokeClear(rs)
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *Circle) CanReRender2D() bool {
	// todo: could optimize by checking for an opaque fill, and same bbox
	return false
}

func (g *Circle) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Circle{}

////////////////////////////////////////////////////////////////////////////////////////
// Ellipse

// 2D ellipse
type Ellipse struct {
	Node2DBase
	Pos   Vec2D `xml:"{cx,cy}" desc:"position of the center of the ellipse"`
	Radii Vec2D `xml:"{rx, ry}" desc:"radii of the ellipse in the horizontal, vertical axes"`
}

var KiT_Ellipse = kit.Types.AddType(&Ellipse{}, nil)

func (g *Ellipse) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Ellipse) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Ellipse) AsLayout2D() *Layout {
	return nil
}

func (g *Ellipse) Init2D() {
	g.Init2DBase()
}

func (g *Ellipse) Style2D() {
	g.Style2DSVG()
	pc := &g.Paint
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	}
}

func (g *Ellipse) Size2D() {
	g.InitLayout2D()
	g.LayData.AllocSize.SetFromPoint(g.BBox2D().Size())
}

func (g *Ellipse) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, false) // no style
	g.Layout2DChildren()
}

func (g *Ellipse) BBox2D() image.Rectangle {
	return g.Paint.BoundingBox(g.Pos.X-g.Radii.X, g.Pos.Y-g.Radii.Y, g.Pos.X+g.Radii.X, g.Pos.Y+g.Radii.Y)
}

func (g *Ellipse) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

func (g *Ellipse) Render2D() {
	if g.PushBounds() {
		pc := &g.Paint
		rs := &g.Viewport.Render
		pc.DrawEllipse(rs, g.Pos.X, g.Pos.Y, g.Radii.X, g.Radii.Y)
		pc.FillStrokeClear(rs)
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *Ellipse) CanReRender2D() bool {
	return false
}

func (g *Ellipse) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Ellipse{}

////////////////////////////////////////////////////////////////////////////////////////
// Line

// a 2D line
type Line struct {
	Node2DBase
	Start Vec2D `xml:"{x1,y1}" desc:"position of the start of the line"`
	End   Vec2D `xml:"{x2, y2}" desc:"position of the end of the line"`
}

var KiT_Line = kit.Types.AddType(&Line{}, nil)

func (g *Line) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Line) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Line) AsLayout2D() *Layout {
	return nil
}

func (g *Line) Init2D() {
	g.Init2DBase()
}

func (g *Line) Style2D() {
	g.Style2DSVG()
	pc := &g.Paint
	if pc.HasNoStrokeOrFill() {
		pc.Off = true
	}
}

func (g *Line) Size2D() {
	g.InitLayout2D()
	g.LayData.AllocSize.SetFromPoint(g.BBox2D().Size())
}

func (g *Line) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, false) // no style
	g.Layout2DChildren()
}

func (g *Line) BBox2D() image.Rectangle {
	return g.Paint.BoundingBox(g.Start.X, g.Start.Y, g.End.X, g.End.Y).Canon()
}

func (g *Line) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

func (g *Line) Render2D() {
	if g.PushBounds() {
		pc := &g.Paint
		rs := &g.Viewport.Render
		pc.DrawLine(rs, g.Start.X, g.Start.Y, g.End.X, g.End.Y)
		pc.Stroke(rs)
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *Line) CanReRender2D() bool {
	return false
}

func (g *Line) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Line{}

////////////////////////////////////////////////////////////////////////////////////////
// Polyline

// 2D Polyline
type Polyline struct {
	Node2DBase
	Points []Vec2D `xml:"points" desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest"`
}

var KiT_Polyline = kit.Types.AddType(&Polyline{}, nil)

func (g *Polyline) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Polyline) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Polyline) AsLayout2D() *Layout {
	return nil
}

func (g *Polyline) Init2D() {
	g.Init2DBase()
}

func (g *Polyline) Style2D() {
	g.Style2DSVG()
	pc := &g.Paint
	if pc.HasNoStrokeOrFill() || len(g.Points) < 2 {
		pc.Off = true
	}
}

func (g *Polyline) Size2D() {
	g.InitLayout2D()
	g.LayData.AllocSize.SetFromPoint(g.BBox2D().Size())
}

func (g *Polyline) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, false) // no style
	g.Layout2DChildren()
}

func (g *Polyline) BBox2D() image.Rectangle {
	return g.Paint.BoundingBoxFromPoints(g.Points)
}

func (g *Polyline) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

func (g *Polyline) Render2D() {
	if g.PushBounds() {
		pc := &g.Paint
		rs := &g.Viewport.Render
		if len(g.Points) < 2 {
			return
		}
		pc.DrawPolyline(rs, g.Points)
		pc.FillStrokeClear(rs)
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *Polyline) CanReRender2D() bool {
	return false
}

func (g *Polyline) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Polyline{}

////////////////////////////////////////////////////////////////////////////////////////
// Polygon

// 2D Polygon
type Polygon struct {
	Node2DBase
	Points []Vec2D `xml:"points" desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest, then does a closepath at the end"`
}

var KiT_Polygon = kit.Types.AddType(&Polygon{}, nil)

func (g *Polygon) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Polygon) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Polygon) AsLayout2D() *Layout {
	return nil
}

func (g *Polygon) Init2D() {
	g.Init2DBase()
}

func (g *Polygon) Style2D() {
	g.Style2DSVG()
	pc := &g.Paint
	if pc.HasNoStrokeOrFill() || len(g.Points) < 2 {
		pc.Off = true
	}
}

func (g *Polygon) Size2D() {
	g.InitLayout2D()
	g.LayData.AllocSize.SetFromPoint(g.BBox2D().Size())
}

func (g *Polygon) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, false) // no style
	g.Layout2DChildren()
}

func (g *Polygon) BBox2D() image.Rectangle {
	return g.Paint.BoundingBoxFromPoints(g.Points)
}

func (g *Polygon) ChildrenBBox2D() image.Rectangle {
	return g.VpBBox
}

func (g *Polygon) Render2D() {
	if g.PushBounds() {
		pc := &g.Paint
		rs := &g.Viewport.Render
		if len(g.Points) < 2 {
			return
		}
		pc.DrawPolygon(rs, g.Points)
		pc.FillStrokeClear(rs)
		g.Render2DChildren()
		g.PopBounds()
	}
}

func (g *Polygon) CanReRender2D() bool {
	return false
}

func (g *Polygon) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Polygon{}

////////////////////////////////////////////////////////////////////////////////////////

// todo: new in SVG2: mesh
