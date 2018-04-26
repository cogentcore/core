// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

// github.com/rcoreilly/goki/ki/kit

import (
	// "fmt"
	"fmt"
	"log"
	"math"
	"reflect"
	"strconv"
)

// Sel implements the "mute" function from here
// http://blog.vladimirvivien.com/2014/03/hacking-go-filter-values-from-multi.html
// provides a way to select a particular return value in a single expression,
// without having a separate assignment in between -- I just call it "Sel" as
// I'm unlikely to remember how to type a mu
func Sel(a ...interface{}) []interface{} {
	return a
}

// IsNil checks if an interface value is nil -- the interface itself could be
// nil, or the value pointed to by the interface could be nil -- this checks
// both, safely
func IsNil(it interface{}) bool {
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
	if IsNil(it) {
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

// ToInt robustlys converts anything to an int64
func ToInt(it interface{}) (int64, bool) {
	if IsNil(it) {
		return 0, false
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

// ToFloat robustly converts anything to a Float64
func ToFloat(it interface{}) (float64, bool) {
	if IsNil(it) {
		return 0.0, false
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

// ToFloat32 robustly converts anything to a Float64
func ToFloat32(it interface{}) (float32, bool) {
	if IsNil(it) {
		return float32(0.0), false
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
	if IsNil(it) {
		return "nil"
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
	case vk == reflect.String: // todo: what about []byte?
		return v.String()
	default:
		strer, ok := it.(fmt.Stringer) // will fail if not impl
		if !ok {
			return fmt.Sprintf("%v", it)
		}
		return strer.String()
	}
}

// SetRobust robustly sets the to value from the from value -- to must be a
// pointer-to -- only for basic field values -- use copier package for more
// complex cases
func SetRobust(to, from interface{}) bool {
	if IsNil(to) {
		return false
	}
	v := reflect.ValueOf(to)
	vnp := NonPtrValue(v)
	if ValueIsZero(vnp) {
		return false
	}
	typ := vnp.Type()
	vp := PtrValue(v)
	vk := vnp.Kind()
	if !vp.Elem().CanSet() {
		log.Printf("ki.SetRobust 'to' cannot be set -- must be a variable or field, not a const or tmp or other value that cannot be set.  Value info: %v\n", vp)
		return false
	}
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		fm, ok := ToInt(from)
		if ok {
			vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
			return true
		}
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		fm, ok := ToInt(from)
		if ok {
			vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
			return true
		}
	case vk == reflect.Bool:
		fm, ok := ToBool(from)
		if ok {
			vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
			return true
		}
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		fm, ok := ToFloat(from)
		if ok {
			vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
			return true
		}
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		// cv := v.Complex()
		// rv := strconv.FormatFloat(real(cv), 'G', -1, 64) + "," + strconv.FormatFloat(imag(cv), 'G', -1, 64)
		// return rv, true
	case vk == reflect.String: // todo: what about []byte?
		fm := ToString(from)
		vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return true
	}

	fv := reflect.ValueOf(from)
	// Just set it if possible to assign
	if fv.Type().AssignableTo(typ) {
		vp.Elem().Set(fv)
		return true
	}
	return false
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

func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MinInt(a, b int) int {
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
