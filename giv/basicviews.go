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
		txt := fmt.Sprintf("%T", npv.Interface())
		ac.SetText(txt)
	}
}

func (vv *StructValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	ac := vv.Widget.(*gi.Action)
	ac.Tooltip, _ = vv.Tag("desc")
	ac.SetProp("padding", units.NewValue(2, units.Px))
	ac.SetProp("margin", units.NewValue(2, units.Px))
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	tynm := kit.NonPtrType(vv.Value.Type()).Name()
	desc, _ := vv.Tag("desc")
	dlg := StructViewDialog(vp, vv.Value.Interface(), DlgOpts{Title: tynm, Prompt: desc, TmpSave: vv.TmpSave}, recv, dlgFunc)
	dlg.SetInactiveState(vv.This.(ValueView).IsInactive())
	svk, ok := dlg.Frame().Children().ElemByType(KiT_StructView, true, 2)
	if ok {
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
	sv.UpdateFields()
}

func (vv *StructInlineValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	sv := vv.Widget.(*StructViewInline)
	sv.Tooltip, _ = vv.Tag("desc")
	sv.StructValView = vv
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	sv.SetStruct(vv.Value.Interface(), vv.TmpSave)
	sv.ViewSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_StructInlineValueView).(*StructInlineValueView)
		// vvv.UpdateWidget() // prob not necc..
		vvv.ViewSig.Emit(vvv.This, 0, nil)
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
	slci := vv.Value.Interface()
	vv.IsArray = kit.NonPtrType(reflect.TypeOf(slci)).Kind() == reflect.Array
	vv.ElType = kit.SliceElType(slci)
	vv.ElIsStruct = (kit.NonPtrType(vv.ElType).Kind() == reflect.Struct)
	ac := vv.Widget.(*gi.Action)
	ac.Tooltip, _ = vv.Tag("desc")
	ac.SetProp("padding", units.NewValue(2, units.Px))
	ac.SetProp("margin", units.NewValue(2, units.Px))
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	tynm := ""
	if vv.IsArray {
		tynm = "Array of "
	} else {
		tynm = "Slice of "
	}
	tynm += kit.NonPtrType(vv.ElType).String()
	desc, _ := vv.Tag("desc")
	slci := vv.Value.Interface()
	if !vv.IsArray && vv.ElIsStruct {
		dlg := TableViewDialog(vp, slci, DlgOpts{Title: tynm, Prompt: desc, TmpSave: vv.TmpSave}, nil, recv, dlgFunc)
		dlg.SetInactiveState(vv.This.(ValueView).IsInactive())
		svk, ok := dlg.Frame().Children().ElemByType(KiT_TableView, true, 2)
		if ok {
			sv := svk.(*TableView)
			sv.SliceValView = vv
			sv.ViewSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				vv, _ := recv.Embed(KiT_SliceValueView).(*SliceValueView)
				vv.UpdateWidget()
				vv.ViewSig.Emit(vv.This, 0, nil)
			})
		}
	} else {
		dlg := SliceViewDialog(vp, slci, DlgOpts{Title: tynm, Prompt: desc, TmpSave: vv.TmpSave}, nil, recv, dlgFunc)
		dlg.SetInactiveState(vv.This.(ValueView).IsInactive())
		svk, ok := dlg.Frame().Children().ElemByType(KiT_SliceView, true, 2)
		if ok {
			sv := svk.(*SliceView)
			sv.SliceValView = vv
			sv.ViewSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				vv, _ := recv.Embed(KiT_SliceValueView).(*SliceValueView)
				vv.UpdateWidget()
				vv.ViewSig.Emit(vv.This, 0, nil)
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
	sv.UpdateValues()
}

func (vv *SliceInlineValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	sv := vv.Widget.(*SliceViewInline)
	sv.Tooltip, _ = vv.Tag("desc")
	sv.SliceValView = vv
	// npv := vv.Value.Elem()
	sv.SetInactiveState(vv.This.(ValueView).IsInactive())
	sv.SetSlice(vv.Value.Interface(), vv.TmpSave)
	sv.ViewSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_SliceInlineValueView).(*SliceInlineValueView)
		vvv.UpdateWidget()
		vvv.ViewSig.Emit(vvv.This, 0, nil)
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
	ac := vv.Widget.(*gi.Action)
	ac.Tooltip, _ = vv.Tag("desc")
	ac.SetProp("padding", units.NewValue(2, units.Px))
	ac.SetProp("margin", units.NewValue(2, units.Px))
	ac.SetProp("border-radius", units.NewValue(4, units.Px))
	ac.ActionSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	tmptyp := kit.NonPtrType(vv.Value.Type())
	desc, _ := vv.Tag("desc")
	mpi := vv.Value.Interface()
	tynm := tmptyp.Name()
	if tynm == "" {
		tynm = tmptyp.String()
	}
	dlg := MapViewDialog(vp, mpi, DlgOpts{Title: tynm, Prompt: desc, TmpSave: vv.TmpSave}, recv, dlgFunc)
	dlg.SetInactiveState(vv.This.(ValueView).IsInactive())
	mvk, ok := dlg.Frame().Children().ElemByType(KiT_MapView, true, 2)
	if ok {
		mv := mvk.(*MapView)
		mv.MapValView = vv
		mv.ViewSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			vv, _ := recv.Embed(KiT_MapValueView).(*MapValueView)
			vv.UpdateWidget()
			vv.ViewSig.Emit(vv.This, 0, nil)
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
	sv.UpdateValues()
}

func (vv *MapInlineValueView) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	sv := vv.Widget.(*MapViewInline)
	sv.Tooltip, _ = vv.Tag("desc")
	sv.MapValView = vv
	// npv := vv.Value.Elem()
	sv.SetInactiveState(vv.This.(ValueView).IsInactive())
	sv.SetMap(vv.Value.Interface(), vv.TmpSave)
	sv.ViewSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.Embed(KiT_MapInlineValueView).(*MapInlineValueView)
		vvv.UpdateWidget()
		vvv.ViewSig.Emit(vvv.This, 0, nil)
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
	mb := vv.Widget.(*gi.MenuButton)
	mb.Tooltip, _ = vv.Tag("desc")
	mb.SetProp("padding", units.NewValue(2, units.Px))
	mb.SetProp("margin", units.NewValue(2, units.Px))
	mb.ResetMenu()
	mb.Menu.AddAction(gi.ActOpts{Label: "Edit"},
		vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			vvv, _ := recv.Embed(KiT_KiPtrValueView).(*KiPtrValueView)
			k := vvv.KiStruct()
			if k != nil {
				mb := vvv.Widget.(*gi.MenuButton)
				vvv.Activate(mb.Viewport, nil, nil)
			}
		})
	mb.Menu.AddAction(gi.ActOpts{Label: "GoGiEditor"},
		vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	k := vv.KiStruct()
	if k == nil {
		return
	}
	desc, _ := vv.Tag("desc")
	tynm := kit.NonPtrType(vv.Value.Type()).Name()
	dlg := StructViewDialog(vp, k, DlgOpts{Title: tynm, Prompt: desc, TmpSave: vv.TmpSave}, recv, dlgFunc)
	dlg.SetInactiveState(vv.This.(ValueView).IsInactive())
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
	cb := vv.Widget.(*gi.CheckBox)
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetInactiveState(vv.This.(ValueView).IsInactive())
	cb.ButtonSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	sb := vv.Widget.(*gi.SpinBox)
	sb.Tooltip, _ = vv.Tag("desc")
	sb.SetInactiveState(vv.This.(ValueView).IsInactive())
	sb.Defaults()
	sb.Step = 1.0
	sb.PageStep = 10.0
	sb.SetProp("#textfield", ki.Props{
		"width": units.NewValue(5, units.Ch),
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
	sb.SpinBoxSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	sb := vv.Widget.(*gi.SpinBox)
	sb.Tooltip, _ = vv.Tag("desc")
	sb.SetInactiveState(vv.This.(ValueView).IsInactive())
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
	if steptag, ok := vv.Tag("step"); ok {
		step, ok := kit.ToFloat32(steptag)
		if ok {
			sb.Step = step
		}
	}

	sb.SpinBoxSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
}

var KiT_EnumValueView = kit.Types.AddType(&EnumValueView{}, nil)

func (vv *EnumValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_ComboBox
	return vv.WidgetTyp
}

func (vv *EnumValueView) EnumType() reflect.Type {
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
	cb := vv.Widget.(*gi.ComboBox)
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetInactiveState(vv.This.(ValueView).IsInactive())
	cb.SetProp("padding", units.NewValue(2, units.Px))
	cb.SetProp("margin", units.NewValue(2, units.Px))

	typ := vv.EnumType()
	cb.ItemsFromEnum(typ, false, 50)
	cb.ComboSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	cb := vv.Widget.(*gi.ComboBox)
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetInactiveState(vv.This.(ValueView).IsInactive())

	typEmbeds := ki.KiT_Node
	if kiv, ok := vv.Owner.(ki.Ki); ok {
		if tep, ok := kiv.PropInherit("type-embeds", true, true); ok {
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

	cb.ComboSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	tf := vv.Widget.(*gi.TextField)
	tf.Tooltip, _ = vv.Tag("desc")
	tf.SetInactiveState(vv.This.(ValueView).IsInactive())
	tf.SetStretchMaxWidth()
	tf.SetProp("min-width", units.NewValue(16, units.Ch))
	if widthtag, ok := vv.Tag("width"); ok {
		width, ok := kit.ToFloat32(widthtag)
		if ok {
			tf.SetMinPrefWidth(units.NewValue(width, units.Ch))
		}
	}
	if maxwidthtag, ok := vv.Tag("max-width"); ok {
		width, ok := kit.ToFloat32(maxwidthtag)
		if ok {
			tf.SetProp("max-width", units.NewValue(width, units.Ch))
		}
	}

	tf.TextFieldSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
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
	tf := vv.Widget.(*gi.TextField)
	tf.Tooltip, _ = vv.Tag("desc")
	tf.SetInactiveState(vv.This.(ValueView).IsInactive())
	tf.SetStretchMaxWidth()
	tf.SetProp("min-width", units.NewValue(16, units.Ch))
	if widthtag, ok := vv.Tag("width"); ok {
		width, ok := kit.ToFloat32(widthtag)
		if ok {
			tf.SetMinPrefWidth(units.NewValue(width, units.Ch))
		}
	}
	if maxwidthtag, ok := vv.Tag("max-width"); ok {
		width, ok := kit.ToFloat32(maxwidthtag)
		if ok {
			tf.SetProp("max-width", units.NewValue(width, units.Ch))
		}
	}

	tf.TextFieldSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			vvv, _ := recv.Embed(KiT_RuneSliceValueView).(*RuneSliceValueView)
			tf := send.(*gi.TextField)
			if vvv.SetValue(tf.Text()) {
				vvv.UpdateWidget() // always update after setting value..
			}
		}
	})
	vv.UpdateWidget()
}
