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
	"goki.dev/mat32/v2"
)

// Base contains the data and logic common to all gradient types.
type Base struct { //gti:add

	// the stops for the gradient; use AddStop to add stops
	Stops []Stop `set:"-"`

	// the spread method used for the gradient if it stops before the end
	Spread SpreadMethods

	// the colorspace algorithm to use for blending colors
	Blend colors.BlendTypes
}

// ColorModel returns the color model used by the gradient, which is [color.RGBAModel]
func (b *Base) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds returns the bounds of the gradient, which are infinite.
func (b *Base) Bounds() image.Rectangle {
	return image.Rect(math.MinInt, math.MinInt, math.MaxInt, math.MaxInt)
}

// GetColor returns the color at the given normalized position along the
// gradient's stops using its spread method and blend algorithm.
func (b *Base) GetColor(pos float32) color.Color {
	d := len(b.Stops)

	// These cases can be taken care of early on
	if b.Spread == PadSpread {
		if pos >= 1 {
			return b.Stops[d-1].Color
		}
		if pos <= 0 {
			return b.Stops[0].Color
		}
	}

	modRange := float32(1)
	if b.Spread == ReflectSpread {
		modRange = 2
	}
	mod := mat32.Mod(pos, modRange)
	if mod < 0 {
		mod += modRange
	}

	place := 0 // Advance to place where mod is greater than the indicated stop
	for place != len(b.Stops) && mod > b.Stops[place].Pos {
		place++
	}
	switch b.Spread {
	case RepeatSpread:
		var s1, s2 Stop
		switch place {
		case 0, d:
			s1, s2 = b.Stops[d-1], b.Stops[0]
		default:
			s1, s2 = b.Stops[place-1], b.Stops[place]
		}
		return b.BlendStops(mod, s1, s2, false)
	case ReflectSpread:
		switch place {
		case 0:
			return b.Stops[0].Color
		case d:
			// Advance to place where mod-1 is greater than the stop indicated by place in reverse of the stop slice.
			// Since this is the reflect b.Spread mode, the mod interval is two, allowing the stop list to be
			// iterated in reverse before repeating the sequence.
			for place != d*2 && mod-1 > (1-b.Stops[d*2-place-1].Pos) {
				place++
			}
			switch place {
			case d:
				return b.Stops[d-1].Color
			case d * 2:
				return b.Stops[0].Color
			default:
				return b.BlendStops(mod-1, b.Stops[d*2-place], b.Stops[d*2-place-1], true)
			}
		default:
			return b.BlendStops(mod, b.Stops[place-1], b.Stops[place], false)
		}
	default: // PadSpread
		switch place {
		case 0:
			return b.Stops[0].Color
		case d:
			return b.Stops[d-1].Color
		default:
			return b.BlendStops(mod, b.Stops[place-1], b.Stops[place], false)
		}
	}
}

// BlendStops blends the given two gradient stops together based on the given position,
// using the gradient's blending algorithm. If flip is true, it flips the given position.
func (b *Base) BlendStops(pos float32, s1, s2 Stop, flip bool) color.Color {
	s1off := s1.Pos
	if s1.Pos > s2.Pos && !flip { // happens in repeat spread mode
		s1off--
		if pos > 1 {
			pos--
		}
	}
	if s2.Pos == s1off {
		return s2.Color
	}
	if flip {
		pos = 1 - pos
	}
	tp := (pos - s1off) / (s2.Pos - s1off)

	return colors.Blend(b.Blend, 100*(1-tp), s1.Color, s2.Color)
}
