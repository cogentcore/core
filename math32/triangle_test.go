// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBarycoordFromPoint(t *testing.T) {
	tests := []struct {
		point, a, b, c Vector3
		expected       Vector3
	}{
		{
			point:    Vec3(0, 0, 0),
			a:        Vec3(0, 0, 0),
			b:        Vec3(1, 0, 0),
			c:        Vec3(0, 1, 0),
			expected: Vec3(1, 0, 0),
		},
		{
			point:    Vec3(0.5, 0.5, 0),
			a:        Vec3(0, 0, 0),
			b:        Vec3(1, 0, 0),
			c:        Vec3(0, 1, 0),
			expected: Vec3(0, 0.5, 0.5),
		},
	}

	for _, tc := range tests {
		result := BarycoordFromPoint(tc.point, tc.a, tc.b, tc.c)
		assert.Equal(t, tc.expected, result)
	}
}
