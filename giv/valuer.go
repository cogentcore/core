// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"image/color"
	"reflect"
	"time"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
)

// This file handles converting values to [Value]s.

func init() {
	gi.SettingsWindow = SettingsWindow
	gi.InspectorWindow = InspectorWindow

	AddValue(icons.Icon(""), func() Value { return &IconValue{} })
	AddValue(gi.FontName(""), func() Value { return &FontValue{} })
	AddValue(gi.Filename(""), func() Value { return &FileValue{} })
	AddValue(keyfun.MapName(""), func() Value { return &KeyMapValue{} })
	AddValue(gti.Type{}, func() Value { return &TypeValue{} })
	AddValue(color.RGBA{}, func() Value { return &ColorValue{} })
	AddValue(key.Chord(""), func() Value { return &KeyChordValue{} })
	AddValue(time.Time{}, func() Value { return &TimeValue{} })
	AddValue(time.Duration(0), func() Value { return &DurationValue{} })
	AddValue(fi.FileTime{}, func() Value { return &TimeValue{} })
}

// Valuer is an interface that types can implement to specify the [Value]
// that should be used to represent them in the GUI. If the return value is nil,
// then the default [Value] for the value will be used. For example:
//
//	func (m *MyType) Value() giv.Value {
//		return &MyValue{}
//	}
type Valuer interface {
	Value() Value
}

// FieldValuer is an interface that struct types can implement to specify the
// [Value] that should be used to represent their fields in the GUI. If the
// return value is nil, then the default [Value] for the value will be used.
type FieldValuer interface {
	FieldValue(field string, fval any) Value
}

// ValueMap is a map from fully package path qualified type names to the corresponding
// [Value] objects that should be used to represent those types. You should add to this
// using [AddValue].
var ValueMap = map[string]func() Value{}

// AddValue indicates that the given value should be represented by the given [Value] object.
// For example:
//
//	AddValue(icons.Icon(""), func() Value { return &IconValue{} })
func AddValue(val any, fun func() Value) {
	nm := gti.TypeNameObj(val)
	ValueMap[nm] = fun
}

// StructTagVal returns the value for given key in given struct tag string
// uses reflect.StructTag Lookup method -- just a wrapper for external
// use (e.g., in Python code)
func StructTagVal(key, tags string) string {
	stag := reflect.StructTag(tags)
	val, _ := stag.Lookup(key)
	return val
}

// ToValue returns the appropriate Value for given item, based only on
// its type -- attempts to get the Valuer interface and failing that,
// falls back on default Kind-based options.  tags are optional tags, e.g.,
// from the field in a struct, that control the view properties -- see the gi wiki
// for details on supported tags -- these are NOT set for the view element, only
// used for options that affect what kind of view to create.
// See FieldToValue for version that takes into account the properties of the owner.
//
//gopy:interface=handle
func ToValue(it any, tags string) Value {
	if it == nil {
		return &NilValue{}
	}
	if vv, ok := it.(Valuer); ok {
		vvo := vv.Value()
		if vvo != nil {
			return vvo
		}
	}
	// try pointer version..
	if vv, ok := laser.PtrInterface(it).(Valuer); ok {
		vvo := vv.Value()
		if vvo != nil {
			return vvo
		}
	}

	if _, ok := it.(enums.BitFlag); ok {
		return &BitFlagValue{}
	}
	if _, ok := it.(enums.Enum); ok {
		return &EnumValue{}
	}

	typ := reflect.TypeOf(it)
	nptyp := laser.NonPtrType(typ)
	vk := typ.Kind()
	// fmt.Printf("vv val %v: typ: %v nptyp: %v kind: %v\n", it, typ.String(), nptyp.String(), vk)

	nptypnm := laser.LongTypeName(nptyp)
	if vvf, has := ValueMap[nptypnm]; has {
		vv := vvf()
		return vv
	}

	forceInline := false
	forceNoInline := false

	stag := reflect.StructTag(tags)
	vtag := stag.Get("view")

	switch vtag {
	case "inline":
		forceInline = true
	case "no-inline":
		forceNoInline = true
	}

	switch {
	case vk >= reflect.Int && vk <= reflect.Uint64:
		if vtag == "slider" {
			return &SliderValue{}
		}
		if _, ok := it.(fmt.Stringer); ok { // use stringer
			return &ValueBase{}
		}
		return &IntValue{}
	case vk == reflect.Bool:
		return &BoolValue{}
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		if vtag == "slider" {
			return &SliderValue{}
		}
		return &FloatValue{} // handles step, min / max etc
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		// todo: special edit with 2 fields..
		return &ValueBase{}
	case vk == reflect.Ptr:
		if ki.IsKi(nptyp) {
			return &KiValue{}
		}
		if laser.AnyIsNil(it) {
			return &NilValue{}
		}
		v := reflect.ValueOf(it)
		if !v.IsZero() {
			// note: interfaces go here:
			// fmt.Printf("vv indirecting on pointer: %v type: %v\n", it, nptyp.String())
			return ToValue(v.Elem().Interface(), tags)
		}
	case vk == reflect.Array, vk == reflect.Slice:
		v := reflect.ValueOf(it)
		sz := v.Len()
		eltyp := laser.SliceElType(it)
		if _, ok := it.([]byte); ok {
			return &ByteSliceValue{}
		}
		if _, ok := it.([]rune); ok {
			return &RuneSliceValue{}
		}
		isstru := (laser.NonPtrType(eltyp).Kind() == reflect.Struct)
		if !forceNoInline && (forceInline || (!isstru && sz <= gi.SystemSettings.SliceInlineLength && !ki.IsKi(eltyp))) {
			return &SliceInlineValue{}
		} else {
			return &SliceValue{}
		}
	case vk == reflect.Map:
		sz := laser.MapStructElsN(it)
		if !forceNoInline && (forceInline || sz <= gi.SystemSettings.MapInlineLength) {
			return &MapInlineValue{}
		} else {
			return &MapValue{}
		}
	case vk == reflect.Struct:
		nfld := laser.AllFieldsN(nptyp)
		if nfld > 0 && !forceNoInline && (forceInline || nfld <= gi.SystemSettings.StructInlineLength) {
			return &StructInlineValue{}
		} else {
			return &StructValue{}
		}
	case vk == reflect.Func:
		if laser.AnyIsNil(it) {
			return &NilValue{}
		}
		return &FuncValue{}
	}
	return &StringValue{} // fallback
}

// FieldToValue returns the appropriate Value for given field on a
// struct -- attempts to get the FieldValuer interface, and falls back on
// ToValue otherwise, using field value (fval)
//
//gopy:interface=handle
func FieldToValue(it any, field string, fval any) Value {
	if it == nil || field == "" {
		return ToValue(fval, "")
	}
	if vv, ok := it.(FieldValuer); ok {
		vvo := vv.FieldValue(field, fval)
		if vvo != nil {
			return vvo
		}
	}
	// try pointer version..
	if vv, ok := laser.PtrInterface(it).(FieldValuer); ok {
		vvo := vv.FieldValue(field, fval)
		if vvo != nil {
			return vvo
		}
	}

	typ := reflect.TypeOf(it)
	nptyp := laser.NonPtrType(typ)

	ftyp, ok := nptyp.FieldByName(field)
	if ok {
		return ToValue(fval, string(ftyp.Tag))
	}
	return ToValue(fval, "")
}
