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

	"goki.dev/colors"
	"goki.dev/mat32/v2"
)

// Gradient is the interface that all gradient types satisfy.
type Gradient interface {
	image.Image

	// AsBase returns the [Base] of the gradient
	AsBase() *Base
}

// Base contains the data and logic common to all gradient types.
type Base struct { //gti:add -setters

	// the stops for the gradient; use AddStop to add stops
	Stops []Stop `set:"-"`

	// the spread method used for the gradient if it stops before the end
	Spread Spreads

	// the colorspace algorithm to use for blending colors
	Blend colors.BlendTypes

	// the units to use for the gradient
	Units Units

	// the bounding box of the object with the gradient; this is used when rendering
	// gradients with [Units] of [ObjectBoundingBox].
	Box mat32.Box2

	// Transform is the transformation matrix applied to the gradient's points.
	Transform mat32.Mat2
}

// Stop represents a single stop in a gradient
type Stop struct {

	// the color of the stop
	Color color.RGBA

	// the position of the stop between 0 and 1
	Pos float32
}

// AddStop adds a new stop with the given color and position to the gradient.
func (b *Base) AddStop(color color.RGBA, pos float32) {
	b.Stops = append(b.Stops, Stop{color, pos})
}

// AsBase returns the [Base] of the gradient
func (b *Base) AsBase() *Base {
	return b
}

// NewBase returns a new [Base] with default values. It should
// only be used in the New functions of gradient types.
func NewBase() Base {
	return Base{
		Box:       mat32.B2(0, 0, 100, 100),
		Transform: mat32.Identity2D(),
	}
}

// ColorModel returns the color model used by the gradient image, which is [color.RGBAModel]
func (b *Base) ColorModel() color.Model {
	return color.RGBAModel
}

// Bounds returns the bounds of the gradient image, which are infinite.
func (b *Base) Bounds() image.Rectangle {
	return image.Rect(-1e9, -1e9, 1e9, 1e9)
}

// CopyFrom copies from the given gradient (cp) onto this gradient (g),
// making new copies of the stops instead of re-using pointers
func CopyFrom(g Gradient, cp Gradient) {
	switch g := g.(type) {
	case *Linear:
		*g = *cp.(*Linear)
	case *Radial:
		*g = *cp.(*Radial)
	}
	g.AsBase().CopyStopsFrom(cp.AsBase())
}

// CopyStopsFrom copies the base gradient stops from the given base gradient,
// if both have gradient stops
func (b *Base) CopyStopsFrom(cp *Base) {
	if len(b.Stops) == 0 || len(cp.Stops) == 0 {
		b.Stops = nil
		return
	}
	if len(b.Stops) != len(cp.Stops) {
		b.Stops = make([]Stop, len(cp.Stops))
	}
	copy(b.Stops, cp.Stops)
}

// ObjectMatrix returns the effective object transformation matrix for a gradient
// with [Units] of [ObjectBoundingBox].
func (b *Base) ObjectMatrix() mat32.Mat2 {
	w, h := b.Box.Size().X, b.Box.Size().Y
	oriX, oriY := b.Box.Min.X, b.Box.Min.Y
	return mat32.Identity2D().Translate(oriX, oriY).Scale(w, h).
		Mul(b.Transform).Scale(1/w, 1/h).Translate(-oriX, -oriY).Inverse()
}

// GetColor returns the color at the given normalized position along the
// gradient's stops using its spread method and blend algorithm.
func (b *Base) GetColor(pos float32) color.Color {
	d := len(b.Stops)

	// These cases can be taken care of early on
	if b.Spread == Pad {
		if pos >= 1 {
			return b.Stops[d-1].Color
		}
		if pos <= 0 {
			return b.Stops[0].Color
		}
	}

	modRange := float32(1)
	if b.Spread == Reflect {
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
	case Repeat:
		var s1, s2 Stop
		switch place {
		case 0, d:
			s1, s2 = b.Stops[d-1], b.Stops[0]
		default:
			s1, s2 = b.Stops[place-1], b.Stops[place]
		}
		return b.BlendStops(mod, s1, s2, false)
	case Reflect:
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
