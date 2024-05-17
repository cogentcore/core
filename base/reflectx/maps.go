// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflectx

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"time"
)

// This file contains helpful functions for dealing with maps
// in the reflect system

// MapValueType returns the type of the value for the given map (which can be
// a pointer to a map or a direct map) -- just Elem() of map type, but using
// this function makes it more explicit what is going on.
func MapValueType(mp any) reflect.Type {
	// return NonPointerUnderlyingValue(reflect.ValueOf(sl)).Type().Elem()
	return NonPointerType(reflect.TypeOf(mp)).Elem()
}

// MapKeyType returns the type of the key for the given map (which can be a
// pointer to a map or a direct map) -- just Key() of map type, but using
// this function makes it more explicit what is going on.
func MapKeyType(mp any) reflect.Type {
	return NonPointerType(reflect.TypeOf(mp)).Key()
}

// WalkMapElements calls a function on all the "basic" elements of
// the given map; it iterates over maps within maps (but not structs
// and slices within maps).
func WalkMapElements(mp any, fun func(mp any, typ reflect.Type, key, val reflect.Value) bool) bool {
	vv := reflect.ValueOf(mp)
	if mp == nil {
		log.Printf("reflectx.MapElsValueFun: must pass a non-nil pointer to the map: %v\n", mp)
		return false
	}
	v := NonPointerValue(vv)
	if !v.IsValid() {
		return true
	}
	typ := v.Type()
	if typ.Kind() != reflect.Map {
		log.Printf("reflectx.MapElsValueFun: non-pointer type is not a map: %v\n", typ.String())
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
			rval = WalkMapElements(vali, fun)
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

// WalkMapStructElements calls a function on all the "basic" elements
// of the given map or struct; it iterates over maps within maps and
// fields within structs.
func WalkMapStructElements(mp any, fun func(mp any, typ reflect.Type, val reflect.Value) bool) bool {
	vv := reflect.ValueOf(mp)
	if mp == nil {
		log.Printf("reflectx.MapElsValueFun: must pass a non-nil pointer to the map: %v\n", mp)
		return false
	}
	v := NonPointerValue(vv)
	if !v.IsValid() {
		return true
	}
	typ := v.Type()
	vk := typ.Kind()
	rval := true
	switch vk {
	case reflect.Map:
		keys := v.MapKeys()
	mapLoop:
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
				rval = WalkMapStructElements(vali, fun)
				if !rval {
					break mapLoop
				}
			case reflect.Struct:
				rval = WalkMapStructElements(vali, fun)
				if !rval {
					break mapLoop
				}
			default:
				rval = fun(vali, typ, val)
				if !rval {
					break mapLoop
				}
			}
		}
	case reflect.Struct:
	structLoop:
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
				rval = WalkMapStructElements(vfi, fun)
				if !rval {
					break structLoop
				}
			case reflect.Struct:
				rval = WalkMapStructElements(vfi, fun)
				if !rval {
					break structLoop
				}
			default:
				rval = fun(vfi, typ, vf)
				if !rval {
					break structLoop
				}
			}
		}
	default:
		log.Printf("reflectx.MapStructElsValueFun: non-pointer type is not a map or struct: %v\n", typ.String())
		return false
	}
	return rval
}

// NumMapStructElements returns the number of elemental fields
// in the given map / struct, using [WalkMapStructElements].
func NumMapStructElements(mp any) int {
	n := 0
	falseErr := WalkMapStructElements(mp, func(mp any, typ reflect.Type, val reflect.Value) bool {
		n++
		return true
	})
	if !falseErr {
		return 0
	}
	return n
}

// MapAdd adds a new blank entry to the map.
func MapAdd(mv any) {
	mpv := reflect.ValueOf(mv)
	mpvnp := NonPointerValue(mpv)
	mvtyp := mpvnp.Type()
	valtyp := MapValueType(mv)
	if valtyp.Kind() == reflect.Interface && valtyp.String() == "interface {}" {
		str := ""
		valtyp = reflect.TypeOf(str)
	}
	nkey := reflect.New(MapKeyType(mv))
	nval := reflect.New(valtyp)
	if mpvnp.IsNil() { // make a new map
		mpv.Elem().Set(reflect.MakeMap(mvtyp))
		mpvnp = NonPointerValue(mpv)
	}
	mpvnp.SetMapIndex(nkey.Elem(), nval.Elem())
}

// MapDelete deletes the given key from the given map.
func MapDelete(mv any, key reflect.Value) {
	mpv := reflect.ValueOf(mv)
	mpvnp := NonPointerValue(mpv)
	mpvnp.SetMapIndex(key, reflect.Value{}) // delete
}

// MapDeleteAll deletes everything from the given map.
func MapDeleteAll(mv any) {
	mpv := reflect.ValueOf(mv)
	mpvnp := NonPointerValue(mpv)
	if mpvnp.Len() == 0 {
		return
	}
	itr := mpvnp.MapRange()
	for itr.Next() {
		mpvnp.SetMapIndex(itr.Key(), reflect.Value{}) // delete
	}
}

// MapSort sorts the keys of the map either by key or by value,
// and returns those keys as a slice of [reflect.Value]s.
func MapSort(mp any, byKey, ascending bool) []reflect.Value {
	mpv := reflect.ValueOf(mp)
	mpvnp := NonPointerValue(mpv)
	keys := mpvnp.MapKeys() // note: this is a slice of reflect.Value!
	if byKey {
		ValueSliceSort(keys, ascending)
	} else {
		MapValueSort(mpvnp, keys, ascending)
	}
	return keys
}

// MapValueSort sorts the keys of the given map by their values.
func MapValueSort(mpvnp reflect.Value, keys []reflect.Value, ascending bool) error {
	if len(keys) == 0 {
		return nil
	}
	keyval := keys[0]
	felval := mpvnp.MapIndex(keyval)
	eltyp := felval.Type()
	elnptyp := NonPointerType(eltyp)
	vk := elnptyp.Kind()
	elval := OnePointerValue(felval)
	elif := elval.Interface()

	// try all the numeric types first!

	switch {
	case vk >= reflect.Int && vk <= reflect.Int64:
		sort.Slice(keys, func(i, j int) bool {
			iv := NonPointerValue(mpvnp.MapIndex(keys[i])).Int()
			jv := NonPointerValue(mpvnp.MapIndex(keys[j])).Int()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		sort.Slice(keys, func(i, j int) bool {
			iv := NonPointerValue(mpvnp.MapIndex(keys[i])).Uint()
			jv := NonPointerValue(mpvnp.MapIndex(keys[j])).Uint()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		sort.Slice(keys, func(i, j int) bool {
			iv := NonPointerValue(mpvnp.MapIndex(keys[i])).Float()
			jv := NonPointerValue(mpvnp.MapIndex(keys[j])).Float()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk == reflect.Struct && ShortTypeName(elnptyp) == "time.Time":
		sort.Slice(keys, func(i, j int) bool {
			iv := NonPointerValue(mpvnp.MapIndex(keys[i])).Interface().(time.Time)
			jv := NonPointerValue(mpvnp.MapIndex(keys[j])).Interface().(time.Time)
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
			iv := NonPointerValue(mpvnp.MapIndex(keys[i])).Interface().(fmt.Stringer).String()
			jv := NonPointerValue(mpvnp.MapIndex(keys[j])).Interface().(fmt.Stringer).String()
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
			iv := NonPointerValue(mpvnp.MapIndex(keys[i])).String()
			jv := NonPointerValue(mpvnp.MapIndex(keys[j])).String()
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

// SetMapRobust robustly sets a map value using [reflect.Value]
// representations of the map, key, and value elements, ensuring that the
// proper types are used for the key and value elements using sensible
// conversions.
func SetMapRobust(mp, ky, val reflect.Value) bool {
	mtyp := mp.Type()
	if mtyp.Kind() != reflect.Map {
		log.Printf("reflectx.SetMapRobust: map arg is not map, is: %v\n", mtyp.String())
		return false
	}
	if !mp.CanSet() {
		log.Printf("reflectx.SetMapRobust: map arg is not settable: %v\n", mtyp.String())
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

// CopyMapRobust robustly copies maps.
func CopyMapRobust(to, from any) error {
	tov := reflect.ValueOf(to)
	fmv := reflect.ValueOf(from)
	tonp := NonPointerValue(tov)
	fmnp := NonPointerValue(fmv)
	totyp := tonp.Type()
	if totyp.Kind() != reflect.Map {
		err := fmt.Errorf("reflectx.CopyMapRobust: 'to' is not map, is: %v", totyp.String())
		log.Println(err)
		return err
	}
	fmtyp := fmnp.Type()
	if fmtyp.Kind() != reflect.Map {
		err := fmt.Errorf("reflectx.CopyMapRobust: 'from' is not map, is: %v", fmtyp.String())
		log.Println(err)
		return err
	}
	if tonp.IsNil() {
		OnePointerValue(tov).Elem().Set(reflect.MakeMap(totyp))
	} else {
		MapDeleteAll(to)
	}
	if fmnp.Len() == 0 {
		return nil
	}
	eltyp := SliceElementType(to)
	itr := fmnp.MapRange()
	for itr.Next() {
		tonp.SetMapIndex(itr.Key(), CloneToType(eltyp, itr.Value().Interface()).Elem())
	}
	return nil
}
