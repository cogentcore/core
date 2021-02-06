// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// SliceViewInline represents a slice as a single line widget, for smaller
// slices and those explicitly marked inline -- constructs widgets in Parts to
// show the key names and editor vals for each value.
type SliceViewInline struct {
	gi.PartsWidgetBase
	Slice        interface{} `desc:"the slice that we are a view onto"`
	SliceValView ValueView   `desc:"ValueView for the slice itself, if this was created within value view framework -- otherwise nil"`
	IsArray      bool        `desc:"whether the slice is actually an array -- no modifications"`
	IsFixedLen   bool        `desc:"whether the slice has a fixed-len flag on it"`
	Changed      bool        `desc:"has the slice been edited?"`
	Values       []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave      ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig      ki.Signal   `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	ViewPath     string      `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`
}

var KiT_SliceViewInline = kit.Types.AddType(&SliceViewInline{}, SliceViewInlineProps)

func (sv *SliceViewInline) Disconnect() {
	sv.PartsWidgetBase.Disconnect()
	sv.ViewSig.DisconnectAll()
}

// SetSlice sets the source slice that we are viewing -- rebuilds the children to represent this slice
func (sv *SliceViewInline) SetSlice(sl interface{}) {
	updt := false
	if sv.Slice != sl {
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
	"EnumType:Flag": gi.KiT_NodeFlags,
	"min-width":     units.NewCh(20),
}

// ConfigParts configures Parts for the current slice
func (sv *SliceViewInline) ConfigParts() {
	if kit.IfaceIsNil(sv.Slice) {
		return
	}
	sv.Parts.Lay = gi.LayoutHoriz
	sv.Parts.SetProp("overflow", gist.OverflowHidden) // no scrollbars!
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
		idxtxt := fmt.Sprintf("%05d", i)
		valnm := fmt.Sprintf("value-%v", idxtxt)
		config.Add(vtyp, valnm)
		sv.Values = append(sv.Values, vv)
	}
	if !sv.IsArray && !sv.IsFixedLen {
		config.Add(gi.KiT_Action, "add-action")
	}
	config.Add(gi.KiT_Action, "edit-action")
	mods, updt := sv.Parts.ConfigChildren(config)
	if !mods {
		updt = sv.Parts.UpdateStart()
	}
	for i, vv := range sv.Values {
		vvb := vv.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_SliceViewInline).(*SliceViewInline)
			svv.SetChanged()
		})
		widg := sv.Parts.Child(i).(gi.Node2D)
		if sv.SliceValView != nil {
			vv.SetTags(sv.SliceValView.AllTags())
		}
		vv.ConfigWidget(widg)
		if sv.IsInactive() {
			widg.AsNode2D().SetInactive()
		}
	}
	if !sv.IsArray && !sv.IsFixedLen {
		adack, err := sv.Parts.Children().ElemFromEndTry(1)
		if err == nil {
			adac := adack.(*gi.Action)
			adac.SetIcon("plus")
			adac.Tooltip = "add an element to the slice"
			adac.ActionSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				svv, _ := recv.Embed(KiT_SliceViewInline).(*SliceViewInline)
				svv.SliceNewAt(-1, true)
			})
		}
	}
	edack, err := sv.Parts.Children().ElemFromEndTry(0)
	if err == nil {
		edac := edack.(*gi.Action)
		edac.SetIcon("edit")
		edac.Tooltip = "edit slice in a dialog window"
		edac.ActionSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.Embed(KiT_SliceViewInline).(*SliceViewInline)
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
			svvvk := dlg.Frame().ChildByType(KiT_SliceView, ki.Embeds, 2)
			if svvvk != nil {
				svvv := svvvk.(*SliceView)
				svvv.SliceValView = svv.SliceValView
				svvv.ViewSig.ConnectOnly(svv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					svvvv, _ := recv.Embed(KiT_SliceViewInline).(*SliceViewInline)
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
