// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cie

import "goki.dev/mat32/v2"

// LABCompress does cube-root compression of the X, Y, Z components
// prior to performing the LAB conversion
func LABCompress(t float32) float32 {
	e := float32(216.0 / 24389.0)
	kappa := float32(24389.0 / 27.0)
	if t > e {
		return mat32.Pow(t, 1.0/3.0)
	}
	return (kappa*t + 16) / 116
}

func LABUncompress(ft float32) float32 {
	e := float32(216.0 / 24389.0)
	kappa := float32(24389.0 / 27.0)
	ft3 := ft * ft * ft
	if ft3 > e {
		return ft3
	}
	return (116*ft - 16) / kappa
}

// XYZToLAB converts a color from XYZ to L*a*b* coordinates
// using the standard D65 illuminant
func XYZToLAB(x, y, z float32) (l, a, b float32) {
	x, y, z = XYZNormD65(x, y, z)
	fx := LABCompress(x)
	fy := LABCompress(y)
	fz := LABCompress(z)
	l = 116*fy - 16
	a = 500 * (fx - fy)
	b = 200 * (fy - fz)
	return
}

// LABToXYZ converts a color from L*a*b* to XYZ coordinates
// using the standard D65 illuminant
func LABToXYZ(l, a, b float32) (x, y, z float32) {
	fy := (l + 16) / 116
	fx := a/500 + fy
	fz := fy - b/200
	x = LABUncompress(fx)
	y = LABUncompress(fy)
	z = LABUncompress(fz)
	x, y, z = XYZDenormD65(x, y, z)
	return
}

// LToY Converts an L* value to a Y value.
// L* in L*a*b* and Y in XYZ measure the same quantity, luminance.
// L* measures perceptual luminance, a linear scale. Y in XYZ
// measures relative luminance, a logarithmic scale.
func LToY(l float32) float32 {
	return 100 * LABUncompress((l+16)/116)
}

// YToL Converts a Y value to an L* value.
// L* in L*a*b* and Y in XYZ measure the same quantity, luminance.
// L* measures perceptual luminance, a linear scale. Y in XYZ
// measures relative luminance, a logarithmic scale.
func YToL(y float32) float32 {
	return LABCompress(y/100)*116 - 16
}
