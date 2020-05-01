// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// basicviews contains all the ValueView's for basic builtin types

////////////////////////////////////////////////////////////////////////////////////////
//  StructValueView

// StructValueView presents a button to edit the struct
type StructValueView struct {
	ValueViewBase
}

var KiT_StructValueView = kit.Types.AddType(&StructValueView{}, nil)

func (vv *StructValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *StructValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	npv := kit.NonPtrValue(vv.Value)
	if kit.ValueIsZero(vv.Value) || kit.ValueIsZero(npv) {
		ac.SetText("nil")
	} else {
		opv := kit.OnePtrUnderlyingValue(vv.Value)
		if lbler, ok := opv.Interface().(gi.Labeler); ok {
			ac.SetText(lbler.Label())
		} else {
			txt := fmt.Sprintf("%T", npv.Interface())
			if txt == "" {
				fmt.Printf("no label for struct!")
			}
			ac.SetText(txt)
		}
	}
}

func (vv *StructValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	ac := vv.Widget.(*gi.Action)
	ac.Tooltip, _ = vv.Tag("desc")
	ac.SetProp("padding", units.NewPx(2))
	ac.SetProp("margin", units.NewPx(2))
	ac.SetProp("border-radius", units.NewPx(4))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_StructValueView).(*StructValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *StructValueView) HasAction() bool {
	return true
}

func (vv *StructValueView) Activate(vp *gi.Viewport2D, recv ki.Ki, dlgFunc ki.RecvFunc) {
	title, newPath, isZero := vv.Label()
	if isZero {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	opv := kit.OnePtrUnderlyingValue(vv.Value)
	desc, _ := vv.Tag("desc")
	if desc == "list" { // todo: not sure where this comes from but it is uninformative
		desc = ""
	}
	inact := vv.This().(ValueView).IsInactive()
	dlg := StructViewDialog(vp, opv.Interface(), DlgOpts{Title: title, Prompt: desc, TmpSave: vv.TmpSave, Inactive: inact, ViewPath: vpath}, recv, dlgFunc)
	svk := dlg.Frame().ChildByType(KiT_StructView, ki.Embeds, 2)
	if svk != nil {
		sv := svk.(*StructView)
		sv.StructValView = vv
		// no need to connect ViewSig
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  StructInlineValueView

// StructInlineValueView presents a StructViewInline for a struct
type StructInlineValueView struct {
	ValueViewBase
}

var KiT_StructInlineValueView = kit.Types.AddType(&StructInlineValueView{}, nil)

func (vv *StructInlineValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_StructViewInline
	return vv.WidgetTyp
}

func (vv *StructInlineValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sv := vv.Widget.(*StructViewInline)
	cst := vv.Value.Interface()
	if sv.Struct != cst {
		sv.SetStruct(cst)
	} else {
		sv.UpdateFields()
	}
}

func (vv *StructInlineValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sv := vv.Widget.(*StructViewInline)
	sv.Tooltip, _ = vv.Tag("desc")
	sv.StructValView = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	sv.SetStruct(vv.Value.Interface())
	sv.ViewSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_StructInlineValueView).(*StructInlineValueView)
		// vvv.UpdateWidget() // prob not necc..
		vvv.ViewSig.Emit(vvv.This(), 0, nil)
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  SliceValueView

// SliceValueView presents a button to edit slices
type SliceValueView struct {
	ValueViewBase
	IsArray    bool         // is an array, not a slice
	ElType     reflect.Type // type of element in the slice -- has pointer if slice has pointers
	ElIsStruct bool         // whether non-pointer element type is a struct or not
}

var KiT_SliceValueView = kit.Types.AddType(&SliceValueView{}, nil)

func (vv *SliceValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *SliceValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	npv := kit.NonPtrValue(vv.Value)
	txt := ""
	if npv.Kind() == reflect.Interface {
		txt = fmt.Sprintf("Slice: %T", npv.Interface())
	} else {
		if vv.IsArray {
			txt = fmt.Sprintf("Array [%v]%v", npv.Len(), vv.ElType.String())
		} else {
			txt = fmt.Sprintf("Slice [%v]%v", npv.Len(), vv.ElType.String())
		}
	}
	ac.SetText(txt)
}

func (vv *SliceValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	slci := vv.Value.Interface()
	vv.IsArray = kit.NonPtrType(reflect.TypeOf(slci)).Kind() == reflect.Array
	if slci != nil && !kit.IfaceIsNil(slci) {
		vv.ElType = kit.SliceElType(slci)
		vv.ElIsStruct = (kit.NonPtrType(vv.ElType).Kind() == reflect.Struct)
	}
	ac := vv.Widget.(*gi.Action)
	ac.Tooltip, _ = vv.Tag("desc")
	ac.SetProp("padding", units.NewPx(2))
	ac.SetProp("margin", units.NewPx(2))
	ac.SetProp("border-radius", units.NewPx(4))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_SliceValueView).(*SliceValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *SliceValueView) HasAction() bool {
	return true
}

func (vv *SliceValueView) Activate(vp *gi.Viewport2D, recv ki.Ki, dlgFunc ki.RecvFunc) {
	title, newPath, isZero := vv.Label()
	if isZero {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	desc, _ := vv.Tag("desc")
	vvp := kit.OnePtrValue(vv.Value)
	if vvp.Kind() != reflect.Ptr {
		log.Printf("giv.SliceValueView: Cannot view slices with non-pointer struct elements\n")
		return
	}
	inact := vv.This().(ValueView).IsInactive()
	slci := vvp.Interface()
	if !vv.IsArray && vv.ElIsStruct {
		dlg := TableViewDialog(vp, slci, DlgOpts{Title: title, Prompt: desc, TmpSave: vv.TmpSave, Inactive: inact, ViewPath: vpath}, nil, recv, dlgFunc)
		svk := dlg.Frame().ChildByType(KiT_TableView, ki.Embeds, 2)
		if svk != nil {
			sv := svk.(*TableView)
			sv.SliceValView = vv
			sv.ViewSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				vv, _ := recv.Embed(KiT_SliceValueView).(*SliceValueView)
				vv.UpdateWidget()
				vv.ViewSig.Emit(vv.This(), 0, nil)
			})
		}
	} else {
		dlg := SliceViewDialog(vp, slci, DlgOpts{Title: title, Prompt: desc, TmpSave: vv.TmpSave, Inactive: inact, ViewPath: vpath}, nil, recv, dlgFunc)
		svk := dlg.Frame().ChildByType(KiT_SliceView, ki.Embeds, 2)
		if svk != nil {
			sv := svk.(*SliceView)
			sv.SliceValView = vv
			sv.ViewSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				vv, _ := recv.Embed(KiT_SliceValueView).(*SliceValueView)
				vv.UpdateWidget()
				vv.ViewSig.Emit(vv.This(), 0, nil)
			})
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  SliceInlineValueView

// SliceInlineValueView presents a SliceViewInline for a map
type SliceInlineValueView struct {
	ValueViewBase
}

var KiT_SliceInlineValueView = kit.Types.AddType(&SliceInlineValueView{}, nil)

func (vv *SliceInlineValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_SliceViewInline
	return vv.WidgetTyp
}

func (vv *SliceInlineValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sv := vv.Widget.(*SliceViewInline)
	csl := vv.Value.Interface()
	if sv.Slice != csl {
		sv.SetSlice(csl)
	} else {
		sv.UpdateValues()
	}
}

func (vv *SliceInlineValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sv := vv.Widget.(*SliceViewInline)
	sv.Tooltip, _ = vv.Tag("desc")
	sv.SliceValView = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	// npv := vv.Value.Elem()
	sv.SetInactiveState(vv.This().(ValueView).IsInactive())
	sv.SetSlice(vv.Value.Interface())
	sv.ViewSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_SliceInlineValueView).(*SliceInlineValueView)
		vvv.UpdateWidget()
		vvv.ViewSig.Emit(vvv.This(), 0, nil)
	})
}

////////////////////////////////////////////////////////////////////////////////////////
//  MapValueView

// MapValueView presents a button to edit maps
type MapValueView struct {
	ValueViewBase
}

var KiT_MapValueView = kit.Types.AddType(&MapValueView{}, nil)

func (vv *MapValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Action
	return vv.WidgetTyp
}

func (vv *MapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Action)
	npv := kit.NonPtrValue(vv.Value)
	mpi := vv.Value.Interface()
	txt := ""
	if npv.Kind() == reflect.Interface {
		txt = fmt.Sprintf("Map: %T", npv.Interface())
	} else {
		txt = fmt.Sprintf("Map: [%v %v]%v", npv.Len(), kit.MapKeyType(mpi).String(), kit.MapValueType(mpi).String())
	}
	ac.SetText(txt)
}

func (vv *MapValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Action)
	ac.Tooltip, _ = vv.Tag("desc")
	ac.SetProp("padding", units.NewPx(2))
	ac.SetProp("margin", units.NewPx(2))
	ac.SetProp("border-radius", units.NewPx(4))
	ac.ActionSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_MapValueView).(*MapValueView)
		ac := vvv.Widget.(*gi.Action)
		vvv.Activate(ac.Viewport, nil, nil)
	})
	vv.UpdateWidget()
}

func (vv *MapValueView) HasAction() bool {
	return true
}

func (vv *MapValueView) Activate(vp *gi.Viewport2D, recv ki.Ki, dlgFunc ki.RecvFunc) {
	title, newPath, isZero := vv.Label()
	if isZero {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	desc, _ := vv.Tag("desc")
	mpi := vv.Value.Interface()
	inact := vv.This().(ValueView).IsInactive()
	dlg := MapViewDialog(vp, mpi, DlgOpts{Title: title, Prompt: desc, TmpSave: vv.TmpSave, Inactive: inact, ViewPath: vpath}, recv, dlgFunc)
	mvk := dlg.Frame().ChildByType(KiT_MapView, ki.Embeds, 2)
	if mvk != nil {
		mv := mvk.(*MapView)
		mv.MapValView = vv
		mv.ViewSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			vv, _ := recv.Embed(KiT_MapValueView).(*MapValueView)
			vv.UpdateWidget()
			vv.ViewSig.Emit(vv.This(), 0, nil)
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  MapInlineValueView

// MapInlineValueView presents a MapViewInline for a map
type MapInlineValueView struct {
	ValueViewBase
}

var KiT_MapInlineValueView = kit.Types.AddType(&MapInlineValueView{}, nil)

func (vv *MapInlineValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_MapViewInline
	return vv.WidgetTyp
}

func (vv *MapInlineValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sv := vv.Widget.(*MapViewInline)
	cmp := vv.Value.Interface()
	if sv.Map != cmp {
		sv.SetMap(cmp)
	} else {
		sv.UpdateValues()
	}
}

func (vv *MapInlineValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sv := vv.Widget.(*MapViewInline)
	sv.Tooltip, _ = vv.Tag("desc")
	sv.MapValView = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	// npv := vv.Value.Elem()
	sv.SetInactiveState(vv.This().(ValueView).IsInactive())
	sv.SetMap(vv.Value.Interface())
	sv.ViewSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_MapInlineValueView).(*MapInlineValueView)
		vvv.UpdateWidget()
		vvv.ViewSig.Emit(vvv.This(), 0, nil)
	})
}

////////////////////////////////////////////////////////////////////////////////////////
//  KiPtrValueView

// KiPtrValueView provides a chooser for pointers to Ki objects
type KiPtrValueView struct {
	ValueViewBase
}

var KiT_KiPtrValueView = kit.Types.AddType(&KiPtrValueView{}, nil)

func (vv *KiPtrValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_MenuButton
	return vv.WidgetTyp
}

// get the Ki struct itself (or nil)
func (vv *KiPtrValueView) KiStruct() ki.Ki {
	if !vv.Value.IsValid() {
		return nil
	}
	if vv.Value.IsNil() {
		return nil
	}
	npv := vv.Value
	if vv.Value.Kind() == reflect.Ptr {
		npv = vv.Value.Elem()
	}
	if npv.Kind() == reflect.Struct {
		npv = vv.Value // go back up
	}
	if !npv.IsNil() {
		k, ok := npv.Interface().(ki.Ki)
		if ok && k != nil {
			return k
		}
	}
	return nil
}

func (vv *KiPtrValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	mb := vv.Widget.(*gi.MenuButton)
	path := "nil"
	k := vv.KiStruct()
	if k != nil {
		path = k.Path()
	}
	mb.SetText(path)
}

func (vv *KiPtrValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	mb := vv.Widget.(*gi.MenuButton)
	mb.Tooltip, _ = vv.Tag("desc")
	mb.SetProp("padding", units.NewPx(2))
	mb.SetProp("margin", units.NewPx(2))
	mb.ResetMenu()
	mb.Menu.AddAction(gi.ActOpts{Label: "Edit"},
		vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			vvv, _ := recv.Embed(KiT_KiPtrValueView).(*KiPtrValueView)
			k := vvv.KiStruct()
			if k != nil {
				mb := vvv.Widget.(*gi.MenuButton)
				vvv.Activate(mb.Viewport, nil, nil)
			}
		})
	mb.Menu.AddAction(gi.ActOpts{Label: "GoGiEditor"},
		vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			vvv, _ := recv.Embed(KiT_KiPtrValueView).(*KiPtrValueView)
			k := vvv.KiStruct()
			if k != nil {
				GoGiEditorDialog(k)
			}
		})
	vv.UpdateWidget()
}

func (vv *KiPtrValueView) HasAction() bool {
	return true
}

func (vv *KiPtrValueView) Activate(vp *gi.Viewport2D, recv ki.Ki, dlgFunc ki.RecvFunc) {
	title, newPath, isZero := vv.Label()
	if isZero {
		return
	}
	k := vv.KiStruct()
	if k == nil {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	desc, _ := vv.Tag("desc")
	inact := vv.This().(ValueView).IsInactive()
	StructViewDialog(vp, k, DlgOpts{Title: title, Prompt: desc, TmpSave: vv.TmpSave, Inactive: inact, ViewPath: vpath}, recv, dlgFunc)
}

////////////////////////////////////////////////////////////////////////////////////////
//  BoolValueView

// BoolValueView presents a checkbox for a boolean
type BoolValueView struct {
	ValueViewBase
}

var KiT_BoolValueView = kit.Types.AddType(&BoolValueView{}, nil)

func (vv *BoolValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_CheckBox
	return vv.WidgetTyp
}

func (vv *BoolValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	cb := vv.Widget.(*gi.CheckBox)
	npv := kit.NonPtrValue(vv.Value)
	bv, _ := kit.ToBool(npv.Interface())
	cb.SetChecked(bv)
}

func (vv *BoolValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	cb := vv.Widget.(*gi.CheckBox)
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetInactiveState(vv.This().(ValueView).IsInactive())
	cb.ButtonSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonToggled) {
			vvv, _ := recv.Embed(KiT_BoolValueView).(*BoolValueView)
			cbb := vvv.Widget.(*gi.CheckBox)
			if vvv.SetValue(cbb.IsChecked()) {
				vvv.UpdateWidget() // always update after setting value..
			}
		}
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  IntValueView

// IntValueView presents a spinbox
type IntValueView struct {
	ValueViewBase
}

var KiT_IntValueView = kit.Types.AddType(&IntValueView{}, nil)

func (vv *IntValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_SpinBox
	return vv.WidgetTyp
}

func (vv *IntValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.SpinBox)
	npv := kit.NonPtrValue(vv.Value)
	fv, ok := kit.ToFloat32(npv.Interface())
	if ok {
		sb.SetValue(fv)
	}
}

func (vv *IntValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sb := vv.Widget.(*gi.SpinBox)
	sb.Tooltip, _ = vv.Tag("desc")
	sb.SetInactiveState(vv.This().(ValueView).IsInactive())
	sb.Defaults()
	sb.Step = 1.0
	sb.PageStep = 10.0
	sb.SetProp("#textfield", ki.Props{
		"width": units.NewCh(5),
	})
	vk := vv.Value.Kind()
	if vk >= reflect.Uint && vk <= reflect.Uint64 {
		sb.SetMin(0)
	}
	if mintag, ok := vv.Tag("min"); ok {
		minv, ok := kit.ToFloat32(mintag)
		if ok {
			sb.SetMin(minv)
		}
	}
	if maxtag, ok := vv.Tag("max"); ok {
		maxv, ok := kit.ToFloat32(maxtag)
		if ok {
			sb.SetMax(maxv)
		}
	}
	if steptag, ok := vv.Tag("step"); ok {
		step, ok := kit.ToFloat32(steptag)
		if ok {
			sb.Step = step
		}
	}
	if fmttag, ok := vv.Tag("format"); ok {
		sb.Format = fmttag
	}
	sb.SpinBoxSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_IntValueView).(*IntValueView)
		sbb := vvv.Widget.(*gi.SpinBox)
		if vvv.SetValue(sbb.Value) {
			vvv.UpdateWidget()
		}
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  FloatValueView

// FloatValueView presents a spinbox
type FloatValueView struct {
	ValueViewBase
}

var KiT_FloatValueView = kit.Types.AddType(&FloatValueView{}, nil)

func (vv *FloatValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_SpinBox
	return vv.WidgetTyp
}

func (vv *FloatValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.SpinBox)
	npv := kit.NonPtrValue(vv.Value)
	fv, ok := kit.ToFloat32(npv.Interface())
	if ok {
		sb.SetValue(fv)
	}
}

func (vv *FloatValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sb := vv.Widget.(*gi.SpinBox)
	sb.Tooltip, _ = vv.Tag("desc")
	sb.SetInactiveState(vv.This().(ValueView).IsInactive())
	sb.Defaults()
	sb.Step = 1.0
	sb.PageStep = 10.0
	if mintag, ok := vv.Tag("min"); ok {
		minv, ok := kit.ToFloat32(mintag)
		if ok {
			sb.HasMin = true
			sb.Min = minv
		}
	}
	if maxtag, ok := vv.Tag("max"); ok {
		maxv, ok := kit.ToFloat32(maxtag)
		if ok {
			sb.HasMax = true
			sb.Max = maxv
		}
	}
	sb.Step = .1 // smaller default
	if steptag, ok := vv.Tag("step"); ok {
		step, ok := kit.ToFloat32(steptag)
		if ok {
			sb.Step = step
		}
	}
	if fmttag, ok := vv.Tag("format"); ok {
		sb.Format = fmttag
	}

	sb.SpinBoxSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_FloatValueView).(*FloatValueView)
		sbb := vvv.Widget.(*gi.SpinBox)
		if vvv.SetValue(sbb.Value) {
			vvv.UpdateWidget()
		}
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  EnumValueView

// EnumValueView presents a combobox for choosing enums
type EnumValueView struct {
	ValueViewBase
	AltType reflect.Type // alternative type, e.g., from EnumType: property
}

var KiT_EnumValueView = kit.Types.AddType(&EnumValueView{}, nil)

func (vv *EnumValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_ComboBox
	return vv.WidgetTyp
}

func (vv *EnumValueView) EnumType() reflect.Type {
	if vv.AltType != nil {
		return vv.AltType
	}
	// derive type indirectly from the interface instead of directly from the value
	// because that works for interface{} types as in property maps
	typ := kit.NonPtrType(reflect.TypeOf(vv.Value.Interface()))
	return typ
}

func (vv *EnumValueView) SetEnumValueFromInt(ival int64) bool {
	typ := vv.EnumType()
	eval := kit.EnumIfaceFromInt64(ival, typ)
	return vv.SetValue(eval)
}

func (vv *EnumValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.ComboBox)
	npv := kit.NonPtrValue(vv.Value)
	iv, ok := kit.ToInt(npv.Interface())
	if ok {
		sb.SetCurIndex(int(iv)) // todo: currently only working for 0-based values
	}
}

func (vv *EnumValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	cb := vv.Widget.(*gi.ComboBox)
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetInactiveState(vv.This().(ValueView).IsInactive())
	cb.SetProp("padding", units.NewPx(2))
	cb.SetProp("margin", units.NewPx(2))

	typ := vv.EnumType()
	cb.ItemsFromEnum(typ, false, 50)
	cb.ComboSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_EnumValueView).(*EnumValueView)
		cbb := vvv.Widget.(*gi.ComboBox)
		eval := cbb.CurVal.(kit.EnumValue)
		if vvv.SetEnumValueFromInt(eval.Value) { // todo: using index
			vvv.UpdateWidget()
		}
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  BitFlagView

// BitFlagView presents a ButtonBox for bitflags
type BitFlagView struct {
	ValueViewBase
	AltType reflect.Type // alternative type, e.g., from EnumType: property
}

var KiT_BitFlagView = kit.Types.AddType(&BitFlagView{}, nil)

func (vv *BitFlagView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_ButtonBox
	return vv.WidgetTyp
}

func (vv *BitFlagView) EnumType() reflect.Type {
	if vv.AltType != nil {
		return vv.AltType
	}
	// derive type indirectly from the interface instead of directly from the value
	// because that works for interface{} types as in property maps
	typ := kit.NonPtrType(reflect.TypeOf(vv.Value.Interface()))
	return typ
}

func (vv *BitFlagView) SetEnumValueFromInt(ival int64) bool {
	typ := vv.EnumType()
	eval := kit.EnumIfaceFromInt64(ival, typ)
	return vv.SetValue(eval)
}

func (vv *BitFlagView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.ButtonBox)
	npv := kit.NonPtrValue(vv.Value)
	iv, ok := kit.ToInt(npv.Interface())
	if ok {
		typ := vv.EnumType()
		sb.UpdateFromBitFlags(typ, int64(iv))
	}
}

func (vv *BitFlagView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	cb := vv.Widget.(*gi.ButtonBox)
	vv.StdConfigWidget(&cb.Parts)
	cb.Parts.Lay = gi.LayoutHoriz
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetInactiveState(vv.This().(ValueView).IsInactive())
	cb.SetProp("padding", units.NewPx(2))
	cb.SetProp("margin", units.NewPx(2))

	typ := vv.EnumType()
	cb.ItemsFromEnum(typ)
	cb.ConfigParts()
	cb.ButtonSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_BitFlagView).(*BitFlagView)
		cbb := vvv.Widget.(*gi.ButtonBox)
		etyp := vvv.EnumType()
		val := cbb.BitFlagsValue(etyp)
		vvv.SetEnumValueFromInt(val)
		// vvv.UpdateWidget()
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  TypeValueView

// TypeValueView presents a combobox for choosing types
type TypeValueView struct {
	ValueViewBase
}

var KiT_TypeValueView = kit.Types.AddType(&TypeValueView{}, nil)

func (vv *TypeValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_ComboBox
	return vv.WidgetTyp
}

func (vv *TypeValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.ComboBox)
	npv := kit.NonPtrValue(vv.Value)
	typ, ok := npv.Interface().(reflect.Type)
	if ok {
		sb.SetCurVal(typ)
	}
}

func (vv *TypeValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	cb := vv.Widget.(*gi.ComboBox)
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetInactiveState(vv.This().(ValueView).IsInactive())

	typEmbeds := ki.KiT_Node
	if kiv, ok := vv.Owner.(ki.Ki); ok {
		if tep, ok := kiv.PropInherit("type-embeds", ki.Inherit, ki.TypeProps); ok {
			if te, ok := tep.(reflect.Type); ok {
				typEmbeds = te
			}
		}
	}
	if tetag, ok := vv.Tag("type-embeds"); ok {
		typ := kit.Types.Type(tetag)
		if typ != nil {
			typEmbeds = typ
		}
	}

	tl := kit.Types.AllEmbedsOf(typEmbeds, true, false)
	cb.ItemsFromTypes(tl, false, true, 50)

	cb.ComboSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_TypeValueView).(*TypeValueView)
		cbb := vvv.Widget.(*gi.ComboBox)
		tval := cbb.CurVal.(reflect.Type)
		if vvv.SetValue(tval) {
			vvv.UpdateWidget()
		}
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  ByteSliceValueView

// ByteSliceValueView presents a textfield of the bytes
type ByteSliceValueView struct {
	ValueViewBase
}

var KiT_ByteSliceValueView = kit.Types.AddType(&ByteSliceValueView{}, nil)

func (vv *ByteSliceValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_TextField
	return vv.WidgetTyp
}

func (vv *ByteSliceValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	tf := vv.Widget.(*gi.TextField)
	npv := kit.NonPtrValue(vv.Value)
	bv, ok := npv.Interface().([]byte)
	if ok {
		tf.SetText(string(bv))
	}
}

func (vv *ByteSliceValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	tf := vv.Widget.(*gi.TextField)
	tf.Tooltip, _ = vv.Tag("desc")
	tf.SetInactiveState(vv.This().(ValueView).IsInactive())
	tf.SetStretchMaxWidth()
	tf.SetProp("min-width", units.NewCh(16))

	tf.TextFieldSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
			vvv, _ := recv.Embed(KiT_ByteSliceValueView).(*ByteSliceValueView)
			tf := send.(*gi.TextField)
			if vvv.SetValue(tf.Text()) {
				vvv.UpdateWidget() // always update after setting value..
			}
		}
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  RuneSliceValueView

// RuneSliceValueView presents a textfield of the bytes
type RuneSliceValueView struct {
	ValueViewBase
}

var KiT_RuneSliceValueView = kit.Types.AddType(&RuneSliceValueView{}, nil)

func (vv *RuneSliceValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_TextField
	return vv.WidgetTyp
}

func (vv *RuneSliceValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	tf := vv.Widget.(*gi.TextField)
	npv := kit.NonPtrValue(vv.Value)
	rv, ok := npv.Interface().([]rune)
	if ok {
		tf.SetText(string(rv))
	}
}

func (vv *RuneSliceValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	tf := vv.Widget.(*gi.TextField)
	tf.Tooltip, _ = vv.Tag("desc")
	tf.SetInactiveState(vv.This().(ValueView).IsInactive())
	tf.SetStretchMaxWidth()
	tf.SetProp("min-width", units.NewCh(16))

	tf.TextFieldSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
			vvv, _ := recv.Embed(KiT_RuneSliceValueView).(*RuneSliceValueView)
			tf := send.(*gi.TextField)
			if vvv.SetValue(tf.Text()) {
				vvv.UpdateWidget() // always update after setting value..
			}
		}
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  NilValueView

// NilValueView presents a label saying 'nil' -- for any nil or otherwise unrepresentable items
type NilValueView struct {
	ValueViewBase
}

var KiT_NilValueView = kit.Types.AddType(&NilValueView{}, nil)

func (vv *NilValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_Label
	return vv.WidgetTyp
}

func (vv *NilValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.Label)
	npv := kit.NonPtrValue(vv.Value)
	tstr := ""
	if !kit.ValueIsZero(npv) {
		tstr = npv.String() // npv.Type().String()
	} else if !kit.ValueIsZero(vv.Value) {
		tstr = vv.Value.String() // vv.Value.Type().String()
	}
	sb.SetText("nil " + tstr)
}

func (vv *NilValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sb := vv.Widget.(*gi.Label)
	sb.Tooltip, _ = vv.Tag("desc")
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  TimeValueView

var DefaultTimeFormat = "2006-01-02 15:04:05 MST"

// TimeValueView presents a checkbox for a boolean
type TimeValueView struct {
	ValueViewBase
}

var KiT_TimeValueView = kit.Types.AddType(&TimeValueView{}, nil)

func (vv *TimeValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_TextField
	return vv.WidgetTyp
}

// TimeVal decodes Value into a *time.Time value -- also handles FileTime case
func (vv *TimeValueView) TimeVal() *time.Time {
	tmi := kit.PtrValue(vv.Value).Interface()
	switch v := tmi.(type) {
	case *time.Time:
		return v
	case *FileTime:
		return (*time.Time)(v)
	}
	return nil
}

func (vv *TimeValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	tf := vv.Widget.(*gi.TextField)
	tm := vv.TimeVal()
	tf.SetText(tm.Format(DefaultTimeFormat))
}

func (vv *TimeValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	tf := vv.Widget.(*gi.TextField)
	tf.SetStretchMaxWidth()
	tf.Tooltip, _ = vv.Tag("desc")
	tf.SetInactiveState(vv.This().(ValueView).IsInactive())
	tf.SetProp("min-width", units.NewCh(float32(len(DefaultTimeFormat)+2)))
	tf.TextFieldSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
			vvv, _ := recv.Embed(KiT_TimeValueView).(*TimeValueView)
			tf := send.(*gi.TextField)
			nt, err := time.Parse(DefaultTimeFormat, tf.Text())
			if err != nil {
				log.Println(err)
			} else {
				tm := vvv.TimeVal()
				*tm = nt
				vvv.ViewSig.Emit(vvv.This(), 0, nil)
				vvv.UpdateWidget()
			}
		}
	})
	vv.UpdateWidget()
}
