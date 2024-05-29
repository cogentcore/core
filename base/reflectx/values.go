// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package reflectx provides a collection of helpers for the reflect
// package in the Go standard library.
package reflectx

import (
	"encoding/json"
	"fmt"
	"image/color"
	"reflect"
	"strconv"
	"time"

	"cogentcore.org/core/base/bools"
	"cogentcore.org/core/base/elide"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/enums"
)

// AnyIsNil checks if an interface value is nil. The interface itself
// could be nil, or the value pointed to by the interface could be nil.
// This safely checks both.
func AnyIsNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	vk := rv.Kind()
	if vk == reflect.Pointer || vk == reflect.Interface || vk == reflect.Map || vk == reflect.Slice || vk == reflect.Func || vk == reflect.Chan {
		return rv.IsNil()
	}
	return false
}

// KindIsBasic returns whether the given [reflect.Kind] is a basic,
// elemental type such as Int, Float, etc.
func KindIsBasic(vk reflect.Kind) bool {
	return vk >= reflect.Bool && vk <= reflect.Complex128
}

// ToBool robustly converts to a bool any basic elemental type
// (including pointers to such) using a big type switch organized
// for greatest efficiency. It tries the [bools.Booler]
// interface if not a bool type. It falls back on reflection when all
// else fails.
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
	npv := Underlying(reflect.ValueOf(v))
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
	npv := Underlying(reflect.ValueOf(v))
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
	npv := Underlying(reflect.ValueOf(v))
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
	npv := Underlying(reflect.ValueOf(v))
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
// If everything else fails, it uses fmt.Sprintf("%v") which always works,
// so there is no need for an error return value. It returns "nil" for any nil
// pointers, and byte is converted as string(byte), not the decimal representation.
func ToString(v any) string {
	nilstr := "nil"
	if AnyIsNil(v) {
		return nilstr
	}
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
	npv := Underlying(reflect.ValueOf(v))
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
		eltyp := SliceElementType(v)
		if eltyp.Kind() == reflect.Uint8 { // []byte
			return string(v.([]byte))
		}
		fallthrough
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToStringPrec robustly converts anything to a String using given precision
// for converting floating values; using a value like 6 truncates the
// nuisance random imprecision of actual floating point values due to the
// fact that they are represented with binary bits.
// Otherwise is identical to ToString for any other cases.
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
// The 'to' value must be a pointer. It copies slices and maps robustly,
// and it can set a struct, slice, or map from a JSON-formatted string
// value. It also handles many other cases, so it is unlikely to fail.
//
// Note that maps are not reset prior to setting, whereas slices are
// set to be fully equivalent to the source slice.
func SetRobust(to, from any) error {
	if sa, ok := to.(SetAnyer); ok {
		err := sa.SetAny(from)
		if err != nil {
			return err
		}
		return nil
	}
	if ss, ok := to.(SetStringer); ok {
		if s, ok := from.(string); ok {
			err := ss.SetString(s)
			if err != nil {
				return err
			}
			return nil
		}
	}
	if es, ok := to.(enums.EnumSetter); ok {
		if en, ok := from.(enums.Enum); ok {
			es.SetInt64(en.Int64())
			return nil
		}
		if str, ok := from.(string); ok {
			return es.SetString(str)
		}
		fm, err := ToInt(from)
		if err != nil {
			return err
		}
		es.SetInt64(fm)
		return nil
	}

	if bv, ok := to.(bools.BoolSetter); ok {
		fb, err := ToBool(from)
		if err != nil {
			return err
		}
		bv.SetBool(fb)
		return nil
	}
	if td, ok := to.(*time.Duration); ok {
		if fs, ok := from.(string); ok {
			fd, err := time.ParseDuration(fs)
			if err != nil {
				return err
			}
			*td = fd
			return nil
		}
	}
	if cd, ok := to.(*color.RGBA); ok {
		fc, err := colors.FromAny(from)
		if err != nil {
			return err
		}
		*cd = fc
		return nil
	}

	if AnyIsNil(to) {
		return fmt.Errorf("got nil destination value")
	}
	v := reflect.ValueOf(to)
	pointer := UnderlyingPointer(v)
	typ := pointer.Elem().Type()
	kind := typ.Kind()
	if !pointer.Elem().CanSet() {
		return fmt.Errorf("destination value cannot be set; it must be a variable or field, not a const or tmp or other value that cannot be set (value: %v of type %T)", pointer, pointer)
	}

	switch {
	case kind >= reflect.Int && kind <= reflect.Int64:
		fm, err := ToInt(from)
		if err != nil {
			return err
		}
		pointer.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return nil
	case kind >= reflect.Uint && kind <= reflect.Uint64:
		fm, err := ToInt(from)
		if err != nil {
			return err
		}
		pointer.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return nil
	case kind == reflect.Bool:
		fm, err := ToBool(from)
		if err != nil {
			return err
		}
		pointer.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return nil
	case kind >= reflect.Float32 && kind <= reflect.Float64:
		fm, err := ToFloat(from)
		if err != nil {
			return err
		}
		pointer.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return nil
	case kind == reflect.String:
		fm := ToString(from)
		pointer.Elem().Set(reflect.ValueOf(fm).Convert(typ))
		return nil
	case kind == reflect.Struct:
		if NonPointerType(reflect.TypeOf(from)).Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(from)), to) // todo: this is not working -- see what marshal says, etc
			if err != nil {
				marsh, _ := json.Marshal(to)
				return fmt.Errorf("error setting struct from string: %w (example format for string: %s)", err, string(marsh))
			}
			return nil
		}
	case kind == reflect.Slice:
		if NonPointerType(reflect.TypeOf(from)).Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(from)), to)
			if err != nil {
				marsh, _ := json.Marshal(to)
				return fmt.Errorf("error setting slice from string: %w (example format for string: %s)", err, string(marsh))
			}
			return nil
		}
		return CopySliceRobust(to, from)
	case kind == reflect.Map:
		if NonPointerType(reflect.TypeOf(from)).Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(from)), to)
			if err != nil {
				marsh, _ := json.Marshal(to)
				return fmt.Errorf("error setting map from string: %w (example format for string: %s)", err, string(marsh))
			}
			return nil
		}
		return CopyMapRobust(to, from)
	}

	fv := reflect.ValueOf(from)
	if fv.Type().AssignableTo(typ) {
		pointer.Elem().Set(fv)
		return nil
	}
	fv = Underlying(reflect.ValueOf(from))
	if fv.Type().AssignableTo(typ) {
		pointer.Elem().Set(fv)
		return nil
	}
	tos := elide.End(fmt.Sprintf("%v", to), 40)
	fms := elide.End(fmt.Sprintf("%v", from), 40)
	return fmt.Errorf("unable to set value %s of type %T (using underlying type: %s) from value %s of type %T (using underlying type: %s): not a supported type pair and direct assigning is not possible", tos, to, typ.String(), fms, from, fv.Type().String())
}
