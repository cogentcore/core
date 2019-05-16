// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"strconv"

	"github.com/goki/ki/floats"
	"github.com/goki/ki/ints"
)

// Sel implements the "mute" function from here
// http://blog.vladimirvivien.com/2014/03/hacking-go-filter-values-from-multi.html
// provides a way to select a particular return value in a single expression,
// without having a separate assignment in between -- I just call it "Sel" as
// I'm unlikely to remember how to type a mu
func Sel(a ...interface{}) []interface{} {
	return a
}

// IfaceIsNil checks if an interface value is nil -- the interface itself could be
// nil, or the value pointed to by the interface could be nil -- this checks
// both, safely
func IfaceIsNil(it interface{}) bool {
	if it == nil {
		return true
	}
	v := reflect.ValueOf(it)
	vk := v.Kind()
	if vk == reflect.Ptr || vk == reflect.Interface || vk == reflect.Map || vk == reflect.Slice || vk == reflect.Func || vk == reflect.Chan {
		return v.IsNil()
	}
	return false
}

// KindIsBasic returns true if the reflect.Kind is a basic type such as Int, Float, etc
func KindIsBasic(vk reflect.Kind) bool {
	if vk >= reflect.Bool && vk <= reflect.Complex128 {
		return true
	}
	return false
}

// ValueIsZero returns true if the reflect.Value is Zero or nil or invalid or
// otherwise doesn't have a useful value -- from
// https://github.com/golang/go/issues/7501
func ValueIsZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	case reflect.Func:
		return v == reflect.Zero(v.Type())
	}
	return false
}

// Convenience functions for converting interface{} (e.g. properties) to given
// types uses the "ok" bool mechanism to report failure -- are as robust and
// general as possible.
//
// WARNING: these violate many of the type-safety features of Go but OTOH give
// maximum robustness, appropriate for the world of end-user settable
// properties, and deal with most common-sense cases, e.g., string <-> number,
// etc.  nil values return !ok

// ToBool robustly converts anything to a bool
func ToBool(it interface{}) (bool, bool) {
	if IfaceIsNil(it) {
		return false, false
	}
	v := NonPtrValue(reflect.ValueOf(it))
	vk := v.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return (v.Int() != 0), true
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return (v.Uint() != 0), true
	case vk == reflect.Bool:
		return v.Bool(), true
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return (v.Float() != 0.0), true
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return (real(v.Complex()) != 0.0), true
	case vk == reflect.String:
		r, err := strconv.ParseBool(v.String())
		if err != nil {
			return false, false
		}
		return r, true
	default:
		return false, false
	}
}

// ToInt robustlys converts anything to an int64 -- uses the ints.Inter ToInt
// interface first if available
func ToInt(it interface{}) (int64, bool) {
	if IfaceIsNil(it) {
		return 0, false
	}
	if inter, ok := it.(ints.Inter); ok {
		return inter.Int(), true
	}
	v := NonPtrValue(reflect.ValueOf(it))
	vk := v.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return v.Int(), true
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return int64(v.Uint()), true
	case vk == reflect.Bool:
		if v.Bool() {
			return 1, true
		}
		return 0, true
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return int64(v.Float()), true
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return int64(real(v.Complex())), true
	case vk == reflect.String:
		r, err := strconv.ParseInt(v.String(), 0, 64)
		if err != nil {
			return 0, false
		}
		return r, true
	default:
		return 0, false
	}
}

// ToFloat robustly converts anything to a Float64 -- uses the floats.Floater Float()
// interface first if available
func ToFloat(it interface{}) (float64, bool) {
	if IfaceIsNil(it) {
		return 0.0, false
	}
	if floater, ok := it.(floats.Floater); ok {
		return floater.Float(), true
	}
	v := NonPtrValue(reflect.ValueOf(it))
	vk := v.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return float64(v.Int()), true
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return float64(v.Uint()), true
	case vk == reflect.Bool:
		if v.Bool() {
			return 1.0, true
		}
		return 0.0, true
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return v.Float(), true
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return real(v.Complex()), true
	case vk == reflect.String:
		r, err := strconv.ParseFloat(v.String(), 64)
		if err != nil {
			return 0.0, false
		}
		return r, true
	default:
		return 0.0, false
	}
}

// ToFloat32 robustly converts anything to a Float64 -- uses the floats.Floater Float()
// interface first if available
func ToFloat32(it interface{}) (float32, bool) {
	if IfaceIsNil(it) {
		return float32(0.0), false
	}
	if floater, ok := it.(floats.Floater); ok {
		return float32(floater.Float()), true
	}
	v := NonPtrValue(reflect.ValueOf(it))
	vk := v.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return float32(v.Int()), true
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return float32(v.Uint()), true
	case vk == reflect.Bool:
		if v.Bool() {
			return 1.0, true
		}
		return 0.0, true
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return float32(v.Float()), true
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return float32(real(v.Complex())), true
	case vk == reflect.String:
		r, err := strconv.ParseFloat(v.String(), 32)
		if err != nil {
			return float32(0.0), false
		}
		return float32(r), true
	default:
		return float32(0.0), false
	}
}

// ToString robustly converts anything to a String -- because Stringer is so
// ubiquitous, and we fall back to fmt.Sprintf(%v) in worst case, this should
// definitely work in all cases, so there is no bool return value
func ToString(it interface{}) string {
	if IfaceIsNil(it) {
		return "nil"
	}
	if stringer, ok := it.(fmt.Stringer); ok {
		return stringer.String()
	}
	v := NonPtrValue(reflect.ValueOf(it))
	vk := v.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case vk == reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'G', -1, 64)
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		cv := v.Complex()
		rv := strconv.FormatFloat(real(cv), 'G', -1, 64) + "," + strconv.FormatFloat(imag(cv), 'G', -1, 64)
		return rv
	case vk == reflect.String:
		return v.String()
	case vk == reflect.Slice:
		eltyp := SliceElType(it)
		if eltyp.Kind() == reflect.Uint8 { // []byte
			return string(it.([]byte))
		}
		fallthrough
	default:
		return fmt.Sprintf("%v", it)
	}
}

// ToStringPrec robustly converts anything to a String using given precision
// for converting floating values -- using a value like 6 truncates the
// nuisance random imprecision of actual floating point values due to the
// fact that they are represented with binary bits.  See ToString
// for more info.
func ToStringPrec(it interface{}, prec int) string {
	if IfaceIsNil(it) {
		return "nil"
	}
	if stringer, ok := it.(fmt.Stringer); ok {
		return stringer.String()
	}
	v := NonPtrValue(reflect.ValueOf(it))
	vk := v.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case vk == reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'G', prec, 64)
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		cv := v.Complex()
		rv := strconv.FormatFloat(real(cv), 'G', prec, 64) + "," + strconv.FormatFloat(imag(cv), 'G', prec, 64)
		return rv
	case vk == reflect.String:
		return v.String()
	case vk == reflect.Slice:
		eltyp := SliceElType(it)
		if eltyp.Kind() == reflect.Uint8 { // []byte
			return string(it.([]byte))
		}
		fallthrough
	default:
		return fmt.Sprintf("%v", it)
	}
}

// SetRobust robustly sets the to value from the from value -- to must be a
// pointer-to -- only for basic field values -- use copier package for more
// complex cases
func SetRobust(to, frm interface{}) bool {
	if IfaceIsNil(to) {
		return false
	}
	v := reflect.ValueOf(to)
	vnp := NonPtrValue(v)
	if !vnp.IsValid() {
		return false
	}
	typ := vnp.Type()
	vp := OnePtrValue(vnp)
	vk := vnp.Kind()
	if !vp.Elem().CanSet() {
		log.Printf("ki.SetRobust 'to' cannot be set -- must be a variable or field, not a const or tmp or other value that cannot be set.  Value info: %v\n", vp)
		return false
	}
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		fm, ok := ToInt(frm)
		if ok {
			vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
			return true
		}
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		fm, ok := ToInt(frm)
		if ok {
			vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
			return true
		}
	case vk == reflect.Bool:
		fm, ok := ToBool(frm)
		if ok {
			vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
			return true
		}
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		fm, ok := ToFloat(frm)
		if ok {
			vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
			return true
		}
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		// cv := v.Complex()
		// rv := strconv.FormatFloat(real(cv), 'G', -1, 64) + "," + strconv.FormatFloat(imag(cv), 'G', -1, 64)
		// return rv, true
	case vk == reflect.String: // todo: what about []byte?
		fm := ToString(frm)
		vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return true
	}

	fv := reflect.ValueOf(frm)
	// Just set it if possible to assign
	if fv.Type().AssignableTo(typ) {
		vp.Elem().Set(fv)
		return true
	}
	return false
}

// SetMapRobust robustly sets a map value using reflect.Value representations
// of the map, key, and value elements, ensuring that the proper types are
// used for the key and value elements using sensible conversions.
// map value must be a valid map value -- that is not checked.
func SetMapRobust(mp, ky, val reflect.Value) bool {
	mtyp := mp.Type()
	if mtyp.Kind() != reflect.Map {
		log.Printf("ki.SetMapRobust: map arg is not map, is: %v\n", mtyp.String())
		return false
	}
	if !mp.CanSet() {
		log.Printf("ki.SetMapRobust: map arg is not settable: %v\n", mtyp.String())
		return false
	}
	ktyp := mtyp.Key()
	etyp := mtyp.Elem()
	if etyp.Kind() == val.Kind() && ky.Kind() == ktyp.Kind() {
		mp.SetMapIndex(ky, val)
		return true
	}
	if ky.Kind() == ktyp.Kind() {
		mp.SetMapIndex(ky, val.Convert(etyp))
		return true
	}
	if etyp.Kind() == val.Kind() {
		mp.SetMapIndex(ky.Convert(ktyp), val)
		return true
	}
	mp.SetMapIndex(ky.Convert(ktyp), val.Convert(etyp))
	return true
}

// MakeMap makes a map that is actually addressable, getting around the hidden
// interface{} that reflect.MakeMap makes, by calling UnhideIfaceValue (from ptrs.go)
func MakeMap(typ reflect.Type) reflect.Value {
	return UnhideIfaceValue(reflect.MakeMap(typ))
}

// MakeSlice makes a map that is actually addressable, getting around the hidden
// interface{} that reflect.MakeSlice makes, by calling UnhideIfaceValue (from ptrs.go)
func MakeSlice(typ reflect.Type, len, cap int) reflect.Value {
	return UnhideIfaceValue(reflect.MakeSlice(typ, len, cap))
}

// CloneToType creates a new object of given type, and uses SetRobust to copy
// an existing value (of perhaps another type) into it -- only expected to
// work for basic types
func CloneToType(typ reflect.Type, val interface{}) reflect.Value {
	if NonPtrType(typ).Kind() == reflect.Map {
		return MakeMap(typ)
	} else if NonPtrType(typ).Kind() == reflect.Slice {
		return MakeSlice(typ, 0, 0)
	}
	vn := reflect.New(typ)
	evi := vn.Interface()
	SetRobust(evi, val)
	return vn
}

// MakeOfType creates a new object of given type with appropriate magic foo to
// make it usable
func MakeOfType(typ reflect.Type) reflect.Value {
	if NonPtrType(typ).Kind() == reflect.Map {
		return MakeMap(typ)
	} else if NonPtrType(typ).Kind() == reflect.Slice {
		return MakeSlice(typ, 0, 0)
	}
	vn := reflect.New(typ)
	return vn
}

////////////////////////////////////////////////////////////////////////////////////////
//  Min / Max for other types..

// math provides Max/Min for 64bit -- these are for specific subtypes

func Max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func Min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// minimum excluding 0
func MinPos(a, b float64) float64 {
	if a > 0.0 && b > 0.0 {
		return math.Min(a, b)
	} else if a > 0.0 {
		return a
	} else if b > 0.0 {
		return b
	}
	return a
}

// minimum excluding 0
func MinPos32(a, b float32) float32 {
	if a > 0.0 && b > 0.0 {
		return Min32(a, b)
	} else if a > 0.0 {
		return a
	} else if b > 0.0 {
		return b
	}
	return a
}
