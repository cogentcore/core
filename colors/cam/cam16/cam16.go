// Copyright (c) 2023, Cogent Core. All rights reserved.
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
	"image/color"

	"cogentcore.org/core/base/num"
	"cogentcore.org/core/colors/cam/cie"
	"cogentcore.org/core/math32"
)

// CAM represents a point in the cam16 color model along 6 dimensions
// representing the perceived hue, colorfulness, and brightness,
// similar to HSL but much more well-calibrated to actual human subjective judgments.
type CAM struct {

	// hue (h) is the spectral identity of the color (red, green, blue etc) in degrees (0-360)
	Hue float32

	// chroma (C) is the colorfulness or saturation of the color -- greyscale colors have no chroma, and fully saturated ones have high chroma
	Chroma float32

	// colorfulness (M) is the absolute chromatic intensity
	Colorfulness float32

	// saturation (s) is the colorfulness relative to brightness
	Saturation float32

	// brightness (Q) is the apparent amount of light from the color, which is not a simple function of actual light energy emitted
	Brightness float32

	// lightness (J) is the brightness relative to a reference white, which varies as a function of chroma and hue
	Lightness float32
}

// RGBA implements the color.Color interface.
func (cam *CAM) RGBA() (r, g, b, a uint32) {
	x, y, z := cam.XYZ()
	rf, gf, bf := cie.XYZ100ToSRGB(x, y, z)
	return cie.SRGBFloatToUint32(rf, gf, bf, 1)
}

// AsRGBA returns the color as a [color.RGBA].
func (cam *CAM) AsRGBA() color.RGBA {
	x, y, z := cam.XYZ()
	rf, gf, bf := cie.XYZ100ToSRGB(x, y, z)
	r, g, b, a := cie.SRGBFloatToUint8(rf, gf, bf, 1)
	return color.RGBA{r, g, b, a}
}

// UCS returns the CAM16-UCS components based on the the CAM values
func (cam *CAM) UCS() (j, m, a, b float32) {
	j = (1 + 100*0.007) * cam.Lightness / (1 + 0.007*cam.Lightness)
	m = math32.Log(1+0.0228*cam.Colorfulness) / 0.0228
	hr := math32.DegToRad(cam.Hue)
	a = m * math32.Cos(hr)
	b = m * math32.Sin(hr)
	return
}

// FromUCS returns CAM values from the given CAM16-UCS coordinates
// (jstar, astar, and bstar), under standard viewing conditions
func FromUCS(j, a, b float32) *CAM {
	return FromUCSView(j, a, b, NewStdView())
}

// FromUCS returns CAM values from the given CAM16-UCS coordinates
// (jstar, astar, and bstar), using the given viewing conditions
func FromUCSView(j, a, b float32, vw *View) *CAM {
	m := math32.Sqrt(a*a + b*b)
	M := (math32.Exp(m*0.0228) - 1) / 0.0228
	c := M / vw.FLRoot
	h := math32.RadToDeg(math32.Atan2(b, a))
	if h < 0 {
		h += 360
	}
	j /= 1 - (j-100)*0.007

	return FromJCHView(j, c, h, vw)
}

// FromJCH returns CAM values from the given lightness (j), chroma (c),
// and hue (h) values under standard viewing condition
func FromJCH(j, c, h float32) *CAM {
	return FromJCHView(j, c, h, NewStdView())
}

// FromJCHView returns CAM values from the given lightness (j), chroma (c),
// and hue (h) values under the given viewing conditions
func FromJCHView(j, c, h float32, vw *View) *CAM {
	cam := &CAM{Lightness: j, Chroma: c, Hue: h}
	cam.Brightness = (4 / vw.C) *
		math32.Sqrt(cam.Lightness/100) *
		(vw.AW + 4) *
		(vw.FLRoot)
	cam.Colorfulness = cam.Chroma * vw.FLRoot
	alpha := cam.Chroma / math32.Sqrt(cam.Lightness/100)
	cam.Saturation = 50 * math32.Sqrt((alpha*vw.C)/(vw.AW+4))
	return cam
}

// FromSRGB returns CAM values from given SRGB color coordinates,
// under standard viewing conditions.  The RGB value range is 0-1,
// and RGB values have gamma correction.
func FromSRGB(r, g, b float32) *CAM {
	return FromXYZ(cie.SRGBToXYZ100(r, g, b))
}

// FromXYZ returns CAM values from given XYZ color coordinate,
// under standard viewing conditions
func FromXYZ(x, y, z float32) *CAM {
	return FromXYZView(x, y, z, NewStdView())
}

// FromXYZView returns CAM values from given XYZ color coordinate,
// under given viewing conditions.  Requires 100-base XYZ coordinates.
func FromXYZView(x, y, z float32, vw *View) *CAM {
	l, m, s := XYZToLMS(x, y, z)
	redVgreen, yellowVblue, grey, greyNorm := LMSToOps(l, m, s, vw)

	hue := SanitizeDegrees(math32.RadToDeg(math32.Atan2(yellowVblue, redVgreen)))
	// achromatic response to color
	ac := grey * vw.NBB

	// CAM16 lightness and brightness
	J := 100 * math32.Pow(ac/vw.AW, vw.C*vw.Z)
	Q := (4 / vw.C) * math32.Sqrt(J/100) * (vw.AW + 4) * (vw.FLRoot)

	huePrime := hue
	if hue < 20.14 {
		huePrime += 360
	}
	eHue := 0.25 * (math32.Cos(huePrime*math32.Pi/180+2) + 3.8)
	p1 := 50000 / 13 * eHue * vw.NC * vw.NCB
	t := p1 * math32.Sqrt(redVgreen*redVgreen+yellowVblue*yellowVblue) / (greyNorm + 0.305)
	alpha := math32.Pow(t, 0.9) * math32.Pow(1.64-math32.Pow(0.29, vw.BgYToWhiteY), 0.73)

	// CAM16 chroma, colorfulness, chroma
	C := alpha * math32.Sqrt(J/100)
	M := C * vw.FLRoot
	s = 50 * math32.Sqrt((alpha*vw.C)/(vw.AW+4))
	return &CAM{Hue: hue, Chroma: C, Colorfulness: M, Saturation: s, Brightness: Q, Lightness: J}
}

// XYZ returns the CAM color as XYZ coordinates
// under standard viewing conditions.
// Returns 100-base XYZ coordinates.
func (cam *CAM) XYZ() (x, y, z float32) {
	return cam.XYZView(NewStdView())
}

// XYZ returns the CAM color as XYZ coordinates
// under the given viewing conditions.
// Returns 100-base XYZ coordinates.
func (cam *CAM) XYZView(vw *View) (x, y, z float32) {
	alpha := float32(0)
	if cam.Chroma != 0 || cam.Lightness != 0 {
		alpha = cam.Chroma / math32.Sqrt(cam.Lightness/100)
	}

	t := math32.Pow(
		alpha/
			math32.Pow(
				1.64-
					math32.Pow(0.29, vw.BgYToWhiteY),
				0.73),
		1.0/0.9)

	hRad := math32.DegToRad(cam.Hue)

	eHue := 0.25 * (math32.Cos(hRad+2) + 3.8)
	ac := vw.AW * math32.Pow(cam.Lightness/100, 1/vw.C/vw.Z)
	p1 := eHue * (50000 / 13) * vw.NC * vw.NCB

	p2 := ac / vw.NBB

	hSin := math32.Sin(hRad)
	hCos := math32.Cos(hRad)

	gamma := 23 *
		(p2 + 0.305) *
		t /
		(23*p1 + 11*t*hCos + 108*t*hSin)
	a := gamma * hCos
	b := gamma * hSin
	rA := (460*p2 + 451*a + 288*b) / 1403
	gA := (460*p2 - 891*a - 261*b) / 1403
	bA := (460*p2 - 220*a - 6300*b) / 1403

	rCBase := max(0, (27.13*num.Abs(rA))/(400-num.Abs(rA)))
	// TODO(kai): their sign function returns 0 for 0, but we return 1, so this might break
	rC := math32.Sign(rA) *
		(100 / vw.FL) *
		math32.Pow(rCBase, 1/0.42)
	gCBase := max(0, (27.13*num.Abs(gA))/(400-num.Abs(gA)))
	gC := math32.Sign(gA) *
		(100 / vw.FL) *
		math32.Pow(gCBase, 1/0.42)
	bCBase := max(0, (27.13*num.Abs(bA))/(400-num.Abs(bA)))
	bC := math32.Sign(bA) *
		(100 / vw.FL) *
		math32.Pow(bCBase, 1/0.42)
	rF := rC / vw.RGBD.X
	gF := gC / vw.RGBD.Y
	bF := bC / vw.RGBD.Z

	x = 1.86206786*rF - 1.01125463*gF + 0.14918677*bF
	y = 0.38752654*rF + 0.62144744*gF - 0.00897398*bF
	z = -0.01584150*rF - 0.03412294*gF + 1.04996444*bF
	return
}
