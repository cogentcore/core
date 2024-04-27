// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cie

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"github.com/stretchr/testify/assert"
)

func TestSRGB(t *testing.T) {
	tolassert.Equal(t, float32(0.00015479876), SRGBToLinearComp(0.002))
	tolassert.Equal(t, float32(0.23302202), SRGBToLinearComp(0.52))

	tolassert.Equal(t, float32(0.012920001), SRGBFromLinearComp(0.001))
	tolassert.Equal(t, float32(0.84338915), SRGBFromLinearComp(0.68))

	rl, gl, bl := SRGBToLinear(0.3, 0.2, 0.6)
	tolassert.Equal(t, float32(0.07323897), rl)
	tolassert.Equal(t, float32(0.033104762), gl)
	tolassert.Equal(t, float32(0.31854683), bl)

	rl, gl, bl = SRGB100ToLinear(0.3, 0.2, 0.6)
	tolassert.Equal(t, float32(7.323897), rl)
	tolassert.Equal(t, float32(3.3104763), gl)
	tolassert.Equal(t, float32(31.854683), bl)

	r, g, b := SRGBFromLinear(0.12, 0.34, 0.78)
	tolassert.Equal(t, float32(0.38109186), r)
	tolassert.Equal(t, float32(0.61803144), g)
	tolassert.Equal(t, float32(0.8962438), b)

	r, g, b = SRGBFromLinear100(12, 34, 78)
	tolassert.Equal(t, float32(0.38109186), r)
	tolassert.Equal(t, float32(0.61803144), g)
	tolassert.Equal(t, float32(0.8962438), b)

	ur, ug, ub, ua := SRGBFloatToUint8(0.36, 0.81, 0.41, 0.9)
	assert.Equal(t, uint8(0x53), ur)
	assert.Equal(t, uint8(0xba), ug)
	assert.Equal(t, uint8(0x5e), ub)
	assert.Equal(t, uint8(0xe6), ua)

	ur32, ug32, ub32, ua32 := SRGBFloatToUint32(0.36, 0.81, 0.41, 0.9)
	assert.Equal(t, uint32(0x52f1), ur32)
	assert.Equal(t, uint32(0xba9f), ug32)
	assert.Equal(t, uint32(0x5e76), ub32)
	assert.Equal(t, uint32(0xe666), ua32)

	fr, fg, fb, fa := SRGBUint8ToFloat(18, 201, 157, 198)
	tolassert.Equal(t, float32(0.09090909), fr)
	tolassert.Equal(t, float32(1.0151515), fg)
	tolassert.Equal(t, float32(0.7929293), fb)
	tolassert.Equal(t, float32(0.7764706), fa)

	fr, fg, fb, fa = SRGBUint32ToFloat(21022, 10836, 15893, 27980)
	tolassert.Equal(t, float32(0.7513223), fr)
	tolassert.Equal(t, float32(0.3872766), fg)
	tolassert.Equal(t, float32(0.56801283), fb)
	tolassert.Equal(t, float32(0.42694744), fa)
}
