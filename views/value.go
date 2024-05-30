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

// ConfigDialoger is an optional interface that [Value]s may implement to
// indicate that they have a dialog associated with them that is configured
// with the ConfigDialog method. The dialog body itself is constructed and run
// using [OpenDialog].
type ConfigDialoger interface {
	// ConfigDialog adds content to the given dialog body for this value.
	// The bool return is false if the value does not use this method
	// (e.g., for simple menu choosers).
	// The returned function is an optional closure to be called
	// in the Ok case, for cases where extra logic is required.
	ConfigDialog(d *core.Body) (bool, func())
}

// OpenDialoger is an optional interface that [Value]s may implement to
// indicate that they have a dialog associated with them that is created,
// configured, and run with the OpenDialog method. This method typically
// calls a separate ConfigDialog method. If the [Value] does not implement
// [OpenDialoger] but does implement [ConfigDialoger], then [OpenDialogBase]
// will be used to create and run the dialog, and [ConfigDialoger.ConfigDialog]
// will be used to configure it.
type OpenDialoger interface {
	// OpenDialog opens the dialog for this Value.
	// Given function closure is called for the Ok action, after value
	// has been updated, if using the dialog as part of another control flow.
	// Note that some cases just pop up a menu chooser, not a full dialog.
	OpenDialog(ctx core.Widget, fun func())
}

// InitValueButton configures the core.Value to open the dialog
// for the given value when clicked and have the appropriate tooltip for that.
// If allowReadOnly is false, the dialog will not be opened if the value
// is read only.
func InitValueButton(v core.Value, allowReadOnly bool) {
	wb := v.AsWidget()
	// windows are never new on mobile
	if !core.TheApp.Platform().IsMobile() {
		wb.SetTooltip("[Shift: new window]")
	}
	v.OnClick(func(e events.Event) {
		if allowReadOnly || !wb.IsReadOnly() {
			v.SetFlag(e.HasAnyModifier(key.Shift), ValueDialogNewWindow)
			OpenDialogValue(v, v.AsWidget(), nil, nil)
		}
	})
}

// OpenDialogValue opens any applicable dialog for the given value in the
// context of the given widget. It first tries [OpenDialoger], then
// [ConfigDialoger] with [OpenDialogBase]. If both of those fail, it
// returns false. It calls the given beforeFunc before opening any dialog.
func OpenDialogValue(v core.Value, ctx core.Widget, fun, beforeFunc func()) bool {
	if od, ok := v.(OpenDialoger); ok {
		if beforeFunc != nil {
			beforeFunc()
		}
		od.OpenDialog(ctx, fun)
		return true
	}
	if cd, ok := v.(ConfigDialoger); ok {
		if beforeFunc != nil {
			beforeFunc()
		}
		OpenDialogValueBase(v, cd, ctx, fun)
		return true
	}
	return false
}

// OpenDialogValueBase is a helper for [OpenDialog] for cases that
// do not implement [OpenDialoger] but do implement [ConfigDialoger]
// to configure the dialog contents.
func OpenDialogValueBase(v core.Value, cd ConfigDialoger, ctx core.Widget, fun func()) {
	opv := reflectx.UnderlyingPointer(reflect.ValueOf(v.WidgetValue()))
	if !opv.IsValid() {
		return
	}
	// title, _, _ := vd.GetTitle()
	title := "need a title"
	obj := opv.Interface()
	if core.RecycleDialog(obj) {
		return
	}
	d := core.NewBody().AddTitle(title) // .AddText(v.Doc())
	ok, okfun := cd.ConfigDialog(d)
	if !ok {
		return
	}

	wb := v.AsWidget()

	// if we don't have anything specific for ok events,
	// we just register an OnClose event and skip the
	// OK and Cancel buttons
	if okfun == nil && fun == nil {
		d.OnClose(func(e events.Event) {
			wb.SendChange()
			v.Update()
		})
	} else {
		// otherwise, we have to make the bottom bar
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).OnClick(func(e events.Event) {
				if okfun != nil {
					okfun()
				}
				wb.SendChange()
				v.Update()
				if fun != nil {
					fun()
				}
			})
		})
	}

	ds := d.NewFullDialog(ctx)
	if v.Is(ValueDialogNewWindow) {
		ds.SetNewWindow(true)
	}
	ds.Run()
}
