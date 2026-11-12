// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

import (
	"image"
	"image/color"
	"image/draw"

	"cogentcore.org/core/colors/cam/cie"
	"cogentcore.org/core/gpu/drawmatrix"
	"cogentcore.org/core/math32"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// Fill fills given color to render target, to given destination region dr.
//   - op is the drawing operation: Src = copy source directly (blit),
//     Over = alpha blend with existing
func (dw *Drawer) Fill(clr color.Color, dr image.Rectangle, op draw.Op) {
	dsz := dw.DestSize()
	tmat := drawmatrix.Config(dsz, math32.Identity3(), dr.Max, dr, false)
	clr4 := math32.NewVector4Color(clr)
	if dw.destIsSRGB() {
		clr4 = linearColor(clr4)
	}
	clr4.ToSlice(tmat.UVP[:], 12) // last column holds color
	if op == Over {
		dw.addOp(fillOver, tmat)
	} else {
		dw.addOp(fillSrc, tmat)
	}
}

// destIsSRGB returns whether the render destination has an sRGB format,
// which encodes the linear colors of the shaders as sRGB, as surfaces do.
func (dw *Drawer) destIsSRGB() bool {
	switch dw.System.Renderer.Render().Format.Format {
	case wgpu.TextureFormatRGBA8UnormSrgb, wgpu.TextureFormatBGRA8UnormSrgb:
		return true
	}
	return false
}

// linearColor returns the given (premultiplied) sRGB color in linear
// space, for a fill of an sRGB destination, which encodes it back to the
// given color (which would otherwise be lighter: 18 becomes 75).
func linearColor(c math32.Vector4) math32.Vector4 {
	if c.W <= 0 {
		return c
	}
	lin := func(v float32) float32 {
		return cie.SRGBToLinearComp(v/c.W) * c.W
	}
	return math32.Vec4(lin(c.X), lin(c.Y), lin(c.Z), c.W)
}
