// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Note: this is based on https://github.com/srwiley/rasterx
// Copyright 2018 All rights reserved.
// Created: 5/12/2018 by S.R.Wiley

package colors

import (
	"image"

	"image/color"

	"github.com/srwiley/rasterx"
	"goki.dev/mat32/v2"
)

// Gradient represents a linear or radial gradient.
type Gradient struct {

	// whether the gradient is a radial gradient (as opposed to a linear one)
	Radial bool `desc:"whether the gradient is a radial gradient (as opposed to a linear one)"`

	// the bounds for linear gradients (x1, y1, x2, and y2 in SVG)
	Bounds mat32.Box2 `desc:"the bounds for linear gradients (x1, y1, x2, and y2 in SVG)"`

	// the center point for radial gradients (cx and cy in SVG)
	Center mat32.Vec2 `desc:"the center point for radial gradients (cx and cy in SVG)"`

	// the focal point for radial gradients (fx and fy in SVG)
	Focal mat32.Vec2 `desc:"the focal point for radial gradients (fx and fy in SVG)"`

	// the radius for radial gradients (r in SVG)
	Radius float32 `desc:"the radius for radial gradients (r in SVG)"`

	// the stops of the gradient
	Stops []GradientStop `desc:"the stops of the gradient"`

	// the matrix for the gradient
	Matrix mat32.Mat2 `desc:"the matrix for the gradient"`

	// the spread methods for the gradient
	Spread SpreadMethods `desc:"the spread methods for the gradient"`

	// the units for the gradient
	Units GradientUnits `desc:"the units for the gradient"`
}

// GradientStop represents a gradient stop in the SVG 2.0 gradient specification
type GradientStop struct {
	Color   color.RGBA // the color of the stop
	Offset  float32    // the offset (position) of the stop
	Opacity float32    // the opacity of the stop
}

// SpreadMethods are the methods used when a gradient reaches
// its end but the object isn't fully filled.
type SpreadMethods int32 //enums:enum

const (
	// PadSpread indicates to have the final color of the gradient fill
	// the object beyond the end of the gradient.
	PadSpread SpreadMethods = iota
	// ReflectSpread indicates to have a gradient repeat in reverse order
	// (offset 1 to 0) to fully fill an object beyond the end of the gradient.
	ReflectSpread
	// RepeatSpread indicates to have a gradient continue in its original order
	// (offset 0 to 1) by jumping back to the start to fully fill an object beyond
	// the end of the gradient.
	RepeatSpread
)

// GradientUnits are the types of SVG gradient units
type GradientUnits int32 //enums:enum

const (
	// ObjectBoundingBox indicates that coordinate values are scaled
	// relative to the size of the object and are specified in the range
	// of 0 to 1.
	ObjectBoundingBox GradientUnits = iota
	// UserSpaceOnUse indicates that coordinate values are specified
	// in the current user coordinate system when the gradient is used.
	UserSpaceOnUse
)

// LinearGradient returns a new linear gradient
func LinearGradient() *Gradient {
	return &Gradient{
		Spread: PadSpread,
		Matrix: mat32.Identity2D(),
		Bounds: mat32.NewBox2(mat32.Vec2{}, mat32.Vec2{0, 1}),
	}
}

// RadialGradient returns a new radial gradient
func RadialGradient() *Gradient {
	return &Gradient{
		Radial: true,
		Spread: PadSpread,
		Matrix: mat32.Identity2D(),
		Center: mat32.Vec2{0.5, 0.5},
		Focal:  mat32.Vec2{0.5, 0.5},
		Radius: 0.5,
	}
}

func (g *Gradient) SetRadial() *Gradient {
	g.Radial = true
	return g
}

func (g *Gradient) SetLinear() *Gradient {
	g.Radial = false
	return g
}

// SetBounds sets the bounds to the given value. It is just
// a simple method that sets the value, and should not be confused
// with [Gradient.SetUserBounds].
func (g *Gradient) SetBounds(bounds mat32.Box2) *Gradient {
	g.Bounds = bounds
	return g
}

func (g *Gradient) SetCenter(center mat32.Vec2) *Gradient {
	g.Center = center
	return g
}

func (g *Gradient) SetFocal(focal mat32.Vec2) *Gradient {
	g.Focal = focal
	return g
}

func (g *Gradient) SetRadius(radius float32) *Gradient {
	g.Radius = radius
	return g
}

func (g *Gradient) AddStop(color color.RGBA, offset, opacity float32) *Gradient {
	g.Stops = append(g.Stops, GradientStop{Color: color, Offset: offset, Opacity: opacity})
	return g
}

func (g *Gradient) SetMatrix(matrix mat32.Mat2) *Gradient {
	g.Matrix = matrix
	return g
}

func (g *Gradient) SetSpread(spread SpreadMethods) *Gradient {
	g.Spread = spread
	return g
}

func (g *Gradient) SetUnits(units GradientUnits) *Gradient {
	g.Units = units
	return g
}

// CopyFrom copies from the given gradient, making new copies
// of the stops instead of re-using pointers
func (g *Gradient) CopyFrom(cp *Gradient) {
	*g = *cp
	if cp.Stops != nil {
		g.Stops = make([]GradientStop, len(cp.Stops))
		copy(g.Stops, cp.Stops)
	}
}

// CopyStopsFrom copies the gradient stops from the given gradient,
// if both have gradient stops
func (g *Gradient) CopyStopsFrom(cp *Gradient) {
	if len(g.Stops) == 0 || len(cp.Stops) == 0 {
		return
	}
	if len(g.Stops) != len(cp.Stops) {
		g.Stops = make([]GradientStop, len(cp.Stops))
	}
	copy(g.Stops, cp.Stops)
}

// SetGradientPoints sets the bounds of the gradient based on the given bounding
// box, taking into account radial gradients and a standard linear left-to-right
// gradient direction. It also sets the type of units to [UserSpaceOnUse].
func (g *Gradient) SetUserBounds(bbox mat32.Box2) {
	g.Units = UserSpaceOnUse
	if g.Radial {
		g.Center = bbox.Min.Add(bbox.Max).MulScalar(.5)
		g.Focal = g.Center
		g.Radius = 0.5 * mat32.Max(bbox.Max.X-bbox.Min.X, bbox.Max.Y-bbox.Min.Y)
	} else {
		g.Bounds = bbox
		g.Bounds.Max.Y = g.Bounds.Min.Y // linear L-R
	}
}

// Points returns the points of the gradient as an array of 5 floats.
// If the gradient is radial, the points are of the form:
//
//	[cx, cy, fx, fy, r]
//
// If the gradient is linear, the points are of the form:
//
//	[x1, y1, x2, y2, 0]
func (g *Gradient) Points() [5]float64 {
	if g.Radial {
		return [5]float64{float64(g.Center.X), float64(g.Center.Y), float64(g.Focal.X), float64(g.Focal.Y), float64(g.Radius)}
	}
	return [5]float64{float64(g.Bounds.Min.X), float64(g.Bounds.Min.Y), float64(g.Bounds.Max.X), float64(g.Bounds.Max.Y), 0}
}

// Rasterx returns the gradient as a [rasterx.Gradient]
func (g *Gradient) Rasterx() *rasterx.Gradient {
	r := &rasterx.Gradient{
		Points: g.Points(),
		Stops:  make([]rasterx.GradStop, len(g.Stops)),
		// one might call this ridiculous, and they would be absolutely correct,
		// but we don't have control over the rasterx source code, and
		// https://github.com/golang/go/issues/12854 hasn't been approved
		Bounds: struct {
			X float64
			Y float64
			W float64
			H float64
		}{
			X: float64(g.Bounds.Min.X),
			Y: float64(g.Bounds.Min.Y),
			W: float64(g.Bounds.Size().X),
			H: float64(g.Bounds.Size().Y),
		},
		Matrix:   MatToRasterx(&g.Matrix),
		Spread:   rasterx.SpreadMethod(g.Spread), // we have the same constant values, so this is okay
		Units:    rasterx.GradientUnits(g.Units), // we have the same constant values, so this is okay
		IsRadial: g.Radial,
	}
	for i, stop := range g.Stops {
		r.Stops[i] = stop.Rasterx()
	}
	return r
}

// MatToRasterx converts the given [mat32.Mat2] to a [rasterx.Matrix2D]
func MatToRasterx(mat *mat32.Mat2) rasterx.Matrix2D {
	return rasterx.Matrix2D{float64(mat.XX), float64(mat.YX), float64(mat.XY), float64(mat.YY), float64(mat.X0), float64(mat.Y0)}
}

// RasterxToMat converts the given [rasterx.Matrix2D] to a [mat32.Mat2]
func RasterxToMat(mat *rasterx.Matrix2D) mat32.Mat2 {
	return mat32.Mat2{float32(mat.A), float32(mat.B), float32(mat.C), float32(mat.D), float32(mat.E), float32(mat.F)}
}

// Rasterx returns the gradient stop as a [rasterx.GradStop]
func (g *GradientStop) Rasterx() rasterx.GradStop {
	return rasterx.GradStop{
		StopColor: g.Color,
		Offset:    float64(g.Offset),
		Opacity:   float64(g.Opacity),
	}
}

// RenderColor returns the color or [rasterx.ColorFunc] for rendering, applying
// the given opacity and bounds.
func (g *Gradient) RenderColor(opacity float32, bounds image.Rectangle, xform mat32.Mat2) any {
	box := mat32.Box2{}
	box.SetFromRect(bounds)
	g.SetUserBounds(box)
	r := g.Rasterx()
	return r.GetColorFunctionUS(float64(opacity), MatToRasterx(&xform))
}

// ApplyXForm transforms the points for the gradient if it has
// [UserSpaceOnUse] units, using the given xform matrix.
func (g *Gradient) ApplyXForm(xf mat32.Mat2) {
	if g.Units == ObjectBoundingBox {
		return
	}
	rot := xf.ExtractRot()
	if g.Radial || rot != 0 || !g.Matrix.IsIdentity() { // radial uses transform instead of points
		g.Matrix = g.Matrix.Mul(xf)
	} else {
		g.Bounds.Min = xf.MulVec2AsPt(g.Bounds.Min)
		g.Bounds.Max = xf.MulVec2AsPt(g.Bounds.Max)
	}
}

// ApplyXFormPt transforms the points for the gradient if it has
// [UserSpaceOnUse] units, using the given xform matrix and center point.
func (g *Gradient) ApplyXFormPt(xf mat32.Mat2, pt mat32.Vec2) {
	if g.Units == ObjectBoundingBox {
		return
	}
	rot := xf.ExtractRot()
	if g.Radial || rot != 0 || !g.Matrix.IsIdentity() { // radial uses transform instead of points
		g.Matrix = g.Matrix.MulCtr(xf, pt)
	} else {
		g.Bounds.Min = xf.MulVec2AsPtCtr(g.Bounds.Min, pt)
		g.Bounds.Max = xf.MulVec2AsPtCtr(g.Bounds.Max, pt)
	}
}
