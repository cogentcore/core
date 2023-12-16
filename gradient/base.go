// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

package gradient

import (
	"image"
	"image/color"
	"math"

	"goki.dev/colors"
)

// Base contains the data and logic common to all gradient types.
type Base struct {

	// the stops for the gradient; use AddStop to add stops
	Stops []Stop `set:"-"`

	// the spread method used for the gradient if it stops before the end
	Spread SpreadMethods

	// the colorspace algorithm to use for blending colors
	Blend colors.BlendTypes
}

// ColorModel returns the color model used by the gradient, which is [color.RGBAModel]
func (l *Linear) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds returns the bounds of the gradient, which are infinite.
func (l *Linear) Bounds() image.Rectangle {
	return image.Rect(math.MinInt, math.MinInt, math.MaxInt, math.MaxInt)
}
