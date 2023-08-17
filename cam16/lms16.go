// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cam16

import "github.com/goki/mat32"

// XYZToLMS converts XYZ to Long, Medium, Short cone-based responses,
// using the CAT16 transform from CIECAM16 color appearance model
// (LiLiWangEtAl17)
func XYZToLMS(x, y, z float32) (l, m, s float32) {
	l = x*0.401288 + y*0.650173 + z*-0.051461
	m = x*-0.250268 + y*1.204414 + z*0.045854
	s = x*-0.002079 + y*0.048952 + z*0.953127
	return
}

// LMSToXYZ converts Long, Medium, Short cone-based responses to XYZ
// using the CAT16 transform from CIECAM16 color appearance model
// (LiLiWangEtAl17)
func LMSToXYZ(l, m, s float32) (x, y, z float32) {
	x = l*1.86206787 + m*-1.0112563 + s*0.14918667
	y = l*0.38752654 + m*0.62144744 + s*-0.00897398
	z = l*-0.01584150 + m*-0.03412294 + s*1.04996444
	return
}

// LuminanceAdaptComp performs luminance adaptation
// and response compression according to the CAM16 model,
// on one component, using equations from HuntLiLuo03
// d = discount factor
// fl = luminance adaptation factor
func LuminanceAdaptComp(v, d, fl float32) float32 {
	vd := v * d
	f := mat32.Pow((fl*mat32.Abs(vd))/100, 0.42)
	return (mat32.Sign(vd) * 400 * f) / (f + 27.13)
}

// LuminanceAdapt performs luminance adaptation
// and response compression according to the CAM16 model,
// on given r,g,b components, using equations from HuntLiLuo03
// and parameters on given viewing conditions
func LuminanceAdapt(l, m, s float32, vw *View) (lA, mA, sA float32) {
	lA = LuminanceAdaptComp(l, vw.RGBD.X, vw.FL)
	mA = LuminanceAdaptComp(m, vw.RGBD.Y, vw.FL)
	sA = LuminanceAdaptComp(s, vw.RGBD.Z, vw.FL)
	return
}

// LMSToOps converts Long, Medium, Short cone-based values to
// opponent redVgreen (a) and yellowVblue (b), and grey (achromatic) values,
// that more closely reflect neural responses.
// greyNorm is a normalizing grey factor used in the CAM16 model.
// Uses the CIECAM16 color appearance model.
func LMSToOps(l, m, s float32, vw *View) (redVgreen, yellowVblue, grey, greyNorm float32) {
	// Discount illuminant and adapt
	lA, mA, sA := LuminanceAdapt(l, m, s, vw)
	redVgreen = (11*lA + -12*mA + sA) / 11
	yellowVblue = (lA + mA - 2*sA) / 9
	// auxiliary components
	grey = (40.0*lA + 20.0*mA + sA) / 20.0          // achromatic response, multiplied * view.NBB
	greyNorm = (20.0*lA + 20.0*mA + 21.0*sA) / 20.0 // normalizing factor
	return
}
