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
	"cogentcore.org/core/math32"
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
	Update(opacity float32, box math32.Box2, objTransform math32.Matrix2)
}

// Base contains the data and logic common to all gradient types.
type Base struct { //types:add -setters

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
	Box math32.Box2

	// Transform is the gradient's own transformation matrix applied to the gradient's points.
	// This is a property of the Gradient itself.
	Transform math32.Matrix2

	// Opacity is the overall object opacity multiplier, applied in conjunction with the
	// stop-level opacity blending.
	Opacity float32

	// ApplyFuncs contains functions that are applied to the color after gradient color is generated.
	// This allows for efficient StateLayer and other post-processing effects
	// to be applied.  The Applier handles other cases, but gradients always
	// must have the Update function called at render time, so they must
	// remain Gradient types.
	ApplyFuncs ApplyFuncs `set:"-"`

	// boxTransform is the Transform applied to the bounding Box,
	// only for [Units] == [ObjectBoundingBox].
	boxTransform math32.Matrix2 `set:"-"`

	// stopsRGB are the computed RGB stops for blend types other than RGB
	stopsRGB []Stop `set:"-"`

	// stopsRGBSrc are the source Stops when StopsRGB were last computed
	stopsRGBSrc []Stop `set:"-"`
}

// Stop represents a single stop in a gradient
type Stop struct {

	// Color of the stop. These should be fully opaque,
	// with opacity specified separately, for best results, as is done in SVG etc.
	Color color.RGBA

	// Pos is the position of the stop in normalized units between 0 and 1.
	Pos float32

	// Opacity is the 0-1 level of opacity for this stop.
	Opacity float32
}

// OpacityColor returns the stop color with its opacity applied,
// along with a global opacity multiplier
func (st *Stop) OpacityColor(opacity float32, apply ApplyFuncs) color.Color {
	return apply.Apply(colors.ApplyOpacity(st.Color, st.Opacity*opacity))
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
type Units int32 //enums:enum -transform lower-camel

const (
	// ObjectBoundingBox indicates that coordinate values are scaled
	// relative to the size of the object and are specified in the
	// normalized range of 0 to 1.
	ObjectBoundingBox Units = iota
	// UserSpaceOnUse indicates that coordinate values are specified
	// in the current user coordinate system when the gradient is used
	// (ie: actual SVG/core coordinates).
	UserSpaceOnUse
)

// AddStop adds a new stop with the given color, position, and
// optional opacity to the gradient.
func (b *Base) AddStop(color color.RGBA, pos float32, opacity ...float32) *Base {
	op := float32(1)
	if len(opacity) > 0 {
		op = opacity[0]
	}
	b.Stops = append(b.Stops, Stop{Color: color, Pos: pos, Opacity: op})
	return b
}

// AsBase returns the [Base] of the gradient
func (b *Base) AsBase() *Base {
	return b
}

// NewBase returns a new [Base] with default values. It should
// only be used in the New functions of gradient types.
func NewBase() Base {
	return Base{
		Blend:     colors.RGB,
		Box:       math32.B2(0, 0, 100, 100),
		Opacity:   1,
		Transform: math32.Identity2(),
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
	cb := cp.AsBase()
	gb := g.AsBase()
	gb.CopyStopsFrom(cb)
	gb.ApplyFuncs = cb.ApplyFuncs.Clone()
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

	if cp.stopsRGB == nil {
		b.stopsRGB = nil
		b.stopsRGBSrc = nil
	} else {
		b.stopsRGB = make([]Stop, len(cp.stopsRGB))
		copy(b.stopsRGB, cp.stopsRGB)
		b.stopsRGBSrc = make([]Stop, len(cp.stopsRGBSrc))
		copy(b.stopsRGBSrc, cp.stopsRGBSrc)
	}
}

// ApplyOpacityToStops multiplies all stop opacities by the given opacity.
func (b *Base) ApplyOpacityToStops(opacity float32) {
	for _, s := range b.Stops {
		s.Opacity *= opacity
	}
	for _, s := range b.stopsRGB {
		s.Opacity *= opacity
	}
	for _, s := range b.stopsRGBSrc {
		s.Opacity *= opacity
	}
}

// updateBase updates the computed fields of the base gradient. It should only be called
// by other gradient types in their [Gradient.Update] functions. It is named updateBase
// to avoid people accidentally calling it instead of [Gradient.Update].
func (b *Base) updateBase() {
	b.computeObjectMatrix()
	b.updateRGBStops()
}

// computeObjectMatrix computes the effective object transformation
// matrix for a gradient with [Units] of [ObjectBoundingBox], setting
// [Base.boxTransform].
func (b *Base) computeObjectMatrix() {
	w, h := b.Box.Size().X, b.Box.Size().Y
	oriX, oriY := b.Box.Min.X, b.Box.Min.Y
	b.boxTransform = math32.Identity2().Translate(oriX, oriY).Scale(w, h).Mul(b.Transform).
		Scale(1/w, 1/h).Translate(-oriX, -oriY).Inverse()
}

// getColor returns the color at the given normalized position along the
// gradient's stops using its spread method and blend algorithm.
func (b *Base) getColor(pos float32) color.Color {
	if b.Blend == colors.RGB {
		return b.getColorImpl(pos, b.Stops)
	}
	return b.getColorImpl(pos, b.stopsRGB)
}

// getColorImpl implements [Base.getColor] with given stops
func (b *Base) getColorImpl(pos float32, stops []Stop) color.Color {
	d := len(stops)
	// These cases can be taken care of early on
	if b.Spread == Pad {
		if pos >= 1 {
			return stops[d-1].OpacityColor(b.Opacity, b.ApplyFuncs)
		}
		if pos <= 0 {
			return stops[0].OpacityColor(b.Opacity, b.ApplyFuncs)
		}
	}

	modRange := float32(1)
	if b.Spread == Reflect {
		modRange = 2
	}
	mod := math32.Mod(pos, modRange)
	if mod < 0 {
		mod += modRange
	}

	place := 0 // Advance to place where mod is greater than the indicated stop
	for place != len(stops) && mod > stops[place].Pos {
		place++
	}
	switch b.Spread {
	case Repeat:
		var s1, s2 Stop
		switch place {
		case 0, d:
			s1, s2 = stops[d-1], stops[0]
		default:
			s1, s2 = stops[place-1], stops[place]
		}
		return b.blendStops(mod, s1, s2, false)
	case Reflect:
		switch place {
		case 0:
			return stops[0].OpacityColor(b.Opacity, b.ApplyFuncs)
		case d:
			// Advance to place where mod-1 is greater than the stop indicated by place in reverse of the stop slice.
			// Since this is the reflect b.Spread mode, the mod interval is two, allowing the stop list to be
			// iterated in reverse before repeating the sequence.
			for place != d*2 && mod-1 > (1-stops[d*2-place-1].Pos) {
				place++
			}
			switch place {
			case d:
				return stops[d-1].OpacityColor(b.Opacity, b.ApplyFuncs)
			case d * 2:
				return stops[0].OpacityColor(b.Opacity, b.ApplyFuncs)
			default:
				return b.blendStops(mod-1, stops[d*2-place], stops[d*2-place-1], true)
			}
		default:
			return b.blendStops(mod, stops[place-1], stops[place], false)
		}
	default: // PadSpread
		switch place {
		case 0:
			return stops[0].OpacityColor(b.Opacity, b.ApplyFuncs)
		case d:
			return stops[d-1].OpacityColor(b.Opacity, b.ApplyFuncs)
		default:
			return b.blendStops(mod, stops[place-1], stops[place], false)
		}
	}
}

// blendStops blends the given two gradient stops together based on the given position,
// using the gradient's blending algorithm. If flip is true, it flips the given position.
func (b *Base) blendStops(pos float32, s1, s2 Stop, flip bool) color.Color {
	s1off := s1.Pos
	if s1.Pos > s2.Pos && !flip { // happens in repeat spread mode
		s1off--
		if pos > 1 {
			pos--
		}
	}
	if s2.Pos == s1off {
		return s2.OpacityColor(b.Opacity, b.ApplyFuncs)
	}
	if flip {
		pos = 1 - pos
	}
	tp := (pos - s1off) / (s2.Pos - s1off)

	opacity := (s1.Opacity*(1-tp) + s2.Opacity*tp) * b.Opacity
	return b.ApplyFuncs.Apply(colors.ApplyOpacity(colors.Blend(colors.RGB, 100*(1-tp), s1.Color, s2.Color), opacity))
}

// updateRGBStops updates stopsRGB from original Stops, for other blend types
func (b *Base) updateRGBStops() {
	if b.Blend == colors.RGB || len(b.Stops) == 0 {
		b.stopsRGB = nil
		b.stopsRGBSrc = nil
		return
	}
	n := len(b.Stops)
	lenEq := false
	if len(b.stopsRGBSrc) == n {
		lenEq = true
		equal := true
		for i := range b.Stops {
			if b.Stops[i] != b.stopsRGBSrc[i] {
				equal = false
				break
			}
		}
		if equal {
			return
		}
	}

	if !lenEq {
		b.stopsRGBSrc = make([]Stop, n)
	}
	copy(b.stopsRGBSrc, b.Stops)

	b.stopsRGB = make([]Stop, 0, n*4)

	tdp := float32(0.05)
	b.stopsRGB = append(b.stopsRGB, b.Stops[0])
	for i := 0; i < n-1; i++ {
		sp := b.Stops[i]
		s := b.Stops[i+1]
		dp := s.Pos - sp.Pos
		np := int(math32.Ceil(dp / tdp))
		if np == 1 {
			b.stopsRGB = append(b.stopsRGB, s)
			continue
		}
		pct := float32(1) / float32(np)
		dopa := s.Opacity - sp.Opacity
		for j := 0; j < np; j++ {
			p := pct * float32(j)
			c := colors.Blend(colors.RGB, 100*p, s.Color, sp.Color)
			pos := sp.Pos + p*dp
			opa := sp.Opacity + p*dopa
			b.stopsRGB = append(b.stopsRGB, Stop{Color: c, Pos: pos, Opacity: opa})
		}
		b.stopsRGB = append(b.stopsRGB, s)
	}
}
