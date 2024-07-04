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
	"fmt"
	"image/color"

	"cogentcore.org/core/colors/cam/cam16"
	"cogentcore.org/core/colors/cam/cie"
)

// HCT represents a color as hue, chroma, and tone. HCT is a color system
// that provides a perceptually accurate color measurement system that can
// also accurately render what colors will appear as in different lighting
// environments. Directly setting the values of the HCT and RGB fields will
// have no effect on the underlying color; instead, use the Set methods
// ([HCT.SetHue], etc). The A field (transparency) can be set directly.
type HCT struct {

	// Hue (h) is the spectral identity of the color
	// (red, green, blue etc) in degrees (0-360)
	Hue float32 `min:"0" max:"360"`

	// Chroma (C) is the colorfulness/saturation of the color.
	// Grayscale colors have no chroma, and fully saturated ones
	// have high chroma. The maximum varies as a function of hue
	// and tone, but 120 is a general upper bound (see
	// [HCT.MaximumChroma] to get a specific value).
	Chroma float32 `min:"0" max:"120"`

	// Tone is the L* component from the LAB (L*a*b*) color system,
	// which is linear in human perception of lightness.
	// It ranges from 0 to 100.
	Tone float32 `min:"0" max:"100"`

	// sRGB standard gamma-corrected 0-1 normalized RGB representation
	// of the color. Critically, components are not premultiplied by alpha.
	R, G, B, A float32
}

// New returns a new HCT representation for given parameters:
// hue = 0..360
// chroma = 0..? depends on other params
// tone = 0..100
// also computes and sets the sRGB normalized, gamma corrected RGB values
// while keeping the sRGB representation within its gamut,
// which may cause the chroma to decrease until it is inside the gamut.
func New(hue, chroma, tone float32) HCT {
	r, g, b := SolveToRGB(hue, chroma, tone)
	return SRGBToHCT(r, g, b)
}

// FromColor constructs a new HCT color from a standard [color.Color].
func FromColor(c color.Color) HCT {
	return Uint32ToHCT(c.RGBA())
}

// SetColor sets the HCT color from a standard [color.Color].
func (h *HCT) SetColor(c color.Color) {
	*h = FromColor(c)
}

// Model is the standard [color.Model] that converts colors to HCT.
var Model = color.ModelFunc(model)

func model(c color.Color) color.Color {
	if h, ok := c.(HCT); ok {
		return h
	}
	return FromColor(c)
}

// RGBA implements the color.Color interface.
// Performs the premultiplication of the RGB components by alpha at this point.
func (h HCT) RGBA() (r, g, b, a uint32) {
	return cie.SRGBFloatToUint32(h.R, h.G, h.B, h.A)
}

// AsRGBA returns a standard color.RGBA type
func (h HCT) AsRGBA() color.RGBA {
	r, g, b, a := cie.SRGBFloatToUint8(h.R, h.G, h.B, h.A)
	return color.RGBA{r, g, b, a}
}

// SetUint32 sets components from unsigned 32bit integers (alpha-premultiplied)
func (h *HCT) SetUint32(r, g, b, a uint32) {
	fr, fg, fb, fa := cie.SRGBUint32ToFloat(r, g, b, a)
	*h = SRGBAToHCT(fr, fg, fb, fa)
}

// SetHue sets the hue of this color. Chroma may decrease because chroma has a
// different maximum for any given hue and tone.
// 0 <= hue < 360; invalid values are corrected.
func (h *HCT) SetHue(hue float32) *HCT {
	r, g, b := SolveToRGB(hue, h.Chroma, h.Tone)
	*h = SRGBAToHCT(r, g, b, h.A)
	return h
}

// WithHue is like [SetHue] except it returns a new color
// instead of setting the existing one.
func (h HCT) WithHue(hue float32) HCT {
	r, g, b := SolveToRGB(hue, h.Chroma, h.Tone)
	return SRGBAToHCT(r, g, b, h.A)
}

// SetChroma sets the chroma of this color (0 to max that depends on other params),
// while keeping the sRGB representation within its gamut,
// which may cause the chroma to decrease until it is inside the gamut.
func (h *HCT) SetChroma(chroma float32) *HCT {
	r, g, b := SolveToRGB(h.Hue, chroma, h.Tone)
	*h = SRGBAToHCT(r, g, b, h.A)
	return h
}

// WithChroma is like [SetChroma] except it returns a new color
// instead of setting the existing one.
func (h HCT) WithChroma(chroma float32) HCT {
	r, g, b := SolveToRGB(h.Hue, chroma, h.Tone)
	return SRGBAToHCT(r, g, b, h.A)
}

// SetTone sets the tone of this color (0 < tone < 100),
// while keeping the sRGB representation within its gamut,
// which may cause the chroma to decrease until it is inside the gamut.
func (h *HCT) SetTone(tone float32) *HCT {
	r, g, b := SolveToRGB(h.Hue, h.Chroma, tone)
	*h = SRGBAToHCT(r, g, b, h.A)
	return h
}

// WithTone is like [SetTone] except it returns a new color
// instead of setting the existing one.
func (h HCT) WithTone(tone float32) HCT {
	r, g, b := SolveToRGB(h.Hue, h.Chroma, tone)
	return SRGBAToHCT(r, g, b, h.A)
}

// MaximumChroma returns the maximum [HCT.Chroma] value for the hue
// and tone of this color. This will always be between 0 and 120.
func (h HCT) MaximumChroma() float32 {
	// WithChroma does a round trip, so the resultant chroma will only
	// be as high as the maximum chroma.
	return h.WithChroma(120).Chroma
}

// SRGBAToHCT returns an HCT from the given SRGBA color coordinates
// under standard viewing conditions. The RGB value range is 0-1,
// and RGB values have gamma correction. The RGB values must not be
// premultiplied by the given alpha value. See [SRGBToHCT] for
// a version that does not take the alpha value.
func SRGBAToHCT(r, g, b, a float32) HCT {
	h := SRGBToHCT(r, g, b)
	h.A = a
	return h
}

// SRGBToHCT returns an HCT from the given SRGB color coordinates
// under standard viewing conditions. The RGB value range is 0-1,
// and RGB values have gamma correction. Alpha is always 1; see
// [SRGBAToHCT] for a version that takes the alpha value.
func SRGBToHCT(r, g, b float32) HCT {
	x, y, z := cie.SRGBToXYZ(r, g, b)
	cam := cam16.FromXYZ(100*x, 100*y, 100*z)
	l, _, _ := cie.XYZToLAB(x, y, z)
	return HCT{Hue: cam.Hue, Chroma: cam.Chroma, Tone: l, R: r, G: g, B: b, A: 1}
}

// Uint32ToHCT returns an HCT from given SRGBA uint32 color coordinates,
// which are used for interchange among image.Color types.
// Uses standard viewing conditions, and RGB values already have gamma correction
// (i.e., they are SRGB values).
func Uint32ToHCT(r, g, b, a uint32) HCT {
	h := HCT{}
	h.SetUint32(r, g, b, a)
	return h
}

func (h HCT) String() string {
	return fmt.Sprintf("hct(%g, %g, %g)", h.Hue, h.Chroma, h.Tone)
}

/*
  // Translate a color into different [ViewingConditions].
  //
  // Colors change appearance. They look different with lights on versus off,
  // the same color, as in hex code, on white looks different when on black.
  // This is called color relativity, most famously explicated by Josef Albers
  // in Interaction of Color.
  //
  // In color science, color appearance models can account for this and
  // calculate the appearance of a color in different settings. HCT is based on
  // CAM16, a color appearance model, and uses it to make these calculations.
  //
  // See [ViewingConditions.make] for parameters affecting color appearance.
  Hct inViewingConditions(ViewingConditions vc) {
    // 1. Use CAM16 to find XYZ coordinates of color in specified VC.
    final cam16 = Cam16.fromInt(toInt());
    final viewedInVc = cam16.xyzInViewingConditions(vc);

    // 2. Create CAM16 of those XYZ coordinates in default VC.
    final recastInVc = Cam16.fromXyzInViewingConditions(
      viewedInVc[0],
      viewedInVc[1],
      viewedInVc[2],
      ViewingConditions.make(),
    );

    // 3. Create HCT from:
    // - CAM16 using default VC with XYZ coordinates in specified VC.
    // - L* converted from Y in XYZ coordinates in specified VC.
    final recastHct = Hct.from(
      recastInVc.hue,
      recastInVc.chroma,
      ColorUtils.lstarFromY(viewedInVc[1]),
    );
    return recastHct;
  }
}

*/
