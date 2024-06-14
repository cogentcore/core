// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"reflect"
	"strconv"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
)

// InlineList represents a slice within a single line of value widgets.
// This is typically used for smaller slices.
type InlineList struct {
	Frame

	// Slice is the slice that we are viewing.
	Slice any `set:"-"`

	// isArray is whether the slice is actually an array.
	isArray bool
}

func (il *InlineList) WidgetValue() any { return &il.Slice }

func (il *InlineList) Init() {
	il.Frame.Init()
	il.Maker(func(p *Plan) {
		sl := reflectx.NonPointerValue(reflectx.UnderlyingPointer(reflect.ValueOf(il.Slice)))

		sz := min(sl.Len(), SystemSettings.SliceInlineLength)
		for i := 0; i < sz; i++ {
			itxt := strconv.Itoa(i)
			val := reflectx.UnderlyingPointer(sl.Index(i)) // deal with pointer lists
			AddNew(p, "value-"+itxt, func() Value {
				return NewValue(val.Interface(), "")
			}, func(w Value) {
				wb := w.AsWidget()
				wb.OnChange(func(e events.Event) { il.SendChange() })
				wb.OnInput(func(e events.Event) {
					il.Send(events.Input, e)
				})
				if il.IsReadOnly() {
					wb.SetReadOnly(true)
				} else {
					wb.AddContextMenu(func(m *Scene) {
						il.ContextMenu(m, i)
					})
				}
				wb.Updater(func() {
					Bind(val.Interface(), w)
					wb.SetReadOnly(il.IsReadOnly())
				})
			})
		}
		if !il.isArray {
			AddAt(p, "add-button", func(w *Button) {
				w.SetIcon(icons.Add).SetType(ButtonTonal)
				w.Tooltip = "Add an element to the list"
				w.OnClick(func(e events.Event) {
					il.SliceNewAt(-1)
				})
			})
		}
		AddAt(p, "edit-button", func(w *Button) {
			w.SetIcon(icons.Edit).SetType(ButtonTonal)
			w.Tooltip = "Edit list in a dialog"
			w.OnClick(func(e events.Event) {
				d := NewBody().AddTitle(il.ValueTitle).AddText(il.Tooltip)
				NewList(d).SetSlice(il.Slice).SetValueTitle(il.ValueTitle)
				d.OnClose(func(e events.Event) {
					il.Update()
					il.SendChange()
				})
				d.RunFullDialog(il)
			})
		})
	})
}

// SetSlice sets the source slice that we are viewing -- rebuilds the children to represent this slice
func (il *InlineList) SetSlice(sl any) *InlineList {
	if reflectx.AnyIsNil(sl) {
		il.Slice = nil
		return il
	}
	newslc := false
	if reflect.TypeOf(sl).Kind() != reflect.Pointer { // prevent crash on non-comparable
		newslc = true
	} else {
		newslc = il.Slice != sl
	}
	if newslc {
		il.Slice = sl
		il.isArray = reflectx.NonPointerType(reflect.TypeOf(sl)).Kind() == reflect.Array
		il.Update()
	}
	return il
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (il *InlineList) SliceNewAt(idx int) {
	if il.isArray {
		return
	}
	reflectx.SliceNewAt(il.Slice, idx)

	il.SendChange()
	il.Update()
}

// SliceDeleteAt deletes element at given index from slice
func (il *InlineList) SliceDeleteAt(idx int) {
	if il.isArray {
		return
	}
	reflectx.SliceDeleteAt(il.Slice, idx)

	il.SendChange()
	il.Update()
}

func (il *InlineList) ContextMenu(m *Scene, idx int) {
	if il.IsReadOnly() || il.isArray {
		return
	}
	NewButton(m).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
		il.SliceNewAt(idx)
	})
	NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		il.SliceDeleteAt(idx)
	})
}
