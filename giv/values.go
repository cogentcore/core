// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"image/color"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/states"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// values contains all the Values for basic builtin types

func init() {
	gi.SettingsWindow = SettingsWindow
	gi.InspectorWindow = InspectorWindow

	ValueMapAdd(icons.Icon(""), func() Value {
		return &IconValue{}
	})
	ValueMapAdd(gi.FontName(""), func() Value {
		return &FontValue{}
	})
	ValueMapAdd(gi.Filename(""), func() Value {
		return &FileValue{}
	})
	ValueMapAdd(keyfun.MapName(""), func() Value {
		return &KeyMapValue{}
	})
	ValueMapAdd(gti.Type{}, func() Value {
		return &TypeValue{}
	})
	ValueMapAdd(color.RGBA{}, func() Value {
		return &ColorValue{}
	})
	ValueMapAdd(key.Chord(""), func() Value {
		return &KeyChordValue{}
	})
	ValueMapAdd(time.Time{}, func() Value {
		return &TimeValue{}
	})
	ValueMapAdd(time.Duration(0), func() Value {
		return &DurationValue{}
	})
	ValueMapAdd(fi.FileTime{}, func() Value {
		return &TimeValue{}
	})
}

//////////////////////////////////////////////////////////////////////////////
//  Valuer -- an interface for selecting Value GUI representation of types

// Valuer interface supplies the appropriate type of Value -- called
// on a given receiver item if defined for that receiver type (tries both
// pointer and non-pointer receivers) -- can use this for custom types to
// provide alternative custom interfaces -- must call Init on Value before
// returning it
type Valuer interface {
	Value() Value
}

// example implementation of Valuer interface -- can't implement on
// non-local types, so all the basic types are handled separately:
//
// func (s string) Value() Value {
// 	return &ValueBase{}
// }

// FieldValuer interface supplies the appropriate type of Value for a
// given field name and current field value on the receiver parent struct --
// called on a given receiver struct if defined for that receiver type (tries
// both pointer and non-pointer receivers) -- if a struct implements this
// interface, then it is used first for structs -- return nil to fall back on
// the default ToValue result
type FieldValuer interface {
	FieldValue(field string, fval any) Value
}

//////////////////////////////////////////////////////////////////////////////
//  ValueMap -- alternative way to connect value view with type

// ValueFunc is a function that returns a new initialized Value
// of an appropriate type as registered in the ValueMap
type ValueFunc func() Value

// The ValueMap is used to connect type names with corresponding Value
// representations of those types -- this can be used when it is not possible
// to use the Valuer interface (e.g., interface methods can only be
// defined within the package that defines the type -- so we need this for
// all types in gi which don't know about giv).
// You must use laser.LongTypeName (full package name + "." . type name) for
// the type name, as that is how it will be looked up.
var ValueMap map[string]ValueFunc

// ValueMapAdd adds a ValueFunc for the type of the given value.
func ValueMapAdd(val any, fun ValueFunc) {
	if ValueMap == nil {
		ValueMap = make(map[string]ValueFunc)
	}
	nm := gti.TypeNameObj(val)
	ValueMap[nm] = fun
}

// StructTagVal returns the value for given key in given struct tag string
// uses reflect.StructTag Lookup method -- just a wrapper for external
// use (e.g., in Python code)
func StructTagVal(key, tags string) string {
	stag := reflect.StructTag(tags)
	val, _ := stag.Lookup(key)
	return val
}

// ToValue returns the appropriate Value for given item, based only on
// its type -- attempts to get the Valuer interface and failing that,
// falls back on default Kind-based options.  tags are optional tags, e.g.,
// from the field in a struct, that control the view properties -- see the gi wiki
// for details on supported tags -- these are NOT set for the view element, only
// used for options that affect what kind of view to create.
// See FieldToValue for version that takes into account the properties of the owner.
//
//gopy:interface=handle
func ToValue(it any, tags string) Value {
	if it == nil {
		return &ValueBase{}
	}
	if vv, ok := it.(Valuer); ok {
		vvo := vv.Value()
		if vvo != nil {
			return vvo
		}
	}
	// try pointer version..
	if vv, ok := laser.PtrInterface(it).(Valuer); ok {
		vvo := vv.Value()
		if vvo != nil {
			return vvo
		}
	}

	if _, ok := it.(enums.BitFlag); ok {
		return &BitFlagValue{}
	}
	if _, ok := it.(enums.Enum); ok {
		return &EnumValue{}
	}

	typ := reflect.TypeOf(it)
	nptyp := laser.NonPtrType(typ)
	vk := typ.Kind()
	// fmt.Printf("vv val %v: typ: %v nptyp: %v kind: %v\n", it, typ.String(), nptyp.String(), vk)

	nptypnm := laser.LongTypeName(nptyp)
	if vvf, has := ValueMap[nptypnm]; has {
		vv := vvf()
		return vv
	}

	forceInline := false
	forceNoInline := false

	/*
		tprops := kit.Types.Properties(typ, false) // don't make
		if tprops != nil {
			if inprop, ok := kit.TypeProp(*tprops, "inline"); ok {
				forceInline, ok = kit.ToBool(inprop)
			}
			if inprop, ok := kit.TypeProp(*tprops, "no-inline"); ok {
				forceNoInline, ok = kit.ToBool(inprop)
			}
		}
	*/

	stag := reflect.StructTag(tags)
	vtag := stag.Get("view")

	switch vtag {
	case "inline":
		forceInline = true
	case "no-inline":
		forceNoInline = true
	}

	switch {
	case vk >= reflect.Int && vk <= reflect.Uint64:
		if vtag == "slider" {
			return &SliderValue{}
		}
		if _, ok := it.(fmt.Stringer); ok { // use stringer
			return &ValueBase{}
		}
		return &IntValue{}
	case vk == reflect.Bool:
		return &BoolValue{}
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		if vtag == "slider" {
			return &SliderValue{}
		}
		return &FloatValue{} // handles step, min / max etc
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		// todo: special edit with 2 fields..
		return &ValueBase{}
	case vk == reflect.Ptr:
		if ki.IsKi(nptyp) {
			return &KiPtrValue{}
		}
		if laser.AnyIsNil(it) {
			return &NilValue{}
		}
		v := reflect.ValueOf(it)
		if !v.IsZero() {
			// note: interfaces go here:
			// fmt.Printf("vv indirecting on pointer: %v type: %v\n", it, nptyp.String())
			return ToValue(v.Elem().Interface(), tags)
		}
	case vk == reflect.Array, vk == reflect.Slice:
		v := reflect.ValueOf(it)
		sz := v.Len()
		eltyp := laser.SliceElType(it)
		if _, ok := it.([]byte); ok {
			return &ByteSliceValue{}
		}
		if _, ok := it.([]rune); ok {
			return &RuneSliceValue{}
		}
		isstru := laser.NonPtrType(eltyp).Kind() == reflect.Struct
		if !forceNoInline && (forceInline || (!isstru && sz <= gi.SystemSettings.SliceInlineLength && !ki.IsKi(eltyp))) {
			return &SliceInlineValue{}
		} else {
			return &SliceValue{}
		}
	case vk == reflect.Map:
		sz := laser.MapStructElsN(it)
		if !forceNoInline && (forceInline || sz <= gi.SystemSettings.MapInlineLength) {
			return &MapInlineValue{}
		} else {
			return &MapValue{}
		}
	case vk == reflect.Struct:
		nfld := laser.AllFieldsN(nptyp)
		if nfld > 0 && !forceNoInline && (forceInline || nfld <= gi.SystemSettings.StructInlineLength) {
			return &StructInlineValue{}
		} else {
			return &StructValue{}
		}
	case vk == reflect.Func:
		if laser.AnyIsNil(it) {
			return &NilValue{}
		}
		return &FuncValue{}
	case vk == reflect.Interface:
		// note: we never get here -- all interfaces are captured by pointer kind above
		// apparently (because the non-ptr vk indirection does that I guess?)
		fmt.Printf("interface kind: %v %v %v\n", nptyp, nptyp.Name(), nptyp.String())
	case vk == reflect.String:
		v := reflect.ValueOf(it)
		str := v.String()
		_ = str
		switch vtag {
		case "text-field", "password":
			return &ValueBase{}
		// TODO(kai): figure out how to return text editor values here
		// case "text-editor":
		// 	return &TextEditorValue{}
		case "filename":
			return &FileValue{}
		default:
			// if strings.Contains(str, "\n") {
			// 	return &TextEditorValue{}
			// }
			return &ValueBase{}
		}
	}
	// fallback.
	return &ValueBase{}
}

// FieldToValue returns the appropriate Value for given field on a
// struct -- attempts to get the FieldValuer interface, and falls back on
// ToValue otherwise, using field value (fval)
// gopy:interface=handle
func FieldToValue(it any, field string, fval any) Value {
	if it == nil || field == "" {
		return ToValue(fval, "")
	}
	if vv, ok := it.(FieldValuer); ok {
		vvo := vv.FieldValue(field, fval)
		if vvo != nil {
			return vvo
		}
	}
	// try pointer version..
	if vv, ok := laser.PtrInterface(it).(FieldValuer); ok {
		vvo := vv.FieldValue(field, fval)
		if vvo != nil {
			return vvo
		}
	}

	typ := reflect.TypeOf(it)
	nptyp := laser.NonPtrType(typ)

	/*
		if pv, has := kit.Types.Prop(nptyp, "EnumType:"+field); has {
			et := pv.(reflect.Type)
			if kit.Enums.IsBitFlag(et) {
				vv := &BitFlagView{}
				vv.AltType = et
				ki.InitNode(vv)
				return vv
			} else {
				vv := &EnumValue{}
				vv.AltType = et
				ki.InitNode(vv)
				return vv
			}
		}
	*/

	ftyp, ok := nptyp.FieldByName(field)
	if ok {
		return ToValue(fval, string(ftyp.Tag))
	}
	return ToValue(fval, "")
}

//////////////////////////////////////////////////////////////////////////////
//  StructValue

// StructValue presents a button to edit the struct
type StructValue struct {
	ValueBase
}

func (vv *StructValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *StructValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	bt := vv.Widget.(*gi.Button)
	npv := laser.NonPtrValue(vv.Value)
	if vv.Value.IsZero() || npv.IsZero() {
		bt.SetTextUpdate("None")
	} else {
		opv := laser.OnePtrUnderlyingValue(vv.Value)
		if lbler, ok := opv.Interface().(gi.Labeler); ok {
			bt.SetTextUpdate(lbler.Label())
		} else {
			bt.SetTextUpdate(laser.FriendlyTypeName(npv.Type()))
		}
	}
}

func (vv *StructValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Icon = icons.Edit
	ConfigDialogWidget(vv, bt, true)
	vv.UpdateWidget()
}

func (vv *StructValue) HasDialog() bool                      { return true }
func (vv *StructValue) OpenDialog(ctx gi.Widget, fun func()) { OpenValueDialog(vv, ctx, fun) }

func (vv *StructValue) ConfigDialog(d *gi.Body) (bool, func()) {
	if vv.Value.IsZero() || laser.NonPtrValue(vv.Value).IsZero() {
		return false, nil
	}
	opv := laser.OnePtrUnderlyingValue(vv.Value)
	stru := opv.Interface()
	vpath := vv.ViewPath + "/" + laser.NonPtrType(opv.Type()).String()
	NewStructView(d).SetStruct(stru).SetViewPath(vpath).SetTmpSave(vv.TmpSave).
		SetReadOnly(vv.IsReadOnly())
	if tb, ok := stru.(gi.Toolbarer); ok {
		d.AddAppBar(tb.ConfigToolbar)
	}
	return true, nil
}

//////////////////////////////////////////////////////////////////////////////
//  StructInlineValue

// StructInlineValue presents a StructViewInline for a struct
type StructInlineValue struct {
	ValueBase
}

func (vv *StructInlineValue) WidgetType() *gti.Type {
	vv.WidgetTyp = StructViewInlineType
	return vv.WidgetTyp
}

func (vv *StructInlineValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	vv.CreateTempIfNotPtr() // essential to always have this set
	sv := vv.Widget.(*StructViewInline)
	cst := vv.Value.Interface()
	sv.SetStruct(cst)
}

func (vv *StructInlineValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	sv := vv.Widget.(*StructViewInline)
	sv.Tooltip = vv.Doc()
	sv.StructValView = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	vv.CreateTempIfNotPtr() // we need our value to be a ptr to a struct -- if not make a tmp
	sv.SetStruct(vv.Value.Interface())
	sv.OnFinal(events.Change, func(e events.Event) {
		vv.SendChange()
	})
	vv.UpdateWidget()
}

//////////////////////////////////////////////////////////////////////////////
//  SliceValue

// SliceValue presents a button to edit slices
type SliceValue struct {
	ValueBase
	IsArray    bool         // is an array, not a slice
	ElType     reflect.Type // type of element in the slice -- has pointer if slice has pointers
	ElIsStruct bool         // whether non-pointer element type is a struct or not
}

func (vv *SliceValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *SliceValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	vv.GetTypeInfo()
	ac := vv.Widget.(*gi.Button)
	npv := laser.OnePtrUnderlyingValue(vv.Value).Elem()
	txt := ""
	if !npv.IsValid() {
		txt = "None"
	} else {
		if npv.Kind() == reflect.Array || !npv.IsNil() {
			bnm := laser.FriendlyTypeName(vv.ElType)
			if strings.HasSuffix(bnm, "s") {
				txt = strcase.ToSentence(fmt.Sprintf("%d lists of %s", npv.Len(), bnm))
			} else {
				txt = strcase.ToSentence(fmt.Sprintf("%d %ss", npv.Len(), bnm))
			}
		} else {
			txt = "None"
		}
	}
	ac.SetTextUpdate(txt)
}

func (vv *SliceValue) GetTypeInfo() {
	slci := vv.Value.Interface()
	vv.IsArray = laser.NonPtrType(reflect.TypeOf(slci)).Kind() == reflect.Array
	if slci != nil && !laser.AnyIsNil(slci) {
		vv.ElType = laser.SliceElType(slci)
		vv.ElIsStruct = laser.NonPtrType(vv.ElType).Kind() == reflect.Struct
	}
}

func (vv *SliceValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.GetTypeInfo()
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Icon = icons.Edit
	ConfigDialogWidget(vv, bt, true)
	vv.UpdateWidget()
}

func (vv *SliceValue) HasDialog() bool                      { return true }
func (vv *SliceValue) OpenDialog(ctx gi.Widget, fun func()) { OpenValueDialog(vv, ctx, fun) }

func (vv *SliceValue) ConfigDialog(d *gi.Body) (bool, func()) {
	if vv.Value.IsZero() || laser.NonPtrValue(vv.Value).IsZero() {
		return false, nil
	}
	vvp := laser.OnePtrValue(vv.Value)
	if vvp.Kind() != reflect.Ptr {
		slog.Error("giv.SliceValue: Cannot view unadressable (non-pointer) slices", "type", vv.Value.Type())
		return false, nil
	}
	slci := vvp.Interface()
	vpath := vv.ViewPath + "/" + laser.NonPtrType(vvp.Type()).String()
	if !vv.IsArray && vv.ElIsStruct {
		tv := NewTableView(d).SetSlice(slci).SetTmpSave(vv.TmpSave).SetViewPath(vpath)
		tv.SetReadOnly(vv.IsReadOnly())
		d.AddAppBar(tv.ConfigToolbar)
	} else {
		sv := NewSliceView(d).SetSlice(slci).SetTmpSave(vv.TmpSave).SetViewPath(vpath)
		sv.SetReadOnly(vv.IsReadOnly())
		d.AddAppBar(sv.ConfigToolbar)
	}
	return true, nil
}

//////////////////////////////////////////////////////////////////////////////
//  SliceInlineValue

// SliceInlineValue presents a SliceViewInline for a map
type SliceInlineValue struct {
	ValueBase
}

func (vv *SliceInlineValue) WidgetType() *gti.Type {
	vv.WidgetTyp = SliceViewInlineType
	return vv.WidgetTyp
}

func (vv *SliceInlineValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sv := vv.Widget.(*SliceViewInline)
	csl := vv.Value.Interface()
	newslc := false
	if reflect.TypeOf(vv.Value).Kind() != reflect.Pointer { // prevent crash on non-comparable
		newslc = true
	} else {
		newslc = sv.Slice != csl
	}
	if newslc {
		sv.SetSlice(csl)
	} else {
		sv.Update()
	}
}

func (vv *SliceInlineValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	sv := vv.Widget.(*SliceViewInline)
	sv.Tooltip = vv.Doc()
	sv.SliceValue = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	// npv := vv.Value.Elem()
	sv.SetSlice(vv.Value.Interface())
	sv.OnFinal(events.Change, func(e events.Event) {
		vv.SendChange()
	})
}

//////////////////////////////////////////////////////////////////////////////
//  MapValue

// MapValue presents a button to edit maps
type MapValue struct {
	ValueBase
}

func (vv *MapValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *MapValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	npv := laser.NonPtrValue(vv.Value)
	mpi := vv.Value.Interface()
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
	bt.SetTextUpdate(txt)
}

func (vv *MapValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Icon = icons.Edit
	ConfigDialogWidget(vv, bt, true)
	vv.UpdateWidget()
}

func (vv *MapValue) HasDialog() bool                      { return true }
func (vv *MapValue) OpenDialog(ctx gi.Widget, fun func()) { OpenValueDialog(vv, ctx, fun) }

func (vv *MapValue) ConfigDialog(d *gi.Body) (bool, func()) {
	if vv.Value.IsZero() || laser.NonPtrValue(vv.Value).IsZero() {
		return false, nil
	}
	mpi := vv.Value.Interface()
	vpath := vv.ViewPath + "/" + laser.NonPtrType(vv.Value.Type()).String()
	mv := NewMapView(d).SetMap(mpi)
	mv.SetViewPath(vpath).SetTmpSave(vv.TmpSave).SetReadOnly(vv.IsReadOnly())
	d.AddAppBar(mv.ConfigToolbar)
	return true, nil
}

//////////////////////////////////////////////////////////////////////////////
//  MapInlineValue

// MapInlineValue presents a MapViewInline for a map
type MapInlineValue struct {
	ValueBase
}

func (vv *MapInlineValue) WidgetType() *gti.Type {
	vv.WidgetTyp = MapViewInlineType
	return vv.WidgetTyp
}

func (vv *MapInlineValue) UpdateWidget() {
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

func (vv *MapInlineValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	sv := vv.Widget.(*MapViewInline)
	sv.Tooltip = vv.Doc()
	sv.MapValView = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	// npv := vv.Value.Elem()
	sv.SetMap(vv.Value.Interface())
	sv.OnFinal(events.Change, func(e events.Event) {
		vv.SendChange()
	})
}

//////////////////////////////////////////////////////////////////////////////
//  KiPtrValue

// KiPtrValue provides a chooser for pointers to Ki objects
type KiPtrValue struct {
	ValueBase
}

func (vv *KiPtrValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

// get the Ki struct itself (or nil)
func (vv *KiPtrValue) KiStruct() ki.Ki {
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

func (vv *KiPtrValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	path := "None"
	k := vv.KiStruct()
	if k != nil && k.This() != nil {
		path = k.AsKi().String()
	}
	bt.SetTextUpdate(path)
}

func (vv *KiPtrValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	ConfigDialogWidget(vv, bt, true)
	vv.UpdateWidget()
}

func (vv *KiPtrValue) HasDialog() bool                      { return true }
func (vv *KiPtrValue) OpenDialog(ctx gi.Widget, fun func()) { OpenValueDialog(vv, ctx, fun) }

func (vv *KiPtrValue) ConfigDialog(d *gi.Body) (bool, func()) {
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

func (vv *BoolValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	cb := vv.Widget.(*gi.Switch)
	cb.Tooltip = vv.Doc()
	cb.Config()
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

func (vv *IntValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
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
	sb.Config()
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

func (vv *FloatValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
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
	sb.Config()
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

func (vv *SliderValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
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
	sl.Config()
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

func (vv *EnumValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	ch := vv.Widget.(*gi.Chooser)
	ch.Tooltip = vv.Doc()

	ev := vv.EnumValue()
	ch.SetEnum(ev)
	ch.Config()
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

func (vv *BitFlagValue) ConfigWidget(w gi.Widget) {
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
	sw.Config()
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

func (vv *TypeValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
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
	cb.Config()
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
		tf.SetTextUpdate(string(bv))
	}
}

func (vv *ByteSliceValue) ConfigWidget(w gi.Widget) {
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
	vv.StdConfigWidget(w)
	tf.Config()

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
		tf.SetTextUpdate(string(rv))
	}
}

func (vv *RuneSliceValue) ConfigWidget(w gi.Widget) {
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
	vv.StdConfigWidget(w)
	tf.Config()

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
	sb := vv.Widget.(*gi.Label)
	sb.SetTextUpdate("None")
}

func (vv *NilValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	sb := vv.Widget.(*gi.Label)
	sb.Tooltip = vv.Doc()
	sb.Config()
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
		bt.SetIconUpdate(icons.Blank)
	} else {
		bt.SetIconUpdate(icons.Icon(txt))
	}
	if sntag, ok := vv.Tag("view"); ok {
		if strings.Contains(sntag, "show-name") {
			if txt == "" {
				txt = "None"
			}
			bt.SetTextUpdate(strcase.ToSentence(txt))
		}
	}
}

func (vv *IconValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
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
	bt.SetProp("font-family", txt)
	bt.SetTextUpdate(txt)
}

func (vv *FontValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
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
	FontChooserSize.ToDots(&wb.Styles.UnContext)
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
	bt.SetTextUpdate(txt)
	if txt != prev {
		bt.SetNeedsLayout(true)
	}
}

func (vv *FileValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
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

func (vv *FuncValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)

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

func (vv *OptionValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)

	fr := vv.Widget.(*gi.Frame)

	gi.NewButton(fr, "unset").SetText("Unset")
	val := vv.Value.FieldByName("Value").Interface()
	NewValue(fr, val, "value")

	vv.UpdateWidget()
}
