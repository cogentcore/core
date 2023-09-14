// Copyright (c) 2018, The GoKi Authors. All rights reserved.
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
environment.  See https://developer.mozilla.org/en/docs/Web/CSS/length -- 1 px = 1/96 in
Also supporting dp = density-independent pixel = 1/160 in
*/
package units

//go:generate enumgen generate

// borrows from golang.org/x/exp/shiny/unit/ but extends with full range of
// css-based viewport-dependent factors

// standard conversion factors -- Px = DPI-independent pixel instead of actual "dot" raw pixel
const (
	PxPerInch = 96.0
	DpPerInch = 160.0
	MmPerInch = 25.4
	CmPerInch = 2.54
	PtPerInch = 72.0
	PcPerInch = 6.0
)

// Units is an enum that represents a unit (px, em, etc)
type Units int32 //enums:enum

const (
	// UnitPx = pixels -- 1px = 1/96th of 1in -- these are NOT raw display pixels
	UnitPx Units = iota

	// UnitDp = density-independent pixels -- 1dp = 1/160th of 1in
	UnitDp

	// UnitEw = percentage of element width (equivalent to CSS % in some contexts)
	UnitEw

	// UnitEh = percentage of element height (equivalent to CSS % in some contexts)
	UnitEh

	// UnitPw = percentage of parent width (equivalent to CSS % in some contexts)
	UnitPw

	// UnitPh = percentage of parent height (equivalent to CSS % in some contexts)
	UnitPh

	// UnitRem = font size of the root element -- defaults to 12pt scaled by DPI factor
	UnitRem

	// UnitEm = font size of the element -- fallback to 12pt by default
	UnitEm

	// UnitEx = x-height of the element's font (size of 'x' glyph) -- fallback to 0.5em by default
	UnitEx

	// UnitCh = width of the '0' glyph in the element's font -- fallback to 0.5em by default
	UnitCh

	// UnitVw = 1% of the viewport's width
	UnitVw

	// UnitVh = 1% of the viewport's height
	UnitVh

	// UnitVmin = 1% of the viewport's smaller dimension
	UnitVmin

	// UnitVmax = 1% of the viewport's larger dimension
	UnitVmax

	// UnitCm = centimeters -- 1cm = 96px/2.54
	UnitCm

	// UnitMm = millimeters -- 1mm = 1/10th of cm
	UnitMm

	// UnitQ = quarter-millimeters -- 1q = 1/40th of cm
	UnitQ

	// UnitIn = inches -- 1in = 2.54cm = 96px
	UnitIn

	// UnitPc = picas -- 1pc = 1/6th of 1in
	UnitPc

	// UnitPt = points -- 1pt = 1/72th of 1in
	UnitPt

	// UnitDot = actual real display pixels -- generally only use internally
	UnitDot
)

var UnitNames = [...]string{
	UnitPx:   "px",
	UnitDp:   "dp",
	UnitEw:   "ew",
	UnitEh:   "eh",
	UnitPw:   "pw",
	UnitPh:   "ph",
	UnitRem:  "rem",
	UnitEm:   "em",
	UnitEx:   "ex",
	UnitCh:   "ch",
	UnitVw:   "vw",
	UnitVh:   "vh",
	UnitVmin: "vmin",
	UnitVmax: "vmax",
	UnitCm:   "cm",
	UnitMm:   "mm",
	UnitQ:    "q",
	UnitIn:   "in",
	UnitPc:   "pc",
	UnitPt:   "pt",
	UnitDot:  "dot",
}
