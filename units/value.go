// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"fmt"
	"log"
	"strings"

	"goki.dev/ki/v2/ki"
	"goki.dev/ki/v2/kit"
	"golang.org/x/image/math/fixed"
)

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
