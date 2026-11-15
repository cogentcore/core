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

// TestSphere tests the methods of [Sphere].
func TestSphere(t *testing.T) {
	s := NewSphere(Vec3(0, 0, 0), 2)
	assert.False(t, s.IsEmpty(nil))
	assert.True(t, s.ContainsPoint(Vec3(1, 1, 1)))
	assert.False(t, s.ContainsPoint(Vec3(2, 2, 0)))
	assert.Equal(t, float32(1), s.DistanceToPoint(Vec3(3, 0, 0)))
	assert.True(t, s.IntersectSphere(Sphere{Center: Vec3(3, 0, 0), Radius: 1}))
	assert.False(t, s.IntersectSphere(Sphere{Center: Vec3(5, 0, 0), Radius: 1}))
	assert.Equal(t, Vec3(2, 0, 0), s.ClampPoint(Vec3(5, 0, 0)))
	assert.Equal(t, Vec3(1, 0, 0), s.ClampPoint(Vec3(1, 0, 0)))
	assert.Equal(t, B3(-2, -2, -2, 2, 2, 2), s.GetBoundingBox())

	m := Identity4()
	m.SetScale(3, 1, 1)
	s.MulMatrix4(m)
	assert.Equal(t, float32(6), s.Radius, "scaled by the largest scale")
	s.Translate(Vec3(1, 0, 0))
	assert.Equal(t, Vec3(1, 0, 0), s.Center)

	s.Set(Vec3(1, 1, 1), 0)
	assert.True(t, s.IsEmpty(nil))
	s.SetFromBox(B3(0, 0, 0, 2, 0, 0))
	assert.Equal(t, Vec3(1, 0, 0), s.Center)
	assert.Equal(t, float32(1), s.Radius)
	s.SetFromPoints([]Vector3{{0, 0, 0}, {4, 0, 0}, {2, 1, 0}}, nil)
	assert.Equal(t, Vec3(2, 0.5, 0), s.Center)
	c := Vec3(0, 0, 0)
	s.SetFromPoints([]Vector3{{0, 3, 0}, {4, 0, 0}}, &c)
	assert.Equal(t, float32(4), s.Radius)
}

// TestPlaneLine tests the methods of [Plane] and [Line3].
func TestPlaneLine(t *testing.T) {
	// the plane z = 1
	p := NewPlane(Vec3(0, 0, 1), -1)
	assert.Equal(t, float32(2), p.DistanceToPoint(Vec3(5, 5, 3)))
	assert.Equal(t, float32(1), p.DistanceToSphere(Sphere{Center: Vec3(0, 0, 3), Radius: 1}))
	assert.Equal(t, Vec3(0, 0, 1), p.CoplanarPoint())

	l := NewLine3(Vec3(0, 0, 0), Vec3(0, 0, 4))
	assert.Equal(t, Vec3(0, 0, 2), l.Center())
	assert.Equal(t, Vec3(0, 0, 4), l.Delta())
	assert.Equal(t, float32(16), l.DistanceSquared())
	assert.Equal(t, float32(4), l.Dist())
	assert.True(t, p.IsIntersectionLine(l))
	pt, ok := p.IntersectLine(l)
	assert.True(t, ok)
	assert.Equal(t, Vec3(0, 0, 1), pt)
	_, ok = p.IntersectLine(NewLine3(Vec3(0, 0, 2), Vec3(0, 0, 4)))
	assert.False(t, ok, "beyond the end of the line")
	_, ok = p.IntersectLine(NewLine3(Vec3(0, 0, 3), Vec3(1, 0, 3)))
	assert.False(t, ok, "parallel")
	pt, ok = p.IntersectLine(NewLine3(Vec3(0, 0, 1), Vec3(1, 0, 1)))
	assert.True(t, ok, "in the plane")
	assert.Equal(t, Vec3(0, 0, 1), pt)
	m := Identity4()
	m.SetTranslation(1, 2, 3)
	assert.Equal(t, NewLine3(Vec3(1, 2, 3), Vec3(1, 2, 7)), l.MulMatrix4(m))
	l.Set(Vec3(1, 1, 1), Vec3(1, 1, 1))
	assert.Equal(t, float32(0), l.Dist())

	p.SetTranslate(Vec3(0, 0, 1))
	assert.Equal(t, Vec3(0, 0, 2), p.CoplanarPoint(), "moved to z = 2")
	p.Negate()
	assert.Equal(t, Vec3(0, 0, -1), p.Norm)
	p.Set(Vec3(0, 2, 0), 4)
	p.Normalize()
	assert.Equal(t, Vec3(0, 1, 0), p.Norm)
	assert.Equal(t, float32(2), p.Off)
	p.SetFromNormalAndCoplanarPoint(Vec3(1, 0, 0), Vec3(3, 9, 9))
	assert.Equal(t, float32(0), p.DistanceToPoint(Vec3(3, 0, 0)))
	p.SetFromCoplanarPoints(Vec3(0, 0, 0), Vec3(1, 0, 0), Vec3(0, 1, 0))
	assert.Equal(t, Vec3(0, 0, 1), p.Norm)
	assert.Equal(t, float32(0), p.Off)
}

// TestTriangle tests the methods of [Triangle].
func TestTriangle(t *testing.T) {
	tr := NewTriangle(Vec3(0, 0, 0), Vec3(2, 0, 0), Vec3(0, 2, 0))
	assert.Equal(t, float32(2), tr.Area())
	assert.InDelta(t, 2.0/3, tr.Midpoint().X, 1e-6)
	assert.Equal(t, Vec3(0, 0, 1), tr.Normal())
	assert.Equal(t, Vec3(0, 0, 1), Normal(tr.A, tr.B, tr.C))
	pl := tr.Plane()
	assert.Equal(t, Vec3(0, 0, 1), pl.Norm)
	assert.True(t, tr.ContainsPoint(Vec3(0.5, 0.5, 0)))
	assert.False(t, tr.ContainsPoint(Vec3(2, 2, 0)))
	assert.True(t, ContainsPoint(Vec3(0.1, 0.1, 0), tr.A, tr.B, tr.C))
	bc := tr.BarycoordFromPoint(Vec3(0, 0, 0))
	assert.Equal(t, Vec3(1, 0, 0), bc)
	assert.Equal(t, Vec3(-2, -1, -1), BarycoordFromPoint(Vec3(1, 1, 0), tr.A, tr.A, tr.A), "degenerate")
	tr.SetFromPointsAndIndices([]Vector3{{9, 9, 9}, {0, 0, 0}, {0, 1, 0}, {1, 0, 0}}, 1, 3, 2)
	assert.Equal(t, float32(0.5), tr.Area())
	tr.Set(Vec3(0, 0, 0), Vec3(1, 0, 0), Vec3(2, 0, 0))
	assert.Equal(t, Vector3{}, tr.Normal(), "collinear")
}

// TestArrays tests [ArrayF32] and [ArrayU32].
func TestArrays(t *testing.T) {
	a := NewArrayF32(0, 8)
	a.Append(1)
	a.AppendVector2(Vec2(2, 3))
	a.AppendVector3(Vec3(4, 5, 6))
	a.AppendVector4(Vec4(7, 8, 9, 10))
	assert.Len(t, a, 10)
	assert.Equal(t, 40, a.NumBytes())
	var v2 Vector2
	a.GetVector2(1, &v2)
	assert.Equal(t, Vec2(2, 3), v2)
	var v3 Vector3
	a.GetVector3(3, &v3)
	assert.Equal(t, Vec3(4, 5, 6), v3)
	var v4 Vector4
	a.GetVector4(6, &v4)
	assert.Equal(t, Vec4(7, 8, 9, 10), v4)
	a.Set(0, 11, 12)
	a.SetVector2(2, Vec2(13, 14))
	a.SetVector3(4, Vec3(15, 16, 17))
	a.SetVector4(6, Vec4(18, 19, 20, 21))
	assert.Equal(t, ArrayF32{11, 12, 13, 14, 15, 16, 18, 19, 20, 21}, a, "the vector4 at 6 overwrites the end of the vector3")
	var b ArrayF32
	b.CopyFrom(a)
	assert.Equal(t, a, b)

	var f32 []float32
	CopyFloat32s(&f32, []float32{1, 2})
	assert.Equal(t, []float32{1, 2}, f32)
	var f64 []float64
	CopyFloat64s(&f64, []float64{3})
	assert.Equal(t, []float64{3}, f64)

	u := NewArrayU32(2, 4)
	u.Append(7)
	u.Set(0, 5, 6)
	assert.Equal(t, ArrayU32{5, 6, 7}, u)
	assert.Equal(t, 12, u.NumBytes())
}
