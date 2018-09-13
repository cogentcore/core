// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// SliceViewInline represents a slice as a single line widget, for smaller
// slices and those explicitly marked inline -- constructs widgets in Parts to
// show the key names and editor vals for each value.
type SliceViewInline struct {
	gi.PartsWidgetBase
	Slice   interface{} `desc:"the slice that we are a view onto"`
	IsArray bool        `desc:"whether the slice is actually an array -- no modifications"`
	Changed bool        `desc:"has the slice been edited?"`
	Values  []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig ki.Signal   `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
}

var KiT_SliceViewInline = kit.Types.AddType(&SliceViewInline{}, SliceViewInlineProps)

// SetSlice sets the source slice that we are viewing -- rebuilds the children to represent this slice
func (sv *SliceViewInline) SetSlice(sl interface{}, tmpSave ValueView) {
	updt := false
	if sv.Slice != sl {
		updt = sv.UpdateStart()
		sv.Slice = sl
		sv.IsArray = kit.NonPtrType(reflect.TypeOf(sl)).Kind() == reflect.Array
	}
	sv.TmpSave = tmpSave
	sv.UpdateFromSlice()
	sv.UpdateEnd(updt)
}

var SliceViewInlineProps = ki.Props{
	"min-width": units.NewValue(20, units.Ch),
}

// ConfigParts configures Parts for the current slice
func (sv *SliceViewInline) ConfigParts() {
	if kit.IfaceIsNil(sv.Slice) {
		return
	}
	sv.Parts.Lay = gi.LayoutHoriz
	config := kit.TypeAndNameList{}
	// always start fresh!
	sv.Values = make([]ValueView, 0)

	mv := reflect.ValueOf(sv.Slice)
	mvnp := kit.NonPtrValue(mv)

	sz := mvnp.Len()
	for i := 0; i < sz; i++ {
		val := kit.OnePtrValue(mvnp.Index(i)) // deal with pointer lists
		vv := ToValueView(val.Interface())
		if vv == nil { // shouldn't happen
			continue
		}
		vv.SetSliceValue(val, sv.Slice, i, sv.TmpSave)
		vtyp := vv.WidgetType()
		idxtxt := fmt.Sprintf("%05d", i)
		valnm := fmt.Sprintf("value-%v", idxtxt)
		config.Add(vtyp, valnm)
		sv.Values = append(sv.Values, vv)
	}
	if !sv.IsArray {
		config.Add(gi.KiT_Action, "AddAction")
	}
	config.Add(gi.KiT_Action, "EditAction")
	mods, updt := sv.Parts.ConfigChildren(config, false)
	if !mods {
		updt = sv.Parts.UpdateStart()
	}
	for i, vv := range sv.Values {
		vvb := vv.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_SliceViewInline).(*SliceViewInline)
			svv.SetChanged()
		})
		widg := sv.Parts.KnownChild(i).(gi.Node2D)
		vv.ConfigWidget(widg)
	}
	if !sv.IsArray {
		adack, ok := sv.Parts.Children().ElemFromEnd(1)
		if ok {
			adac := adack.(*gi.Action)
			adac.SetIcon("plus")
			adac.Tooltip = "add an element to the slice"
			adac.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				svv, _ := recv.Embed(KiT_SliceViewInline).(*SliceViewInline)
				svv.SliceNewAt(-1, true)
			})
		}
	}
	edack, ok := sv.Parts.Children().ElemFromEnd(0)
	if ok {
		edac := edack.(*gi.Action)
		edac.SetIcon("edit")
		edac.Tooltip = "edit slice in a dialog window"
		edac.ActionSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_SliceViewInline).(*SliceViewInline)
			elType := kit.NonPtrType(reflect.TypeOf(svv.Slice).Elem().Elem())
			tynm := "Slice of " + kit.NonPtrType(elType).Name()
			dlg := SliceViewDialog(svv.Viewport, svv.Slice, DlgOpts{Title: tynm, TmpSave: svv.TmpSave}, nil, nil, nil)
			svvvk, ok := dlg.Frame().Children().ElemByType(KiT_SliceView, true, 2)
			if ok {
				svvv := svvvk.(*SliceView)
				svvv.ViewSig.ConnectOnly(svv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					svvvv, _ := recv.Embed(KiT_SliceViewInline).(*SliceViewInline)
					svvvv.ViewSig.Emit(svvvv.This, 0, nil)
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
	sv.ViewSig.Emit(sv.This, 0, nil)
}

// SliceNewAt inserts a new blank element at given index in the slice -- -1
// means the end
func (sv *SliceViewInline) SliceNewAt(idx int, reconfig bool) {
	if sv.IsArray {
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
