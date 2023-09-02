// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hsl

import (
	"fmt"
	"image/color"

	"goki.dev/mat32/v2"
)

// HSL represents the Hue [0..360], Saturation [0..1], and Luminance
// (lightness) [0..1] of the color using float32 values
// In general the Alpha channel is not used for HSL but is maintained
// so it can be used to fully represent an RGBA color value.
// When converting to RGBA, alpha multiplies the RGB components.
type HSL struct {

	// [min: 0] [max: 360] [step: 5] the hue of the color
	H float32 `min:"0" max:"360" step:"5" desc:"the hue of the color"`

	// [min: 0] [max: 1] [step: 0.05] the saturation of the color
	S float32 `min:"0" max:"1" step:"0.05" desc:"the saturation of the color"`

	// [min: 0] [max: 1] [step: 0.05] the luminance (lightness) of the color
	L float32 `min:"0" max:"1" step:"0.05" desc:"the luminance (lightness) of the color"`

	// [min: 0] [max: 1] [step: 0.05] the transparency of the color
	A float32 `min:"0" max:"1" step:"0.05" desc:"the transparency of the color"`
}

// New returns a new HSL representation for given parameters:
// hue = 0..360
// saturation = 0..1
// lightness = 0..1
// A is automatically set to 1
func New(hue, saturation, lightness float32) HSL {
	return HSL{hue, saturation, lightness, 1}
}

// FromColor constructs a new HSL color from a standard [color.Color]
func FromColor(c color.Color) HSL {
	h := HSL{}
	h.SetColor(c)
	return h
}

// Model is the standard [color.Model] that converts colors to HSL.
var Model = color.ModelFunc(model)

func model(c color.Color) color.Color {
	if h, ok := c.(HSL); ok {
		return h
	}
	return FromColor(c)
}

// Implements the [color.Color] interface
// Performs the premultiplication of the RGB components by alpha at this point.
func (h HSL) RGBA() (r, g, b, a uint32) {
	fr, fg, fb := HSLtoRGBf32(h.H, h.S, h.L)
	r = uint32(fr*h.A*65535.0 + 0.5)
	g = uint32(fg*h.A*65535.0 + 0.5)
	b = uint32(fb*h.A*65535.0 + 0.5)
	a = uint32(h.A*65535.0 + 0.5)
	return
}

// AsRGBA returns a standard color.RGBA type
func (h HSL) AsRGBA() color.RGBA {
	fr, fg, fb := HSLtoRGBf32(h.H, h.S, h.L)
	return color.RGBA{uint8(fr*h.A*255.0 + 0.5), uint8(fg*h.A*255.0 + 0.5), uint8(fb*h.A*255.0 + 0.5), uint8(h.A*255.0 + 0.5)}
}

// SetUint32 sets components from unsigned 32bit integers (alpha-premultiplied)
func (h *HSL) SetUint32(r, g, b, a uint32) {
	fa := float32(a) / 65535.0
	fr := (float32(r) / 65535.0) / fa
	fg := (float32(g) / 65535.0) / fa
	fb := (float32(b) / 65535.0) / fa

	h.H, h.S, h.L = RGBtoHSLf32(fr, fg, fb)
	h.A = fa
}

// SetColor sets from a standard color.Color
func (h *HSL) SetColor(ci color.Color) {
	if ci == nil {
		h.SetToNil()
		return
	}
	r, g, b, a := ci.RGBA()
	h.SetUint32(r, g, b, a)
}

func (h *HSL) SetToNil() {
	h.H = 0
	h.S = 0
	h.L = 0
	h.A = 0
}

// Round rounds the HSL values (H to the nearest 1
// and S, L, and A to the nearest 0.01)
func (h *HSL) Round() {
	h.H = mat32.Round(h.H)
	h.S = mat32.Round(h.S*100) / 100
	h.L = mat32.Round(h.L*100) / 100
	h.A = mat32.Round(h.A*100) / 100
}

// HSLtoRGBf32 converts HSL values to RGB float32 0..1 values (non alpha-premultiplied) -- based on https://stackoverflow.com/questions/2353211/hsl-to-rgb-color-conversion, https://www.w3.org/TR/css-color-3/ and github.com/lucasb-eyer/go-colorful
func HSLtoRGBf32(h, s, l float32) (r, g, b float32) {
	if s == 0 {
		r = l
		g = l
		b = l
		return
	}

	h = h / 360.0 // convert to normalized 0-1 h
	var q float32
	if l < 0.5 {
		q = l * (1.0 + s)
	} else {
		q = l + s - l*s
	}
	p := 2.0*l - q
	r = HueToRGBf32(p, q, h+1.0/3.0)
	g = HueToRGBf32(p, q, h)
	b = HueToRGBf32(p, q, h-1.0/3.0)
	return
}

func HueToRGBf32(p, q, t float32) float32 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6.0*t
	}
	if t < .5 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6.0
	}
	return p
}

// RGBtoHSLf32 converts RGB 0..1 values (non alpha-premultiplied) to HSL -- based on https://stackoverflow.com/questions/2353211/hsl-to-rgb-color-conversion, https://www.w3.org/TR/css-color-3/ and github.com/lucasb-eyer/go-colorful
func RGBtoHSLf32(r, g, b float32) (h, s, l float32) {
	min := mat32.Min(mat32.Min(r, g), b)
	max := mat32.Max(mat32.Max(r, g), b)

	l = (max + min) / 2.0

	if min == max {
		s = 0
		h = 0
	} else {
		d := max - min
		if l > 0.5 {
			s = d / (2.0 - max - min)
		} else {
			s = d / (max + min)
		}
		switch max {
		case r:
			h = (g - b) / d
			if g < b {
				h += 6.0
			}
		case g:
			h = 2.0 + (b-r)/d
		case b:
			h = 4.0 + (r-g)/d
		}

		h *= 60

		if h < 0 {
			h += 360
		}
	}
	return
}

func (h *HSL) String() string {
	return fmt.Sprintf("hsl(%g, %g, %g)", h.H, h.S, h.L)
}
