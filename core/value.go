// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
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
// to customize how the associated widget value is set from the [WidgetBase.BindValue].
type ValueSetter interface {

	// SetWidgetValue sets the associated widget value from the [WidgetBase.BindValue].
	SetWidgetValue() error
}

// OnBinder is an optional interface that [Value]s can implement to
// do something when the widget is bound to [WidgetBase.BindValue].
type OnBinder interface {

	// OnBind is called when the widget is bound to [WidgetBase.BindValue].
	OnBind()
}

// Bind binds the given value to the given [Value] such that the values of
// the two will be linked and updated appropriately after [events.Change] events
// and during [Widget.ConfigWidget]. It returns the widget to enable method chaining.
func Bind[T Value](value any, vw T) T {
	wb := vw.AsWidget()
	if value == wb.BindValue {
		return vw
	}
	wb.BindValue = value
	wb.ValueUpdate = func() {
		if vws, ok := any(vw).(ValueSetter); ok {
			ErrorSnackbar(vw, vws.SetWidgetValue())
		} else {
			ErrorSnackbar(vw, reflectx.SetRobust(vw.WidgetValue(), value))
		}
	}
	wb.ValueOnChange = func() {
		ErrorSnackbar(vw, reflectx.SetRobust(value, vw.WidgetValue()))
	}
	if ob, ok := any(vw).(OnBinder); ok {
		ob.OnBind()
	}
	return vw
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
