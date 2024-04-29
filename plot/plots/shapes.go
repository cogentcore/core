// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plots

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
)

type Shapes int32 //enums:enum

const (
	// Ring is the outline of a circle
	Ring Shapes = iota

	// Circle is a solid circle
	Circle

	// Square is the outline of a square
	Square

	// Box is a filled square
	Box

	// Triangle is the outline of a triangle
	Triangle

	// Pyramid is a filled triangle
	Pyramid

	// Plus is a plus sign
	Plus

	// Cross is a big X
	Cross
)

// DrawShape draws the given shape
func DrawShape(pc *paint.Context, pos math32.Vector2, size float32, shape Shapes) {
	switch shape {
	case Ring:
		DrawRing(pc, pos, size)
	case Circle:
		DrawCircle(pc, pos, size)
	case Square:
		DrawSquare(pc, pos, size)
	case Box:
		DrawBox(pc, pos, size)
	case Triangle:
		DrawTriangle(pc, pos, size)
	case Pyramid:
		DrawPyramid(pc, pos, size)
	case Plus:
		DrawPlus(pc, pos, size)
	case Cross:
		DrawCross(pc, pos, size)
	}
}

func DrawRing(pc *paint.Context, pos math32.Vector2, size float32) {
	pc.DrawCircle(pos.X, pos.Y, size)
	pc.Stroke()
}

func DrawCircle(pc *paint.Context, pos math32.Vector2, size float32) {
	pc.DrawCircle(pos.X, pos.Y, size)
	pc.FillStrokeClear()
}

func DrawSquare(pc *paint.Context, pos math32.Vector2, size float32) {
	x := size * 0.9
	pc.MoveTo(pos.X-x, pos.Y-x)
	pc.LineTo(pos.X+x, pos.Y-x)
	pc.LineTo(pos.X+x, pos.Y+x)
	pc.LineTo(pos.X-x, pos.Y+x)
	pc.ClosePath()
	pc.Stroke()
}

func DrawBox(pc *paint.Context, pos math32.Vector2, size float32) {
	x := size * 0.9
	pc.MoveTo(pos.X-x, pos.Y-x)
	pc.LineTo(pos.X+x, pos.Y-x)
	pc.LineTo(pos.X+x, pos.Y+x)
	pc.LineTo(pos.X-x, pos.Y+x)
	pc.ClosePath()
	pc.FillStrokeClear()
}

func DrawTriangle(pc *paint.Context, pos math32.Vector2, size float32) {
	x := size * 0.9
	pc.MoveTo(pos.X, pos.Y-x)
	pc.LineTo(pos.X-x, pos.Y+x)
	pc.LineTo(pos.X+x, pos.Y+x)
	pc.ClosePath()
	pc.Stroke()
}

func DrawPyramid(pc *paint.Context, pos math32.Vector2, size float32) {
	x := size * 0.9
	pc.MoveTo(pos.X, pos.Y-x)
	pc.LineTo(pos.X-x, pos.Y+x)
	pc.LineTo(pos.X+x, pos.Y+x)
	pc.ClosePath()
	pc.FillStrokeClear()
}

func DrawPlus(pc *paint.Context, pos math32.Vector2, size float32) {
	x := size * 1.05
	pc.MoveTo(pos.X-x, pos.Y)
	pc.LineTo(pos.X+x, pos.Y)
	pc.MoveTo(pos.X, pos.Y-x)
	pc.LineTo(pos.X, pos.Y+x)
	pc.ClosePath()
	pc.Stroke()
}

func DrawCross(pc *paint.Context, pos math32.Vector2, size float32) {
	x := size * 0.9
	pc.MoveTo(pos.X-x, pos.Y-x)
	pc.LineTo(pos.X+x, pos.Y+x)
	pc.MoveTo(pos.X+x, pos.Y-x)
	pc.LineTo(pos.X-x, pos.Y+x)
	pc.ClosePath()
	pc.Stroke()
}
