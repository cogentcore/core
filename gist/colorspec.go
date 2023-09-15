// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"fmt"
	"image"

	"image/color"

	"github.com/srwiley/rasterx"
	"goki.dev/colors"
	"goki.dev/mat32/v2"
)

// Color defines a standard color object for GUI use, with RGBA values, and
// all the usual necessary conversion functions to / from names, strings, etc

// ColorSpec fully specifies the color for rendering -- used in FillStyle and
// StrokeStyle
type ColorSpec struct {

	// source of color (solid, gradient)
	Source ColorSources `desc:"source of color (solid, gradient)"`

	// color for solid color source
	Color color.RGBA `desc:"color for solid color source"`

	// gradient parameters for gradient color source
	Gradient *rasterx.Gradient `desc:"gradient parameters for gradient color source"`
}

// see colorparse.go for ColorSpec.SetString() method

// ColorSources determine how the color is generated -- used in FillStyle and StrokeStyle
type ColorSources int32 //enums:enum

const (
	SolidColor ColorSources = iota
	LinearGradient
	RadialGradient
)

// GradientPoints defines points within the gradient
type GradientPoints int32 //enums:enum

const (
	GpX1 GradientPoints = iota
	GpY1
	GpX2
	GpY2
)

// IsNil tests for nil solid or gradient colors
func (cs *ColorSpec) IsNil() bool {
	if cs.Source == SolidColor {
		return colors.IsNil(cs.Color)
	}
	return cs.Gradient == nil
}

// ColorOrNil returns the solid color if non-nil, or nil otherwise -- for
// consumers that handle nil colors
func (cs *ColorSpec) ColorOrNil() color.Color {
	if colors.IsNil(cs.Color) {
		return nil
	}
	return cs.Color
}

// SetSolid sets a solid color
func (cs *ColorSpec) SetSolid(cl color.RGBA) {
	cs.Color = cl
	cs.Source = SolidColor
	cs.Gradient = nil
}

// SetColor sets a solid color from a standard [image/color.Color]
func (cs *ColorSpec) SetColor(cl color.Color) {
	cs.Color = colors.AsRGBA(cl)
	cs.Source = SolidColor
	cs.Gradient = nil
}

// SetName sets a solid color by name
func (cs *ColorSpec) SetName(name string) {
	cs.Color = colors.LogFromName(name)
	cs.Source = SolidColor
	cs.Gradient = nil
}

// Copy copies a gradient, making new copies of the stops instead of
// re-using pointers
func (cs *ColorSpec) CopyFrom(cp *ColorSpec) {
	*cs = *cp
	if cp.Gradient != nil {
		cs.Gradient = &rasterx.Gradient{}
		*cs.Gradient = *cp.Gradient
		sn := len(cp.Gradient.Stops)
		cs.Gradient.Stops = make([]rasterx.GradStop, sn)
		copy(cs.Gradient.Stops, cp.Gradient.Stops)
	}
}

// CopyStopsFrom copies gradient stops from other color spec, if both
// have gradient stops
func (cs *ColorSpec) CopyStopsFrom(cp *ColorSpec) {
	if cp.Gradient == nil || cs.Gradient == nil {
		return
	}
	sn := len(cp.Gradient.Stops)
	if sn == 0 {
		return
	}
	if len(cs.Gradient.Stops) != sn {
		cs.Gradient.Stops = make([]rasterx.GradStop, sn)
	}
	copy(cs.Gradient.Stops, cp.Gradient.Stops)
}

// NewLinearGradient creates a new Linear gradient in spec, sets Source
// to LinearGradient.
func (cs *ColorSpec) NewLinearGradient() {
	cs.Source = LinearGradient
	cs.Gradient = &rasterx.Gradient{IsRadial: false, Matrix: rasterx.Identity, Spread: rasterx.PadSpread}
	cs.Gradient.Bounds.W = 1
	cs.Gradient.Bounds.H = 1
}

// NewRadialGradient creates a new Radial gradient in spec, sets Source
// to RadialGradient.
func (cs *ColorSpec) NewRadialGradient() {
	cs.Source = RadialGradient
	cs.Gradient = &rasterx.Gradient{IsRadial: true, Matrix: rasterx.Identity, Spread: rasterx.PadSpread}
	cs.Gradient.Bounds.W = 1
	cs.Gradient.Bounds.H = 1
}

// SetGradientPoints sets UserSpaceOnUse points for gradient based on given bounding box
func (cs *ColorSpec) SetGradientPoints(bbox mat32.Box2) {
	if cs.Gradient == nil {
		return
	}
	cs.Gradient.Units = rasterx.UserSpaceOnUse
	if cs.Gradient.IsRadial {
		ctr := bbox.Min.Add(bbox.Max).MulScalar(.5)
		rad := 0.5 * mat32.Max(bbox.Max.X-bbox.Min.X, bbox.Max.Y-bbox.Min.Y)
		cs.Gradient.Points = [5]float64{float64(ctr.X), float64(ctr.Y), float64(ctr.X), float64(ctr.Y), float64(rad)}
	} else {
		cs.Gradient.Points = [5]float64{float64(bbox.Min.X), float64(bbox.Min.Y), float64(bbox.Max.X), float64(bbox.Min.Y), 0} // linear R-L
	}
}

// SetShadowGradient sets a linear gradient starting at given color and going
// down to transparent based on given color and direction spec (defaults to
// "to down")
func (cs *ColorSpec) SetShadowGradient(cl color.Color, dir string) {
	cs.Color = colors.AsRGBA(cl)
	if dir == "" {
		dir = "to down"
	}
	cs.SetString(fmt.Sprintf("linear-gradient(%v, lighter-0, transparent)", dir), nil)
	cs.Source = LinearGradient
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
func (cs *ColorSpec) RenderColor(opacity float32, bounds image.Rectangle, xform mat32.Mat2) any {
	if cs.Source == SolidColor || cs.Gradient == nil {
		return rasterx.ApplyOpacity(cs.Color, float64(opacity))
	} else {
		if cs.Source == RadialGradient {
			cs.Gradient.IsRadial = true
		} else {
			cs.Gradient.IsRadial = false
		}
		SetGradientBounds(cs.Gradient, bounds)
		return cs.Gradient.GetColorFunctionUS(float64(opacity), MatToRasterx(&xform))
	}
}

// SetIFace sets the color spec from given interface value, e.g., for map[string]any
// key is an optional property key for error -- always logs errors
func (c *ColorSpec) SetIFace(val any, ctxt Context, key string) error {
	switch valv := val.(type) {
	case string:
		c.SetString(valv, ctxt)
	case *color.RGBA:
		c.SetColor(*valv)
	case *ColorSpec:
		*c = *valv
	case color.Color:
		c.SetColor(valv)
	}
	return nil
}

// ApplyXForm transforms the points for a UserSpaceOnUse gradient
func (c *ColorSpec) ApplyXForm(xf mat32.Mat2) {
	if c.Gradient == nil {
		return
	}
	if c.Gradient.Units == rasterx.ObjectBoundingBox {
		return
	}
	mat := RasterxToMat(&c.Gradient.Matrix)
	rot := xf.ExtractRot()
	if c.Gradient.IsRadial || rot != 0 || !mat.IsIdentity() { // radial uses transform instead of points
		mat = mat.Mul(xf)
		c.Gradient.Matrix = MatToRasterx(&mat)
	} else {
		p1 := mat32.Vec2{float32(c.Gradient.Points[0]), float32(c.Gradient.Points[1])}
		p1 = xf.MulVec2AsPt(p1)
		p2 := mat32.Vec2{float32(c.Gradient.Points[2]), float32(c.Gradient.Points[3])}
		p2 = xf.MulVec2AsPt(p2)
		c.Gradient.Points[0] = float64(p1.X)
		c.Gradient.Points[1] = float64(p1.Y)
		c.Gradient.Points[2] = float64(p2.X)
		c.Gradient.Points[3] = float64(p2.Y)
	}
}

// ApplyXFormPt transforms the points for a UserSpaceOnUse gradient
// relative to a given center point
func (c *ColorSpec) ApplyXFormPt(xf mat32.Mat2, pt mat32.Vec2) {
	if c.Gradient == nil {
		return
	}
	if c.Gradient.Units == rasterx.ObjectBoundingBox {
		return
	}
	mat := RasterxToMat(&c.Gradient.Matrix)
	rot := xf.ExtractRot()
	if c.Gradient.IsRadial || rot != 0 || !mat.IsIdentity() { // radial uses transform instead of points
		mat = mat.MulCtr(xf, pt)
		c.Gradient.Matrix = MatToRasterx(&mat)
	} else {
		p1 := mat32.Vec2{float32(c.Gradient.Points[0]), float32(c.Gradient.Points[1])}
		p1 = xf.MulVec2AsPtCtr(p1, pt)
		p2 := mat32.Vec2{float32(c.Gradient.Points[2]), float32(c.Gradient.Points[3])}
		p2 = xf.MulVec2AsPtCtr(p2, pt)
		c.Gradient.Points[0] = float64(p1.X)
		c.Gradient.Points[1] = float64(p1.Y)
		c.Gradient.Points[2] = float64(p2.X)
		c.Gradient.Points[3] = float64(p2.Y)
	}
}
