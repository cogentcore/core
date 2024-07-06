// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package colors provides named colors, utilities for manipulating colors,
// and Material Design 3 color schemes, palettes, and keys in Go.
package colors

//go:generate core generate

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"

	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/colors/cam/hsl"
	"cogentcore.org/core/math32"
)

// IsNil returns whether the color is the nil initial default color
func IsNil(c color.Color) bool {
	return AsRGBA(c) == color.RGBA{}
}

// FromRGB makes a new RGBA color from the given
// RGB uint8 values, using 255 for A.
func FromRGB(r, g, b uint8) color.RGBA {
	return color.RGBA{r, g, b, 255}
}

// FromNRGBA makes a new RGBA color from the given
// non-alpha-premultiplied RGBA uint8 values.
func FromNRGBA(r, g, b, a uint8) color.RGBA {
	return AsRGBA(color.NRGBA{r, g, b, a})
}

// AsRGBA returns the given color as an RGBA color
func AsRGBA(c color.Color) color.RGBA {
	if c == nil {
		return color.RGBA{}
	}
	return color.RGBAModel.Convert(c).(color.RGBA)
}

// FromFloat64 makes a new RGBA color from the given 0-1
// normalized floating point numbers (alpha-premultiplied)
func FromFloat64(r, g, b, a float64) color.RGBA {
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), uint8(a * 255)}
}

// FromFloat32 makes a new RGBA color from the given 0-1
// normalized floating point numbers (alpha-premultiplied)
func FromFloat32(r, g, b, a float32) color.RGBA {
	return color.RGBA{uint8(r * 255), uint8(g * 255), uint8(b * 255), uint8(a * 255)}
}

// ToFloat32 returns 0-1 normalized floating point numbers from given color
// (alpha-premultiplied)
func ToFloat32(c color.Color) (r, g, b, a float32) {
	f := NRGBAF32Model.Convert(c).(NRGBAF32)
	r = f.R
	g = f.G
	b = f.B
	a = f.A
	return
}

// ToFloat64 returns 0-1 normalized floating point numbers from given color
// (alpha-premultiplied)
func ToFloat64(c color.Color) (r, g, b, a float64) {
	f := NRGBAF32Model.Convert(c).(NRGBAF32)
	r = float64(f.R)
	g = float64(f.G)
	b = float64(f.B)
	a = float64(f.A)
	return
}

// AsString returns the given color as a string,
// using its String method if it exists, and formatting
// it as rgba(r, g, b, a) otherwise.
func AsString(c color.Color) string {
	if s, ok := c.(fmt.Stringer); ok {
		return s.String()
	}
	r := AsRGBA(c)
	return fmt.Sprintf("rgba(%d, %d, %d, %d)", r.R, r.G, r.B, r.A)
}

// FromName returns the color value specified
// by the given CSS standard color name.
func FromName(name string) (color.RGBA, error) {
	c, ok := Map[name]
	if !ok {
		return color.RGBA{}, errors.New("colors.FromName: name not found: " + name)
	}
	return c, nil
}

// FromString returns a color value from the given string.
// FromString accepts the following types of strings: standard
// color names, hex, rgb, rgba, hsl, hsla, hct, and hcta values,
// "none" or "off", or any of the transformations listed below.
// The transformations use the given single base color as their starting
// point; if you do not provide a base color, they will use [Transparent]
// as their starting point. The transformations are:
//
//   - currentcolor = base color
//   - inverse = inverse of base color
//   - lighten-VAL or darken-VAL: VAL is amount to lighten or darken (using HCT), e.g., lighter-10 is 10 higher tone
//   - saturate-VAL or desaturate-VAL: manipulates the chroma level in HCT by VAL
//   - spin-VAL: manipulates the hue level in HCT by VAL
//   - clearer-VAL or opaquer-VAL: manipulates the alpha level by VAL
//   - blend-VAL-color: blends given percent of given color relative to base in RGB space
func FromString(str string, base ...color.Color) (color.RGBA, error) {
	if len(str) == 0 { // consider it null
		return color.RGBA{}, nil
	}
	lstr := strings.ToLower(str)
	switch {
	case lstr[0] == '#':
		return FromHex(str)
	case strings.HasPrefix(lstr, "rgb("), strings.HasPrefix(lstr, "rgba("):
		val := lstr[strings.Index(lstr, "(")+1:]
		val = strings.TrimRight(val, ")")
		val = strings.Trim(val, "%")
		var r, g, b, a int
		a = 255
		if strings.Count(val, ",") == 3 {
			format := "%d,%d,%d,%d"
			fmt.Sscanf(val, format, &r, &g, &b, &a)
		} else {
			format := "%d,%d,%d"
			fmt.Sscanf(val, format, &r, &g, &b)
		}
		return FromNRGBA(uint8(r), uint8(g), uint8(b), uint8(a)), nil
	case strings.HasPrefix(lstr, "hsl("), strings.HasPrefix(lstr, "hsla("):
		val := lstr[strings.Index(lstr, "(")+1:]
		val = strings.TrimRight(val, ")")
		val = strings.Trim(val, "%")
		var h, s, l, a int
		a = 255
		if strings.Count(val, ",") == 3 {
			format := "%d,%d,%d,%d"
			fmt.Sscanf(val, format, &h, &s, &l, &a)
		} else {
			format := "%d,%d,%d"
			fmt.Sscanf(val, format, &h, &s, &l)
		}
		return WithA(hsl.New(float32(h), float32(s)/100.0, float32(l)/100.0), uint8(a)), nil
	case strings.HasPrefix(lstr, "hct("), strings.HasPrefix(lstr, "hcta("):
		val := lstr[strings.Index(lstr, "(")+1:]
		val = strings.TrimRight(val, ")")
		val = strings.Trim(val, "%")
		var h, c, t, a int
		a = 255
		if strings.Count(val, ",") == 3 {
			format := "%d,%d,%d,%d"
			fmt.Sscanf(val, format, &h, &c, &t, &a)
		} else {
			format := "%d,%d,%d"
			fmt.Sscanf(val, format, &h, &c, &t)
		}
		return WithA(hct.New(float32(h), float32(c), float32(t)), uint8(a)), nil
	default:
		var bc color.Color = Transparent
		if len(base) > 0 {
			bc = base[0]
		}

		if hidx := strings.Index(lstr, "-"); hidx > 0 {
			cmd := lstr[:hidx]
			valstr := lstr[hidx+1:]
			val64, err := strconv.ParseFloat(valstr, 32)
			if err != nil && cmd != "blend" { // blend handles separately
				return color.RGBA{}, fmt.Errorf("colors.FromString: error getting numeric value from %q: %w", valstr, err)
			}
			val := float32(val64)
			switch cmd {
			case "lighten":
				return hct.Lighten(bc, val), nil
			case "darken":
				return hct.Darken(bc, val), nil
			case "highlight":
				return hct.Highlight(bc, val), nil
			case "samelight":
				return hct.Samelight(bc, val), nil
			case "saturate":
				return hct.Saturate(bc, val), nil
			case "desaturate":
				return hct.Desaturate(bc, val), nil
			case "spin":
				return hct.Spin(bc, val), nil
			case "clearer":
				return Clearer(bc, val), nil
			case "opaquer":
				return Opaquer(bc, val), nil
			case "blend":
				clridx := strings.Index(valstr, "-")
				if clridx < 0 {
					return color.RGBA{}, fmt.Errorf("colors.FromString: blend color spec not found; format is: blend-PCT-color, got: %v; PCT-color is: %v", lstr, valstr)
				}
				bvalstr := valstr[:clridx]
				val64, err := strconv.ParseFloat(bvalstr, 32)
				if err != nil {
					return color.RGBA{}, fmt.Errorf("colors.FromString: error getting numeric value from %q: %w", bvalstr, err)
				}
				val := float32(val64)
				clrstr := valstr[clridx+1:]
				othc, err := FromString(clrstr, bc)
				return BlendRGB(val, bc, othc), err
			}
		}
		switch lstr {
		case "none", "off":
			return color.RGBA{}, nil
		case "transparent":
			return Transparent, nil
		case "currentcolor":
			return AsRGBA(bc), nil
		case "inverse":
			return Inverse(bc), nil
		default:
			return FromName(lstr)
		}
	}
}

// FromAny returns a color from the given value of any type.
// It handles values of types string, [color.Color], [*color.Color],
// [image.Image], and [*image.Image]. It takes an optional base color
// for relative transformations
// (see [FromString]).
func FromAny(val any, base ...color.Color) (color.RGBA, error) {
	switch vv := val.(type) {
	case string:
		return FromString(vv, base...)
	case color.Color:
		return AsRGBA(vv), nil
	case *color.Color:
		return AsRGBA(*vv), nil
	case image.Image:
		return ToUniform(vv), nil
	case *image.Image:
		return ToUniform(*vv), nil
	default:
		return color.RGBA{}, fmt.Errorf("colors.FromAny: could not get color from value %v of type %T", val, val)
	}
}

// FromHex parses the given non-alpha-premultiplied hex color string
// and returns the resulting alpha-premultiplied color.
func FromHex(hex string) (color.RGBA, error) {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b, a int
	a = 255
	if len(hex) == 3 {
		format := "%1x%1x%1x"
		fmt.Sscanf(hex, format, &r, &g, &b)
		r |= r << 4
		g |= g << 4
		b |= b << 4
	} else if len(hex) == 6 {
		format := "%02x%02x%02x"
		fmt.Sscanf(hex, format, &r, &g, &b)
	} else if len(hex) == 8 {
		format := "%02x%02x%02x%02x"
		fmt.Sscanf(hex, format, &r, &g, &b, &a)
	} else {
		return color.RGBA{}, fmt.Errorf("colors.FromHex: could not process %q", hex)
	}
	return AsRGBA(color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(a)}), nil
}

// AsHex returns the color as a standard 2-hexadecimal-digits-per-component
// non-alpha-premultiplied hex color string.
func AsHex(c color.Color) string {
	if c == nil {
		return "nil"
	}
	r := color.NRGBAModel.Convert(c).(color.NRGBA)
	if r.A == 255 {
		return fmt.Sprintf("#%02X%02X%02X", r.R, r.G, r.B)
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", r.R, r.G, r.B, r.A)
}

// WithR returns the given color with the red
// component (R) set to the given alpha-premultiplied value
func WithR(c color.Color, r uint8) color.RGBA {
	rc := AsRGBA(c)
	rc.R = r
	return rc
}

// WithG returns the given color with the green
// component (G) set to the given alpha-premultiplied value
func WithG(c color.Color, g uint8) color.RGBA {
	rc := AsRGBA(c)
	rc.G = g
	return rc
}

// WithB returns the given color with the blue
// component (B) set to the given alpha-premultiplied value
func WithB(c color.Color, b uint8) color.RGBA {
	rc := AsRGBA(c)
	rc.B = b
	return rc
}

// WithA returns the given color with the
// transparency (A) set to the given value,
// with the color premultiplication updated.
func WithA(c color.Color, a uint8) color.RGBA {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	n.A = a
	return AsRGBA(n)
}

// WithAF32 returns the given color with the
// transparency (A) set to the given float32 value
// between 0 and 1, with the color premultiplication updated.
func WithAF32(c color.Color, a float32) color.RGBA {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	a = math32.Clamp(a, 0, 1)
	n.A = uint8(a * 255)
	return AsRGBA(n)
}

// ApplyOpacity applies the given opacity (0-1) to the given color
// and returns the result. It is different from [WithAF32] in that it
// sets the transparency (A) value of the color to the current value
// times the given value instead of just directly overriding it.
func ApplyOpacity(c color.Color, opacity float32) color.RGBA {
	r := AsRGBA(c)
	if opacity >= 1 {
		return r
	}
	a := r.A
	// new A is current A times opacity
	return WithA(c, uint8(float32(a)*opacity))
}

// ApplyOpacityNRGBA applies the given opacity (0-1) to the given color
// and returns the result. It is different from [WithAF32] in that it
// sets the transparency (A) value of the color to the current value
// times the given value instead of just directly overriding it.
// It is the [color.NRGBA] version of [ApplyOpacity].
func ApplyOpacityNRGBA(c color.Color, opacity float32) color.NRGBA {
	r := color.NRGBAModel.Convert(c).(color.NRGBA)
	if opacity >= 1 {
		return r
	}
	a := r.A
	// new A is current A times opacity
	return color.NRGBA{r.R, r.G, r.B, uint8(float32(a) * opacity)}
}

// Clearer returns a color that is the given amount
// more transparent (lower alpha value) in terms of
// RGBA absolute alpha from 0 to 100, with the color
// premultiplication updated.
func Clearer(c color.Color, amount float32) color.RGBA {
	f32 := NRGBAF32Model.Convert(c).(NRGBAF32)
	f32.A -= amount / 100
	f32.A = math32.Clamp(f32.A, 0, 1)
	return AsRGBA(f32)
}

// Opaquer returns a color that is the given amount
// more opaque (higher alpha value) in terms of
// RGBA absolute alpha from 0 to 100,
// with the color premultiplication updated.
func Opaquer(c color.Color, amount float32) color.RGBA {
	f32 := NRGBAF32Model.Convert(c).(NRGBAF32)
	f32.A += amount / 100
	f32.A = math32.Clamp(f32.A, 0, 1)
	return AsRGBA(f32)
}

// Inverse returns the inverse of the given color
// (255 - each component). It does not change the
// alpha channel.
func Inverse(c color.Color) color.RGBA {
	r := AsRGBA(c)
	return color.RGBA{255 - r.R, 255 - r.G, 255 - r.B, r.A}
}

// Add adds given color deltas to this color, safely avoiding overflow > 255
func Add(c, dc color.Color) color.RGBA {
	r, g, b, a := c.RGBA()      // uint32
	dr, dg, db, da := dc.RGBA() // uint32
	r = (r + dr) >> 8
	g = (g + dg) >> 8
	b = (b + db) >> 8
	a = (a + da) >> 8
	if r > 255 {
		r = 255
	}
	if g > 255 {
		g = 255
	}
	if b > 255 {
		b = 255
	}
	if a > 255 {
		a = 255
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
}

// Sub subtracts given color deltas from this color, safely avoiding underflow < 0
func Sub(c, dc color.Color) color.RGBA {
	r, g, b, a := c.RGBA()      // uint32
	dr, dg, db, da := dc.RGBA() // uint32
	r = (r - dr) >> 8
	g = (g - dg) >> 8
	b = (b - db) >> 8
	a = (a - da) >> 8
	if r > 255 { // overflow
		r = 0
	}
	if g > 255 {
		g = 0
	}
	if b > 255 {
		b = 0
	}
	if a > 255 {
		a = 0
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
}
