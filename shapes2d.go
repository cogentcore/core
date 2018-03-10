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

// todo: can we rename GiRect -> Rect without colliding with anything else??

////////////////////////////////////////////////////////////////////////////////////////

// a 2D rectangle, optionally with rounded corners
type GiRect struct {
	GiNode2D
	Pos    Point2D `svg:"{x,y}",desc:"position of top-left corner"`
	Size   Size2D  `svg:"{width,height}",desc:"size of viewbox within parent Viewport2D"`
	Radius Point2D `svg:"{rx,ry}",desc:"radii for curved corners, as a proportion of width, height"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtGiRect = ki.KiTypes.AddType(&GiRect{})

func (g *GiRect) Node2D() *GiNode2D {
	return &g.GiNode2D
}

func (g *GiRect) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *GiRect) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBox(g.Pos.X, g.Pos.Y, g.Pos.X+g.Size.X, g.Pos.Y+g.Size.Y)
}

func (g *GiRect) Render2D(vp *Viewport2D) bool {
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

// a 2D circle
type GiCircle struct {
	GiNode2D
	Pos    Point2D `svg:"{cx,cy}",desc:"position of the center of the circle"`
	Radius float64 `svg:"r",desc:"radius of the circle"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtGiCircle = ki.KiTypes.AddType(&GiCircle{})

func (g *GiCircle) Node2D() *GiNode2D {
	return &g.GiNode2D
}

func (g *GiCircle) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *GiCircle) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBox(g.Pos.X-g.Radius, g.Pos.Y-g.Radius, g.Pos.X+g.Radius, g.Pos.Y+g.Radius)
}

func (g *GiCircle) Render2D(vp *Viewport2D) bool {
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

// a 2D ellipse
type GiEllipse struct {
	GiNode2D
	Pos   Point2D `svg:"{cx,cy}",desc:"position of the center of the ellipse"`
	Radii Size2D  `svg:"{rx, ry}",desc:"radii of the ellipse in the horizontal, vertical axes"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtGiEllipse = ki.KiTypes.AddType(&GiEllipse{})

func (g *GiEllipse) Node2D() *GiNode2D {
	return &g.GiNode2D
}

func (g *GiEllipse) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *GiEllipse) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBox(g.Pos.X-g.Radii.X, g.Pos.Y-g.Radii.Y, g.Pos.X+g.Radii.X, g.Pos.Y+g.Radii.Y)
}

func (g *GiEllipse) Render2D(vp *Viewport2D) bool {
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
type GiLine struct {
	GiNode2D
	Start Point2D `svg:"{x1,y1}",desc:"position of the start of the line"`
	End   Point2D `svg:"{x2, y2}",desc:"position of the end of the line"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtGiLine = ki.KiTypes.AddType(&GiLine{})

func (g *GiLine) Node2D() *GiNode2D {
	return &g.GiNode2D
}

func (g *GiLine) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *GiLine) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBox(g.Start.X, g.Start.Y, g.End.X, g.End.Y).Canon()
}

func (g *GiLine) Render2D(vp *Viewport2D) bool {
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

// a 2D Polyline
type GiPolyline struct {
	GiNode2D
	Points []Point2D `svg:"points",desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtGiPolyline = ki.KiTypes.AddType(&GiPolyline{})

func (g *GiPolyline) Node2D() *GiNode2D {
	return &g.GiNode2D
}

func (g *GiPolyline) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *GiPolyline) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBoxFromPoints(g.Points)
}

func (g *GiPolyline) Render2D(vp *Viewport2D) bool {
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

// a 2D Polygon
type GiPolygon struct {
	GiNode2D
	Points []Point2D `svg:"points",desc:"the coordinates to draw -- does a moveto on the first, then lineto for all the rest, then does a closepath at the end"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KtGiPolygon = ki.KiTypes.AddType(&GiPolygon{})

func (g *GiPolygon) Node2D() *GiNode2D {
	return &g.GiNode2D
}

func (g *GiPolygon) InitNode2D(vp *Viewport2D) bool {
	g.NodeSig.Connect(vp.This, SignalViewport2D)
	return true
}

func (g *GiPolygon) Node2DBBox(vp *Viewport2D) image.Rectangle {
	return vp.BoundingBoxFromPoints(g.Points)
}

func (g *GiPolygon) Render2D(vp *Viewport2D) bool {
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
