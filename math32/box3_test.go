// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBox3MulMatrix4(t *testing.T) {
	b := Box3{
		Min: Vector3{X: 1, Y: 2, Z: 3},
		Max: Vector3{X: 4, Y: 5, Z: 6},
	}
	m := &Matrix4{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}

	expected := Box3{
		Min: Vector3{X: 51, Y: 58, Z: 65},
		Max: Vector3{X: 96, Y: 112, Z: 128},
	}

	result := b.MulMatrix4(m)

	assert.Equal(t, expected, result)
}

func TestBox3MulQuat(t *testing.T) {
	b := Box3{
		Min: Vector3{X: 1, Y: 2, Z: 3},
		Max: Vector3{X: 4, Y: 5, Z: 6},
	}
	q := Quat{
		X: 0.5,
		Y: 0.5,
		Z: 0.5,
		W: 0.5,
	}

	expected := Box3{
		Min: Vector3{X: 3, Y: 1, Z: 2},
		Max: Vector3{X: 6, Y: 4, Z: 5},
	}

	result := b.MulQuat(q)

	assert.Equal(t, expected, result)
}

func TestBox3MVProjToNDC(t *testing.T) {
	b := Box3{
		Min: Vector3{X: 1, Y: 2, Z: 3},
		Max: Vector3{X: 4, Y: 5, Z: 6},
	}
	m := &Matrix4{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}

	expected := Box3{
		Min: Vector3{X: 0.6388889, Y: 0.7592593, Z: 0.8796296},
		Max: Vector3{X: 0.7222222, Y: 0.8148148, Z: 0.9074074},
	}

	result := b.MVProjToNDC(m)

	assert.Equal(t, expected, result)
}
