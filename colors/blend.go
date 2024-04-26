// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"
	"log/slog"

	"cogentcore.org/core/colors/cam/cam16"
	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/math32"
)

// BlendTypes are different algorithms (colorspaces) to use for blending
// the color stop values in generating the gradients.
type BlendTypes int32 //enums:enum

const (
	// HCT uses the hue, chroma, and tone space and generally produces the best results,
	// but at a slight performance cost.
	HCT BlendTypes = iota

	// RGB uses raw RGB space, which is the standard space that most other programs use.
	// It produces decent results with maximum performance.
	RGB

	// CAM16 is an alternative colorspace, similar to HCT, but not quite as good.
	CAM16
)

// Blend returns a color that is the given proportion between the first
// and second color. For example, 0.1 indicates to blend 10% of the first
// color and 90% of the second. Blending is done using the given blending
// algorithm.
func Blend(bt BlendTypes, p float32, x, y color.Color) color.RGBA {
	switch bt {
	case HCT:
		return hct.Blend(p, x, y)
	case RGB:
		return BlendRGB(p, x, y)
	case CAM16:
		return cam16.Blend(p, x, y)
	}
	slog.Error("got unexpected blend type", "type", bt)
	return color.RGBA{}
}

// BlendRGB returns a color that is the given proportion between the first
// and second color in RGB colorspace. For example, 0.1 indicates to blend
// 10% of the first color and 90% of the second. Blending is done directly
// on non-premultiplied
// RGB values, and a correctly premultiplied color is returned.
func BlendRGB(pct float32, x, y color.Color) color.RGBA {
	fx := NRGBAF32Model.Convert(x).(NRGBAF32)
	fy := NRGBAF32Model.Convert(y).(NRGBAF32)
	pct = math32.Clamp(pct, 0, 100.0)
	px := pct / 100
	py := 1.0 - px
	fx.R = px*fx.R + py*fy.R
	fx.G = px*fx.G + py*fy.G
	fx.B = px*fx.B + py*fy.B
	fx.A = px*fx.A + py*fy.A
	return AsRGBA(fx)
}

// m is the maximum color value returned by [image.Color.RGBA]
const m = 1<<16 - 1

// AlphaBlend blends the two colors, handling alpha blending correctly.
// The source color is figuratively placed "on top of" the destination color.
func AlphaBlend(dst, src color.Color) color.RGBA {
	res := color.RGBA{}

	dr, dg, db, da := dst.RGBA()
	sr, sg, sb, sa := src.RGBA()
	a := (m - sa)

	res.R = uint8((uint32(dr)*a/m + sr) >> 8)
	res.G = uint8((uint32(dg)*a/m + sg) >> 8)
	res.B = uint8((uint32(db)*a/m + sb) >> 8)
	res.A = uint8((uint32(da)*a/m + sa) >> 8)
	return res
}
