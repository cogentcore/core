// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package Units supports full range of CSS-style length units (em, px, dp, etc)

The unit is stored along with a value, and can be converted at a later point into
a raw display pixel value using the Context which contains all the necessary reference
values to perform the conversion.  Typically the unit value is parsed early from a style
and then converted later once the context is fully resolved.  The Value also holds the
converted value (Dots) so it can be used directly without further re-conversion.

'Dots' are used as term for underlying raw display pixels because "Pixel" and the px unit
are actually not conventionally used as raw display pixels in the current HiDPI
environment.  See https://developer.mozilla.org/en/docs/Web/CSS/length: 1 px = 1/96 in
Also supporting dp = density-independent pixel = 1/160 in
*/
package units

//go:generate core generate
//go:generate go run gen.go

// borrows from golang.org/x/exp/shiny/unit but extends with full range of
// css-based viewport-dependent factors

// Standard conversion factors. Px is a logical, DPI-independent pixel instead of an actual "dot" raw pixel.
const (
	PxPerInch = 96
	DpPerInch = 160
	MmPerInch = 25.4
	CmPerInch = 2.54
	PtPerInch = 72
	PcPerInch = 6
)

// Units is an enum that represents a unit (dp, px, em, etc).
// See the website documentation on units for more information.
type Units int32 //enums:enum -trim-prefix Unit -transform lower

const (
	// UnitDp represents density-independent pixels. 1dp is 1/160 in.
	// Inches are not necessarily the same as actual physical inches, as they
	// depend on the DPI, so dp values may correspond to different physical sizes
	// on different displays, but they will look correct.
	UnitDp Units = iota

	// UnitPx represents logical pixels. 1px is 1/96 in.
	// These are not raw display pixels, for which you should use dots.
	// Dp is a more common unit for general use.
	UnitPx

	// UnitEw represents percentage of element width, which is equivalent to CSS % in some contexts.
	UnitEw

	// UnitEh represents percentage of element height, which is equivalent to CSS % in some contexts.
	UnitEh

	// UnitPw represents percentage of parent width, which is equivalent to CSS % in some contexts.
	UnitPw

	// UnitPh represents percentage of parent height, which is equivalent to CSS % in some contexts.
	UnitPh

	// NOTE: rem must go before em for parsing order to work

	// UnitRem represents the font size of the root element, which is always 16dp.
	UnitRem

	// UnitEm represents the font size of the element.
	UnitEm

	// UnitEx represents x-height of the element's font (size of 'x' glyph).
	// It falls back to a default of 0.5em.
	UnitEx

	// UnitCh represents width of the '0' glyph in the element's font.
	// It falls back to a default of 0.5em.
	UnitCh

	// UnitVw represents percentage of viewport (Scene) width.
	UnitVw

	// UnitVh represents percentage of viewport (Scene) height.
	UnitVh

	// UnitVmin represents percentage of the smaller dimension of the viewport (Scene).
	UnitVmin

	// UnitVmax represents percentage of the larger dimension of the viewport (Scene).
	UnitVmax

	// UnitCm represents logical centimeters. 1cm is 1/2.54 in.
	UnitCm

	// UnitMm represents logical millimeters. 1mm is 1/10 cm.
	UnitMm

	// UnitQ represents logical quarter-millimeters. 1q is 1/40 cm.
	UnitQ

	// UnitIn represents logical inches. 1in is 2.54cm or 96px.
	// This is similar to CSS inches in that it is not necessarily the same
	// as an actual physical inch; it is dependent on the DPI of the display.
	UnitIn

	// UnitPc represents logical picas. 1pc is 1/6 in.
	UnitPc

	// UnitPt represents points. 1pt is 1/72 in.
	UnitPt

	// UnitDot represents real display pixels. They are generally only used internally.
	UnitDot
)
