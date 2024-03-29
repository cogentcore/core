// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package units

import (
	"fmt"
	"strings"

	"log/slog"

	"cogentcore.org/core/laser"
	"golang.org/x/image/math/fixed"
)

// NOTE: we have empty labels for value fields, because there is a natural
// flow of the unit values without it. "{{Value}} {{Unit}}" without labels
// makes sense and provides a nicer end-user experience.

// Value and units, and converted value into raw pixels (dots in DPI)
type Value struct { //gti:add

	// the value in terms of the specified unit
	Val float32 `label:""`

	// the unit used for the value
	Un Units `label:""`

	// the computed value in raw pixels (dots in DPI)
	Dots float32 `view:"-"`

	// custom function that returns the dots of the value;
	// if non-nil, it overrides all other fields;
	// if nil, standard ToDots with the other fields is used
	Custom func(uc *Context) float32 `view:"-" json:"-" xml:"-"`
}

// New creates a new value with the given unit type
func New(val float32, un Units) Value {
	return Value{Val: val, Un: un}
}

// Set sets the value and units of an existing value
func (v *Value) Set(val float32, un Units) {
	v.Val = val
	v.Un = un
}

// Zero returns a new zero (0) value.
func Zero() Value {
	return Value{Un: UnitDot}
}

// Zero sets the value to zero (0).
func (v *Value) Zero() {
	v.Val = 0
	v.Un = UnitDot
	v.Dots = 0
}

// Dot returns a new dots value.
// Dots are actual real display pixels, which are generally only used internally.
func Dot(val float32) Value {
	return Value{Val: val, Un: UnitDot, Dots: val}
}

// Dot sets the value directly in terms of dots.
// Dots are actual real display pixels, which are generally only used internally.
func (v *Value) Dot(val float32) {
	v.Val = val
	v.Un = UnitDot
	v.Dots = val
}

// Custom returns a new custom value that has the dots
// of the value returned by the given function.
func Custom(fun func(uc *Context) float32) Value {
	return Value{Custom: fun}
}

// SetCustom sets the value to be a custom value that has
// the dots of the value returned by the given function.
func (v *Value) SetCustom(fun func(uc *Context) float32) {
	v.Custom = fun
}

// ToDots converts value to raw display pixels (dots as in DPI), setting also
// the Dots field
func (v *Value) ToDots(uc *Context) float32 {
	if v.Custom != nil {
		v.Dots = v.Custom(uc)
	} else {
		v.Dots = uc.ToDots(v.Val, v.Un)
	}
	return v.Dots
}

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
	return fmt.Sprintf("%g%s", v.Val, v.Un.String())
}

// SetString sets value from a string
func (v *Value) SetString(str string) error {
	trstr := strings.TrimSpace(strings.Replace(str, "%", "pct", -1))
	sz := len(trstr)
	if sz < 2 {
		vc, err := laser.ToFloat(str)
		if err != nil {
			return fmt.Errorf("(units.Value).SetString: unable to convert string value %q into a number: %w", trstr, err)
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
	un := UnitPx // default to pixels
	for _, u := range UnitsValues() {
		nm := u.String()
		unsz := len(nm)
		if ends[unsz-1] == nm {
			numstr = trstr[:sz-unsz]
			un = u
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

// SetAny sets value from an interface value representation as from map[string]any
// key is optional property key for error message -- always logs the error
func (v *Value) SetAny(iface any, key string) error {
	switch val := iface.(type) {
	case string:
		v.SetString(val)
	case Value:
		*v = val
	case *Value:
		*v = *val
	default: // assume Px as an implicit default
		valflt, err := laser.ToFloat(iface)
		if err == nil {
			v.Set(float32(valflt), UnitPx)
		} else {
			err := fmt.Errorf("units.Value: could not set property %q from value: %v of type: %T: %w", key, val, val, err)
			slog.Error(err.Error())
			return err
		}
	}
	return nil
}

/*

// SetFmProp sets value from property of given key name in given list of properties
// -- returns true if property found and set, error for any errors in setting
// property
func (v *Value) SetFmProp(key string, props map[string]any) (bool, error) {
	pv, ok := props[key]
	if !ok {
		return false, nil
	}
	return true, v.SetAny(pv, key)
}

// SetFmInheritProp sets value from property of given key name in inherited or
// type properties from given Ki.ki type -- returns true if property found and
// set, error for any errors in setting property
func (v *Value) SetFmInheritProp(key string, k ki.Ki, inherit, typ bool) (bool, error) {
	pv, ok := k.PropInherit(key, inherit, typ)
	if !ok {
		return false, nil
	}
	return true, v.SetAny(pv, key)
}
*/
