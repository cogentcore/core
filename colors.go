// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"strconv"
	"strings"

	"github.com/goki/cam/hsl"
	"github.com/goki/mat32"
)

// IsNil returns whether the color is the nil initial default color
func IsNil(c color.Color) bool {
	return c == color.RGBA{}
}

// SetToNil sets the given color to a nil initial default color
func SetToNil(c *color.Color) {
	*c = color.RGBA{}
}

// AsRGBA returns the given color as an RGBA color
func AsRGBA(c color.Color) color.RGBA {
	if c == nil {
		return color.RGBA{}
	}
	return color.RGBAModel.Convert(c).(color.RGBA)
}

// AsString returns the given color as a string,
// using its String method if it exists, and formatting
// it as rgba(r, g, b, a) otherwise.
func AsString(c color.Color) string {
	if s, ok := c.(fmt.Stringer); ok {
		return s.String()
	}
	r, g, b, a := c.RGBA()
	return fmt.Sprintf("rgba(%d, %d, %d, %d)", r, g, b, a)
}

// FromName returns the color value specified
// by the given CSS standard color name. It returns
// an error if the name is not found; see [MustFromName]
// and [LogFromName] for versions that do not return an error.
func FromName(name string) (color.RGBA, error) {
	c, ok := Map[name]
	if !ok {
		return color.RGBA{}, errors.New("colors.FromName: name not found: " + name)
	}
	return c, nil
}

// MustFromName returns the color value specified
// by the given CSS standard color name. It panics
// if the name is not found; see [FromName]
// for a version that returns an error.
func MustFromName(name string) color.RGBA {
	c, err := FromName(name)
	if err != nil {
		panic("colors.MustFromName: " + err.Error())
	}
	return c
}

// LogFromName returns the color value specified
// by the given CSS standard color name. It logs an error
// if the name is not found; see [FromName]
// for a version that returns an error.
func LogFromName(name string) color.RGBA {
	c, err := FromName(name)
	if err != nil {
		log.Println("error: colors.LogFromName: " + err.Error())
	}
	return c
}

// FromString returns a color value from the given string.
// It returns any resulting error; see [MustFromString] and
// [LogFromString] for versions that do not return an error.
// FromString accepts the following types of strings: hex values,
// standard color names, "none" or "off", or
// any of the following transformations (which
// use the base color as the starting point):
// * inverse = inverse of base color
// * lighten-PCT or darken-PCT: PCT is amount to lighten or darken (using HSL), e.g., 10=10%
// * saturate-PCT or desaturate-PCT: manipulates the saturation level in HSL by PCT
// * clearer-PCT or opaquer-PCT: manipulates the alpha level by PCT
// * blend-PCT-color: blends given percent of given color name relative to base
func FromString(str string, base color.Color) (color.RGBA, error) {
	if len(str) == 0 { // consider it null
		return color.RGBA{}, nil
	}
	lstr := strings.ToLower(str)
	switch {
	case lstr[0] == '#':
		return FromHex(str)
	case strings.HasPrefix(lstr, "hsl("):
		val := lstr[4:]
		val = strings.TrimRight(val, ")")
		format := "%d,%d,%d"
		var h, s, l int
		fmt.Sscanf(val, format, &h, &s, &l)
		return hsl.New(float32(h), float32(s)/100.0, float32(l)/100.0).AsRGBA(), nil
	case strings.HasPrefix(lstr, "rgb("):
		val := lstr[4:]
		val = strings.TrimRight(val, ")")
		val = strings.Trim(val, "%")
		var r, g, b, a int
		a = 255
		format := "%d,%d,%d"
		if strings.Count(val, ",") == 4 {
			format = "%d,%d,%d,%d"
			fmt.Sscanf(val, format, &r, &g, &b, &a)
		} else {
			fmt.Sscanf(val, format, &r, &g, &b)
		}
		return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}, nil
	case strings.HasPrefix(lstr, "rgba("):
		val := lstr[5:]
		val = strings.TrimRight(val, ")")
		val = strings.Trim(val, "%")
		var r, g, b, a int
		format := "%d,%d,%d,%d"
		fmt.Sscanf(val, format, &r, &g, &b, &a)
		return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}, nil
	default:
		if hidx := strings.Index(lstr, "-"); hidx > 0 {
			cmd := lstr[:hidx]
			pctstr := lstr[hidx+1:]
			pct64, err := strconv.ParseFloat(pctstr, 32)
			if err != nil && cmd != "blend" { // blend handles separately
				return color.RGBA{}, fmt.Errorf("colors.FromString: error getting percent from '%s': %w", pctstr, err)
			}
			pct := float32(pct64)
			switch cmd {
			case "lighten":
				return hsl.Lighten(base, pct), nil
			case "darken":
				return hsl.Darken(base, pct), nil
			case "highlight":
				return hsl.Highlight(base, pct), nil
			case "samelight":
				return hsl.Samelight(base, pct), nil
			case "saturate":
				return hsl.Saturate(base, pct), nil
			case "desaturate":
				return hsl.Desaturate(base, pct), nil
			case "clearer":
				return Clearer(base, pct), nil
			case "opaquer":
				return Opaquer(base, pct), nil
			case "blend":
				clridx := strings.Index(pctstr, "-")
				if clridx < 0 {
					return color.RGBA{}, fmt.Errorf("colors.FromString: blend color spec not found; format is: blend-PCT-color, got: %v; PCT-color is: %v", lstr, pctstr)
				}
				pctstr = lstr[hidx+1 : clridx]
				pct64, err := strconv.ParseFloat(pctstr, 32)
				if err != nil {
					return color.RGBA{}, fmt.Errorf("colors.FromString: error getting percent from '%s': %w", pctstr, err)
				}
				pct := float32(pct64)
				clrstr := lstr[clridx+1:]
				othc, err := FromString(clrstr, base)
				return Blend(pct, base, othc), err
			}
		}
		switch lstr {
		case "none", "off":
			return color.RGBA{}, nil
		case "transparent":
			return Transparent, nil
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

// MustFromString returns a color value from the given string.
// It panics on any resulting error; see [FromString] for
// more information and a version that returns an error.
func MustFromString(str string, base color.Color) color.RGBA {
	c, err := FromString(str, base)
	if err != nil {
		panic("colors.MustFromString: " + err.Error())
	}
	return c
}

// LogFromString returns a color value from the given string.
// It logs any resulting error; see [FromString] for
// more information and a version that returns an error.
func LogFromString(str string, base color.Color) color.RGBA {
	c, err := FromString(str, base)
	if err != nil {
		log.Println("error: colors.LogFromString: " + err.Error())
	}
	return c
}

// FromAny returns a color from the given value of any type.
// It handles values of types string and [color.Color].
// It returns any error; see [MustFromAny] and [LogFromAny]
// for versions that do not return an error.
func FromAny(val any, base color.Color) (color.RGBA, error) {
	switch valv := val.(type) {
	case string:
		return FromString(valv, base)
	case color.Color:
		return AsRGBA(valv), nil
	default:
		return color.RGBA{}, fmt.Errorf("colors.FromAny: could not set color from value %v of type %T", val, val)
	}
}

// MustFromAny returns a color value from the given value.
// of any type. It panics on any resulting error; see [FromAny]
// for more information and a version that returns an error.
func MustFromAny(val any, base color.Color) color.RGBA {
	c, err := FromAny(val, base)
	if err != nil {
		panic("colors.MustFromAny: " + err.Error())
	}
	return c
}

// LogFromAny returns a color value from the given value.
// of any type. It logs any resulting error; see [FromAny]
// for more information and a version that returns an error.
func LogFromAny(val any, base color.Color) color.RGBA {
	c, err := FromAny(val, base)
	if err != nil {
		log.Println("error: colors.LogFromAny: " + err.Error())
	}
	return c
}

// FromHex parses the given hex color string
// and returns the resulting color. It returns any
// resulting error; see [MustFromHex] for a
// version that does not return an error.
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
		return color.RGBA{}, errors.New("colors.FromHex: could not process: " + hex)
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}, nil
}

// MustFromHex parses the given hex color string
// and returns the resulting color. It panics on any
// resulting error; see [FromHex] for a version
// that returns an error.
func MustFromHex(hex string) color.RGBA {
	c, err := FromHex(hex)
	if err != nil {
		panic("colors.MustFromHex: " + err.Error())
	}
	return c
}

// LogFromHex parses the given hex color string
// and returns the resulting color. It logs any
// resulting error; see [FromHex] for a version
// that returns an error.
func LogFromHex(hex string) color.RGBA {
	c, err := FromHex(hex)
	if err != nil {
		log.Println("error: colors.LogFromHex: " + err.Error())
	}
	return c
}

// AsHex returns the color as a standard
// 2-hexadecimal-digits-per-component string
func AsHex(c color.Color) string {
	if c == nil {
		return "nil"
	}
	r := AsRGBA(c)
	return fmt.Sprintf("#%02X%02X%02X%02X", r.R, r.G, r.B, r.A)
}

// SetR returns the given color with the red
// component (R) set to the given value
func SetR(c color.Color, r uint8) color.RGBA {
	rc := AsRGBA(c)
	rc.R = r
	return rc
}

// SetG returns the given color with the green
// component (G) set to the given value
func SetG(c color.Color, g uint8) color.RGBA {
	rc := AsRGBA(c)
	rc.G = g
	return rc
}

// SetB returns the given color with the blue
// component (B) set to the given value
func SetB(c color.Color, b uint8) color.RGBA {
	rc := AsRGBA(c)
	rc.B = b
	return rc
}

// SetA returns the given color with the
// transparency (A) set to the given value
func SetA(c color.Color, a uint8) color.RGBA {
	rc := AsRGBA(c)
	rc.A = a
	return rc
}

// SetAF32 returns the given color with the
// transparency (A) set to the given float32 value
// between 0 and 1
func SetAF32(c color.Color, a float32) color.RGBA {
	rc := AsRGBA(c)
	a = mat32.Clamp(a, 0, 1)
	rc.A = uint8(a * 255)
	return rc
}

// Clearer returns a color that is the given amount
// more transparent (lower alpha value) in terms of
// RGBA absolute alpha from 0 to 100.
func Clearer(c color.Color, amount float32) color.RGBA {
	f32 := NRGBAf32Model.Convert(c).(NRGBAf32)
	f32.A -= amount / 100
	f32.A = mat32.Clamp(f32.A, 0, 1)
	return AsRGBA(f32)
}

// Opaquer returns a color that is the given amount
// more opaque (higher alpha value) in terms of
// RGBA absolute alpha from 0 to 100.
func Opaquer(c color.Color, amount float32) color.RGBA {
	f32 := NRGBAf32Model.Convert(c).(NRGBAf32)
	f32.A += amount / 100
	f32.A = mat32.Clamp(f32.A, 0, 1)
	return AsRGBA(f32)
}

// Add adds the two given colors together, safely avoiding overflow > 255
func Add(x, y color.Color) color.RGBA {
	xr, xg, xb, xa := x.RGBA()
	yr, yg, yb, ya := y.RGBA()
	r := xr + yr
	g := xg + yg
	b := xb + yb
	a := xa + ya
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

// Sub subtracts the second color from the first color,
// safely avoiding underflow < 0
func Sub(x, y color.Color) color.RGBA {
	xr, xg, xb, xa := x.RGBA()
	yr, yg, yb, ya := y.RGBA()
	r := xr - yr
	g := xg - yg
	b := xb - yb
	a := xa - ya
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

// Blend returns a color that is the given percent blend between the first
// and second color -- 10 = 10% of the second and 90% of the first, etc --
// blending is done directly on non-pre-multiplied RGB values
func Blend(pct float32, x, y color.Color) color.RGBA {
	f32 := NRGBAf32Model.Convert(x).(NRGBAf32)
	othc := NRGBAf32Model.Convert(y).(NRGBAf32)
	pct = mat32.Clamp(pct, 0, 100.0)
	oth := pct / 100.0
	me := 1.0 - pct/100.0
	f32.R = me*f32.R + oth*othc.R
	f32.G = me*f32.G + oth*othc.G
	f32.B = me*f32.B + oth*othc.B
	f32.A = me*f32.A + oth*othc.A
	return AsRGBA(f32)
}

// Inverse returns the inverse of the given color
// (255 - each component);
// does not change the alpha channel.
func Inverse(c color.Color) color.RGBA {
	r := AsRGBA(c)
	return color.RGBA{255 - r.R, 255 - r.G, 255 - r.B, r.A}
}
