// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/srwiley/rasterx:
// Copyright 2018 by the rasterx Authors. All rights reserved.
// Created 2018 by S.R.Wiley

// Package gradient provides linear, radial, and conic color gradients.
package gradient

//go:generate core generate

import (
	"image"
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/mat32"
)

// Gradient is the interface that all gradient types satisfy.
type Gradient interface {
	image.Image

	// AsBase returns the [Base] of the gradient
	AsBase() *Base

	// Update updates the computed fields of the gradient, using
	// the given object opacity, current bounding box, and additional
	// object-level transform (i.e., the current painting transform),
	// which is applied in addition to the gradient's own Transform.
	// This must be called before rendering the gradient, and it should only be called then.
	Update(opacity float32, box mat32.Box2, objTransform mat32.Mat2)
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

	// Transform is the gradient's own transformation matrix applied to the gradient's points.
	// This is a property of the Gradient itself.
	Transform mat32.Mat2

	// Opacity is the overall object opacity multiplier, applied in conjunction with the
	// stop-level opacity blending.
	Opacity float32

	// boxTransform is the Transform applied to the bounding Box,
	// only for [Units] == [ObjectBoundingBox].
	boxTransform mat32.Mat2 `set:"-"`
}

// Stop represents a single stop in a gradient
type Stop struct {

	// the color of the stop. these should be fully opaque,
	// with opacity specified separately, for best results, as is done in SVG etc.
	Color color.Color

	// Opacity is the 0-1 level of opacity for this stop
	Opacity float32

	// the position of the stop between 0 and 1
	Pos float32
}

// OpacityColor returns the stop color with its opacity applied,
// along with a global opacity multiplier
func (st *Stop) OpacityColor(opacity float32) color.Color {
	return colors.ApplyOpacity(st.Color, st.Opacity*opacity)
}

// Spreads are the spread methods used when a gradient reaches
// its end but the object isn't yet fully filled.
type Spreads int32 //enums:enum -transform lower

const (
	// Pad indicates to have the final color of the gradient fill
	// the object beyond the end of the gradient.
	Pad Spreads = iota
	// Reflect indicates to have a gradient repeat in reverse order
	// (offset 1 to 0) to fully fill an object beyond the end of the gradient.
	Reflect
	// Repeat indicates to have a gradient continue in its original order
	// (offset 0 to 1) by jumping back to the start to fully fill an object beyond
	// the end of the gradient.
	Repeat
)

// Units are the types of units used for gradient coordinate values
type Units int32 //enums:enum -transform camel-lower

const (
	// ObjectBoundingBox indicates that coordinate values are scaled
	// relative to the size of the object and are specified in the
	// normalized range of 0 to 1.
	ObjectBoundingBox Units = iota
	// UserSpaceOnUse indicates that coordinate values are specified
	// in the current user coordinate system when the gradient is used
	// (ie: actual SVG/gi coordinates).
	UserSpaceOnUse
)

// AddStop adds a new stop with the given color and position to the gradient.
func (b *Base) AddStop(color color.RGBA, pos float32, opacity ...float32) {
	op := float32(1)
	if len(opacity) > 0 {
		op = opacity[0]
	}
	b.Stops = append(b.Stops, Stop{color, op, pos})
}

// AsBase returns the [Base] of the gradient
func (b *Base) AsBase() *Base {
	return b
}

// NewBase returns a new [Base] with default values. It should
// only be used in the New functions of gradient types.
func NewBase() Base {
	return Base{
		Blend:     colors.RGB, // TODO(kai): figure out a better solution to this
		Box:       mat32.B2(0, 0, 100, 100),
		Transform: mat32.Identity2(),
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
// making new copies of the stops instead of re-using pointers.
// It assumes the gradients are of the same type.
func CopyFrom(g Gradient, cp Gradient) {
	switch g := g.(type) {
	case *Linear:
		*g = *cp.(*Linear)
	case *Radial:
		*g = *cp.(*Radial)
	}
	g.AsBase().CopyStopsFrom(cp.AsBase())
}

// CopyOf returns a copy of the given gradient, making copies of the stops
// instead of re-using pointers.
func CopyOf(g Gradient) Gradient {
	var res Gradient
	switch g := g.(type) {
	case *Linear:
		res = &Linear{}
		CopyFrom(res, g)
	case *Radial:
		res = &Radial{}
		CopyFrom(res, g)
	}
	return res
}

// CopyStopsFrom copies the base gradient stops from the given base gradient
func (b *Base) CopyStopsFrom(cp *Base) {
	b.Stops = make([]Stop, len(cp.Stops))
	copy(b.Stops, cp.Stops)
}

// UpdateBase updates the computed fields of the base gradient. It should only be called
// by other gradient types in their [Gradient.Update] functions. It is named UpdateBase
// to avoid people accidentally calling it instead of [Gradient.Update].
func (b *Base) UpdateBase() {
	b.ComputeObjectMatrix()
}

// ComputeObjectMatrix computes the effective object transformation
// matrix for a gradient with [Units] of [ObjectBoundingBox], setting
// [Base.boxTransform].
func (b *Base) ComputeObjectMatrix() {
	w, h := b.Box.Size().X, b.Box.Size().Y
	oriX, oriY := b.Box.Min.X, b.Box.Min.Y
	b.boxTransform = mat32.Identity2().Translate(oriX, oriY).Scale(w, h).Mul(b.Transform).
		Scale(1/w, 1/h).Translate(-oriX, -oriY).Inverse()
}

// GetColor returns the color at the given normalized position along the
// gradient's stops using its spread method and blend algorithm.
func (b *Base) GetColor(pos float32) color.Color {
	d := len(b.Stops)

	// These cases can be taken care of early on
	if b.Spread == Pad {
		if pos >= 1 {
			return b.Stops[d-1].OpacityColor(b.Opacity)
		}
		if pos <= 0 {
			return b.Stops[0].OpacityColor(b.Opacity)
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
			return b.Stops[0].OpacityColor(b.Opacity)
		case d:
			// Advance to place where mod-1 is greater than the stop indicated by place in reverse of the stop slice.
			// Since this is the reflect b.Spread mode, the mod interval is two, allowing the stop list to be
			// iterated in reverse before repeating the sequence.
			for place != d*2 && mod-1 > (1-b.Stops[d*2-place-1].Pos) {
				place++
			}
			switch place {
			case d:
				return b.Stops[d-1].OpacityColor(b.Opacity)
			case d * 2:
				return b.Stops[0].OpacityColor(b.Opacity)
			default:
				return b.BlendStops(mod-1, b.Stops[d*2-place], b.Stops[d*2-place-1], true)
			}
		default:
			return b.BlendStops(mod, b.Stops[place-1], b.Stops[place], false)
		}
	default: // PadSpread
		switch place {
		case 0:
			return b.Stops[0].OpacityColor(b.Opacity)
		case d:
			return b.Stops[d-1].OpacityColor(b.Opacity)
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
		return s2.OpacityColor(b.Opacity)
	}
	if flip {
		pos = 1 - pos
	}
	tp := (pos - s1off) / (s2.Pos - s1off)

	opacity := (s1.Opacity*(1-tp) + s2.Opacity*tp) * b.Opacity
	return colors.ApplyOpacity(colors.Blend(b.Blend, 100*(1-tp), s1.Color, s2.Color), opacity)
}
