// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"reflect"

	"github.com/rcoreilly/goki/ki"
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
	typ := kit.NonPtrType(reflect.TypeOf(it))
	vk := typ.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Uint64:
		// todo: spinbox -- could set some properties here based on kind..
		vv := ValueViewBase{}
		vv.Init(&vv)
		return &vv
	case vk == reflect.Bool:
		// todo: togglebutton
		vv := ValueViewBase{}
		vv.Init(&vv)
		return &vv
	// case vk >= reflect.Float32 && vk <= reflect.Float64: // just default
	// 	vv := ValueViewBase{}
	// 	vv.Init(&vv)
	// 	return &vv
	case vk >= reflect.Complex64 && vk <= reflect.Complex128:
		// todo: special edit with 2 fields..
		vv := ValueViewBase{}
		vv.Init(&vv)
		return &vv
	case vk == reflect.Ptr: // nothing possible for plain pointers -- ki.Ptr can do it though..
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
		vv := MapValueView{}
		vv.Init(&vv)
		return &vv
	case vk == reflect.Struct:
		// todo: check inline, use that if possible
		vv := StructValueView{}
		vv.Init(&vv)
		return &vv
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

// ValueView is an interface for representing values (e.g., fields) in Views
type ValueView interface {
	ki.Ki
	// AsValueViewBase gives access to the basic data fields so that the interface doesn't need to provide accessors for them
	AsValueViewBase() *ValueViewBase
	// SetStructValue sets the value, owner and field information for a struct field
	SetStructValue(val reflect.Value, owner interface{}, field *reflect.StructField)
	// SetMapValue sets the value, owner and map key information for a map element
	SetMapValue(val reflect.Value, owner interface{}, key interface{})
	// SetSliceValue sets the value, owner and index information for a slice element
	SetSliceValue(val reflect.Value, owner interface{}, idx int)
	// WidgetType returns an appropriate type of widget to represent the current value
	WidgetType() reflect.Type
	// UpdateWidget updates the widget representation to reflect the current value
	UpdateWidget()
	// ConfigWidget configures a widget of WidgetType for representing the value, including setting up the signal connections to set the value when the user edits it (values are always set immediately when the widget is updated)
	ConfigWidget(widg Node2D)
}

// ValueViewBase provides the basis for implementations of the ValueView interface, representing values in the interface -- it implements a generic TextField representation of the string value, and provides the generic fallback for everything that doesn't provide a specific ValueViewer type
type ValueViewBase struct {
	ki.Node
	Value     reflect.Value        `desc:"the reflect.Value representation of the value"`
	Owner     interface{}          `desc:"the object that owns this value, either a struct, slice, or map, if non-nil -- if a Ki Node, then SetField is used to set value, to provide proper updating"`
	OwnerType reflect.Type         `desc:"non-pointer type of the Owner, for convenience"`
	Field     *reflect.StructField `desc:"if Owner is a struct, this is the reflect.StructField associated with the value"`
	Key       interface{}          `desc:"if Owner is a map, this is the key for this value in the map"`
	Idx       int                  `desc:"if Owner is a slice, this is the index for the value in the slice"`
	WidgetTyp reflect.Type         `desc:"type of widget to create -- cached during WidgetType method -- chosen based on the ValueView type and reflect.Value type -- see ValueViewer interface"`
	Widget    Node2D               `desc:"the widget used to display and edit the value in the interface -- this is created for us externally and we cache it during ConfigWidget"`
	Label     string               `desc:"label for displaying this item -- based on Field.Name and optional label Tag value"`
}

var KiT_ValueViewBase = kit.Types.AddType(&ValueViewBase{}, nil)

func (vv *ValueViewBase) AsValueViewBase() *ValueViewBase {
	return vv
}

func (vv *ValueViewBase) SetStructValue(val reflect.Value, owner interface{}, field *reflect.StructField) {
	vv.Value = val
	vv.Owner = owner
	vv.Field = field
}

func (vv *ValueViewBase) SetMapValue(val reflect.Value, owner interface{}, key interface{}) {
	vv.Value = val
	vv.Owner = owner
	vv.Key = key
}

func (vv *ValueViewBase) SetSliceValue(val reflect.Value, owner interface{}, idx int) {
	vv.Value = val
	vv.Owner = owner
	vv.Idx = idx
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
	vv.UpdateWidget()
	tf.TextFieldSig.DisconnectAll() // these are re-used, so key to disconnect!
	tf.TextFieldSig.Connect(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_ValueViewBase).(*ValueViewBase)
		tf := send.(*TextField)
		if vvv.Owner != nil {
			if kiv, ok := vvv.Owner.(ki.Ki); ok {
				kiv.SetField(vvv.Field.Name, tf.Text) // does updates
				vvv.UpdateWidget()                    // always update after setting value..
				return
			}
		}
		kit.SetRobust(kit.PtrValue(vvv.Value).Interface(), tf.Text)
		vvv.UpdateWidget() // always update after setting value..
	})
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
	txt := fmt.Sprintf("%v", vv.Value.Type().Elem().Name())
	mb.SetText(txt)
}

func (vv *StructValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	mb := vv.Widget.(*MenuButton)
	mb.ResetMenu()
	mb.AddMenuText("Edit Struct", vv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_StructValueView).(*StructValueView)
		mb := vvv.Widget.(*MenuButton)
		PromptDialog(mb.Viewport, "Struct Value View", "Sorry, slice editor not implemented yet -- would show up here", true, false, nil, nil)
	})
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
	npv := vv.Value.Elem()
	sz := npv.Len()
	txt := fmt.Sprintf("[%v] %v", sz, npv.Type().Elem().Name())
	mb.SetText(txt)
}

func (vv *SliceValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	mb := vv.Widget.(*MenuButton)
	mb.ResetMenu()
	mb.AddMenuText("Edit Slice", vv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_SliceValueView).(*SliceValueView)
		mb := vvv.Widget.(*MenuButton)
		PromptDialog(mb.Viewport, "Slice Value View", "Sorry, slice editor not implemented yet -- would show up here", true, false, nil, nil)
	})
}

////////////////////////////////////////////////////////////////////////////////////////
//  MapValueView

// MapValueView presents a button to edit slices
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
	npv := vv.Value.Elem()
	sz := npv.Len()
	txt := fmt.Sprintf("[%v] %v", sz, npv.Type().Elem().Name())
	mb.SetText(txt)
}

func (vv *MapValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg
	vv.UpdateWidget()
	mb := vv.Widget.(*MenuButton)
	mb.ResetMenu()
	mb.AddMenuText("Edit Map", vv.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_MapValueView).(*MapValueView)
		mb := vvv.Widget.(*MenuButton)
		PromptDialog(mb.Viewport, "Map Value View", "Sorry, map editor not implemented yet -- would show up here", true, false, nil, nil)
	})
}
