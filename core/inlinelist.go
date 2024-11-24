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
	"cogentcore.org/core/tree"
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
	il.Maker(func(p *tree.Plan) {
		sl := reflectx.Underlying(reflect.ValueOf(il.Slice))

		sz := min(sl.Len(), SystemSettings.SliceInlineLength)
		for i := 0; i < sz; i++ {
			itxt := strconv.Itoa(i)
			tree.AddNew(p, "value-"+itxt, func() Value {
				val := reflectx.UnderlyingPointer(sl.Index(i))
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
						il.contextMenu(m, i)
					})
				}
				wb.Updater(func() {
					// We need to get the current value each time:
					sl := reflectx.Underlying(reflect.ValueOf(il.Slice))
					val := reflectx.UnderlyingPointer(sl.Index(i))
					Bind(val.Interface(), w)
					wb.SetReadOnly(il.IsReadOnly())
				})
			})
		}
		if !il.isArray && !il.IsReadOnly() {
			tree.AddAt(p, "add-button", func(w *Button) {
				w.SetIcon(icons.Add).SetType(ButtonTonal)
				w.Tooltip = "Add an element to the list"
				w.OnClick(func(e events.Event) {
					il.NewAt(-1)
				})
			})
		}
	})
}

// SetSlice sets the source slice that we are viewing.
// It rebuilds the children to represent this slice.
func (il *InlineList) SetSlice(sl any) *InlineList {
	if reflectx.IsNil(reflect.ValueOf(sl)) {
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

// NewAt inserts a new blank element at the given index in the slice.
// -1 indicates to insert the element at the end.
func (il *InlineList) NewAt(idx int) {
	if il.isArray {
		return
	}
	reflectx.SliceNewAt(il.Slice, idx)
	il.UpdateChange()
}

// DeleteAt deletes the element at the given index from the slice.
func (il *InlineList) DeleteAt(idx int) {
	if il.isArray {
		return
	}
	reflectx.SliceDeleteAt(il.Slice, idx)
	il.UpdateChange()
}

func (il *InlineList) contextMenu(m *Scene, idx int) {
	if il.IsReadOnly() || il.isArray {
		return
	}
	NewButton(m).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
		il.NewAt(idx)
	})
	NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		il.DeleteAt(idx)
	})
	NewButton(m).SetText("Open in dialog").SetIcon(icons.OpenInNew).OnClick(func(e events.Event) {
		d := NewBody(il.ValueTitle)
		NewText(d).SetType(TextSupporting).SetText(il.Tooltip)
		NewList(d).SetSlice(il.Slice).SetValueTitle(il.ValueTitle).SetReadOnly(il.IsReadOnly())
		d.OnClose(func(e events.Event) {
			il.UpdateChange()
		})
		d.RunFullDialog(il)
	})
}
