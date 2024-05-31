// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"reflect"
	"strings"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
)

// Value is a widget that has an associated value representation.
// It can be bound to a value using [Bind].
type Value interface {
	Widget

	// WidgetValue returns the pointer to the associated value of the widget.
	WidgetValue() any
}

// ValueSetter is an optional interface that [Value]s can implement
// to customize how the associated widget value is set from the given value.
type ValueSetter interface {

	// SetWidgetValue sets the associated widget value from the given value.
	SetWidgetValue(value any) error
}

// OnBinder is an optional interface that [Value]s can implement to
// do something when the widget is bound to the given value.
type OnBinder interface {

	// OnBind is called when the widget is bound to given value.
	OnBind(value any)
}

// Bind binds the given value to the given [Value] such that the values of
// the two will be linked and updated appropriately after [events.Change] events
// and during [Widget.UpdateWidget]. It returns the widget to enable method chaining.
func Bind[T Value](value any, vw T) T {
	wb := vw.AsWidget()
	wb.ValueUpdate = func() {
		if vws, ok := any(vw).(ValueSetter); ok {
			ErrorSnackbar(vw, vws.SetWidgetValue(value))
		} else {
			ErrorSnackbar(vw, reflectx.SetRobust(vw.WidgetValue(), value))
		}
	}
	wb.ValueOnChange = func() {
		ErrorSnackbar(vw, reflectx.SetRobust(value, vw.WidgetValue()))
	}
	wb.ValueContext = labels.FriendlyTypeName(reflectx.NonPointerType(reflect.TypeOf(value)))
	if ob, ok := any(vw).(OnBinder); ok {
		ob.OnBind(value)
	}
	wb.ValueUpdate() // we update it with the initial value immediately
	return vw
}

// Note: SetValueContext must be defined manually so that it is not generated
// for all embedding widget types.

// SetValueContext sets the [WidgetBase.ValueContext] of the widget,
// which is a record of parent value names that have led up to this [Value].
func (wb *WidgetBase) SetValueContext(context string) *WidgetBase {
	wb.ValueContext = context
	return wb
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

const shiftNewWindow = "[Shift: new window]"

// InitValueButton configures the given [Value] to open a dialog representing
// its value in accordance with the given dialog construction function when clicked.
// It also sets the tooltip of the widget appropriately. If allowReadOnly is false,
// the dialog will not be opened if the widget is read only. It also takes an optional
// function to call after the dialog is accepted.
func InitValueButton(v Value, allowReadOnly bool, make func(d *Body), after ...func()) {
	wb := v.AsWidget()
	// windows are never new on mobile
	if !TheApp.Platform().IsMobile() {
		wb.SetTooltip(shiftNewWindow)
	}
	v.OnClick(func(e events.Event) {
		if allowReadOnly || !wb.IsReadOnly() {
			v.SetFlag(e.HasAnyModifier(key.Shift), ValueDialogNewWindow)
			OpenValueDialog(v, v.AsWidget(), make, after...)
		}
	})
}

// OpenValueDialog opens a new value dialog for the given [Value] using the
// given context widget, the given function for constructing the dialog, and the
// optional given function to call after the dialog is accepted.
func OpenValueDialog(v Value, ctx Widget, make func(d *Body), after ...func()) {
	opv := reflectx.UnderlyingPointer(reflect.ValueOf(v.WidgetValue()))
	if !opv.IsValid() {
		return
	}
	obj := opv.Interface()
	if RecycleDialog(obj) {
		return
	}
	wb := v.AsWidget()
	d := NewBody().AddTitle(wb.ValueContext)
	if text := strings.TrimPrefix(wb.Tooltip, shiftNewWindow); text != "" {
		d.AddText(text)
	}
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
		d.AddBottomBar(func(parent Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).OnClick(func(e events.Event) {
				after[0]()
				wb.SendChange()
				v.Update()
			})
		})
	}

	if v.Is(ValueDialogNewWindow) {
		d.RunWindowDialog(ctx)
	} else {
		d.RunFullDialog(ctx)
	}
}
