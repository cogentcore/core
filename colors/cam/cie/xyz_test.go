// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cie

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
)

func TestXYZ(t *testing.T) {
	x, y, z := SRGBLinToXYZ(0.5, 0.6, 0.7)
	tolassert.Equal(t, float32(0.5470991), x)
	tolassert.Equal(t, float32(0.58596003), y)
	tolassert.Equal(t, float32(0.74640036), z)

	rl, gl, bl := XYZToSRGBLin(x, y, z)
	tolassert.Equal(t, float32(0.5000365), rl)
	tolassert.Equal(t, float32(0.60003513), gl)
	tolassert.Equal(t, float32(0.69988275), bl)

	x, y, z = SRGBToXYZ(0.5, 0.6, 0.7)
	tolassert.Equal(t, float32(0.283043), x)
	tolassert.Equal(t, float32(0.3056746), y)
	tolassert.Equal(t, float32(0.46783832), z)

	r, g, b := XYZToSRGB(x, y, z)
	tolassert.Equal(t, float32(0.50004405), r)
	tolassert.Equal(t, float32(0.60001075), g)
	tolassert.Equal(t, float32(0.699947), b)

	x, y, z = SRGBToXYZ100(0.5, 0.6, 0.7)
	tolassert.Equal(t, float32(28.304298), x)
	tolassert.Equal(t, float32(30.56746), y)
	tolassert.Equal(t, float32(46.783833), z)

	r, g, b = XYZ100ToSRGB(x, y, z)
	tolassert.Equal(t, float32(0.5000441), r)
	tolassert.Equal(t, float32(0.60001075), g)
	tolassert.Equal(t, float32(0.699947), b)

	xr, yr, zr := XYZNormD65(0.43, 0.81, 0.19)
	tolassert.Equal(t, float32(0.45240778), xr)
	tolassert.Equal(t, float32(0.81), yr)
	tolassert.Equal(t, float32(0.17449923), zr)

	xr, yr, zr = XYZDenormD65(xr, yr, zr)
	tolassert.Equal(t, float32(0.43), xr)
	tolassert.Equal(t, float32(0.81), yr)
	tolassert.Equal(t, float32(0.19), zr)
}
