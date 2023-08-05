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

	// UnitPct = percentage of surrounding contextual element
	UnitPct

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

//go:generate stringer -type=Units

var TypeUnits = kit.Enums.AddEnumAltLower(UnitsN, kit.NotBitFlag, nil, "Unit")

func (ev Units) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Units) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

var UnitNames = [...]string{
	UnitPx:   "px",
	UnitDp:   "dp",
	UnitPct:  "pct",
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

	// VpW is viewport width in dots
	VpW float32

	// VpH is viewport height in dots
	VpH float32

	// ElW is width of surrounding contextual element in dots
	ElW float32

	// ElH is height of surrounding contextual element in dots
	ElH float32
}

// Defaults are generic defaults
func (uc *Context) Defaults() {
	uc.DPI = PxPerInch
	uc.FontEm = 12.0
	uc.FontEx = 6.0
	uc.FontCh = 6.0
	uc.FontRem = 12.0
	uc.VpW = 800.0
	uc.VpH = 600.0
	uc.ElW = uc.VpW
	uc.ElH = uc.VpH
}

// Set sets the context values
func (uc *Context) Set(em, ex, ch, rem, vpw, vph, elw, elh float32) {
	uc.SetSizes(vpw, vph, elw, elh)
	uc.SetFont(em, ex, ch, rem)
}

// SetSizes sets the context values for non-font sizes -- el is ignored if zero
func (uc *Context) SetSizes(vpw, vph, elw, elh float32) {
	if vpw != 0 {
		uc.VpW = vpw
	}
	if vph != 0 {
		uc.VpH = vph
	}
	if elw != 0 {
		uc.ElW = elw
	}
	if elh != 0 {
		uc.ElH = elh
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
func (uc *Context) ToDotsFactor(un Units) float32 {
	if uc.DPI == 0 {
		// log.Printf("gi/units Context was not initialized -- falling back on defaults\n")
		uc.Defaults()
	}
	switch un {
	case UnitPct:
		return 0.01 * uc.ElW // todo: height should be in terms of Elh.. but width is much more common
	case UnitEm:
		return uc.FontEm
	case UnitEx:
		return uc.FontEx
	case UnitCh:
		return uc.FontCh
	case UnitRem:
		return uc.FontRem
	case UnitVw:
		return 0.01 * uc.VpW
	case UnitVh:
		return 0.01 * uc.VpH
	case UnitVmin:
		return kit.Min32(uc.VpW, uc.VpH)
	case UnitVmax:
		return kit.Max32(uc.VpW, uc.VpH)
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
	return val * uc.ToDotsFactor(un)
}

// PxToDots just converts a value from pixels to dots
func (uc *Context) PxToDots(val float32) float32 {
	return val * uc.ToDotsFactor(UnitPx)
}

// DotsToPx just converts a value from dots to pixels
func (uc *Context) DotsToPx(val float32) float32 {
	return val / uc.ToDotsFactor(UnitPx)
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
}

var TypeValue = kit.Types.AddType(&Value{}, ValueProps)

var ValueProps = ki.Props{
	"style-prop": true,
}

// NewValue creates a new value with given units
func NewValue(val float32, un Units) Value {
	return Value{val, un, 0.0}
}

// Px creates a new Px value
func Px(val float32) Value {
	return Value{val, UnitPx, 0.0}
}

// Rem creates a new Rem value
func Rem(val float32) Value {
	return Value{val, UnitRem, 0.0}
}

// Em creates a new Em value
func Em(val float32) Value {
	return Value{val, UnitEm, 0.0}
}

// Ex creates a new Ex value
func Ex(val float32) Value {
	return Value{val, UnitEx, 0.0}
}

// Ch creates a new Ch value
func Ch(val float32) Value {
	return Value{val, UnitCh, 0.0}
}

// Pt creates a new Pt value
func Pt(val float32) Value {
	return Value{val, UnitPt, 0.0}
}

// Pct creates a new Pct value
func Pct(val float32) Value {
	return Value{val, UnitPct, 0.0}
}

// Dp creates a new Dp value
func Dp(val float32) Value {
	return Value{val, UnitDp, 0.0}
}

// Dot creates a new Dot value
func Dot(val float32) Value {
	return Value{val, UnitDot, 0.0}
}

// Set sets value and units of an existing value
func (v *Value) Set(val float32, un Units) {
	v.Val = val
	v.Un = un
}

// SetPx sets value in Px
func (v *Value) SetPx(val float32) {
	v.Val = val
	v.Un = UnitPx
}

// SetRem sets value in Rem
func (v *Value) SetRem(val float32) {
	v.Val = val
	v.Un = UnitRem
}

// SetEm sets value in Em
func (v *Value) SetEm(val float32) {
	v.Val = val
	v.Un = UnitEm
}

// SetEx sets value in Ex
func (v *Value) SetEx(val float32) {
	v.Val = val
	v.Un = UnitEx
}

// SetCh sets value in Ch
func (v *Value) SetCh(val float32) {
	v.Val = val
	v.Un = UnitCh
}

// SetPt sets value in Pt
func (v *Value) SetPt(val float32) {
	v.Val = val
	v.Un = UnitPt
}

// SetPct sets value in Pct
func (v *Value) SetPct(val float32) {
	v.Val = val
	v.Un = UnitPct
}

// SetDp sets value in Dp
func (v *Value) SetDp(val float32) {
	v.Val = val
	v.Un = UnitPx
}

// SetDot sets value in Dots directly
func (v *Value) SetDot(val float32) {
	v.Val = val
	v.Un = UnitDot
	v.Dots = val
}

// ToDots converts value to raw display pixels (dots as in DPI), setting also
// the Dots field
func (v *Value) ToDots(ctxt *Context) float32 {
	v.Dots = ctxt.ToDots(v.Val, v.Un)
	return v.Dots
}

// ToDotsFixed converts value to raw display pixels (dots in DPI) in
// fixed-point 26.6 format for rendering
func (v *Value) ToDotsFixed(ctxt *Context) fixed.Int26_6 {
	return fixed.Int26_6(v.ToDots(ctxt))
}

// Convert converts value to the given units, given unit context
func (v *Value) Convert(to Units, ctxt *Context) Value {
	dots := v.ToDots(ctxt)
	return Value{dots / ctxt.ToDotsFactor(to), to, dots}
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
