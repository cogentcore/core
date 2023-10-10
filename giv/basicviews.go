// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log"
	"log/slog"
	"reflect"
	"time"

	"goki.dev/enums"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// basicviews contains all the ValueView's for basic builtin types

////////////////////////////////////////////////////////////////////////////////////////
//  StructValueView

// StructValueView presents a button to edit the struct
type StructValueView struct {
	ValueViewBase
}

func (vv *StructValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *StructValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	npv := laser.NonPtrValue(vv.Value)
	if laser.ValueIsZero(vv.Value) || laser.ValueIsZero(npv) {
		bt.SetText("nil")
	} else {
		opv := laser.OnePtrUnderlyingValue(vv.Value)
		if lbler, ok := opv.Interface().(gi.Labeler); ok {
			bt.SetText(lbler.Label())
		} else {
			txt := fmt.Sprintf("%T", npv.Interface())
			if txt == "" {
				fmt.Printf("no label for struct!")
			}
			bt.SetText(txt)
		}
	}
}

func (vv *StructValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	ac := vv.Widget.(*gi.Button)
	ac.Icon = icons.Edit
	ac.Tooltip, _ = vv.Tag("desc")
	ac.OnClick(func(e events.Event) {
		vv.OpenDialog(ac, nil)
	})
	vv.UpdateWidget()
}

func (vv *StructValueView) HasButton() bool {
	return true
}

func (vv *StructValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	title, newPath, isZero := vv.Label()
	if isZero {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	opv := laser.OnePtrUnderlyingValue(vv.Value)
	desc, _ := vv.Tag("desc")
	if desc == "list" { // todo: not sure where this comes from but it is uninformative
		desc = ""
	}
	inact := vv.This().(ValueView).IsInactive()
	dlg := StructViewDialog(vv.Widget, DlgOpts{Title: title, Prompt: desc, TmpSave: vv.TmpSave, Inactive: inact, ViewPath: vpath}, opv.Interface(), nil)
	svk := dlg.Stage.Scene.ChildByType(StructViewType, ki.Embeds, 2)
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

func (vv *StructInlineValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = StructViewInlineType
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

func (vv *StructInlineValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sv := vv.Widget.(*StructViewInline)
	sv.Tooltip, _ = vv.Tag("desc")
	sv.StructValView = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	sv.SetStruct(vv.Value.Interface())
	sv.OnChange(func(e events.Event) {
		// vv.UpdateWidget() // not needed?
		vv.SendChange()
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

func (vv *SliceValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *SliceValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	npv := laser.NonPtrValue(vv.Value)
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

func (vv *SliceValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	slci := vv.Value.Interface()
	vv.IsArray = laser.NonPtrType(reflect.TypeOf(slci)).Kind() == reflect.Array
	if slci != nil && !laser.AnyIsNil(slci) {
		vv.ElType = laser.SliceElType(slci)
		vv.ElIsStruct = (laser.NonPtrType(vv.ElType).Kind() == reflect.Struct)
	}
	ac := vv.Widget.(*gi.Button)
	ac.Icon = icons.Edit
	ac.Tooltip, _ = vv.Tag("desc")
	ac.OnClick(func(e events.Event) {
		vv.OpenDialog(ac, nil)
	})
	vv.UpdateWidget()
}

func (vv *SliceValueView) HasButton() bool {
	return true
}

func (vv *SliceValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	title, newPath, isZero := vv.Label()
	if isZero {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	desc, _ := vv.Tag("desc")
	vvp := laser.OnePtrValue(vv.Value)
	if vvp.Kind() != reflect.Ptr {
		log.Printf("giv.SliceValueView: Cannot view slices with non-pointer struct elements\n")
		return
	}
	inact := vv.This().(ValueView).IsInactive()
	slci := vvp.Interface()
	if !vv.IsArray && vv.ElIsStruct {
		dlg := TableViewDialog(vv.Widget, DlgOpts{Title: title, Prompt: desc, TmpSave: vv.TmpSave, Inactive: inact, ViewPath: vpath}, slci, nil, nil)
		svk := dlg.Stage.Scene.ChildByType(TableViewType, ki.Embeds, 2)
		if svk != nil {
			sv := svk.(*TableView)
			sv.SliceValView = vv
			sv.OnChange(func(e events.Event) {
				vv.UpdateWidget()
				vv.SendChange()
			})
		}
	} else {
		dlg := SliceViewDialog(vv.Widget, DlgOpts{Title: title, Prompt: desc, TmpSave: vv.TmpSave, Inactive: inact, ViewPath: vpath}, slci, nil, nil)
		svk := dlg.Stage.Scene.ChildByType(SliceViewType, ki.Embeds, 2)
		if svk != nil {
			sv := svk.(*SliceView)
			sv.SliceValView = vv
			sv.OnChange(func(e events.Event) {
				vv.UpdateWidget()
				vv.SendChange()
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

func (vv *SliceInlineValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = SliceViewInlineType
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

func (vv *SliceInlineValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sv := vv.Widget.(*SliceViewInline)
	sv.Tooltip, _ = vv.Tag("desc")
	sv.SliceValView = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	// npv := vv.Value.Elem()
	sv.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)
	sv.SetSlice(vv.Value.Interface())
	sv.OnChange(func(e events.Event) {
		vv.UpdateWidget()
		vv.SendChange()
	})
}

////////////////////////////////////////////////////////////////////////////////////////
//  MapValueView

// MapValueView presents a button to edit maps
type MapValueView struct {
	ValueViewBase
}

func (vv *MapValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *MapValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ac := vv.Widget.(*gi.Button)
	npv := laser.NonPtrValue(vv.Value)
	mpi := vv.Value.Interface()
	txt := ""
	if npv.Kind() == reflect.Interface {
		txt = fmt.Sprintf("Map: %T", npv.Interface())
	} else {
		txt = fmt.Sprintf("Map: [%v %v]%v", npv.Len(), laser.MapKeyType(mpi).String(), laser.MapValueType(mpi).String())
	}
	ac.SetText(txt)
}

func (vv *MapValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	ac := vv.Widget.(*gi.Button)
	ac.Icon = icons.Edit
	ac.Tooltip, _ = vv.Tag("desc")
	ac.OnClick(func(e events.Event) {
		vv.OpenDialog(ac, nil)
	})
	vv.UpdateWidget()
}

func (vv *MapValueView) HasButton() bool {
	return true
}

func (vv *MapValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	title, newPath, isZero := vv.Label()
	if isZero {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	desc, _ := vv.Tag("desc")
	mpi := vv.Value.Interface()
	inact := vv.This().(ValueView).IsInactive()
	dlg := MapViewDialog(vv.Widget, DlgOpts{Title: title, Prompt: desc, TmpSave: vv.TmpSave, Inactive: inact, ViewPath: vpath}, mpi, fun)
	mvk := dlg.Stage.Scene.ChildByType(MapViewType, ki.Embeds, 2)
	if mvk != nil {
		mv := mvk.(*MapView)
		mv.MapValView = vv
		mv.OnChange(func(e events.Event) {
			vv.UpdateWidget()
			vv.SendChange()
		})
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  MapInlineValueView

// MapInlineValueView presents a MapViewInline for a map
type MapInlineValueView struct {
	ValueViewBase
}

func (vv *MapInlineValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = MapViewInlineType
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

func (vv *MapInlineValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sv := vv.Widget.(*MapViewInline)
	sv.Tooltip, _ = vv.Tag("desc")
	sv.MapValView = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	// npv := vv.Value.Elem()
	sv.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)
	sv.SetMap(vv.Value.Interface())
	sv.OnChange(func(e events.Event) {
		vv.UpdateWidget()
		vv.SendChange()
	})
}

////////////////////////////////////////////////////////////////////////////////////////
//  KiPtrValueView

// KiPtrValueView provides a chooser for pointers to Ki objects
type KiPtrValueView struct {
	ValueViewBase
}

func (vv *KiPtrValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
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
	mb := vv.Widget.(*gi.Button)
	path := "nil"
	k := vv.KiStruct()
	if k != nil {
		path = k.Path()
	}
	mb.SetText(path)
}

func (vv *KiPtrValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	mb := vv.Widget.(*gi.Button)
	mb.Indicator = icons.KeyboardArrowDown
	mb.Tooltip, _ = vv.Tag("desc")
	mb.ResetMenu()
	mb.Menu.AddButton(gi.ActOpts{Label: "Edit"}, func(bt *gi.Button) {
		k := vv.KiStruct()
		if k != nil {
			mb := vv.Widget.(*gi.Button)
			vv.OpenDialog(mb, nil)
		}
	})
	mb.Menu.AddButton(gi.ActOpts{Label: "GoGiEditor"}, func(bt *gi.Button) {
		k := vv.KiStruct()
		if k != nil {
			GoGiEditorDialog(k)
		}
	})
	vv.UpdateWidget()
}

func (vv *KiPtrValueView) HasButton() bool {
	return true
}

func (vv *KiPtrValueView) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
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
	StructViewDialog(ctx, DlgOpts{Title: title, Prompt: desc, TmpSave: vv.TmpSave, Inactive: inact, ViewPath: vpath}, k, fun)
}

////////////////////////////////////////////////////////////////////////////////////////
//  BoolValueView

// BoolValueView presents a checkbox for a boolean
type BoolValueView struct {
	ValueViewBase
}

func (vv *BoolValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.SwitchType
	return vv.WidgetTyp
}

func (vv *BoolValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	cb := vv.Widget.(*gi.Switch)
	npv := laser.NonPtrValue(vv.Value)
	bv, _ := laser.ToBool(npv.Interface())
	cb.SetState(bv, states.Checked)
}

func (vv *BoolValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	cb := vv.Widget.(*gi.Switch)
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)
	cb.OnChange(func(e events.Event) {
		if vv.SetValue(cb.StateIs(states.Checked)) {
			vv.UpdateWidget() // always update after setting value..
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

func (vv *IntValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.SpinBoxType
	return vv.WidgetTyp
}

func (vv *IntValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.SpinBox)
	npv := laser.NonPtrValue(vv.Value)
	fv, err := laser.ToFloat32(npv.Interface())
	if err != nil {
		sb.SetValue(fv)
	}
}

func (vv *IntValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sb := vv.Widget.(*gi.SpinBox)
	sb.Tooltip, _ = vv.Tag("desc")
	sb.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)
	sb.Step = 1.0
	sb.PageStep = 10.0
	// STYTODO: figure out what to do about this
	// sb.Parts.AddChildStyler("textfield", 0, gi.StylerParent(vv), func(tf *gi.WidgetBase) {
	// 	s.Width.SetCh(5)
	// })
	vk := vv.Value.Kind()
	if vk >= reflect.Uint && vk <= reflect.Uint64 {
		sb.SetMin(0)
	}
	if mintag, ok := vv.Tag("min"); ok {
		minv, err := laser.ToFloat32(mintag)
		if err != nil {
			sb.SetMin(minv)
		}
	}
	if maxtag, ok := vv.Tag("max"); ok {
		maxv, err := laser.ToFloat32(maxtag)
		if err != nil {
			sb.SetMax(maxv)
		}
	}
	if steptag, ok := vv.Tag("step"); ok {
		step, err := laser.ToFloat32(steptag)
		if err != nil {
			sb.Step = step
		}
	}
	if fmttag, ok := vv.Tag("format"); ok {
		sb.Format = fmttag
	}
	sb.OnChange(func(e events.Event) {
		if vv.SetValue(sb.Value) {
			vv.UpdateWidget()
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

func (vv *FloatValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.SpinBoxType
	return vv.WidgetTyp
}

func (vv *FloatValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.SpinBox)
	npv := laser.NonPtrValue(vv.Value)
	fv, err := laser.ToFloat32(npv.Interface())
	if err != nil {
		sb.SetValue(fv)
	}
}

func (vv *FloatValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sb := vv.Widget.(*gi.SpinBox)
	sb.Tooltip, _ = vv.Tag("desc")
	sb.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)
	sb.Step = 1.0
	sb.PageStep = 10.0
	if mintag, ok := vv.Tag("min"); ok {
		minv, err := laser.ToFloat32(mintag)
		if err != nil {
			sb.HasMin = true
			sb.Min = minv
		}
	}
	if maxtag, ok := vv.Tag("max"); ok {
		maxv, err := laser.ToFloat32(maxtag)
		if err != nil {
			sb.HasMax = true
			sb.Max = maxv
		}
	}
	sb.Step = .1 // smaller default
	if steptag, ok := vv.Tag("step"); ok {
		step, err := laser.ToFloat32(steptag)
		if err != nil {
			sb.Step = step
		}
	}
	if fmttag, ok := vv.Tag("format"); ok {
		sb.Format = fmttag
	}

	sb.OnChange(func(e events.Event) {
		if vv.SetValue(sb.Value) {
			vv.UpdateWidget()
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

func (vv *EnumValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ChooserType
	return vv.WidgetTyp
}

func (vv *EnumValueView) EnumValue() enums.Enum {
	ev, ok := vv.Value.Interface().(enums.Enum)
	if ok {
		return ev
	}
	slog.Error("giv.EnumValueView: type must be enums.Enum")
	return nil
}

func (vv *EnumValueView) SetEnumValueFromInt(ival int64) bool {
	// typ := vv.EnumType()
	// eval := laser.EnumIfaceFromInt64(ival, typ)
	return vv.SetValue(ival)
}

func (vv *EnumValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.Chooser)
	npv := laser.NonPtrValue(vv.Value)
	iv, err := laser.ToInt(npv.Interface())
	if err != nil {
		sb.SetCurIndex(int(iv)) // todo: currently only working for 0-based values
	}
}

func (vv *EnumValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	cb := vv.Widget.(*gi.Chooser)
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)

	ev := vv.EnumValue()
	cb.ItemsFromEnum(ev, false, 50)
	cb.OnChange(func(e events.Event) {
		cval := cb.CurVal.(enums.Enum)
		if vv.SetEnumValueFromInt(cval.Int64()) { // todo: using index
			vv.UpdateWidget()
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

func (vv *BitFlagView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonBoxType
	return vv.WidgetTyp
}

func (vv *BitFlagView) EnumValue() enums.BitFlag {
	ev, ok := vv.Value.Interface().(enums.BitFlag)
	if ok {
		return ev
	}
	slog.Error("giv.BitFlagView: type must be enums.BitFlag")
	return nil
}

func (vv *BitFlagView) SetEnumValueFromInt(ival int64) bool {
	// todo: needs to set flags?
	// typ := vv.EnumType()
	// eval := laser.EnumIfaceFromInt64(ival, typ)
	return vv.SetValue(ival)
}

func (vv *BitFlagView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bb := vv.Widget.(*gi.ButtonBox)
	_ = bb
	npv := laser.NonPtrValue(vv.Value)
	iv, err := laser.ToInt(npv.Interface())
	_ = iv
	if err != nil {
		// ev := vv.EnumValue() // todo:
		// bb.UpdateFromBitFlags(typ, int64(iv))
	}
}

func (vv *BitFlagView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	cb := vv.Widget.(*gi.ButtonBox)
	// vv.StdConfigWidget(cb.Parts)
	// cb.Parts.Lay = gi.LayoutHoriz
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)

	// todo!
	ev := vv.EnumValue()
	_ = ev
	// cb.ItemsFromEnum(ev)
	// cb.ConfigParts(sc)
	// cb.ButtonSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	vvv, _ := recv.Embed(TypeBitFlagView).(*BitFlagView)
	// 	cbb := vvv.Widget.(*gi.ButtonBox)
	// 	etyp := vvv.EnumType()
	// 	val := cbb.BitFlagsValue(etyp)
	// 	vvv.SetEnumValueFromInt(val)
	// 	// vvv.UpdateWidget()
	// })
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  TypeValueView

// TypeValueView presents a combobox for choosing types
type TypeValueView struct {
	ValueViewBase
}

func (vv *TypeValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ChooserType
	return vv.WidgetTyp
}

func (vv *TypeValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.Chooser)
	npv := laser.NonPtrValue(vv.Value)
	typ, ok := npv.Interface().(*gti.Type)
	if ok {
		sb.SetCurVal(typ)
	}
}

func (vv *TypeValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	cb := vv.Widget.(*gi.Chooser)
	cb.Tooltip, _ = vv.Tag("desc")
	cb.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)

	typEmbeds := ki.NodeType
	// if kiv, ok := vv.Owner.(ki.Ki); ok {
	// 	if tep, ok := kiv.PropInherit("type-embeds", ki.Inherit, ki.TypeProps); ok {
	// 		// todo:
	// 		// if te, ok := tep.(reflect.Type); ok {
	// 		// 	typEmbeds = te
	// 		// }
	// 	}
	// }
	if tetag, ok := vv.Tag("type-embeds"); ok {
		typ := gti.TypeByName(tetag)
		if typ != nil {
			typEmbeds = typ
		}
	}

	tl := gti.AllEmbeddersOf(typEmbeds)
	cb.ItemsFromTypes(tl, false, true, 50)

	cb.OnChange(func(e events.Event) {
		tval := cb.CurVal.(*gti.Type)
		if vv.SetValue(tval) {
			vv.UpdateWidget()
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

func (vv *ByteSliceValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.TextFieldType
	return vv.WidgetTyp
}

func (vv *ByteSliceValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	tf := vv.Widget.(*gi.TextField)
	npv := laser.NonPtrValue(vv.Value)
	bv, ok := npv.Interface().([]byte)
	if ok {
		tf.SetText(string(bv))
	}
}

func (vv *ByteSliceValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	tf := vv.Widget.(*gi.TextField)
	tf.Tooltip, _ = vv.Tag("desc")
	tf.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)
	// STYTODO: figure out how how to handle these kinds of styles
	tf.AddStyles(func(s *styles.Style) {
		s.MinWidth.SetCh(16)
		s.MaxWidth.SetDp(-1)
	})

	tf.OnChange(func(e events.Event) {
		if vv.SetValue(tf.Text()) {
			vv.UpdateWidget() // always update after setting value..
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

func (vv *RuneSliceValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.TextFieldType
	return vv.WidgetTyp
}

func (vv *RuneSliceValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	tf := vv.Widget.(*gi.TextField)
	npv := laser.NonPtrValue(vv.Value)
	rv, ok := npv.Interface().([]rune)
	if ok {
		tf.SetText(string(rv))
	}
}

func (vv *RuneSliceValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	tf := vv.Widget.(*gi.TextField)
	tf.Tooltip, _ = vv.Tag("desc")
	tf.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)
	tf.AddStyles(func(s *styles.Style) {
		s.MinWidth.SetCh(16)
		s.MaxWidth.SetDp(-1)
	})

	tf.OnChange(func(e events.Event) {
		if vv.SetValue(tf.Text()) {
			vv.UpdateWidget() // always update after setting value..
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

func (vv *NilValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.LabelType
	return vv.WidgetTyp
}

func (vv *NilValueView) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.Label)
	npv := laser.NonPtrValue(vv.Value)
	tstr := ""
	if !laser.ValueIsZero(npv) {
		tstr = npv.String() // npv.Type().String()
	} else if !laser.ValueIsZero(vv.Value) {
		tstr = vv.Value.String() // vv.Value.Type().String()
	}
	sb.SetText("nil " + tstr)
}

func (vv *NilValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	sb := vv.Widget.(*gi.Label)
	sb.Tooltip, _ = vv.Tag("desc")
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  TimeValueView

var DefaultTimeFormat = "2006-01-02 15:04:05 MST"

// TimeValueView presents a text field for a time
type TimeValueView struct {
	ValueViewBase
}

func (vv *TimeValueView) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.TextFieldType
	return vv.WidgetTyp
}

// TimeVal decodes Value into a *time.Time value -- also handles FileTime case
func (vv *TimeValueView) TimeVal() *time.Time {
	tmi := laser.PtrValue(vv.Value).Interface()
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

func (vv *TimeValueView) ConfigWidget(widg gi.Widget) {
	vv.Widget = widg
	vv.StdConfigWidget(widg)
	tf := vv.Widget.(*gi.TextField)
	tf.SetStretchMaxWidth()
	tf.Tooltip, _ = vv.Tag("desc")
	tf.SetState(vv.This().(ValueView).IsInactive(), states.Disabled)
	tf.AddStyles(func(s *styles.Style) {
		tf.Style.MinWidth.SetCh(float32(len(DefaultTimeFormat) + 2))
	})
	tf.OnChange(func(e events.Event) {
		nt, err := time.Parse(DefaultTimeFormat, tf.Text())
		if err != nil {
			log.Println(err)
		} else {
			tm := vv.TimeVal()
			*tm = nt
			// vv.SendChange()
			vv.UpdateWidget()
		}
	})
	vv.UpdateWidget()
}
