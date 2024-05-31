// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"reflect"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
)

// Value is a widget that has an associated value representation.
// It can be bound to a value using [Bind].
type Value interface {
	Widget

	// WidgetValue returns the pointer to the associated value of the widget.
	WidgetValue() any
}

// ValueSetter is an optional interface that [Value]s can implement
// to customize how the associated widget value is set from the given value.
type ValueSetter interface {

	// SetWidgetValue sets the associated widget value from the given value.
	SetWidgetValue(value any) error
}

// OnBinder is an optional interface that [Value]s can implement to
// do something when the widget is bound to the given value.
type OnBinder interface {

	// OnBind is called when the widget is bound to given value.
	OnBind(value any)
}

// Bind binds the given value to the given [Value] such that the values of
// the two will be linked and updated appropriately after [events.Change] events
// and during [Widget.UpdateWidget]. It returns the widget to enable method chaining.
func Bind[T Value](value any, vw T) T {
	wb := vw.AsWidget()
	wb.ValueUpdate = func() {
		if vws, ok := any(vw).(ValueSetter); ok {
			ErrorSnackbar(vw, vws.SetWidgetValue(value))
		} else {
			ErrorSnackbar(vw, reflectx.SetRobust(vw.WidgetValue(), value))
		}
	}
	wb.ValueOnChange = func() {
		ErrorSnackbar(vw, reflectx.SetRobust(value, vw.WidgetValue()))
	}
	wb.ValueContext = labels.FriendlyTypeName(reflectx.NonPointerType(reflect.TypeOf(value)))
	if ob, ok := any(vw).(OnBinder); ok {
		ob.OnBind(value)
	}
	wb.ValueUpdate() // we update it with the initial value immediately
	return vw
}

// Note: SetValueContext must be defined manually so that it is not generated
// for all embedding widget types.

// SetValueContext sets the [WidgetBase.ValueContext] of the widget,
// which is a record of parent value names that have led up to this [Value].
func (wb *WidgetBase) SetValueContext(context string) *WidgetBase {
	wb.ValueContext = context
	return wb
}

// JoinValueContext returns a [WidgetBase.ValueContext] string composed
// of two elements, with a • separator, handling the cases where
// either or both can be empty.
func JoinValueContext(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + " • " + b
	}
}
