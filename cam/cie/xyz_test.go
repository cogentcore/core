// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cie

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXYZ(t *testing.T) {
	x, y, z := SRGBLinToXYZ(0.5, 0.6, 0.7)
	assert.Equal(t, float32(0.5470991), x)
	assert.Equal(t, float32(0.58596003), y)
	assert.Equal(t, float32(0.74640036), z)

	rl, gl, bl := XYZToSRGBLin(x, y, z)
	assert.Equal(t, float32(0.5000365), rl)
	assert.Equal(t, float32(0.60003513), gl)
	assert.Equal(t, float32(0.69988275), bl)
}
