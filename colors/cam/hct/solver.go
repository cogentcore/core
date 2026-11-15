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

package hct

import (
	"cogentcore.org/core/colors/cam/cam16"
	"cogentcore.org/core/colors/cam/cie"
	"cogentcore.org/core/math32"
)

// SolveToRGBLin Finds an sRGB linear color (represented by math32.Vector3, 0-100 range)
// with the given hue, chroma, and tone, if possible.
// if not possible to represent the target values, the hue and tone will be
// sufficiently close, and chroma will be maximized.
func SolveToRGBLin(hue, chroma, tone float32) math32.Vector3 {
	if chroma < 0.0001 || tone < 0.0001 || tone > 99.9999 {
		y := cie.LToY(tone)
		return math32.Vec3(y, y, y)
	}
	tone = math32.Clamp(tone, 0, 100)
	hueDeg := cam16.SanitizeDegrees(hue)
	hueRad := math32.DegToRad(hueDeg)
	y := cie.LToY(tone)
	exact := FindResultByJ(hueRad, chroma, y)
	if exact != nil {
		return *exact
	}
	return BisectToLimit(y, hueRad)
}

// SolveToRGB Finds an sRGB (gamma corrected, 0-1 range) color
// with the given hue, chroma, and tone, if possible.
// if not possible to represent the target values, the hue and tone will be
// sufficiently close, and chroma will be maximized.
func SolveToRGB(hue, chroma, tone float32) (r, g, b float32) {
	lin := SolveToRGBLin(hue, chroma, tone)
	r, g, b = cie.SRGBFromLinear100(lin.X, lin.Y, lin.Z)
	return
}

// FindResultByJ finds a color with the given hue, chroma, and Y.
// @param hue_radians The desired hue in radians.
// @param chroma The desired chroma.
// @param y The desired Y.
// @return The desired color as linear sRGB values.
func FindResultByJ(hueRad, chroma, y float32) *math32.Vector3 {
	// Initial estimate of j.
	j := math32.Sqrt(y) * 11

	// ===========================================================
	// Operations inlined from Cam16 to avoid repeated calculation
	// ===========================================================
	vw := cam16.NewStdView()
	tInnerCoeff := 1 / math32.Pow(1.64-math32.Pow(0.29, vw.BgYToWhiteY), 0.73)
	eHue := 0.25 * (math32.Cos(hueRad+2) + 3.8)
	p1 := eHue * (50000 / 13) * vw.NC * vw.NCB
	hSin := math32.Sin(hueRad)
	hCos := math32.Cos(hueRad)
	for itr := range 5 {
		jNorm := j / 100
		alpha := float32(0)
		if !(chroma == 0 || j == 0) {
			alpha = chroma / math32.Sqrt(jNorm)
		}
		t := math32.Pow(alpha*tInnerCoeff, 1/0.9)
		ac := vw.AW * math32.Pow(jNorm, 1/vw.C/vw.Z)
		p2 := ac / vw.NBB
		gamma := 23 * (p2 + 0.305) * t / (23*p1 + 11*t*hCos + 108*t*hSin)
		a := gamma * hCos
		b := gamma * hSin
		rA := (460*p2 + 451*a + 288*b) / 1403
		gA := (460*p2 - 891*a - 261*b) / 1403
		bA := (460*p2 - 220*a - 6300*b) / 1403
		rCScaled := cam16.InverseChromaticAdapt(rA)
		gCScaled := cam16.InverseChromaticAdapt(gA)
		bCScaled := cam16.InverseChromaticAdapt(bA)
		scaled := math32.Vec3(rCScaled, gCScaled, bCScaled)
		linrgb := MatMul(scaled, kLinrgbFromScaledDiscount)

		if linrgb.X < 0 || linrgb.Y < 0 || linrgb.Z < 0 {
			return nil
		}
		kR := kYFromLinrgb[0]
		kG := kYFromLinrgb[1]
		kB := kYFromLinrgb[2]
		fnj := kR*linrgb.X + kG*linrgb.Y + kB*linrgb.Z
		if fnj <= 0 {
			return nil
		}
		if itr == 4 || math32.Abs(fnj-y) < 0.002 {
			if linrgb.X > 100.01 || linrgb.Y > 100.01 || linrgb.Z > 100.01 {
				return nil
			}
			return &linrgb
		}
		// Iterates with Newton method,
		// Using 2 * fn(j) / j as the approximation of fn'(j)
		j = j - (fnj-y)*j/(2*fnj)
	}
	return nil
}
