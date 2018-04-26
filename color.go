// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"

	"image/color"
	"log"
	"strings"

	"github.com/chewxy/math32"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/kit"
	"golang.org/x/image/colornames"
)

// Color defines a standard color object for GUI use, with RGBA values, and
// all the usual necessary conversion functions to / from names, strings, etc

// Color extends image/color.RGBA with more methods for converting to / from
// strings etc -- it has standard uint8 0..255 color values
type Color struct {
	R, G, B, A uint8
}

var KiT_Color = kit.Types.AddType(&Color{}, ColorProps)

var ColorProps = ki.Props{
	"style-prop": true,
}

// implements color.Color interface -- returns values in range 0x0000 - 0xffff
func (c Color) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R)
	r |= r << 8
	g = uint32(c.G)
	g |= g << 8
	b = uint32(c.B)
	b |= b << 8
	a = uint32(c.A)
	a |= a << 8
	return
}

var NilColor Color

// check if color is the nil initial default color -- a = 0 means fully transparent black
func (c *Color) IsNil() bool {
	if c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0 {
		return true
	}
	return false
}

func (c *Color) SetToNil() {
	c.R = 0
	c.G = 0
	c.B = 0
	c.A = 0
}

func (c *Color) SetColor(ci color.Color) {
	var r, g, b, a uint32
	r, g, b, a = ci.RGBA()
	c.SetUInt32(r, g, b, a)
}

func (c *Color) SetUInt8(r, g, b, a uint8) {
	c.R = r
	c.G = g
	c.B = b
	c.A = a
}

func (c *Color) SetUInt32(r, g, b, a uint32) {
	c.R = uint8(r >> 8) // convert back to uint8
	c.G = uint8(g >> 8)
	c.B = uint8(b >> 8)
	c.A = uint8(a >> 8)
}

func (c *Color) SetInt(r, g, b, a int) {
	c.SetUInt32(uint32(r), uint32(g), uint32(b), uint32(a))
}

// Convert from 0-1 normalized floating point numbers
func (c *Color) SetFloat64(r, g, b, a float64) {
	c.SetUInt8(uint8(r*255.0), uint8(g*255.0), uint8(b*255.0), uint8(a*255.0))
}

// Convert from 0-1 normalized floating point numbers
func (c *Color) SetFloat32(r, g, b, a float32) {
	c.SetUInt8(uint8(r*255.0), uint8(g*255.0), uint8(b*255.0), uint8(a*255.0))
}

// Convert from 0-1 normalized floating point numbers, non alpha-premultiplied
func (c *Color) SetNPFloat32(r, g, b, a float32) {
	r *= a
	g *= a
	b *= a
	c.SetFloat32(r, g, b, a)
}

// Convert to 0-1 normalized floating point numbers, still alpha-premultiplied
func (c Color) ToFloat32() (r, g, b, a float32) {
	r = float32(c.R) / 255.0
	g = float32(c.G) / 255.0
	b = float32(c.B) / 255.0
	a = float32(c.A) / 255.0
	return
}

// Convert to 0-1 normalized floating point numbers, not alpha premultiplied
func (c Color) ToNPFloat32() (r, g, b, a float32) {
	r, g, b, a = c.ToFloat32()
	if a != 0 {
		r /= a
		g /= a
		b /= a
	}
	return
}

// Convert from HSLA: [0..360], Saturation [0..1], and Luminance
// (lightness) [0..1] of the color using float32 values
func (c *Color) SetHSLA(h, s, l, a float32) {
	r, g, b := HSLtoRGBf32(h, s, l)
	c.SetNPFloat32(r, g, b, a)
}

// Convert from HSL: [0..360], Saturation [0..1], and Luminance
// (lightness) [0..1] of the color using float32 values
func (c *Color) SetHSL(h, s, l float32) {
	r, g, b := HSLtoRGBf32(h, s, l)
	c.SetNPFloat32(r, g, b, float32(c.A)/255.0)
}

// Convert to HSLA: [0..360], Saturation [0..1], and Luminance
// (lightness) [0..1] of the color using float32 values
func (c *Color) ToHSLA() (h, s, l, a float32) {
	r, g, b, a := c.ToNPFloat32()
	h, s, l = RGBtoHSLf32(r, g, b)
	return
}

func cvtPctStringErr(gotpct bool, pctstr string) {
	if !gotpct {
		log.Printf("gi.Color.SetString -- percent was not converted from: %v\n", pctstr)
	}
}

// SetString sets color value from string, including # hex specs, standard
// color names, "none" or "off", or the following transformations (which
// use a non-nil base color as the starting point, if it is provided):
//
// * lighter-PCT or darker-PCT: PCT is amount to lighten or darken (using HSL), e.g., 10=10%
// * saturate-PCT or pastel-PCT: manipulates the saturation level in HSL by PCT
// * clearer-PCT or opaquer-PCT: manipulates the alpha level by PCT
// * blend-PCT-color: blends given percent of given color name relative to base (or current)
//
func (c *Color) SetString(str string, base color.Color) error {
	if len(str) == 0 { // consider it null
		c.SetToNil()
	}
	low := strings.ToLower(str)
	if low[0] == '#' {
		return c.ParseHex(str)
	} else if strings.HasPrefix(low, "hsl(") {
		val := strings.TrimLeft(low, "hsl(")
		val = strings.TrimRight(val, ")")
		format := "%d,%d,%d"
		var h, s, l int
		fmt.Sscanf(val, format, &h, &s, &l)
		c.SetHSL(float32(h), float32(s)/100.0, float32(l)/100.0)
	} else if strings.HasPrefix(low, "rgb(") {
		val := strings.TrimLeft(low, "rgb(")
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
		c.SetUInt8(uint8(r), uint8(g), uint8(b), uint8(a))
	} else {
		if hidx := strings.Index(low, "-"); hidx > 0 {
			cmd := low[:hidx]
			pctstr := low[hidx+1:]
			pct, gotpct := kit.ToFloat32(pctstr)
			switch cmd {
			case "lighter":
				cvtPctStringErr(gotpct, pctstr)
				if base != nil {
					c.SetColor(base)
				}
				c.SetColor(c.Lighter(pct))
				return nil
			case "darker":
				cvtPctStringErr(gotpct, pctstr)
				if base != nil {
					c.SetColor(base)
				}
				c.SetColor(c.Darker(pct))
				return nil
			case "saturate":
				cvtPctStringErr(gotpct, pctstr)
				if base != nil {
					c.SetColor(base)
				}
				c.SetColor(c.Saturate(pct))
				return nil
			case "pastel":
				cvtPctStringErr(gotpct, pctstr)
				if base != nil {
					c.SetColor(base)
				}
				c.SetColor(c.Pastel(pct))
				return nil
			case "clearer":
				cvtPctStringErr(gotpct, pctstr)
				if base != nil {
					c.SetColor(base)
				}
				c.SetColor(c.Clearer(pct))
				return nil
			case "opaquer":
				cvtPctStringErr(gotpct, pctstr)
				if base != nil {
					c.SetColor(base)
				}
				c.SetColor(c.Opaquer(pct))
				return nil
			case "blend":
				if base != nil {
					c.SetColor(base)
				}
				clridx := strings.Index(pctstr, "-")
				if clridx < 0 {
					err := fmt.Errorf("gi.Color.SetString -- blend color spec not found -- format is: blend-PCT-color, got: %v -- PCT-color is: %v\n", low, pctstr)
					return err
				}
				pctstr = low[hidx+1 : clridx]
				pct, gotpct = kit.ToFloat32(pctstr)
				cvtPctStringErr(gotpct, pctstr)
				clrstr := low[clridx+1:]
				othc, err := ColorFromString(clrstr, base)
				c.SetColor(c.Blend(pct, &othc))
				return err
			}
		}
		switch low {
		case "none", "off":
			c.SetToNil()
			return nil
		case "transparent":
			c.SetUInt8(0xFF, 0xFF, 0xFF, 0)
			return nil
		default:
			nc, ok := colornames.Map[low]
			if !ok {
				err := fmt.Errorf("gi Color String: name not found %v", str)
				log.Printf("%v\n", err)
				return err
			} else {
				c.SetUInt8(nc.R, nc.G, nc.B, nc.A)
			}
		}
	}
	return nil
}

// ColorFromString returns a new color set from given string and optional base
// color for transforms -- see SetString
func ColorFromString(str string, base color.Color) (Color, error) {
	var c Color
	err := c.SetString(str, base)
	return c, err
}

// parse Hex color -- this is from fogelman/gg I think..
func (c *Color) ParseHex(x string) error {
	x = strings.TrimPrefix(x, "#")
	var r, g, b, a int
	a = 255
	got := false
	if len(x) == 3 {
		format := "%1x%1x%1x"
		fmt.Sscanf(x, format, &r, &g, &b)
		r |= r << 4
		g |= g << 4
		b |= b << 4
		got = true
	} else if len(x) == 6 {
		format := "%02x%02x%02x"
		fmt.Sscanf(x, format, &r, &g, &b)
		got = true
	} else if len(x) == 8 {
		format := "%02x%02x%02x%02x"
		fmt.Sscanf(x, format, &r, &g, &b, &a)
		got = true
	} else {
		err := fmt.Errorf("gi Color ParseHex could not process: %v", x)
		log.Printf("%v\n", err)
		return err
	}

	if got {
		c.R = uint8(r)
		c.G = uint8(g)
		c.B = uint8(b)
		c.A = uint8(a)
	}
	return nil
}

// Lighter returns a color that is lighter by the given percent, e.g., 50 = 50%
// lighter, relative to maximum possible lightness -- converts to HSL,
// multiplies the L factor, and then converts back to RGBA
func (c *Color) Lighter(pct float32) Color {
	hsl := HSLAModel.Convert(*c).(HSLA)
	pct = InRange32(pct, 0, 100.0)
	hsl.L += (1.0 - hsl.L) * (pct / 100.0)
	return ColorModel.Convert(hsl).(Color)
}

// Darker returns a color that is darker by the given percent, e.g., 50 = 50%
// darker, relative to maximum possible darkness -- converts to HSL,
// multiplies the L factor, and then converts back to RGBA
func (c *Color) Darker(pct float32) Color {
	hsl := HSLAModel.Convert(*c).(HSLA)
	pct = InRange32(pct, 0, 100.0)
	hsl.L -= hsl.L * (pct / 100.0)
	return ColorModel.Convert(hsl).(Color)
}

// Saturate returns a color that is more saturated by the given percent: 100 =
// 100% more saturated, etc -- converts to HSL, multiplies the S factor, and
// then converts back to RGBA
func (c *Color) Saturate(pct float32) Color {
	hsl := HSLAModel.Convert(*c).(HSLA)
	pct = InRange32(pct, 0, 100.0)
	hsl.S += (1.0 - hsl.S) * (pct / 100.0)
	return ColorModel.Convert(hsl).(Color)
}

// Pastel returns a color that is less saturated (more pastel-like) by the
// given percent: 100 = 100% less saturated (i.e., grey) -- converts to HSL,
// multiplies the S factor, and then converts back to RGBA
func (c *Color) Pastel(pct float32) Color {
	hsl := HSLAModel.Convert(*c).(HSLA)
	pct = InRange32(pct, 0, 100.0)
	hsl.S -= hsl.S * (pct / 100.0)
	return ColorModel.Convert(hsl).(Color)
}

// Clearer returns a color that is given percent more transparent (lower alpha
// value) relative to current alpha level
func (c *Color) Clearer(pct float32) Color {
	f32 := NRGBAf32Model.Convert(*c).(NRGBAf32)
	pct = InRange32(pct, 0, 100.0)
	f32.A -= f32.A * (pct / 100.0)
	return ColorModel.Convert(f32).(Color)
}

// Opaquer returns a color that is given percent more opaque (higher alpha
// value) relative to current alpha level
func (c *Color) Opaquer(pct float32) Color {
	f32 := NRGBAf32Model.Convert(*c).(NRGBAf32)
	pct = InRange32(pct, 0, 100.0)
	f32.A += (1.0 - f32.A) * (pct / 100.0)
	return ColorModel.Convert(f32).(Color)
}

// Blend returns a color that is the given percent blend between current color
// and given clr -- 10 = 10% of the clr and 90% of the current color, etc --
// blending is done directly on non-pre-multiplied RGB values
func (c *Color) Blend(pct float32, clr color.Color) Color {
	f32 := NRGBAf32Model.Convert(*c).(NRGBAf32)
	othc := NRGBAf32Model.Convert(clr).(NRGBAf32)
	pct = InRange32(pct, 0, 100.0)
	oth := pct / 100.0
	me := 1.0 - pct/100.0
	f32.R = me*f32.A + oth*othc.R
	f32.G = me*f32.G + oth*othc.G
	f32.B = me*f32.B + oth*othc.B
	f32.A = me*f32.A + oth*othc.A
	return ColorModel.Convert(f32).(Color)
}

/////////////////////////////////////////////////////////////////////////////
//  float32 RGBA color

// RGBAf32 stores alpha-premultiplied RGBA values in float32 0..1 normalized
// format -- more useful for converting to other spaces
type RGBAf32 struct {
	R, G, B, A float32
}

// Implements the color.Color interface
func (c RGBAf32) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R*65535.0 + 0.5)
	g = uint32(c.G*65535.0 + 0.5)
	b = uint32(c.B*65535.0 + 0.5)
	a = uint32(c.A*65535.0 + 0.5)
	return
}

// NRGBAf32 stores non-alpha-premultiplied RGBA values in float32 0..1
// normalized format -- more useful for converting to other spaces
type NRGBAf32 struct {
	R, G, B, A float32
}

// Implements the color.Color interface
func (c NRGBAf32) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R*c.A*65535.0 + 0.5)
	g = uint32(c.G*c.A*65535.0 + 0.5)
	b = uint32(c.B*c.A*65535.0 + 0.5)
	a = uint32(c.A*65535.0 + 0.5)
	return
}

/////////////////////////////////////////////////////////////////////////////
//  HSLA color -- HSL is proposed to be supported in CSS3 and seems better than HSV

// Hsl returns the Hue [0..360], Saturation [0..1], and Luminance (lightness) [0..1] of the color.

// HSLA represents the Hue [0..360], Saturation [0..1], and Luminance
// (lightness) [0..1] of the color using float32 values
type HSLA struct {
	H, S, L, A float32
}

// Implements the color.Color interface
func (c HSLA) RGBA() (r, g, b, a uint32) {
	fr, fg, fb := HSLtoRGBf32(c.H, c.S, c.L)
	r = uint32(fr*c.A*65535.0 + 0.5)
	g = uint32(fg*c.A*65535.0 + 0.5)
	b = uint32(fb*c.A*65535.0 + 0.5)
	a = uint32(c.A*65535.0 + 0.5)
	return
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
	r = hueToRGBf32(p, q, h+1.0/3.0)
	g = hueToRGBf32(p, q, h)
	b = hueToRGBf32(p, q, h-1.0/3.0)
	return
}

func hueToRGBf32(p, q, t float32) float32 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
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
	min := math32.Min(math32.Min(r, g), b)
	max := math32.Max(math32.Max(r, g), b)

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

///////////////////////////////////////////////////////////////////////
// Models for conversion

var (
	ColorModel    color.Model = color.ModelFunc(colorModel)
	RGBAf32Model  color.Model = color.ModelFunc(rgbaf32Model)
	NRGBAf32Model color.Model = color.ModelFunc(nrgbaf32Model)
	HSLAModel     color.Model = color.ModelFunc(hslaf32Model)
)

func colorModel(c color.Color) color.Color {
	if rg, ok := c.(color.RGBA); ok {
		return Color{rg.R, rg.G, rg.B, rg.A}
	}
	if _, ok := c.(Color); ok {
		return c
	}
	r, g, b, a := c.RGBA()
	return Color{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func rgbaf32Model(c color.Color) color.Color {
	if _, ok := c.(RGBAf32); ok {
		return c
	}
	r, g, b, a := c.RGBA()
	return RGBAf32{float32(r) / 65535.0, float32(g) / 65535.0, float32(b) / 65535.0, float32(a) / 65535.0}
}

func nrgbaf32Model(c color.Color) color.Color {
	if _, ok := c.(NRGBAf32); ok {
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
	return NRGBAf32{float32(r) / 65535.0, float32(g) / 65535.0, float32(b) / 65535.0, float32(a) / 65535.0}
}

func hslaf32Model(c color.Color) color.Color {
	if _, ok := c.(HSLA); ok {
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
	fr := float32(r) / 65535.0
	fg := float32(g) / 65535.0
	fb := float32(b) / 65535.0
	fa := float32(a) / 65535.0

	h, s, l := RGBtoHSLf32(fr, fg, fb)

	return HSLA{h, s, l, fa}
}
