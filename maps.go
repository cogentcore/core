// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"
)

// This file contains helpful functions for dealing with maps, in the reflect
// system

// MakeMap makes a map that is actually addressable, getting around the hidden
// interface{} that reflect.MakeMap makes, by calling UnhideIfaceValue (from ptrs.go)
func MakeMap(typ reflect.Type) reflect.Value {
	return UnhideAnyValue(reflect.MakeMap(typ))
}

// MapValueType returns the type of the value for the given map (which can be
// a pointer to a map or a direct map) -- just Elem() of map type, but using
// this function makes it more explicit what is going on.
func MapValueType(mp any) reflect.Type {
	return NonPtrType(reflect.TypeOf(mp)).Elem()
}

// MapKeyType returns the type of the key for the given map (which can be a
// pointer to a map or a direct map) -- just Key() of map type, but using
// this function makes it more explicit what is going on.
func MapKeyType(mp any) reflect.Type {
	return NonPtrType(reflect.TypeOf(mp)).Key()
}

// MapElsValueFun calls a function on all the "basic" elements of given map --
// iterates over maps within maps (but not structs, slices within maps).
func MapElsValueFun(mp any, fun func(mp any, typ reflect.Type, key, val reflect.Value) bool) bool {
	vv := reflect.ValueOf(mp)
	if mp == nil {
		log.Printf("laser.MapElsValueFun: must pass a non-nil pointer to the map: %v\n", mp)
		return false
	}
	v := NonPtrValue(vv)
	if !v.IsValid() {
		return true
	}
	typ := v.Type()
	if typ.Kind() != reflect.Map {
		log.Printf("laser.MapElsValueFun: non-pointer type is not a map: %v\n", typ.String())
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
func MapElsN(mp any) int {
	n := 0
	falseErr := MapElsValueFun(mp, func(mp any, typ reflect.Type, key, val reflect.Value) bool {
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
func MapStructElsValueFun(mp any, fun func(mp any, typ reflect.Type, val reflect.Value) bool) bool {
	vv := reflect.ValueOf(mp)
	if mp == nil {
		log.Printf("laser.MapElsValueFun: must pass a non-nil pointer to the map: %v\n", mp)
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
			if AnyIsNil(vali) {
				continue
			}
			vt := reflect.TypeOf(vali)
			if vt == nil {
				continue
			}
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
		log.Printf("laser.MapStructElsValueFun: non-pointer type is not a map or struct: %v\n", typ.String())
		return false
	}
	return rval
}

// MapStructElsN returns number of elemental fields in given map / struct types
func MapStructElsN(mp any) int {
	n := 0
	falseErr := MapStructElsValueFun(mp, func(mp any, typ reflect.Type, val reflect.Value) bool {
		n++
		return true
	})
	if falseErr == false {
		return 0
	}
	return n
}

// MapAdd adds a new blank entry to the map
func MapAdd(mv any) {
	mpv := reflect.ValueOf(mv)
	mpvnp := NonPtrValue(mpv)
	mvtyp := mpvnp.Type()
	valtyp := MapValueType(mv)
	if valtyp.Kind() == reflect.Interface && valtyp.String() == "interface {}" {
		str := ""
		valtyp = reflect.TypeOf(str)
	}
	nkey := reflect.New(MapKeyType(mv))
	nval := reflect.New(valtyp)
	if mpvnp.IsNil() { // make a new map
		nmp := MakeMap(mvtyp)
		mpv.Elem().Set(nmp.Elem())
		mpvnp = NonPtrValue(mpv)
	}
	mpvnp.SetMapIndex(nkey.Elem(), nval.Elem())
}

// MapDelete deletes a key-value from the map (set key to a zero value)
func MapDelete(mv any, key any) {
	mpv := reflect.ValueOf(mv)
	mpvnp := NonPtrValue(mpv)
	mpvnp.SetMapIndex(reflect.ValueOf(key), reflect.Value{}) // delete
}

// MapDeleteValue deletes a key-value from the map (set key to a zero value)
// -- key is already a reflect.Value
func MapDeleteValue(mv any, key reflect.Value) {
	mpv := reflect.ValueOf(mv)
	mpvnp := NonPtrValue(mpv)
	mpvnp.SetMapIndex(key, reflect.Value{}) // delete
}

// MapDeleteAll deletes everything from map
func MapDeleteAll(mv any) {
	mpv := reflect.ValueOf(mv)
	mpvnp := NonPtrValue(mpv)
	if mpvnp.Len() == 0 {
		return
	}
	itr := mpvnp.MapRange()
	for itr.Next() {
		mpvnp.SetMapIndex(itr.Key(), reflect.Value{}) // delete
	}
}

// MapSort sorts keys of map either by key or by value, returns those keys as
// a slice of reflect.Value, as returned by reflect.Value.MapKeys() method
func MapSort(mp any, byKey, ascending bool) []reflect.Value {
	mpv := reflect.ValueOf(mp)
	mpvnp := NonPtrValue(mpv)
	keys := mpvnp.MapKeys() // note: this is a slice of reflect.Value!
	if byKey {
		ValueSliceSort(keys, ascending)
	} else {
		MapValueSort(mpvnp, keys, ascending)
	}
	return keys
}

// MapValueSort sorts keys of map by values
func MapValueSort(mpvnp reflect.Value, keys []reflect.Value, ascending bool) error {
	if len(keys) == 0 {
		return nil
	}
	keyval := keys[0]
	felval := mpvnp.MapIndex(keyval)
	eltyp := felval.Type()
	elnptyp := NonPtrType(eltyp)
	vk := elnptyp.Kind()
	elval := OnePtrValue(felval)
	elif := elval.Interface()

	// try all the numeric types first!

	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		sort.Slice(keys, func(i, j int) bool {
			iv := NonPtrValue(mpvnp.MapIndex(keys[i])).Int()
			jv := NonPtrValue(mpvnp.MapIndex(keys[j])).Int()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		sort.Slice(keys, func(i, j int) bool {
			iv := NonPtrValue(mpvnp.MapIndex(keys[i])).Uint()
			jv := NonPtrValue(mpvnp.MapIndex(keys[j])).Uint()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		sort.Slice(keys, func(i, j int) bool {
			iv := NonPtrValue(mpvnp.MapIndex(keys[i])).Float()
			jv := NonPtrValue(mpvnp.MapIndex(keys[j])).Float()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk == reflect.Struct && ShortTypeName(elnptyp) == "time.Time":
		sort.Slice(keys, func(i, j int) bool {
			iv := NonPtrValue(mpvnp.MapIndex(keys[i])).Interface().(time.Time)
			jv := NonPtrValue(mpvnp.MapIndex(keys[j])).Interface().(time.Time)
			if ascending {
				return iv.Before(jv)
			}
			return jv.Before(iv)
		})
	}

	// this stringer case will likely pick up most of the rest
	switch elif.(type) {
	case fmt.Stringer:
		sort.Slice(keys, func(i, j int) bool {
			iv := NonPtrValue(mpvnp.MapIndex(keys[i])).Interface().(fmt.Stringer).String()
			jv := NonPtrValue(mpvnp.MapIndex(keys[j])).Interface().(fmt.Stringer).String()
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
		sort.Slice(keys, func(i, j int) bool {
			iv := NonPtrValue(mpvnp.MapIndex(keys[i])).String()
			jv := NonPtrValue(mpvnp.MapIndex(keys[j])).String()
			if ascending {
				return strings.ToLower(iv) < strings.ToLower(jv)
			}
			return strings.ToLower(iv) > strings.ToLower(jv)
		})
		return nil
	}

	err := fmt.Errorf("MapValueSort: unable to sort elements of type: %v", eltyp.String())
	log.Println(err)
	return err
}

// SetMapRobust robustly sets a map value using reflect.Value representations
// of the map, key, and value elements, ensuring that the proper types are
// used for the key and value elements using sensible conversions.
// map value must be a valid map value -- that is not checked.
func SetMapRobust(mp, ky, val reflect.Value) bool {
	mtyp := mp.Type()
	if mtyp.Kind() != reflect.Map {
		log.Printf("ki.SetMapRobust: map arg is not map, is: %v\n", mtyp.String())
		return false
	}
	if !mp.CanSet() {
		log.Printf("ki.SetMapRobust: map arg is not settable: %v\n", mtyp.String())
		return false
	}
	ktyp := mtyp.Key()
	etyp := mtyp.Elem()
	if etyp.Kind() == val.Kind() && ky.Kind() == ktyp.Kind() {
		mp.SetMapIndex(ky, val)
		return true
	}
	if ky.Kind() == ktyp.Kind() {
		mp.SetMapIndex(ky, val.Convert(etyp))
		return true
	}
	if etyp.Kind() == val.Kind() {
		mp.SetMapIndex(ky.Convert(ktyp), val)
		return true
	}
	mp.SetMapIndex(ky.Convert(ktyp), val.Convert(etyp))
	return true
}

// CopyMapRobust robustly copies maps using SetRobust method for the elements.
func CopyMapRobust(to, fm any) error {
	tov := reflect.ValueOf(to)
	fmv := reflect.ValueOf(fm)
	tonp := NonPtrValue(tov)
	fmnp := NonPtrValue(fmv)
	totyp := tonp.Type()
	if totyp.Kind() != reflect.Map {
		err := fmt.Errorf("ki.CopyMapRobust: 'to' is not map, is: %v", totyp.String())
		log.Println(err)
		return err
	}
	fmtyp := fmnp.Type()
	if fmtyp.Kind() != reflect.Map {
		err := fmt.Errorf("ki.CopyMapRobust: 'from' is not map, is: %v", fmtyp.String())
		log.Println(err)
		return err
	}
	if tonp.IsNil() {
		OnePtrValue(tov).Elem().Set(MakeMap(totyp).Elem())
	} else {
		MapDeleteAll(to)
	}
	if fmnp.Len() == 0 {
		return nil
	}
	eltyp := SliceElType(to)
	itr := fmnp.MapRange()
	for itr.Next() {
		tonp.SetMapIndex(itr.Key(), CloneToType(eltyp, itr.Value().Interface()).Elem())
	}
	return nil
}
