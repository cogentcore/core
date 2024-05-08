// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"github.com/stretchr/testify/assert"
)

func TestRay(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)

	ray := Ray{
		Origin: origin,
		Dir:    dir,
	}

	assert.Equal(t, origin, ray.Origin)
	assert.Equal(t, dir, ray.Dir)
}

func TestNewRay(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)

	ray := NewRay(origin, dir)

	assert.Equal(t, origin, ray.Origin)
	assert.Equal(t, dir, ray.Dir)
}

func TestSet(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)

	ray := Ray{}
	ray.Set(origin, dir)

	assert.Equal(t, origin, ray.Origin)
	assert.Equal(t, dir, ray.Dir)
}

func TestRayAt(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)

	ray := Ray{
		Origin: origin,
		Dir:    dir,
	}

	t1 := float32(2)
	expected1 := Vec3(9, 12, 15)
	result1 := ray.At(t1)
	assert.Equal(t, expected1, result1)

	t2 := float32(0.5)
	expected2 := Vec3(3, 4.5, 6)
	result2 := ray.At(t2)
	assert.Equal(t, expected2, result2)

	t3 := float32(-1)
	expected3 := Vec3(-3, -3, -3)
	result3 := ray.At(t3)
	assert.Equal(t, expected3, result3)
}

func TestRayRecast(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)

	ray := Ray{
		Origin: origin,
		Dir:    dir,
	}

	t1 := float32(2)
	expected1 := Vec3(9, 12, 15)
	ray.Recast(t1)
	assert.Equal(t, expected1, ray.Origin)

	t2 := float32(0.5)
	expected2 := Vec3(11, 14.5, 18)
	ray.Recast(t2)
	assert.Equal(t, expected2, ray.Origin)

	t3 := float32(-1)
	expected3 := Vec3(7, 9.5, 12)
	ray.Recast(t3)
	assert.Equal(t, expected3, ray.Origin)
}

func TestRayClosestPointToPoint(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)
	ray := Ray{
		Origin: origin,
		Dir:    dir,
	}

	point1 := Vec3(2, 3, 4)
	expected1 := Vec3(61, 77, 93)
	result1 := ray.ClosestPointToPoint(point1)
	assert.Equal(t, expected1, result1)

	point2 := Vec3(0, 1, 2)
	expected2 := Vec3(1, 2, 3)
	result2 := ray.ClosestPointToPoint(point2)
	assert.Equal(t, expected2, result2)

	point3 := Vec3(5, 7, 9)
	expected3 := Vec3(309, 387, 465)
	result3 := ray.ClosestPointToPoint(point3)
	assert.Equal(t, expected3, result3)
}

func TestRayDistanceToPoint(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)
	ray := Ray{
		Origin: origin,
		Dir:    dir,
	}

	point1 := Vec3(2, 3, 4)
	expected1 := float32(129.91536)
	result1 := ray.DistanceToPoint(point1)
	tolassert.Equal(t, expected1, result1)

	point2 := Vec3(0, 1, 2)
	expected2 := float32(1.316074)
	result2 := ray.DistanceToPoint(point2)
	tolassert.Equal(t, expected2, result2)

	point3 := Vec3(5, 7, 9)
	expected3 := float32(666.8973)
	result3 := ray.DistanceToPoint(point3)
	tolassert.Equal(t, expected3, result3)
}

func TestRayDistanceSquaredToPoint(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)
	ray := Ray{
		Origin: origin,
		Dir:    dir,
	}

	point1 := Vec3(2, 3, 4)
	expected1 := float32(16878)
	result1 := ray.DistanceSquaredToPoint(point1)
	assert.Equal(t, expected1, result1)

	point2 := Vec3(0, 1, 2)
	expected2 := float32(1.7320508)
	result2 := ray.DistanceSquaredToPoint(point2)
	assert.Equal(t, expected2, result2)

	point3 := Vec3(5, 7, 9)
	expected3 := float32(444752)
	result3 := ray.DistanceSquaredToPoint(point3)
	assert.Equal(t, expected3, result3)
}

func TestRayDistanceSquaredToSegment(t *testing.T) {
	tests := []struct {
		name               string
		ray                Ray
		v0, v1             Vector3
		optPointOnRay      *Vector3
		optPointOnSegment  *Vector3
		expectedSqrDist    float32
		expectedPointOnRay Vector3
		expectedPointOnSeg Vector3
	}{
		{
			ray: Ray{
				Origin: Vec3(1, 2, 3),
				Dir:    Vec3(4, 5, 6),
			},
			v0:                 Vec3(2, 3, 4),
			v1:                 Vec3(5, 6, 7),
			expectedSqrDist:    -3552,
			expectedPointOnRay: Vec3(241, 302, 363),
			expectedPointOnSeg: Vec3(5, 6, 7),
		},
		{
			ray: Ray{
				Origin: Vec3(1, 2, 3),
				Dir:    Vec3(4, 5, 6),
			},
			v0:                 Vec3(7, 8, 9),
			v1:                 Vec3(10, 11, 12),
			expectedSqrDist:    -17982,
			expectedPointOnRay: Vec3(541, 677, 813),
			expectedPointOnSeg: Vec3(10, 11, 12),
		},
	}

	for _, test := range tests {
		test.optPointOnRay = &Vector3{}
		test.optPointOnSegment = &Vector3{}
		sqrDist := test.ray.DistanceSquaredToSegment(test.v0, test.v1, test.optPointOnRay, test.optPointOnSegment)
		tolassert.Equal(t, test.expectedSqrDist, sqrDist)
		tolassert.Equal(t, test.expectedPointOnRay.X, test.optPointOnRay.X)
		tolassert.Equal(t, test.expectedPointOnRay.Y, test.optPointOnRay.Y)
		tolassert.Equal(t, test.expectedPointOnRay.Z, test.optPointOnRay.Z)
		tolassert.Equal(t, test.expectedPointOnSeg.X, test.optPointOnSegment.X)
		tolassert.Equal(t, test.expectedPointOnSeg.Y, test.optPointOnSegment.Y)
		tolassert.Equal(t, test.expectedPointOnSeg.Z, test.optPointOnSegment.Z)
	}
}

func TestRayIsIntersectionSphere(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)
	ray := Ray{
		Origin: origin,
		Dir:    dir,
	}

	sphere1 := Sphere{
		Center: Vec3(0, 0, 0),
		Radius: 10,
	}
	assert.True(t, ray.IsIntersectionSphere(sphere1))

	sphere2 := Sphere{
		Center: Vec3(10, 10, 10),
		Radius: 5,
	}
	assert.False(t, ray.IsIntersectionSphere(sphere2))
}

func TestRayIntersectSphere(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)
	ray := Ray{
		Origin: origin,
		Dir:    dir,
	}

	sphere1 := Sphere{
		Center: Vec3(0, 0, 0),
		Radius: 10,
	}
	expected1 := Vec3(6.2666473, 8.583309, 10.899971)
	result1, intersect1 := ray.IntersectSphere(sphere1)
	assert.True(t, intersect1)
	assert.Equal(t, expected1, result1)

	sphere2 := Sphere{
		Center: Vec3(1, 1, 1),
		Radius: 0.1,
	}
	expected2 := Vec3(0, -1, -2)
	result2, intersect2 := ray.IntersectSphere(sphere2)
	assert.False(t, intersect2)
	assert.Equal(t, expected2, result2)
}

func TestRayIntersectBox(t *testing.T) {
	ray := Ray{
		Origin: Vec3(1, 2, 3),
		Dir:    Vec3(4, 5, 6),
	}

	box1 := Box3{
		Min: Vec3(0, 0, 0),
		Max: Vec3(10, 10, 10),
	}
	expected1 := Vec3(5.666667, 7.833334, 10)
	result1, intersect1 := ray.IntersectBox(box1)
	assert.True(t, intersect1)
	assert.Equal(t, expected1, result1)

	box2 := Box3{
		Min: Vec3(2, 2, 2),
		Max: Vec3(5, 5, 5),
	}
	expected2 := Vec3(2, 3.25, 4.5)
	result2, intersect2 := ray.IntersectBox(box2)
	assert.True(t, intersect2)
	assert.Equal(t, expected2, result2)

	box3 := Box3{
		Min: Vec3(-10, -10, -10),
		Max: Vec3(-5, -5, -5),
	}
	expected3 := Vec3(1, 2, 3)
	result3, intersect3 := ray.IntersectBox(box3)
	assert.False(t, intersect3)
	assert.Equal(t, expected3, result3)
}

func TestRayIntersectTriangle(t *testing.T) {
	ray := Ray{
		Origin: Vec3(1, 2, 3),
		Dir:    Vec3(4, 5, 6),
	}

	a := Vec3(0, 0, 0)
	b := Vec3(1, 0, 0)
	c := Vec3(0, 1, 0)

	expected1 := Vec3(1, 2, 3)
	result1, intersect1 := ray.IntersectTriangle(a, b, c, false)
	assert.False(t, intersect1)
	assert.Equal(t, expected1, result1)

	expected2 := Vec3(1, 2, 3)
	result2, intersect2 := ray.IntersectTriangle(a, b, c, true)
	assert.False(t, intersect2)
	assert.Equal(t, expected2, result2)

	d := Vec3(0, 0, 1)

	expected3 := Vec3(1, 2, 3)
	result3, intersect3 := ray.IntersectTriangle(a, b, d, false)
	assert.False(t, intersect3)
	assert.Equal(t, expected3, result3)

	expected4 := Vec3(1, 2, 3)
	result4, intersect4 := ray.IntersectTriangle(b, c, d, false)
	assert.False(t, intersect4)
	assert.Equal(t, expected4, result4)
}

func TestRayApplyMatrix4(t *testing.T) {
	origin := Vec3(1, 2, 3)
	dir := Vec3(4, 5, 6)
	mat4 := Matrix4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}

	ray := Ray{
		Origin: origin,
		Dir:    dir,
	}

	ray.ApplyMatrix4(&mat4)

	expectedOrigin := Vec3(1, 2, 3)
	expectedDir := Vec3(0.45584232, 0.5698029, 0.6837635)

	assert.Equal(t, expectedOrigin, ray.Origin)
	assert.Equal(t, expectedDir, ray.Dir)
}
