// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import "cogentcore.org/core/types"

// ValueWidgeter is an interface that types can implement to specify the
// [ValueWidget] that should be used to represent them in the GUI.
type ValueWidgeter interface {

	// ValueWidget returns the [ValueWidget] that should be used to represent
	// the value in the GUI. If it returns nil, then [ToValueWidget] will
	// fall back onto the next step. This function does NOT need to call [core.Bind].
	ValueWidget() ValueWidget
}

// ValueWidgetTypes is a map of functions that return a [ValueWidget]
// for a value of a certain fully package path qualified type name.
// You can add to this using [AddValueWidgetType]. It is used by [ToValueWidget].
// If a function returns nil, it falls back onto the next step.
var ValueWidgetTypes = map[string]func(value any) ValueWidget{}

// ValueWidgetConverters is a slice of functions that return a [ValueWidget]
// for a value. It is used by [ToValueWidget]. If a function returns nil,
// it falls back on the next function in the slice, and if all functions return nil,
// it falls back on the default bindings. These functions do NOT need to call
// [core.Bind].
var ValueWidgetConverters []func(value any) ValueWidget

// ToValueWidget converts the given value into an appropriate [ValueWidget]
// whose associated value is bound to the given value. It first checks the
// [ValueWidgeter] interface, then the [ValueWidgetTypes], then the
// [ValueWidgetConverters], and finally it falls back on a set of default
// bindings. If any step results in nil, it falls back on the next step.
func ToValueWidget(value any) ValueWidget {
	if vwr, ok := value.(ValueWidgeter); ok {
		if vw := vwr.ValueWidget(); vw != nil {
			return vw
		}
	}
	if vwt, ok := ValueWidgetTypes[types.TypeNameObj(value)]; ok {
		if vw := vwt(value); vw != nil {
			return vw
		}
	}
	for _, converter := range ValueWidgetConverters {
		if vw := converter(value); vw != nil {
			return vw
		}
	}
	return nil
}
