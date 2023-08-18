// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"strings"

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
// an error if the name is not found; see [ColorFromName]
// for a version that does not return an error.
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

// SetString sets color value from string, including # hex specs, standard
// color names, "none" or "off", or the following transformations (which
// use a non-nil base color as the starting point, if it is provided):
// inverse = inverse of base color
//
// * lighter-PCT or darker-PCT: PCT is amount to lighten or darken (using HSL), e.g., 10=10%
// * saturate-PCT or pastel-PCT: manipulates the saturation level in HSL by PCT
// * clearer-PCT or opaquer-PCT: manipulates the alpha level by PCT
// * blend-PCT-color: blends given percent of given color name relative to base (or current)
func FromString(str string, base color.Color) (color.RGBA, error) {
	return color.RGBA{}, nil
	// if len(str) == 0 { // consider it null
	// 	return color.RGBA{}, nil
	// }
	// lstr := strings.ToLower(str)
	// switch {
	// case lstr[0] == '#':
	// 	return ColorFromHex(str)
	// case strings.HasPrefix(lstr, "hsl("):
	// 	val := lstr[4:]
	// 	val = strings.TrimRight(val, ")")
	// 	format := "%d,%d,%d"
	// 	var h, s, l int
	// 	fmt.Sscanf(val, format, &h, &s, &l)
	// 	return hsl.NewHSL(float32(h), float32(s)/100.0, float32(l)/100.0).AsRGBA(), nil
	// case strings.HasPrefix(lstr, "rgb("):
	// 	val := lstr[4:]
	// 	val = strings.TrimRight(val, ")")
	// 	val = strings.Trim(val, "%")
	// 	var r, g, b, a int
	// 	a = 255
	// 	format := "%d,%d,%d"
	// 	if strings.Count(val, ",") == 4 {
	// 		format = "%d,%d,%d,%d"
	// 		fmt.Sscanf(val, format, &r, &g, &b, &a)
	// 	} else {
	// 		fmt.Sscanf(val, format, &r, &g, &b)
	// 	}
	// 	c.SetUInt8(uint8(r), uint8(g), uint8(b), uint8(a))
	// case strings.HasPrefix(lstr, "rgba("):
	// 	val := lstr[5:]
	// 	val = strings.TrimRight(val, ")")
	// 	val = strings.Trim(val, "%")
	// 	var r, g, b, a int
	// 	format := "%d,%d,%d,%d"
	// 	fmt.Sscanf(val, format, &r, &g, &b, &a)
	// 	c.SetUInt8(uint8(r), uint8(g), uint8(b), uint8(a))
	// case strings.HasPrefix(lstr, "pref("):
	// 	val := lstr[5:]
	// 	val = strings.TrimRight(val, ")")
	// 	clr := ThePrefs.PrefColor(val)
	// 	if clr != nil {
	// 		*c = *clr
	// 	}
	// default:
	// 	if hidx := strings.Index(lstr, "-"); hidx > 0 {
	// 		cmd := lstr[:hidx]
	// 		pctstr := lstr[hidx+1:]
	// 		pct, gotpct := kit.ToFloat32(pctstr)
	// 		switch cmd {
	// 		case "lighter":
	// 			cvtPctStringErr(gotpct, pctstr)
	// 			if base != nil {
	// 				c.SetColor(base)
	// 			}
	// 			c.SetColor(c.Lighter(pct))
	// 			return nil
	// 		case "darker":
	// 			cvtPctStringErr(gotpct, pctstr)
	// 			if base != nil {
	// 				c.SetColor(base)
	// 			}
	// 			c.SetColor(c.Darker(pct))
	// 			return nil
	// 		case "highlight":
	// 			cvtPctStringErr(gotpct, pctstr)
	// 			if base != nil {
	// 				c.SetColor(base)
	// 			}
	// 			c.SetColor(c.Highlight(pct))
	// 			return nil
	// 		case "samelight":
	// 			cvtPctStringErr(gotpct, pctstr)
	// 			if base != nil {
	// 				c.SetColor(base)
	// 			}
	// 			c.SetColor(c.Samelight(pct))
	// 			return nil
	// 		case "saturate":
	// 			cvtPctStringErr(gotpct, pctstr)
	// 			if base != nil {
	// 				c.SetColor(base)
	// 			}
	// 			c.SetColor(c.Saturate(pct))
	// 			return nil
	// 		case "pastel":
	// 			cvtPctStringErr(gotpct, pctstr)
	// 			if base != nil {
	// 				c.SetColor(base)
	// 			}
	// 			c.SetColor(c.Pastel(pct))
	// 			return nil
	// 		case "clearer":
	// 			cvtPctStringErr(gotpct, pctstr)
	// 			if base != nil {
	// 				c.SetColor(base)
	// 			}
	// 			c.SetColor(c.Clearer(pct))
	// 			return nil
	// 		case "opaquer":
	// 			cvtPctStringErr(gotpct, pctstr)
	// 			if base != nil {
	// 				c.SetColor(base)
	// 			}
	// 			c.SetColor(c.Opaquer(pct))
	// 			return nil
	// 		case "blend":
	// 			if base != nil {
	// 				c.SetColor(base)
	// 			}
	// 			clridx := strings.Index(pctstr, "-")
	// 			if clridx < 0 {
	// 				err := fmt.Errorf("gi.Color.SetString -- blend color spec not found -- format is: blend-PCT-color, got: %v -- PCT-color is: %v", lstr, pctstr)
	// 				return err
	// 			}
	// 			pctstr = lstr[hidx+1 : clridx]
	// 			pct, gotpct = kit.ToFloat32(pctstr)
	// 			cvtPctStringErr(gotpct, pctstr)
	// 			clrstr := lstr[clridx+1:]
	// 			othc, err := ColorFromString(clrstr, base)
	// 			c.SetColor(c.Blend(pct, &othc))
	// 			return err
	// 		}
	// 	}
	// 	switch lstr {
	// 	case "none", "off":
	// 		c.SetToNil()
	// 		return nil
	// 	case "transparent":
	// 		c.SetUInt8(0xFF, 0xFF, 0xFF, 0)
	// 		return nil
	// 	case "inverse":
	// 		if base != nil {
	// 			c.SetColor(base)
	// 		}
	// 		c.SetColor(c.Inverse())
	// 		return nil
	// 	default:
	// 		return c.SetName(lstr)
	// 	}
	// }
	// return nil
}

func MustFromString(str string, base color.Color) color.RGBA {
	c, err := FromString(str, base)
	if err != nil {
		panic("colors.MustFromString: " + err.Error())
	}
	return c
}

func LogFromString(str string, base color.Color) color.RGBA {
	c, err := FromString(str, base)
	if err != nil {
		log.Println("error: colors.LogFromString: " + err.Error())
	}
	return c
}

func FromAny(val any, base color.Color) (color.RGBA, error) {
	return color.RGBA{}, nil
}

func MustFromAny(val any, base color.Color) color.RGBA {
	c, err := FromAny(val, base)
	if err != nil {
		panic("colors.MustFromAny: " + err.Error())
	}
	return c
}

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
	// TODO: add blend
	return color.RGBA{}
	// f32 := NRGBAf32Model.Convert(*c).(NRGBAf32)
	// othc := NRGBAf32Model.Convert(clr).(NRGBAf32)
	// pct = mat32.Clamp(pct, 0, 100.0)
	// oth := pct / 100.0
	// me := 1.0 - pct/100.0
	// f32.R = me*f32.R + oth*othc.R
	// f32.G = me*f32.G + oth*othc.G
	// f32.B = me*f32.B + oth*othc.B
	// f32.A = me*f32.A + oth*othc.A
	// return ColorModel.Convert(f32).(Color)
}
