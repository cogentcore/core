// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/image/math/fixed"
)

// assertVec3 asserts that the vectors are equal within the tolerance.
func assertVec3(t *testing.T, want, got Vector3, msg ...any) {
	t.Helper()
	assert.InDelta(t, want.X, got.X, 1e-4, msg...)
	assert.InDelta(t, want.Y, got.Y, 1e-4, msg...)
	assert.InDelta(t, want.Z, got.Z, 1e-4, msg...)
}

// TestSRGB tests converting colors to and from linear gamma.
func TestSRGB(t *testing.T) {
	for _, v := range []float32{0, 0.002, 0.1, 0.5, 1} {
		assert.InDelta(t, v, SRGBToLinear(SRGBFromLinear(v)), 1e-5, "round trip %v", v)
	}
	assert.InDelta(t, 0.214, SRGBToLinear(0.5), 1e-3)
	assert.InDelta(t, 0.002*12.92, SRGBFromLinear(0.002), 1e-6, "the linear part")

	v3 := NewVector3Color(color.RGBA{255, 0, 128, 255})
	assertVec3(t, Vec3(1, 0, 128.0/255), v3)
	assertVec3(t, v3, v3.SRGBFromLinear().SRGBToLinear())
	v4 := NewVector4Color(color.RGBA{0, 0, 0, 128})
	assert.InDelta(t, 128.0/255, v4.W, 1e-4)
	v4 = NewVector4Color(color.RGBA{255, 128, 0, 255})
	l := v4.SRGBToLinear()
	assert.Equal(t, v4.W, l.W, "alpha is not converted")
	assert.InDelta(t, v4.Y, l.SRGBFromLinear().Y, 1e-5)
}

// TestGeom2DInt tests integer geometry and fitting regions in windows.
func TestGeom2DInt(t *testing.T) {
	g := Geom2DInt{Pos: image.Pt(10, 20), Size: image.Pt(30, 40)}
	assert.Equal(t, image.Rect(10, 20, 40, 60), g.Bounds())
	assert.Equal(t, B2(10, 20, 40, 60), g.Box2())
	assert.Equal(t, image.Rect(0, 0, 30, 40), g.SizeRect())
	var h Geom2DInt
	h.SetRect(image.Rect(1, 2, 5, 7))
	assert.Equal(t, Geom2DInt{Pos: image.Pt(1, 2), Size: image.Pt(4, 5)}, h)

	win := Geom2DInt{Size: image.Pt(100, 50)}
	st := Geom2DInt{Pos: image.Pt(90, -5), Size: image.Pt(30, 60)}
	fit := st.FitInWindow(win)
	assert.Equal(t, Geom2DInt{Pos: image.Pt(70, 0), Size: image.Pt(30, 50)}, fit, "moved, then shrunk")
}

// TestMatrix2More tests more 2D transforms and parsing.
func TestMatrix2More(t *testing.T) {
	m := Rotate2DAround(Pi/2, Vec2(1, 1))
	p := m.MulPoint(Vec2(2, 1))
	assert.InDelta(t, 1, p.X, 1e-5, "rotated about (1,1)")
	assert.InDelta(t, 2, p.Y, 1e-5)

	sk := Skew2D(Pi/4, 0)
	p = sk.MulPoint(Vec2(0, 1))
	assert.InDelta(t, 1, p.X, 1e-5, "skewed by 45 degrees")
	assert.Equal(t, sk, Identity2().Skew(Pi/4, 0))

	a := Translate2D(1, 2)
	a.SetMul(Scale2D(2, 3))
	assert.Equal(t, Translate2D(1, 2).Mul(Scale2D(2, 3)), a)
	// the scales are those of the decomposition, the larger first
	sx, sy := Rotate2D(0.5).Scale(2, 3).ExtractScale()
	assert.InDelta(t, 3, sx, 1e-4)
	assert.InDelta(t, 2, sy, 1e-4)

	// fixed point: 64 per unit, including the translation
	fp := Translate2D(1, 2).Scale(2, 2).MulFixedPoint(fixed.Point26_6{X: 64, Y: 32})
	assert.Equal(t, fixed.Point26_6{X: 3 * 64, Y: 3 * 64}, fp)

	for s, want := range map[string]float32{"90": Pi / 2, "90deg": Pi / 2, "200grad": Pi, "1.5rad": 1.5, "-45DEG": -Pi / 4} {
		ang, err := ParseAngle32(s)
		assert.NoError(t, err, s)
		assert.InDelta(t, want, ang, 1e-5, s)
	}
	_, err := ParseAngle32("xdeg")
	assert.Error(t, err)
	_, err = ParseFloat32("x")
	assert.Error(t, err)

	// the angles of skews are in degrees, as for rotate
	var s Matrix2
	assert.NoError(t, s.SetString("skewX(45)"))
	assert.InDelta(t, 1, s.MulPoint(Vec2(0, 1)).X, 1e-5)
	assert.NoError(t, s.SetString("skewY(45)"))
	assert.InDelta(t, 1, s.MulPoint(Vec2(1, 0)).Y, 1e-5)
	assert.NoError(t, s.SetString("skew(45, 0)"))
	assert.InDelta(t, 1, s.MulPoint(Vec2(0, 1)).X, 1e-5)
	assert.NoError(t, s.SetString("translateX(3) translateY(4); scaleX(2) scaleY(3)"))
	assert.Equal(t, Translate2D(3, 4).Scale(2, 3), s)
	assert.NoError(t, s.SetString("none"))
	assert.Equal(t, Identity2(), s)
	assert.Error(t, s.SetString("bogus(1)"))
	assert.Error(t, s.SetString("translate"))
	// wrong numbers of values are logged, and ignored
	for _, bad := range []string{"matrix(1 2)", "translate(1 2 3)", "translateX(1 2)", "translateY(1 2)", "scale(1 2 3)", "scaleX(1 2)", "scaleY(1 2)", "rotate(1 2)", "skew(1)", "skewX(1 2)", "skewY(1 2)"} {
		assert.NoError(t, s.SetString(bad), bad)
		assert.Equal(t, Identity2(), s, bad)
	}
	assert.Error(t, PointsCheckN([]float32{1}, 2, "x"))
}

// TestVector3More tests more 3D vector operations.
func TestVector3More(t *testing.T) {
	v := Vec3(1, 2, 3)
	assert.Equal(t, "(1, 2, 3)", v.String())
	assert.Equal(t, "v.Set(1, 2, 3)", v.GenGoSet("v"))
	assert.False(t, v.IsNaN())
	assert.True(t, Vec3(0, NaN(), 0).IsNaN())

	m3 := Matrix3{0, 1, 0, -1, 0, 0, 0, 0, 1} // 90 degrees about z
	assertVec3(t, Vec3(-2, 1, 3), v.MulMatrix3(&m3))
	m4 := Identity4()
	m4.SetTranslation(1, 1, 1)
	assertVec3(t, Vec3(2, 3, 4), v.MulMatrix4AsVector4(m4, 1), "a point is translated")
	assertVec3(t, v, v.MulMatrix4AsVector4(m4, 0), "a direction is not")
	assertVec3(t, v, v.MulProjection(Identity4()))
	var pos Vector3
	pos.SetFromMatrixPos(m4)
	assertVec3(t, Vec3(1, 1, 1), pos)

	size, off := Vec2(200, 100), Vec2(10, 20)
	for _, flip := range []bool{false, true} {
		w := Vec3(0.5, -0.5, 0).NDCToWindow(size, off, 0, 1, flip)
		n := Vec2(w.X, w.Y).WindowToNDC(size, off, flip)
		assert.InDelta(t, 0.5, n.X, 1e-5)
		assert.InDelta(t, -0.5, n.Y, 1e-5)
		assert.InDelta(t, 0.5, w.Z, 1e-5)
	}

	assertVec3(t, Vec3(1, 0, 0), v.ProjectOnVector(Vec3(5, 0, 0)))
	assertVec3(t, Vec3(0, 2, 3), v.ProjectOnPlane(Vec3(1, 0, 0)))
	assertVec3(t, Vec3(1, -2, 3), v.Reflect(Vec3(0, 1, 0)))

	for _, e := range []Vector3{{0.1, 0.2, 0.3}, {-0.5, 0.4, 1}, {0, Pi / 2, 0}} {
		var mat Matrix4
		mat.SetRotationFromEuler(e)
		got := NewEulerAnglesFromMatrix(&mat)
		var back Matrix4
		back.SetRotationFromEuler(got)
		assert.InDeltaSlice(t, mat[:], back[:], 1e-4, "the same rotation: %v", e)
	}
	// the quaternion of Euler angles uses another order of the axes than
	// the matrix (ZYX instead of XYZ), so they agree only for one axis
	for _, e := range []Vector3{{0.4, 0, 0}, {0, -0.3, 0}, {0, 0, 1.2}} {
		var fromQ Vector3
		fromQ.SetEulerAnglesFromQuat(NewQuatEuler(e))
		assertVec3(t, e, fromQ, e)
	}

	for _, n := range []Vector3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 1}} {
		t1, t2 := n.RandomTangents()
		assert.InDelta(t, 0, t1.Dot(n), 1e-5, n)
		assert.InDelta(t, 0, t2.Dot(n), 1e-5, n)
		assert.InDelta(t, 0, t1.Dot(t2), 1e-5, n)
	}
	var zero Vector3
	t1, t2 := zero.RandomTangents()
	assert.Equal(t, Vec3(1, 0, 0), t1, "the axes for no vector")
	assert.Equal(t, Vec3(0, 1, 0), t2)
}

// TestRayPlane tests intersecting rays with planes.
func TestRayPlane(t *testing.T) {
	pl := *NewPlane(Vec3(0, 1, 0), -2) // y = 2
	down := NewRay(Vec3(1, 5, 0), Vec3(0, -1, 0))
	assert.True(t, down.IsIntersectPlane(pl))
	assert.InDelta(t, 3, down.DistanceToPlane(pl), 1e-5)
	p, ok := down.IntersectPlane(pl)
	assert.True(t, ok)
	assertVec3(t, Vec3(1, 2, 0), p)

	up := NewRay(Vec3(1, 5, 0), Vec3(0, 1, 0))
	assert.False(t, up.IsIntersectPlane(pl))
	assert.True(t, IsNaN(up.DistanceToPlane(pl)))
	_, ok = up.IntersectPlane(pl)
	assert.False(t, ok)

	along := NewRay(Vec3(0, 5, 0), Vec3(1, 0, 0))
	assert.True(t, IsNaN(along.DistanceToPlane(pl)), "parallel")
	in := NewRay(Vec3(0, 2, 0), Vec3(1, 0, 0))
	assert.True(t, in.IsIntersectPlane(pl))
	assert.Equal(t, float32(0), in.DistanceToPlane(pl), "in the plane")

	// the distance to segments, in each region of the closest points
	r := NewRay(Vec3(0, 0, 0), Vec3(1, 0, 0))
	var onRay, onSeg Vector3
	assert.InDelta(t, 1, r.DistanceSquaredToSegment(Vec3(2, 1, -1), Vec3(2, 1, 1), &onRay, &onSeg), 1e-5)
	assertVec3(t, Vec3(2, 0, 0), onRay)
	assertVec3(t, Vec3(2, 1, 0), onSeg)
	assert.InDelta(t, 4+1, r.DistanceSquaredToSegment(Vec3(-2, 1, 0), Vec3(-2, 1, 1), nil, nil), 1e-5, "behind the ray")
	assert.InDelta(t, 1, r.DistanceSquaredToSegment(Vec3(1, 1, 0), Vec3(3, 1, 0), nil, nil), 1e-5, "parallel")
	assert.InDelta(t, 4+1, r.DistanceSquaredToSegment(Vec3(-3, 1, 0), Vec3(-2, 1, 0), nil, nil), 1e-5, "parallel behind")
	assert.InDelta(t, 1+1, r.DistanceSquaredToSegment(Vec3(1, 1, 1), Vec3(1, 1, 3), nil, nil), 1e-5, "beyond the segment start")
	assert.InDelta(t, 1+1, r.DistanceSquaredToSegment(Vec3(1, 1, -3), Vec3(1, 1, -1), nil, nil), 1e-5, "beyond the segment end")
}

// TestQuatMore tests quaternions from rotation matrices of each kind, from
// opposite unit vectors, fast normalization, and spherical interpolation.
func TestQuatMore(t *testing.T) {
	axes := []Vector3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 0}}
	for _, axis := range axes {
		for _, ang := range []float32{0.3, Pi * 0.9} {
			q := NewQuatAxisAngle(axis.Normal(), ang)
			var m Matrix4
			m.SetRotationFromQuat(q)
			var got Quat
			got.SetFromRotationMatrix(&m)
			v := Vec3(0.3, -0.7, 0.2)
			assertVec3(t, q.MulVector(v), got.MulVector(v), axis, ang)
		}
	}

	var q Quat
	q.SetFromUnitVectors(Vec3(1, 0, 0), Vec3(-1, 0, 0))
	assertVec3(t, Vec3(-1, 0, 0), q.MulVector(Vec3(1, 0, 0)), "opposite")
	q.SetFromUnitVectors(Vec3(0, 0, 1), Vec3(0, 0, -1))
	assertVec3(t, Vec3(0, 0, -1), q.MulVector(Vec3(0, 0, 1)), "opposite on z")

	n := NewQuat(0, 0, 0, 1.01)
	n.NormalizeFast()
	assert.InDelta(t, 1, n.Length(), 1e-3)
	z := NewQuat(0, 0, 0, Sqrt(3))
	z.NormalizeFast()
	assert.Equal(t, NewQuatIdentity(), z)

	a := NewQuatAxisAngle(Vec3(0, 0, 1), 0)
	b := NewQuatAxisAngle(Vec3(0, 0, 1), 1)
	s := a
	s.Slerp(b, 0)
	assert.Equal(t, a, s)
	s.Slerp(b, 1)
	assert.Equal(t, b, s)
	s = a
	s.Slerp(b, 0.5)
	assertVec3(t, NewQuatAxisAngle(Vec3(0, 0, 1), 0.5).MulVector(Vec3(1, 0, 0)), s.MulVector(Vec3(1, 0, 0)), "halfway")
	neg := NewQuat(-b.X, -b.Y, -b.Z, -b.W) // the same rotation
	s = a
	s.Slerp(neg, 0.5)
	assertVec3(t, NewQuatAxisAngle(Vec3(0, 0, 1), 0.5).MulVector(Vec3(1, 0, 0)), s.MulVector(Vec3(1, 0, 0)), "the short way")
	s = a
	s.Slerp(a, 0.5)
	assert.Equal(t, a, s, "the same")
	c := NewQuatAxisAngle(Vec3(0, 0, 1), 0.01)
	s = a
	s.Slerp(c, 0.5)
	assert.InDelta(t, 1, s.Length(), 1e-5, "nearly the same")
}
