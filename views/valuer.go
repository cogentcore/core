// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/tree"
)

// This file handles converting values to [Value]s.

func init() {
	core.SettingsWindow = SettingsWindow
	core.InspectorWindow = InspectorWindow

	AddValue(icons.Icon(""), func() Value { return &IconValue{} })
	AddValue(core.FontName(""), func() Value { return &FontValue{} })
	AddValue(core.Filename(""), func() Value { return &FileValue{} })
	AddValue(keyfun.MapName(""), func() Value { return &KeyMapValue{} })
	AddValue(gti.Type{}, func() Value { return &TypeValue{} })
	AddValue(color.RGBA{}, func() Value { return &ColorValue{} })
	AddValue(image.Uniform{}, func() Value { return &ColorValue{} })
	AddValue(key.Chord(""), func() Value { return &KeyChordValue{} })
	AddValue(time.Time{}, func() Value { return &TimeValue{} })
	AddValue(time.Duration(0), func() Value { return &DurationValue{} })
	AddValue(fileinfo.FileTime{}, func() Value { return &TimeValue{} })
}

// Valuer is an interface that types can implement to specify the [Value]
// that should be used to represent them in the GUI. If the return value is nil,
// then the default [Value] for the value will be used. For example:
//
//	func (m *MyType) Value() views.Value {
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

// ToValue converts the given value into its appropriate [Value] representation,
// using its type, tags, and value. It checks the [Valuer] interface and the
// [ValueMap] before falling back on checks for standard primitive and compound
// types. If it can not find any good representation for the value, it falls back
// on [StringValue]. The tags are optional tags in [reflect.StructTag] format that
// can affect what [Value] is returned; see the Tags page in the Cogent Core Docs
// to learn more. You should use [FieldToValue] when making a value in the context
// of a broader struct owner.
//
//gopy:interface=handle
func ToValue(val any, tags string) Value {
	if val == nil {
		return &NilValue{}
	}
	if vl, ok := val.(Valuer); ok {
		v := vl.Value()
		if v != nil {
			return v
		}
	}
	if vl, ok := laser.PtrInterface(val).(Valuer); ok {
		v := vl.Value()
		if v != nil {
			return v
		}
	}

	if _, ok := val.(enums.BitFlag); ok {
		return &BitFlagValue{}
	}
	if _, ok := val.(enums.Enum); ok {
		return &EnumValue{}
	}

	typ := reflect.TypeOf(val)
	nptyp := laser.NonPtrType(typ)
	vk := typ.Kind()

	nptypnm := laser.LongTypeName(nptyp)
	if vf, has := ValueMap[nptypnm]; has {
		v := vf()
		if v != nil {
			return v
		}
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
	case vk >= reflect.Int && vk <= reflect.Float64:
		if vtag == "slider" {
			return &SliderValue{}
		}
		if _, ok := val.(fmt.Stringer); ok {
			return &StringValue{}
		}
		return &NumberValue{}
	case vk == reflect.Bool:
		return &BoolValue{}
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		// TODO: special value for complex numbers with two fields
		return &StringValue{}
	case vk == reflect.Ptr:
		if tree.IsNode(nptyp) {
			return &KiValue{}
		}
		if laser.AnyIsNil(val) {
			return &NilValue{}
		}
		v := reflect.ValueOf(val)
		if !v.IsZero() {
			return ToValue(v.Elem().Interface(), tags)
		}
	case vk == reflect.Array, vk == reflect.Slice:
		v := reflect.ValueOf(val)
		sz := v.Len()
		eltyp := laser.SliceElType(val)
		if _, ok := val.([]byte); ok {
			return &ByteSliceValue{}
		}
		if _, ok := val.([]rune); ok {
			return &RuneSliceValue{}
		}
		isstru := (laser.NonPtrType(eltyp).Kind() == reflect.Struct)
		if !forceNoInline && (forceInline || (!isstru && sz <= core.SystemSettings.SliceInlineLength && !tree.IsNode(eltyp))) {
			return &SliceInlineValue{}
		} else {
			return &SliceValue{}
		}
	case vk == reflect.Map:
		sz := laser.MapStructElsN(val)
		if !forceNoInline && (forceInline || sz <= core.SystemSettings.MapInlineLength) {
			return &MapInlineValue{}
		} else {
			return &MapValue{}
		}
	case vk == reflect.Struct:
		nfld := laser.AllFieldsN(nptyp)
		if nfld > 0 && !forceNoInline && (forceInline || nfld <= core.SystemSettings.StructInlineLength) {
			return &StructInlineValue{}
		} else {
			return &StructValue{}
		}
	case vk == reflect.Func:
		if laser.AnyIsNil(val) {
			return &NilValue{}
		}
		return &FuncValue{}
	}
	return &StringValue{} // fallback
}

// FieldToValue converts the given value into its appropriate [Value] representation,
// using its type, tags, value, field name, and parent struct. It checks [FieldValuer]
// before falling back on [ToValue], which you should use for values that are not struct
// fields.
//
//gopy:interface=handle
func FieldToValue(str any, field string, val any) Value {
	if str == nil || field == "" {
		return ToValue(val, "")
	}
	if vl, ok := str.(FieldValuer); ok {
		v := vl.FieldValue(field, val)
		if v != nil {
			return v
		}
	}
	if vl, ok := laser.PtrInterface(str).(FieldValuer); ok {
		v := vl.FieldValue(field, val)
		if v != nil {
			return v
		}
	}

	typ := reflect.TypeOf(str)
	nptyp := laser.NonPtrType(typ)

	ftyp, ok := nptyp.FieldByName(field)
	if ok {
		return ToValue(val, string(ftyp.Tag))
	}
	return ToValue(val, "")
}
