// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"errors"
	"fmt"
	"image/color"
	"strings"
)

// IsNil returns if the color is the nil initial default color
func IsNil(c color.Color) bool {
	return c == color.RGBA{}
}

// SetToNil sets the given color to a nil initial default color
func SetToNil(c *color.Color) {
	*c = color.RGBA{}
}

// AsRGBA returns the given color as an RGBA color
func AsRGBA(c color.Color) color.RGBA {
	return color.RGBAModel.Convert(c).(color.RGBA)
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
func ColorFromString(str string, base color.Color) (color.RGBA, error) {
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

func MustColorFromString(str string, base color.Color) color.RGBA {
	c, _ := ColorFromString(str, base)
	return c
}

// ColorFromHex parses the given hex color string
// and returns the resulting color. It returns any
// resulting error; see [MustColorFromHex] for a
// version that does not return an error.
func ColorFromHex(hex string) (color.RGBA, error) {
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
		return color.RGBA{}, errors.New("colors.ColorFromHex: could not process: " + hex)
	}
	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}, nil
}

// MustColorFromHex parses the given hex color string
// and returns the resulting color. It panics on any
// resultinge rror; see [ColorFromHex] for a version
// that returns an error.
func MustColorFromHex(hex string) color.RGBA {
	c, err := ColorFromHex(hex)
	if err != nil {
		panic("colors.MustColorFromHex: " + err.Error())
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
