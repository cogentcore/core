// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumWorkgroups(t *testing.T) {
	nx, ny := NumWorkgroups1D(4_194_304, 64)
	assert.Equal(t, 65536, nx)
	assert.Equal(t, 1, ny)
	assert.Equal(t, 4_194_304, nx*ny*64)

	nx, ny = NumWorkgroups1D(4_194_304+64, 64)
	assert.Equal(t, 32769, nx)
	assert.Equal(t, 2, ny)
	assert.GreaterOrEqual(t, nx*ny*64, 4_194_304+64)

	nx, ny = NumWorkgroups1D(4_194_304+90, 64)
	assert.Equal(t, 32769, nx)
	assert.Equal(t, 2, ny)
	assert.GreaterOrEqual(t, nx*ny*64, 4_194_304+90)

	nx, ny = NumWorkgroups1D(4_194_304+129, 64)
	assert.Equal(t, 32770, nx)
	assert.Equal(t, 2, ny)
	assert.GreaterOrEqual(t, nx*ny*64, 4_194_304+129)

	nx, ny = NumWorkgroups1D(4_194_304-64, 64)
	assert.Equal(t, 65535, nx)
	assert.Equal(t, 1, ny)
	assert.GreaterOrEqual(t, nx*ny*64, 4_194_304-64)

	nx, ny = NumWorkgroups1D(4_194_304-90, 64)
	assert.Equal(t, 65535, nx)
	assert.Equal(t, 1, ny)
	assert.GreaterOrEqual(t, nx*ny*64, 4_194_304-90)

	nx, ny = NumWorkgroups1D(4_194_304*64, 64)
	assert.Equal(t, 65536, nx)
	assert.Equal(t, 64, ny)
	assert.GreaterOrEqual(t, nx*ny*64, 4_194_304*64)

	nx, ny = NumWorkgroups1D(4_194_304*64, 64)
	assert.Equal(t, 65536, nx)
	assert.Equal(t, 64, ny)
	assert.GreaterOrEqual(t, nx*ny*64, 4_194_304*64)

	nx, ny = NumWorkgroups2D(4_194_304*64, 4, 16)
	assert.Equal(t, 65536, nx)
	assert.Equal(t, 64, ny)
	assert.GreaterOrEqual(t, nx*ny*64, 4_194_304*64)
}
