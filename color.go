// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package Gi (GoGi) provides a complete Graphical Interface based on GoKi Tree Node structs

	2D and 3D scenegraphs supported, each rendering to respective Viewport2D or 3D
	which in turn can be integrated within the other type of scenegraph.
	Within 2D scenegraph, the following are supported
		* SVG-based rendering nodes for basic shapes, paths, curves, arcs etc, with SVG / CSS properties
		* Widget nodes for GUI actions (Buttons, etc), with support for full SVG-based rendering of styles, using Qt-based naming and functionality, including TreeView, TableView
		* Layouts for placing widgets, based on QtQuick model
		* Primary geometry is managed in terms of inherited position offsets from top-left,
		  in a widget-like manner, but svg-based transforms also supported.
*/
package gi

import (
	"fmt"
	"golang.org/x/image/colornames"
	// "gopkg.in/go-playground/colors.v1"
	"image/color"
	"log"
	"strings"
)

// our color object
type Color struct {
	Rgba color.RGBA
}

func (c *Color) RGBA() (r, g, b, a uint32) {
	return c.Rgba.RGBA()
}

// check if color is the nil initial default color -- a = 0 means fully transparent black
func (c *Color) IsNil() bool {
	if c.Rgba.R == 0 && c.Rgba.G == 0 && c.Rgba.B == 0 && c.Rgba.A == 0 {
		return true
	}
	return false
}

func (c *Color) SetColor(ci color.Color) {
	var r, g, b, a uint32
	r, g, b, a = ci.RGBA()
	c.SetRGBAUInt32(r, g, b, a)
}

func (c *Color) SetRGBAUInt8(r, g, b, a uint8) {
	c.Rgba.R = r
	c.Rgba.G = g
	c.Rgba.B = b
	c.Rgba.A = a
}

func (c *Color) SetRGBAUInt32(r, g, b, a uint32) {
	c.Rgba.R = uint8(r / 0x101) // convert back to uint8
	c.Rgba.G = uint8(g / 0x101)
	c.Rgba.B = uint8(b / 0x101)
	c.Rgba.A = uint8(a / 0x101)
}

func (c *Color) FromString(nm string) error {
	if len(nm) == 0 {
		err := fmt.Errorf("gi Color FromString: empty name")
		log.Printf("%v\n", err)
		return err
	}
	if nm[0] == '#' {
		return c.ParseHex(nm)
	} else {
		low := strings.ToLower(nm)
		nc, ok := colornames.Map[low]
		if !ok {
			err := fmt.Errorf("gi Color FromString: name not found %v", nm)
			log.Printf("%v\n", err)
			return err
		} else {
			c.Rgba = nc
		}
	}
	return nil
}

func ColorFromString(nm string) (Color, error) {
	var c Color
	err := c.FromString(nm)
	return c, err
}

// parse Hex color
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
		c.Rgba.R = uint8(r)
		c.Rgba.G = uint8(g)
		c.Rgba.B = uint8(b)
		c.Rgba.A = uint8(a)
	}
	return nil
}
