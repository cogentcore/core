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

type Units int32

const (
	// Px = pixels -- 1px = 1/96th of 1in -- these are NOT raw display pixels
	Px Units = iota

	// Dp = density-independent pixels -- 1dp = 1/160th of 1in
	Dp

	// Pct = percentage of surrounding contextual element
	Pct

	// Rem = font size of the root element -- defaults to 12pt scaled by DPI factor
	Rem

	// Em = font size of the element -- fallback to 12pt by default
	Em

	// Ex = x-height of the element's font (size of 'x' glyph) -- fallback to 0.5em by default
	Ex

	// Ch = width of the '0' glyph in the element's font -- fallback to 0.5em by default
	Ch

	// Vw = 1% of the viewport's width
	Vw

	// Vh = 1% of the viewport's height
	Vh

	// Vmin = 1% of the viewport's smaller dimension
	Vmin

	// Vmax = 1% of the viewport's larger dimension
	Vmax

	// Cm = centimeters -- 1cm = 96px/2.54
	Cm

	// Mm = millimeters -- 1mm = 1/10th of cm
	Mm

	// Q = quarter-millimeters -- 1q = 1/40th of cm
	Q

	// In = inches -- 1in = 2.54cm = 96px
	In

	// Pc = picas -- 1pc = 1/6th of 1in
	Pc

	// Pt = points -- 1pt = 1/72th of 1in
	Pt

	// Dot = actual real display pixels -- generally only use internally
	Dot

	UnitsN
)

//go:generate stringer -type=Units

var KiT_Units = kit.Enums.AddEnumAltLower(UnitsN, kit.NotBitFlag, nil, "")

func (ev Units) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Units) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

var UnitNames = [...]string{
	Px:   "px",
	Dp:   "dp",
	Pct:  "pct",
	Rem:  "rem",
	Em:   "em",
	Ex:   "ex",
	Ch:   "ch",
	Vw:   "vw",
	Vh:   "vh",
	Vmin: "vmin",
	Vmax: "vmax",
	Cm:   "cm",
	Mm:   "mm",
	Q:    "q",
	In:   "in",
	Pc:   "pc",
	Pt:   "pt",
	Dot:  "dot",
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
	case Pct:
		return 0.01 * uc.ElW // todo: height should be in terms of Elh.. but width is much more common
	case Em:
		return uc.FontEm
	case Ex:
		return uc.FontEx
	case Ch:
		return uc.FontCh
	case Rem:
		return uc.FontRem
	case Vw:
		return 0.01 * uc.VpW
	case Vh:
		return 0.01 * uc.VpH
	case Vmin:
		return kit.Min32(uc.VpW, uc.VpH)
	case Vmax:
		return kit.Max32(uc.VpW, uc.VpH)
	case Cm:
		return uc.DPI / CmPerInch
	case Mm:
		return uc.DPI / MmPerInch
	case Q:
		return uc.DPI / (4.0 * MmPerInch)
	case In:
		return uc.DPI
	case Pc:
		return uc.DPI / PcPerInch
	case Pt:
		return uc.DPI / PtPerInch
	case Px:
		return uc.DPI / PxPerInch
	case Dp:
		return uc.DPI / DpPerInch
	case Dot:
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
	return val * uc.ToDotsFactor(Px)
}

// DotsToPx just converts a value from dots to pixels
func (uc *Context) DotsToPx(val float32) float32 {
	return val / uc.ToDotsFactor(Px)
}

////////////////////////////////////////////////////////////////////////
//   Value

// Value and units, and converted value into raw pixels (dots in DPI)
type Value struct {
	Val  float32
	Un   Units
	Dots float32
}

var KiT_Value = kit.Types.AddType(&Value{}, ValueProps)

var ValueProps = ki.Props{
	"style-prop": true,
}

// NewValue creates a new value with given units
func NewValue(val float32, un Units) Value {
	return Value{val, un, 0.0}
}

// NewPx creates a new Px value
func NewPx(val float32) Value {
	return Value{val, Px, 0.0}
}

// NewEm creates a new Em value
func NewEm(val float32) Value {
	return Value{val, Em, 0.0}
}

// NewEx creates a new Ex value
func NewEx(val float32) Value {
	return Value{val, Ex, 0.0}
}

// NewCh creates a new Ch value
func NewCh(val float32) Value {
	return Value{val, Ch, 0.0}
}

// NewPt creates a new Pt value
func NewPt(val float32) Value {
	return Value{val, Pt, 0.0}
}

// NewPct creates a new Pct value
func NewPct(val float32) Value {
	return Value{val, Pct, 0.0}
}

// NewDp creates a new Dp value
func NewDp(val float32) Value {
	return Value{val, Dp, 0.0}
}

// NewDot creates a new Dot value
func NewDot(val float32) Value {
	return Value{val, Dot, 0.0}
}

// Set sets value and units of an existing value
func (v *Value) Set(val float32, un Units) {
	v.Val = val
	v.Un = un
}

// SetPx sets value in Px
func (v *Value) SetPx(val float32) {
	v.Val = val
	v.Un = Px
}

// SetEm sets value in Em
func (v *Value) SetEm(val float32) {
	v.Val = val
	v.Un = Em
}

// SetEx sets value in Ex
func (v *Value) SetEx(val float32) {
	v.Val = val
	v.Un = Ex
}

// SetCh sets value in Ch
func (v *Value) SetCh(val float32) {
	v.Val = val
	v.Un = Ch
}

// SetPt sets value in Pt
func (v *Value) SetPt(val float32) {
	v.Val = val
	v.Un = Pt
}

// SetPct sets value in Pct
func (v *Value) SetPct(val float32) {
	v.Val = val
	v.Un = Pct
}

// SetDp sets value in Dp
func (v *Value) SetDp(val float32) {
	v.Val = val
	v.Un = Px
}

// SetDot sets value in Dots directly
func (v *Value) SetDot(val float32) {
	v.Val = val
	v.Un = Dot
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
func (v *Value) SetString(str string) {
	trstr := strings.TrimSpace(strings.Replace(str, "%", "pct", -1))
	sz := len(trstr)
	if sz < 2 {
		vc, _ := kit.ToFloat(str)
		v.Val = float32(vc)
		v.Un = Px
		return
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
	var un Units = Px // default to pixels
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
	fmt.Sscanf(strings.TrimSpace(numstr), "%g", &val)
	v.Set(val, un)
}

// StringToValue converts a string to a value representation.
func StringToValue(str string) Value {
	var v Value
	v.SetString(str)
	return v
}

// SetIFace sets value from an interface value representation as from ki.Props
// key is optional property key for error message -- always logs the error
func (v *Value) SetIFace(iface interface{}, key string) error {
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
			v.Set(float32(valflt), Px)
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
