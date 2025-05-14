// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package intersect

import (
	"fmt"
	"testing"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"github.com/stretchr/testify/assert"
)

func TestEllipse(t *testing.T) {
	tolEqualVec2(t, ellipseDeriv2(2.0, 1.0, math32.Pi/2.0, 0.0), math32.Vector2{0.0, -2.0})
	assert.InDelta(t, EllipseCurvatureRadius(2.0, 1.0, true, 0.0), 0.5, 1.0e-5)
	assert.InDelta(t, EllipseCurvatureRadius(2.0, 1.0, false, 0.0), -0.5, 1.0e-5)
	assert.InDelta(t, EllipseCurvatureRadius(2.0, 1.0, true, math32.Pi/2.0), 4.0, 1.0e-5)
	assert.True(t, math32.IsNaN(EllipseCurvatureRadius(2.0, 0.0, true, 0.0)))

	// https://www.wolframalpha.com/input/?i=arclength+x%28t%29%3D2*cos+t%2C+y%28t%29%3Dsin+t+for+t%3D0+to+0.5pi
	assert.InDelta(t, ellipseLength(2.0, 1.0, 0.0, math32.Pi/2.0), 2.4221102220, 1.0e-5)
}

func TestXMonotoneEllipse(t *testing.T) {
	assert.InDeltaSlice(t, xmonotoneEllipticArc(math32.Vector2{0.0, 0.0}, 100.0, 50.0, 0.0, false, false, math32.Vector2{0.0, 100.0}), ppath.MustParseSVGPath("M0 0A100 50 0 0 0 -100 50A100 50 0 0 0 0 100"), 1.0e-4)

	// defer setEpsilon(1e-3)()
	assert.InDeltaSlice(t, xmonotoneEllipticArc(math32.Vector2{0.0, 0.0}, 50.0, 25.0, math32.Pi/4.0, false, false, math32.Vector2{100.0 / math32.Sqrt(2.0), 100.0 / math32.Sqrt(2.0)}), ppath.MustParseSVGPath("M0 0A50 25 45 0 0 -4.1731 11.6383A50 25 45 0 0 70.71067811865474 70.71067811865474"), 1.0e-4)
}

func TestFlattenEllipse(t *testing.T) {
	// defer setEpsilon(1e-3)()
	tolerance := float32(1.0)

	// circular
	assert.InDeltaSlice(t, FlattenEllipticArc(math32.Vector2{0.0, 0.0}, 100.0, 100.0, 0.0, false, false, math32.Vector2{200.0, 0.0}, tolerance), ppath.MustParseSVGPath("M0 0L3.855789619238635 30.62763190857508L20.85117757566036 62.58773789575414L48.032233398236286 86.49342322102808L81.90102498412543 99.26826345535623L118.09897501587452 99.26826345535625L151.96776660176369 86.4934232210281L179.1488224243396 62.58773789575416L196.14421038076136 30.62763190857507L200 0"), 1.0e-4)
}

func TestEllipseSplit(t *testing.T) {
	mid, large0, large1, ok := ellipseSplit(2.0, 1.0, 0.0, 0.0, 0.0, math32.Pi, 0.0, math32.Pi/2.0)
	assert.True(t, ok)
	tolEqualVec2(t, math32.Vec2(0, 1), mid, 1.0e-7)
	assert.True(t, !large0)
	assert.True(t, !large1)

	_, _, _, ok = ellipseSplit(2.0, 1.0, 0.0, 0.0, 0.0, math32.Pi, 0.0, -math32.Pi/2.0)
	assert.True(t, !ok)

	mid, large0, large1, ok = ellipseSplit(2.0, 1.0, 0.0, 0.0, 0.0, 0.0, math32.Pi*7.0/4.0, math32.Pi/2.0)
	assert.True(t, ok)
	tolEqualVec2(t, math32.Vec2(0, 1), mid, 1.0e-7)
	assert.True(t, !large0)
	assert.True(t, large1)

	mid, large0, large1, ok = ellipseSplit(2.0, 1.0, 0.0, 0.0, 0.0, 0.0, math32.Pi*7.0/4.0, math32.Pi*3.0/2.0)
	assert.True(t, ok)
	tolEqualVec2(t, math32.Vec2(0, -1), mid, 1.0e-7)
	assert.True(t, large0)
	assert.True(t, !large1)
}

func TestQuadraticBezier(t *testing.T) {
	p0, p1, p2, q0, q1, q2 := quadraticBezierSplit(math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{1.0, 1.0}, 0.5)
	tolEqualVec2(t, p0, math32.Vector2{0.0, 0.0})
	tolEqualVec2(t, p1, math32.Vector2{0.5, 0.0})
	tolEqualVec2(t, p2, math32.Vector2{0.75, 0.25})
	tolEqualVec2(t, q0, math32.Vector2{0.75, 0.25})
	tolEqualVec2(t, q1, math32.Vector2{1.0, 0.5})
	tolEqualVec2(t, q2, math32.Vector2{1.0, 1.0})
}

func TestQuadraticBezierPos(t *testing.T) {
	var tests = []struct {
		p0, p1, p2 math32.Vector2
		t          float32
		q          math32.Vector2
	}{
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{1.0, 1.0}, 0.0, math32.Vector2{0.0, 0.0}},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{1.0, 1.0}, 0.5, math32.Vector2{0.75, 0.25}},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{1.0, 1.0}, 1.0, math32.Vector2{1.0, 1.0}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v--%v", tt.p0, tt.p1, tt.p2, tt.t), func(t *testing.T) {
			q := quadraticBezierPos(tt.p0, tt.p1, tt.p2, tt.t)
			tolEqualVec2(t, q, tt.q, 1.0e-5)
		})
	}
}

func TestQuadraticBezierLength(t *testing.T) {
	var tests = []struct {
		p0, p1, p2 math32.Vector2
		l          float32
	}{
		{math32.Vector2{0.0, 0.0}, math32.Vector2{0.5, 0.0}, math32.Vector2{2.0, 0.0}, 2.0},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{2.0, 0.0}, 2.0},

		// https://www.wolframalpha.com/input/?i=length+of+the+curve+%7Bx%3D2*%281-t%29*t*1.00+%2B+t%5E2*1.00%2C+y%3Dt%5E2*1.00%7D+from+0+to+1
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{1.0, 1.0}, 1.623225},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v", tt.p0, tt.p1, tt.p2), func(t *testing.T) {
			l := quadraticBezierLength(tt.p0, tt.p1, tt.p2)
			assert.InDelta(t, l, tt.l, 1e-6)
		})
	}
}

func TestCubicBezierNormal(t *testing.T) {
	p0, p1, p2, p3 := math32.Vector2{0.0, 0.0}, math32.Vector2{2.0 / 3.0, 0.0}, math32.Vector2{1.0, 1.0 / 3.0}, math32.Vector2{1.0, 1.0}
	var tests = []struct {
		p0, p1, p2, p3 math32.Vector2
		t              float32
		q              math32.Vector2
	}{
		{p0, p1, p2, p3, 0.0, math32.Vector2{0.0, -1.0}},
		{p0, p0, p1, p3, 0.0, math32.Vector2{0.0, -1.0}},
		{p0, p0, p0, p1, 0.0, math32.Vector2{0.0, -1.0}},
		{p0, p0, p0, p0, 0.0, math32.Vector2{0.0, 0.0}},
		{p0, p1, p2, p3, 1.0, math32.Vector2{1.0, 0.0}},
		{p0, p2, p3, p3, 1.0, math32.Vector2{1.0, 0.0}},
		{p2, p3, p3, p3, 1.0, math32.Vector2{1.0, 0.0}},
		{p3, p3, p3, p3, 1.0, math32.Vector2{0.0, 0.0}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v%v--%v", tt.p0, tt.p1, tt.p2, tt.p3, tt.t), func(t *testing.T) {
			q := CubicBezierNormal(tt.p0, tt.p1, tt.p2, tt.p3, tt.t, 1.0)
			tolEqualVec2(t, q, tt.q, 1.0e-5)
		})
	}
}

func TestQuadraticBezierDistance(t *testing.T) {
	var tests = []struct {
		p0, p1, p2 math32.Vector2
		q          math32.Vector2
		d          float32
	}{
		{math32.Vector2{0.0, 0.0}, math32.Vector2{4.0, 6.0}, math32.Vector2{8.0, 0.0}, math32.Vector2{9.0, 0.5}, math32.Sqrt(1.25)},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 1.0}, math32.Vector2{2.0, 0.0}, math32.Vector2{0.0, 0.0}, 0.0},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 1.0}, math32.Vector2{2.0, 0.0}, math32.Vector2{1.0, 1.0}, 0.5},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 1.0}, math32.Vector2{2.0, 0.0}, math32.Vector2{2.0, 0.0}, 0.0},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 1.0}, math32.Vector2{2.0, 0.0}, math32.Vector2{1.0, 0.0}, 0.5},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 1.0}, math32.Vector2{2.0, 0.0}, math32.Vector2{-1.0, 0.0}, 1.0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v--%v", tt.p0, tt.p1, tt.p2, tt.q), func(t *testing.T) {
			d := quadraticBezierDistance(tt.p0, tt.p1, tt.p2, tt.q)
			assert.Equal(t, d, tt.d)
		})
	}
}

func TestXMonotoneQuadraticBezier(t *testing.T) {
	assert.InDeltaSlice(t, xmonotoneQuadraticBezier(math32.Vector2{2.0, 0.0}, math32.Vector2{0.0, 1.0}, math32.Vector2{2.0, 2.0}), ppath.MustParseSVGPath("M2 0Q1 0.5 1 1Q1 1.5 2 2"), 1.0e-5)
}

func TestQuadraticBezierFlatten(t *testing.T) {
	tolerance := float32(0.1)
	tests := []struct {
		path     string
		expected string
	}{
		{"Q1 0 1 1", "L0.8649110641 0.4L1 1"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			path := ppath.MustParseSVGPath(tt.path)
			p0 := math32.Vector2{path[1], path[2]}
			p1 := math32.Vector2{path[5], path[6]}
			p2 := math32.Vector2{path[7], path[8]}

			p := FlattenQuadraticBezier(p0, p1, p2, tolerance)
			assert.InDeltaSlice(t, p, ppath.MustParseSVGPath(tt.expected), 1.0e-5)
		})
	}
}

func TestCubicBezierPos(t *testing.T) {
	p0, p1, p2, p3 := math32.Vector2{0.0, 0.0}, math32.Vector2{2.0 / 3.0, 0.0}, math32.Vector2{1.0, 1.0 / 3.0}, math32.Vector2{1.0, 1.0}
	var tests = []struct {
		p0, p1, p2, p3 math32.Vector2
		t              float32
		q              math32.Vector2
	}{
		{p0, p1, p2, p3, 0.0, math32.Vector2{0.0, 0.0}},
		{p0, p1, p2, p3, 0.5, math32.Vector2{0.75, 0.25}},
		{p0, p1, p2, p3, 1.0, math32.Vector2{1.0, 1.0}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v%v--%v", tt.p0, tt.p1, tt.p2, tt.p3, tt.t), func(t *testing.T) {
			q := cubicBezierPos(tt.p0, tt.p1, tt.p2, tt.p3, tt.t)
			tolEqualVec2(t, q, tt.q, 1.0e-5)
		})
	}
}

func TestCubicBezierDeriv2(t *testing.T) {
	p0, p1, p2, p3 := math32.Vector2{0.0, 0.0}, math32.Vector2{2.0 / 3.0, 0.0}, math32.Vector2{1.0, 1.0 / 3.0}, math32.Vector2{1.0, 1.0}
	var tests = []struct {
		p0, p1, p2, p3 math32.Vector2
		t              float32
		q              math32.Vector2
	}{
		{p0, p1, p2, p3, 0.0, math32.Vector2{-2.0, 2.0}},
		{p0, p1, p2, p3, 0.5, math32.Vector2{-2.0, 2.0}},
		{p0, p1, p2, p3, 1.0, math32.Vector2{-2.0, 2.0}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v%v--%v", tt.p0, tt.p1, tt.p2, tt.p3, tt.t), func(t *testing.T) {
			q := cubicBezierDeriv2(tt.p0, tt.p1, tt.p2, tt.p3, tt.t)
			tolEqualVec2(t, q, tt.q, 1.0e-5)
		})
	}
}

func TestCubicBezierCurvatureRadius(t *testing.T) {
	p0, p1, p2, p3 := math32.Vector2{0.0, 0.0}, math32.Vector2{2.0 / 3.0, 0.0}, math32.Vector2{1.0, 1.0 / 3.0}, math32.Vector2{1.0, 1.0}
	var tests = []struct {
		p0, p1, p2, p3 math32.Vector2
		t              float32
		r              float32
	}{
		{p0, p1, p2, p3, 0.0, 2.0},
		{p0, p1, p2, p3, 0.5, 1.0 / math32.Sqrt(2)},
		{p0, p1, p2, p3, 1.0, 2.0},
		{p0, math32.Vector2{1.0, 0.0}, math32.Vector2{2.0, 0.0}, math32.Vector2{3.0, 0.0}, 0.0, math32.NaN()},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v%v--%v", tt.p0, tt.p1, tt.p2, tt.p3, tt.t), func(t *testing.T) {
			r := CubicBezierCurvatureRadius(tt.p0, tt.p1, tt.p2, tt.p3, tt.t)
			if math32.IsNaN(tt.r) {
				assert.True(t, math32.IsNaN(r))
			} else {
				assert.Equal(t, r, tt.r)
			}
		})
	}
}

func TestCubicBezierLength(t *testing.T) {
	p0, p1, p2, p3 := math32.Vector2{0.0, 0.0}, math32.Vector2{2.0 / 3.0, 0.0}, math32.Vector2{1.0, 1.0 / 3.0}, math32.Vector2{1.0, 1.0}
	var tests = []struct {
		p0, p1, p2, p3 math32.Vector2
		l              float32
	}{
		// https://www.wolframalpha.com/input/?i=length+of+the+curve+%7Bx%3D3*%281-t%29%5E2*t*0.666667+%2B+3*%281-t%29*t%5E2*1.00+%2B+t%5E3*1.00%2C+y%3D3*%281-t%29*t%5E2*0.333333+%2B+t%5E3*1.00%7D+from+0+to+1
		{p0, p1, p2, p3, 1.623225},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v%v%v%v", tt.p0, tt.p1, tt.p2, tt.p3), func(t *testing.T) {
			l := cubicBezierLength(tt.p0, tt.p1, tt.p2, tt.p3)
			assert.InDelta(t, l, tt.l, 1e-6)
		})
	}
}

func TestCubicBezierSplit(t *testing.T) {
	p0, p1, p2, p3, q0, q1, q2, q3 := cubicBezierSplit(math32.Vector2{0.0, 0.0}, math32.Vector2{2.0 / 3.0, 0.0}, math32.Vector2{1.0, 1.0 / 3.0}, math32.Vector2{1.0, 1.0}, 0.5)
	tolEqualVec2(t, p0, math32.Vector2{0.0, 0.0})
	tolEqualVec2(t, p1, math32.Vector2{1.0 / 3.0, 0.0})
	tolEqualVec2(t, p2, math32.Vector2{7.0 / 12.0, 1.0 / 12.0})
	tolEqualVec2(t, p3, math32.Vector2{0.75, 0.25})
	tolEqualVec2(t, q0, math32.Vector2{0.75, 0.25})
	tolEqualVec2(t, q1, math32.Vector2{11.0 / 12.0, 5.0 / 12.0})
	tolEqualVec2(t, q2, math32.Vector2{1.0, 2.0 / 3.0})
	tolEqualVec2(t, q3, math32.Vector2{1.0, 1.0})
}

func TestCubicBezierStrokeHelpers(t *testing.T) {
	p0, p1, p2, p3 := math32.Vector2{0.0, 0.0}, math32.Vector2{2.0 / 3.0, 0.0}, math32.Vector2{1.0, 1.0 / 3.0}, math32.Vector2{1.0, 1.0}

	p := ppath.Path{}
	addCubicBezierLine(&p, p0, p1, p0, p0, 0.0, 0.5)
	assert.True(t, p.Empty())

	p = ppath.Path{}
	addCubicBezierLine(&p, p0, p1, p2, p3, 0.0, 0.5)
	assert.InDeltaSlice(t, p, ppath.MustParseSVGPath("L0 -0.5"), 1.0e-5)

	p = ppath.Path{}
	addCubicBezierLine(&p, p0, p1, p2, p3, 1.0, 0.5)
	assert.InDeltaSlice(t, p, ppath.MustParseSVGPath("L1.5 1"), 1.0e-5)
}

func TestXMonotoneCubicBezier(t *testing.T) {
	assert.InDeltaSlice(t, xmonotoneCubicBezier(math32.Vector2{1.0, 0.0}, math32.Vector2{0.0, 0.0}, math32.Vector2{0.0, 1.0}, math32.Vector2{1.0, 1.0}), ppath.MustParseSVGPath("M1 0C0.5 0 0.25 0.25 0.25 0.5C0.25 0.75 0.5 1 1 1"), 1.0e-5)
	assert.InDeltaSlice(t, xmonotoneCubicBezier(math32.Vector2{0.0, 0.0}, math32.Vector2{3.0, 0.0}, math32.Vector2{-2.0, 1.0}, math32.Vector2{1.0, 1.0}), ppath.MustParseSVGPath("M0 0C0.75 0 1 0.0625 1 0.15625C1 0.34375 0.0 0.65625 0.0 0.84375C0.0 0.9375 0.25 1 1 1"), 1.0e-5)
}

func TestCubicBezierStrokeFlatten(t *testing.T) {
	tests := []struct {
		path      string
		d         float32
		tolerance float32
		expected  string
	}{
		{"C0.666667 0 1 0.333333 1 1", 0.5, 0.5, "L1.5 1"},
		{"C0.666667 0 1 0.333333 1 1", 0.5, 0.125, "L1.376154 0.308659L1.5 1"},
		{"C1 0 2 1 3 2", 0.0, 0.1, "L1.095445 0.351314L2.579154 1.581915L3 2"},
		{"C0 0 1 0 2 2", 0.0, 0.1, "L1.22865 0.8L2 2"},       // p0 == p1
		{"C1 1 2 2 3 5", 0.0, 0.1, "L2.481111 3.612482L3 5"}, // s2 == 0
	}
	origEpsilon := ppath.Epsilon
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			ppath.Epsilon = origEpsilon
			path := ppath.MustParseSVGPath(tt.path)
			p0 := math32.Vector2{path[1], path[2]}
			p1 := math32.Vector2{path[5], path[6]}
			p2 := math32.Vector2{path[7], path[8]}
			p3 := math32.Vector2{path[9], path[10]}

			p := ppath.Path{}
			FlattenSmoothCubicBezier(&p, p0, p1, p2, p3, tt.d, tt.tolerance)
			ppath.Epsilon = 1e-6
			assert.InDeltaSlice(t, p, ppath.MustParseSVGPath(tt.expected), 1.0e-5)
		})
	}
	ppath.Epsilon = origEpsilon
}

func TestCubicBezierInflectionPoints(t *testing.T) {
	tests := []struct {
		p0, p1, p2, p3 math32.Vector2
		x1, x2         float32
	}{
		{math32.Vector2{0.0, 0.0}, math32.Vector2{0.0, 1.0}, math32.Vector2{1.0, 1.0}, math32.Vector2{1.0, 0.0}, math32.NaN(), math32.NaN()},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 1.0}, math32.Vector2{0.0, 1.0}, math32.Vector2{1.0, 0.0}, 0.5, math32.NaN()},

		// see "Analysis of Inflection math32.Vector2s for Planar Cubic Bezier Curve" by Z.Zhang et al. from 2009
		// https://cie.nwsuaf.edu.cn/docs/20170614173651207557.pdf
		{math32.Vector2{16, 467}, math32.Vector2{185, 95}, math32.Vector2{673, 545}, math32.Vector2{810, 17}, 0.4565900353, math32.NaN()},
		{math32.Vector2{859, 676}, math32.Vector2{13, 422}, math32.Vector2{781, 12}, math32.Vector2{266, 425}, 0.6810755245, 0.7052992723},
		{math32.Vector2{872, 686}, math32.Vector2{11, 423}, math32.Vector2{779, 13}, math32.Vector2{220, 376}, 0.5880709424, 0.8868629954},
		{math32.Vector2{819, 566}, math32.Vector2{43, 18}, math32.Vector2{826, 18}, math32.Vector2{25, 533}, 0.4761686269, 0.5392953369},
		{math32.Vector2{884, 574}, math32.Vector2{135, 14}, math32.Vector2{678, 14}, math32.Vector2{14, 566}, 0.3208363269, 0.6822908688},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v %v %v %v", tt.p0, tt.p1, tt.p2, tt.p3), func(t *testing.T) {
			x1, x2 := findInflectionPointCubicBezier(tt.p0, tt.p1, tt.p2, tt.p3)
			assert.InDeltaSlice(t, []float32{x1, x2}, []float32{tt.x1, tt.x2}, 1.0e-5)
		})
	}
}

func TestCubicBezierInflectionPointRange(t *testing.T) {
	tests := []struct {
		p0, p1, p2, p3 math32.Vector2
		t, tolerance   float32
		x1, x2         float32
	}{
		{math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 1.0}, math32.Vector2{0.0, 1.0}, math32.Vector2{1.0, 0.0}, math32.NaN(), 0.25, math32.Inf(1.0), math32.Inf(1.0)},

		// p0==p1==p2
		{math32.Vector2{0.0, 0.0}, math32.Vector2{0.0, 0.0}, math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, 0.0, 0.25, 0.0, 1.0},

		// p0==p1, s3==0
		{math32.Vector2{0.0, 0.0}, math32.Vector2{0.0, 0.0}, math32.Vector2{1.0, 0.0}, math32.Vector2{1.0, 0.0}, 0.0, 0.25, 0.0, 1.0},

		// all within tolerance
		{math32.Vector2{0.0, 0.0}, math32.Vector2{0.0, 1.0}, math32.Vector2{1.0, 1.0}, math32.Vector2{1.0, 0.0}, 0.5, 1.0, -0.0503212081, 1.0503212081},
		{math32.Vector2{0.0, 0.0}, math32.Vector2{0.0, 1.0}, math32.Vector2{1.0, 1.0}, math32.Vector2{1.0, 0.0}, 0.5, 1e-9, 0.4994496788, 0.5005503212},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v %v %v %v", tt.p0, tt.p1, tt.p2, tt.p3), func(t *testing.T) {
			x1, x2 := findInflectionPointRangeCubicBezier(tt.p0, tt.p1, tt.p2, tt.p3, tt.t, tt.tolerance)
			assert.InDeltaSlice(t, []float32{x1, x2}, []float32{tt.x1, tt.x2}, 1.0e-5)
		})
	}
}

func TestCubicBezierFlatten(t *testing.T) {
	tests := []struct {
		p []math32.Vector2
	}{
		// see "Analysis of Inflection math32.Vector2s for Planar Cubic Bezier Curve" by Z.Zhang et al. from 2009
		// https://cie.nwsuaf.edu.cn/docs/20170614173651207557.pdf
		{[]math32.Vector2{{16, 467}, {185, 95}, {673, 545}, {810, 17}}},
		{[]math32.Vector2{{859, 676}, {13, 422}, {781, 12}, {266, 425}}},
		{[]math32.Vector2{{872, 686}, {11, 423}, {779, 13}, {220, 376}}},
		{[]math32.Vector2{{819, 566}, {43, 18}, {826, 18}, {25, 533}}},
		{[]math32.Vector2{{884, 574}, {135, 14}, {678, 14}, {14, 566}}},

		// be aware that we offset the bezier by 0.1
		// single inflection point, ranges outside t=[0,1]
		{[]math32.Vector2{{0, 0}, {1, 1}, {0, 1}, {1, 0}}},

		// two inflection points, ranges outside t=[0,1]
		{[]math32.Vector2{{0, 0}, {0.9, 1}, {0.1, 1}, {1, 0}}},

		// one inflection point, max range outside t=[0,1]
		{[]math32.Vector2{{0, 0}, {80, 100}, {80, -100}, {100, 0}}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v %v %v %v", tt.p[0], tt.p[1], tt.p[2], tt.p[3]), func(t *testing.T) {
			length := cubicBezierLength(tt.p[0], tt.p[1], tt.p[2], tt.p[3])
			flatLength := Length(FlattenCubicBezier(tt.p[0], tt.p[1], tt.p[2], tt.p[3], 0.0, 0.001))
			assert.InDelta(t, flatLength, length, 0.25)
		})
	}

	tolEqualBox2(t, math32.B2(0.0, -5.0, 32.4787516156, 15.0), Bounds(FlattenCubicBezier(math32.Vector2{0, 0}, math32.Vector2{30, 0}, math32.Vector2{30, 10}, math32.Vector2{25, 10}, 5.0, 0.01)), 1.0e-5)
}
