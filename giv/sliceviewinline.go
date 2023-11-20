// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"
	"strconv"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// SliceViewInline represents a slice as a single line widget,
// for smaller slices and those explicitly marked inline.
type SliceViewInline struct {
	gi.Layout

	// the slice that we are a view onto
	Slice any `set:"-"`

	// Value for the slice itself, if this was created within value view framework -- otherwise nil
	SliceValView Value

	// whether the slice is actually an array -- no modifications
	IsArray bool

	// whether the slice has a fixed-len flag on it
	IsFixedLen bool

	// has the slice been edited?
	Changed bool `set:"-"`

	// Value representations of the fields
	Values []Value `json:"-" xml:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `view:"-" json:"-" xml:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string
}

func (sv *SliceViewInline) OnInit() {
	sv.SliceViewInlineStyles()
}

func (sv *SliceViewInline) SliceViewInlineStyles() {
	sv.Style(func(s *styles.Style) {
		s.Align.Y = styles.AlignCenter
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
				if sv.SliceValView != nil {
					newPath := ""
					isZero := false
					title, newPath, isZero = sv.SliceValView.AsValueBase().GetTitle()
					if isZero {
						return
					}
					vpath = sv.ViewPath + "/" + newPath
				} else {
					elType := laser.NonPtrType(reflect.TypeOf(sv.Slice).Elem().Elem())
					title = "Slice of " + laser.NonPtrType(elType).Name()
				}
				d := gi.NewBody(sv).AddTitle(title).FullWindow(true)
				NewSliceView(d).SetViewPath(vpath).SetSlice(sv.Slice).SetTmpSave(sv.TmpSave)
				d.OnAccept(func(e events.Event) {
					if sv.SliceValView != nil { // todo: this is not updating
						sv.SliceValView.UpdateWidget()
					}
				}).Run()
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
		newslc = (sv.Slice != sl)
	}
	if newslc {
		sv.Slice = sl
		sv.IsArray = laser.NonPtrType(reflect.TypeOf(sl)).Kind() == reflect.Array
		sv.IsFixedLen = false
		if sv.SliceValView != nil {
			_, sv.IsFixedLen = sv.SliceValView.Tag("fixed-len")
		}
		sv.Update()
	}
	return sv
}

func (sv *SliceViewInline) ConfigWidget(sc *gi.Scene) {
	sv.ConfigSlice(sc)
}

// ConfigSlice configures children for slice view
func (sv *SliceViewInline) ConfigSlice(sc *gi.Scene) bool {
	if laser.AnyIsNil(sv.Slice) {
		return false
	}
	config := ki.Config{}
	// always start fresh!
	sv.Values = make([]Value, 0)

	mv := reflect.ValueOf(sv.Slice)
	mvnp := laser.NonPtrValue(mv)

	sz := min(mvnp.Len(), SliceInlineLen)
	for i := 0; i < sz; i++ {
		val := laser.OnePtrUnderlyingValue(mvnp.Index(i)) // deal with pointer lists
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
	mods, updt := sv.ConfigChildren(config)
	if !mods {
		updt = sv.UpdateStart()
	}
	for i, vv := range sv.Values {
		vvb := vv.AsValueBase()
		vvb.OnChange(func(e events.Event) { sv.SetChanged() })
		w := sv.Child(i).(gi.Widget)
		if sv.SliceValView != nil {
			vv.SetTags(sv.SliceValView.AllTags())
		}
		vv.ConfigWidget(w, sc)
		if sv.IsReadOnly() {
			w.AsWidget().SetReadOnly(true)
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
		edbt.Tooltip = "edit slice in a dialog window"
	}
	sv.UpdateEndLayout(updt)
	return updt
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
	updt := sv.UpdateStart()
	defer sv.UpdateEndLayout(updt)

	laser.SliceNewAt(sv.Slice, idx)

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.SetChanged()
	sv.Update()
}

func (sv *SliceViewInline) UpdateValues() {
	updt := sv.UpdateStart()
	for _, vv := range sv.Values {
		vv.UpdateWidget()
	}
	sv.UpdateEndRender(updt)
}
