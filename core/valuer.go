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

// ValueWidgeter is an interface that types can implement to specify the
// [ValueWidget] that should be used to represent them in the GUI.
type ValueWidgeter interface {

	// ValueWidget returns the [ValueWidget] that should be used to represent
	// the value in the GUI. If it returns nil, then [ToValueWidget] will
	// fall back onto the next step. This function must NOT call [Bind].
	ValueWidget() ValueWidget
}

// ValueWidgetTypes is a map of functions that return a [ValueWidget]
// for a value of a certain fully package path qualified type name.
// It is used by [ToValueWidget]. If a function returns nil, it falls
// back onto the next step. You can add to this using the [AddValueWidgetType]
// helper function. These functions must NOT call [Bind].
var ValueWidgetTypes = map[string]func(value any) ValueWidget{}

// ValueWidgetConverters is a slice of functions that return a [ValueWidget]
// for a value. It is used by [ToValueWidget]. If a function returns nil,
// it falls back on the next function in the slice, and if all functions return nil,
// it falls back on the default bindings. These functions must NOT call [Bind].
// These functions are called in sequential order, so you can insert
// a function at the start to take precedence over others.
var ValueWidgetConverters []func(value any) ValueWidget

// AddValueWidgetType binds the given value type to the given [ValueWidget] type,
// meaning that [ToValueWidget] will return a new [ValueWidget] of the given type
// when it receives values of the given value type. It uses [ValueWidgetTypes].
// This function is called with various standard types automatically.
func AddValueWidgetType[T any, W ValueWidget]() {
	var v T
	name := types.TypeNameValue(v)
	ValueWidgetTypes[name] = func(value any) ValueWidget {
		return tree.New[W]()
	}
}

func init() {
	AddValueWidgetType[bool, *Switch]()
}

// NewValueWidget converts the given value into an appropriate [ValueWidget]
// whose associated value is bound to the given value. The given value should
// typically be a pointer. It also adds the resulting [ValueWidget] to the given
// optional parent if it specified. The specifics on how it determines what type
// of [ValueWidget] to make are further documented on [ToValueWidget].
func NewValueWidget(value any, parent ...tree.Node) ValueWidget {
	vw := ToValueWidget(value)
	Bind(value, vw)
	if len(parent) > 0 {
		parent[0].AddChild(vw)
	}
	return vw
}

// ToValueWidget converts the given value into an appropriate [ValueWidget].
// The given value should typically be a pointer. It does NOT call [Bind];
// see [NewValueWidget] for a version that does. It first checks the
// [ValueWidgeter] interface, then the [ValueWidgetConverters], then
// the [ValueWidgetTypes], and finally it falls back on a set of default
// bindings. If any step results in nil, it falls back on the next step.
func ToValueWidget(value any) ValueWidget {
	if vwr, ok := value.(ValueWidgeter); ok {
		if vw := vwr.ValueWidget(); vw != nil {
			return vw
		}
	}
	for _, converter := range ValueWidgetConverters {
		if vw := converter(value); vw != nil {
			return vw
		}
	}
	typ := reflectx.NonPointerType(reflect.TypeOf(value))
	if vwt, ok := ValueWidgetTypes[types.TypeName(typ)]; ok {
		if vw := vwt(value); vw != nil {
			return vw
		}
	}

	if _, ok := value.(enums.Enum); ok {
		return NewChooser()
	}

	kind := typ.Kind()
	switch {
	case kind >= reflect.Int && kind <= reflect.Float64:
		if _, ok := value.(fmt.Stringer); ok {
			return NewTextField()
		}
		return NewSpinner()
	}
	return NewTextField()
}
