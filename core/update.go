// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"

	"cogentcore.org/core/tree"
)

// UpdateWidget updates the widget by running [WidgetBase.Updaters] in
// sequential descending (reverse) order after calling [WidgetBase.ValueUpdate].
// This includes applying the result of [WidgetBase.Make].
//
// UpdateWidget differs from [WidgetBase.Update] in that it only updates the widget
// itself and not any of its children. Also, it does not restyle the widget or trigger
// a new layout pass, while [WidgetBase.Update] does. End-user code should typically
// call [WidgetBase.Update], not UpdateWidget.
func (wb *WidgetBase) UpdateWidget() *WidgetBase {
	if wb.ValueUpdate != nil {
		wb.ValueUpdate()
	}
	wb.RunUpdaters()
	return wb
}

// UpdateTree calls [WidgetBase.UpdateWidget] on every widget in the tree
// starting with this one and going down.
func (wb *WidgetBase) UpdateTree() {
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		cwb.UpdateWidget()
		return tree.Continue
	})
}

// Update updates the widget and all of its children by running [WidgetBase.UpdateWidget]
// and [WidgetBase.Style] on each one, and triggering a new layout pass with
// [WidgetBase.NeedsLayout]. It is the main way that end users should trigger widget
// updates, and it is guaranteed to fully update a widget to the current state.
// For example, it should be called after making any changes to the core properties
// of a widget, such as the text of [Text], the icon of a [Button], or the slice
// of a [Table].
//
// Update differs from [WidgetBase.UpdateWidget] in that it updates the widget and all
// of its children down the tree, whereas [WidgetBase.UpdateWidget] only updates the widget
// itself. Also, Update also calls [WidgetBase.Style] and [WidgetBase.NeedsLayout],
// whereas [WidgetBase.UpdateWidget] does not. End-user code should typically call Update,
// not [WidgetBase.UpdateWidget].
//
// If you are calling this in a separate goroutine outside of the main
// configuration, rendering, and event handling structure, you need to
// call [WidgetBase.AsyncLock] and [WidgetBase.AsyncUnlock] before and
// after this, respectively.
func (wb *WidgetBase) Update() { //types:add
	if DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace Update:", wb)
	}
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		cwb.UpdateWidget()
		cw.Style()
		return tree.Continue
	})
	wb.NeedsLayout()
}

// UpdateRender is the same as [WidgetBase.Update], except that it calls
// [WidgetBase.NeedsRender] instead of [WidgetBase.NeedsLayout].
// This should be called when the changes made to the widget do not
// require a new layout pass (if you change the size, spacing, alignment,
// or other layout properties of the widget, you need a new layout pass
// and should call [WidgetBase.Update] instead).
func (wb *WidgetBase) UpdateRender() {
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		cwb.UpdateWidget()
		cw.Style()
		return tree.Continue
	})
	wb.NeedsRender()
}
