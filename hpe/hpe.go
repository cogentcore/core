// Copyright (c) 2021, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hpe

import "goki.dev/cam/cie"

// XYZToLMS convert XYZ to Long, Medium, Short cone-based responses,
// using the Hunt-Pointer-Estevez transform.
// This is closer to the actual response functions of the L,M,S cones apparently.
func XYZToLMS(x, y, z float32) (l, m, s float32) {
	l = 0.38971*x + 0.68898*y + -0.07868*z
	m = -0.22981*x + 1.18340*y + 0.04641*z
	s = z
	return
}

// SRGBLinToLMS converts sRGB linear to Long, Medium, Short cone-based responses,
// using the Hunt-Pointer-Estevez transform.
// This is closer to the actual response functions of the L,M,S cones apparently.
func SRGBLinToLMS(rl, gl, bl float32) (l, m, s float32) {
	l = 0.30567503*rl + 0.62274014*gl + 0.04530167*bl
	m = 0.15771291*rl + 0.7697197*gl + 0.08807348*bl
	s = 0.0193*rl + 0.1192*gl + 0.9505*bl
	return
}

// SRGBToLMS converts sRGB to Long, Medium, Short cone-based responses,
// using the Hunt-Pointer-Estevez transform.
// This is closer to the actual response functions of the L,M,S cones apparently.
func SRGBToLMS(r, g, b float32) (l, m, s float32) {
	rl, gl, bl := cie.SRGBToLinear(r, g, b)
	l, m, s = SRGBLinToLMS(rl, gl, bl)
	return
}

/*
  func LMStoXYZ(float& X, float& Y, float& Z,
                                    L, M, S) {
    X = 1.096124f * L + 0.4296f * Y + -0.1624f * Z;
    Y = -0.7036f * X + 1.6975f * Y + 0.0061f * Z;
    Z = 0.0030f * X + 0.0136f * Y + 0.9834 * Z;
  }
  // convert Long, Medium, Short cone-based responses to XYZ, using the Hunt-Pointer-Estevez transform -- this is closer to the actual response functions of the L,M,S cones apparently
*/
