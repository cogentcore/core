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

import (
	"fmt"
	"log"
	"strings"

	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"golang.org/x/image/math/fixed"
)

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
type Units int32

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

	UnitsN
)

//go:generate stringer -output stringer.go -type=Units

var TypeUnits = kit.Enums.AddEnumAltLower(UnitsN, kit.NotBitFlag, nil, "Unit")

func (ev Units) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Units) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

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

// Context specifies everything about the current context necessary for converting the number
// into specific display-dependent pixels
type Context struct {

	// DPI is dots-per-inch of the display
	DPI float32

	// FontEm is the point size of the font in raw dots (not points)
	FontEm float32

	// FontEx is the height x-height of font in points (size of 'x' glyph)
	FontEx float32

	// FontCh is the ch-size character size of font in points (width of '0' glyph)
	FontCh float32

	// FontRem is rem-size of font in points -- root Em size -- typically 12 point
	FontRem float32

	// Vw is viewport width in dots
	Vw float32

	// Vh is viewport height in dots
	Vh float32

	// Ew is width of element in dots
	Ew float32

	// Eh is height of element in dots
	Eh float32

	// Pw is width of parent in dots
	Pw float32

	// Ph is height of parent in dots
	Ph float32
}

// Defaults are generic defaults
func (uc *Context) Defaults() {
	uc.DPI = PxPerInch
	uc.FontEm = 12.0
	uc.FontEx = 6.0
	uc.FontCh = 6.0
	uc.FontRem = 12.0
	uc.Vw = 800.0
	uc.Vh = 600.0
	uc.Ew = uc.Vw
	uc.Eh = uc.Vh
	uc.Pw = uc.Vw
	uc.Ph = uc.Vh
}

// Set sets the context values to the given values
func (uc *Context) Set(em, ex, ch, rem, vw, vh, ew, eh, pw, ph float32) {
	uc.SetSizes(vw, vh, ew, eh, pw, ph)
	uc.SetFont(em, ex, ch, rem)
}

// SetSizes sets the context values for the non-font sizes
// to the given values; the values are ignored if they are zero.
func (uc *Context) SetSizes(vw, vh, ew, eh, pw, ph float32) {
	if vw != 0 {
		uc.Vw = vw
	}
	if vh != 0 {
		uc.Vh = vh
	}
	if ew != 0 {
		uc.Ew = ew
	}
	if eh != 0 {
		uc.Eh = eh
	}
	if pw != 0 {
		uc.Pw = pw
	}
	if ph != 0 {
		uc.Ph = ph
	}
}

// SetFont sets the context values for fonts: note these are already in raw
// DPI dots, not points or anything else
func (uc *Context) SetFont(em, ex, ch, rem float32) {
	uc.FontEm = em
	uc.FontEx = ex
	uc.FontCh = ch
	uc.FontRem = rem
}

// ToDotsFact returns factor needed to convert given unit into raw pixels (dots in DPI)
func (uc *Context) Dots(un Units) float32 {
	if uc.DPI == 0 {
		// log.Printf("gi/units Context was not initialized -- falling back on defaults\n")
		uc.Defaults()
	}
	switch un {
	case UnitEw:
		return 0.01 * uc.Ew
	case UnitEh:
		return 0.01 * uc.Eh
	case UnitEm:
		return uc.FontEm
	case UnitEx:
		return uc.FontEx
	case UnitCh:
		return uc.FontCh
	case UnitRem:
		return uc.FontRem
	case UnitVw:
		return 0.01 * uc.Vw
	case UnitVh:
		return 0.01 * uc.Vh
	case UnitVmin:
		return kit.Min32(uc.Vw, uc.Vh)
	case UnitVmax:
		return kit.Max32(uc.Vw, uc.Vh)
	case UnitCm:
		return uc.DPI / CmPerInch
	case UnitMm:
		return uc.DPI / MmPerInch
	case UnitQ:
		return uc.DPI / (4.0 * MmPerInch)
	case UnitIn:
		return uc.DPI
	case UnitPc:
		return uc.DPI / PcPerInch
	case UnitPt:
		return uc.DPI / PtPerInch
	case UnitPx:
		return uc.DPI / PxPerInch
	case UnitDp:
		return uc.DPI / DpPerInch
	case UnitDot:
		return 1.0
	}
	return uc.DPI
}

// ToDots converts value in given units into raw display pixels (dots in DPI)
func (uc *Context) ToDots(val float32, un Units) float32 {
	return val * uc.Dots(un)
}

// PxToDots just converts a value from pixels to dots
func (uc *Context) PxToDots(val float32) float32 {
	return val * uc.Dots(UnitPx)
}

// DotsToPx just converts a value from dots to pixels
func (uc *Context) DotsToPx(val float32) float32 {
	return val / uc.Dots(UnitPx)
}

////////////////////////////////////////////////////////////////////////
//   Value

// Value and units, and converted value into raw pixels (dots in DPI)
type Value struct {

	// the value in terms of the specified unit
	Val float32 `label:"Value" desc:"the value in terms of the specified unit"`

	// the unit used for the value
	Un Units `label:"Unit" desc:"the unit used for the value"`

	// the computed value in raw pixels (dots in DPI)
	Dots float32 `inactive:"+" desc:"the computed value in raw pixels (dots in DPI)"`

	// function to compute dots from units, using arbitrary expressions; if nil, standard ToDots is used
	DotsFunc func(uc *Context) float32 `desc:"function to compute dots from units, using arbitrary expressions; if nil, standard ToDots is used"`
}

var TypeValue = kit.Types.AddType(&Value{}, ValueProps)

var ValueProps = ki.Props{
	"style-prop": true,
}

// New creates a new value with the given unit type
func New(val float32, un Units) Value {
	return Value{Val: val, Un: un}
}

// Px creates a new value of type [UnitPx]
// (1px = 1/96 of 1in; not raw display pixels)
func Px(val float32) Value {
	return Value{Val: val, Un: UnitPx}
}

// Dp creates a new value of type [UnitDp]
// (density-independent pixels; 1dp = 1/160 of 1in)
func Dp(val float32) Value {
	return Value{Val: val, Un: UnitDp}
}

// Ew creates a new value of type [UnitEw]
// (percentage of element width)
func Ew(val float32) Value {
	return Value{Val: val, Un: UnitEw}
}

// Eh creates a new value of type [UnitEh]
// (percentage of element height)
func Eh(val float32) Value {
	return Value{Val: val, Un: UnitEh}
}

// Pw creates a new value of type [UnitPw]
// (percentage of parent width)
func Pw(val float32) Value {
	return Value{Val: val, Un: UnitPw}
}

// Ph creates a new value of type [UnitPh]
// (percentage of parent height)
func Ph(val float32) Value {
	return Value{Val: val, Un: UnitPh}
}

// Rem creates a new value of type [UnitRem]
// (font size of the root element)
func Rem(val float32) Value {
	return Value{Val: val, Un: UnitRem}
}

// Em creates a new value of type [UnitEm]
// (font size of the element)
func Em(val float32) Value {
	return Value{Val: val, Un: UnitEm}
}

// Ex creates a new value of type [UnitEx]
// (height of 'x' in the font of the element)
func Ex(val float32) Value {
	return Value{Val: val, Un: UnitEx}
}

// Ch creates a new value of type [UnitCh]
// (width of '0' in the font of the element)
func Ch(val float32) Value {
	return Value{Val: val, Un: UnitCh}
}

// Vw creates a new value of type [UnitVw]
// (percentage of viewport width)
func Vw(val float32) Value {
	return Value{Val: val, Un: UnitVw}
}

// Vh creates a new value of type [UnitVh]
// (percentage of viewport height)
func Vh(val float32) Value {
	return Value{Val: val, Un: UnitVh}
}

// Vmin creates a new value of type [UnitVmin]
// (percentage of viewport's smaller dimension)
func Vmin(val float32) Value {
	return Value{Val: val, Un: UnitVmin}
}

// Vmax creates a new value of type [UnitVmax]
// (percentage of viewport's bigger dimension)
func Vmax(val float32) Value {
	return Value{Val: val, Un: UnitVmax}
}

// Cm creates a new value of type [UnitCm]
// (centimeters; 1cm = 96px/2.54)
func Cm(val float32) Value {
	return Value{Val: val, Un: UnitCm}
}

// Mm creates a new value of type [UnitMm]
// (millimeters; 1mm = 1/10 of 1cm)
func Mm(val float32) Value {
	return Value{Val: val, Un: UnitMm}
}

// Q creates a new value of type [UnitQ]
// (quarter-millimeters; 1q = 1/40 of 1cm)
func Q(val float32) Value {
	return Value{Val: val, Un: UnitQ}
}

// In creates a new value of type [UnitIn]
// (inches; 1in = 96px)
func In(val float32) Value {
	return Value{Val: val, Un: UnitIn}
}

// Pc creates a new value of type [UnitPc]
// (picas; 1pc = 1/6 of 1in)
func Pc(val float32) Value {
	return Value{Val: val, Un: UnitPc}
}

// Pt creates a new value of type [UnitPt]
// (points; 1pt = 1/72 of 1in)
func Pt(val float32) Value {
	return Value{Val: val, Un: UnitPt}
}

// Dot creates a new value of type [UnitDot]
// (actual raw display pixels)
func Dot(val float32) Value {
	return Value{Val: val, Un: UnitDot}
}

// Set sets the value and units of an existing value
func (v *Value) Set(val float32, un Units) {
	v.Val = val
	v.Un = un
}

// SetPx sets the value in terms of [UnitPx]
// (1px = 1/96 of 1in; not raw display pixels)
func (v *Value) SetPx(val float32) {
	v.Val = val
	v.Un = UnitPx
}

// SetDp sets the value in terms of [UnitDp]
// (density-independent pixels; 1dp = 1/160 of 1in)
func (v *Value) SetDp(val float32) {
	v.Val = val
	v.Un = UnitDp
}

// SetEw sets the value in terms of [UnitEw]
// (percentage of element width)
func (v *Value) SetEw(val float32) {
	v.Val = val
	v.Un = UnitEw
}

// SetEh sets the value in terms of [UnitEh]
// (percentage of element height)
func (v *Value) SetEh(val float32) {
	v.Val = val
	v.Un = UnitEh
}

// SetPw sets the value in terms of [UnitPw]
// (percentage of parent width)
func (v *Value) SetPw(val float32) {
	v.Val = val
	v.Un = UnitPw
}

// SetPh sets the value in terms of [UnitPh]
// (percentage of parent height)
func (v *Value) SetPh(val float32) {
	v.Val = val
	v.Un = UnitPh
}

// SetRem sets the value in terms of [UnitRem]
// (font size of the root element)
func (v *Value) SetRem(val float32) {
	v.Val = val
	v.Un = UnitRem
}

// SetEm sets the value in terms of [UnitEm]
// (font size of the element)
func (v *Value) SetEm(val float32) {
	v.Val = val
	v.Un = UnitEm
}

// SetEx sets the value in terms of [UnitEx]
// (height of 'x' in the font of the element)
func (v *Value) SetEx(val float32) {
	v.Val = val
	v.Un = UnitEx
}

// SetCh sets the value in terms of [UnitCh]
// (width of '0' in the font of the element)
func (v *Value) SetCh(val float32) {
	v.Val = val
	v.Un = UnitCh
}

// SetVw sets the value in terms of [UnitVw]
// (percentage of viewport width)
func (v *Value) SetVw(val float32) {
	v.Val = val
	v.Un = UnitVw
}

// SetVh sets the value in terms of [UnitVh]
// (percentage of viewport height)
func (v *Value) SetVh(val float32) {
	v.Val = val
	v.Un = UnitVh
}

// SetVmin sets the value in terms of [UnitVmin]
// (percentage of viewport's smaller dimension)
func (v *Value) SetVmin(val float32) {
	v.Val = val
	v.Un = UnitVmin
}

// SetVmax sets the value in terms of [UnitVmax]
// (percentage of viewport's bigger dimension)
func (v *Value) SetVmax(val float32) {
	v.Val = val
	v.Un = UnitVmax
}

// SetCm sets the value in terms of [UnitCm]
// (centimeters; 1cm = 96px/2.54)
func (v *Value) SetCm(val float32) {
	v.Val = val
	v.Un = UnitCm
}

// SetMm sets the value in terms of [UnitMm]
// (millimeters; 1mm = 1/10 of 1cm)
func (v *Value) SetMm(val float32) {
	v.Val = val
	v.Un = UnitMm
}

// SetQ sets the value in terms of [UnitQ]
// (quarter-millimeters; 1q = 1/40 of 1cm)
func (v *Value) SetQ(val float32) {
	v.Val = val
	v.Un = UnitQ
}

// SetIn sets the value in terms of [UnitIn]
// (inches; 1in = 96px)
func (v *Value) SetIn(val float32) {
	v.Val = val
	v.Un = UnitIn
}

// SetPc sets the value in terms of [UnitPc]
// (picas; 1pc = 1/6 of 1in)
func (v *Value) SetPc(val float32) {
	v.Val = val
	v.Un = UnitPc
}

// SetPt sets the value in terms of [UnitPt]
// (points; 1pt = 1/72 of 1in)
func (v *Value) SetPt(val float32) {
	v.Val = val
	v.Un = UnitPt
}

// SetDot sets the value in terms of [UnitDots] directly
// (actual raw display pixels)
func (v *Value) SetDot(val float32) {
	v.Val = val
	v.Un = UnitDot
	v.Dots = val
}

// ToDots converts value to raw display pixels (dots as in DPI), setting also
// the Dots field
func (v *Value) ToDots(uc *Context) float32 {
	if v.DotsFunc != nil {
		v.Dots = v.DotsFunc(uc)
	} else {
		v.Dots = uc.ToDots(v.Val, v.Un)
	}
	return v.Dots
}

// example todots func
// v.DotsFunc = func(uc *Context) float32 {
// 	return uc.Vw(50) - uc.Em(4)
// }

// ToDotsFixed converts value to raw display pixels (dots in DPI) in
// fixed-point 26.6 format for rendering
func (v *Value) ToDotsFixed(uc *Context) fixed.Int26_6 {
	return fixed.Int26_6(v.ToDots(uc))
}

// Convert converts value to the given units, given unit context
func (v *Value) Convert(to Units, uc *Context) Value {
	dots := v.ToDots(uc)
	return Value{Val: dots / uc.Dots(to), Un: to, Dots: dots}
}

// String implements the fmt.Stringer interface.
func (v *Value) String() string {
	return fmt.Sprintf("%g%s", v.Val, UnitNames[v.Un])
}

// SetString sets value from a string
func (v *Value) SetString(str string) error {
	trstr := strings.TrimSpace(strings.Replace(str, "%", "pct", -1))
	sz := len(trstr)
	if sz < 2 {
		vc, ok := kit.ToFloat(str)
		if !ok {
			return fmt.Errorf("(units.Value).SetString: unable to convert string value '%s' into a number", trstr)
		}
		v.Val = float32(vc)
		v.Un = UnitPx
		return nil
	}
	var ends [4]string
	ends[0] = strings.ToLower(trstr[sz-1:])
	ends[1] = strings.ToLower(trstr[sz-2:])
	if sz > 3 {
		ends[2] = strings.ToLower(trstr[sz-3:])
	}
	if sz > 4 {
		ends[3] = strings.ToLower(trstr[sz-4:])
	}

	var numstr string
	var un Units = UnitPx // default to pixels
	for i, nm := range UnitNames {
		unsz := len(nm)
		if ends[unsz-1] == nm {
			numstr = trstr[:sz-unsz]
			un = Units(i)
			break
		}
	}
	if len(numstr) == 0 { // no units
		numstr = trstr
	}
	var val float32
	trspc := strings.TrimSpace(numstr)
	n, err := fmt.Sscanf(trspc, "%g", &val)
	if err != nil {
		return fmt.Errorf("(units.Value).SetString: error scanning string '%s': %w", trspc, err)
	}
	if n == 0 {
		return fmt.Errorf("(units.Value).SetString: no arguments parsed from string '%s'", trspc)
	}
	v.Set(val, un)
	return nil
}

// StringToValue converts a string to a value representation.
func StringToValue(str string) Value {
	var v Value
	v.SetString(str)
	return v
}

// SetIFace sets value from an interface value representation as from ki.Props
// key is optional property key for error message -- always logs the error
func (v *Value) SetIFace(iface any, key string) error {
	switch val := iface.(type) {
	case string:
		v.SetString(val)
	case Value:
		*v = val
	case *Value:
		*v = *val
	default: // assume Px as an implicit default
		valflt, ok := kit.ToFloat(iface)
		if ok {
			v.Set(float32(valflt), UnitPx)
		} else {
			err := fmt.Errorf("units.Value could not set property: %v from: %v type: %T", key, val, val)
			log.Println(err)
			return err
		}
	}
	return nil
}

// SetFmProp sets value from property of given key name in given list of properties
// -- returns true if property found and set, error for any errors in setting
// property
func (v *Value) SetFmProp(key string, props ki.Props) (bool, error) {
	pv, ok := props[key]
	if !ok {
		return false, nil
	}
	return true, v.SetIFace(pv, key)
}

// SetFmInheritProp sets value from property of given key name in inherited or
// type properties from given Ki.ki type -- returns true if property found and
// set, error for any errors in setting property
func (v *Value) SetFmInheritProp(key string, k ki.Ki, inherit, typ bool) (bool, error) {
	pv, ok := k.PropInherit(key, inherit, typ)
	if !ok {
		return false, nil
	}
	return true, v.SetIFace(pv, key)
}
