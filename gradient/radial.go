// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package gradient

import (
	"image/color"

	"goki.dev/colors"
	"goki.dev/mat32/v2"
)

// Radial represents a radial gradient. It implements the [image.Image] interface.
type Radial struct { //gti:add -setters
	Base

	// the center point of the gradient (cx and cy in SVG)
	Center mat32.Vec2

	// the focal point of the gradient (fx and fy in SVG)
	Focal mat32.Vec2

	// the radius of the gradient (r in SVG)
	Radius float32
}

// NewRadial returns a new centered [Radial] gradient.
func NewRadial() *Radial {
	return &Radial{
		// default is fully centered
		Center: mat32.V2(0.5, 0.5),
		Focal:  mat32.V2(0.5, 0.5),
		Radius: 0.5,
	}
}

// AddStop adds a new stop with the given color and position to the radial gradient.
func (r *Radial) AddStop(color color.RGBA, pos float32) *Radial {
	r.Stops = append(r.Stops, Stop{color, pos})
	return r
}

// At returns the color of the radial gradient at the given point
func (r *Radial) At(x, y int) color.Color {
	switch len(r.Stops) {
	case 0:
		return colors.Black // default color for gradient without stops
	case 1:
		return r.Stops[0].Color
	}
}
