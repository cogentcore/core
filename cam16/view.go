// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from https://github.com/material-foundation/material-color-utilities
// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cam16

import (
	"goki.dev/cam/cie"
	"goki.dev/mat32/v2"
)

// View represents viewing conditions under which a color is being perceived,
// which greatly affects the subjective perception.  Defaults represent the
// standard defined such conditions, under which the CAM16 computations operate.
type View struct {

	// white point illumination -- typically cie.WhiteD65
	WhitePoint mat32.Vec3 `desc:"white point illumination -- typically cie.WhiteD65"`

	// [def: 200] the ambient light strength in lux
	Luminance float32 `def:"200" desc:"the ambient light strength in lux"`

	// [def: 50] the average luminance of 10 degrees around the color in question
	BgLuminance float32 `def:"50" desc:"the average luminance of 10 degrees around the color in question"`

	// [def: 2] the brightness of the entire environment
	Surround float32 `def:"2" desc:"the brightness of the entire environment"`

	// [def: false] whether the person's eyes have adapted to the lighting
	Adapted bool `def:"false" desc:"whether the person's eyes have adapted to the lighting"`

	// [view: -] computed from Luminance
	AdaptingLuminance float32 `view:"-" desc:"computed from Luminance"`

	// [view: -]
	BgYToWhiteY float32 `view:"-"`

	// [view: -]
	AW float32 `view:"-"`

	// [view: -] luminance level induction factor
	NBB float32 `view:"-" desc:"luminance level induction factor"`

	// [view: -] luminance level induction factor
	NCB float32 `view:"-" desc:"luminance level induction factor"`

	// [view: -] exponential nonlinearity
	C float32 `view:"-" desc:"exponential nonlinearity"`

	// [view: -] chromatic induction factor
	NC float32 `view:"-" desc:"chromatic induction factor"`

	// [view: -] luminance-level adaptation factor, based on the HuntLiLuo03 equations
	FL float32 `view:"-" desc:"luminance-level adaptation factor, based on the HuntLiLuo03 equations"`

	// [view: -] FL to the 1/4 power
	FLRoot float32 `view:"-" desc:"FL to the 1/4 power"`

	// [view: -]
	Z float32 `view:"-" desc:"base exponential nonlinearity`

	// [view: -]
	DRGBInverse mat32.Vec3 `view:"-" desc:"inverse of the RGBD factors`

	// [view: -] cone responses to white point, adjusted for discounting
	RGBD mat32.Vec3 `view:"-" desc:"cone responses to white point, adjusted for discounting"`
}

// NewView returns a new view with all parameters initialized based on given major params
func NewView(whitePoint mat32.Vec3, lum, bgLum, surround float32, adapt bool) *View {
	vw := &View{WhitePoint: whitePoint, Luminance: lum, BgLuminance: bgLum, Surround: surround, Adapted: adapt}
	vw.Update()
	return vw
}

// TheStdView is the standard viewing conditions view
// returned by NewStdView if already created.
var TheStdView *View

// NewStdView returns a new standard viewing conditions model
// returns TheStdView if already created
func NewStdView() *View {
	if TheStdView != nil {
		return TheStdView
	}
	TheStdView = NewView(cie.WhiteD65, 200, 50, 2, false)
	return TheStdView
}

// Update updates all the computed values based on main parameters
func (vw *View) Update() {
	vw.AdaptingLuminance = (vw.Luminance / mat32.Pi) * (cie.LToY(50) / 100)
	// A background of pure black is non-physical and leads to infinities that
	// represent the idea that any color viewed in pure black can't be seen.
	vw.BgLuminance = mat32.Max(0.1, vw.BgLuminance)

	// Transform test illuminant white in XYZ to 'cone'/'rgb' responses
	rW, gW, bW := XYZToLMS(vw.WhitePoint.X, vw.WhitePoint.Y, vw.WhitePoint.Z)

	// Scale input surround, domain (0, 2), to CAM16 surround, domain (0.8, 1.0)
	vw.Surround = mat32.Clamp(vw.Surround, 0, 2)
	f := 0.8 + (vw.Surround / 10)
	// "Exponential non-linearity"
	if f >= 0.9 {
		vw.C = mat32.Lerp(0.59, 0.69, ((f - 0.9) * 10))
	} else {
		vw.C = mat32.Lerp(0.525, 0.59, ((f - 0.8) * 10))
	}
	// Calculate degree of adaptation to illuminant
	d := float32(1)
	if !vw.Adapted {
		d = f * (1 - ((1 / 3.6) * mat32.Exp((-vw.AdaptingLuminance-42)/92)))
	}

	// Per Li et al, if D is greater than 1 or less than 0, set it to 1 or 0.
	d = mat32.Clamp(d, 0, 1)

	// chromatic induction factor
	vw.NC = f

	// Cone responses to the whitePoint, r/g/b/W, adjusted for discounting.
	//
	// Why use 100 instead of the white point's relative luminance?
	//
	// Some papers and implementations, for both CAM02 and CAM16, use the Y
	// value of the reference white instead of 100. Fairchild's Color Appearance
	// Models (3rd edition) notes that this is in error: it was included in the
	// CIE 2004a report on CIECAM02, but, later parts of the conversion process
	// account for scaling of appearance relative to the white point relative
	// luminance. This part should simply use 100 as luminance.
	vw.RGBD.X = d*(100/rW) + 1 - d
	vw.RGBD.Y = d*(100/gW) + 1 - d
	vw.RGBD.Z = d*(100/bW) + 1 - d

	// Factor used in calculating meaningful factors
	k := 1 / (5*vw.AdaptingLuminance + 1)
	k4 := k * k * k * k
	k4F := 1 - k4

	// Luminance-level adaptation factor
	vw.FL = (k4 * vw.AdaptingLuminance) +
		(0.1 * k4F * k4F * mat32.Pow(5*vw.AdaptingLuminance, 1.0/3.0))

	vw.FLRoot = mat32.Pow(vw.FL, 0.25)

	// Intermediate factor, ratio of background relative luminance to white relative luminance
	n := cie.LToY(vw.BgLuminance) / vw.WhitePoint.Y
	vw.BgYToWhiteY = n

	// Base exponential nonlinearity
	// note Schlomer 2018 has a typo and uses 1.58, the correct factor is 1.48
	vw.Z = 1.48 + mat32.Sqrt(n)

	// Luminance-level induction factors
	vw.NBB = 0.725 / mat32.Pow(n, 0.2)
	vw.NCB = vw.NBB

	// Discounted cone responses to the white point, adjusted for post-saturation
	// adaptation perceptual nonlinearities.
	rA, gA, bA := LuminanceAdapt(rW, gW, bW, vw)

	vw.AW = ((40*rA + 20*gA + bA) / 20) * vw.NBB
}
