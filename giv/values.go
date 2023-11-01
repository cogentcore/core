// Copyright (c) 2018, The GoKi Authors. All rights reserved.
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

	"goki.dev/enums"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/paint"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/laser"
	"goki.dev/pi/v2/filecat"
)

// values contains all the Values for basic builtin types

func init() {
	gi.TheViewIFace = &ViewIFace{}
	ValueMapAdd(laser.LongTypeName(reflect.TypeOf(icons.Icon(""))), func() Value {
		return &IconValue{}
	})
	ValueMapAdd(laser.LongTypeName(reflect.TypeOf(gi.FontName(""))), func() Value {
		return &FontValue{}
	})
	ValueMapAdd(laser.LongTypeName(reflect.TypeOf(gi.FileName(""))), func() Value {
		return &FileValue{}
	})
	ValueMapAdd(laser.LongTypeName(reflect.TypeOf(keyfun.MapName(""))), func() Value {
		return &KeyMapValue{}
	})
	ValueMapAdd(laser.LongTypeName(reflect.TypeOf(gi.ColorName(""))), func() Value {
		return &ColorNameValue{}
	})
	ValueMapAdd(laser.LongTypeName(reflect.TypeOf(key.Chord(""))), func() Value {
		return &KeyChordValue{}
	})
	ValueMapAdd(laser.LongTypeName(reflect.TypeOf(gi.HiStyleName(""))), func() Value {
		return &HiStyleValue{}
	})
	ValueMapAdd(laser.LongTypeName(reflect.TypeOf(time.Time{})), func() Value {
		return &TimeValue{}
	})
	ValueMapAdd(laser.LongTypeName(reflect.TypeOf(filecat.FileTime{})), func() Value {
		return &TimeValue{}
	})
}

var (
	// MapInlineLen is the number of map elements at or below which an inline
	// representation of the map will be presented -- more convenient for small
	// numbers of properties
	MapInlineLen = 2

	// StructInlineLen is the number of elemental struct fields at or below which an inline
	// representation of the struct will be presented -- more convenient for small structs
	StructInlineLen = 4

	// SliceInlineLen is the number of slice elements at or below which inline will be used
	SliceInlineLen = 4
)

////////////////////////////////////////////////////////////////////////////////////////
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

////////////////////////////////////////////////////////////////////////////////////////
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

// ValueMapAdd adds a ValueFunc for a given type name.
// You must use laser.LongTypeName (full package name + "." . type name) for
// the type name, as that is how it will be looked up.
func ValueMapAdd(typeNm string, fun ValueFunc) {
	if ValueMap == nil {
		ValueMap = make(map[string]ValueFunc)
	}
	ValueMap[typeNm] = fun
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
// gopy:interface=handle
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

	if tags != "" {
		stag := reflect.StructTag(tags)
		if vwtag, ok := stag.Lookup("view"); ok {
			switch vwtag {
			case "inline":
				forceInline = true
			case "no-inline":
				forceNoInline = true
			}
		}
	}

	switch {
	case vk >= reflect.Int && vk <= reflect.Uint64:
		if _, ok := it.(time.Duration); ok {
			return &DurationValue{}
		}
		if _, ok := it.(fmt.Stringer); ok { // use stringer
			return &ValueBase{}
		}
		return &IntValue{}
	case vk == reflect.Bool:
		return &BoolValue{}
	case vk >= reflect.Float32 && vk <= reflect.Float64:
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
		if !laser.ValueIsZero(v) {
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
		isstru := (laser.NonPtrType(eltyp).Kind() == reflect.Struct)
		if !forceNoInline && (forceInline || (!isstru && sz <= SliceInlineLen && !ki.IsKi(eltyp))) {
			return &SliceInlineValue{}
		} else {
			return &SliceValue{}
		}
	case vk == reflect.Map:
		sz := laser.MapStructElsN(it)
		if !forceNoInline && (forceInline || sz <= MapInlineLen) {
			return &MapInlineValue{}
		} else {
			return &MapValue{}
		}
	case vk == reflect.Struct:
		// note: we need to handle these here b/c cannot define new methods for gi types
		switch nptyp {
		case laser.TypeFor[color.RGBA]():
			return &ColorValue{}
		case laser.TypeFor[gti.Type]():
			return &TypeValue{}
		}
		nfld := laser.AllFieldsN(nptyp)
		if nfld > 0 && !forceNoInline && (forceInline || nfld <= StructInlineLen) {
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
		if strings.Contains(str, "\n") {
			return &TextEditorValue{}
		}
		return &ValueBase{}
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

////////////////////////////////////////////////////////////////////////////////////////
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

func (vv *StructValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
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
	bt.Tooltip = vv.Doc()
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *StructValue) HasButton() bool {
	return true
}

func (vv *StructValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	title, newPath, isZero := vv.GetTitle()
	if isZero {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	opv := laser.OnePtrUnderlyingValue(vv.Value)
	readOnly := vv.IsReadOnly()
	stru := opv.Interface()
	if gi.RecycleDialog(stru) {
		return
	}
	d := gi.NewDialog(vv.Widget).Title(title).Prompt(vv.Doc()).FullWindow(true)
	NewStructView(d).SetViewPath(vpath).SetTmpSave(vv.TmpSave).SetStruct(stru).SetState(readOnly, states.ReadOnly)
	d.OnAccept(func(e events.Event) {
		vv.UpdateWidget()
		vv.SendChange()
		if fun != nil {
			fun(d)
		}
	}).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
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
	sv := vv.Widget.(*StructViewInline)
	cst := vv.Value.Interface()
	if sv.Struct != cst {
		sv.SetStruct(cst)
	} else {
		sv.UpdateFields()
	}
}

func (vv *StructInlineValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	sv := vv.Widget.(*StructViewInline)
	sv.Sc = sc
	sv.Tooltip = vv.Doc()
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
	npv := laser.NonPtrValue(vv.Value)
	txt := ""
	if !npv.IsValid() {
		txt = "nil"
	} else if npv.Kind() == reflect.Interface {
		txt = fmt.Sprintf("Slice: %T", npv.Interface())
	} else {
		if npv.Kind() == reflect.Array {
			txt = fmt.Sprintf("Array [%v]%v", npv.Len(), vv.ElType.String())
		} else if npv.IsNil() {
			txt = "nil"
		} else {
			txt = fmt.Sprintf("Slice [%v]%v", npv.Len(), vv.ElType.String())
		}
	}
	ac.SetText(txt)
}

func (vv *SliceValue) GetTypeInfo() {
	slci := vv.Value.Interface()
	vv.IsArray = laser.NonPtrType(reflect.TypeOf(slci)).Kind() == reflect.Array
	if slci != nil && !laser.AnyIsNil(slci) {
		vv.ElType = laser.SliceElType(slci)
		vv.ElIsStruct = (laser.NonPtrType(vv.ElType).Kind() == reflect.Struct)
	}
}

func (vv *SliceValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
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
	bt.Tooltip = vv.Doc()
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *SliceValue) HasButton() bool {
	return true
}

func (vv *SliceValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	title, newPath, isZero := vv.GetTitle()
	if isZero {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	vvp := laser.OnePtrValue(vv.Value)
	if vvp.Kind() != reflect.Ptr {
		slog.Error("giv.SliceValue: Cannot view unadressable (non-pointer) slices", "type", vv.Value.Type())
		return
	}
	readOnly := vv.IsReadOnly()
	slci := vvp.Interface()
	if !vv.IsArray && vv.ElIsStruct {
		d := gi.NewDialog(vv.Widget).Title(title).Prompt(vv.Doc()).FullWindow(true)
		NewTableView(d).SetSlice(slci).SetTmpSave(vv.TmpSave).SetViewPath(vpath).SetState(readOnly, states.ReadOnly)
		d.OnAccept(func(e events.Event) {
			vv.UpdateWidget()
			vv.SendChange()
			if fun != nil {
				fun(d)
			}
		}).Run()
	} else {
		d := gi.NewDialog(vv.Widget).Title(title).Prompt(vv.Doc()).FullWindow(true)
		NewSliceView(d).SetSlice(slci).SetTmpSave(vv.TmpSave).SetViewPath(vpath).SetState(readOnly, states.ReadOnly)
		d.OnAccept(func(e events.Event) {
			vv.UpdateWidget()
			vv.SendChange()
			if fun != nil {
				fun(d)
			}
		}).Run()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
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
		newslc = (sv.Slice != csl)
	}
	if newslc {
		sv.SetSlice(csl)
	} else {
		sv.UpdateValues()
	}
}

func (vv *SliceInlineValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	sv := vv.Widget.(*SliceViewInline)
	sv.Sc = sc
	sv.Tooltip = vv.Doc()
	sv.SliceValView = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	// npv := vv.Value.Elem()
	sv.SetSlice(vv.Value.Interface())
	sv.OnChange(func(e events.Event) {
		vv.SendChange()
	})
}

////////////////////////////////////////////////////////////////////////////////////////
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
		txt = "nil"
	} else if npv.Kind() == reflect.Interface {
		txt = fmt.Sprintf("Map: %T", npv.Interface())
	} else {
		txt = fmt.Sprintf("Map: [%v %v]%v", npv.Len(), laser.MapKeyType(mpi).String(), laser.MapValueType(mpi).String())
	}
	bt.SetText(txt)
}

func (vv *MapValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Icon = icons.Edit
	bt.Tooltip = vv.Doc()
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *MapValue) HasButton() bool {
	return true
}

func (vv *MapValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	title, newPath, isZero := vv.GetTitle()
	if isZero {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	mpi := vv.Value.Interface()
	readOnly := vv.IsReadOnly()
	d := gi.NewDialog(vv.Widget).Title(title).Prompt(vv.Doc()).FullWindow(true)
	NewMapView(d).SetMap(mpi).SetTmpSave(vv.TmpSave).SetViewPath(vpath).SetState(readOnly, states.ReadOnly)
	d.OnAccept(func(e events.Event) {
		vv.UpdateWidget()
		vv.SendChange()
		if fun != nil {
			fun(d)
		}
	}).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
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

func (vv *MapInlineValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	sv := vv.Widget.(*MapViewInline)
	sv.Sc = sc
	sv.Tooltip = vv.Doc()
	sv.MapValView = vv
	sv.ViewPath = vv.ViewPath
	sv.TmpSave = vv.TmpSave
	// npv := vv.Value.Elem()
	sv.SetMap(vv.Value.Interface())
	sv.OnChange(func(e events.Event) {
		vv.SendChange()
	})
}

////////////////////////////////////////////////////////////////////////////////////////
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
	path := "nil"
	k := vv.KiStruct()
	if k != nil {
		path = k.Path()
	}
	bt.SetText(path)
}

func (vv *KiPtrValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Indicator = icons.KeyboardArrowDown
	bt.Tooltip = vv.Doc()
	bt.Menu = func(m *gi.Scene) {
		gi.NewButton(m, "edit").SetText("Edit").OnClick(func(e events.Event) {
			k := vv.KiStruct()
			if k != nil {
				bt := vv.Widget.(*gi.Button)
				vv.OpenDialog(bt, nil)
			}
		})
		gi.NewButton(m, "gogi-editor").SetText("GoGi editor").OnClick(func(e events.Event) {
			k := vv.KiStruct()
			if k != nil && !vv.IsReadOnly() {
				GoGiEditorDialog(k)
			}
		})
	}
	bt.Config(sc)
	vv.UpdateWidget()
}

func (vv *KiPtrValue) HasButton() bool {
	return true
}

func (vv *KiPtrValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	title, newPath, isZero := vv.GetTitle()
	if isZero {
		return
	}
	k := vv.KiStruct()
	if k == nil {
		return
	}
	vpath := vv.ViewPath + "/" + newPath
	readOnly := vv.IsReadOnly()
	d := gi.NewDialog(ctx).Title(title).Prompt(vv.Doc()).FullWindow(true)
	NewStructView(d).SetStruct(k).SetTmpSave(vv.TmpSave).SetViewPath(vpath).SetState(readOnly, states.ReadOnly)
	d.OnAccept(func(e events.Event) {
		vv.UpdateWidget()
		vv.SendChange()
		if fun != nil {
			fun(d)
		}
	}).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
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

func (vv *BoolValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	cb := vv.Widget.(*gi.Switch)
	cb.Tooltip = vv.Doc()
	cb.Config(sc)
	cb.OnChange(func(e events.Event) {
		vv.SetValue(cb.StateIs(states.Checked))
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
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

func (vv *IntValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
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
	// 	s.Width.SetCh(5)
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
	sb.Config(sc)
	sb.OnChange(func(e events.Event) {
		vv.SetValue(sb.Value)
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
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

func (vv *FloatValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
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
	if mintag, ok := vv.Tag("min"); ok {
		minv, err := laser.ToFloat32(mintag)
		if err == nil {
			sb.HasMin = true
			sb.Min = minv
		} else {
			slog.Error("Float Min Value:", "error:", err)
		}
	}
	if maxtag, ok := vv.Tag("max"); ok {
		maxv, err := laser.ToFloat32(maxtag)
		if err == nil {
			sb.HasMax = true
			sb.Max = maxv
		} else {
			slog.Error("Float Max Value:", "error:", err)
		}
	}
	sb.Step = .1 // smaller default
	if steptag, ok := vv.Tag("step"); ok {
		step, err := laser.ToFloat32(steptag)
		if err == nil {
			sb.Step = step
		} else {
			slog.Error("Float Step Value:", "error:", err)
		}
	}
	if fmttag, ok := vv.Tag("format"); ok {
		sb.Format = fmttag
	}
	sb.Config(sc)
	sb.OnChange(func(e events.Event) {
		vv.SetValue(sb.Value)
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
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
	ev, ok := vv.Value.Interface().(enums.Enum)
	if ok {
		return ev
	}
	slog.Error("giv.EnumValue: type must be enums.Enum")
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
	ch.SetCurVal(npv.Interface())
	// iv, err := laser.ToInt(npv.Interface())
	// if err == nil {
	// 	ch.SetCurIndex(int(iv)) // todo: currently only working for 0-based values
	// } else {
	// 	slog.Error("Enum Value:", err)
	// }
}

func (vv *EnumValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	ch := vv.Widget.(*gi.Chooser)
	ch.Tooltip = vv.Doc()

	ev := vv.EnumValue()
	ch.ItemsFromEnum(ev, false, 50)
	ch.Config(sc)
	ch.OnChange(func(e events.Event) {
		vv.SetValue(ch.CurVal)
		// cval := ch.CurVal.(enums.Enum)
		// vv.SetEnumValueFromInt(cval.Int64()) // todo: using index
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  BitFlagView

// BitFlagValue presents chip [gi.Switches] for editing bitflags
type BitFlagValue struct {
	ValueBase
}

func (vv *BitFlagValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.SwitchesType
	return vv.WidgetTyp
}

func (vv *BitFlagValue) EnumValue() enums.BitFlag {
	ev, ok := vv.Value.Interface().(enums.BitFlag)
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
	// todo: needs to set flags?
	// typ := vv.EnumType()
	// eval := laser.EnumIfaceFromInt64(ival, typ)
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

func (vv *BitFlagValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	sw := vv.Widget.(*gi.Switches)
	sw.SetType(gi.SwitchChip)
	// vv.StdConfigWidget(cb.Parts)
	// cb.Parts.Lay = gi.LayoutHoriz
	sw.Tooltip = vv.Doc()

	ev := vv.EnumValue()
	sw.ItemsFromEnum(ev)
	sw.Config(sc)
	// cb.ButtonSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	vvv, _ := recv.Embed(TypeBitFlagView).(*BitFlagView)
	// 	cbb := vvv.Widget.(*gi.Switches)
	// 	etyp := vvv.EnumType()
	// 	val := cbb.BitFlagsValue(etyp)
	// 	vvv.SetEnumValueFromInt(val)
	// 	// vvv.UpdateWidget()
	// })
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
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
	fmt.Println(npv.Type().String())
	typ, ok := npv.Interface().(*gti.Type)
	if ok {
		sb.SetCurVal(typ)
	}
}

func (vv *TypeValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
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
	cb.Config(sc)
	cb.OnChange(func(e events.Event) {
		tval := cb.CurVal.(*gti.Type)
		vv.SetValue(tval)
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
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
		tf.Update()
	}
}

func (vv *ByteSliceValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	tf := vv.Widget.(*gi.TextField)
	tf.Tooltip = vv.Doc()
	// STYTODO: figure out how how to handle these kinds of styles
	tf.Style(func(s *styles.Style) {
		s.MinWidth.Ch(16)
		s.MaxWidth.Dp(-1)
	})
	tf.Config(sc)

	tf.OnChange(func(e events.Event) {
		vv.SetValue(tf.Text())
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
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
		tf.Update()
	}
}

func (vv *RuneSliceValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	tf := vv.Widget.(*gi.TextField)
	tf.Tooltip = vv.Doc()
	tf.Style(func(s *styles.Style) {
		s.MinWidth.Ch(16)
		s.MaxWidth.Dp(-1)
	})
	tf.Config(sc)

	tf.OnChange(func(e events.Event) {
		vv.SetValue(tf.Text())
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
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
	npv := laser.NonPtrValue(vv.Value)
	tstr := ""
	if !laser.ValueIsZero(npv) {
		tstr = npv.String() // npv.Type().String()
	} else if !laser.ValueIsZero(vv.Value) {
		tstr = vv.Value.String() // vv.Value.Type().String()
	}
	sb.SetText("nil " + tstr)
}

func (vv *NilValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	sb := vv.Widget.(*gi.Label)
	sb.Tooltip = vv.Doc()
	sb.Config(sc)
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
//  TimeValue

var DefaultTimeFormat = "2006-01-02 15:04:05 MST"

// TimeValue presents a text field for a time
type TimeValue struct {
	ValueBase
}

func (vv *TimeValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.TextFieldType
	return vv.WidgetTyp
}

// TimeVal decodes Value into a *time.Time value -- also handles FileTime case
func (vv *TimeValue) TimeVal() *time.Time {
	tmi := laser.PtrValue(vv.Value).Interface()
	switch v := tmi.(type) {
	case *time.Time:
		return v
	case *filecat.FileTime:
		return (*time.Time)(v)
	}
	return nil
}

func (vv *TimeValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	tf := vv.Widget.(*gi.TextField)
	tm := vv.TimeVal()
	tf.SetText(tm.Format(DefaultTimeFormat))
	tf.Update()
}

func (vv *TimeValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	tf := vv.Widget.(*gi.TextField)
	tf.SetStretchMaxWidth()
	tf.Tooltip = vv.Doc()
	tf.Style(func(s *styles.Style) {
		tf.Styles.MinWidth.Ch(float32(len(DefaultTimeFormat) + 2))
	})
	tf.Config(sc)
	tf.OnChange(func(e events.Event) {
		nt, err := time.Parse(DefaultTimeFormat, tf.Text())
		if err != nil {
			slog.Error(err.Error())
		} else {
			tm := vv.TimeVal()
			*tm = nt
			// vv.SendChange()
			vv.UpdateWidget()
		}
	})
	vv.UpdateWidget()
}

////////////////////////////////////////////////////////////////////////////////////////
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
				txt = "none"
			}
			bt.SetText(txt)
		}
	}
	bt.Update() // icon always requires redraw in case changed
}

func (vv *IconValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *IconValue) HasDialog() bool {
	return true
}

func (vv *IconValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	ics := icons.All()
	cur := icons.Icon(laser.ToString(vv.Value.Interface()))
	d := gi.NewDialog(ctx).Title("Select an icon").Prompt(vv.Doc()).FullWindow(true)
	NewSliceView(d).SetStyleFunc(func(w gi.Widget, s *styles.Style, row int) {
		w.(*gi.Button).SetText(string(ics[row]))
		s.SetStretchMaxWidth()
	}).
		SetSlice(&ics).SetSelVal(cur).BindSelectDialog(d, &si)
	d.OnAccept(func(e events.Event) {
		if si >= 0 {
			ic := icons.AllIcons[si]
			vv.SetValue(ic)
			vv.UpdateWidget()
		}
		if fun != nil {
			fun(d)
		}
	}).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
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
	bt.SetText(txt)
	bt.Update()
}

func (vv *FontValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(vv.Widget, nil)
	})
	vv.UpdateWidget()
}

func (vv *FontValue) HasDialog() bool {
	return true
}

// show fonts in a bigger size so you can actually see the differences
var FontChooserSize = units.Pt(18)

func (vv *FontValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	wb := ctx.AsWidget()
	FontChooserSize.ToDots(&wb.Styles.UnContext)
	paint.FontLibrary.OpenAllFonts(int(FontChooserSize.Dots))
	fi := paint.FontLibrary.FontInfo
	cur := gi.FontName(laser.ToString(vv.Value.Interface()))
	d := gi.NewDialog(ctx).Title("Select a Font").Prompt(vv.Doc()).FullWindow(true)
	NewTableView(d).SetStyleFunc(func(w gi.Widget, s *styles.Style, row, col int) {
		if col != 4 {
			return
		}
		s.Font.Family = fi[row].Name
		s.Font.Stretch = fi[row].Stretch
		s.Font.Weight = fi[row].Weight
		s.Font.Style = fi[row].Style
		s.Font.Size = FontChooserSize
	}).SetSlice(&fi).SetSelVal(cur).BindSelectDialog(d, &si)
	d.OnAccept(func(e events.Event) {
		if si >= 0 {
			fi := paint.FontLibrary.FontInfo[si]
			vv.SetValue(fi.Name)
			vv.UpdateWidget()
		}
		if fun != nil {
			fun(d)
		}
	}).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  FileValue

// FileValue presents an action for displaying a FileName and selecting
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
	bt.SetText(txt)
	bt.Update()
}

func (vv *FileValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		bt := vv.Widget.(*gi.Button)
		// e.SetHandled()
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *FileValue) HasDialog() bool {
	return true
}

func (vv *FileValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	cur := laser.ToString(vv.Value.Interface())
	ext, _ := vv.Tag("ext")
	d := gi.NewDialog(ctx).Title(vv.Name()).Prompt(vv.Doc()).FullWindow(true)
	fv := NewFileView(d).SetFilename(cur, ext)
	fv.OnSelect(func(e events.Event) {
		cur = fv.SelectedFile()
	}).OnDoubleClick(func(e events.Event) {
		cur = fv.SelectedFile()
		d.AcceptDialog()
	})
	d.Cancel().Ok().OnAccept(func(e events.Event) {
		vv.SetValue(cur)
		vv.UpdateWidget()
		if fun != nil {
			fun(d)
		}
	}).Run()
}

////////////////////////////////////////////////////////////////////////////////////////
//  VersCtrlValue

// VersCtrlSystems is a list of supported Version Control Systems.
// These must match the VCS Types from goki/pi/vci which in turn
// is based on masterminds/vcs
var VersCtrlSystems = []string{"git", "svn", "bzr", "hg"}

// IsVersCtrlSystem returns true if the given string matches one of the
// standard VersCtrlSystems -- uses lowercase version of str.
func IsVersCtrlSystem(str string) bool {
	stl := strings.ToLower(str)
	for _, vcn := range VersCtrlSystems {
		if stl == vcn {
			return true
		}
	}
	return false
}

// VersCtrlName is the name of a version control system
type VersCtrlName string

func VersCtrlNameProper(vc string) VersCtrlName {
	vcl := strings.ToLower(vc)
	for _, vcnp := range VersCtrlSystems {
		vcnpl := strings.ToLower(vcnp)
		if strings.Compare(vcl, vcnpl) == 0 {
			return VersCtrlName(vcnp)
		}
	}
	return ""
}

// Value registers VersCtrlValue as the viewer of VersCtrlName
func (kn VersCtrlName) Value() Value {
	return &VersCtrlValue{}
}

// VersCtrlValue presents an action for displaying an VersCtrlName and selecting
// from StringPopup
type VersCtrlValue struct {
	ValueBase
}

func (vv *VersCtrlValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *VersCtrlValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	if txt == "" {
		txt = "(none)"
	}
	bt.SetText(txt)
	bt.Update()
}

func (vv *VersCtrlValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(vv.Widget, nil)
	})
	vv.UpdateWidget()
}

func (vv *VersCtrlValue) HasDialog() bool {
	return true
}

func (vv *VersCtrlValue) OpenDialog(ctx gi.Widget, fun func(dlg *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	// TODO(kai/menu): add back StringsChooserPopup here
	// cur := laser.ToString(vv.Value.Interface())
	// gi.StringsChooserPopup(VersCtrlSystems, cur, ctx, func(ac *gi.Button) {
	// 	vv.SetValue(ac.Text)
	// 	vv.UpdateWidget()
	// })
}

// TextEditorValue presents a [texteditor.Editor] for editing longer text
type TextEditorValue struct {
	ValueBase
}

func (vv *TextEditorValue) WidgetType() *gti.Type {
	vv.WidgetTyp = texteditor.EditorType
	return vv.WidgetTyp
}

func (vv *TextEditorValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	sb := vv.Widget.(*texteditor.Editor)
	npv := laser.NonPtrValue(vv.Value)
	sb.Buf.SetText([]byte(npv.String()))
}

func (vv *TextEditorValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)

	tb := texteditor.NewBuf()
	tb.Stat()

	tv := w.(*texteditor.Editor)
	tv.SetBuf(tb)

	vv.UpdateWidget()
}

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
	fbt.SetFunc(fun)
}

func (vv *FuncValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
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
