// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"

	"cogentcore.org/core/tree"
)

// Updater adds a new function to [WidgetBase.Updaters], which are called in sequential
// descending (reverse) order in [WidgetBase.UpdateWidget] to update the widget.
func (wb *WidgetBase) Updater(updater func()) {
	wb.Updaters = append(wb.Updaters, updater)
}

// Maker adds a new function to [WidgetBase.Makers], which are called in sequential
// ascending order in [WidgetBase.Make] to make the plan for how the widget's children
// should be configured.
func (wb *WidgetBase) Maker(maker func(p *Plan)) {
	wb.Makers = append(wb.Makers, maker)
}

// UpdateWidget updates the widget by running [WidgetBase.Updaters] in
// sequential descending (reverse) order after calling [WidgetBase.ValueUpdate].
// This includes applying the result of [WidgetBase.Make].
//
// UpdateWidget differs from [WidgetBase.Update] in that it only updates the widget
// itself and not any of its children. Also, it does not restyle the widget or trigger
// a new layout pass, while [WidgetBase.Update] does. End-user code should typically
// call [WidgetBase.Update], not UpdateWidget.
func (wb *WidgetBase) UpdateWidget() {
	if wb.ValueUpdate != nil {
		wb.ValueUpdate()
	}
	for i := len(wb.Updaters) - 1; i >= 0; i-- {
		wb.Updaters[i]()
	}
}

// updateFromMake updates the widget using [WidgetBase.Make].
// It is the base Updater added to [WidgetBase.Updaters] in OnInit.
func (wb *WidgetBase) updateFromMake() {
	p := Plan{}
	wb.Make(&p)
	p.Update(wb)
}

// Make makes a plan for how the widget's children should be structured.
// It does this by running [WidgetBase.Makers] in sequential ascending order.
// Make is called by [WidgetBase.UpdateWidget] to determine how the widget
// should be updated.
func (wb *WidgetBase) Make(p *Plan) {
	for _, maker := range wb.Makers {
		maker(p)
	}
}

// UpdateTree calls [WidgetBase.UpdateWidget] on every widget in the tree
// starting with this one and going down.
func (wb *WidgetBase) UpdateTree() {
	wb.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		wb.UpdateWidget()
		return tree.Continue
	})
}

// Update updates the widget and all of its children by running [WidgetBase.UpdateWidget]
// and [WidgetBase.ApplyStyle] on each one, and triggering a new layout pass with
// [WidgetBase.NeedsLayout]. It is the main way that end users should trigger widget
// updates, and it is guaranteed to fully update a widget to the current state.
// For example, it should be called after making any changes to the core properties
// of a widget, such as the text of [Text], the icon of a [Button], or the slice
// of a [Table].
//
// Update differs from [WidgetBase.UpdateWidget] in that it updates the widget and all
// of its children down the tree, whereas [WidgetBase.UpdateWidget] only updates the widget
// itself. Also, Update also calls [WidgetBase.ApplyStyle] and [WidgetBase.NeedsLayout],
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
	wb.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		wb.UpdateWidget()
		wi.ApplyStyle()
		return tree.Continue
	})
	wb.NeedsLayout()
}
