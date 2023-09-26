// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Note: this is based on https://github.com/srwiley/rasterx
// Copyright 2018 All rights reserved.
// Created: 5/12/2018 by S.R.Wiley

package colors

import (
	"fmt"
	"image"

	"image/color"

	"github.com/srwiley/rasterx"
	"goki.dev/mat32/v2"
)

// Gradient represents a linear gradient, radial gradient, or solid color.
type Gradient struct {

	// source of color (solid, linear gradient, radial gradient)
	Source GradientSources `desc:"source of color (solid, linear gradient, radial gradient)"`

	// the solid color for solid gradients
	Color color.RGBA `desc:"the solid color for solid gradients"`

	// the high and low points for linear gradients (x1, x2, y1, and y2 in SVG)
	Points mat32.Box2 `desc:"the high and low points for linear gradients (x1, x2, y1, and y2 in SVG)"`

	// the center point for radial gradients (cx and cy in SVG)
	Center mat32.Vec2 `desc:"the center point for radial gradients (cx and cy in SVG)"`

	// the focal point for radial gradients (fx and fy in SVG)
	Focal mat32.Vec2 `desc:"the focal point for radial gradients (fx and fy in SVG)"`

	// the radius for radial gradients (r in SVG)
	Radius float32 `desc:"the radius for radial gradients (r in SVG)"`

	// the stops of the gradient
	Stops []GradientStop `desc:"the stops of the gradient"`

	// the bounds of the gradient
	Bounds mat32.Box2 `desc:"the bounds of the gradient"`

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
	Offset  float32    // the offset (position) of the stop (0-1)
	Opacity float32    // the opacity of the stop (0-1)
}

// GradientSources represent the ways in which a [Gradient] can be specified.
type GradientSources int32 //enums:enum

const (
	// Solid indicates a solid color.
	Solid GradientSources = iota
	// LinearGradient indicates a linear gradient.
	LinearGradient
	// RadialGradient indicates a radial gradient.
	RadialGradient
)

// SpreadMethods are the methods used when a gradient reaches
// its end but the object isn't fully filled.
type SpreadMethods int32 //enums:enum

const (
	// SpreadPad indicates to have the final color of the gradient fill
	// the object beyond the end of the gradient.
	SpreadPad SpreadMethods = iota
	// SpreadReflect indicates to have a gradient repeat in reverse order
	// (offset 1 to 0) to fully fill an object beyond the end of the gradient.
	SpreadReflect
	// SpreadRepeat indicates to have a gradient continue in its original order
	// (offset 0 to 1) by jumping back to the start to fully fill an object beyond
	// the end of the gradient.
	SpreadRepeat
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

// IsNil returns whether the gradient is effectively nil (has no color).
func (g *Gradient) IsNil() bool {
	if g == nil {
		return true
	}
	if g.Source == Solid {
		return IsNil(g.Color)
	}
	return g.Stops == nil
}

// ColorOrNil returns the solid color if non-nil, or nil otherwise;
// it is for consumers that handle nil colors.
func (g *Gradient) ColorOrNil() color.Color {
	if IsNil(g.Color) {
		return nil
	}
	return g.Color
}

// SetSolid sets the gradient to the given solid color
func (g *Gradient) SetSolid(cl color.RGBA) {
	*g = Gradient{
		Color:  cl,
		Source: Solid,
	}
}

// SetColor sets the gradient to the given solid standard [color.Color]
func (g *Gradient) SetColor(cl color.Color) {
	*g = Gradient{
		Color:  AsRGBA(cl),
		Source: Solid,
	}
}

// SetName sets the gradient to the solid color with the given name
func (g *Gradient) SetName(name string) {
	*g = Gradient{
		Color:  LogFromName(name),
		Source: Solid,
	}
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

// NewLinearGradient sets the gradient to a new linear gradient.
func (g *Gradient) NewLinearGradient() {
	*g = Gradient{
		Source: LinearGradient,
		Spread: SpreadPad,
		Matrix: mat32.Identity2D(),
		Bounds: mat32.NewBox2(mat32.Vec2{}, mat32.Vec2{1, 1}),
	}

}

// NewRadialGradient sets the gradient to a new radial gradient.
func (g *Gradient) NewRadialGradient() {
	g.Source = RadialGradient
	g.Gradient = &rasterx.Gradient{IsRadial: true, Matrix: rasterx.Identity, Spread: rasterx.PadSpread}
	g.Gradient.Bounds.W = 1
	g.Gradient.Bounds.H = 1
}

// SetGradientPoints sets UserSpaceOnUse points for gradient based on given bounding box
func (g *Gradient) SetGradientPoints(bbox mat32.Box2) {
	if g.Gradient == nil {
		return
	}
	g.Gradient.Units = rasterx.UserSpaceOnUse
	if g.Gradient.IsRadial {
		ctr := bbox.Min.Add(bbox.Max).MulScalar(.5)
		rad := 0.5 * mat32.Max(bbox.Max.X-bbox.Min.X, bbox.Max.Y-bbox.Min.Y)
		g.Gradient.Points = [5]float64{float64(ctr.X), float64(ctr.Y), float64(ctr.X), float64(ctr.Y), float64(rad)}
	} else {
		g.Gradient.Points = [5]float64{float64(bbox.Min.X), float64(bbox.Min.Y), float64(bbox.Max.X), float64(bbox.Min.Y), 0} // linear R-L
	}
}

// SetShadowGradient sets a linear gradient starting at given color and going
// down to transparent based on given color and direction spec (defaults to
// "to down")
func (g *Gradient) SetShadowGradient(cl color.Color, dir string) {
	g.Color = AsRGBA(cl)
	if dir == "" {
		dir = "to down"
	}
	g.SetString(fmt.Sprintf("linear-gradient(%v, lighter-0, transparent)", dir), nil)
	g.Source = LinearGradient
}

// SetGradientBounds sets bounds of the gradient
func SetGradientBounds(grad *rasterx.Gradient, bounds image.Rectangle) {
	grad.Bounds.X = float64(bounds.Min.X)
	grad.Bounds.Y = float64(bounds.Min.Y)
	sz := bounds.Size()
	grad.Bounds.W = float64(sz.X)
	grad.Bounds.H = float64(sz.Y)
}

// CopyGradient copies a gradient, making new copies of the stops instead of
// re-using pointers
func CopyGradient(dst, src *rasterx.Gradient) {
	*dst = *src
	sn := len(src.Stops)
	dst.Stops = make([]rasterx.GradStop, sn)
	copy(dst.Stops, src.Stops)
}

func MatToRasterx(mat *mat32.Mat2) rasterx.Matrix2D {
	return rasterx.Matrix2D{float64(mat.XX), float64(mat.YX), float64(mat.XY), float64(mat.YY), float64(mat.X0), float64(mat.Y0)}
}

func RasterxToMat(mat *rasterx.Matrix2D) mat32.Mat2 {
	return mat32.Mat2{float32(mat.A), float32(mat.B), float32(mat.C), float32(mat.D), float32(mat.E), float32(mat.F)}
}

// RenderColor gets the color for rendering, applying opacity and bounds for
// gradients
func (g *Gradient) RenderColor(opacity float32, bounds image.Rectangle, xform mat32.Mat2) any {
	if g.Source == Solid || g.Gradient == nil {
		return rasterx.ApplyOpacity(g.Color, float64(opacity))
	} else {
		if g.Source == RadialGradient {
			g.Gradient.IsRadial = true
		} else {
			g.Gradient.IsRadial = false
		}
		SetGradientBounds(g.Gradient, bounds)
		return g.Gradient.GetColorFunctionUS(float64(opacity), MatToRasterx(&xform))
	}
}

// SetIFace sets the color spec from given interface value, e.g., for map[string]any
// key is an optional property key for error -- always logs errors
func (g *Gradient) SetAny(val any, ctxt Context, key string) error {
	switch valv := val.(type) {
	case string:
		g.SetString(valv, ctxt)
	case *color.RGBA:
		g.SetColor(*valv)
	case *Gradient:
		*g = *valv
	case color.Color:
		g.SetColor(valv)
	}
	return nil
}

// ApplyXForm transforms the points for a UserSpaceOnUse gradient
func (g *Gradient) ApplyXForm(xf mat32.Mat2) {
	if g.Gradient == nil {
		return
	}
	if g.Gradient.Units == rasterx.ObjectBoundingBox {
		return
	}
	mat := RasterxToMat(&g.Gradient.Matrix)
	rot := xf.ExtractRot()
	if g.Gradient.IsRadial || rot != 0 || !mat.IsIdentity() { // radial uses transform instead of points
		mat = mat.Mul(xf)
		g.Gradient.Matrix = MatToRasterx(&mat)
	} else {
		p1 := mat32.Vec2{float32(g.Gradient.Points[0]), float32(g.Gradient.Points[1])}
		p1 = xf.MulVec2AsPt(p1)
		p2 := mat32.Vec2{float32(g.Gradient.Points[2]), float32(g.Gradient.Points[3])}
		p2 = xf.MulVec2AsPt(p2)
		g.Gradient.Points[0] = float64(p1.X)
		g.Gradient.Points[1] = float64(p1.Y)
		g.Gradient.Points[2] = float64(p2.X)
		g.Gradient.Points[3] = float64(p2.Y)
	}
}

// ApplyXFormPt transforms the points for a UserSpaceOnUse gradient
// relative to a given center point
func (g *Gradient) ApplyXFormPt(xf mat32.Mat2, pt mat32.Vec2) {
	if g.Gradient == nil {
		return
	}
	if g.Gradient.Units == rasterx.ObjectBoundingBox {
		return
	}
	mat := RasterxToMat(&g.Gradient.Matrix)
	rot := xf.ExtractRot()
	if g.Gradient.IsRadial || rot != 0 || !mat.IsIdentity() { // radial uses transform instead of points
		mat = mat.MulCtr(xf, pt)
		g.Gradient.Matrix = MatToRasterx(&mat)
	} else {
		p1 := mat32.Vec2{float32(g.Gradient.Points[0]), float32(g.Gradient.Points[1])}
		p1 = xf.MulVec2AsPtCtr(p1, pt)
		p2 := mat32.Vec2{float32(g.Gradient.Points[2]), float32(g.Gradient.Points[3])}
		p2 = xf.MulVec2AsPtCtr(p2, pt)
		g.Gradient.Points[0] = float64(p1.X)
		g.Gradient.Points[1] = float64(p1.Y)
		g.Gradient.Points[2] = float64(p2.X)
		g.Gradient.Points[3] = float64(p2.Y)
	}
}
