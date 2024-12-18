// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles/units"
)

// PointStyle has style properties for drawing points as different shapes.
type PointStyle struct { //types:add -setters
	// On indicates whether to plot points.
	On DefaultOffOn

	// Shape to draw.
	Shape Shapes

	// Color is the stroke color image specification.
	// Setting to nil turns line off.
	Color image.Image

	// Fill is the color to fill solid regions, in a plot-specific
	// way (e.g., the area below a Line plot, the bar color).
	// Use nil to disable filling.
	Fill image.Image

	// Width is the line width for point glyphs, with a default of 1 Pt (point).
	// Setting to 0 turns line off.
	Width units.Value

	// Size of shape to draw for each point.
	// Defaults to 4 Pt (point).
	Size units.Value
}

func (ps *PointStyle) Defaults() {
	ps.Color = colors.Scheme.OnSurface
	ps.Fill = colors.Scheme.OnSurface
	ps.Width.Pt(1)
	ps.Size.Pt(4)
}

// SetStroke sets the stroke style in plot paint to current line style.
// returns false if either the Width = 0 or Color is nil
func (ps *PointStyle) SetStroke(pt *Plot) bool {
	if ps.On == Off || ps.Color == nil {
		return false
	}
	pc := pt.Paint
	uc := &pc.UnitContext
	ps.Width.ToDots(uc)
	ps.Size.ToDots(uc)
	if ps.Width.Dots == 0 || ps.Size.Dots == 0 {
		return false
	}
	pc.StrokeStyle.Width = ps.Width
	pc.StrokeStyle.Color = ps.Color
	pc.StrokeStyle.ToDots(uc)
	pc.FillStyle.Color = ps.Fill
	return true
}

// DrawShape draws the given shape
func (ps *PointStyle) DrawShape(pc *paint.Context, pos math32.Vector2) {
	size := ps.Size.Dots
	if size == 0 {
		return
	}
	switch ps.Shape {
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

// Shapes has the options for how to draw points in the plot.
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
