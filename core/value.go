// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
)

// ValueWidget is a widget that has an associated value representation.
// ValueWidgets can be bound to values using [Bind].
type ValueWidget interface {
	Widget

	// WidgetValue returns the pointer to the associated value of the widget.
	WidgetValue() any
}

// Bind binds the given value to the given [ValueWidget] such that the values of
// the two will be linked and updated appropriately after [events.Change] events
// and during [Widget.ConfigWidget].
func Bind(value any, vw ValueWidget) {
	wb := vw.AsWidget()
	wb.ValueUpdate = func() {
		errors.Log(reflectx.SetRobust(vw.WidgetValue(), value))
	}
	wb.ValueOnChange = func() {
		errors.Log(reflectx.SetRobust(value, vw.WidgetValue()))
	}
}
