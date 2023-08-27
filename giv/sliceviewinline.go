// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"goki.dev/gi/gi"
	"goki.dev/gi/gist"
	"goki.dev/gi/icons"
)

// SliceViewInline represents a slice as a single line widget, for smaller
// slices and those explicitly marked inline -- constructs widgets in Parts to
// show the key names and editor vals for each value.
type SliceViewInline struct {
	gi.PartsWidgetBase

	// the slice that we are a view onto
	Slice any `desc:"the slice that we are a view onto"`

	// ValueView for the slice itself, if this was created within value view framework -- otherwise nil
	SliceValView ValueView `desc:"ValueView for the slice itself, if this was created within value view framework -- otherwise nil"`

	// whether the slice is actually an array -- no modifications
	IsArray bool `desc:"whether the slice is actually an array -- no modifications"`

	// whether the slice has a fixed-len flag on it
	IsFixedLen bool `desc:"whether the slice has a fixed-len flag on it"`

	// has the slice been edited?
	Changed bool `desc:"has the slice been edited?"`

	// ValueView representations of the fields
	Values []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave ValueView `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`

	// signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update
	ViewSig ki.Signal `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`
}

var TypeSliceViewInline = kit.Types.AddType(&SliceViewInline{}, SliceViewInlineProps)

func (sv *SliceViewInline) OnInit() {
	sv.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		s.MinWidth.SetCh(20)
	})
}

func (sv *SliceViewInline) OnChildAdded(child ki.Ki) {
	if w := gi.KiAsWidget(child); w != nil {
		switch w.Name() {
		case "Parts":
			parts := child.(*gi.Layout)
			parts.Lay = gi.LayoutHoriz
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.Overflow = gist.OverflowHidden // no scrollbars!
			})
		}
	}
}

func (sv *SliceViewInline) Disconnect() {
	sv.PartsWidgetBase.Disconnect()
	sv.ViewSig.DisconnectAll()
}

// SetSlice sets the source slice that we are viewing -- rebuilds the children to represent this slice
func (sv *SliceViewInline) SetSlice(sl any) {
	if kit.IfaceIsNil(sl) {
		sv.Slice = nil
		return
	}
	updt := false
	newslc := false
	if reflect.TypeOf(sl).Kind() != reflect.Pointer { // prevent crash on non-comparable
		newslc = true
	} else {
		newslc = (sv.Slice != sl)
	}
	if newslc {
		updt = sv.UpdateStart()
		sv.Slice = sl
		sv.IsArray = kit.NonPtrType(reflect.TypeOf(sl)).Kind() == reflect.Array
		sv.IsFixedLen = false
		if sv.SliceValView != nil {
			_, sv.IsFixedLen = sv.SliceValView.Tag("fixed-len")
		}
		sv.SetFullReRender()
	}
	sv.UpdateFromSlice()
	sv.UpdateEnd(updt)
}

var SliceViewInlineProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
}

// ConfigParts configures Parts for the current slice
func (sv *SliceViewInline) ConfigParts() {
	if kit.IfaceIsNil(sv.Slice) {
		return
	}
	config := kit.TypeAndNameList{}
	// always start fresh!
	sv.Values = make([]ValueView, 0)

	mv := reflect.ValueOf(sv.Slice)
	mvnp := kit.NonPtrValue(mv)

	sz := ints.MinInt(mvnp.Len(), SliceInlineLen)
	for i := 0; i < sz; i++ {
		val := kit.OnePtrUnderlyingValue(mvnp.Index(i)) // deal with pointer lists
		vv := ToValueView(val.Interface(), "")
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
		config.Add(gi.TypeAction, "add-action")
	}
	config.Add(gi.TypeAction, "edit-action")
	mods, updt := sv.Parts.ConfigChildren(config)
	if !mods {
		updt = sv.Parts.UpdateStart()
	}
	for i, vv := range sv.Values {
		vvb := vv.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			svv, _ := recv.Embed(TypeSliceViewInline).(*SliceViewInline)
			svv.SetChanged()
		})
		widg := sv.Parts.Child(i).(gi.Node2D)
		if sv.SliceValView != nil {
			vv.SetTags(sv.SliceValView.AllTags())
		}
		vv.ConfigWidget(widg)
		if sv.IsDisabled() {
			widg.AsNode2D().SetDisabled()
		}
	}
	if !sv.IsArray && !sv.IsFixedLen {
		adack, err := sv.Parts.Children().ElemFromEndTry(1)
		if err == nil {
			adac := adack.(*gi.Action)
			adac.SetIcon(icons.Add)
			adac.Tooltip = "add an element to the slice"
			adac.ActionSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
				svv, _ := recv.Embed(TypeSliceViewInline).(*SliceViewInline)
				svv.SliceNewAt(-1, true)
			})
		}
	}
	edack, err := sv.Parts.Children().ElemFromEndTry(0)
	if err == nil {
		edac := edack.(*gi.Action)
		edac.SetIcon(icons.Edit)
		edac.Tooltip = "edit slice in a dialog window"
		edac.ActionSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
			svv, _ := recv.Embed(TypeSliceViewInline).(*SliceViewInline)
			vpath := svv.ViewPath
			title := ""
			if svv.SliceValView != nil {
				newPath := ""
				isZero := false
				title, newPath, isZero = svv.SliceValView.AsValueViewBase().Label()
				if isZero {
					return
				}
				vpath = svv.ViewPath + "/" + newPath
			} else {
				elType := kit.NonPtrType(reflect.TypeOf(svv.Slice).Elem().Elem())
				title = "Slice of " + kit.NonPtrType(elType).Name()
			}
			dlg := SliceViewDialog(svv.Viewport, svv.Slice, DlgOpts{Title: title, TmpSave: svv.TmpSave, ViewPath: vpath}, nil, nil, nil)
			svvvk := dlg.Frame().ChildByType(TypeSliceView, ki.Embeds, 2)
			if svvvk != nil {
				svvv := svvvk.(*SliceView)
				svvv.SliceValView = svv.SliceValView
				svvv.ViewSig.ConnectOnly(svv.This(), func(recv, send ki.Ki, sig int64, data any) {
					svvvv, _ := recv.Embed(TypeSliceViewInline).(*SliceViewInline)
					svvvv.ViewSig.Emit(svvvv.This(), 0, nil)
				})
			}
		})
	}
	sv.Parts.UpdateEnd(updt)
}

// SetChanged sets the Changed flag and emits the ViewSig signal for the
// SliceView, indicating that some kind of edit / change has taken place to
// the table data.  It isn't really practical to record all the different
// types of changes, so this is just generic.
func (sv *SliceViewInline) SetChanged() {
	sv.Changed = true
	sv.ViewSig.Emit(sv.This(), 0, nil)
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (sv *SliceViewInline) SliceNewAt(idx int, reconfig bool) {
	if sv.IsArray || sv.IsFixedLen {
		return
	}

	updt := sv.UpdateStart()
	defer sv.UpdateEnd(updt)

	kit.SliceNewAt(sv.Slice, idx)

	if sv.TmpSave != nil {
		sv.TmpSave.SaveTmp()
	}
	sv.SetChanged()
	if reconfig {
		sv.SetFullReRender()
		sv.UpdateFromSlice()
	}
}

func (sv *SliceViewInline) UpdateFromSlice() {
	sv.ConfigParts()
}

func (sv *SliceViewInline) UpdateValues() {
	updt := sv.UpdateStart()
	for _, vv := range sv.Values {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
}

func (sv *SliceViewInline) Style2D() {
	sv.ConfigParts()
	sv.PartsWidgetBase.Style2D()
}

func (sv *SliceViewInline) Render2D() {
	if sv.FullReRenderIfNeeded() {
		return
	}
	if sv.PushBounds() {
		sv.ConfigParts()
		sv.Render2DParts()
		sv.Render2DChildren()
		sv.PopBounds()
	}
}
