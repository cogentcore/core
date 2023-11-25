// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image/color"

	"goki.dev/colors"
	"goki.dev/girl/units"
	"goki.dev/mat32/v2"
)

// note: background-color is in FontStyle as it is needed to make that the
// only style needed for text render styling

// // Background has style parameters for backgrounds
// type Background struct {
// 	// todo: all the properties not yet implemented -- mostly about images
// 	// Image is like a PaintServer -- includes gradients etc
// 	// Attachment -- how the image moves
// 	// Clip -- how to clip the image
// 	// Origin
// 	// Position
// 	// Repeat
// 	// Size
// }

// func (b *Background) Defaults() {
// 	b.Color.SetColor(White)
// }

// BorderStyles determines how to draw the border
type BorderStyles int32 //enums:enum -trim-prefix Border

const (
	BorderSolid BorderStyles = iota
	BorderDotted
	BorderDashed
	BorderDouble
	BorderGroove
	BorderRidge
	BorderInset
	BorderOutset
	BorderNone
	BorderHidden
)

// IMPORTANT: any changes here must be updated in style_props.go StyleBorderFuncs

// Border contains style parameters for borders
type Border struct { //gti:add

	// prop: border-style = how to draw the border
	Style Sides[BorderStyles] `xml:"style"`

	// prop: border-width = width of the border
	Width SideValues `xml:"width" view:"inline"`

	// prop: border-radius = rounding of the corners
	Radius SideValues `xml:"radius" view:"inline"`

	// prop: border-color = color of the border
	Color SideColors `xml:"color" view:"inline"`
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (bs *Border) ToDots(uc *units.Context) {
	bs.Width.ToDots(uc)
	bs.Radius.ToDots(uc)
}

// Pre-configured border radius values, based on
// https://m3.material.io/styles/shape/shape-scale-tokens
var (
	// BorderRadiusNone indicates to use no border radius,
	// which creates a fully rectangular element
	BorderRadiusNone = NewSideValues(units.Zero())
	// BorderRadiusExtraSmall indicates to use extra small
	// 4dp rounded corners
	BorderRadiusExtraSmall = NewSideValues(units.Dp(4))
	// BorderRadiusExtraSmallTop indicates to use extra small
	// 4dp rounded corners on the top of the element and no
	// border radius on the bottom of the element
	BorderRadiusExtraSmallTop = NewSideValues(units.Dp(4), units.Dp(4), units.Zero(), units.Zero())
	// BorderRadiusSmall indicates to use small
	// 8dp rounded corners
	BorderRadiusSmall = NewSideValues(units.Dp(8))
	// BorderRadiusMedium indicates to use medium
	// 12dp rounded corners
	BorderRadiusMedium = NewSideValues(units.Dp(12))
	// BorderRadiusLarge indicates to use large
	// 16dp rounded corners
	BorderRadiusLarge = NewSideValues(units.Dp(16))
	// BorderRadiusLargeEnd indicates to use large
	// 16dp rounded corners on the end (right side)
	// of the element and no border radius elsewhere
	BorderRadiusLargeEnd = NewSideValues(units.Zero(), units.Dp(16), units.Dp(16), units.Zero())
	// BorderRadiusLargeTop indicates to use large
	// 16dp rounded corners on the top of the element
	// and no border radius on the bottom of the element
	BorderRadiusLargeTop = NewSideValues(units.Dp(16), units.Dp(16), units.Zero(), units.Zero())
	// BorderRadiusExtraLarge indicates to use extra large
	// 28dp rounded corners
	BorderRadiusExtraLarge = NewSideValues(units.Dp(28))
	// BorderRadiusExtraLargeTop indicates to use extra large
	// 28dp rounded corners on the top of the element
	// and no border radius on the bottom of the element
	BorderRadiusExtraLargeTop = NewSideValues(units.Dp(28), units.Dp(28), units.Zero(), units.Zero())
	// BorderRadiusFull indicates to use a full border radius,
	// which creates a circular/pill-shaped object.
	// It is defined to be a value that the width/height of an object
	// will never exceed.
	BorderRadiusFull = NewSideValues(units.Dp(1_000_000_000))
)

// IMPORTANT: any changes here must be updated in style_props.go StyleShadowFuncs

// style parameters for shadows
type Shadow struct { //gti:add

	// prop: .h-offset = horizontal offset of shadow -- positive = right side, negative = left side
	HOffset units.Value `xml:".h-offset"`

	// prop: .v-offset = vertical offset of shadow -- positive = below, negative = above
	VOffset units.Value `xml:".v-offset"`

	// prop: .blur = blur radius -- higher numbers = more blurry
	Blur units.Value `xml:".blur"`

	// prop: .spread = spread radius -- positive number increases size of shadow, negative decreases size
	Spread units.Value `xml:".spread"`

	// prop: .color = color of the shadow
	Color color.RGBA `xml:".color"`

	// prop: .inset = shadow is inset within box instead of outset outside of box
	Inset bool `xml:".inset"`
}

func (s *Shadow) HasShadow() bool {
	return s.HOffset.Dots != 0 || s.VOffset.Dots != 0 || s.Blur.Dots != 0 || s.Spread.Dots != 0
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (s *Shadow) ToDots(uc *units.Context) {
	s.HOffset.ToDots(uc)
	s.VOffset.ToDots(uc)
	s.Blur.ToDots(uc)
	s.Spread.ToDots(uc)
}

// BasePos returns the position at which the base box shadow
// (the actual solid, unblurred box part) should be rendered
// if the shadow is on an element with the given starting position.
func (s *Shadow) BasePos(startPos mat32.Vec2) mat32.Vec2 {
	// Offset directly affects position.
	// We need to subtract spread
	// to compensate for size changes and stay centered.
	return startPos.Add(mat32.NewVec2(s.HOffset.Dots, s.VOffset.Dots)).SubScalar(s.Spread.Dots)
}

// BaseSize returns the total size the base box shadow
// (the actual solid, unblurred part) should be if
// the shadow is on an element with the given starting size.
func (s *Shadow) BaseSize(startSize mat32.Vec2) mat32.Vec2 {
	// Spread goes on all sides, so need to count twice per dimension.
	return startSize.AddScalar(2 * s.Spread.Dots)
}

// Pos returns the position at which the blurred box shadow
// should start if the shadow is on an element
// with the given starting position.
func (s *Shadow) Pos(startPos mat32.Vec2) mat32.Vec2 {
	// We need to subtract half of blur
	// to compensate for size changes and stay centered.
	return s.BasePos(startPos).SubScalar(s.Blur.Dots / 2)
}

// Size returns the total size occupied by the blurred box shadow
// if the shadow is on an element with the given starting size.
func (s *Shadow) Size(startSize mat32.Vec2) mat32.Vec2 {
	// Blur goes on all sides, but it is rendered as half of actual
	// because CSS does the same, so we only count it once.
	return s.BaseSize(startSize).AddScalar(s.Blur.Dots)
}

// Margin returns the effective margin created by the
// shadow on each side in terms of raw display dots.
// It should be added to margin for sizing considerations.
func (s *Shadow) Margin() SideFloats {
	// Spread benefits every side.
	// Offset goes either way, depending on side.
	// Every side must be positive.

	return NewSideFloats(
		mat32.Max(s.Spread.Dots-s.VOffset.Dots+s.Blur.Dots/2, 0),
		mat32.Max(s.Spread.Dots+s.HOffset.Dots+s.Blur.Dots/2, 0),
		mat32.Max(s.Spread.Dots+s.VOffset.Dots+s.Blur.Dots/2, 0),
		mat32.Max(s.Spread.Dots-s.HOffset.Dots+s.Blur.Dots/2, 0),
	)
}

// AddBoxShadow adds the given box shadows to the style
func (s *Style) AddBoxShadow(shadow ...Shadow) {
	if s.BoxShadow == nil {
		s.BoxShadow = []Shadow{}
	}
	s.BoxShadow = append(s.BoxShadow, shadow...)
}

// BoxShadowStartPos returns the position and size of the
// area in which all of the box shadows are rendered, using
// [Shadow.Pos] and [Shadow.Size]. It should be used as the
// bounds to clear to prevent growing shadows.
func (s *Style) BoxShadowPosSize(startPos, startSize mat32.Vec2) (pos mat32.Vec2, sz mat32.Vec2) {
	// Need to think in terms of min/max bounds
	// to get accurate pos and size if the shadow
	// with the biggest size does not have the smallest pos
	minPos := startPos
	maxMax := startPos.Add(startSize) // max upper (max) bound
	for _, sh := range s.BoxShadow {
		curPos := sh.Pos(startPos)
		curSz := sh.Size(startSize)

		minPos = minPos.Min(curPos)
		maxMax = maxMax.Max(curPos.Add(curSz))
	}
	return minPos, maxMax.Sub(minPos)
}

// BoxShadowMargin returns the effective box
// shadow margin of the style, calculated through [Shadow.Margin]
func (s *Style) BoxShadowMargin() SideFloats {
	return BoxShadowMargin(s.BoxShadow)
}

// MaxBoxShadowMargin returns the maximum effective box
// shadow margin of the style, calculated through [Shadow.Margin]
func (s *Style) MaxBoxShadowMargin() SideFloats {
	return BoxShadowMargin(s.MaxBoxShadow)
}

// BoxShadowMargin returns the maximum effective box shadow margin
// of the given box shadows, calculated through [Shadow.Margin].
func BoxShadowMargin(shadows []Shadow) SideFloats {
	max := SideFloats{}
	for _, sh := range shadows {
		max = max.Max(sh.Margin())
	}
	return max
}

// BoxShadowToDots runs ToDots on all box shadow
// unit values to compile down to raw pixels
func (s *Style) BoxShadowToDots(uc *units.Context) {
	for i := range s.BoxShadow {
		s.BoxShadow[i].ToDots(uc)
	}
	for i := range s.MaxBoxShadow {
		s.MaxBoxShadow[i].ToDots(uc)
	}
}

// HasBoxShadow returns whether the style has
// any box shadows
func (s *Style) HasBoxShadow() bool {
	for _, sh := range s.BoxShadow {
		if sh.HasShadow() {
			return true
		}
	}
	return false
}

// Pre-configured box shadow values, based on
// those in Material 3.

// BoxShadow0 returns the shadows
// to be used on Elevation 0 elements.
// There are no shadows part of BoxShadow0,
// so applying it is purely semantic.
func BoxShadow0() []Shadow {
	return []Shadow{}
}

// BoxShadow1 contains the shadows
// to be used on Elevation 1 elements.
func BoxShadow1() []Shadow {
	return []Shadow{
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(3),
			Blur:    units.Dp(1),
			Spread:  units.Dp(-2),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.2),
		},
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(2),
			Blur:    units.Dp(2),
			Spread:  units.Zero(),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.14),
		},
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(1),
			Blur:    units.Dp(5),
			Spread:  units.Zero(),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
		},
	}
}

// BoxShadow2 returns the shadows
// to be used on Elevation 2 elements.
func BoxShadow2() []Shadow {
	return []Shadow{
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(2),
			Blur:    units.Dp(4),
			Spread:  units.Dp(-1),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.2),
		},
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(4),
			Blur:    units.Dp(5),
			Spread:  units.Zero(),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.14),
		},
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(1),
			Blur:    units.Dp(10),
			Spread:  units.Zero(),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
		},
	}
}

// TODO: figure out why 3 and 4 are the same

// BoxShadow3 returns the shadows
// to be used on Elevation 3 elements.
func BoxShadow3() []Shadow {
	return []Shadow{
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(5),
			Blur:    units.Dp(5),
			Spread:  units.Dp(-3),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.2),
		},
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(8),
			Blur:    units.Dp(10),
			Spread:  units.Dp(1),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.14),
		},
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(3),
			Blur:    units.Dp(14),
			Spread:  units.Dp(2),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
		},
	}
}

// BoxShadow4 returns the shadows
// to be used on Elevation 4 elements.
func BoxShadow4() []Shadow {
	return []Shadow{
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(5),
			Blur:    units.Dp(5),
			Spread:  units.Dp(-3),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.2),
		},
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(8),
			Blur:    units.Dp(10),
			Spread:  units.Dp(1),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.14),
		},
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(3),
			Blur:    units.Dp(14),
			Spread:  units.Dp(2),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
		},
	}
}

// BoxShadow5 returns the shadows
// to be used on Elevation 5 elements.
func BoxShadow5() []Shadow {
	return []Shadow{
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(8),
			Blur:    units.Dp(10),
			Spread:  units.Dp(-6),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.2),
		},
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(16),
			Blur:    units.Dp(24),
			Spread:  units.Dp(2),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.14),
		},
		{
			HOffset: units.Zero(),
			VOffset: units.Dp(6),
			Blur:    units.Dp(30),
			Spread:  units.Dp(5),
			Color:   colors.SetAF32(colors.Scheme.Shadow, 0.12),
		},
	}
}
