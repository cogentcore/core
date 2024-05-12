// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

// ValueWidgeter is an interface that types can implement to specify the
// [ValueWidget] that should be used to represent them in the GUI.
type ValueWidgeter interface {

	// ValueWidget returns the [ValueWidget] that should be used to represent
	// the value in the GUI. If it returns nil, then the default [ValueWidget]
	// for the value will be used instead, as specified by the documentation
	// of [ToValueWidget].
	ValueWidget() ValueWidget
}

// ValueWidgetConverters is a slice of functions that convert a value into
// a [ValueWidget]. It is used by [ToValueWidget], after [ValueWidgeter] but
// before the default bindings. If a function returns nil, it falls back on
// the next function in the slice, and if all functions return nil, it falls
// back on the default bindings.
var ValueWidgetConverters []func(value any) ValueWidget

// ToValueWidget converts the given value into an appropriate [ValueWidget]
// whose associated value is bound to the given value. It first checks the
// [ValueWidgeter] interface, then the [ValueWidgetConverters], and finally
// falls back on a set of default bindings. If any step results in nil, it
// falls back on the next step.
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
	return nil
}
