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
// (including pointers to such)
// using a big type switch organized for greatest efficiency.
// tries the glop/bools/.Booler interface if not a bool type.
//
//gopy:interface=handle
func ToBool(v any) (bool, bool) {
	switch vt := v.(type) {
	case bool:
		return vt, true
	case *bool:
		return *vt, true
	}

	if br, ok := v.(bools.Booler); ok {
		return br.Bool(), true
	}

	switch vt := v.(type) {
	case int:
		return vt != 0, true
	case *int:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case int32:
		return vt != 0, true
	case *int32:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case int64:
		return vt != 0, true
	case *int64:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case uint8:
		return vt != 0, true
	case *uint8:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case float64:
		return vt != 0, true
	case *float64:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case float32:
		return vt != 0, true
	case *float32:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case string:
		r, err := strconv.ParseBool(vt)
		if err != nil {
			return false, false
		}
		return r, true
	case *string:
		if vt == nil {
			return false, false
		}
		r, err := strconv.ParseBool(*vt)
		if err != nil {
			return false, false
		}
		return r, true
	case int8:
		return vt != 0, true
	case *int8:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case int16:
		return vt != 0, true
	case *int16:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case uint16:
		return vt != 0, true
	case *uint16:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case uint32:
		return vt != 0, true
	case *uint32:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case uint64:
		return vt != 0, true
	case *uint64:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	case uintptr:
		return vt != 0, true
	case *uintptr:
		if vt == nil {
			return false, false
		}
		return *vt != 0, true
	}

	return false, false
}

// ToInt robustly converts to an int64 any basic elemental type
// (including pointers to such)
// using a big type switch organized for greatest efficiency.
//
//gopy:interface=handle
func ToInt(v any) (int64, bool) {
	switch vt := v.(type) {
	case int:
		return int64(vt), true
	case *int:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	case int32:
		return int64(vt), true
	case *int32:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	case int64:
		return vt, true
	case *int64:
		if vt == nil {
			return 0, false
		}
		return *vt, true
	case uint8:
		return int64(vt), true
	case *uint8:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	case float64:
		return int64(vt), true
	case *float64:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	case float32:
		return int64(vt), true
	case *float32:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	case bool:
		if vt {
			return 1, true
		}
		return 0, true
	case *bool:
		if vt == nil {
			return 0, false
		}
		if *vt {
			return 1, true
		}
		return 0, true
	case string:
		r, err := strconv.ParseInt(vt, 0, 64)
		if err != nil {
			return 0, false
		}
		return r, true
	case *string:
		if vt == nil {
			return 0, false
		}
		r, err := strconv.ParseInt(*vt, 0, 64)
		if err != nil {
			return 0, false
		}
		return r, true
	case int8:
		return int64(vt), true
	case *int8:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	case int16:
		return int64(vt), true
	case *int16:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	case uint16:
		return int64(vt), true
	case *uint16:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	case uint32:
		return int64(vt), true
	case *uint32:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	case uint64:
		return int64(vt), true
	case *uint64:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	case uintptr:
		return int64(vt), true
	case *uintptr:
		if vt == nil {
			return 0, false
		}
		return int64(*vt), true
	}
	return 0, false
}

// ToFloat robustly converts to a float64 any basic elemental type
// (including pointers to such)
// using a big type switch organized for greatest efficiency.
//
//gopy:interface=handle
func ToFloat(v any) (float64, bool) {
	switch vt := v.(type) {
	case float64:
		return vt, true
	case *float64:
		if vt == nil {
			return 0, false
		}
		return *vt, true
	case float32:
		return float64(vt), true
	case *float32:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	case int:
		return float64(vt), true
	case *int:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	case int32:
		return float64(vt), true
	case *int32:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	case int64:
		return float64(vt), true
	case *int64:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	case uint8:
		return float64(vt), true
	case *uint8:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	case bool:
		if vt {
			return 1, true
		}
		return 0, true
	case *bool:
		if vt == nil {
			return 0, false
		}
		if *vt {
			return 1, true
		}
		return 0, true
	case string:
		r, err := strconv.ParseFloat(vt, 64)
		if err != nil {
			return 0.0, false
		}
		return r, true
	case *string:
		if vt == nil {
			return 0, false
		}
		r, err := strconv.ParseFloat(*vt, 64)
		if err != nil {
			return 0.0, false
		}
		return r, true
	case int8:
		return float64(vt), true
	case *int8:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	case int16:
		return float64(vt), true
	case *int16:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	case uint16:
		return float64(vt), true
	case *uint16:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	case uint32:
		return float64(vt), true
	case *uint32:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	case uint64:
		return float64(vt), true
	case *uint64:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	case uintptr:
		return float64(vt), true
	case *uintptr:
		if vt == nil {
			return 0, false
		}
		return float64(*vt), true
	}
	return 0, false
}

// ToFloat32 robustly converts to a float32 any basic elemental type
// (including pointers to such)
// using a big type switch organized for greatest efficiency.
//
//gopy:interface=handle
func ToFloat32(v any) (float32, bool) {
	switch vt := v.(type) {
	case float32:
		return vt, true
	case *float32:
		if vt == nil {
			return 0, false
		}
		return *vt, true
	case float64:
		return float32(vt), true
	case *float64:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	case int:
		return float32(vt), true
	case *int:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	case int32:
		return float32(vt), true
	case *int32:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	case int64:
		return float32(vt), true
	case *int64:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	case uint8:
		return float32(vt), true
	case *uint8:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	case bool:
		if vt {
			return 1, true
		}
		return 0, true
	case *bool:
		if vt == nil {
			return 0, false
		}
		if *vt {
			return 1, true
		}
		return 0, true
	case string:
		r, err := strconv.ParseFloat(vt, 32)
		if err != nil {
			return 0.0, false
		}
		return float32(r), true
	case *string:
		if vt == nil {
			return 0, false
		}
		r, err := strconv.ParseFloat(*vt, 32)
		if err != nil {
			return 0.0, false
		}
		return float32(r), true
	case int8:
		return float32(vt), true
	case *int8:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	case int16:
		return float32(vt), true
	case *int16:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	case uint16:
		return float32(vt), true
	case *uint16:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	case uint32:
		return float32(vt), true
	case *uint32:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	case uint64:
		return float32(vt), true
	case *uint64:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	case uintptr:
		return float32(vt), true
	case *uintptr:
		if vt == nil {
			return 0, false
		}
		return float32(*vt), true
	}
	return 0, false
}

// ToString robustly converts anything to a String
// using a big type switch organized for greatest efficiency.
// First checks for string or []byte and returns that immediately,
// then checks for the Stringer interface as the preferred conversion
// (e.g., for enums), and then falls back on strconv calls for numeric types.
// If everything else fails, it uses Sprintf("%v") which always works,
// so there is no need for a bool = false return.
// * returns "nil" for any nil pointers
// * byte is converted as string(byte) not the decimal representation
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

	if AnyIsNil(v) {
		return nilstr
	}
	return fmt.Sprintf("%v", v)
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
		log.Printf("laser.SetRobust 'to' cannot be set -- must be a variable or field, not a const or tmp or other value that cannot be set.  Value info: %v\n", vp)
		return false
	}

	if es, ok := to.(enums.EnumSetter); ok {
		if str, ok := frm.(string); ok {
			es.SetString(str)
			return true
		}
		fm, ok := ToInt(frm)
		if ok {
			es.SetInt64(int64(fm))
			return true
		}
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
	case vk == reflect.String:
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
