// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"reflect"
	"strconv"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/reflectx"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// SliceViewInline represents a slice as a single line widget,
// for smaller slices and those explicitly marked inline.
type SliceViewInline struct {
	core.Layout

	// the slice that we are a view onto
	Slice any `set:"-"`

	// SliceValue is the Value for the slice itself
	// if this was created within the Value framework.
	// Otherwise, it is nil.
	SliceValue Value `set:"-"`

	// whether the slice is actually an array -- no modifications
	IsArray bool `set:"-"`

	// whether the slice has a fixed-len flag on it
	IsFixedLen bool `set:"-"`

	// has the slice been edited?
	Changed bool `set:"-"`

	// Value representations of the fields
	Values []Value `json:"-" xml:"-" set:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// size of map when gui was configured
	ConfigSize int `set:"-"`
}

func (sv *SliceViewInline) OnInit() {
	sv.Layout.OnInit()
	sv.SetStyles()
}

func (sv *SliceViewInline) SetStyles() {
	sv.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
	})
	sv.OnWidgetAdded(func(w core.Widget) {
		switch w.PathFrom(sv) {
		case "add-button":
			ab := w.(*core.Button)
			w.Style(func(s *styles.Style) {
				ab.SetType(core.ButtonTonal)
			})
			ab.OnClick(func(e events.Event) {
				sv.SliceNewAt(-1)
			})
		case "edit-button":
			w.Style(func(s *styles.Style) {
				w.(*core.Button).SetType(core.ButtonTonal)
			})
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
					elType := reflectx.NonPtrType(reflect.TypeOf(sv.Slice).Elem().Elem())
					title = "Slice of " + reflectx.NonPtrType(elType).Name()
				}
				d := core.NewBody().AddTitle(title)
				NewSliceView(d).SetViewPath(vpath).SetSlice(sv.Slice)
				d.OnClose(func(e events.Event) {
					sv.Update()
					sv.SendChange()
				})
				d.NewFullDialog(sv).Run()
			})
		}
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
		sv.IsArray = reflectx.NonPtrType(reflect.TypeOf(sl)).Kind() == reflect.Array
		sv.IsFixedLen = false
		if sv.SliceValue != nil {
			_, sv.IsFixedLen = sv.SliceValue.Tag("fixed-len")
		}
		sv.Update()
	}
	return sv
}

func (sv *SliceViewInline) Config() {
	sv.DeleteChildren()
	if reflectx.AnyIsNil(sv.Slice) {
		sv.ConfigSize = 0
		return
	}
	config := tree.Config{}
	// always start fresh!
	sv.Values = make([]Value, 0)

	sl := reflectx.NonPtrValue(reflectx.OnePtrUnderlyingValue(reflect.ValueOf(sv.Slice)))
	sv.ConfigSize = sl.Len()

	sz := min(sl.Len(), core.SystemSettings.SliceInlineLength)
	for i := 0; i < sz; i++ {
		val := reflectx.OnePtrUnderlyingValue(sl.Index(i)) // deal with pointer lists
		vv := ToValue(val.Interface(), "")
		vv.SetSliceValue(val, sv.Slice, i, sv.ViewPath)
		vtyp := vv.WidgetType()
		idxtxt := strconv.Itoa(i)
		valnm := "value-" + idxtxt
		config.Add(vtyp, valnm)
		sv.Values = append(sv.Values, vv)
	}
	if !sv.IsArray && !sv.IsFixedLen {
		config.Add(core.ButtonType, "add-button")
	}
	config.Add(core.ButtonType, "edit-button")
	sv.ConfigChildren(config)
	for i, vv := range sv.Values {
		vv.OnChange(func(e events.Event) { sv.SetChanged() })
		w := sv.Child(i).(core.Widget)
		if sv.SliceValue != nil {
			vv.SetTags(sv.SliceValue.AllTags())
		}
		Config(vv, w)
		wb := w.AsWidget()
		wb.OnInput(func(e events.Event) {
			if tag, _ := vv.Tag("immediate"); tag == "+" {
				wb.SendChange(e)
				sv.SendChange(e)
			}
			sv.Send(events.Input, e)
		})
		if sv.IsReadOnly() {
			wb.SetReadOnly(true)
		} else {
			wb.AddContextMenu(func(m *core.Scene) {
				sv.ContextMenu(m, i)
			})
		}
	}
	if !sv.IsArray && !sv.IsFixedLen {
		adbti, err := sv.Children().ElemFromEndTry(1)
		if err == nil {
			adbt := adbti.(*core.Button)
			adbt.SetType(core.ButtonTonal)
			adbt.SetIcon(icons.Add)
			adbt.Tooltip = "add an element to the slice"
		}
	}
	edbti, err := sv.Children().ElemFromEndTry(0)
	if err == nil {
		edbt := edbti.(*core.Button)
		edbt.SetType(core.ButtonTonal)
		edbt.SetIcon(icons.Edit)
		edbt.Tooltip = "edit in a dialog"
	}
	sv.NeedsLayout()
}

// SetChanged sets the Changed flag and emits the ViewSig signal for the
// SliceView, indicating that some kind of edit / change has taken place to
// the table data.  It isn't really practical to record all the different
// types of changes, so this is just generic.
func (sv *SliceViewInline) SetChanged() {
	sv.Changed = true
	sv.SendChange()
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (sv *SliceViewInline) SliceNewAt(idx int) {
	if sv.IsArray || sv.IsFixedLen {
		return
	}
	reflectx.SliceNewAt(sv.Slice, idx)

	sv.SetChanged()
	sv.Update()
}

// SliceDeleteAt deletes element at given index from slice
func (sv *SliceViewInline) SliceDeleteAt(idx int) {
	if sv.IsArray || sv.IsFixedLen {
		return
	}
	reflectx.SliceDeleteAt(sv.Slice, idx)

	sv.SetChanged()
	sv.Update()
}

func (sv *SliceViewInline) ContextMenu(m *core.Scene, idx int) {
	if sv.IsReadOnly() || sv.IsArray || sv.IsFixedLen {
		return
	}
	core.NewButton(m).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
		sv.SliceNewAt(idx)
	})
	core.NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		sv.SliceDeleteAt(idx)
	})
}

func (sv *SliceViewInline) UpdateValues() {
	for _, vv := range sv.Values {
		vv.Update()
	}
	sv.NeedsRender()
}

func (sv *SliceViewInline) SliceSizeChanged() bool {
	if reflectx.AnyIsNil(sv.Slice) {
		return sv.ConfigSize != 0
	}
	sl := reflectx.NonPtrValue(reflectx.OnePtrUnderlyingValue(reflect.ValueOf(sv.Slice)))
	return sv.ConfigSize != sl.Len()
}

func (sv *SliceViewInline) SizeUp() {
	if sv.SliceSizeChanged() {
		sv.Update()
	}
	sv.Layout.SizeUp()
}
