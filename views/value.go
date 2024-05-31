// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

//go:generate core generate

import (
	"reflect"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
)

// InitValueButton configures the given [core.Value] to open a dialog representing
// its value in accordance with the given dialog construction function when clicked.
// It also sets the tooltip of the widget appropriately. If allowReadOnly is false,
// the dialog will not be opened if the widget is read only. It also takes an optional
// function to call after the dialog is accepted.
func InitValueButton(v core.Value, allowReadOnly bool, make func(b *core.Body), after ...func()) {
	wb := v.AsWidget()
	// windows are never new on mobile
	if !core.TheApp.Platform().IsMobile() {
		wb.SetTooltip("[Shift: new window]")
	}
	v.OnClick(func(e events.Event) {
		if allowReadOnly || !wb.IsReadOnly() {
			v.SetFlag(e.HasAnyModifier(key.Shift), ValueDialogNewWindow) // TODO(config): make this a widget flag
			OpenValueDialog(v, v.AsWidget(), make, after...)
		}
	})
}

// OpenValueDialog opens a new value dialog for the given [core.Value] using the
// given context widget, the given function for constructing the dialog, and the
// optional given function to call after the dialog is accepted.
func OpenValueDialog(v core.Value, ctx core.Widget, make func(b *core.Body), after ...func()) {
	opv := reflectx.UnderlyingPointer(reflect.ValueOf(v.WidgetValue()))
	if !opv.IsValid() {
		return
	}
	obj := opv.Interface()
	if core.RecycleDialog(obj) {
		return
	}
	wb := v.AsWidget()
	d := core.NewBody().AddTitle(wb.ValueContext).AddText(wb.Tooltip)
	make(d)

	// if we don't have anything specific for ok events,
	// we just register an OnClose event and skip the
	// OK and Cancel buttons
	if len(after) == 0 {
		d.OnClose(func(e events.Event) {
			wb.SendChange()
			v.Update()
		})
	} else {
		// otherwise, we have to make the bottom bar
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).OnClick(func(e events.Event) {
				after[0]()
				wb.SendChange()
				v.Update()
			})
		})
	}

	ds := d.NewFullDialog(ctx)
	if v.Is(ValueDialogNewWindow) {
		ds.SetNewWindow(true)
	}
	ds.Run()
}
