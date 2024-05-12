// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

// ValueWidget is a widget that has an associated value representation.
// ValueWidgets can be bound to values using [Bind].
type ValueWidget interface {
	Widget

	// WidgetValue returns the pointer to the associated value of the widget.
	WidgetValue() any
}
