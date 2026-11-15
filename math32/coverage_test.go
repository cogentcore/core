// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"image"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/image/math/fixed"
)

// TestMathFuncs tests that the float32 math functions match those of the
// math package, for some arguments in their domains.
func TestMathFuncs(t *testing.T) {
	unary := map[string]struct {
		f32 func(float32) float32
		f64 func(float64) float64
		x   float32
	}{
		"Acosh":       {Acosh, math.Acosh, 2},
		"Asinh":       {Asinh, math.Asinh, 0.7},
		"Atan":        {Atan, math.Atan, 0.7},
		"Atanh":       {Atanh, math.Atanh, 0.3},
		"Cbrt":        {Cbrt, math.Cbrt, 27},
		"Cosh":        {Cosh, math.Cosh, 0.5},
		"Erf":         {Erf, math.Erf, 0.5},
		"Erfc":        {Erfc, math.Erfc, 0.5},
		"Erfcinv":     {Erfcinv, math.Erfcinv, 0.5},
		"Erfinv":      {Erfinv, math.Erfinv, 0.5},
		"Exp":         {Exp, math.Exp, 1.5},
		"Exp2":        {Exp2, math.Exp2, 3},
		"Expm1":       {Expm1, math.Expm1, 0.01},
		"Gamma":       {Gamma, math.Gamma, 4.5},
		"J0":          {J0, math.J0, 1},
		"J1":          {J1, math.J1, 1},
		"Log":         {Log, math.Log, 10},
		"Log10":       {Log10, math.Log10, 1000},
		"Log1p":       {Log1p, math.Log1p, 0.01},
		"Log2":        {Log2, math.Log2, 8},
		"Logb":        {Logb, math.Logb, 8},
		"RoundToEven": {RoundToEven, math.RoundToEven, 2.5},
		"Sinh":        {Sinh, math.Sinh, 0.5},
		"Tanh":        {Tanh, math.Tanh, 0.5},
		"Trunc":       {Trunc, math.Trunc, -2.7},
		"Y0":          {Y0, math.Y0, 1},
		"Y1":          {Y1, math.Y1, 1},
	}
	for name, f := range unary {
		assert.InDelta(t, f.f64(float64(f.x)), float64(f.f32(f.x)), 1e-4, name)
	}
	binary := map[string]struct {
		f32  func(float32, float32) float32
		f64  func(float64, float64) float64
		x, y float32
	}{
		"Copysign":  {Copysign, math.Copysign, 3, -1},
		"Dim":       {Dim, math.Dim, 5, 3},
		"Hypot":     {Hypot, math.Hypot, 3, 4},
		"Nextafter": {Nextafter, func(x, y float64) float64 { return float64(math.Nextafter32(float32(x), float32(y))) }, 1, 2},
		"Pow":       {Pow, math.Pow, 2, 10},
		"Remainder": {Remainder, math.Remainder, 7, 3},
	}
	for name, f := range binary {
		assert.InDelta(t, f.f64(float64(f.x), float64(f.y)), float64(f.f32(f.x, f.y)), 1e-4, name)
	}

	assert.InDelta(t, 180, RadToDeg(Pi), 1e-4)
	assert.Equal(t, float32(-1), Sign(-2))
	assert.Equal(t, float32(1), Sign(0))
	assert.Equal(t, float32(7), FMA(2, 3, 1))
	frac, exp := Frexp(8)
	assert.Equal(t, float32(0.5), frac)
	assert.Equal(t, 4, exp)
	assert.Equal(t, float32(8), Ldexp(0.5, 4))
	assert.Equal(t, float32(3), Ilogb(8))
	assert.True(t, IsInf(Inf(1), 1))
	assert.True(t, IsInf(Inf(-1), -1))
	assert.False(t, IsInf(1, 0))
	it, fr := Modf(3.25)
	assert.Equal(t, float32(3), it)
	assert.Equal(t, float32(0.25), fr)
	assert.Equal(t, float32(1000), Pow10(3))
	assert.True(t, Signbit(-1))
	assert.InDelta(t, math.Jn(2, 1), float64(Jn(2, 1)), 1e-5)
	assert.InDelta(t, math.Yn(2, 1), float64(Yn(2, 1)), 1e-4)
	lg, sign := Lgamma(4.5)
	lg64, sign64 := math.Lgamma(4.5)
	assert.InDelta(t, lg64, float64(lg), 1e-4)
	assert.Equal(t, sign64, sign)
	assert.Equal(t, float32(5), Lerp(0, 10, 0.5))

	assert.Equal(t, float32(2), MinPos(2, 3))
	assert.Equal(t, float32(3), MinPos(-1, 3))
	assert.Equal(t, float32(2), MinPos(2, 0))
	assert.Equal(t, float32(-1), MinPos(-1, -2), "the first if neither is positive")
	assert.Equal(t, float32(3), MaxPos(2, 3))
	assert.Equal(t, float32(3), MaxPos(-1, 3))
	assert.Equal(t, float32(2), MaxPos(2, -5))
}

// TestBox2 tests the methods of [Box2].
func TestBox2(t *testing.T) {
	b := B2FromRect(image.Rect(1, 2, 5, 8))
	assert.Equal(t, B2(1, 2, 5, 8), b)
	assert.Equal(t, image.Rect(1, 2, 5, 8), b.ToRect())
	fb := B2FromFixed(b.ToFixed())
	assert.Equal(t, b, fb)
	assert.Equal(t, fixed.I(5), b.ToFixed().Max.X)
	assert.False(t, b.IsEmpty())
	e := B2Empty()
	assert.True(t, e.IsEmpty())

	assert.Equal(t, Vec2(3, 5), b.Center())
	assert.Equal(t, Vec2(4, 6), b.Size())
	assert.True(t, b.ContainsPoint(Vec2(2, 3)))
	assert.False(t, b.ContainsPoint(Vec2(0, 3)))
	assert.True(t, b.ContainsBox(B2(2, 3, 4, 4)))
	assert.False(t, b.ContainsBox(B2(2, 3, 6, 4)))
	assert.True(t, b.IntersectsBox(B2(4, 7, 9, 9)))
	assert.False(t, b.IntersectsBox(B2(6, 0, 9, 9)))
	assert.Equal(t, Vec2(5, 2), b.ClampPoint(Vec2(9, -1)))
	assert.Equal(t, float32(5), b.DistanceToPoint(Vec2(8, 12)))
	assert.Equal(t, float32(0), b.DistanceToPoint(Vec2(2, 3)))
	assert.Equal(t, B2(4, 7, 5, 8), b.Intersect(B2(4, 7, 9, 9)))
	assert.Equal(t, B2(1, 2, 9, 9), b.Union(B2(4, 7, 9, 9)))
	assert.Equal(t, B2(2, 4, 6, 10), b.Translate(Vec2(1, 2)))
	assert.Equal(t, float32(3), b.ProjectX(0.5))
	assert.Equal(t, float32(8), b.ProjectY(1))
	assert.Equal(t, B2(1, 2, 5, 8), B2(5, 8, 1, 2).Canon())

	var c Box2
	c.Set(nil, nil)
	assert.True(t, c.IsEmpty(), "nil is infinite: empty")
	c.Set(&Vector2{1, 1}, &Vector2{2, 3})
	assert.Equal(t, B2(1, 1, 2, 3), c)
	c.SetFromPoints([]Vector2{{3, 1}, {-1, 4}, {0, 0}})
	assert.Equal(t, B2(-1, 0, 3, 4), c)
	c.SetFromRect(image.Rect(0, 0, 2, 2))
	c.ExpandByVector(Vec2(1, 2))
	assert.Equal(t, B2(-1, -2, 3, 4), c)
	c.ExpandByScalar(1)
	assert.Equal(t, B2(-2, -3, 4, 5), c)
	c.ExpandByBox(B2(0, 0, 9, 1))
	assert.Equal(t, B2(-2, -3, 9, 5), c)
	c.SetFromCenterAndSize(Vec2(1, 1), Vec2(4, 2))
	assert.Equal(t, B2(-1, 0, 3, 2), c)

	assert.True(t, RectInNotEmpty(image.Rect(1, 1, 2, 2), image.Rect(0, 0, 3, 3)))
	assert.False(t, RectInNotEmpty(image.Rectangle{}, image.Rect(0, 0, 3, 3)), "an empty rect is not in")
}

// TestBox3 tests the methods of [Box3].
func TestBox3(t *testing.T) {
	b := B3(0, 0, 0, 2, 4, 6)
	assert.False(t, b.IsEmpty())
	assert.True(t, B3Empty().IsEmpty())
	assert.Equal(t, Vec3(1, 2, 3), b.Center())
	assert.Equal(t, Vec3(2, 4, 6), b.Size())
	assert.True(t, b.ContainsPoint(Vec3(1, 1, 1)))
	assert.False(t, b.ContainsPoint(Vec3(1, 5, 1)))
	assert.True(t, b.ContainsBox(B3(1, 1, 1, 2, 2, 2)))
	assert.False(t, b.ContainsBox(B3(1, 1, 1, 3, 2, 2)))
	assert.True(t, b.IntersectsBox(B3(1, 1, 1, 3, 9, 9)))
	assert.False(t, b.IntersectsBox(B3(3, 0, 0, 4, 1, 1)))
	assert.Equal(t, Vec3(2, 0, 6), b.ClampPoint(Vec3(5, -1, 7)))
	assert.InDelta(t, 3, b.DistanceToPoint(Vec3(5, 4, 6)), 1e-6)
	sp := b.GetBoundingSphere()
	assert.Equal(t, Vec3(1, 2, 3), sp.Center)
	assert.InDelta(t, Sqrt(14), sp.Radius, 1e-5)
	assert.Equal(t, B3(1, 1, 1, 2, 4, 6), b.Intersect(B3(1, 1, 1, 9, 9, 9)))
	assert.Equal(t, B3(-1, 0, 0, 2, 4, 9), b.Union(B3(-1, 1, 1, 0, 2, 9)))
	assert.Equal(t, B3(1, 1, 1, 3, 5, 7), b.Translate(Vec3(1, 1, 1)))

	var c Box3
	c.Set(&Vector3{1, 1, 1}, &Vector3{2, 2, 2})
	assert.Equal(t, B3(1, 1, 1, 2, 2, 2), c)
	c.SetMinMax(Vec3(0, 0, 0), Vec3(1, 1, 1))
	c.ExpandByPoints([]Vector3{{2, -1, 0}})
	assert.Equal(t, B3(0, -1, 0, 2, 1, 1), c)
	c.SetFromPoints([]Vector3{{1, 2, 3}, {-1, 0, 5}})
	assert.Equal(t, B3(-1, 0, 3, 1, 2, 5), c)
	c.ExpandByBox(B3(0, 0, 0, 1, 1, 1))
	assert.Equal(t, B3(-1, 0, 0, 1, 2, 5), c)
	c.ExpandByVector(Vec3(1, 1, 1))
	c.ExpandByScalar(1)
	assert.Equal(t, B3(-3, -2, -2, 3, 4, 7), c)
	c.SetFromCenterAndSize(Vec3(0, 0, 0), Vec3(2, 4, 6))
	assert.Equal(t, B3(-1, -2, -3, 1, 2, 3), c)
}

// TestQuatOps tests the arithmetic and accessors of [Quat].
func TestQuatOps(t *testing.T) {
	id := NewQuatIdentity()
	assert.True(t, id.IsIdentity())
	assert.False(t, id.IsNil())
	assert.True(t, Quat{}.IsNil())
	assert.False(t, id.IsNaN())
	assert.True(t, NewQuat(NaN(), 0, 0, 1).IsNaN())
	assert.Equal(t, "(0, 0, 0, 1)", id.String())

	var q Quat
	q.Set(1, 2, 3, 4)
	arr := make([]float32, 5)
	q.ToArray(arr, 1)
	assert.Equal(t, []float32{0, 1, 2, 3, 4}, arr)
	var r Quat
	r.FromArray(arr, 1)
	assert.Equal(t, q, r)
	r.SetIdentity()
	assert.True(t, r.IsIdentity())
	assert.Contains(t, NewQuatAxisAngle(Vec3(0, 0, 1), 1).GenGoSet(".Rot"), ".Rot.SetFromAxisAngle(math32.Vec3(0, 0, 1), 1)")
	assert.Equal(t, "math32.Quat{1, 2, 3, 4}", q.GenGoNew())

	assert.Equal(t, float32(30), q.LengthSq())
	assert.Equal(t, float32(30), q.Dot(q))
	assert.Equal(t, NewQuat(-1, -2, -3, 4), q.Conjugate())
	c := q
	c.SetConjugate()
	assert.Equal(t, q.Conjugate(), c)
	assert.Equal(t, NewQuat(2, 4, 6, 8), q.MulScalar(2))
	assert.Equal(t, NewQuat(2, 4, 6, 8), q.Add(q))

	// a rotation times its inverse is the identity
	rot := NewQuatAxisAngle(Vec3(0, 0, 1), Pi/3)
	p := rot.Mul(rot.Inverse())
	assert.InDelta(t, 1, p.W, 1e-6)
	assert.InDelta(t, 0, p.Z, 1e-6)
	inv := rot
	inv.SetInverse()
	assert.Equal(t, rot.Inverse(), inv)
	sm := rot
	sm.SetMul(rot)
	two := NewQuatAxisAngle(Vec3(0, 0, 1), 2*Pi/3)
	assert.InDelta(t, two.Z, sm.Z, 1e-6)
	assert.InDelta(t, two.W, sm.W, 1e-6)

	e := NewQuatEuler(Vec3(0, 0, Pi/2))
	v := e.MulVector(Vec3(1, 0, 0))
	assert.InDelta(t, 0, v.X, 1e-6)
	assert.InDelta(t, 1, v.Y, 1e-6)
}
