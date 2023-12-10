// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

//go:generate goki generate ./...

import (
	"errors"
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"goki.dev/cam/hct"
	"goki.dev/cam/hsl"
	"goki.dev/mat32/v2"
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
	return color.RGBA{uint8(r * 255.0), uint8(g * 255.0), uint8(b * 255.0), uint8(a * 255.0)}
}

// FromFloat32 makes a new RGBA color from the given 0-1
// normalized floating point numbers (alpha-premultiplied)
func FromFloat32(r, g, b, a float32) color.RGBA {
	return color.RGBA{uint8(r * 255.0), uint8(g * 255.0), uint8(b * 255.0), uint8(a * 255.0)}
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
// "none" or "off", or any of the following transformations
// (which use the base color as the starting point):
//
//   - currentcolor = base color
//   - inverse = inverse of base color
//   - lighten-VAL or darken-VAL: VAL is amount to lighten or darken (using HCT), e.g., lighter-10 is 10 higher tone
//   - saturate-VAL or desaturate-VAL: manipulates the chroma level in HCT by VAL
//   - spin-VAL: manipulates the hue level in HCT by VAL
//   - clearer-VAL or opaquer-VAL: manipulates the alpha level by VAL
//   - blend-VAL-color: blends given percent of given color name relative to base
func FromString(str string, base color.Color) (color.RGBA, error) {
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
				return hct.Lighten(base, val), nil
			case "darken":
				return hct.Darken(base, val), nil
			case "highlight":
				return hct.Highlight(base, val), nil
			case "samelight":
				return hct.Samelight(base, val), nil
			case "saturate":
				return hct.Saturate(base, val), nil
			case "desaturate":
				return hct.Desaturate(base, val), nil
			case "spin":
				return hct.Spin(base, val), nil
			case "clearer":
				return Clearer(base, val), nil
			case "opaquer":
				return Opaquer(base, val), nil
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
				othc, err := FromString(clrstr, base)
				return Blend(val, base, othc), err
			}
		}
		switch lstr {
		case "none", "off":
			return color.RGBA{}, nil
		case "transparent":
			return Transparent, nil
		case "currentcolor":
			return AsRGBA(base), nil
		case "inverse":
			if base != nil {
				return Inverse(base), nil
			}
			return color.RGBA{}, errors.New("colors.FromString: base color must be provided for inverse color transformation")
		default:
			return FromName(lstr)
		}
	}
}

// FromAny returns a color from the given value of any type.
// It handles values of types string and [color.Color].
func FromAny(val any, base color.Color) (color.RGBA, error) {
	switch valv := val.(type) {
	case string:
		return FromString(valv, base)
	case color.Color:
		return AsRGBA(valv), nil
	default:
		return color.RGBA{}, fmt.Errorf("colors.FromAny: could not get color from value %v of type %T", val, val)
	}
}

// FromHex parses the given hex color string
// and returns the resulting color.
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
	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}, nil
}

// AsHex returns the color as a standard
// 2-hexadecimal-digits-per-component string
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
	a = mat32.Clamp(a, 0, 1)
	n.A = uint8(a * 255)
	return AsRGBA(n)
}

// Clearer returns a color that is the given amount
// more transparent (lower alpha value) in terms of
// RGBA absolute alpha from 0 to 100, with the color
// premultiplication updated.
func Clearer(c color.Color, amount float32) color.RGBA {
	f32 := NRGBAF32Model.Convert(c).(NRGBAF32)
	f32.A -= amount / 100
	f32.A = mat32.Clamp(f32.A, 0, 1)
	return AsRGBA(f32)
}

// Opaquer returns a color that is the given amount
// more opaque (higher alpha value) in terms of
// RGBA absolute alpha from 0 to 100,
// with the color premultiplication updated.
func Opaquer(c color.Color, amount float32) color.RGBA {
	f32 := NRGBAF32Model.Convert(c).(NRGBAF32)
	f32.A += amount / 100
	f32.A = mat32.Clamp(f32.A, 0, 1)
	return AsRGBA(f32)
}

// Blend returns a color that is the given percent blend between the first
// and second color; 10 = 10% of the first and 90% of the second, etc;
// blending is done directly on non-premultiplied RGB values, and
// a correctly premultiplied color is returned.
func Blend(pct float32, x, y color.Color) color.RGBA {
	fx := NRGBAF32Model.Convert(x).(NRGBAF32)
	fy := NRGBAF32Model.Convert(y).(NRGBAF32)
	pct = mat32.Clamp(pct, 0, 100.0)
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
