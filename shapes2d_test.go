// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"math/rand"
	"testing"
)

func TestRect(t *testing.T) {
	parent := NewViewport2D(400, 400)
	parent.InitName(parent, "vp1")
	vpfill := parent.AddNewChild(KiT_Viewport2DFill, "vpfill").(*Viewport2DFill)
	vpfill.SetProp("fill", "#FFF")

	rect1 := vpfill.AddNewChild(KiT_Rect, "rect1").(*Rect)
	rect1.SetProp("fill", "#008800")
	rect1.SetProp("stroke", "#0000FF")
	rect1.SetProp("stroke-width", 5.0)
	// rect1.SetProp("stroke-linejoin", "round")
	rect1.Pos = Vec2D{10, 10}
	rect1.Size = Vec2D{100, 100}

	parent.FullRender2DTree()
	// parent.SavePNG("test_rect.png")
}

func TestShapesAll(t *testing.T) {
	width := 400
	height := 400
	parent := NewViewport2D(width, height)
	parent.InitName(parent, "vp1")
	vpfill := parent.AddNewChild(KiT_Viewport2DFill, "vpfill").(*Viewport2DFill)
	vpfill.SetProp("fill", "#FFF")

	rect1 := vpfill.AddNewChild(KiT_Rect, "rect1").(*Rect)
	rect1.SetProp("fill", "#008800")
	rect1.SetProp("stroke", "#0000FF")
	rect1.SetProp("stroke-width", 5.0)
	rect1.Pos = Vec2D{10, 10}
	rect1.Size = Vec2D{100, 100}

	circle1 := vpfill.AddNewChild(KiT_Circle, "circle1").(*Circle)
	circle1.SetProp("fill", "none") // todo: need to process
	circle1.SetProp("stroke", "#CC0000")
	circle1.SetProp("stroke-width", 2.0)
	circle1.Pos = Vec2D{200, 100}
	circle1.Radius = 40

	ellipse1 := circle1.AddNewChild(KiT_Ellipse, "ellipse1").(*Ellipse)
	ellipse1.SetProp("fill", "#55000055")
	ellipse1.SetProp("stroke", "#880000")
	ellipse1.SetProp("stroke-width", 2.0)
	ellipse1.Pos = Vec2D{100, 100}
	ellipse1.Radii = Vec2D{80, 20}

	line1 := vpfill.AddNewChild(KiT_Line, "line1").(*Line)
	line1.SetProp("stroke", "#888800")
	line1.SetProp("stroke-width", 5.0)
	line1.Start = Vec2D{100, 100}
	line1.End = Vec2D{150, 200}

	polyline1 := vpfill.AddNewChild(KiT_Polyline, "polyline1").(*Polyline)
	polyline1.SetProp("stroke", "#888800")
	polyline1.SetProp("stroke-width", 4.0)

	for i := 0; i < 10; i++ {
		x1 := rand.Float32() * float32(width)
		y1 := rand.Float32() * float32(height)
		polyline1.Points = append(polyline1.Points, Vec2D{x1, y1})
	}

	polygon1 := vpfill.AddNewChild(KiT_Polygon, "polygon1").(*Polygon)
	polygon1.SetProp("fill", "#55005555")
	polygon1.SetProp("stroke", "#888800")
	polygon1.SetProp("stroke-width", 4.0)

	for i := 0; i < 10; i++ {
		x1 := rand.Float32() * float32(width)
		y1 := rand.Float32() * float32(height)
		polygon1.Points = append(polygon1.Points, Vec2D{x1, y1})
	}

	parent.FullRender2DTree()
	parent.SavePNG("test_shape_all.png")
}
