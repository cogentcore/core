// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"goki.dev/enums"
	"goki.dev/glop/bools"
)

// Has convenience functions for converting any (e.g. properties) to given
// types uses the "ok" bool mechanism to report failure -- are as robust and
// general as possible.
//
// WARNING: these violate many of the type-safety features of Go but OTOH give
// maximum robustness, appropriate for the world of end-user settable
// properties, and deal with most common-sense cases, e.g., string <-> number,
// etc.  nil values return !ok

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

// ToBool robustly converts to a bool any basic elemental type
// (including pointers to such) using a big type switch organized
// for greatest efficiency. It tries the [goki.dev/glop/bools.Booler]
// interface if not a bool type. It falls back on reflection when all
// else fails.
//
//gopy:interface=handle
func ToBool(v any) (bool, error) {
	switch vt := v.(type) {
	case bool:
		return vt, nil
	case *bool:
		if vt == nil {
			return false, fmt.Errorf("got nil *bool")
		}
		return *vt, nil
	}

	if br, ok := v.(bools.Booler); ok {
		return br.Bool(), nil
	}

	switch vt := v.(type) {
	case int:
		return vt != 0, nil
	case *int:
		if vt == nil {
			return false, fmt.Errorf("got nil *int")
		}
		return *vt != 0, nil
	case int32:
		return vt != 0, nil
	case *int32:
		if vt == nil {
			return false, fmt.Errorf("got nil *int32")
		}
		return *vt != 0, nil
	case int64:
		return vt != 0, nil
	case *int64:
		if vt == nil {
			return false, fmt.Errorf("got nil *int64")
		}
		return *vt != 0, nil
	case uint8:
		return vt != 0, nil
	case *uint8:
		if vt == nil {
			return false, fmt.Errorf("got nil *uint8")
		}
		return *vt != 0, nil
	case float64:
		return vt != 0, nil
	case *float64:
		if vt == nil {
			return false, fmt.Errorf("got nil *float64")
		}
		return *vt != 0, nil
	case float32:
		return vt != 0, nil
	case *float32:
		if vt == nil {
			return false, fmt.Errorf("got nil *float32")
		}
		return *vt != 0, nil
	case string:
		r, err := strconv.ParseBool(vt)
		if err != nil {
			return false, err
		}
		return r, nil
	case *string:
		if vt == nil {
			return false, fmt.Errorf("got nil *string")
		}
		r, err := strconv.ParseBool(*vt)
		if err != nil {
			return false, err
		}
		return r, nil
	case int8:
		return vt != 0, nil
	case *int8:
		if vt == nil {
			return false, fmt.Errorf("got nil *int8")
		}
		return *vt != 0, nil
	case int16:
		return vt != 0, nil
	case *int16:
		if vt == nil {
			return false, fmt.Errorf("got nil *int16")
		}
		return *vt != 0, nil
	case uint:
		return vt != 0, nil
	case *uint:
		if vt == nil {
			return false, fmt.Errorf("got nil *uint")
		}
		return *vt != 0, nil
	case uint16:
		return vt != 0, nil
	case *uint16:
		if vt == nil {
			return false, fmt.Errorf("got nil *uint16")
		}
		return *vt != 0, nil
	case uint32:
		return vt != 0, nil
	case *uint32:
		if vt == nil {
			return false, fmt.Errorf("got nil *uint32")
		}
		return *vt != 0, nil
	case uint64:
		return vt != 0, nil
	case *uint64:
		if vt == nil {
			return false, fmt.Errorf("got nil *uint64")
		}
		return *vt != 0, nil
	case uintptr:
		return vt != 0, nil
	case *uintptr:
		if vt == nil {
			return false, fmt.Errorf("got nil *uintptr")
		}
		return *vt != 0, nil
	}

	// then fall back on reflection
	if AnyIsNil(v) {
		return false, fmt.Errorf("got nil value of type %T", v)
	}
	npv := NonPtrValue(reflect.ValueOf(v))
	vk := npv.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return (npv.Int() != 0), nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return (npv.Uint() != 0), nil
	case vk == reflect.Bool:
		return npv.Bool(), nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return (npv.Float() != 0.0), nil
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return (real(npv.Complex()) != 0.0), nil
	case vk == reflect.String:
		r, err := strconv.ParseBool(npv.String())
		if err != nil {
			return false, err
		}
		return r, nil
	default:
		return false, fmt.Errorf("got value %v of unsupported type %T", v, v)
	}
}

// ToInt robustly converts to an int64 any basic elemental type
// (including pointers to such) using a big type switch organized
// for greatest efficiency, only falling back on reflection when all
// else fails.
//
//gopy:interface=handle
func ToInt(v any) (int64, error) {
	switch vt := v.(type) {
	case int:
		return int64(vt), nil
	case *int:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int")
		}
		return int64(*vt), nil
	case int32:
		return int64(vt), nil
	case *int32:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int32")
		}
		return int64(*vt), nil
	case int64:
		return vt, nil
	case *int64:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int64")
		}
		return *vt, nil
	case uint8:
		return int64(vt), nil
	case *uint8:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint8")
		}
		return int64(*vt), nil
	case float64:
		return int64(vt), nil
	case *float64:
		if vt == nil {
			return 0, fmt.Errorf("got nil *float64")
		}
		return int64(*vt), nil
	case float32:
		return int64(vt), nil
	case *float32:
		if vt == nil {
			return 0, fmt.Errorf("got nil *float32")
		}
		return int64(*vt), nil
	case bool:
		if vt {
			return 1, nil
		}
		return 0, nil
	case *bool:
		if vt == nil {
			return 0, fmt.Errorf("got nil *bool")
		}
		if *vt {
			return 1, nil
		}
		return 0, nil
	case string:
		r, err := strconv.ParseInt(vt, 0, 64)
		if err != nil {
			return 0, err
		}
		return r, nil
	case *string:
		if vt == nil {
			return 0, fmt.Errorf("got nil *string")
		}
		r, err := strconv.ParseInt(*vt, 0, 64)
		if err != nil {
			return 0, err
		}
		return r, nil
	case enums.Enum:
		return vt.Int64(), nil
	case int8:
		return int64(vt), nil
	case *int8:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int8")
		}
		return int64(*vt), nil
	case int16:
		return int64(vt), nil
	case *int16:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int16")
		}
		return int64(*vt), nil
	case uint:
		return int64(vt), nil
	case *uint:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint")
		}
		return int64(*vt), nil
	case uint16:
		return int64(vt), nil
	case *uint16:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint16")
		}
		return int64(*vt), nil
	case uint32:
		return int64(vt), nil
	case *uint32:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint32")
		}
		return int64(*vt), nil
	case uint64:
		return int64(vt), nil
	case *uint64:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint64")
		}
		return int64(*vt), nil
	case uintptr:
		return int64(vt), nil
	case *uintptr:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uintptr")
		}
		return int64(*vt), nil
	}

	// then fall back on reflection
	if AnyIsNil(v) {
		return 0, fmt.Errorf("got nil value of type %T", v)
	}
	npv := NonPtrValue(reflect.ValueOf(v))
	vk := npv.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return npv.Int(), nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return int64(npv.Uint()), nil
	case vk == reflect.Bool:
		if npv.Bool() {
			return 1, nil
		}
		return 0, nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return int64(npv.Float()), nil
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return int64(real(npv.Complex())), nil
	case vk == reflect.String:
		r, err := strconv.ParseInt(npv.String(), 0, 64)
		if err != nil {
			return 0, err
		}
		return r, nil
	default:
		return 0, fmt.Errorf("got value %v of unsupported type %T", v, v)
	}
}

// ToFloat robustly converts to a float64 any basic elemental type
// (including pointers to such) using a big type switch organized for
// greatest efficiency, only falling back on reflection when all else fails.
//
//gopy:interface=handle
func ToFloat(v any) (float64, error) {
	switch vt := v.(type) {
	case float64:
		return vt, nil
	case *float64:
		if vt == nil {
			return 0, fmt.Errorf("got nil *float64")
		}
		return *vt, nil
	case float32:
		return float64(vt), nil
	case *float32:
		if vt == nil {
			return 0, fmt.Errorf("got nil *float32")
		}
		return float64(*vt), nil
	case int:
		return float64(vt), nil
	case *int:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int")
		}
		return float64(*vt), nil
	case int32:
		return float64(vt), nil
	case *int32:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int32")
		}
		return float64(*vt), nil
	case int64:
		return float64(vt), nil
	case *int64:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int64")
		}
		return float64(*vt), nil
	case uint8:
		return float64(vt), nil
	case *uint8:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint8")
		}
		return float64(*vt), nil
	case bool:
		if vt {
			return 1, nil
		}
		return 0, nil
	case *bool:
		if vt == nil {
			return 0, fmt.Errorf("got nil *bool")
		}
		if *vt {
			return 1, nil
		}
		return 0, nil
	case string:
		r, err := strconv.ParseFloat(vt, 64)
		if err != nil {
			return 0.0, err
		}
		return r, nil
	case *string:
		if vt == nil {
			return 0, fmt.Errorf("got nil *string")
		}
		r, err := strconv.ParseFloat(*vt, 64)
		if err != nil {
			return 0.0, err
		}
		return r, nil
	case int8:
		return float64(vt), nil
	case *int8:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int8")
		}
		return float64(*vt), nil
	case int16:
		return float64(vt), nil
	case *int16:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int16")
		}
		return float64(*vt), nil
	case uint:
		return float64(vt), nil
	case *uint:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint")
		}
		return float64(*vt), nil
	case uint16:
		return float64(vt), nil
	case *uint16:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint16")
		}
		return float64(*vt), nil
	case uint32:
		return float64(vt), nil
	case *uint32:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint32")
		}
		return float64(*vt), nil
	case uint64:
		return float64(vt), nil
	case *uint64:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint64")
		}
		return float64(*vt), nil
	case uintptr:
		return float64(vt), nil
	case *uintptr:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uintptr")
		}
		return float64(*vt), nil
	}

	// then fall back on reflection
	if AnyIsNil(v) {
		return 0, fmt.Errorf("got nil value of type %T", v)
	}
	npv := NonPtrValue(reflect.ValueOf(v))
	vk := npv.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return float64(npv.Int()), nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return float64(npv.Uint()), nil
	case vk == reflect.Bool:
		if npv.Bool() {
			return 1, nil
		}
		return 0, nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return npv.Float(), nil
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return real(npv.Complex()), nil
	case vk == reflect.String:
		r, err := strconv.ParseFloat(npv.String(), 64)
		if err != nil {
			return 0, err
		}
		return r, nil
	default:
		return 0, fmt.Errorf("got value %v of unsupported type %T", v, v)
	}
}

// ToFloat32 robustly converts to a float32 any basic elemental type
// (including pointers to such) using a big type switch organized for
// greatest efficiency, only falling back on reflection when all else fails.
//
//gopy:interface=handle
func ToFloat32(v any) (float32, error) {
	switch vt := v.(type) {
	case float32:
		return vt, nil
	case *float32:
		if vt == nil {
			return 0, fmt.Errorf("got nil *float32")
		}
		return *vt, nil
	case float64:
		return float32(vt), nil
	case *float64:
		if vt == nil {
			return 0, fmt.Errorf("got nil *float64")
		}
		return float32(*vt), nil
	case int:
		return float32(vt), nil
	case *int:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int")
		}
		return float32(*vt), nil
	case int32:
		return float32(vt), nil
	case *int32:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int32")
		}
		return float32(*vt), nil
	case int64:
		return float32(vt), nil
	case *int64:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int64")
		}
		return float32(*vt), nil
	case uint8:
		return float32(vt), nil
	case *uint8:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint8")
		}
		return float32(*vt), nil
	case bool:
		if vt {
			return 1, nil
		}
		return 0, nil
	case *bool:
		if vt == nil {
			return 0, fmt.Errorf("got nil *bool")
		}
		if *vt {
			return 1, nil
		}
		return 0, nil
	case string:
		r, err := strconv.ParseFloat(vt, 32)
		if err != nil {
			return 0, err
		}
		return float32(r), nil
	case *string:
		if vt == nil {
			return 0, fmt.Errorf("got nil *string")
		}
		r, err := strconv.ParseFloat(*vt, 32)
		if err != nil {
			return 0, err
		}
		return float32(r), nil
	case int8:
		return float32(vt), nil
	case *int8:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int8")
		}
		return float32(*vt), nil
	case int16:
		return float32(vt), nil
	case *int16:
		if vt == nil {
			return 0, fmt.Errorf("got nil *int8")
		}
		return float32(*vt), nil
	case uint:
		return float32(vt), nil
	case *uint:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint")
		}
		return float32(*vt), nil
	case uint16:
		return float32(vt), nil
	case *uint16:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint16")
		}
		return float32(*vt), nil
	case uint32:
		return float32(vt), nil
	case *uint32:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint32")
		}
		return float32(*vt), nil
	case uint64:
		return float32(vt), nil
	case *uint64:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uint64")
		}
		return float32(*vt), nil
	case uintptr:
		return float32(vt), nil
	case *uintptr:
		if vt == nil {
			return 0, fmt.Errorf("got nil *uintptr")
		}
		return float32(*vt), nil
	}

	// then fall back on reflection
	if AnyIsNil(v) {
		return 0, fmt.Errorf("got nil value of type %T", v)
	}
	npv := NonPtrValue(reflect.ValueOf(v))
	vk := npv.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return float32(npv.Int()), nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return float32(npv.Uint()), nil
	case vk == reflect.Bool:
		if npv.Bool() {
			return 1, nil
		}
		return 0, nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return float32(npv.Float()), nil
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return float32(real(npv.Complex())), nil
	case vk == reflect.String:
		r, err := strconv.ParseFloat(npv.String(), 32)
		if err != nil {
			return 0, err
		}
		return float32(r), nil
	default:
		return 0, fmt.Errorf("got value %v of unsupported type %T", v, v)
	}
}

// ToString robustly converts anything to a String
// using a big type switch organized for greatest efficiency.
// First checks for string or []byte and returns that immediately,
// then checks for the Stringer interface as the preferred conversion
// (e.g., for enums), and then falls back on strconv calls for numeric types.
// If everything else fails, it uses Sprintf("%v") which always works,
// so there is no need for an error return value.
//   - returns "nil" for any nil pointers
//   - byte is converted as string(byte) not the decimal representation
//
//gopy:interface=handle
func ToString(v any) string {
	nilstr := "nil"
	switch vt := v.(type) {
	case string:
		return vt
	case *string:
		if vt == nil {
			return nilstr
		}
		return *vt
	case []byte:
		return string(vt)
	case *[]byte:
		if vt == nil {
			return nilstr
		}
		return string(*vt)
	}

	if stringer, ok := v.(fmt.Stringer); ok {
		return stringer.String()
	}

	switch vt := v.(type) {
	case bool:
		if vt {
			return "true"
		}
		return "false"
	case *bool:
		if vt == nil {
			return nilstr
		}
		if *vt {
			return "true"
		}
		return "false"
	case int:
		return strconv.FormatInt(int64(vt), 10)
	case *int:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatInt(int64(*vt), 10)
	case int32:
		return strconv.FormatInt(int64(vt), 10)
	case *int32:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatInt(int64(*vt), 10)
	case int64:
		return strconv.FormatInt(vt, 10)
	case *int64:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatInt(*vt, 10)
	case uint8: // byte, converts as string char
		return string(vt)
	case *uint8:
		if vt == nil {
			return nilstr
		}
		return string(*vt)
	case float64:
		return strconv.FormatFloat(vt, 'G', -1, 64)
	case *float64:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatFloat(*vt, 'G', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(vt), 'G', -1, 32)
	case *float32:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatFloat(float64(*vt), 'G', -1, 32)
	case uintptr:
		return fmt.Sprintf("%#x", uintptr(vt))
	case *uintptr:
		if vt == nil {
			return nilstr
		}
		return fmt.Sprintf("%#x", uintptr(*vt))

	case int8:
		return strconv.FormatInt(int64(vt), 10)
	case *int8:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatInt(int64(*vt), 10)
	case int16:
		return strconv.FormatInt(int64(vt), 10)
	case *int16:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatInt(int64(*vt), 10)
	case uint:
		return strconv.FormatInt(int64(vt), 10)
	case *uint:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatInt(int64(*vt), 10)
	case uint16:
		return strconv.FormatInt(int64(vt), 10)
	case *uint16:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatInt(int64(*vt), 10)
	case uint32:
		return strconv.FormatInt(int64(vt), 10)
	case *uint32:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatInt(int64(*vt), 10)
	case uint64:
		return strconv.FormatInt(int64(vt), 10)
	case *uint64:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatInt(int64(*vt), 10)
	case complex64:
		return strconv.FormatFloat(float64(real(vt)), 'G', -1, 32) + "," + strconv.FormatFloat(float64(imag(vt)), 'G', -1, 32)
	case *complex64:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatFloat(float64(real(*vt)), 'G', -1, 32) + "," + strconv.FormatFloat(float64(imag(*vt)), 'G', -1, 32)
	case complex128:
		return strconv.FormatFloat(real(vt), 'G', -1, 64) + "," + strconv.FormatFloat(imag(vt), 'G', -1, 64)
	case *complex128:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatFloat(real(*vt), 'G', -1, 64) + "," + strconv.FormatFloat(imag(*vt), 'G', -1, 64)
	}

	// then fall back on reflection
	if AnyIsNil(v) {
		return nilstr
	}
	npv := NonPtrValue(reflect.ValueOf(v))
	vk := npv.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return strconv.FormatInt(npv.Int(), 10)
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return strconv.FormatUint(npv.Uint(), 10)
	case vk == reflect.Bool:
		return strconv.FormatBool(npv.Bool())
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return strconv.FormatFloat(npv.Float(), 'G', -1, 64)
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		cv := npv.Complex()
		rv := strconv.FormatFloat(real(cv), 'G', -1, 64) + "," + strconv.FormatFloat(imag(cv), 'G', -1, 64)
		return rv
	case vk == reflect.String:
		return npv.String()
	case vk == reflect.Slice:
		eltyp := SliceElType(v)
		if eltyp.Kind() == reflect.Uint8 { // []byte
			return string(v.([]byte))
		}
		fallthrough
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToStringPrec robustly converts anything to a String using given precision
// for converting floating values -- using a value like 6 truncates the
// nuisance random imprecision of actual floating point values due to the
// fact that they are represented with binary bits.
// Otherwise is identical to ToString for any other cases.
//
//gopy:interface=handle
func ToStringPrec(v any, prec int) string {
	nilstr := "nil"
	switch vt := v.(type) {
	case string:
		return vt
	case *string:
		if vt == nil {
			return nilstr
		}
		return *vt
	case []byte:
		return string(vt)
	case *[]byte:
		if vt == nil {
			return nilstr
		}
		return string(*vt)
	}

	if stringer, ok := v.(fmt.Stringer); ok {
		return stringer.String()
	}

	switch vt := v.(type) {
	case float64:
		return strconv.FormatFloat(vt, 'G', prec, 64)
	case *float64:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatFloat(*vt, 'G', prec, 64)
	case float32:
		return strconv.FormatFloat(float64(vt), 'G', prec, 32)
	case *float32:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatFloat(float64(*vt), 'G', prec, 32)
	case complex64:
		return strconv.FormatFloat(float64(real(vt)), 'G', prec, 32) + "," + strconv.FormatFloat(float64(imag(vt)), 'G', prec, 32)
	case *complex64:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatFloat(float64(real(*vt)), 'G', prec, 32) + "," + strconv.FormatFloat(float64(imag(*vt)), 'G', prec, 32)
	case complex128:
		return strconv.FormatFloat(real(vt), 'G', prec, 64) + "," + strconv.FormatFloat(imag(vt), 'G', prec, 64)
	case *complex128:
		if vt == nil {
			return nilstr
		}
		return strconv.FormatFloat(real(*vt), 'G', prec, 64) + "," + strconv.FormatFloat(imag(*vt), 'G', prec, 64)
	}
	return ToString(v)
}

// SetRobust robustly sets the 'to' value from the 'from' value.
// destination must be a pointer-to. Copies slices and maps robustly,
// and can set a struct, slice or map from a JSON-formatted string from value.
// Note that a map is _not_ reset prior to setting, whereas a slice length
// is set to the source length and is thus equivalent to the source slice.
//
//gopy:interface=handle
func SetRobust(to, frm any) error {
	if sa, ok := to.(SetAnyer); ok {
		err := sa.SetAny(frm)
		if err != nil {
			return err
		}
		return nil
	}
	if ss, ok := to.(SetStringer); ok {
		if s, ok := frm.(string); ok {
			err := ss.SetString(s)
			if err != nil {
				return err
			}
			return nil
		}
	}

	if AnyIsNil(to) {
		return fmt.Errorf("got nil destination value")
	}
	v := reflect.ValueOf(to)
	vnp := NonPtrValue(v)
	if !vnp.IsValid() {
		return fmt.Errorf("got invalid destination value %v of type %T", to, to)
	}
	typ := vnp.Type()
	vp := OnePtrValue(vnp)
	vk := vnp.Kind()
	if !vp.Elem().CanSet() {
		return fmt.Errorf("destination value cannot be set; it must be a variable or field, not a const or tmp or other value that cannot be set (value: %v of type %T)", vp, vp)
	}

	if es, ok := to.(enums.EnumSetter); ok {
		if en, ok := frm.(enums.Enum); ok {
			es.SetInt64(en.Int64())
			return nil
		}
		if str, ok := frm.(string); ok {
			return es.SetString(str)
		}
		fm, err := ToInt(frm)
		if err != nil {
			return err
		}
		es.SetInt64(fm)
		return nil
	}
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		fm, err := ToInt(frm)
		if err != nil {
			return err
		}
		vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		fm, err := ToInt(frm)
		if err != nil {
			return err
		}
		vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return nil
	case vk == reflect.Bool:
		fm, err := ToBool(frm)
		if err != nil {
			return err
		}
		vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		fm, err := ToFloat(frm)
		if err != nil {
			return err
		}
		vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return nil
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		// cv := v.Complex()
		// rv := strconv.FormatFloat(real(cv), 'G', -1, 64) + "," + strconv.FormatFloat(imag(cv), 'G', -1, 64)
		// return rv, nil
	case vk == reflect.String:
		fm := ToString(frm)
		vp.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return nil
	case vk == reflect.Struct:
		if NonPtrType(reflect.TypeOf(frm)).Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(frm)), to) // todo: this is not working -- see what marshal says, etc
			if err != nil {
				marsh, _ := json.Marshal(to)
				return fmt.Errorf("error setting struct from string: %w (example format for string: %s)", err, string(marsh))
			}
			return nil
		}
	case vk == reflect.Slice:
		if NonPtrType(reflect.TypeOf(frm)).Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(frm)), to)
			if err != nil {
				marsh, _ := json.Marshal(to)
				return fmt.Errorf("error setting slice from string: %w (example format for string: %s)", err, string(marsh))
			}
			return nil
		}
		return CopySliceRobust(to, frm)
	case vk == reflect.Map:
		if NonPtrType(reflect.TypeOf(frm)).Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(frm)), to)
			if err != nil {
				marsh, _ := json.Marshal(to)
				return fmt.Errorf("error setting map from string: %w (example format for string: %s)", err, string(marsh))
			}
			return nil
		}
		return CopyMapRobust(to, frm)
	}

	fv := reflect.ValueOf(frm)
	// Just set it if possible to assign
	if fv.Type().AssignableTo(typ) {
		vp.Elem().Set(fv)
		return nil
	}
	npfv := NonPtrValue(fv)
	if npfv.Type().AssignableTo(typ) {
		vp.Elem().Set(npfv)
	}
	return fmt.Errorf("unable to set value %v of type %T from value %v of type %T (not a supported type pair and direct assigning is not possible)", to, to, frm, frm)
}
