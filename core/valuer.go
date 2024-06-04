// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/enums"
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
// It is used by [ToValue]. If a function returns nil, it falls
// back onto the next step. You can add to this using the [AddValueType]
// helper function. These functions must NOT call [Bind].
var ValueTypes = map[string]func(value any) Value{}

// AddValueType binds the given value type to the given [Value] type,
// meaning that [ToValue] will return a new [Value] of the given type
// when it receives values of the given value type. It uses [ValueTypes].
// This function is called with various standard types automatically.
func AddValueType[T any, W Value]() {
	var v T
	name := types.TypeNameValue(v)
	ValueTypes[name] = func(value any) Value {
		return tree.New[W]()
	}
}

// ValueConverters is a slice of functions that return a [Value]
// for a value, using optional tags context to inform the selection.
// It is used by [ToValue]. If a function returns nil,
// it falls back on the next function in the slice, and if all functions return nil,
// it falls back on the default bindings. These functions must NOT call [Bind].
// These functions are called in sequential order, so you can insert
// a function at the start to take precedence over others.
// You can add to this using the [AddValueConverter] helper function.
var ValueConverters []func(value any, tags reflect.StructTag) Value

// AddValueConverter adds a converter function to [ValueConverters].
func AddValueConverter(f func(value any, tags reflect.StructTag) Value) {
	ValueConverters = append(ValueConverters, f)
}

// NewValue converts the given value into an appropriate [Value]
// whose associated value is bound to the given value. The given value must
// be a pointer. It uses the given optional struct tags for additional context
// and to determine styling properties via [StyleFromTags]. It also adds the
// resulting [Value] to the given optional parent if it specified. The specifics
// on how it determines what type of [Value] to make are further
// documented on [ToValue].
func NewValue(value any, tags reflect.StructTag, parent ...tree.Node) Value {
	vw := ToValue(value, tags)
	if tags != "" {
		StyleFromTags(vw, tags)
	}
	Bind(value, vw)
	if len(parent) > 0 {
		parent[0].AddChild(vw)
	}
	return vw
}

// ToValue converts the given value into an appropriate [Value],
// using the given optional struct tags for additional context.
// The given value should typically be a pointer. It does NOT call [Bind];
// see [NewValue] for a version that does. It first checks the
// [Valuer] interface, then the [ValueTypes], then
// the [ValueConverters], and finally it falls back on a set of default
// bindings. If any step results in nil, it falls back on the next step.
func ToValue(value any, tags reflect.StructTag) Value {
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
	if !uv.IsValid() {
		return ToValue(reflect.New(rv.Type()).Interface(), tags)
	}
	typ := uv.Type()
	if vwt, ok := ValueTypes[types.TypeName(typ)]; ok {
		if vw := vwt(value); vw != nil {
			return vw
		}
	}
	for _, converter := range ValueConverters {
		if vw := converter(value, tags); vw != nil {
			return vw
		}
	}

	if _, ok := value.(enums.BitFlag); ok {
		return NewSwitches()
	}
	if enum, ok := value.(enums.Enum); ok {
		if len(enum.Values()) < 4 {
			return NewSwitches()
		}
		return NewChooser()
	}

	kind := typ.Kind()
	switch {
	case kind >= reflect.Int && kind <= reflect.Float64:
		if _, ok := value.(fmt.Stringer); ok {
			return NewTextField()
		}
		return NewSpinner()
	case kind == reflect.Bool:
		return NewSwitch()
	}

	return NewTextField()
}
