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

	"cogentcore.org/core/base/errors"
)

// This file contains helpful functions for dealing with maps
// in the reflect system

// MapValueType returns the type of the value for the given map (which can be
// a pointer to a map or a direct map); just Elem() of map type, but using
// this function makes it more explicit what is going on.
func MapValueType(mp any) reflect.Type {
	return NonPointerType(reflect.TypeOf(mp)).Elem()
}

// MapKeyType returns the type of the key for the given map (which can be a
// pointer to a map or a direct map); just Key() of map type, but using
// this function makes it more explicit what is going on.
func MapKeyType(mp any) reflect.Type {
	return NonPointerType(reflect.TypeOf(mp)).Key()
}

// MapAdd adds a new blank entry to the map.
func MapAdd(mv any) {
	mpv := reflect.ValueOf(mv)
	mpvnp := Underlying(mpv)
	mvtyp := mpvnp.Type()
	valtyp := MapValueType(mv)
	if valtyp.Kind() == reflect.Interface && valtyp.String() == "interface {}" {
		valtyp = reflect.TypeOf("")
	}
	nkey := reflect.New(MapKeyType(mv))
	nval := reflect.New(valtyp)
	if mpvnp.IsNil() { // make a new map
		mpv.Elem().Set(reflect.MakeMap(mvtyp))
		mpvnp = Underlying(mpv)
	}
	mpvnp.SetMapIndex(nkey.Elem(), nval.Elem())
}

// MapDelete deletes the given key from the given map.
func MapDelete(mv any, key reflect.Value) {
	mpv := reflect.ValueOf(mv)
	mpvnp := Underlying(mpv)
	mpvnp.SetMapIndex(key, reflect.Value{}) // delete
}

// MapDeleteAll deletes everything from the given map.
func MapDeleteAll(mv any) {
	mpv := reflect.ValueOf(mv)
	mpvnp := Underlying(mpv)
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
	mpvnp := Underlying(mpv)
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
			iv := Underlying(mpvnp.MapIndex(keys[i])).Int()
			jv := Underlying(mpvnp.MapIndex(keys[j])).Int()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Uint && vk <= reflect.Uint64:
		sort.Slice(keys, func(i, j int) bool {
			iv := Underlying(mpvnp.MapIndex(keys[i])).Uint()
			jv := Underlying(mpvnp.MapIndex(keys[j])).Uint()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		sort.Slice(keys, func(i, j int) bool {
			iv := Underlying(mpvnp.MapIndex(keys[i])).Float()
			jv := Underlying(mpvnp.MapIndex(keys[j])).Float()
			if ascending {
				return iv < jv
			}
			return iv > jv
		})
		return nil
	case vk == reflect.Struct && ShortTypeName(elnptyp) == "time.Time":
		sort.Slice(keys, func(i, j int) bool {
			iv := Underlying(mpvnp.MapIndex(keys[i])).Interface().(time.Time)
			jv := Underlying(mpvnp.MapIndex(keys[j])).Interface().(time.Time)
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
			iv := Underlying(mpvnp.MapIndex(keys[i])).Interface().(fmt.Stringer).String()
			jv := Underlying(mpvnp.MapIndex(keys[j])).Interface().(fmt.Stringer).String()
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
			iv := Underlying(mpvnp.MapIndex(keys[i])).String()
			jv := Underlying(mpvnp.MapIndex(keys[j])).String()
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
	tonp := Underlying(tov)
	fmnp := Underlying(fmv)
	totyp := tonp.Type()
	if totyp.Kind() != reflect.Map {
		err := fmt.Errorf("reflectx.CopyMapRobust: 'to' is not map, is: %v", totyp)
		return errors.Log(err)
	}
	fmtyp := fmnp.Type()
	if fmtyp.Kind() != reflect.Map {
		err := fmt.Errorf("reflectx.CopyMapRobust: 'from' is not map, is: %v", fmtyp)
		return errors.Log(err)
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
