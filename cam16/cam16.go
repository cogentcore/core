// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// adapted from: https://github.com/material-foundation/material-color-utilities
// Copyright 2022 Google LLC
// Licensed under the Apache License, Version 2.0 (the "License")

package cam16

import (
	"github.com/goki/cam/cie"
	"github.com/goki/mat32"
)

// CAM represents a point in the cam16 color model along 6 dimensions
// representing the perceived hue, colorfulness, and brightness,
// similar to HSL but much more well-calibrated to actual human subjective judgments.
type CAM struct {

	// hue (h) is the spectral identity of the color (red, green, blue etc) in degrees (0-360)
	Hue float32 `desc:"hue (h) is the spectral identity of the color (red, green, blue etc) in degrees (0-360)"`

	// chroma (C) is the colorfulness or saturation of the color -- greyscale colors have no chroma, and fully saturated ones have high chroma
	Chroma float32 `desc:"chroma (C) is the colorfulness or saturation of the color -- greyscale colors have no chroma, and fully saturated ones have high chroma"`

	// colorfulness (M) is the absolute chromatic intensity
	Colorfulness float32 `desc:"colorfulness (M) is the absolute chromatic intensity"`

	// saturation (s) is the colorfulness relative to brightness
	Saturation float32 `desc:"saturation (s) is the colorfulness relative to brightness"`

	// brightness (Q) is the apparent amount of light from the color, which is not a simple function of actual light energy emitted
	Brightness float32 `desc:"brightness (Q) is the apparent amount of light from the color, which is not a simple function of actual light energy emitted"`

	// lightness (J) is the brightness relative to a reference white, which varies as a function of chroma and hue
	Lightness float32 `desc:"lightness (J) is the brightness relative to a reference white, which varies as a function of chroma and hue"`
}

// UCS returns the CAM16-UCS components based on the the CAM values
func (cam *CAM) UCS() (j, m, a, b float32) {
	j = (1 + 100*0.007) * cam.Lightness / (1 + 0.007*cam.Lightness)
	m = mat32.Log(1+0.0228*cam.Colorfulness) / 0.0228
	hr := mat32.DegToRad(cam.Hue)
	a = m * mat32.Cos(hr)
	b = m * mat32.Sin(hr)
	return
}

// SRGBToCAM returns CAM values from given SRGB color coordinates,
// under standard viewing conditions.  The RGB value range is 0-1,
// and RGB values have gamma correction.
func SRGBToCAM(r, g, b float32) *CAM {
	return XYZToCAM(cie.SRGB100ToXYZ(r, g, b))
}

// XYZToCAM returns CAM values from given XYZ color coordinate,
// under standard viewing conditions
func XYZToCAM(x, y, z float32) *CAM {
	return XYZToCAMView(x, y, z, NewStdView())
}

// XYZToCAMView returns CAM values from given XYZ color coordinate,
// under given viewing conditions.  Requires 100-base XYZ coordinates.
func XYZToCAMView(x, y, z float32, vw *View) *CAM {
	l, m, s := XYZToLMS(x, y, z)
	redVgreen, yellowVblue, grey, greyNorm := LMSToOps(l, m, s, vw)

	hue := SanitizeDeg(mat32.RadToDeg(mat32.Atan2(yellowVblue, redVgreen)))
	// achromatic response to color
	ac := grey * vw.NBB

	// CAM16 lightness and brightness
	J := 100 * mat32.Pow(ac/vw.AW, vw.C*vw.Z)
	Q := (4 / vw.C) * mat32.Sqrt(J/100) * (vw.AW + 4) * (vw.FLRoot)

	huePrime := hue
	if hue < 20.14 {
		huePrime += 360
	}
	eHue := 0.25 * (mat32.Cos(huePrime*mat32.Pi/180+2) + 3.8)
	p1 := 50000 / 13 * eHue * vw.NC * vw.NCB
	t := p1 * mat32.Sqrt(redVgreen*redVgreen+yellowVblue*yellowVblue) / (greyNorm + 0.305)
	alpha := mat32.Pow(t, 0.9) * mat32.Pow(1.64-mat32.Pow(0.29, vw.BgYToWhiteY), 0.73)

	// CAM16 chroma, colorfulness, chroma
	C := alpha * mat32.Sqrt(J/100)
	M := C * vw.FLRoot
	s = 50 * mat32.Sqrt((alpha*vw.C)/(vw.AW+4))
	return &CAM{Hue: hue, Chroma: C, Colorfulness: M, Saturation: s, Brightness: Q, Lightness: J}
}
