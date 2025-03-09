// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image/color"
	"reflect"
	"time"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// Valuer is an interface that types can implement to specify the
// [Value] that should be used to represent them in the GUI.
type Valuer interface {

	// Value returns the [Value] that should be used to represent
	// the value in the GUI. If it returns nil, then [ToValue] will
	// fall back onto the next step. This function must NOT call [Bind].
	Value() Value
}

// ValueTypes is a map of functions that return a [Value]
// for a value of a certain fully package path qualified type name.
// It is used by [toValue]. If a function returns nil, it falls
// back onto the next step. You can add to this using the [AddValueType]
// helper function. These functions must NOT call [Bind].
var ValueTypes = map[string]func(value any) Value{}

// AddValueType binds the given value type to the given [Value] [tree.NodeValue]
// type, meaning that [toValue] will return a new [Value] of the given type
// when it receives values of the given value type. It uses [ValueTypes].
// This function is called with various standard types automatically.
func AddValueType[T any, W tree.NodeValue]() {
	var v T
	name := types.TypeNameValue(v)
	ValueTypes[name] = func(value any) Value {
		return any(tree.New[W]()).(Value)
	}
}

// NewValue converts the given value into an appropriate [Value]
// whose associated value is bound to the given value. The given value must
// be a pointer. It uses the given optional struct tags for additional context
// and to determine styling properties via [styleFromTags]. It also adds the
// resulting [Value] to the given optional parent if it specified. The specifics
// on how it determines what type of [Value] to make are further
// documented on [toValue].
func NewValue(value any, tags reflect.StructTag, parent ...tree.Node) Value {
	vw := toValue(value, tags)
	if tags != "" {
		styleFromTags(vw, tags)
	}
	Bind(value, vw, string(tags))
	if len(parent) > 0 {
		parent[0].AsTree().AddChild(vw)
	}
	return vw
}

// toValue converts the given value into an appropriate [Value],
// using the given optional struct tags for additional context.
// The given value should typically be a pointer. It does NOT call [Bind];
// see [NewValue] for a version that does. It first checks the
// [Valuer] interface, then the [ValueTypes], and finally it falls
// back on a set of default bindings. If any step results in nil,
// it falls back on the next step.
func toValue(value any, tags reflect.StructTag) Value {
	if vwr, ok := value.(Valuer); ok {
		if vw := vwr.Value(); vw != nil {
			return vw
		}
	}
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return NewText()
	}
	uv := reflectx.Underlying(rv)
	typ := uv.Type()
	if vwt, ok := ValueTypes[types.TypeName(typ)]; ok {
		if vw := vwt(value); vw != nil {
			return vw
		}
	}

	// Default bindings:

	if _, ok := value.(enums.BitFlag); ok {
		return NewSwitches()
	}
	if enum, ok := value.(enums.Enum); ok {
		if len(enum.Values()) < 4 {
			return NewSwitches()
		}
		return NewChooser()
	}
	if _, ok := value.(color.Color); ok {
		return NewColorButton()
	}
	if _, ok := value.(tree.Node); ok {
		return NewTreeButton()
	}

	inline := tags.Get("display") == "inline"
	noInline := tags.Get("display") == "no-inline"

	kind := typ.Kind()
	switch {
	case kind >= reflect.Int && kind <= reflect.Float64:
		if _, ok := value.(fmt.Stringer); ok {
			return NewTextField()
		}
		return NewSpinner()
	case kind == reflect.Bool:
		return NewSwitch()
	case kind == reflect.Struct:
		num := reflectx.NumAllFields(uv)
		if !noInline && (inline || num <= SystemSettings.StructInlineLength) {
			return NewForm().SetInline(true)
		}
		return NewFormButton()
	case kind == reflect.Map:
		len := uv.Len()
		if !noInline && (inline || len <= SystemSettings.MapInlineLength) {
			return NewKeyedList().SetInline(true)
		}
		return NewKeyedListButton()
	case kind == reflect.Array, kind == reflect.Slice:
		sz := uv.Len()
		elemType := reflectx.SliceElementType(value)
		if _, ok := value.([]byte); ok {
			return NewTextField()
		}
		if _, ok := value.([]rune); ok {
			return NewTextField()
		}
		isStruct := (reflectx.NonPointerType(elemType).Kind() == reflect.Struct)
		if !noInline && (inline || (!isStruct && sz <= SystemSettings.SliceInlineLength && !tree.IsNode(elemType))) {
			return NewInlineList()
		}
		return NewListButton()
	case kind == reflect.Func:
		return NewFuncButton()
	}

	return NewTextField() // final fallback
}

func init() {
	AddValueType[icons.Icon, IconButton]()
	AddValueType[time.Time, TimeInput]()
	AddValueType[time.Duration, DurationInput]()
	AddValueType[types.Type, TypeChooser]()
	AddValueType[Filename, FileButton]()
	// AddValueType[FontName, FontButton]()
	AddValueType[FontName, TextField]()
	AddValueType[keymap.MapName, KeyMapButton]()
	AddValueType[key.Chord, KeyChordButton]()
	AddValueType[HighlightingName, HighlightingButton]()
}
