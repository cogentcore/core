// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package reflectx provides a collection of helpers for the reflect
// package in the Go standard library.
package reflectx

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"strconv"
	"time"

	"cogentcore.org/core/base/bools"
	"cogentcore.org/core/base/elide"
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/enums"
)

// IsNil returns whether the given value is nil or invalid.
// If it is a non-nillable type, it does not check whether
// it is nil to avoid panics.
func IsNil(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}

// KindIsBasic returns whether the given [reflect.Kind] is a basic,
// elemental type such as Int, Float, etc.
func KindIsBasic(vk reflect.Kind) bool {
	return vk >= reflect.Bool && vk <= reflect.Complex128
}

// KindIsNumber returns whether the given [reflect.Kind] is a numeric
// type such as Int, Float, etc.
func KindIsNumber(vk reflect.Kind) bool {
	return vk >= reflect.Int && vk <= reflect.Complex128
}

// KindIsInt returns whether the given [reflect.Kind] is an int
// type such as int, int32 etc.
func KindIsInt(vk reflect.Kind) bool {
	return vk >= reflect.Int && vk <= reflect.Uintptr
}

// KindIsFloat returns whether the given [reflect.Kind] is a
// float32 or float64.
func KindIsFloat(vk reflect.Kind) bool {
	return vk >= reflect.Float32 && vk <= reflect.Float64
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
			return false, errors.New("got nil *bool")
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
			return false, errors.New("got nil *int")
		}
		return *vt != 0, nil
	case int32:
		return vt != 0, nil
	case *int32:
		if vt == nil {
			return false, errors.New("got nil *int32")
		}
		return *vt != 0, nil
	case int64:
		return vt != 0, nil
	case *int64:
		if vt == nil {
			return false, errors.New("got nil *int64")
		}
		return *vt != 0, nil
	case uint8:
		return vt != 0, nil
	case *uint8:
		if vt == nil {
			return false, errors.New("got nil *uint8")
		}
		return *vt != 0, nil
	case float64:
		return vt != 0, nil
	case *float64:
		if vt == nil {
			return false, errors.New("got nil *float64")
		}
		return *vt != 0, nil
	case float32:
		return vt != 0, nil
	case *float32:
		if vt == nil {
			return false, errors.New("got nil *float32")
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
			return false, errors.New("got nil *string")
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
			return false, errors.New("got nil *int8")
		}
		return *vt != 0, nil
	case int16:
		return vt != 0, nil
	case *int16:
		if vt == nil {
			return false, errors.New("got nil *int16")
		}
		return *vt != 0, nil
	case uint:
		return vt != 0, nil
	case *uint:
		if vt == nil {
			return false, errors.New("got nil *uint")
		}
		return *vt != 0, nil
	case uint16:
		return vt != 0, nil
	case *uint16:
		if vt == nil {
			return false, errors.New("got nil *uint16")
		}
		return *vt != 0, nil
	case uint32:
		return vt != 0, nil
	case *uint32:
		if vt == nil {
			return false, errors.New("got nil *uint32")
		}
		return *vt != 0, nil
	case uint64:
		return vt != 0, nil
	case *uint64:
		if vt == nil {
			return false, errors.New("got nil *uint64")
		}
		return *vt != 0, nil
	case uintptr:
		return vt != 0, nil
	case *uintptr:
		if vt == nil {
			return false, errors.New("got nil *uintptr")
		}
		return *vt != 0, nil
	}

	// then fall back on reflection
	uv := Underlying(reflect.ValueOf(v))
	if IsNil(uv) {
		return false, fmt.Errorf("got nil value of type %T", v)
	}
	vk := uv.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return (uv.Int() != 0), nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return (uv.Uint() != 0), nil
	case vk == reflect.Bool:
		return uv.Bool(), nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return (uv.Float() != 0.0), nil
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return (real(uv.Complex()) != 0.0), nil
	case vk == reflect.String:
		r, err := strconv.ParseBool(uv.String())
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
			return 0, errors.New("got nil *int")
		}
		return int64(*vt), nil
	case int32:
		return int64(vt), nil
	case *int32:
		if vt == nil {
			return 0, errors.New("got nil *int32")
		}
		return int64(*vt), nil
	case int64:
		return vt, nil
	case *int64:
		if vt == nil {
			return 0, errors.New("got nil *int64")
		}
		return *vt, nil
	case uint8:
		return int64(vt), nil
	case *uint8:
		if vt == nil {
			return 0, errors.New("got nil *uint8")
		}
		return int64(*vt), nil
	case float64:
		return int64(vt), nil
	case *float64:
		if vt == nil {
			return 0, errors.New("got nil *float64")
		}
		return int64(*vt), nil
	case float32:
		return int64(vt), nil
	case *float32:
		if vt == nil {
			return 0, errors.New("got nil *float32")
		}
		return int64(*vt), nil
	case bool:
		if vt {
			return 1, nil
		}
		return 0, nil
	case *bool:
		if vt == nil {
			return 0, errors.New("got nil *bool")
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
			return 0, errors.New("got nil *string")
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
			return 0, errors.New("got nil *int8")
		}
		return int64(*vt), nil
	case int16:
		return int64(vt), nil
	case *int16:
		if vt == nil {
			return 0, errors.New("got nil *int16")
		}
		return int64(*vt), nil
	case uint:
		return int64(vt), nil
	case *uint:
		if vt == nil {
			return 0, errors.New("got nil *uint")
		}
		return int64(*vt), nil
	case uint16:
		return int64(vt), nil
	case *uint16:
		if vt == nil {
			return 0, errors.New("got nil *uint16")
		}
		return int64(*vt), nil
	case uint32:
		return int64(vt), nil
	case *uint32:
		if vt == nil {
			return 0, errors.New("got nil *uint32")
		}
		return int64(*vt), nil
	case uint64:
		return int64(vt), nil
	case *uint64:
		if vt == nil {
			return 0, errors.New("got nil *uint64")
		}
		return int64(*vt), nil
	case uintptr:
		return int64(vt), nil
	case *uintptr:
		if vt == nil {
			return 0, errors.New("got nil *uintptr")
		}
		return int64(*vt), nil
	}

	// then fall back on reflection
	uv := Underlying(reflect.ValueOf(v))
	if IsNil(uv) {
		return 0, fmt.Errorf("got nil value of type %T", v)
	}
	vk := uv.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return uv.Int(), nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return int64(uv.Uint()), nil
	case vk == reflect.Bool:
		if uv.Bool() {
			return 1, nil
		}
		return 0, nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return int64(uv.Float()), nil
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return int64(real(uv.Complex())), nil
	case vk == reflect.String:
		r, err := strconv.ParseInt(uv.String(), 0, 64)
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
			return 0, errors.New("got nil *float64")
		}
		return *vt, nil
	case float32:
		return float64(vt), nil
	case *float32:
		if vt == nil {
			return 0, errors.New("got nil *float32")
		}
		return float64(*vt), nil
	case int:
		return float64(vt), nil
	case *int:
		if vt == nil {
			return 0, errors.New("got nil *int")
		}
		return float64(*vt), nil
	case int32:
		return float64(vt), nil
	case *int32:
		if vt == nil {
			return 0, errors.New("got nil *int32")
		}
		return float64(*vt), nil
	case int64:
		return float64(vt), nil
	case *int64:
		if vt == nil {
			return 0, errors.New("got nil *int64")
		}
		return float64(*vt), nil
	case uint8:
		return float64(vt), nil
	case *uint8:
		if vt == nil {
			return 0, errors.New("got nil *uint8")
		}
		return float64(*vt), nil
	case bool:
		if vt {
			return 1, nil
		}
		return 0, nil
	case *bool:
		if vt == nil {
			return 0, errors.New("got nil *bool")
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
			return 0, errors.New("got nil *string")
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
			return 0, errors.New("got nil *int8")
		}
		return float64(*vt), nil
	case int16:
		return float64(vt), nil
	case *int16:
		if vt == nil {
			return 0, errors.New("got nil *int16")
		}
		return float64(*vt), nil
	case uint:
		return float64(vt), nil
	case *uint:
		if vt == nil {
			return 0, errors.New("got nil *uint")
		}
		return float64(*vt), nil
	case uint16:
		return float64(vt), nil
	case *uint16:
		if vt == nil {
			return 0, errors.New("got nil *uint16")
		}
		return float64(*vt), nil
	case uint32:
		return float64(vt), nil
	case *uint32:
		if vt == nil {
			return 0, errors.New("got nil *uint32")
		}
		return float64(*vt), nil
	case uint64:
		return float64(vt), nil
	case *uint64:
		if vt == nil {
			return 0, errors.New("got nil *uint64")
		}
		return float64(*vt), nil
	case uintptr:
		return float64(vt), nil
	case *uintptr:
		if vt == nil {
			return 0, errors.New("got nil *uintptr")
		}
		return float64(*vt), nil
	}

	// then fall back on reflection
	uv := Underlying(reflect.ValueOf(v))
	if IsNil(uv) {
		return 0, fmt.Errorf("got nil value of type %T", v)
	}
	vk := uv.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return float64(uv.Int()), nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return float64(uv.Uint()), nil
	case vk == reflect.Bool:
		if uv.Bool() {
			return 1, nil
		}
		return 0, nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return uv.Float(), nil
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return real(uv.Complex()), nil
	case vk == reflect.String:
		r, err := strconv.ParseFloat(uv.String(), 64)
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
			return 0, errors.New("got nil *float32")
		}
		return *vt, nil
	case float64:
		return float32(vt), nil
	case *float64:
		if vt == nil {
			return 0, errors.New("got nil *float64")
		}
		return float32(*vt), nil
	case int:
		return float32(vt), nil
	case *int:
		if vt == nil {
			return 0, errors.New("got nil *int")
		}
		return float32(*vt), nil
	case int32:
		return float32(vt), nil
	case *int32:
		if vt == nil {
			return 0, errors.New("got nil *int32")
		}
		return float32(*vt), nil
	case int64:
		return float32(vt), nil
	case *int64:
		if vt == nil {
			return 0, errors.New("got nil *int64")
		}
		return float32(*vt), nil
	case uint8:
		return float32(vt), nil
	case *uint8:
		if vt == nil {
			return 0, errors.New("got nil *uint8")
		}
		return float32(*vt), nil
	case bool:
		if vt {
			return 1, nil
		}
		return 0, nil
	case *bool:
		if vt == nil {
			return 0, errors.New("got nil *bool")
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
			return 0, errors.New("got nil *string")
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
			return 0, errors.New("got nil *int8")
		}
		return float32(*vt), nil
	case int16:
		return float32(vt), nil
	case *int16:
		if vt == nil {
			return 0, errors.New("got nil *int8")
		}
		return float32(*vt), nil
	case uint:
		return float32(vt), nil
	case *uint:
		if vt == nil {
			return 0, errors.New("got nil *uint")
		}
		return float32(*vt), nil
	case uint16:
		return float32(vt), nil
	case *uint16:
		if vt == nil {
			return 0, errors.New("got nil *uint16")
		}
		return float32(*vt), nil
	case uint32:
		return float32(vt), nil
	case *uint32:
		if vt == nil {
			return 0, errors.New("got nil *uint32")
		}
		return float32(*vt), nil
	case uint64:
		return float32(vt), nil
	case *uint64:
		if vt == nil {
			return 0, errors.New("got nil *uint64")
		}
		return float32(*vt), nil
	case uintptr:
		return float32(vt), nil
	case *uintptr:
		if vt == nil {
			return 0, errors.New("got nil *uintptr")
		}
		return float32(*vt), nil
	}

	// then fall back on reflection
	uv := Underlying(reflect.ValueOf(v))
	if IsNil(uv) {
		return 0, fmt.Errorf("got nil value of type %T", v)
	}
	vk := uv.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return float32(uv.Int()), nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return float32(uv.Uint()), nil
	case vk == reflect.Bool:
		if uv.Bool() {
			return 1, nil
		}
		return 0, nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return float32(uv.Float()), nil
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		return float32(real(uv.Complex())), nil
	case vk == reflect.String:
		r, err := strconv.ParseFloat(uv.String(), 32)
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
	// TODO: this reflection is unideal for performance, but we need it to prevent panics,
	// so this whole "greatest efficiency" type switch is kind of pointless.
	rv := reflect.ValueOf(v)
	if IsNil(rv) {
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
	uv := Underlying(rv)
	if IsNil(uv) {
		return nilstr
	}
	vk := uv.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return strconv.FormatInt(uv.Int(), 10)
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return strconv.FormatUint(uv.Uint(), 10)
	case vk == reflect.Bool:
		return strconv.FormatBool(uv.Bool())
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return strconv.FormatFloat(uv.Float(), 'G', -1, 64)
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		cv := uv.Complex()
		rv := strconv.FormatFloat(real(cv), 'G', -1, 64) + "," + strconv.FormatFloat(imag(cv), 'G', -1, 64)
		return rv
	case vk == reflect.String:
		return uv.String()
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
	rto := reflect.ValueOf(to)
	pto := UnderlyingPointer(rto)
	if IsNil(pto) {
		// If the original value is a non-nil pointer, we can just use it
		// even though the underlying pointer is nil (this happens when there
		// is a pointer to a nil pointer; see #1365).
		if !IsNil(rto) && rto.Kind() == reflect.Pointer {
			pto = rto
		} else {
			// Otherwise, we cannot recover any meaningful value.
			return errors.New("got nil destination value")
		}
	}
	pito := pto.Interface()

	totyp := pto.Elem().Type()
	tokind := totyp.Kind()
	if !pto.Elem().CanSet() {
		return fmt.Errorf("destination value cannot be set; it must be a variable or field, not a const or tmp or other value that cannot be set (value: %v of type %T)", pto, pto)
	}

	// images should not be copied per content: just set the pointer!
	// otherwise the original images (esp colors!) are altered.
	// TODO: #1394 notes the more general ambiguity about deep vs. shallow pointer copy.
	if img, ok := to.(*image.Image); ok {
		if fimg, ok := from.(image.Image); ok {
			*img = fimg
			return nil
		}
	}

	// first we do the generic AssignableTo case
	if rto.Kind() == reflect.Pointer {
		fv := reflect.ValueOf(from)
		if fv.IsValid() {
			if fv.Type().AssignableTo(totyp) {
				pto.Elem().Set(fv)
				return nil
			}
			ufvp := UnderlyingPointer(fv)
			if ufvp.IsValid() && ufvp.Type().AssignableTo(totyp) {
				pto.Elem().Set(ufvp)
				return nil
			}
			ufv := ufvp.Elem()
			if ufv.IsValid() && ufv.Type().AssignableTo(totyp) {
				pto.Elem().Set(ufv)
				return nil
			}
		} else {
			return nil
		}
	}

	if sa, ok := pito.(SetAnyer); ok {
		err := sa.SetAny(from)
		if err != nil {
			return err
		}
		return nil
	}
	if ss, ok := pito.(SetStringer); ok {
		if s, ok := from.(string); ok {
			err := ss.SetString(s)
			if err != nil {
				return err
			}
			return nil
		}
	}
	if es, ok := pito.(enums.EnumSetter); ok {
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

	if bv, ok := pito.(bools.BoolSetter); ok {
		fb, err := ToBool(from)
		if err != nil {
			return err
		}
		bv.SetBool(fb)
		return nil
	}
	if td, ok := pito.(*time.Duration); ok {
		if fs, ok := from.(string); ok {
			fd, err := time.ParseDuration(fs)
			if err != nil {
				return err
			}
			*td = fd
			return nil
		}
	}

	if fc, err := colors.FromAny(from); err == nil {
		switch c := pito.(type) {
		case *color.RGBA:
			*c = fc
			return nil
		case *image.Uniform:
			c.C = fc
			return nil
		case SetColorer:
			c.SetColor(fc)
			return nil
		case *image.Image:
			*c = colors.Uniform(fc)
			return nil
		}
	}

	ftyp := NonPointerType(reflect.TypeOf(from))

	switch {
	case tokind >= reflect.Int && tokind <= reflect.Int64:
		fm, err := ToInt(from)
		if err != nil {
			return err
		}
		pto.Elem().Set(reflect.ValueOf(fm).Convert(totyp))
		return nil
	case tokind >= reflect.Uint && tokind <= reflect.Uint64:
		fm, err := ToInt(from)
		if err != nil {
			return err
		}
		pto.Elem().Set(reflect.ValueOf(fm).Convert(totyp))
		return nil
	case tokind == reflect.Bool:
		fm, err := ToBool(from)
		if err != nil {
			return err
		}
		pto.Elem().Set(reflect.ValueOf(fm).Convert(totyp))
		return nil
	case tokind >= reflect.Float32 && tokind <= reflect.Float64:
		fm, err := ToFloat(from)
		if err != nil {
			return err
		}
		pto.Elem().Set(reflect.ValueOf(fm).Convert(totyp))
		return nil
	case tokind == reflect.String:
		fm := ToString(from)
		pto.Elem().Set(reflect.ValueOf(fm).Convert(totyp))
		return nil
	case tokind == reflect.Struct:
		if ftyp.Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(from)), to) // todo: this is not working -- see what marshal says, etc
			if err != nil {
				marsh, _ := json.Marshal(to)
				return fmt.Errorf("error setting struct from string: %w (example format for string: %s)", err, string(marsh))
			}
			return nil
		}
	case tokind == reflect.Slice:
		if ftyp.Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(from)), to)
			if err != nil {
				marsh, _ := json.Marshal(to)
				return fmt.Errorf("error setting slice from string: %w (example format for string: %s)", err, string(marsh))
			}
			return nil
		}
		return CopySliceRobust(to, from)
	case tokind == reflect.Map:
		if ftyp.Kind() == reflect.String {
			err := json.Unmarshal([]byte(ToString(from)), to)
			if err != nil {
				marsh, _ := json.Marshal(to)
				return fmt.Errorf("error setting map from string: %w (example format for string: %s)", err, string(marsh))
			}
			return nil
		}
		return CopyMapRobust(to, from)
	}

	tos := elide.End(fmt.Sprintf("%v", to), 40)
	fms := elide.End(fmt.Sprintf("%v", from), 40)
	return fmt.Errorf("unable to set value %s of type %T (using underlying type: %s) from value %s of type %T (using underlying type: %s): not a supported type pair and direct assigning is not possible", tos, to, totyp.String(), fms, from, LongTypeName(Underlying(reflect.ValueOf(from)).Type()))
}
