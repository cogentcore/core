// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"reflect"
	"strconv"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
)

// SliceViewInline represents a slice within a single line of value widgets.
// This is typically used for smaller slices.
type SliceViewInline struct {
	core.Frame

	// Slice is the slice that we are viewing.
	Slice any `set:"-"`

	// SliceValue is the Value for the slice itself
	// if this was created within the Value framework.
	// Otherwise, it is nil.
	SliceValue Value `set:"-"`

	// ViewPath is a record of parent view names that have led up to this view.
	// It is displayed as extra contextual information in view dialogs.
	ViewPath string

	// isArray is whether the slice is actually an array.
	isArray bool

	// isFixedLength is whether the slice has a fixed-length flag on it.
	isFixedLength bool

	// size when configured
	configSize int
}

func (sv *SliceViewInline) WidgetValue() any { return &sv.Slice }

func (sv *SliceViewInline) OnInit() {
	sv.Frame.OnInit()
	sv.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
	})
}

// SetSlice sets the source slice that we are viewing -- rebuilds the children to represent this slice
func (sv *SliceViewInline) SetSlice(sl any) *SliceViewInline {
	if reflectx.AnyIsNil(sl) {
		sv.Slice = nil
		return sv
	}
	newslc := false
	if reflect.TypeOf(sl).Kind() != reflect.Pointer { // prevent crash on non-comparable
		newslc = true
	} else {
		newslc = sv.Slice != sl
	}
	if newslc {
		sv.Slice = sl
		sv.isArray = reflectx.NonPointerType(reflect.TypeOf(sl)).Kind() == reflect.Array
		sv.isFixedLength = false
		if sv.SliceValue != nil {
			_, sv.isFixedLength = sv.SliceValue.Tag("fixed-len")
		}
		sv.Update()
	}
	return sv
}

func (sv *SliceViewInline) Config(c *core.Config) {
	sl := reflectx.NonPointerValue(reflectx.OnePointerUnderlyingValue(reflect.ValueOf(sv.Slice)))

	sz := min(sl.Len(), core.SystemSettings.SliceInlineLength)
	sv.configSize = sz
	for i := 0; i < sz; i++ {
		itxt := strconv.Itoa(i)
		val := reflectx.OnePointerUnderlyingValue(sl.Index(i)) // deal with pointer lists
		core.Configure(c, "value-"+itxt, func() core.Value {
			w := core.NewValue(val.Interface())
			wb := w.AsWidget()
			// vv.SetSliceValue(val, sv.Slice, i, sv.ViewPath)
			wb.OnChange(func(e events.Event) { sv.SendChange() })
			// if sv.SliceValue != nil {
			// 	vv.SetTags(sv.SliceValue.AllTags())
			// }
			wb.OnInput(func(e events.Event) {
				// if tag, _ := vv.Tag("immediate"); tag == "+" {
				// 	wb.SendChange(e)
				// 	sv.SendChange(e)
				// }
				sv.Send(events.Input, e)
			})
			if sv.IsReadOnly() {
				wb.SetReadOnly(true)
			} else {
				wb.AddContextMenu(func(m *core.Scene) {
					sv.ContextMenu(m, i)
				})
			}
			return w
		}, func(w core.Value) {
			wb := w.AsWidget()
			core.Bind(val.Interface(), w)
			// vv.SetSliceValue(val, sv.Slice, i, sv.ViewPath)
			wb.SetReadOnly(sv.IsReadOnly())
		})
	}
	if !sv.isArray && !sv.isFixedLength {
		core.Configure(c, "add-button", func() *core.Button {
			w := core.NewButton().SetIcon(icons.Add).SetType(core.ButtonTonal)
			w.Tooltip = "Add an element to the list"
			w.OnClick(func(e events.Event) {
				sv.SliceNewAt(-1)
			})
			return w
		})
	}
	core.Configure(c, "edit-button", func() *core.Button {
		w := core.NewButton().SetIcon(icons.Edit).SetType(core.ButtonTonal)
		w.Tooltip = "Edit list in a dialog"
		w.OnClick(func(e events.Event) {
			vpath := sv.ViewPath
			title := ""
			if sv.SliceValue != nil {
				newPath := ""
				isZero := false
				title, newPath, isZero = sv.SliceValue.AsValueData().GetTitle()
				if isZero {
					return
				}
				vpath = JoinViewPath(sv.ViewPath, newPath)
			} else {
				title = reflectx.FriendlySliceLabel(reflect.ValueOf(sv.Slice))
			}
			d := core.NewBody().AddTitle(title)
			NewSliceView(d).SetViewPath(vpath).SetSlice(sv.Slice)
			d.OnClose(func(e events.Event) {
				sv.Update()
				sv.SendChange()
			})
			d.RunFullDialog(sv)
		})
		return w
	})
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (sv *SliceViewInline) SliceNewAt(idx int) {
	if sv.isArray || sv.isFixedLength {
		return
	}
	reflectx.SliceNewAt(sv.Slice, idx)

	sv.SendChange()
	sv.Update()
}

// SliceDeleteAt deletes element at given index from slice
func (sv *SliceViewInline) SliceDeleteAt(idx int) {
	if sv.isArray || sv.isFixedLength {
		return
	}
	reflectx.SliceDeleteAt(sv.Slice, idx)

	sv.SendChange()
	sv.Update()
}

func (sv *SliceViewInline) ContextMenu(m *core.Scene, idx int) {
	if sv.IsReadOnly() || sv.isArray || sv.isFixedLength {
		return
	}
	core.NewButton(m).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
		sv.SliceNewAt(idx)
	})
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		sv.SliceDeleteAt(idx)
	})
}

func (sv *SliceViewInline) SliceSizeChanged() bool {
	if reflectx.AnyIsNil(sv.Slice) {
		return sv.configSize != 0
	}
	sl := reflectx.NonPointerValue(reflectx.OnePointerUnderlyingValue(reflect.ValueOf(sv.Slice)))
	return sv.configSize != sl.Len()
}

func (sv *SliceViewInline) SizeUp() {
	if sv.SliceSizeChanged() {
		sv.Update()
	}
	sv.Frame.SizeUp()
}
