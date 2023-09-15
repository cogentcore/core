// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
)

// Has convenience functions for converting any (e.g. properties) to given
// types uses the "ok" bool mechanism to report failure -- are as robust and
// general as possible.
//
// WARNING: these violate many of the type-safety features of Go but OTOH give
// maximum robustness, appropriate for the world of end-user settable
// properties, and deal with most common-sense cases, e.g., string <-> number,
// etc.  nil values return !ok

type Signed interface {
	int | int8 | int16 | int32 | int64
}

type Unsigned interface {
	uint | uint8 | uint16 | uint32 | uint64
}

type Integer interface {
	Signed | Unsigned
}

type Float interface {
	float32 | float64
}

type Number interface {
	Signed | Unsigned | Float
}

// ConvertNumber converts any number to any other, using generics
func ConvertNumber[T1 Number, T2 Number](dst *T1, v T2) { *dst = T1(v) }

// AnyIsNil checks if an interface value is nil -- the interface itself could be
// nil, or the value pointed to by the interface could be nil -- this checks
// both, safely
// gopy:interface=handle
func AnyIsNil(it any) bool {
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

// KindIsBasic returns true if the reflect.Kind is a basic, elemental
// type such as Int, Float, etc
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

// ToBool robustly converts anything to a bool
// gopy:interface=handle
func ToBool(it any) (bool, bool) {
	// first check for most likely cases for greatest efficiency
	switch bt := it.(type) {
	case bool:
		return bt, true
	case *bool:
		return *bt, true
	case int:
		return bt != 0, true
	case *int:
		return *bt != 0, true
	case int32:
		return bt != 0, true
	case int64:
		return bt != 0, true
	case byte:
		return bt != 0, true
	case float64:
		return bt != 0, true
	case *float64:
		return *bt != 0, true
	case float32:
		return bt != 0, true
	case *float32:
		return *bt != 0, true
	case string:
		r, err := strconv.ParseBool(bt)
		if err != nil {
			return false, false
		}
		return r, true
	case *string:
		r, err := strconv.ParseBool(*bt)
		if err != nil {
			return false, false
		}
		return r, true
	}

	// then fall back on reflection
	if AnyIsNil(it) {
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

// ToInt robustly converts anything to an int64
// gopy:interface=handle
func ToInt(it any) (int64, bool) {
	// first check for most likely cases for greatest efficiency
	switch it := it.(type) {
	case bool:
		if it {
			return 1, true
		}
		return 0, true
	case *bool:
		if *it {
			return 1, true
		}
		return 0, true
	case int:
		return int64(it), true
	case *int:
		return int64(*it), true
	case int32:
		return int64(it), true
	case *int32:
		return int64(*it), true
	case int64:
		return it, true
	case *int64:
		return *it, true
	case byte:
		return int64(it), true
	case *byte:
		return int64(*it), true
	case float64:
		return int64(it), true
	case *float64:
		return int64(*it), true
	case float32:
		return int64(it), true
	case *float32:
		return int64(*it), true
	case string:
		r, err := strconv.ParseInt(it, 0, 64)
		if err != nil {
			return 0, false
		}
		return r, true
	case *string:
		r, err := strconv.ParseInt(*it, 0, 64)
		if err != nil {
			return 0, false
		}
		return r, true
	}

	// then fall back on reflection
	if AnyIsNil(it) {
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
// gopy:interface=handle
func ToFloat(it any) (float64, bool) {
	// first check for most likely cases for greatest efficiency
	switch it := it.(type) {
	case bool:
		if it {
			return 1, true
		}
		return 0, true
	case *bool:
		if *it {
			return 1, true
		}
		return 0, true
	case int:
		return float64(it), true
	case *int:
		return float64(*it), true
	case int32:
		return float64(it), true
	case *int32:
		return float64(*it), true
	case int64:
		return float64(it), true
	case *int64:
		return float64(*it), true
	case byte:
		return float64(it), true
	case *byte:
		return float64(*it), true
	case float64:
		return it, true
	case *float64:
		return *it, true
	case float32:
		return float64(it), true
	case *float32:
		return float64(*it), true
	case string:
		r, err := strconv.ParseFloat(it, 64)
		if err != nil {
			return 0.0, false
		}
		return r, true
	case *string:
		r, err := strconv.ParseFloat(*it, 64)
		if err != nil {
			return 0.0, false
		}
		return r, true
	}

	// then fall back on reflection
	if AnyIsNil(it) {
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

// ToFloat32 robustly converts anything to a Float32
// gopy:interface=handle
func ToFloat32(it any) (float32, bool) {
	// first check for most likely cases for greatest efficiency
	switch it := it.(type) {
	case bool:
		if it {
			return 1, true
		}
		return 0, true
	case *bool:
		if *it {
			return 1, true
		}
		return 0, true
	case int:
		return float32(it), true
	case *int:
		return float32(*it), true
	case int32:
		return float32(it), true
	case *int32:
		return float32(*it), true
	case int64:
		return float32(it), true
	case *int64:
		return float32(*it), true
	case byte:
		return float32(it), true
	case *byte:
		return float32(*it), true
	case float64:
		return float32(it), true
	case *float64:
		return float32(*it), true
	case float32:
		return it, true
	case *float32:
		return *it, true
	case string:
		r, err := strconv.ParseFloat(it, 32)
		if err != nil {
			return 0.0, false
		}
		return float32(r), true
	case *string:
		r, err := strconv.ParseFloat(*it, 32)
		if err != nil {
			return 0.0, false
		}
		return float32(r), true
	}

	// then fall back on reflection
	if AnyIsNil(it) {
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
// gopy:interface=handle
func ToString(it any) string {
	// first check for most likely cases for greatest efficiency
	switch it := it.(type) {
	case string:
		return it
	case *string:
		return *it
	case bool:
		if it {
			return "true"
		}
		return "false"
	case *bool:
		if *it {
			return "true"
		}
		return "false"
	case int:
		return strconv.FormatInt(int64(it), 10)
	case *int:
		return strconv.FormatInt(int64(*it), 10)
	case int32:
		return strconv.FormatInt(int64(it), 10)
	case *int32:
		return strconv.FormatInt(int64(*it), 10)
	case int64:
		return strconv.FormatInt(it, 10)
	case *int64:
		return strconv.FormatInt(*it, 10)
	case byte:
		return strconv.FormatInt(int64(it), 10)
	case *byte:
		return strconv.FormatInt(int64(*it), 10)
	case float64:
		return strconv.FormatFloat(it, 'G', -1, 64)
	case *float64:
		return strconv.FormatFloat(*it, 'G', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(it), 'G', -1, 32)
	case *float32:
		return strconv.FormatFloat(float64(*it), 'G', -1, 32)
	case uintptr:
		return fmt.Sprintf("%#x", uintptr(it))
	case *uintptr:
		return fmt.Sprintf("%#x", uintptr(*it))
	}

	if stringer, ok := it.(fmt.Stringer); ok {
		return stringer.String()
	}
	if AnyIsNil(it) {
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
// gopy:interface=handle
func ToStringPrec(it any, prec int) string {
	if AnyIsNil(it) {
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

// SetRobust robustly sets the 'to' value from the 'from' value.
// destination must be a pointer-to. Copies slices and maps robustly,
// and can set a struct, slice or map from a JSON-formatted string from value.
// gopy:interface=handle
func SetRobust(to, frm any) bool {
	if AnyIsNil(to) {
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
	case vk == reflect.Struct:
		if NonPtrType(reflect.TypeOf(frm)).Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(frm)), to) // todo: this is not working -- see what marshal says, etc
			if err != nil {
				marsh, _ := json.Marshal(to)
				log.Println("laser.SetRobust, struct from string:", err, "for example:", string(marsh))
			}
			return err == nil
		}
	case vk == reflect.Slice:
		if NonPtrType(reflect.TypeOf(frm)).Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(frm)), to)
			if err != nil {
				marsh, _ := json.Marshal(to)
				log.Println("laser.SetRobust, slice from string:", err, "for example:", string(marsh))
			}
			return err == nil
		}
		err := CopySliceRobust(to, frm)
		return err == nil
	case vk == reflect.Map:
		if NonPtrType(reflect.TypeOf(frm)).Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(frm)), to)
			if err != nil {
				marsh, _ := json.Marshal(to)
				log.Println("laser.SetRobust, map from string:", err, "for example:", string(marsh))
			}
			return err == nil
		}
		err := CopyMapRobust(to, frm)
		return err == nil
	}

	fv := reflect.ValueOf(frm)
	// Just set it if possible to assign
	if fv.Type().AssignableTo(typ) {
		vp.Elem().Set(fv)
		return true
	}
	return false
}
