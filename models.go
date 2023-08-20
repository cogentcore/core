// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import "image/color"

// RGBAF32 stores alpha-premultiplied RGBA values in a float32 0 to 1
// normalized format, which is more useful for converting to other spaces
type RGBAF32 struct {
	R, G, B, A float32
}

// RGBA implements the color.Color interface
func (c RGBAF32) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R*65535.0 + 0.5)
	g = uint32(c.G*65535.0 + 0.5)
	b = uint32(c.B*65535.0 + 0.5)
	a = uint32(c.A*65535.0 + 0.5)
	return
}

// FromRGBAF32 returns the color specified by the given float32
// alpha-premultiplied RGBA values in the range 0 to 1
func FromRGBAF32(r, g, b, a float32) color.RGBA {
	return AsRGBA(RGBAF32{r, g, b, a})
}

// NRGBAF32 stores non-alpha-premultiplied RGBA values in a float32 0 to 1
// normalized format, which is more useful for converting to other spaces
type NRGBAF32 struct {
	R, G, B, A float32
}

// RGBA implements the color.Color interface
func (c NRGBAF32) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R*c.A*65535.0 + 0.5)
	g = uint32(c.G*c.A*65535.0 + 0.5)
	b = uint32(c.B*c.A*65535.0 + 0.5)
	a = uint32(c.A*65535.0 + 0.5)
	return
}

// FromNRGBAF32 returns the color specified by the given float32
// non alpha-premultiplied RGBA values in the range 0 to 1
func FromNRGBAF32(r, g, b, a float32) color.RGBA {
	return AsRGBA(NRGBAF32{r, g, b, a})
}

var (
	// RGBAF32Model is the model for converting colors to [RGBAF32] colors
	RGBAF32Model color.Model = color.ModelFunc(rgbaf32Model)
	// NRGBAF32Model is the model for converting colors to [NRGBAF32] colors
	NRGBAF32Model color.Model = color.ModelFunc(nrgbaf32Model)
)

func rgbaf32Model(c color.Color) color.Color {
	if _, ok := c.(RGBAF32); ok {
		return c
	}
	r, g, b, a := c.RGBA()
	return RGBAF32{float32(r) / 65535.0, float32(g) / 65535.0, float32(b) / 65535.0, float32(a) / 65535.0}
}

func nrgbaf32Model(c color.Color) color.Color {
	if _, ok := c.(NRGBAF32); ok {
		return c
	}
	r, g, b, a := c.RGBA()
	if a > 0 {
		// Since color.Color is alpha pre-multiplied, we need to divide the
		// RGB values by alpha again in order to get back the original RGB.
		r *= 0xffff
		r /= a
		g *= 0xffff
		g /= a
		b *= 0xffff
		b /= a
	}
	return NRGBAF32{float32(r) / 65535.0, float32(g) / 65535.0, float32(b) / 65535.0, float32(a) / 65535.0}
}
