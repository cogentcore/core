// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package styles

import (
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/styles/units"
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
type BorderStyles int32 //enums:enum -trim-prefix Border -transform kebab

const (
	// BorderSolid indicates to render a solid border.
	BorderSolid BorderStyles = iota

	// BorderDotted indicates to render a dotted border.
	BorderDotted

	// BorderDashed indicates to render a dashed border.
	BorderDashed

	// TODO(kai): maybe implement these at some point if there
	// is ever an actual use case for them

	// BorderDouble is not currently supported.
	BorderDouble

	// BorderGroove is not currently supported.
	BorderGroove

	// BorderRidge is not currently supported.
	BorderRidge

	// BorderInset is not currently supported.
	BorderInset

	// BorderOutset is not currently supported.
	BorderOutset

	// BorderNone indicates to render no border.
	BorderNone
)

// IMPORTANT: any changes here must be updated in style_properties.go StyleBorderFuncs

// Border contains style parameters for borders
type Border struct { //types:add

	// Style specifies how to draw the border
	Style sides.Sides[BorderStyles]

	// Width specifies the width of the border
	Width sides.Values `display:"inline"`

	// Radius specifies the radius (rounding) of the corners
	Radius sides.Values `display:"inline"`

	// Offset specifies how much, if any, the border is offset
	// from its element. It is only applicable in the standard
	// box model, which is used by [paint.Painter.DrawStdBox] and
	// all standard GUI elements.
	Offset sides.Values `display:"inline"`

	// Color specifies the color of the border
	Color sides.Sides[image.Image] `display:"inline"`
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (bs *Border) ToDots(uc *units.Context) {
	bs.Width.ToDots(uc)
	bs.Radius.ToDots(uc)
	bs.Offset.ToDots(uc)
}

// Pre-configured border radius values, based on
// https://m3.material.io/styles/shape/shape-scale-tokens
var (
	// BorderRadiusExtraSmall indicates to use extra small
	// 4dp rounded corners
	BorderRadiusExtraSmall = sides.NewValues(units.Dp(4))

	// BorderRadiusExtraSmallTop indicates to use extra small
	// 4dp rounded corners on the top of the element and no
	// border radius on the bottom of the element
	BorderRadiusExtraSmallTop = sides.NewValues(units.Dp(4), units.Dp(4), units.Zero(), units.Zero())

	// BorderRadiusSmall indicates to use small
	// 8dp rounded corners
	BorderRadiusSmall = sides.NewValues(units.Dp(8))

	// BorderRadiusMedium indicates to use medium
	// 12dp rounded corners
	BorderRadiusMedium = sides.NewValues(units.Dp(12))

	// BorderRadiusLarge indicates to use large
	// 16dp rounded corners
	BorderRadiusLarge = sides.NewValues(units.Dp(16))

	// BorderRadiusLargeEnd indicates to use large
	// 16dp rounded corners on the end (right side)
	// of the element and no border radius elsewhere
	BorderRadiusLargeEnd = sides.NewValues(units.Zero(), units.Dp(16), units.Dp(16), units.Zero())

	// BorderRadiusLargeTop indicates to use large
	// 16dp rounded corners on the top of the element
	// and no border radius on the bottom of the element
	BorderRadiusLargeTop = sides.NewValues(units.Dp(16), units.Dp(16), units.Zero(), units.Zero())

	// BorderRadiusExtraLarge indicates to use extra large
	// 28dp rounded corners
	BorderRadiusExtraLarge = sides.NewValues(units.Dp(28))

	// BorderRadiusExtraLargeTop indicates to use extra large
	// 28dp rounded corners on the top of the element
	// and no border radius on the bottom of the element
	BorderRadiusExtraLargeTop = sides.NewValues(units.Dp(28), units.Dp(28), units.Zero(), units.Zero())

	// BorderRadiusFull indicates to use a full border radius,
	// which creates a circular/pill-shaped object.
	// It is defined to be a value that the width/height of an object
	// will never exceed.
	BorderRadiusFull = sides.NewValues(units.Dp(1_000_000_000))
)

// IMPORTANT: any changes here must be updated in style_properties.go StyleShadowFuncs

// style parameters for shadows
type Shadow struct { //types:add

	// OffsetX is th horizontal offset of the shadow.
	// Positive moves it right, negative moves it left.
	OffsetX units.Value

	// OffsetY is the vertical offset of the shadow.
	// Positive moves it down, negative moves it up.
	OffsetY units.Value

	// Blur specifies the blur radius of the shadow.
	// Higher numbers make it more blurry.
	Blur units.Value

	// Spread specifies the spread radius of the shadow.
	// Positive numbers increase the size of the shadow,
	// and negative numbers decrease the size.
	Spread units.Value

	// Color specifies the color of the shadow.
	Color image.Image

	// Inset specifies whether the shadow is inset within the
	// box instead of outset outside of the box.
	// TODO: implement.
	Inset bool
}

func (s *Shadow) HasShadow() bool {
	return s.OffsetX.Dots != 0 || s.OffsetY.Dots != 0 || s.Blur.Dots != 0 || s.Spread.Dots != 0
}

// ToDots runs ToDots on unit values, to compile down to raw pixels
func (s *Shadow) ToDots(uc *units.Context) {
	s.OffsetX.ToDots(uc)
	s.OffsetY.ToDots(uc)
	s.Blur.ToDots(uc)
	s.Spread.ToDots(uc)
}

// BasePos returns the position at which the base box shadow
// (the actual solid, unblurred box part) should be rendered
// if the shadow is on an element with the given starting position.
func (s *Shadow) BasePos(startPos math32.Vector2) math32.Vector2 {
	// Offset directly affects position.
	// We need to subtract spread
	// to compensate for size changes and stay centered.
	return startPos.Add(math32.Vec2(s.OffsetX.Dots, s.OffsetY.Dots)).SubScalar(s.Spread.Dots)
}

// BaseSize returns the total size the base box shadow
// (the actual solid, unblurred part) should be if
// the shadow is on an element with the given starting size.
func (s *Shadow) BaseSize(startSize math32.Vector2) math32.Vector2 {
	// Spread goes on all sides, so need to count twice per dimension.
	return startSize.AddScalar(2 * s.Spread.Dots)
}

// Pos returns the position at which the blurred box shadow
// should start if the shadow is on an element
// with the given starting position.
func (s *Shadow) Pos(startPos math32.Vector2) math32.Vector2 {
	// We need to subtract half of blur
	// to compensate for size changes and stay centered.
	return s.BasePos(startPos).SubScalar(s.Blur.Dots / 2)
}

// Size returns the total size occupied by the blurred box shadow
// if the shadow is on an element with the given starting size.
func (s *Shadow) Size(startSize math32.Vector2) math32.Vector2 {
	// Blur goes on all sides, but it is rendered as half of actual
	// because CSS does the same, so we only count it once.
	return s.BaseSize(startSize).AddScalar(s.Blur.Dots)
}

// Margin returns the effective margin created by the
// shadow on each side in terms of raw display dots.
// It should be added to margin for sizing considerations.
func (s *Shadow) Margin() sides.Floats {
	// Spread benefits every side.
	// Offset goes either way, depending on side.
	// Every side must be positive.

	// note: we are using EdgeBlurFactors with radiusFactor = 1
	// (sigma == radius), so we divide Blur / 2 relative to the
	// CSS standard of sigma = blur / 2 (i.e., our sigma = blur,
	// so we divide Blur / 2 to achieve the same effect).
	// This works fine for low-opacity blur factors (the edges are
	// so transparent that you can't really see beyond 1 sigma if
	// you used radiusFactor = 2).
	// If a higher-contrast shadow is used, it would look better
	// with radiusFactor = 2, and you'd have to remove this /2 factor.

	sdots := float32(0)
	if s.Blur.Dots > 0 {
		sdots = math32.Ceil(0.5 * s.Blur.Dots)
		if sdots < 2 { // for tight dp = 1 case, the render antialiasing requires a min width..
			sdots = 2
		}
	}

	return sides.NewFloats(
		math32.Max(s.Spread.Dots-s.OffsetY.Dots+sdots, 0),
		math32.Max(s.Spread.Dots+s.OffsetX.Dots+sdots, 0),
		math32.Max(s.Spread.Dots+s.OffsetY.Dots+sdots, 0),
		math32.Max(s.Spread.Dots-s.OffsetX.Dots+sdots, 0),
	)
}

// AddBoxShadow adds the given box shadows to the style
func (s *Style) AddBoxShadow(shadow ...Shadow) {
	if s.BoxShadow == nil {
		s.BoxShadow = []Shadow{}
	}
	s.BoxShadow = append(s.BoxShadow, shadow...)
}

// BoxShadowMargin returns the effective box
// shadow margin of the style, calculated through [Shadow.Margin]
func (s *Style) BoxShadowMargin() sides.Floats {
	return BoxShadowMargin(s.BoxShadow)
}

// MaxBoxShadowMargin returns the maximum effective box
// shadow margin of the style, calculated through [Shadow.Margin]
func (s *Style) MaxBoxShadowMargin() sides.Floats {
	return BoxShadowMargin(s.MaxBoxShadow)
}

// BoxShadowMargin returns the maximum effective box shadow margin
// of the given box shadows, calculated through [Shadow.Margin].
func BoxShadowMargin(shadows []Shadow) sides.Floats {
	max := sides.Floats{}
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
			OffsetX: units.Zero(),
			OffsetY: units.Dp(3),
			Blur:    units.Dp(1),
			Spread:  units.Dp(-2),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.2),
		},
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(2),
			Blur:    units.Dp(2),
			Spread:  units.Zero(),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.14),
		},
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(1),
			Blur:    units.Dp(5),
			Spread:  units.Zero(),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.12),
		},
	}
}

// BoxShadow2 returns the shadows
// to be used on Elevation 2 elements.
func BoxShadow2() []Shadow {
	return []Shadow{
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(2),
			Blur:    units.Dp(4),
			Spread:  units.Dp(-1),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.2),
		},
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(4),
			Blur:    units.Dp(5),
			Spread:  units.Zero(),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.14),
		},
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(1),
			Blur:    units.Dp(10),
			Spread:  units.Zero(),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.12),
		},
	}
}

// TODO: figure out why 3 and 4 are the same

// BoxShadow3 returns the shadows
// to be used on Elevation 3 elements.
func BoxShadow3() []Shadow {
	return []Shadow{
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(5),
			Blur:    units.Dp(5),
			Spread:  units.Dp(-3),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.2),
		},
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(8),
			Blur:    units.Dp(10),
			Spread:  units.Dp(1),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.14),
		},
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(3),
			Blur:    units.Dp(14),
			Spread:  units.Dp(2),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.12),
		},
	}
}

// BoxShadow4 returns the shadows
// to be used on Elevation 4 elements.
func BoxShadow4() []Shadow {
	return []Shadow{
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(5),
			Blur:    units.Dp(5),
			Spread:  units.Dp(-3),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.2),
		},
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(8),
			Blur:    units.Dp(10),
			Spread:  units.Dp(1),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.14),
		},
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(3),
			Blur:    units.Dp(14),
			Spread:  units.Dp(2),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.12),
		},
	}
}

// BoxShadow5 returns the shadows
// to be used on Elevation 5 elements.
func BoxShadow5() []Shadow {
	return []Shadow{
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(8),
			Blur:    units.Dp(10),
			Spread:  units.Dp(-6),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.2),
		},
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(16),
			Blur:    units.Dp(24),
			Spread:  units.Dp(2),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.14),
		},
		{
			OffsetX: units.Zero(),
			OffsetY: units.Dp(6),
			Blur:    units.Dp(30),
			Spread:  units.Dp(5),
			Color:   gradient.ApplyOpacity(colors.Scheme.Shadow, 0.12),
		},
	}
}
