// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Note: this is based on https://github.com/srwiley/rasterx
// Copyright 2018 All rights reserved.
// Created: 5/12/2018 by S.R.Wiley

package colors

import (
	"image/color"
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
	Stops []GradientStop

	// the spread methods for the gradient
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

// GradientStop represents a single stop in a [Gradient]
type GradientStop struct {
	// the color of the stop
	Color color.RGBA

	// the offset (position) of the stop (0-1)
	Offset float32

	// the opacity of the stop (0-1)
	Opacity float32
}

// GradientTypes are the different types of gradients available
// (linear, radial, and conic).
type GradientTypes int32 //enums:enum

const (
	// Linear is a linear gradient
	Linear GradientTypes = iota

	// Radial is a radial gradient
	Radial

	// Conic is a conic gradient
	Conic
)

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

// BlendTypes are different algorithms (colorspaces) to use for blending
// the color stop values in generating the gradients.
type BlendTypes int32 //enums:enum

const (
	// HCT uses hue, chroma, tone space and generally produces the best results
	HCT BlendTypes = iota

	// RGB uses raw RGB space and was used in v1 and is used in most other colormap
	// software, so to reproduce existing results, select this option.
	RGB

	// CAM16 is an alternative colorspace, similar to HCT, but not quite as good.
	CAM16
)

// LinearGradient returns a new linear gradient
func LinearGradient() *Gradient {
	return &Gradient{
		Type:   Linear,
		Spread: PadSpread,
		End:    mat32.Vec2{0, 1},
		Matrix: mat32.Identity2D(),
		Bounds: mat32.NewBox2(mat32.Vec2{}, mat32.Vec2{1, 1}),
	}
}

// RadialGradient returns a new radial gradient
func RadialGradient() *Gradient {
	return &Gradient{
		Type:   Radial,
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
	g.Stops = append(g.Stops, GradientStop{Color: color, Offset: offset, Opacity: opacity})
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
	g.Bounds = bbox
	g.Units = UserSpaceOnUse
	switch g.Type {
	case Linear:
		g.Start = bbox.Min
		g.End = bbox.Max
		// default is linear left-to-right, so we keep the starting and ending Y the same
		g.End.Y = g.Start.Y
	case Radial:
		g.Center = bbox.Min.Add(bbox.Max).MulScalar(.5)
		g.Focal = g.Center
		g.Radius = 0.5 * max(bbox.Size().X, bbox.Size().Y)
	case Conic:
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
		return g.Stops[i].Offset < g.Stops[j].Offset
	})

	w, h := g.Bounds.Size().X, g.Bounds.Size().Y
	oriX, oriY := g.Bounds.Min.X, g.Bounds.Min.Y
	gradT := mat32.Identity2D().Translate(oriX, oriY).Scale(w, h).
		Mul(g.Matrix).Scale(1/w, 1/h).Translate(-oriX, -oriY).Inverse()

	if g.Type == Radial {
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
	}
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
	// if dd == 0.0 {
	// 	fmt.Println("zero delta")
	// }
	return FuncRender(func(x, y int) color.Color {
		pt := mat32.Vec2{float32(x) + 0.5, float32(y) + 0.5}
		df := pt.Sub(s)
		return g.ColorAt((d.X*df.X+d.Y*df.Y)/dd, opacity)
	})
}

// ColorAt takes the given paramaterized value along the gradient's stops and
// returns a color depending on [Gradient.Spread] and [Gradient.Stops].
func (g *Gradient) ColorAt(v, opacity float32) color.Color {
	d := len(g.Stops)
	// These cases can be taken care of early on
	if v >= 1.0 && g.Spread == PadSpread {
		s := g.Stops[d-1]
		return ApplyOpacity(s.Color, s.Opacity*opacity)
	}
	if v <= 0.0 && g.Spread == PadSpread {
		return ApplyOpacity(g.Stops[0].Color, g.Stops[0].Opacity*opacity)
	}

	modRange := float32(1)
	if g.Spread == ReflectSpread {
		modRange = 2
	}
	mod := mat32.Mod(v, modRange)
	if mod < 0 {
		mod += modRange
	}

	place := 0 // Advance to place where mod is greater than the indicated stop
	for place != len(g.Stops) && mod > g.Stops[place].Offset {
		place++
	}
	switch g.Spread {
	case RepeatSpread:
		var s1, s2 GradientStop
		switch place {
		case 0, d:
			s1, s2 = g.Stops[d-1], g.Stops[0]
		default:
			s1, s2 = g.Stops[place-1], g.Stops[place]
		}
		return g.BlendStops(mod, opacity, s1, s2, false)
	case ReflectSpread:
		switch place {
		case 0:
			return ApplyOpacity(g.Stops[0].Color, g.Stops[0].Opacity*opacity)
		case d:
			// Advance to place where mod-1 is greater than the stop indicated by place in reverse of the stop slice.
			// Since this is the reflect spead mode, the mod interval is two, allowing the stop list to be
			// iterated in reverse before repeating the sequence.
			for place != d*2 && mod-1 > (1-g.Stops[d*2-place-1].Offset) {
				place++
			}
			switch place {
			case d:
				s := g.Stops[d-1]
				return ApplyOpacity(s.Color, s.Opacity*opacity)
			case d * 2:
				return ApplyOpacity(g.Stops[0].Color, g.Stops[0].Opacity*opacity)
			default:
				return g.BlendStops(mod-1, opacity,
					g.Stops[d*2-place], g.Stops[d*2-place-1], true)
			}
		default:
			return g.BlendStops(mod, opacity,
				g.Stops[place-1], g.Stops[place], false)
		}
	default: // PadSpread
		switch place {
		case 0:
			return ApplyOpacity(g.Stops[0].Color, g.Stops[0].Opacity*opacity)
		case len(g.Stops):
			s := g.Stops[len(g.Stops)-1]
			return ApplyOpacity(s.Color, s.Opacity*opacity)
		default:
			return g.BlendStops(mod, opacity, g.Stops[place-1], g.Stops[place], false)
		}
	}
}

// BlendStops blends the given two gradient stops together based on the given value and opacity.
func (g *Gradient) BlendStops(v, opacity float32, s1, s2 GradientStop, flip bool) color.Color {
	s1off := s1.Offset
	if s1.Offset > s2.Offset && !flip { // happens in repeat spread mode
		s1off--
		if v > 1 {
			v--
		}
	}
	if s2.Offset == s1off {
		return ApplyOpacity(s2.Color, s2.Opacity)
	}
	if flip {
		v = 1 - v
	}
	tp := (v - s1off) / (s2.Offset - s1off)

	return ApplyOpacity(Blend(g.Blend, 100*(1-tp), s1.Color, s2.Color), (s1.Opacity*(1-tp)+s2.Opacity*tp)*opacity)
}

// ApplyTransform transforms the points for the gradient if it has
// [UserSpaceOnUse] units, using the given transform matrix.
func (g *Gradient) ApplyTransform(xf mat32.Mat2) {
	if g.Units == ObjectBoundingBox {
		return
	}
	rot := xf.ExtractRot()
	if g.Type == Radial || rot != 0 || !g.Matrix.IsIdentity() { // radial uses transform instead of points
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
	if g.Type == Radial || rot != 0 || !g.Matrix.IsIdentity() { // radial uses transform instead of points
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
