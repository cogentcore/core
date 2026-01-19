// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import (
	"fmt"
	"testing"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
)

func TestEllipse(t *testing.T) {
	tolEqualVec2(t, EllipsePos(2.0, 1.0, math32.Pi/2.0, 1.0, 0.5, 0.0), math32.Vector2{1.0, 2.5})
	tolEqualVec2(t, EllipseDeriv(2.0, 1.0, math32.Pi/2.0, true, 0.0), math32.Vector2{-1.0, 0.0})
	tolEqualVec2(t, EllipseDeriv(2.0, 1.0, math32.Pi/2.0, false, 0.0), math32.Vector2{1.0, 0.0})

	assert.InDelta(t, EllipseRadiiCorrection(math32.Vector2{0.0, 0.0}, 0.1, 0.1, 0.0, math32.Vector2{1.0, 0.0}), 5.0, 1.0e-5)
}

func TestEllipseToCenter(t *testing.T) {
	var tests = []struct {
		x1, y1       float32
		rx, ry, phi  float32
		large, sweep bool
		x2, y2       float32

		cx, cy, theta0, theta1 float32
	}{
		{0.0, 0.0, 2.0, 2.0, 0.0, false, false, 2.0, 2.0, 2.0, 0.0, math32.Pi, math32.Pi / 2.0},
		{0.0, 0.0, 2.0, 2.0, 0.0, true, false, 2.0, 2.0, 0.0, 2.0, math32.Pi * 3.0 / 2.0, 0.0},
		{0.0, 0.0, 2.0, 2.0, 0.0, true, true, 2.0, 2.0, 2.0, 0.0, math32.Pi, math32.Pi * 5.0 / 2.0},
		{0.0, 0.0, 2.0, 1.0, math32.Pi / 2.0, false, false, 1.0, 2.0, 1.0, 0.0, math32.Pi / 2.0, 0.0},

		// radius correction
		{0.0, 0.0, 0.1, 0.1, 0.0, false, false, 1.0, 0.0, 0.5, 0.0, math32.Pi, 0.0},

		// start == end
		{0.0, 0.0, 1.0, 1.0, 0.0, false, false, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0},

		// precision issues
		{8.2, 18.0, 0.2, 0.2, 0.0, false, true, 7.8, 18.0, 8.0, 18.0, 0.0, math32.Pi},
		{7.8, 18.0, 0.2, 0.2, 0.0, false, true, 8.2, 18.0, 8.0, 18.0, math32.Pi, 2.0 * math32.Pi},

		// bugs
		{-1.0 / math32.Sqrt(2), 0.0, 1.0, 1.0, 0.0, false, false, 1.0 / math32.Sqrt(2.0), 0.0, 0.0, -1.0 / math32.Sqrt(2.0), 3.0 / 4.0 * math32.Pi, 1.0 / 4.0 * math32.Pi},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("(%g,%g) %g %g %g %v %v (%g,%g)", tt.x1, tt.y1, tt.rx, tt.ry, tt.phi, tt.large, tt.sweep, tt.x2, tt.y2), func(t *testing.T) {
			cx, cy, theta0, theta1 := EllipseToCenter(tt.x1, tt.y1, tt.rx, tt.ry, tt.phi, tt.large, tt.sweep, tt.x2, tt.y2)
			tolassert.EqualTolSlice(t, []float32{cx, cy, theta0, theta1}, []float32{tt.cx, tt.cy, tt.theta0, tt.theta1}, 1.0e-2)
		})
	}

	//cx, cy, theta0, theta1 := EllipseToCenter(0.0, 0.0, 2.0, 2.0, 0.0, false, false, 2.0, 2.0)
	//test.Float(t, cx, 2.0)
	//test.Float(t, cy, 0.0)
	//test.Float(t, theta0, math32.Pi)
	//test.Float(t, theta1, math32.Pi/2.0)

	//cx, cy, theta0, theta1 = EllipseToCenter(0.0, 0.0, 2.0, 2.0, 0.0, true, false, 2.0, 2.0)
	//test.Float(t, cx, 0.0)
	//test.Float(t, cy, 2.0)
	//test.Float(t, theta0, math32.Pi*3.0/2.0)
	//test.Float(t, theta1, 0.0)

	//cx, cy, theta0, theta1 = EllipseToCenter(0.0, 0.0, 2.0, 2.0, 0.0, true, true, 2.0, 2.0)
	//test.Float(t, cx, 2.0)
	//test.Float(t, cy, 0.0)
	//test.Float(t, theta0, math32.Pi)
	//test.Float(t, theta1, math32.Pi*5.0/2.0)

	//cx, cy, theta0, theta1 = EllipseToCenter(0.0, 0.0, 2.0, 1.0, math32.Pi/2.0, false, false, 1.0, 2.0)
	//test.Float(t, cx, 1.0)
	//test.Float(t, cy, 0.0)
	//test.Float(t, theta0, math32.Pi/2.0)
	//test.Float(t, theta1, 0.0)

	//cx, cy, theta0, theta1 = EllipseToCenter(0.0, 0.0, 0.1, 0.1, 0.0, false, false, 1.0, 0.0)
	//test.Float(t, cx, 0.5)
	//test.Float(t, cy, 0.0)
	//test.Float(t, theta0, math32.Pi)
	//test.Float(t, theta1, 0.0)

	//cx, cy, theta0, theta1 = EllipseToCenter(0.0, 0.0, 1.0, 1.0, 0.0, false, false, 0.0, 0.0)
	//test.Float(t, cx, 0.0)
	//test.Float(t, cy, 0.0)
	//test.Float(t, theta0, 0.0)
	//test.Float(t, theta1, 0.0)
}

func TestArcToQuad(t *testing.T) {
	assert.InDeltaSlice(t, ArcToQuad(math32.Vector2{0.0, 0.0}, 100.0, 100.0, 0.0, false, false, math32.Vector2{200.0, 0.0}), MustParseSVGPath("Q0 100 100 100Q200 100 200 0"), 1.0e-4)
}

func TestArcToCube(t *testing.T) {
	// defer setEpsilon(1e-3)()
	assert.InDeltaSlice(t, ArcToCube(math32.Vector2{0.0, 0.0}, 100.0, 100.0, 0.0, false, false, math32.Vector2{200.0, 0.0}), MustParseSVGPath("C0 54.858 45.142 100 100 100C154.858 100 200 54.858 200 0"), 1.0e-3)
}

func TestQuadraticBezier(t *testing.T) {
	p1, p2 := QuadraticToCubicBezier(math32.Vector2{0.0, 0.0}, math32.Vector2{1.5, 0.0}, math32.Vector2{3.0, 0.0})
	tolEqualVec2(t, p1, math32.Vector2{1.0, 0.0})
	tolEqualVec2(t, p2, math32.Vector2{2.0, 0.0})

	p1, p2 = QuadraticToCubicBezier(math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{1.0, 1.0})
	tolEqualVec2(t, p1, math32.Vector2{2.0 / 3.0, 0.0})
	tolEqualVec2(t, p2, math32.Vector2{1.0, 1.0 / 3.0})
}

func TestQuadraticBezierDeriv(t *testing.T) {
	var tests = []struct {
		p0, p1, p2 math32.Vector2
		t          float32
		q          math32.Vector2
	}{
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{1.0, 1.0}, 0.0, math32.Vector2{2.0, 0.0}},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{1.0, 1.0}, 0.5, math32.Vector2{1.0, 1.0}},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{1.0, 1.0}, 1.0, math32.Vector2{0.0, 2.0}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v--%v", tt.p0, tt.p1, tt.p2, tt.t), func(t *testing.T) {
			q := QuadraticBezierDeriv(tt.p0, tt.p1, tt.p2, tt.t)
			tolEqualVec2(t, q, tt.q, 1.0e-5)
		})
	}
}

func TestCubicBezierDeriv(t *testing.T) {
	p0, p1, p2, p3 := math32.Vector2{0.0, 0.0}, math32.Vector2{2.0 / 3.0, 0.0}, math32.Vector2{1.0, 1.0 / 3.0}, math32.Vector2{1.0, 1.0}
	var tests = []struct {
		p0, p1, p2, p3 math32.Vector2
		t              float32
		q              math32.Vector2
	}{
		{p0, p1, p2, p3, 0.0, math32.Vector2{2.0, 0.0}},
		{p0, p1, p2, p3, 0.5, math32.Vector2{1.0, 1.0}},
		{p0, p1, p2, p3, 1.0, math32.Vector2{0.0, 2.0}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v%v--%v", tt.p0, tt.p1, tt.p2, tt.p3, tt.t), func(t *testing.T) {
			q := CubicBezierDeriv(tt.p0, tt.p1, tt.p2, tt.p3, tt.t)
			tolEqualVec2(t, q, tt.q, 1.0e-5)
		})
	}
}
