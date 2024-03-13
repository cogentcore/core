// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/states"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// This file contains the standard [Value]s built into giv.

// StringValue represents any value with a text field.
type StringValue struct {
	ValueBase[*gi.TextField]
}

func (v *StringValue) Config() {
	if vtag, _ := v.Tag("view"); vtag == "password" {
		v.Widget.SetTypePassword()
	}
	if vl, ok := v.Value.Interface().(gi.Validator); ok {
		v.Widget.SetValidator(vl.Validate)
	}
	if fv, ok := v.Owner.(gi.FieldValidator); ok {
		v.Widget.SetValidator(func() error {
			return fv.ValidateField(v.Field.Name)
		})
	}

	v.Widget.OnFinal(events.Change, func(e events.Event) {
		if v.SetValue(v.Widget.Text()) {
			v.Update()
		}
	})
}

func (v *StringValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	if npv.Kind() == reflect.Interface && npv.IsZero() {
		v.Widget.SetText("None")
	} else {
		txt := laser.ToString(v.Value.Interface())
		v.Widget.SetText(txt)
	}
}

// StructValue represents a struct value with a button.
type StructValue struct {
	ValueBase[*gi.Button]
}

func (v *StructValue) Config() {
	v.Widget.SetType(gi.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogWidget(v, true)
}

func (v *StructValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	if v.Value.IsZero() || npv.IsZero() {
		v.Widget.SetText("None")
	} else {
		opv := laser.OnePtrUnderlyingValue(v.Value)
		if lbler, ok := opv.Interface().(gi.Labeler); ok {
			v.Widget.SetText(lbler.Label())
		} else {
			v.Widget.SetText(laser.FriendlyTypeName(npv.Type()))
		}
	}
	v.Widget.Update()
}

func (v *StructValue) ConfigDialog(d *gi.Body) (bool, func()) {
	if v.Value.IsZero() || laser.NonPtrValue(v.Value).IsZero() {
		return false, nil
	}
	opv := laser.OnePtrUnderlyingValue(v.Value)
	str := opv.Interface()
	NewStructView(d).SetStruct(str).SetViewPath(v.ViewPath).SetTmpSave(v.TmpSave).
		SetReadOnly(v.IsReadOnly())
	if tb, ok := str.(gi.Toolbarer); ok {
		d.AddAppBar(tb.ConfigToolbar)
	}
	return true, nil
}

// StructInlineValue represents a struct value with a [StructViewInline].
type StructInlineValue struct {
	ValueBase[*StructViewInline]
}

func (v *StructInlineValue) Config() {
	v.Widget.StructValue = v
	v.Widget.ViewPath = v.ViewPath
	v.Widget.TmpSave = v.TmpSave
	v.Widget.SetStruct(v.Value.Interface())
	v.Widget.OnChange(func(e events.Event) {
		v.SendChange(e)
	})
}

func (v *StructInlineValue) Update() {
	v.Widget.SetStruct(v.Value.Interface())
}

// SliceValue represents a slice or array value with a button.
type SliceValue struct {
	ValueBase[*gi.Button]
}

func (v *SliceValue) Config() {
	v.Widget.SetType(gi.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogWidget(v, true)
}

func (v *SliceValue) Update() {
	npv := laser.OnePtrUnderlyingValue(v.Value).Elem()
	txt := ""
	if !npv.IsValid() {
		txt = "None"
	} else {
		if npv.Kind() == reflect.Array || !npv.IsNil() {
			bnm := laser.FriendlyTypeName(laser.SliceElType(v.Value.Interface()))
			if strings.HasSuffix(bnm, "s") {
				txt = strcase.ToSentence(fmt.Sprintf("%d lists of %s", npv.Len(), bnm))
			} else {
				txt = strcase.ToSentence(fmt.Sprintf("%d %ss", npv.Len(), bnm))
			}
		} else {
			txt = "None"
		}
	}
	v.Widget.SetText(txt).Update()
}

func (v *SliceValue) ConfigDialog(d *gi.Body) (bool, func()) {
	npv := laser.NonPtrValue(v.Value)
	if v.Value.IsZero() || npv.IsZero() {
		return false, nil
	}
	vvp := laser.OnePtrValue(v.Value)
	if vvp.Kind() != reflect.Ptr {
		slog.Error("giv.SliceValue: Cannot view unadressable (non-pointer) slices", "type", v.Value.Type())
		return false, nil
	}
	slci := vvp.Interface()
	if npv.Kind() != reflect.Array && laser.NonPtrType(laser.SliceElType(v.Value.Interface())).Kind() == reflect.Struct {
		tv := NewTableView(d).SetSlice(slci).SetTmpSave(v.TmpSave).SetViewPath(v.ViewPath)
		tv.SetReadOnly(v.IsReadOnly())
		d.AddAppBar(tv.ConfigToolbar)
	} else {
		sv := NewSliceView(d).SetSlice(slci).SetTmpSave(v.TmpSave).SetViewPath(v.ViewPath)
		sv.SetReadOnly(v.IsReadOnly())
		d.AddAppBar(sv.ConfigToolbar)
	}
	return true, nil
}

// SliceInlineValue represents a slice or array value with a [SliceViewInline].
type SliceInlineValue struct {
	ValueBase[*SliceViewInline]
}

func (v *SliceInlineValue) Config() {
	v.Widget.SliceValue = v
	v.Widget.ViewPath = v.ViewPath
	v.Widget.TmpSave = v.TmpSave
	v.Widget.SetSlice(v.Value.Interface())
	v.Widget.OnChange(func(e events.Event) {
		v.SendChange(e)
	})
}

func (v *SliceInlineValue) Update() {
	csl := v.Value.Interface()
	newslc := false
	if reflect.TypeOf(v.Value).Kind() != reflect.Pointer { // prevent crash on non-comparable
		newslc = true
	} else {
		newslc = v.Widget.Slice != csl
	}
	if newslc {
		v.Widget.SetSlice(csl)
	} else {
		v.Widget.Update()
	}
}

// MapValue represents a map value with a button.
type MapValue struct {
	ValueBase[*gi.Button]
}

func (v *MapValue) Config() {
	v.Widget.SetType(gi.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogWidget(v, true)
}

func (v *MapValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	mpi := v.Value.Interface()
	txt := ""
	if !npv.IsValid() || npv.IsNil() {
		txt = "None"
	} else {
		bnm := laser.FriendlyTypeName(laser.MapValueType(mpi))
		if strings.HasSuffix(bnm, "s") {
			txt = strcase.ToSentence(fmt.Sprintf("%d lists of %s", npv.Len(), bnm))
		} else {
			txt = strcase.ToSentence(fmt.Sprintf("%d %ss", npv.Len(), bnm))
		}
	}
	v.Widget.SetText(txt).Update()
}

func (v *MapValue) ConfigDialog(d *gi.Body) (bool, func()) {
	if v.Value.IsZero() || laser.NonPtrValue(v.Value).IsZero() {
		return false, nil
	}
	mpi := v.Value.Interface()
	mv := NewMapView(d).SetMap(mpi)
	mv.SetViewPath(v.ViewPath).SetTmpSave(v.TmpSave).SetReadOnly(v.IsReadOnly())
	d.AddAppBar(mv.ConfigToolbar)
	return true, nil
}

// MapInlineValue represents a map value with a [MapViewInline].
type MapInlineValue struct {
	ValueBase[*MapViewInline]
}

func (v *MapInlineValue) Config() {
	v.Widget.MapValue = v
	v.Widget.ViewPath = v.ViewPath
	v.Widget.TmpSave = v.TmpSave
	v.Widget.SetMap(v.Value.Interface())
	v.Widget.OnChange(func(e events.Event) {
		v.SendChange(e)
	})
}

func (v *MapInlineValue) Update() {
	cmp := v.Value.Interface()
	if v.Widget.Map != cmp {
		v.Widget.SetMap(cmp)
	} else {
		v.Widget.UpdateValues()
	}
}

//////////////////////////////////////////////////////////////////////////////
//  KiPtrValue

// KiValue provides a button for inspecting pointers to Ki objects
type KiValue struct {
	ValueBase
}

func (vv *KiValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

// get the Ki struct itself (or nil)
func (vv *KiValue) KiStruct() ki.Ki {
	if !vv.Value.IsValid() {
		return nil
	}
	if vv.Value.IsNil() {
		return nil
	}
	opv := laser.OnePtrValue(vv.Value)
	if opv.IsNil() {
		return nil
	}
	k, ok := opv.Interface().(ki.Ki)
	if ok && k != nil {
		return k
	}
	return nil
}

func (vv *KiValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	path := "None"
	k := vv.KiStruct()
	if k != nil && k.This() != nil {
		path = k.AsKi().String()
	}
	bt.SetText(path).Update()
}

func (vv *KiValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	ConfigDialogWidget(vv, bt, true)
	vv.UpdateWidget()
}

func (vv *KiValue) HasDialog() bool                      { return true }
func (vv *KiValue) OpenDialog(ctx gi.Widget, fun func()) { OpenValueDialog(vv, ctx, fun) }

func (vv *KiValue) ConfigDialog(d *gi.Body) (bool, func()) {
	k := vv.KiStruct()
	if k == nil {
		return false, nil
	}
	InspectorView(d, k)
	return true, nil
}

//////////////////////////////////////////////////////////////////////////////
//  BoolValue

// BoolValue presents a checkbox for a boolean
type BoolValue struct {
	ValueBase
}

func (vv *BoolValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.SwitchType
	return vv.WidgetTyp
}

func (vv *BoolValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	cb := vv.Widget.(*gi.Switch)
	npv := laser.NonPtrValue(vv.Value)
	bv, _ := laser.ToBool(npv.Interface())
	cb.SetState(bv, states.Checked)
}

func (vv *BoolValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	cb := vv.Widget.(*gi.Switch)
	cb.Tooltip = vv.Doc()
	cb.OnFinal(events.Change, func(e events.Event) {
		vv.SetValue(cb.IsChecked())
	})
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  IntValue

// IntValue presents a spinner
type IntValue struct {
	ValueBase
}

func (vv *IntValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.SpinnerType
	return vv.WidgetTyp
}

func (vv *IntValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.Spinner)
	npv := laser.NonPtrValue(vv.Value)
	fv, err := laser.ToFloat32(npv.Interface())
	if err == nil {
		sb.SetValue(fv)
	} else {
		slog.Error("IntValue set", "error", err)
	}
}

func (vv *IntValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	sb := vv.Widget.(*gi.Spinner)
	sb.Tooltip = vv.Doc()
	sb.Step = 1.0
	sb.PageStep = 10.0
	// STYTODO: figure out what to do about this
	// sb.Parts.AddChildStyler("textfield", 0, gi.StylerParent(vv), func(tf *gi.WidgetBase) {
	// 	s.Min.X.SetCh(5)
	// })
	vk := vv.Value.Kind()
	if vk >= reflect.Uint && vk <= reflect.Uint64 {
		sb.SetMin(0)
	}
	if mintag, ok := vv.Tag("min"); ok {
		minv, err := laser.ToFloat32(mintag)
		if err == nil {
			sb.SetMin(minv)
		} else {
			slog.Error("Int Min Value:", "error:", err)
		}
	}
	if maxtag, ok := vv.Tag("max"); ok {
		maxv, err := laser.ToFloat32(maxtag)
		if err == nil {
			sb.SetMax(maxv)
		} else {
			slog.Error("Int Max Value:", "error:", err)
		}
	}
	if steptag, ok := vv.Tag("step"); ok {
		step, err := laser.ToFloat32(steptag)
		if err == nil {
			sb.Step = step
		} else {
			slog.Error("Int Step Value:", "error:", err)
		}
	}
	if fmttag, ok := vv.Tag("format"); ok {
		sb.Format = fmttag
	}
	sb.OnFinal(events.Change, func(e events.Event) {
		vv.SetValue(sb.Value)
	})
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  FloatValue

// FloatValue presents a spinner
type FloatValue struct {
	ValueBase
}

func (vv *FloatValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.SpinnerType
	return vv.WidgetTyp
}

func (vv *FloatValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.Spinner)
	npv := laser.NonPtrValue(vv.Value)
	fv, err := laser.ToFloat32(npv.Interface())
	if err == nil {
		sb.SetValue(fv)
	} else {
		slog.Error("Float Value set", "error:", err)
	}
}

func (vv *FloatValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	sb := vv.Widget.(*gi.Spinner)
	sb.Tooltip = vv.Doc()
	sb.PageStep = 10.0
	if mintag, ok := vv.Tag("min"); ok {
		minv, err := laser.ToFloat32(mintag)
		if err == nil {
			sb.HasMin = true
			sb.Min = minv
		} else {
			slog.Error("Invalid float min value", "value", mintag, "err", err)
		}
	}
	if maxtag, ok := vv.Tag("max"); ok {
		maxv, err := laser.ToFloat32(maxtag)
		if err == nil {
			sb.HasMax = true
			sb.Max = maxv
		} else {
			slog.Error("Invalid float max value", "value", maxtag, "err", err)
		}
	}
	sb.Step = .1 // smaller default
	if steptag, ok := vv.Tag("step"); ok {
		step, err := laser.ToFloat32(steptag)
		if err == nil {
			sb.Step = step
		} else {
			slog.Error("Invalid float step value", "value", steptag, "err", err)
		}
	}
	if fmttag, ok := vv.Tag("format"); ok {
		sb.Format = fmttag
	}
	sb.OnFinal(events.Change, func(e events.Event) {
		vv.SetValue(sb.Value)
	})
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  SliderValue

// SliderValue presents a slider
type SliderValue struct {
	ValueBase
}

func (vv *SliderValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.SliderType
	return vv.WidgetTyp
}

func (vv *SliderValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sl := vv.Widget.(*gi.Slider)
	npv := laser.NonPtrValue(vv.Value)
	fv, err := laser.ToFloat32(npv.Interface())
	if err == nil {
		sl.SetValue(fv)
	} else {
		slog.Error("Float Value set", "error:", err)
	}
}

func (vv *SliderValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	sl := vv.Widget.(*gi.Slider)
	sl.Tooltip = vv.Doc()

	if mintag, ok := vv.Tag("min"); ok {
		minv, err := laser.ToFloat32(mintag)
		if err == nil {
			sl.Min = minv
		} else {
			slog.Error("Float Min Value:", "error:", err)
		}
	}
	if maxtag, ok := vv.Tag("max"); ok {
		maxv, err := laser.ToFloat32(maxtag)
		if err == nil {
			sl.Max = maxv
		} else {
			slog.Error("Float Max Value:", "error:", err)
		}
	}
	if steptag, ok := vv.Tag("step"); ok {
		step, err := laser.ToFloat32(steptag)
		if err == nil {
			sl.Step = step
		} else {
			slog.Error("Float Step Value:", "error:", err)
		}
	}
	sl.OnFinal(events.Change, func(e events.Event) {
		vv.SetValue(sl.Value)
	})
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  EnumValue

// EnumValue presents a chooser for choosing enums
type EnumValue struct {
	ValueBase
}

func (vv *EnumValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ChooserType
	return vv.WidgetTyp
}

func (vv *EnumValue) EnumValue() enums.Enum {
	ev, ok := laser.OnePtrUnderlyingValue(vv.Value).Interface().(enums.Enum)
	if ok {
		return ev
	}
	slog.Error("giv.EnumValue: type must be enums.Enum", "was", vv.Value.Type())
	return nil
}

// func (vv *EnumValue) SetEnumValueFromInt(ival int64) bool {
// 	// typ := vv.EnumType()
// 	// eval := laser.EnumIfaceFromInt64(ival, typ)
// 	return vv.SetValue(ival)
// }

func (vv *EnumValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	ch := vv.Widget.(*gi.Chooser)
	npv := laser.NonPtrValue(vv.Value)
	ch.SetCurrentValue(npv.Interface())

	// iv, err := laser.ToInt(npv.Interface())
	// if err == nil {
	// 	ch.SetCurIndex(int(iv)) // todo: currently only working for 0-based values
	// } else {
	// 	slog.Error("Enum Value:", err)
	// }
}

func (vv *EnumValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	ch := vv.Widget.(*gi.Chooser)
	ch.Tooltip = vv.Doc()

	ev := vv.EnumValue()
	ch.SetEnum(ev)
	ch.OnFinal(events.Change, func(e events.Event) {
		vv.SetValue(ch.CurrentItem.Value)
	})
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  BitFlagView

// BitFlagValue presents chip [gi.Switches] for editing bitflags
type BitFlagValue struct {
	ValueBase
}

func (vv *BitFlagValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.SwitchesType
	return vv.WidgetTyp
}

func (vv *BitFlagValue) EnumValue() enums.BitFlagSetter {
	ev, ok := vv.Value.Interface().(enums.BitFlagSetter)
	if !ok {
		slog.Error("giv.BitFlagView: type must be enums.BitFlag")
		return nil
	}
	// special case to use [ki.Ki.FlagType] if we are the Flags field
	if vv.Field != nil && vv.Field.Name == "Flags" {
		if k, ok := vv.Owner.(ki.Ki); ok {
			return k.FlagType()
		}
	}
	return ev
}

func (vv *BitFlagValue) SetEnumValueFromInt(ival int64) bool {
	return vv.SetValue(ival)
}

func (vv *BitFlagValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sw := vv.Widget.(*gi.Switches)
	ev := vv.EnumValue()
	sw.UpdateFromBitFlag(ev)
}

func (vv *BitFlagValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	sw := vv.Widget.(*gi.Switches)
	sw.SetType(gi.SwitchChip)
	sw.Tooltip = vv.Doc()

	ev := vv.EnumValue()
	sw.SetEnum(ev)
	sw.OnChange(func(e events.Event) {
		sw.BitFlagValue(vv.EnumValue())
	})
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  TypeValue

// TypeValue presents a chooser for choosing types
type TypeValue struct {
	ValueBase
}

func (vv *TypeValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ChooserType
	return vv.WidgetTyp
}

func (vv *TypeValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*gi.Chooser)
	npv := laser.OnePtrValue(vv.Value)
	typ, ok := npv.Interface().(*gti.Type)
	if ok {
		sb.SetCurrentValue(typ)
	}
}

func (vv *TypeValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	cb := vv.Widget.(*gi.Chooser)
	cb.Tooltip = vv.Doc()

	// typEmbeds := ki.NodeType
	typEmbeds := gi.WidgetBaseType
	if tetag, ok := vv.Tag("type-embeds"); ok {
		typ := gti.TypeByName(tetag)
		if typ != nil {
			typEmbeds = typ
		}
	}

	tl := gti.AllEmbeddersOf(typEmbeds)
	cb.SetTypes(tl)
	cb.OnFinal(events.Change, func(e events.Event) {
		tval := cb.CurrentItem.Value.(*gti.Type)
		vv.SetValue(tval)
	})
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  ByteSliceValue

// ByteSliceValue presents a textfield of the bytes
type ByteSliceValue struct {
	ValueBase
}

func (vv *ByteSliceValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.TextFieldType
	return vv.WidgetTyp
}

func (vv *ByteSliceValue) UpdateWidget() {
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

func (vv *ByteSliceValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	tf := vv.Widget.(*gi.TextField)
	tf.Tooltip = vv.Doc()
	// STYTODO: figure out how how to handle these kinds of styles
	tf.Style(func(s *styles.Style) {
		s.Min.X.Ch(16)
	})
	vv.StdConfig(w)

	tf.OnFinal(events.Change, func(e events.Event) {
		vv.SetValue(tf.Text())
	})
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  RuneSliceValue

// RuneSliceValue presents a textfield of the bytes
type RuneSliceValue struct {
	ValueBase
}

func (vv *RuneSliceValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.TextFieldType
	return vv.WidgetTyp
}

func (vv *RuneSliceValue) UpdateWidget() {
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

func (vv *RuneSliceValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	tf := vv.Widget.(*gi.TextField)
	tf.Tooltip = vv.Doc()
	tf.Style(func(s *styles.Style) {
		s.Min.X.Ch(16)
	})
	vv.StdConfig(w)

	tf.OnFinal(events.Change, func(e events.Event) {
		vv.SetValue(tf.Text())
	})
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  NilValue

// NilValue presents a label saying 'nil' -- for any nil or otherwise unrepresentable items
type NilValue struct {
	ValueBase
}

func (vv *NilValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.LabelType
	return vv.WidgetTyp
}

func (vv *NilValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	lb := vv.Widget.(*gi.Label)
	lb.SetText("None")
}

func (vv *NilValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	lb := vv.Widget.(*gi.Label)
	lb.Tooltip = vv.Doc()
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  IconValue

// IconValue presents an action for displaying an IconName and selecting
// icons from IconChooserDialog
type IconValue struct {
	ValueBase
}

func (vv *IconValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *IconValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if icons.Icon(txt).IsNil() {
		bt.SetIcon(icons.Blank)
	} else {
		bt.SetIcon(icons.Icon(txt))
	}
	if sntag, ok := vv.Tag("view"); ok {
		if strings.Contains(sntag, "show-name") {
			if txt == "" {
				txt = "None"
			}
			bt.SetText(strcase.ToSentence(txt))
		}
	}
	bt.Update()
}

func (vv *IconValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	ConfigDialogWidget(vv, bt, false)
	vv.UpdateWidget()
}

func (vv *IconValue) HasDialog() bool { return true }
func (vv *IconValue) OpenDialog(ctx gi.Widget, fun func()) {
	OpenValueDialog(vv, ctx, fun, "Select an icon")
}

func (vv *IconValue) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	ics := icons.All()
	cur := icons.Icon(laser.ToString(vv.Value.Interface()))
	NewSliceView(d).SetStyleFunc(func(w gi.Widget, s *styles.Style, row int) {
		w.(*gi.Button).SetText(strcase.ToSentence(string(ics[row])))
	}).SetSlice(&ics).SetSelVal(cur).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			ic := icons.AllIcons[si]
			vv.SetValue(ic)
			vv.UpdateWidget()
		}
	}
}

//////////////////////////////////////////////////////////////////////////////
//  FontValue

// FontValue presents an action for displaying a FontName and selecting
// fonts from FontChooserDialog
type FontValue struct {
	ValueBase
}

func (vv *FontValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *FontValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	bt.SetText(txt).Update()
}

func (vv *FontValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Style(func(s *styles.Style) {
		// TODO(kai): fix this not working (probably due to medium font weight)
		s.Font.Family = laser.ToString(vv.Value.Interface())
	})
	ConfigDialogWidget(vv, bt, false)
	vv.UpdateWidget()
}

func (vv *FontValue) HasDialog() bool { return true }
func (vv *FontValue) OpenDialog(ctx gi.Widget, fun func()) {
	OpenValueDialog(vv, ctx, fun, "Select a font")
}

// show fonts in a bigger size so you can actually see the differences
var FontChooserSize = units.Pt(18)

func (vv *FontValue) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	wb := vv.Widget.AsWidget()
	FontChooserSize.ToDots(&wb.Styles.UnitContext)
	paint.FontLibrary.OpenAllFonts(int(FontChooserSize.Dots))
	fi := paint.FontLibrary.FontInfo
	cur := gi.FontName(laser.ToString(vv.Value.Interface()))
	NewTableView(d).SetStyleFunc(func(w gi.Widget, s *styles.Style, row, col int) {
		if col != 4 {
			return
		}
		s.Font.Family = fi[row].Name
		s.Font.Stretch = fi[row].Stretch
		s.Font.Weight = fi[row].Weight
		s.Font.Style = fi[row].Style
		s.Font.Size = FontChooserSize
	}).SetSlice(&fi).SetSelVal(cur).SetSelField("Name").BindSelect(&si)

	return true, func() {
		if si >= 0 {
			fi := paint.FontLibrary.FontInfo[si]
			vv.SetValue(fi.Name)
			vv.UpdateWidget()
		}
	}
}

//////////////////////////////////////////////////////////////////////////////
//  FileValue

// FileValue presents an action for displaying a Filename and selecting
// icons from FileChooserDialog
type FileValue struct {
	ValueBase
}

func (vv *FileValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *FileValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(click to open file chooser)"
	}
	prev := bt.Text
	bt.SetText(txt)
	if txt != prev {
		bt.Update()
	}
}

func (vv *FileValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	ConfigDialogWidget(vv, bt, false)
	vv.UpdateWidget()
}

func (vv *FileValue) HasDialog() bool                      { return true }
func (vv *FileValue) OpenDialog(ctx gi.Widget, fun func()) { OpenValueDialog(vv, ctx, fun) }

func (vv *FileValue) ConfigDialog(d *gi.Body) (bool, func()) {
	vv.SetFlag(true, ValueDialogNewWindow) // default to new window on supported platforms
	cur := laser.ToString(vv.Value.Interface())
	ext, _ := vv.Tag("ext")
	fv := NewFileView(d).SetFilename(cur, ext)
	d.AddAppBar(fv.ConfigToolbar)
	return true, func() {
		cur = fv.SelectedFile()
		vv.SetValue(cur)
		vv.UpdateWidget()
	}
}

//////////////////////////////////////////////////////////////////////////////
//  FuncValue

// FuncValue presents a [FuncButton] for viewing the information of and calling a function
type FuncValue struct {
	ValueBase
}

func (vv *FuncValue) WidgetType() *gti.Type {
	vv.WidgetTyp = FuncButtonType
	return vv.WidgetTyp
}

func (vv *FuncValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	fbt := vv.Widget.(*FuncButton)
	fun := laser.NonPtrValue(vv.Value).Interface()
	// if someone is viewing an arbitrary function, there is a good chance
	// that it is not added to gti (and that is out of their control)
	// (eg: in the inspector).
	fbt.SetWarnUnadded(false)
	fbt.SetFunc(fun)
}

func (vv *FuncValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)

	fbt := vv.Widget.(*FuncButton)
	fbt.Type = gi.ButtonTonal

	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  OptionValue

// OptionValue presents an [option.Option]
type OptionValue struct {
	ValueBase
}

func (vv *OptionValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.FrameType
	return vv.WidgetTyp
}

func (vv *OptionValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
}

func (vv *OptionValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)

	fr := vv.Widget.(*gi.Frame)

	gi.NewButton(fr, "unset").SetText("Unset")
	val := vv.Value.FieldByName("Value").Interface()
	NewValue(fr, val, "value")

	vv.UpdateWidget()
}
