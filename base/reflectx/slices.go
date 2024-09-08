// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
)

// This file contains helpful functions for dealing with slices
// in the reflect system

// SliceElementType returns the type of the elements of the given slice (which can be
// a pointer to a slice or a direct slice); just [reflect.Type.Elem] of slice type, but
// using this function bypasses any pointer issues and makes it more explicit what is going on.
func SliceElementType(sl any) reflect.Type {
	return Underlying(reflect.ValueOf(sl)).Type().Elem()
}

// SliceElementValue returns a new [reflect.Value] of the [SliceElementType].
func SliceElementValue(sl any) reflect.Value {
	return NonNilNew(SliceElementType(sl)).Elem()
}

// SliceNewAt inserts a new blank element at the given index in the given slice.
// -1 means the end.
func SliceNewAt(sl any, idx int) {
	up := UnderlyingPointer(reflect.ValueOf(sl))
	np := up.Elem()
	val := SliceElementValue(sl)
	sz := np.Len()
	np = reflect.Append(np, val)
	if idx >= 0 && idx < sz {
		reflect.Copy(np.Slice(idx+1, sz+1), np.Slice(idx, sz))
		np.Index(idx).Set(val)
	}
	up.Elem().Set(np)
}

// SliceDeleteAt deletes the element at the given index in the given slice.
func SliceDeleteAt(sl any, idx int) {
	svl := OnePointerValue(reflect.ValueOf(sl))
	svnp := NonPointerValue(svl)
	svtyp := svnp.Type()
	nval := reflect.New(svtyp.Elem())
	sz := svnp.Len()
	reflect.Copy(svnp.Slice(idx, sz-1), svnp.Slice(idx+1, sz))
	svnp.Index(sz - 1).Set(nval.Elem())
	svl.Elem().Set(svnp.Slice(0, sz-1))
}

// SliceSort sorts a slice of basic values (see [StructSliceSort] for sorting a
// slice-of-struct using a specific field), using float, int, string, and [time.Time]
// conversions.
func SliceSort(sl any, ascending bool) error {
	sv := reflect.ValueOf(sl)
	svnp := NonPointerValue(sv)
	if svnp.Len() == 0 {
		return nil
	}
	eltyp := SliceElementType(sl)
	elnptyp := NonPointerType(eltyp)
	vk := elnptyp.Kind()
	elval := OnePointerValue(svnp.Index(0))
	elif := elval.Interface()

	// try all the numeric types first!
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPointerValue(svnp.Index(i)).Int()
			jv := NonPointerValue(svnp.Index(j)).Int()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPointerValue(svnp.Index(i)).Uint()
			jv := NonPointerValue(svnp.Index(j)).Uint()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPointerValue(svnp.Index(i)).Float()
			jv := NonPointerValue(svnp.Index(j)).Float()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk == reflect.Struct && ShortTypeName(elnptyp) == "time.Time":
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPointerValue(svnp.Index(i)).Interface().(time.Time)
			jv := NonPointerValue(svnp.Index(j)).Interface().(time.Time)
			if ascending {
				return iv.Before(jv)
			}
			return jv.Before(iv)
		})
	}

	// this stringer case will likely pick up most of the rest
	switch elif.(type) {
	case fmt.Stringer:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPointerValue(svnp.Index(i)).Interface().(fmt.Stringer).String()
			jv := NonPointerValue(svnp.Index(j)).Interface().(fmt.Stringer).String()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	}

	// last resort!
	switch {
	case vk == reflect.String:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPointerValue(svnp.Index(i)).String()
			jv := NonPointerValue(svnp.Index(j)).String()
			if ascending {
				return strings.ToLower(iv) < strings.ToLower(jv)
			}
			return strings.ToLower(iv) > strings.ToLower(jv)
		})
		return nil
	}

	err := fmt.Errorf("SortSlice: unable to sort elements of type: %v", eltyp.String())
	return errors.Log(err)
}

// StructSliceSort sorts the given slice of structs according to the given field
// indexes and sort direction, using float, int, string, and [time.Time] conversions.
// It will panic if the field indexes are invalid.
func StructSliceSort(structSlice any, fieldIndex []int, ascending bool) error {
	sv := reflect.ValueOf(structSlice)
	svnp := NonPointerValue(sv)
	if svnp.Len() == 0 {
		return nil
	}
	structTyp := SliceElementType(structSlice)
	structNptyp := NonPointerType(structTyp)
	fld := structNptyp.FieldByIndex(fieldIndex) // not easy to check.
	vk := fld.Type.Kind()
	structVal := OnePointerValue(svnp.Index(0))
	fieldVal := structVal.Elem().FieldByIndex(fieldIndex)
	fieldIf := fieldVal.Interface()

	// try all the numeric types first!
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePointerValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fieldIndex).Int()
			jval := OnePointerValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fieldIndex).Int()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePointerValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fieldIndex).Uint()
			jval := OnePointerValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fieldIndex).Uint()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePointerValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fieldIndex).Float()
			jval := OnePointerValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fieldIndex).Float()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk == reflect.Struct && ShortTypeName(fld.Type) == "time.Time":
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePointerValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fieldIndex).Interface().(time.Time)
			jval := OnePointerValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fieldIndex).Interface().(time.Time)
			if ascending {
				return iv.Before(jv)
			}
			return jv.Before(iv)
		})
	}

	// this stringer case will likely pick up most of the rest
	switch fieldIf.(type) {
	case fmt.Stringer:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePointerValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fieldIndex).Interface().(fmt.Stringer).String()
			jval := OnePointerValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fieldIndex).Interface().(fmt.Stringer).String()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	}

	// last resort!
	switch {
	case vk == reflect.String:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePointerValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fieldIndex).String()
			jval := OnePointerValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fieldIndex).String()
			if ascending {
				return strings.ToLower(iv) < strings.ToLower(jv)
			}
			return strings.ToLower(iv) > strings.ToLower(jv)
		})
		return nil
	}

	err := fmt.Errorf("SortStructSlice: unable to sort on field of type: %v", fld.Type.String())
	return errors.Log(err)
}

// ValueSliceSort sorts a slice of [reflect.Value]s using basic types where possible.
func ValueSliceSort(sl []reflect.Value, ascending bool) error {
	if len(sl) == 0 {
		return nil
	}
	felval := sl[0] // reflect.Value
	eltyp := felval.Type()
	elnptyp := NonPointerType(eltyp)
	vk := elnptyp.Kind()
	elval := OnePointerValue(felval)
	elif := elval.Interface()

	// try all the numeric types first!
	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		sort.Slice(sl, func(i, j int) bool {
			iv := NonPointerValue(sl[i]).Int()
			jv := NonPointerValue(sl[j]).Int()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		sort.Slice(sl, func(i, j int) bool {
			iv := NonPointerValue(sl[i]).Uint()
			jv := NonPointerValue(sl[j]).Uint()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		sort.Slice(sl, func(i, j int) bool {
			iv := NonPointerValue(sl[i]).Float()
			jv := NonPointerValue(sl[j]).Float()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk == reflect.Struct && ShortTypeName(elnptyp) == "time.Time":
		sort.Slice(sl, func(i, j int) bool {
			iv := NonPointerValue(sl[i]).Interface().(time.Time)
			jv := NonPointerValue(sl[j]).Interface().(time.Time)
			if ascending {
				return iv.Before(jv)
			}
			return jv.Before(iv)
		})
	}

	// this stringer case will likely pick up most of the rest
	switch elif.(type) {
	case fmt.Stringer:
		sort.Slice(sl, func(i, j int) bool {
			iv := NonPointerValue(sl[i]).Interface().(fmt.Stringer).String()
			jv := NonPointerValue(sl[j]).Interface().(fmt.Stringer).String()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	}

	// last resort!
	switch {
	case vk == reflect.String:
		sort.Slice(sl, func(i, j int) bool {
			iv := NonPointerValue(sl[i]).String()
			jv := NonPointerValue(sl[j]).String()
			if ascending {
				return strings.ToLower(iv) < strings.ToLower(jv)
			}
			return strings.ToLower(iv) > strings.ToLower(jv)
		})
		return nil
	}

	err := fmt.Errorf("ValueSliceSort: unable to sort elements of type: %v", eltyp.String())
	return errors.Log(err)
}

// CopySliceRobust robustly copies slices.
func CopySliceRobust(to, from any) error {
	tov := reflect.ValueOf(to)
	fmv := reflect.ValueOf(from)
	tonp := Underlying(tov)
	fmnp := Underlying(fmv)
	totyp := tonp.Type()
	if totyp.Kind() != reflect.Slice {
		err := fmt.Errorf("reflectx.CopySliceRobust: 'to' is not slice, is: %v", totyp.String())
		return errors.Log(err)
	}
	fmtyp := fmnp.Type()
	if fmtyp.Kind() != reflect.Slice {
		err := fmt.Errorf("reflectx.CopySliceRobust: 'from' is not slice, is: %v", fmtyp.String())
		return errors.Log(err)
	}
	fmlen := fmnp.Len()
	if tonp.IsNil() {
		tonp.Set(reflect.MakeSlice(totyp, fmlen, fmlen))
	} else {
		if tonp.Len() > fmlen {
			tonp.SetLen(fmlen)
		}
	}
	for i := 0; i < fmlen; i++ {
		tolen := tonp.Len()
		if i >= tolen {
			SliceNewAt(to, i)
		}
		SetRobust(PointerValue(tonp.Index(i)).Interface(), fmnp.Index(i).Interface())
	}
	return nil
}
