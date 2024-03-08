// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"
	"strconv"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
)

// SliceViewInline represents a slice as a single line widget,
// for smaller slices and those explicitly marked inline.
type SliceViewInline struct {
	gi.Layout

	// the slice that we are a view onto
	Slice any `set:"-"`

	// SliceValue is the Value for the slice itself
	// if this was created within the Value framework.
	// Otherwise, it is nil.
	SliceValue Value

	// whether the slice is actually an array -- no modifications
	IsArray bool `set:"-"`

	// whether the slice has a fixed-len flag on it
	IsFixedLen bool

	// has the slice been edited?
	Changed bool `set:"-"`

	// Value representations of the fields
	Values []Value `json:"-" xml:"-" set:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `view:"-" json:"-" xml:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// size of map when gui configed
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
	sv.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(sv) {
		case "add-action":
			ab := w.(*gi.Button)
			w.Style(func(s *styles.Style) {
				ab.SetType(gi.ButtonTonal)
			})
			ab.OnClick(func(e events.Event) {
				sv.SliceNewAt(-1)
			})
		case "edit-action":
			w.Style(func(s *styles.Style) {
				w.(*gi.Button).SetType(gi.ButtonTonal)
			})
			w.OnClick(func(e events.Event) {
				vpath := sv.ViewPath
				title := ""
				if sv.SliceValue != nil {
					newPath := ""
					isZero := false
					title, newPath, isZero = sv.SliceValue.AsValueBase().GetTitle()
					if isZero {
						return
					}
					vpath = JoinViewPath(sv.ViewPath, newPath)
				} else {
					elType := laser.NonPtrType(reflect.TypeOf(sv.Slice).Elem().Elem())
					title = "Slice of " + laser.NonPtrType(elType).Name()
				}
				d := gi.NewBody().AddTitle(title)
				NewSliceView(d).SetViewPath(vpath).SetSlice(sv.Slice).SetTmpSave(sv.TmpSave)
				d.AddBottomBar(func(pw gi.Widget) {
					d.AddCancel(pw)
					d.AddOk(pw).OnClick(func(e events.Event) {
						sv.SendChange()
					})
				})
				d.NewFullDialog(sv).Run()
			})
		}
	})
}

// SetSlice sets the source slice that we are viewing -- rebuilds the children to represent this slice
func (sv *SliceViewInline) SetSlice(sl any) *SliceViewInline {
	if laser.AnyIsNil(sl) {
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
		sv.IsArray = laser.NonPtrType(reflect.TypeOf(sl)).Kind() == reflect.Array
		sv.IsFixedLen = false
		if sv.SliceValue != nil {
			_, sv.IsFixedLen = sv.SliceValue.Tag("fixed-len")
		}
		sv.Update()
	}
	return sv
}

func (sv *SliceViewInline) ConfigWidget() {
	sv.DeleteChildren()
	if laser.AnyIsNil(sv.Slice) {
		sv.ConfigSize = 0
		return
	}
	config := ki.Config{}
	// always start fresh!
	sv.Values = make([]Value, 0)

	sl := laser.NonPtrValue(laser.OnePtrUnderlyingValue(reflect.ValueOf(sv.Slice)))
	sv.ConfigSize = sl.Len()

	sz := min(sl.Len(), gi.SystemSettings.SliceInlineLength)
	for i := 0; i < sz; i++ {
		val := laser.OnePtrUnderlyingValue(sl.Index(i)) // deal with pointer lists
		vv := ToValue(val.Interface(), "")
		if vv == nil { // shouldn't happen
			fmt.Printf("nil value view!\n")
			continue
		}
		vv.SetSliceValue(val, sv.Slice, i, sv.TmpSave, sv.ViewPath)
		vtyp := vv.WidgetType()
		idxtxt := strconv.Itoa(i)
		valnm := "value-" + idxtxt
		config.Add(vtyp, valnm)
		sv.Values = append(sv.Values, vv)
	}
	if !sv.IsArray && !sv.IsFixedLen {
		config.Add(gi.ButtonType, "add-action")
	}
	config.Add(gi.ButtonType, "edit-action")
	sv.ConfigChildren(config)
	for i, vv := range sv.Values {
		vvb := vv.AsValueBase()
		vvb.OnChange(func(e events.Event) { sv.SetChanged() })
		w := sv.Child(i).(gi.Widget)
		if sv.SliceValue != nil {
			vv.SetTags(sv.SliceValue.AllTags())
		}
		vv.ConfigWidget(w)
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
			wb.AddContextMenu(func(m *gi.Scene) {
				sv.ContextMenu(m, i)
			})
		}
	}
	if !sv.IsArray && !sv.IsFixedLen {
		adbti, err := sv.Children().ElemFromEndTry(1)
		if err == nil {
			adbt := adbti.(*gi.Button)
			adbt.SetType(gi.ButtonTonal)
			adbt.SetIcon(icons.Add)
			adbt.Tooltip = "add an element to the slice"
		}
	}
	edbti, err := sv.Children().ElemFromEndTry(0)
	if err == nil {
		edbt := edbti.(*gi.Button)
		edbt.SetType(gi.ButtonTonal)
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
	laser.SliceNewAt(sv.Slice, idx)

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.SetChanged()
	sv.Update()
}

// SliceDeleteAt deletes element at given index from slice
func (sv *SliceViewInline) SliceDeleteAt(idx int) {
	if sv.IsArray || sv.IsFixedLen {
		return
	}
	laser.SliceDeleteAt(sv.Slice, idx)

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.SetChanged()
	sv.Update()
}

func (sv *SliceViewInline) ContextMenu(m *gi.Scene, idx int) {
	if sv.IsReadOnly() || sv.IsArray || sv.IsFixedLen {
		return
	}
	gi.NewButton(m).SetText("Add").SetIcon(icons.Add).OnClick(func(e events.Event) {
		sv.SliceNewAt(idx)
	})
	gi.NewButton(m).SetText("Delete").SetIcon(icons.Delete).OnClick(func(e events.Event) {
		sv.SliceDeleteAt(idx)
	})
}

func (sv *SliceViewInline) UpdateValues() {
	for _, vv := range sv.Values {
		vv.UpdateWidget()
	}
	sv.NeedsRender()
}

func (sv *SliceViewInline) SliceSizeChanged() bool {
	if laser.AnyIsNil(sv.Slice) {
		return sv.ConfigSize != 0
	}
	sl := laser.NonPtrValue(laser.OnePtrUnderlyingValue(reflect.ValueOf(sv.Slice)))
	return sv.ConfigSize != sl.Len()
}

func (sv *SliceViewInline) SizeUp() {
	if sv.SliceSizeChanged() {
		sv.Update()
	}
	sv.Layout.SizeUp()
}
