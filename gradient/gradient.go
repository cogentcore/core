// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

// Package gradient provides linear, radial, and conic color gradients.
package gradient

import (
	"image/color"
	"log/slog"
	"sort"

	"goki.dev/mat32/v2"
)

// Gradient represents a linear or radial gradient.
type Gradient struct { //gti:add -setters

	// the type of gradient (linear, radial, or conic)
	Type GradientTypes

	// the starting point for linear gradients (x1 and y1 in SVG)
	Start mat32.Vec2

	// the ending point for linear gradients (x2 and y2 in SVG)
	End mat32.Vec2

	// the center point for radial and conic gradients (cx and cy in SVG)
	Center mat32.Vec2

	// the focal point for radial gradients (fx and fy in SVG)
	Focal mat32.Vec2

	// the radius for radial gradients (r in SVG)
	Radius float32

	// the starting clockwise rotation of conic gradients (0-1) (<angle> in css)
	Rotation float32

	// the stops of the gradient
	Stops []Stop

	// the spread method used for the gradient
	Spread SpreadMethods

	// the units used for the gradient
	Units GradientUnits

	// the colorspace algorithm to use for blending colors
	Blend BlendTypes

	// the bounds of the gradient; this should typically not be set by end-users
	Bounds mat32.Box2

	// the matrix for the gradient; this should typically not be set by end-users
	Matrix mat32.Mat2
}

// Stop represents a single stop in a gradient
type Stop struct {

	// the color of the stop
	Color color.RGBA

	// the position of the stop between 0 and 1
	Pos float32
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

// NewLinearGradient returns a new linear gradient
func NewLinearGradient() *Gradient {
	return &Gradient{
		Type:   LinearGradient,
		Spread: PadSpread,
		End:    mat32.Vec2{0, 1},
		Matrix: mat32.Identity2D(),
		Bounds: mat32.NewBox2(mat32.Vec2{}, mat32.Vec2{1, 1}),
	}
}

// NewRadialGradient returns a new radial gradient
func NewRadialGradient() *Gradient {
	return &Gradient{
		Type:   RadialGradient,
		Spread: PadSpread,
		Matrix: mat32.Identity2D(),
		Center: mat32.Vec2{0.5, 0.5},
		Focal:  mat32.Vec2{0.5, 0.5},
		Radius: 0.5,
		Bounds: mat32.NewBox2(mat32.Vec2{}, mat32.Vec2{1, 1}),
	}
}

// AddStop adds a new stop with the given color, offset, and opacity to the gradient.
func (g *Gradient) AddStop(color color.RGBA, offset, opacity float32) *Gradient {
	g.Stops = append(g.Stops, Stop{Color: color, Pos: offset, Opacity: opacity})
	return g
}

// CopyFrom copies from the given gradient, making new copies
// of the stops instead of re-using pointers
func (g *Gradient) CopyFrom(cp *Gradient) {
	*g = *cp
	if cp.Stops != nil {
		g.Stops = make([]Stop, len(cp.Stops))
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
		g.Stops = make([]Stop, len(cp.Stops))
	}
	copy(g.Stops, cp.Stops)
}

// SetGradientPoints sets the bounds of the gradient based on the given bounding
// box, taking into account radial gradients and a standard linear left-to-right
// gradient direction. It also sets the type of units to [UserSpaceOnUse].
func (g *Gradient) SetUserBounds(bbox mat32.Box2) {
	g.Bounds = bbox
	g.Units = UserSpaceOnUse
	switch g.Type {
	case LinearGradient:
		g.Start = bbox.Min
		g.End = bbox.Max
		// default is linear left-to-right, so we keep the starting and ending Y the same
		g.End.Y = g.Start.Y
	case RadialGradient:
		g.Center = bbox.Min.Add(bbox.Max).MulScalar(.5)
		g.Focal = g.Center
		g.Radius = 0.5 * max(bbox.Size().X, bbox.Size().Y)
	case ConicGradient:
		g.Center = bbox.Min.Add(bbox.Max).MulScalar(.5)
	}
}

// RenderColor returns the [Render] color for rendering, applying the given opacity.
func (g *Gradient) RenderColor(opacity float32) Render {
	return g.RenderColorTransform(opacity, mat32.Identity2D())
}

const epsilonF = 1e-5

// RenderColorTransform returns the render color using the given user space object transform matrix
func (g *Gradient) RenderColorTransform(opacity float32, objMatrix mat32.Mat2) Render {
	switch len(g.Stops) {
	case 0:
		return SolidRender(ApplyOpacity(color.RGBA{0, 0, 0, 255}, opacity)) // default error color for gradient w/o stops.
	case 1:
		return SolidRender(ApplyOpacity(g.Stops[0].Color, opacity)) // Illegal, I think, should really should not happen.
	}

	// sort by offset in ascending order
	sort.Slice(g.Stops, func(i, j int) bool {
		return g.Stops[i].Pos < g.Stops[j].Pos
	})

	w, h := g.Bounds.Size().X, g.Bounds.Size().Y
	oriX, oriY := g.Bounds.Min.X, g.Bounds.Min.Y
	gradT := mat32.Identity2D().Translate(oriX, oriY).Scale(w, h).
		Mul(g.Matrix).Scale(1/w, 1/h).Translate(-oriX, -oriY).Inverse()

	switch g.Type {
	case LinearGradient:
		s, e := g.Start, g.End
		if g.Units == ObjectBoundingBox {
			s = g.Bounds.Min.Add(g.Bounds.Size().Mul(s))
			e = g.Bounds.Min.Add(g.Bounds.Size().Mul(e))

			d := e.Sub(s)
			dd := d.X*d.X + d.Y*d.Y // self inner prod
			return FuncRender(func(x, y int) color.Color {
				pt := gradT.MulVec2AsPt(mat32.Vec2{float32(x) + 0.5, float32(y) + 0.5})
				df := pt.Sub(s)
				return g.ColorAt((d.X*df.X+d.Y*df.Y)/dd, opacity)
			})
		}

		s = g.Matrix.MulVec2AsPt(s)
		e = g.Matrix.MulVec2AsPt(e)
		s = objMatrix.MulVec2AsPt(s)
		e = objMatrix.MulVec2AsPt(e)
		d := e.Sub(s)
		dd := d.X*d.X + d.Y*d.Y
		return FuncRender(func(x, y int) color.Color {
			pt := mat32.Vec2{float32(x) + 0.5, float32(y) + 0.5}
			df := pt.Sub(s)
			return g.ColorAt((d.X*df.X+d.Y*df.Y)/dd, opacity)
		})
	case RadialGradient:
		c, f, r := g.Center, g.Focal, mat32.NewVec2Scalar(g.Radius)
		if g.Units == ObjectBoundingBox {
			c = g.Bounds.Min.Add(g.Bounds.Size().Mul(c))
			f = g.Bounds.Min.Add(g.Bounds.Size().Mul(f))
			r.SetMul(g.Bounds.Size())
		} else {
			c = g.Matrix.MulVec2AsPt(c)
			f = g.Matrix.MulVec2AsPt(f)
			r = g.Matrix.MulVec2AsVec(r)

			c = objMatrix.MulVec2AsPt(c)
			f = objMatrix.MulVec2AsPt(f)
			r = objMatrix.MulVec2AsVec(r)
		}

		if c == f {
			// When the focus and center are the same things are much simpler;
			// t is just distance from center
			// scaled by the bounds aspect ratio times r
			if g.Units == ObjectBoundingBox {
				return FuncRender(func(x, y int) color.Color {
					pt := gradT.MulVec2AsPt(mat32.Vec2{float32(x) + 0.5, float32(y) + 0.5})
					d := pt.Sub(c)
					return g.ColorAt(mat32.Sqrt(d.X*d.X/(r.X*r.X)+(d.Y*d.Y)/(r.Y*r.Y)), opacity)
				})
			}
			return FuncRender(func(x, y int) color.Color {
				pt := mat32.Vec2{float32(x) + 0.5, float32(y) + 0.5}
				d := pt.Sub(c)
				return g.ColorAt(mat32.Sqrt(d.X*d.X/(r.X*r.X)+(d.Y*d.Y)/(r.Y*r.Y)), opacity)
			})
		}
		f.SetDiv(r)
		c.SetDiv(r)

		df := f.Sub(c)

		if df.X*df.X+df.Y*df.Y > 1 { // Focus outside of circle; use intersection
			// point of line from center to focus and circle as per SVG specs.
			nf, intersects := RayCircleIntersectionF(f, c, c, 1.0-epsilonF)
			f = nf
			if !intersects {
				return SolidRender(FromRGB(255, 255, 0)) // should not happen
			}
		}
		if g.Units == ObjectBoundingBox {
			return FuncRender(func(x, y int) color.Color {
				pt := gradT.MulVec2AsPt(mat32.Vec2{float32(x) + 0.5, float32(y) + 0.5})
				e := pt.Div(r)

				t1, intersects := RayCircleIntersectionF(e, f, c, 1)
				if !intersects { // In this case, use the last stop color
					s := g.Stops[len(g.Stops)-1]
					return ApplyOpacity(s.Color, s.Opacity*opacity)
				}
				td := t1.Sub(f)
				d := e.Sub(f)
				if td.X*td.X+td.Y*td.Y < epsilonF {
					s := g.Stops[len(g.Stops)-1]
					return ApplyOpacity(s.Color, s.Opacity*opacity)
				}
				return g.ColorAt(mat32.Sqrt(d.X*d.X+d.Y*d.Y)/mat32.Sqrt(td.X*td.X+td.Y*td.Y), opacity)
			})
		}
		return FuncRender(func(x, y int) color.Color {
			pt := mat32.Vec2{float32(x) + 0.5, float32(y) + 0.5}
			e := pt.Div(r)

			t1, intersects := RayCircleIntersectionF(e, f, c, 1)
			if !intersects { // In this case, use the last stop color
				s := g.Stops[len(g.Stops)-1]
				return ApplyOpacity(s.Color, s.Opacity*opacity)
			}
			td := t1.Sub(f)
			d := e.Sub(f)
			if td.X*td.X+td.Y*td.Y < epsilonF {
				s := g.Stops[len(g.Stops)-1]
				return ApplyOpacity(s.Color, s.Opacity*opacity)
			}
			return g.ColorAt(mat32.Sqrt(d.X*d.X+d.Y*d.Y)/mat32.Sqrt(td.X*td.X+td.Y*td.Y), opacity)
		})
	case ConicGradient:

	}
	slog.Error("got unexpected gradient type", "type", g.Type)
	return Render{}
}

// ApplyTransform transforms the points for the gradient if it has
// [UserSpaceOnUse] units, using the given transform matrix.
func (g *Gradient) ApplyTransform(xf mat32.Mat2) {
	if g.Units == ObjectBoundingBox {
		return
	}
	rot := xf.ExtractRot()
	if g.Type == RadialGradient || rot != 0 || !g.Matrix.IsIdentity() { // radial uses transform instead of points
		g.Matrix = g.Matrix.Mul(xf)
	} else {
		g.Bounds.Min = xf.MulVec2AsPt(g.Bounds.Min)
		g.Bounds.Max = xf.MulVec2AsPt(g.Bounds.Max)
	}
}

// ApplyTransformPt transforms the points for the gradient if it has
// [UserSpaceOnUse] units, using the given transform matrix and center point.
func (g *Gradient) ApplyTransformPt(xf mat32.Mat2, pt mat32.Vec2) {
	if g.Units == ObjectBoundingBox {
		return
	}
	rot := xf.ExtractRot()
	if g.Type == RadialGradient || rot != 0 || !g.Matrix.IsIdentity() { // radial uses transform instead of points
		g.Matrix = g.Matrix.MulCtr(xf, pt)
	} else {
		g.Bounds.Min = xf.MulVec2AsPtCtr(g.Bounds.Min, pt)
		g.Bounds.Max = xf.MulVec2AsPtCtr(g.Bounds.Max, pt)
	}
}

// RayCircleIntersectionF calculates in floating point the points of intersection of
// a ray starting at s2 passing through s1 and a circle in fixed point.
// Returns intersects == false if no solution is possible. If two
// solutions are possible, the point closest to s2 is returned
func RayCircleIntersectionF(s1, s2, c mat32.Vec2, r float32) (pt mat32.Vec2, intersects bool) {
	n := s2.X - c.X // Calculating using 64* rather than divide
	m := s2.Y - c.Y

	e := s2.X - s1.X
	d := s2.Y - s1.Y

	// Quadratic normal form coefficients
	A, B, C := e*e+d*d, -2*(e*n+m*d), n*n+m*m-r*r

	D := B*B - 4*A*C

	if D <= 0 {
		return // No intersection or is tangent
	}

	D = mat32.Sqrt(D)
	t1, t2 := (-B+D)/(2*A), (-B-D)/(2*A)
	p1OnSide := t1 > 0
	p2OnSide := t2 > 0

	switch {
	case p1OnSide && p2OnSide:
		if t2 < t1 { // both on ray, use closest to s2
			t1 = t2
		}
	case p2OnSide: // Only p2 on ray
		t1 = t2
	case p1OnSide: // only p1 on ray
	default: // Neither solution is on the ray
		return
	}
	return mat32.Vec2{(n - e*t1) + c.X, (m - d*t1) + c.Y}, true
}
