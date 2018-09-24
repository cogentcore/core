// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"
)

// This file contains helpful functions for dealing with slices, in the reflect
// system

// SliceElType returns the type of the elements for the given slice (which can be
// a pointer to a slice or a direct slice) -- just Elem() of slice type, but using
// this function makes it more explicit what is going on.
func SliceElType(sl interface{}) reflect.Type {
	return NonPtrType(reflect.TypeOf(sl)).Elem()
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func SliceNewAt(sl interface{}, idx int) {
	sltyp := SliceElType(sl)
	slptr := sltyp.Kind() == reflect.Ptr

	svl := reflect.ValueOf(sl)
	svnp := NonPtrValue(svl)

	nval := reflect.New(NonPtrType(sltyp)) // make the concrete el
	if !slptr {
		nval = nval.Elem() // use concrete value
	}
	sz := svnp.Len()
	svnp = reflect.Append(svnp, nval)
	if idx >= 0 && idx < sz {
		reflect.Copy(svnp.Slice(idx+1, sz+1), svnp.Slice(idx, sz))
		svnp.Index(idx).Set(nval)
	}
	svl.Elem().Set(svnp)
}

// SliceDeleteAt deletes element at given index from slice
func SliceDeleteAt(sl interface{}, idx int) {
	svl := reflect.ValueOf(sl)
	svnp := NonPtrValue(svl)
	svtyp := svnp.Type()
	nval := reflect.New(svtyp.Elem())
	sz := svnp.Len()
	reflect.Copy(svnp.Slice(idx, sz-1), svnp.Slice(idx+1, sz))
	svnp.Index(sz - 1).Set(nval.Elem())
	svl.Elem().Set(svnp.Slice(0, sz-1))
}

// SliceSort sorts a slice of basic values (see StructSliceSort for sorting a
// slice-of-struct using a specific field), trying kit.Floater Float(),
// kit.Inter Int(), interfaces first, and then falling back on reflect.Kind
// float, int, string conversions (first fmt.Stringer String()) and supporting
// time.Time directly as well.
func SliceSort(sl interface{}, ascending bool) error {
	sv := reflect.ValueOf(sl)
	svnp := NonPtrValue(sv)
	if svnp.Len() == 0 {
		return nil
	}
	eltyp := SliceElType(sl)
	elnptyp := NonPtrType(eltyp)
	vk := elnptyp.Kind()
	elval := OnePtrValue(svnp.Index(0))
	elif := NonPtrValue(elval).Interface()

	switch elif.(type) {
	case Floater:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPtrValue(svnp.Index(i)).Interface().(Floater).Float()
			jv := NonPtrValue(svnp.Index(j)).Interface().(Floater).Float()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	case Inter:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPtrValue(svnp.Index(i)).Interface().(Inter).Int()
			jv := NonPtrValue(svnp.Index(j)).Interface().(Inter).Int()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	}

	// try all the numeric types first!

	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPtrValue(svnp.Index(i)).Int()
			jv := NonPtrValue(svnp.Index(j)).Int()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPtrValue(svnp.Index(i)).Uint()
			jv := NonPtrValue(svnp.Index(j)).Uint()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPtrValue(svnp.Index(i)).Float()
			jv := NonPtrValue(svnp.Index(j)).Float()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	case vk == reflect.Struct && FullTypeName(elnptyp) == "time.Time":
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPtrValue(svnp.Index(i)).Interface().(time.Time)
			jv := NonPtrValue(svnp.Index(j)).Interface().(time.Time)
			if ascending {
				return iv.Before(jv)
			} else {
				return jv.Before(iv)
			}
		})
	}

	// this stringer case will likely pick up most of the rest
	switch elif.(type) {
	case fmt.Stringer:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPtrValue(svnp.Index(i)).Interface().(fmt.Stringer).String()
			jv := NonPtrValue(svnp.Index(j)).Interface().(fmt.Stringer).String()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	}

	// last resort!
	switch {
	case vk == reflect.String:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			iv := NonPtrValue(svnp.Index(i)).String()
			jv := NonPtrValue(svnp.Index(j)).String()
			if ascending {
				return strings.ToLower(iv) < strings.ToLower(jv)
			} else {
				return strings.ToLower(iv) > strings.ToLower(jv)
			}
		})
		return nil
	}

	err := fmt.Errorf("SortSlice: unable to sort elements of type: %v", eltyp.String())
	log.Println(err)
	return err
}

// StructSliceSort sorts a slice of a struct according to the given field
// indexes and sort direction, trying kit.Floater Float(), kit.Inter Int(),
// interfaces first, and then falling back on reflect.Kind float, int, string
// conversions (first fmt.Stringer String()) and supporting time.Time directly
// as well.  There is no direct method for checking the field indexes so those
// are assumed to be accurate -- will panic if not!
func StructSliceSort(struSlice interface{}, fldIdx []int, ascending bool) error {
	sv := reflect.ValueOf(struSlice)
	svnp := NonPtrValue(sv)
	if svnp.Len() == 0 {
		return nil
	}
	struTyp := SliceElType(struSlice)
	struNpTyp := NonPtrType(struTyp)
	fld := struNpTyp.FieldByIndex(fldIdx) // not easy to check.
	vk := fld.Type.Kind()
	struVal := OnePtrValue(svnp.Index(0))
	fldVal := struVal.Elem().FieldByIndex(fldIdx)
	fldIf := NonPtrValue(fldVal).Interface()

	switch fldIf.(type) {
	case Floater:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePtrValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fldIdx).Interface().(Floater).Float()
			jval := OnePtrValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fldIdx).Interface().(Floater).Float()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	case Inter:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePtrValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fldIdx).Interface().(Inter).Int()
			jval := OnePtrValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fldIdx).Interface().(Inter).Int()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	}

	// try all the numeric types first!

	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePtrValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fldIdx).Int()
			jval := OnePtrValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fldIdx).Int()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePtrValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fldIdx).Uint()
			jval := OnePtrValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fldIdx).Uint()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePtrValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fldIdx).Float()
			jval := OnePtrValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fldIdx).Float()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	case vk == reflect.Struct && FullTypeName(fld.Type) == "time.Time":
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePtrValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fldIdx).Interface().(time.Time)
			jval := OnePtrValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fldIdx).Interface().(time.Time)
			if ascending {
				return iv.Before(jv)
			} else {
				return jv.Before(iv)
			}
		})
	}

	// this stringer case will likely pick up most of the rest
	switch fldIf.(type) {
	case fmt.Stringer:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePtrValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fldIdx).Interface().(fmt.Stringer).String()
			jval := OnePtrValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fldIdx).Interface().(fmt.Stringer).String()
			if ascending {
				return iv < jv
			} else {
				return iv > jv
			}
		})
		return nil
	}

	// last resort!
	switch {
	case vk == reflect.String:
		sort.Slice(svnp.Interface(), func(i, j int) bool {
			ival := OnePtrValue(svnp.Index(i))
			iv := ival.Elem().FieldByIndex(fldIdx).String()
			jval := OnePtrValue(svnp.Index(j))
			jv := jval.Elem().FieldByIndex(fldIdx).String()
			if ascending {
				return strings.ToLower(iv) < strings.ToLower(jv)
			} else {
				return strings.ToLower(iv) > strings.ToLower(jv)
			}
		})
		return nil
	}

	err := fmt.Errorf("SortStructSlice: unable to sort on field of type: %v\n", fld.Type.String())
	log.Println(err)
	return err
}
