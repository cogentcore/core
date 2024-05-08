// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

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
