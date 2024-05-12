// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/base/reflectx"
)

// ValueWidget is a widget that has an associated value representation.
// It can be bound to a value using [Bind].
type ValueWidget interface {
	Widget

	// WidgetValue returns the pointer to the associated value of the widget.
	WidgetValue() any
}

// ValueWidgetSetter is an optional interface that [ValueWidget]s can implement
// to customize how the associated widget value is set from the bound value.
type ValueWidgetSetter interface {

	// SetWidgetValue sets the associated widget value from the bound value.
	SetWidgetValue(value any) error
}

// Bind binds the given value to the given [ValueWidget] such that the values of
// the two will be linked and updated appropriately after [events.Change] events
// and during [Widget.ConfigWidget]. It returns the widget to enable method chaining.
func Bind[T ValueWidget](value any, vw T) T {
	wb := vw.AsWidget()
	wb.ValueUpdate = func() {
		if vws, ok := ValueWidget(vw).(ValueWidgetSetter); ok {
			ErrorSnackbar(vw, vws.SetWidgetValue(value))
		} else {
			ErrorSnackbar(vw, reflectx.SetRobust(vw.WidgetValue(), value))
		}
	}
	wb.ValueOnChange = func() {
		ErrorSnackbar(vw, reflectx.SetRobust(value, vw.WidgetValue()))
	}
	return vw
}
