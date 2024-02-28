// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cie

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLAB(t *testing.T) {
	assert.Equal(t, float32(0.887904), LABCompress(0.7))
	assert.Equal(t, float32(0.21600002), LABUncompress(0.6))

	l, a, b := XYZToLAB(0.1, 0.3, 0.5)
	assert.Equal(t, float32(61.65422), l)
	assert.Equal(t, float32(-98.673805), a)
	assert.Equal(t, float32(-20.413673), b)

	x, y, z := LABToXYZ(28, 14, 36.2)
	assert.Equal(t, float32(0.06422656), x)
	assert.Equal(t, float32(0.054573778), y)
	assert.Equal(t, float32(0.008442593), z)

	assert.Equal(t, float32(2.3023312), LToY(17))
	assert.Equal(t, float32(21.579498), YToL(3.4))
}
