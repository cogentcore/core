// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

import (
	"log"
	"reflect"
)

// This file contains helpful functions for dealing with maps, in the reflect
// system

// MapElsValueFun calls a function on all the "basic" elements of given map --
// iterates over maps within maps (but not structs, slices within maps).
func MapElsValueFun(mp interface{}, fun func(mp interface{}, typ reflect.Type, key, val reflect.Value) bool) bool {
	vv := reflect.ValueOf(mp)
	if mp == nil {
		log.Printf("kit.MapElsValueFun: must pass a non-nil pointer to the map: %v\n", mp)
		return false
	}
	v := NonPtrValue(vv)
	if !v.IsValid() {
		return true
	}
	typ := v.Type()
	if typ.Kind() != reflect.Map {
		log.Printf("kit.MapElsValueFun: non-pointer type is not a map: %v\n", typ.String())
		return false
	}
	rval := true
	keys := v.MapKeys()
	for _, key := range keys {
		val := v.MapIndex(key)
		vali := val.Interface()
		// vt := val.Type()
		vt := reflect.TypeOf(vali)
		// fmt.Printf("key %v val %v kind: %v\n", key, val, vt.Kind())
		if vt.Kind() == reflect.Map {
			rval = MapElsValueFun(vali, fun)
			if !rval {
				break
			}
			// } else if vt.Kind() == reflect.Struct { // todo
			// 	rval = MapElsValueFun(vali, fun)
			// 	if !rval {
			// 		break
			// 	}
		} else {
			rval = fun(vali, typ, key, val)
			if !rval {
				break
			}
		}
	}
	return rval
}

// MapElsN returns number of elemental fields in given map type
func MapElsN(mp interface{}) int {
	n := 0
	falseErr := MapElsValueFun(mp, func(mp interface{}, typ reflect.Type, key, val reflect.Value) bool {
		n++
		return true
	})
	if falseErr == false {
		return 0
	}
	return n
}

// MapStructElsValueFun calls a function on all the "basic" elements of given
// map or struct -- iterates over maps within maps and fields within structs
func MapStructElsValueFun(mp interface{}, fun func(mp interface{}, typ reflect.Type, val reflect.Value) bool) bool {
	vv := reflect.ValueOf(mp)
	if mp == nil {
		log.Printf("kit.MapElsValueFun: must pass a non-nil pointer to the map: %v\n", mp)
		return false
	}
	v := NonPtrValue(vv)
	if !v.IsValid() {
		return true
	}
	typ := v.Type()
	vk := typ.Kind()
	rval := true
	switch vk {
	case reflect.Map:
		keys := v.MapKeys()
		for _, key := range keys {
			val := v.MapIndex(key)
			vali := val.Interface()
			vt := reflect.TypeOf(vali)
			vtk := vt.Kind()
			switch vtk {
			case reflect.Map:
				rval = MapStructElsValueFun(vali, fun)
				if !rval {
					break
				}
			case reflect.Struct:
				rval = MapStructElsValueFun(vali, fun)
				if !rval {
					break
				}
			default:
				rval = fun(vali, typ, val)
				if !rval {
					break
				}
			}
		}
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			vf := v.Field(i)
			if !vf.CanInterface() {
				continue
			}
			vfi := vf.Interface()
			if vfi == mp {
				continue
			}
			vtk := f.Type.Kind()
			switch vtk {
			case reflect.Map:
				rval = MapStructElsValueFun(vfi, fun)
				if !rval {
					break
				}
			case reflect.Struct:
				rval = MapStructElsValueFun(vfi, fun)
				if !rval {
					break
				}
			default:
				rval = fun(vfi, typ, vf)
				if !rval {
					break
				}
			}
		}
	default:
		log.Printf("kit.MapStructElsValueFun: non-pointer type is not a map or struct: %v\n", typ.String())
		return false
	}
	return rval
}

// MapStructElsN returns number of elemental fields in given map / struct types
func MapStructElsN(mp interface{}) int {
	n := 0
	falseErr := MapStructElsValueFun(mp, func(mp interface{}, typ reflect.Type, val reflect.Value) bool {
		n++
		return true
	})
	if falseErr == false {
		return 0
	}
	return n
}
