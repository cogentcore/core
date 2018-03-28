// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

// github.com/rcoreilly/goki/ki/kit

import (
	// "fmt"
	"math"
	"reflect"
	"strconv"
)

// Convenience functions for converting interface{} (e.g. properties) to given
// types uses the "ok" bool mechanism to report failure -- are as robust and
// general as possible.
//
// WARNING: these violate many of the type-safety features of Go but OTOH give
// maximum robustness, appropriate for the world of end-user settable
// properties, and deal with most common-sense cases, e.g., string <-> number,
// etc.  nil values return !ok

// robustly convert anything to a bool
func ToBool(it interface{}) (bool, bool) {
	if it == nil {
		return false, false
	}
	v := reflect.ValueOf(it)
	vk := v.Kind()
	if vk == reflect.Ptr {
		v = v.Elem()
		vk = v.Kind()
	}
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

// robustly convert anything to an int64
func ToInt(it interface{}) (int64, bool) {
	if it == nil {
		return 0, false
	}
	v := reflect.ValueOf(it)
	vk := v.Kind()
	if vk == reflect.Ptr {
		v = v.Elem()
		vk = v.Kind()
	}
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

// robustly convert anything to a Float64
func ToFloat(it interface{}) (float64, bool) {
	if it == nil {
		return 0.0, false
	}
	v := reflect.ValueOf(it)
	vk := v.Kind()
	if vk == reflect.Ptr {
		v = v.Elem()
		vk = v.Kind()
	}
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

// robustly convert anything to a String
func ToString(it interface{}) (string, bool) {
	if it == nil {
		return "", false
	}
	v := reflect.ValueOf(it)
	vk := v.Kind()
	if vk == reflect.Ptr {
		v = v.Elem()
		vk = v.Kind()
	}
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), true
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), true
	case vk == reflect.Bool:
		return strconv.FormatBool(v.Bool()), true
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'G', -1, 64), true
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		cv := v.Complex()
		rv := strconv.FormatFloat(real(cv), 'G', -1, 64) + "," + strconv.FormatFloat(imag(cv), 'G', -1, 64)
		return rv, true
	case vk == reflect.String: // todo: what about []byte?
		return v.String(), true
	default:
		return "", false
	}
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
