// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// adapted from: https://github.com/material-foundation/material-color-utilities
// Copyright 2022 Google LLC
// Licensed under the Apache License, Version 2.0 (the "License")

package hct

import (
	"image/color"

	"github.com/goki/cam/cam16"
	"github.com/goki/cam/cie"
)

// HCT, hue, chroma, and tone. A color system that provides a perceptually
// accurate color measurement system that can also accurately render what
// colors will appear as in different lighting environments.
type HCT struct {

	// [min: 0] [max: 360] hue (h) is the spectral identity of the color (red, green, blue etc) in degrees (0-360)
	Hue float32 `min:"0" max:"360" desc:"hue (h) is the spectral identity of the color (red, green, blue etc) in degrees (0-360)"`

	// [min: 0] [max: 150] chroma (C) is the colorfulness or saturation of the color -- greyscale colors have no chroma, and fully saturated ones have high chroma.  The maximum varies as a function of hue and tone, but 150 is an upper bound.
	Chroma float32 `min:"0" max:"150" desc:"chroma (C) is the colorfulness or saturation of the color -- greyscale colors have no chroma, and fully saturated ones have high chroma.  The maximum varies as a function of hue and tone, but 150 is an upper bound."`

	// [min: 0] [max: 100] tone is the L* component from the LAB (L*a*b*) color system, which is linear in human perception of lightness
	Tone float32 `min:"0" max:"100" desc:"tone is the L* component from the LAB (L*a*b*) color system, which is linear in human perception of lightness"`

	// sRGB standard gamma-corrected 0-1 normalized RGB representation of the color.  Critically, components are not premultiplied by alpha
	R, G, B, A float32 `desc:"sRGB standard gamma-corrected 0-1 normalized RGB representation of the color.  Critically, components are not premultiplied by alpha"`
}

// NewHCT returns a new HCT representation for given parameters:
// hue = 0..360
// chroma = 0..? depends on other params
// tone = 0..100
// also computes and sets the sRGB normalized, gamma corrected R,G,B values
// while keeping the sRGB representation within its gamut,
// which may cause the chroma to decrease until it is inside the gamut.
func NewHCT(hue, chroma, tone float32) HCT {
	r, g, b := SolveToRGB(hue, chroma, tone)
	return SRGBToHCT(r, g, b)
}

// RGBA implements the color.Color interface.
// Performs the premultiplication of the RGB components by alpha at this point.
func (h *HCT) RGBA() (r, g, b, a uint32) {
	r = uint32(h.R*h.A*65535.0 + 0.5)
	g = uint32(h.G*h.A*65535.0 + 0.5)
	b = uint32(h.B*h.A*65535.0 + 0.5)
	a = uint32(h.A*65535.0 + 0.5)
	return
}

// AsRGBA returns a standard color.RGBA type
func (h *HCT) AsRGBA() color.RGBA {
	return color.RGBA{uint8(h.R*h.A*255.0 + 0.5), uint8(h.G*h.A*255.0 + 0.5), uint8(h.B*h.A*255.0 + 0.5), uint8(h.A*255.0 + 0.5)}
}

// SetUint32 sets components from unsigned 32bit integers (alpha-premultiplied)
func (h *HCT) SetUint32(r, g, b, a uint32) {
	h.A = float32(a) / 65535.0
	h.R = (float32(r) / 65535.0) / h.A
	h.G = (float32(g) / 65535.0) / h.A
	h.B = (float32(b) / 65535.0) / h.A
}

// SetColor sets from a standard color.Color
func (h *HCT) SetColor(ci color.Color) {
	if ci == nil {
		h.SetToNil()
		return
	}
	r, g, b, a := ci.RGBA()
	h.SetUint32(r, g, b, a)
}

func (h *HCT) SetToNil() {
	*h = SRGBToHCT(0, 0, 0)
	h.A = 0
}

// SetHue sets the hue of this color. Chroma may decrease because chroma has a
// different maximum for any given hue and tone.
// 0 <= hue < 360; invalid values are corrected.
func (h *HCT) SetHue(hue float32) {
	r, g, b := SolveToRGB(hue, h.Chroma, h.Tone)
	*h = SRGBToHCT(r, g, b)
}

// SetChroma sets the chroma of this color (0 to max that depends on other params),
// while keeping the sRGB representation within its gamut,
// which may cause the chroma to decrease until it is inside the gamut.
func (h *HCT) SetChroma(chroma float32) {
	r, g, b := SolveToRGB(h.Hue, chroma, h.Tone)
	*h = SRGBToHCT(r, g, b)
}

// SetTone sets the tone of this color (0 < tone < 100),
// while keeping the sRGB representation within its gamut,
// which may cause the chroma to decrease until it is inside the gamut.
func (h *HCT) SetTone(tone float32) {
	r, g, b := SolveToRGB(h.Hue, h.Chroma, tone)
	*h = SRGBToHCT(r, g, b)
}

// SRGBToHCT returns an HCT from given SRGB color coordinates,
// under standard viewing conditions.  The RGB value range is 0-1,
// and RGB values have gamma correction.  Alpha is always 1.
func SRGBToHCT(r, g, b float32) HCT {
	x, y, z := cie.SRGBToXYZ(r, g, b)
	cam := cam16.XYZToCAM(100*x, 100*y, 100*z)
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
