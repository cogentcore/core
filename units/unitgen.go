// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

// Px returns a new px value:
// UnitPx = pixels -- 1px = 1/96th of 1in -- these are NOT raw display pixels
func Px(val float32) Value {
	return Value{Val: val, Un: UnitPx}
}

// Dp returns a new dp value:
// UnitDp = density-independent pixels -- 1dp = 1/160th of 1in
func Dp(val float32) Value {
	return Value{Val: val, Un: UnitDp}
}

// Ew returns a new ew value:
// UnitEw = percentage of element width (equivalent to CSS % in some contexts)
func Ew(val float32) Value {
	return Value{Val: val, Un: UnitEw}
}

// Eh returns a new eh value:
// UnitEh = percentage of element height (equivalent to CSS % in some contexts)
func Eh(val float32) Value {
	return Value{Val: val, Un: UnitEh}
}

// Pw returns a new pw value:
// UnitPw = percentage of parent width (equivalent to CSS % in some contexts)
func Pw(val float32) Value {
	return Value{Val: val, Un: UnitPw}
}

// Ph returns a new ph value:
// UnitPh = percentage of parent height (equivalent to CSS % in some contexts)
func Ph(val float32) Value {
	return Value{Val: val, Un: UnitPh}
}

// Rem returns a new rem value:
// UnitRem = font size of the root element -- defaults to 12pt scaled by DPI factor
func Rem(val float32) Value {
	return Value{Val: val, Un: UnitRem}
}

// Em returns a new em value:
// UnitEm = font size of the element -- fallback to 12pt by default
func Em(val float32) Value {
	return Value{Val: val, Un: UnitEm}
}

// Ex returns a new ex value:
// UnitEx = x-height of the element&#39;s font (size of &#39;x&#39; glyph) -- fallback to 0.5em by default
func Ex(val float32) Value {
	return Value{Val: val, Un: UnitEx}
}

// Ch returns a new ch value:
// UnitCh = width of the &#39;0&#39; glyph in the element&#39;s font -- fallback to 0.5em by default
func Ch(val float32) Value {
	return Value{Val: val, Un: UnitCh}
}

// Vw returns a new vw value:
// UnitVw = 1% of the viewport&#39;s width
func Vw(val float32) Value {
	return Value{Val: val, Un: UnitVw}
}

// Vh returns a new vh value:
// UnitVh = 1% of the viewport&#39;s height
func Vh(val float32) Value {
	return Value{Val: val, Un: UnitVh}
}

// Vmin returns a new vmin value:
// UnitVmin = 1% of the viewport&#39;s smaller dimension
func Vmin(val float32) Value {
	return Value{Val: val, Un: UnitVmin}
}

// Vmax returns a new vmax value:
// UnitVmax = 1% of the viewport&#39;s larger dimension
func Vmax(val float32) Value {
	return Value{Val: val, Un: UnitVmax}
}

// Cm returns a new cm value:
// UnitCm = centimeters -- 1cm = 96px/2.54
func Cm(val float32) Value {
	return Value{Val: val, Un: UnitCm}
}

// Mm returns a new mm value:
// UnitMm = millimeters -- 1mm = 1/10th of cm
func Mm(val float32) Value {
	return Value{Val: val, Un: UnitMm}
}

// Q returns a new q value:
// UnitQ = quarter-millimeters -- 1q = 1/40th of cm
func Q(val float32) Value {
	return Value{Val: val, Un: UnitQ}
}

// In returns a new in value:
// UnitIn = inches -- 1in = 2.54cm = 96px
func In(val float32) Value {
	return Value{Val: val, Un: UnitIn}
}

// Pc returns a new pc value:
// UnitPc = picas -- 1pc = 1/6th of 1in
func Pc(val float32) Value {
	return Value{Val: val, Un: UnitPc}
}

// Pt returns a new pt value:
// UnitPt = points -- 1pt = 1/72th of 1in
func Pt(val float32) Value {
	return Value{Val: val, Un: UnitPt}
}

// Dot returns a new dot value:
// UnitDot = actual real display pixels -- generally only use internally
func Dot(val float32) Value {
	return Value{Val: val, Un: UnitDot}
}
