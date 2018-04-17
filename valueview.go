// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  ValueView -- an interface for representing values (e.g., fields) in Views

// ValueViewer interface supplies the appropriate type of ValueView -- called on a given receiver item if defined for that receiver type (tries both pointer and non-pointer receivers) -- can use this for custom types to provide alternative custom interfaces
type ValueViewer interface {
	ValueView() ValueView
}

// example implementation of ValueViewer interface -- can't implment on non-local types, so all
// the basic types are handled separately
// func (s string) ValueView() ValueView {
// 	vv := ValueViewBase{}
// 	vv.Init(&vv)
// 	return &vv
// }

// FieldValueViewer interface supplies the appropriate type of ValueView for a given field name and current field value on the receiver parent struct -- called on a given receiver struct if defined for that receiver type (tries both pointer and non-pointer receivers) -- if a struct implements this interface, then it is used first for structs -- return nil to fall back on the default ToValueView result
type FieldValueViewer interface {
	FieldValueView(field string, fval interface{}) ValueView
}

// ToValueView returns the appropriate ValueView for given item, based only on its type -- attempts to get the ValueViewer interface and failing that, falls back on default Kind-based options --  see FieldToValueView, MapToValueView, SliceToValue view for versions that take into account the properties of the owner (used in those appropriate contexts)
func ToValueView(it interface{}) ValueView {
	if it == nil {
		vv := ValueViewBase{}
		vv.Init(&vv)
		return &vv
	}
	if vv, ok := it.(ValueViewer); ok {
		vvo := vv.ValueView()
		if vvo != nil {
			return vvo
		}
	}
	// try pointer version..
	if vv, ok := kit.PtrInterface(it).(ValueViewer); ok {
		vvo := vv.ValueView()
		if vvo != nil {
			return vvo
		}
	}
	typ := reflect.TypeOf(it)
	nptyp := kit.NonPtrType(typ)
	typrops := kit.Types.Properties(typ.Name(), false) // don't make
	vk := typ.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Uint64:
		if kit.Enums.TypeRegistered(nptyp) { // todo: bitfield
			vv := EnumValueView{}
			vv.Init(&vv)
			return &vv
		} else {
			vv := IntValueView{}
			vv.Init(&vv)
			return &vv
		}
	case vk == reflect.Bool:
		vv := BoolValueView{}
		vv.Init(&vv)
		return &vv
	case vk >= reflect.Float32 && vk <= reflect.Float64:
		vv := FloatValueView{} // handles step, min / max etc
		vv.Init(&vv)
		return &vv
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		// todo: special edit with 2 fields..
		vv := ValueViewBase{}
		vv.Init(&vv)
		return &vv
	case vk == reflect.Ptr:
		if ki.IsKi(nptyp) {
			vv := KiPtrValueView{}
			vv.Init(&vv)
			return &vv
		}
		return nil
	case vk == reflect.Slice:
		vv := SliceValueView{}
		vv.Init(&vv)
		return &vv
	case vk == reflect.Array:
		vv := SliceValueView{} // probably works?
		vv.Init(&vv)
		return &vv
	case vk == reflect.Map:
		v := reflect.ValueOf(it)
		sz := v.Len()
		if sz > 0 && sz <= 4 { // todo: param somewhere
			vv := MapInlineValueView{}
			vv.Init(&vv)
			return &vv
		} else {
			vv := MapValueView{}
			vv.Init(&vv)
			return &vv
		}
	case vk == reflect.Struct:
		inline := false
		if typrops != nil {
			inprop, ok := typrops["inline"]
			if ok {
				inline, ok = kit.ToBool(inprop)
			}
		}
		if inline || typ.NumField() <= 5 {
			vv := StructInlineValueView{}
			vv.Init(&vv)
			return &vv
		} else {
			vv := StructValueView{}
			vv.Init(&vv)
			return &vv
		}
	}
	// fallback.
	vv := ValueViewBase{}
	vv.Init(&vv)
	return &vv
}

// FieldToValueView returns the appropriate ValueView for given field on a struct -- attempts to get the FieldValueViewer interface, and falls back on ToValueView otherwise, using field value (fval)
func FieldToValueView(it interface{}, field string, fval interface{}) ValueView {
	if it == nil || field == "" {
		return ToValueView(fval)
	}
	if vv, ok := it.(FieldValueViewer); ok {
		vvo := vv.FieldValueView(field, fval)
		if vvo != nil {
			return vvo
		}
	}
	// try pointer version..
	if vv, ok := kit.PtrInterface(it).(FieldValueViewer); ok {
		vvo := vv.FieldValueView(field, fval)
		if vvo != nil {
			return vvo
		}
	}
	// fallback
	return ToValueView(fval)
}

// ValueView is an interface for representing values (e.g., fields, map values, slice values) in Views (StructView, MapView, etc) -- the different types of ValueView are for different Kinds of values (bool, float, etc) -- which can have different Kinds of owners -- the ValueVuewBase class supports all the basic fields for managing the owner kinds
type ValueView interface {
	ki.Ki
	// AsValueViewBase gives access to the basic data fields so that the interface doesn't need to provide accessors for them
	AsValueViewBase() *ValueViewBase
	// SetStructValue sets the value, owner and field information for a struct field
	SetStructValue(val reflect.Value, owner interface{}, field *reflect.StructField)
	// SetMapKey sets the key value and owner for a map key
	SetMapKey(val reflect.Value, owner interface{})
	// SetMapValue sets the value, owner and map key information for a map element -- needs pointer to ValueView representation of key to track current key value
	SetMapValue(val reflect.Value, owner interface{}, key interface{}, keyView ValueView)
	// SetSliceValue sets the value, owner and index information for a slice element
	SetSliceValue(val reflect.Value, owner interface{}, idx int)
	// OwnerKind returns the reflect.Kind of the owner: Struct, Map, or Slice
	OwnerKind() reflect.Kind
	// IsReadOnly returns whether the value is read-only -- e.g., Map owners have ReadOnly values, and some fields can be marked as ReadOnly using a struct tag
	IsReadOnly() bool
	// WidgetType returns an appropriate type of widget to represent the current value
	WidgetType() reflect.Type
	// UpdateWidget updates the widget representation to reflect the current value
	UpdateWidget()
	// ConfigWidget configures a widget of WidgetType for representing the value, including setting up the signal connections to set the value when the user edits it (values are always set immediately when the widget is updated)
	ConfigWidget(widg Node2D)
	// Val returns the reflect.Value representation for this item
	Val() reflect.Value
	// SetValue sets the value (if not ReadOnly), using Ki.SetField for Ki types and kit.SetRobust otherwise
	SetValue(val interface{}) bool
	// FieldTag returns tag associated with this field, if this is a field in a struct ("" otherwise or if tag not set)
	FieldTag(tagName string) string
}

// ValueViewBase provides the basis for implementations of the ValueView interface, representing values in the interface -- it implements a generic TextField representation of the string value, and provides the generic fallback for everything that doesn't provide a specific ValueViewer type
type ValueViewBase struct {
	ki.Node
	Value     reflect.Value        `desc:"the reflect.Value representation of the value"`
	OwnKind   reflect.Kind         `desc:"kind of owner that we have -- reflect.Struct, .Map, .Slice are supported"`
	IsMapKey  bool                 `desc:"for OwnKind = Map, this value represents the Key -- otherwise the Value"`
	Owner     interface{}          `desc:"the object that owns this value, either a struct, slice, or map, if non-nil -- if a Ki Node, then SetField is used to set value, to provide proper updating"`
	OwnerType reflect.Type         `desc:"non-pointer type of the Owner, for convenience"`
	Field     *reflect.StructField `desc:"if Owner is a struct, this is the reflect.StructField associated with the value"`
	Key       interface{}          `desc:"if Owner is a map, and this is a value, this is the key for this value in the map"`
	KeyView   ValueView            `desc:"if Owner is a map, and this is a value, this is the value view representing the key -- its value has the *current* value of the key, which can be edited"`
	Idx       int                  `desc:"if Owner is a slice, this is the index for the value in the slice"`
	WidgetTyp reflect.Type         `desc:"type of widget to create -- cached during WidgetType method -- chosen based on the ValueView type and reflect.Value type -- see ValueViewer interface"`
	Widget    Node2D               `desc:"the widget used to display and edit the value in the interface -- this is created for us externally and we cache it during ConfigWidget"`
	Label     string               `desc:"label for displaying this item -- based on Field.Name and optional label Tag value"`
}

var KiT_ValueViewBase = kit.Types.AddType(&ValueViewBase{}, ValueViewBaseProps)

var ValueViewBaseProps = map[string]interface{}{
	"base-type": true,
}

func (vv *ValueViewBase) AsValueViewBase() *ValueViewBase {
	return vv
}

func (vv *ValueViewBase) SetStructValue(val reflect.Value, owner interface{}, field *reflect.StructField) {
	vv.OwnKind = reflect.Struct
	vv.Value = val
	vv.Owner = owner
	vv.Field = field
}

func (vv *ValueViewBase) SetMapKey(key reflect.Value, owner interface{}) {
	vv.OwnKind = reflect.Map
	vv.IsMapKey = true
	vv.Value = key
	vv.Owner = owner
}

func (vv *ValueViewBase) SetMapValue(val reflect.Value, owner interface{}, key interface{}, keyView ValueView) {
	vv.OwnKind = reflect.Map
	vv.Value = val
	vv.Owner = owner
	vv.Key = key
	vv.KeyView = keyView
}

func (vv *ValueViewBase) SetSliceValue(val reflect.Value, owner interface{}, idx int) {
	vv.OwnKind = reflect.Slice
	vv.Value = val
	vv.Owner = owner
	vv.Idx = idx
}

// we have this one accessor b/c it is more useful for outside consumers vs. internal usage
func (vv *ValueViewBase) OwnerKind() reflect.Kind {
	return vv.OwnKind
}

func (vv *ValueViewBase) IsReadOnly() bool {
	if vv.OwnKind == reflect.Struct {
		rotag := vv.FieldTag("read-only")
		if rotag != "" {
			return true
		}
	}
	return false
}

func (vv *ValueViewBase) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_TextField
	return vv.WidgetTyp
}

func (vv *ValueViewBase) UpdateWidget() {
	tf := vv.Widget.(*TextField)
	txt := kit.ToString(vv.Value.Interface())
	tf.SetText(txt)
}

func (vv *ValueViewBase) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	tf := vv.Widget.(*TextField)
	tf.SetProp("max-width", -1) // todo..
	bitflag.SetState(tf.Flags(), vv.IsReadOnly(), int(ReadOnly))
	vv.UpdateWidget()
	tf.TextFieldSig.DisconnectAll() // these are re-used, so key to disconnect!
	tf.TextFieldSig.Connect(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_ValueViewBase).(*ValueViewBase)
		tf := send.(*TextField)
		if vvv.SetValue(tf.Text) {
			vvv.UpdateWidget() // always update after setting value..
		}
	})
}

func (vv *ValueViewBase) Val() reflect.Value {
	return vv.Value
}

func (vv *ValueViewBase) SetValue(val interface{}) bool {
	if vv.IsReadOnly() {
		return false
	}
	if vv.Owner != nil {
		switch vv.OwnKind {
		case reflect.Struct:
			if kiv, ok := vv.Owner.(ki.Ki); ok {
				return kiv.SetField(vv.Field.Name, val)
			} else {
				return kit.SetRobust(kit.PtrValue(vv.Value).Interface(), val)
			}
		case reflect.Map:
			ov := kit.NonPtrValue(reflect.ValueOf(vv.Owner))
			if vv.IsMapKey {
				nv := reflect.ValueOf(val)                // new key value
				cv := ov.MapIndex(vv.Value)               // get current value
				ov.SetMapIndex(vv.Value, reflect.Value{}) // delete old key
				ov.SetMapIndex(nv, cv)                    // set new key to current value
				vv.Value = nv                             // update value to new key
			} else {
				if vv.KeyView != nil {
					ck := vv.KeyView.AsValueViewBase().Value // current key value
					ov.SetMapIndex(ck, reflect.ValueOf(val))
				} else { // static, key not editable?
					ov.SetMapIndex(reflect.ValueOf(vv.Key), reflect.ValueOf(val))
				}
			}
		case reflect.Slice:
			// should be addressable?
			return kit.SetRobust(kit.PtrValue(vv.Value).Interface(), val)
		}
	} else {
		return kit.SetRobust(kit.PtrValue(vv.Value).Interface(), val)
	}
	return false
}

func (vv *ValueViewBase) FieldTag(tagName string) string {
	if !(vv.Owner != nil && vv.OwnKind == reflect.Struct) {
		return ""
	}
	return vv.Field.Tag.Get(tagName)
}

// check for interface implementation
var _ ValueView = &ValueViewBase{}

////////////////////////////////////////////////////////////////////////////////////////
//  StructValueView

// StructValueView presents a button to edit slices
type StructValueView struct {
	ValueViewBase
}

var KiT_StructValueView = kit.Types.AddType(&StructValueView{}, nil)

func (vv *StructValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_MenuButton
	return vv.WidgetTyp
}

func (vv *StructValueView) UpdateWidget() {
	mb := vv.Widget.(*MenuButton)
	npv := kit.NonPtrValue(vv.Value)
	txt := fmt.Sprintf("%v", npv.Type().Name())
	mb.SetText(txt)
}

func (vv *StructValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	mb := vv.Widget.(*MenuButton)
	mb.SetProp("padding", units.NewValue(2, units.Px))
	mb.SetProp("margin", units.NewValue(2, units.Px))
	mb.ResetMenu()
	mb.AddMenuText("Edit Struct", vv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_StructValueView).(*StructValueView)
		mb := vvv.Widget.(*MenuButton)
		StructViewDialog(mb.Viewport, vv.Value.Interface(), "Struct Value View", "", nil, nil)
	})
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
	// sv := vv.Widget.(*StructViewInline)
	// npv := vv.Value.Elem()
	// sv.SetChecked(npv.Bool())
}

func (vv *StructInlineValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	sv := vv.Widget.(*StructViewInline)
	// npv := vv.Value.Elem()
	sv.SetStruct(vv.Value.Interface())
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
	// sv := vv.Widget.(*MapViewInline)
	// npv := kit.NonPtrValue(vv.Value)
	// sv.SetChecked(npv.Bool())
}

func (vv *MapInlineValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	sv := vv.Widget.(*MapViewInline)
	// npv := vv.Value.Elem()
	sv.SetMap(vv.Value.Interface())
}

////////////////////////////////////////////////////////////////////////////////////////
//  SliceValueView

// SliceValueView presents a button to edit slices
type SliceValueView struct {
	ValueViewBase
}

var KiT_SliceValueView = kit.Types.AddType(&SliceValueView{}, nil)

func (vv *SliceValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_MenuButton
	return vv.WidgetTyp
}

func (vv *SliceValueView) UpdateWidget() {
	mb := vv.Widget.(*MenuButton)
	npv := kit.NonPtrValue(vv.Value)
	sz := npv.Len()
	txt := fmt.Sprintf("[%v] %v", sz, npv.Type().Elem().Name())
	mb.SetText(txt)
}

func (vv *SliceValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	mb := vv.Widget.(*MenuButton)
	mb.SetProp("padding", units.NewValue(2, units.Px))
	mb.SetProp("margin", units.NewValue(2, units.Px))
	mb.ResetMenu()
	mb.AddMenuText("Edit Slice", vv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_SliceValueView).(*SliceValueView)
		mb := vvv.Widget.(*MenuButton)
		SliceViewDialog(mb.Viewport, vv.Value.Interface(), "Slice Value View", "", nil, nil)
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
	vv.WidgetTyp = KiT_MenuButton
	return vv.WidgetTyp
}

func (vv *MapValueView) UpdateWidget() {
	mb := vv.Widget.(*MenuButton)
	npv := kit.NonPtrValue(vv.Value)
	sz := npv.Len()
	txt := fmt.Sprintf("[%v] %v", sz, npv.Type().Elem().Name())
	mb.SetText(txt)
}

func (vv *MapValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	mb := vv.Widget.(*MenuButton)
	mb.SetProp("padding", units.NewValue(2, units.Px))
	mb.SetProp("margin", units.NewValue(2, units.Px))
	mb.ResetMenu()
	mb.AddMenuText("Edit Map", vv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_MapValueView).(*MapValueView)
		mb := vvv.Widget.(*MenuButton)
		MapViewDialog(mb.Viewport, vv.Value.Interface(), "Map Value View", "", nil, nil)
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
	vv.WidgetTyp = KiT_MenuButton
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
	mb := vv.Widget.(*MenuButton)
	path := "nil"
	k := vv.KiStruct()
	if k != nil {
		path = k.Path()
	}
	mb.SetText(path)
}

func (vv *KiPtrValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	mb := vv.Widget.(*MenuButton)
	mb.SetProp("padding", units.NewValue(2, units.Px))
	mb.SetProp("margin", units.NewValue(2, units.Px))
	mb.ResetMenu()
	mb.AddMenuText("Edit", vv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_KiPtrValueView).(*KiPtrValueView)
		k := vvv.KiStruct()
		if k != nil {
			mb := vvv.Widget.(*MenuButton)
			StructViewDialog(mb.Viewport, k, "Struct Value View", "", nil, nil)
		}
	})
	mb.AddMenuText("Select", vv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_KiPtrValueView).(*KiPtrValueView)
		mb := vvv.Widget.(*MenuButton)
		PromptDialog(mb.Viewport, "KiPtr Value View", "Sorry, Ki object chooser  not implemented yet -- would show up here", true, false, nil, nil)
	})
	mb.AddMenuText("GoGiEditor", vv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_KiPtrValueView).(*KiPtrValueView)
		k := vvv.KiStruct()
		if k != nil {
			GoGiEditorOf(k)
		}
	})
}

////////////////////////////////////////////////////////////////////////////////////////
//  BoolValueView

// BoolValueView presents a checkbox for a boolean
type BoolValueView struct {
	ValueViewBase
}

var KiT_BoolValueView = kit.Types.AddType(&BoolValueView{}, nil)

func (vv *BoolValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_CheckBox
	return vv.WidgetTyp
}

func (vv *BoolValueView) UpdateWidget() {
	cb := vv.Widget.(*CheckBox)
	npv := kit.NonPtrValue(vv.Value)
	cb.SetChecked(npv.Bool())
}

func (vv *BoolValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	cb := vv.Widget.(*CheckBox)
	cb.SetReadOnlyState(vv.IsReadOnly())
	cb.ButtonSig.DisconnectAll()
	cb.ButtonSig.Connect(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_BoolValueView).(*BoolValueView)
		cbb := vvv.Widget.(*CheckBox)
		if vvv.SetValue(cbb.IsChecked()) {
			vvv.UpdateWidget() // always update after setting value..
		}
	})
}

////////////////////////////////////////////////////////////////////////////////////////
//  IntValueView

// IntValueView presents a spinbox
type IntValueView struct {
	ValueViewBase
}

var KiT_IntValueView = kit.Types.AddType(&IntValueView{}, nil)

func (vv *IntValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_SpinBox
	return vv.WidgetTyp
}

func (vv *IntValueView) UpdateWidget() {
	sb := vv.Widget.(*SpinBox)
	npv := kit.NonPtrValue(vv.Value)
	fv, ok := kit.ToFloat(npv.Interface())
	if ok {
		sb.SetValue(fv)
	}
}

func (vv *IntValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	sb := vv.Widget.(*SpinBox)
	sb.SetReadOnlyState(vv.IsReadOnly())
	sb.Defaults()
	sb.Step = 1.0
	sb.PageStep = 10.0
	sb.SetProp("#textfield", map[string]interface{}{
		"width": units.NewValue(5, units.Ex),
	})
	vk := vv.Value.Kind()
	if vk >= reflect.Uint && vk <= reflect.Uint64 {
		sb.SetMin(0)
	}
	// todo: make a utility for this kind of thing..
	mintag := vv.FieldTag("min")
	if mintag != "" {
		min, err := strconv.ParseFloat(mintag, 64)
		if err == nil {
			sb.SetMin(min)
		}
	}
	maxtag := vv.FieldTag("max")
	if maxtag != "" {
		max, err := strconv.ParseFloat(maxtag, 64)
		if err == nil {
			sb.SetMax(max)
		}
	}
	steptag := vv.FieldTag("step")
	if steptag != "" {
		step, err := strconv.ParseFloat(steptag, 64)
		if err == nil {
			sb.Step = step
		}
	}
	sb.SpinBoxSig.DisconnectAll()
	sb.SpinBoxSig.Connect(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_IntValueView).(*IntValueView)
		sbb := vvv.Widget.(*SpinBox)
		if vvv.SetValue(sbb.Value) {
			vvv.UpdateWidget()
		}
	})
}

////////////////////////////////////////////////////////////////////////////////////////
//  FloatValueView

// FloatValueView presents a spinbox
type FloatValueView struct {
	ValueViewBase
}

var KiT_FloatValueView = kit.Types.AddType(&FloatValueView{}, nil)

func (vv *FloatValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_SpinBox
	return vv.WidgetTyp
}

func (vv *FloatValueView) UpdateWidget() {
	sb := vv.Widget.(*SpinBox)
	npv := kit.NonPtrValue(vv.Value)
	fv, ok := kit.ToFloat(npv.Interface())
	if ok {
		sb.SetValue(fv)
	}
}

func (vv *FloatValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	sb := vv.Widget.(*SpinBox)
	sb.SetReadOnlyState(vv.IsReadOnly())
	sb.Defaults()
	sb.Step = 1.0
	sb.PageStep = 10.0
	// todo: make a utility for this kind of thing..
	mintag := vv.FieldTag("min")
	if mintag != "" {
		min, err := strconv.ParseFloat(mintag, 64)
		if err == nil {
			sb.HasMin = true
			sb.Min = min
		}
	}
	maxtag := vv.FieldTag("max")
	if maxtag != "" {
		max, err := strconv.ParseFloat(maxtag, 64)
		if err == nil {
			sb.HasMax = true
			sb.Max = max
		}
	}
	steptag := vv.FieldTag("step")
	if steptag != "" {
		step, err := strconv.ParseFloat(steptag, 64)
		if err == nil {
			sb.Step = step
		}
	}

	sb.SpinBoxSig.DisconnectAll()
	sb.SpinBoxSig.Connect(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_FloatValueView).(*FloatValueView)
		sbb := vvv.Widget.(*SpinBox)
		if vvv.SetValue(sbb.Value) {
			vvv.UpdateWidget()
		}
	})
}

////////////////////////////////////////////////////////////////////////////////////////
//  EnumValueView

// EnumValueView presents a combobox for choosing enums
type EnumValueView struct {
	ValueViewBase
}

var KiT_EnumValueView = kit.Types.AddType(&EnumValueView{}, nil)

func (vv *EnumValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_ComboBox
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
	sb := vv.Widget.(*ComboBox)
	npv := kit.NonPtrValue(vv.Value)
	iv, ok := kit.ToInt(npv.Interface())
	if ok {
		sb.SetCurIndex(int(iv)) // todo: currently only working for 0-based values
	}
}

func (vv *EnumValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	cb := vv.Widget.(*ComboBox)
	cb.SetReadOnlyState(vv.IsReadOnly())

	typ := vv.EnumType()
	cb.ItemsFromEnum(typ, false, 50)

	vv.UpdateWidget()

	cb.ComboSig.DisconnectAll()
	cb.ComboSig.Connect(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_EnumValueView).(*EnumValueView)
		cbb := vvv.Widget.(*ComboBox)
		eval := cbb.CurVal.(kit.EnumValue)
		if vvv.SetEnumValueFromInt(eval.Value) { // todo: using index
			vvv.UpdateWidget()
		}
	})
}
