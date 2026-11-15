// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNumWorkgroups tests that the work groups are enough for all of the
// elements, with no more than [MaxWorkgroupsPerDimension] per dimension,
// and only one row (ny) when they fit in it.
func TestNumWorkgroups(t *testing.T) {
	for _, n := range []int{1, 64, 4_194_304 - 90, 4_194_304 - 64, 4_194_304, 4_194_304 + 64, 4_194_304 + 129, 4_194_304 * 64} {
		nx, ny := NumWorkgroups1D(n, 64)
		assert.LessOrEqual(t, nx, MaxWorkgroupsPerDimension, "n: %d", n)
		assert.GreaterOrEqual(t, nx*ny*64, n, "n: %d", n)
		if n <= MaxWorkgroupsPerDimension*64 {
			assert.Equal(t, 1, ny, "n: %d", n)
		}
		nx, ny = NumWorkgroups2D(n, 4, 16)
		assert.LessOrEqual(t, nx, MaxWorkgroupsPerDimension, "2D n: %d", n)
		assert.GreaterOrEqual(t, nx*ny*64, n, "2D n: %d", n)
	}
	nx, ny := NumWorkgroups1D(4_194_304, 64)
	assert.Equal(t, 32768, nx, "65536 groups do not fit in one row")
	assert.Equal(t, 2, ny)
	nx, ny = NumWorkgroups1D(4_194_304-64, 64)
	assert.Equal(t, 65535, nx)
	assert.Equal(t, 1, ny)
}
