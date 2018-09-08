// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

func init() {
	gi.TheViewIFace = &ViewIFace{}
}

// MapInlineLen is the number of map elements at or below which an inline
// representation of the map will be presented -- more convenient for small
// #'s of props
var MapInlineLen = 6

// StructInlineLen is the number of elemental struct fields at or below which an inline
// representation of the struct will be presented -- more convenient for small structs
var StructInlineLen = 6

////////////////////////////////////////////////////////////////////////////////////////
//  ValueView -- an interface for representing values (e.g., fields) in Views

// ValueViewer interface supplies the appropriate type of ValueView -- called
// on a given receiver item if defined for that receiver type (tries both
// pointer and non-pointer receivers) -- can use this for custom types to
// provide alternative custom interfaces -- must call Init on ValueView before
// returning it
type ValueViewer interface {
	ValueView() ValueView
}

// example implementation of ValueViewer interface -- can't implment on
// non-local types, so all the basic types are handled separately:
//
// func (s string) ValueView() ValueView {
// 	vv := ValueViewBase{}
// 	vv.Init(&vv)
// 	return &vv
// }

// FieldValueViewer interface supplies the appropriate type of ValueView for a
// given field name and current field value on the receiver parent struct --
// called on a given receiver struct if defined for that receiver type (tries
// both pointer and non-pointer receivers) -- if a struct implements this
// interface, then it is used first for structs -- return nil to fall back on
// the default ToValueView result
type FieldValueViewer interface {
	FieldValueView(field string, fval interface{}) ValueView
}

// ToValueView returns the appropriate ValueView for given item, based only on
// its type -- attempts to get the ValueViewer interface and failing that,
// falls back on default Kind-based options -- see FieldToValueView,
// MapToValueView, SliceToValue view for versions that take into account the
// properties of the owner (used in those appropriate contexts)
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
	typrops := kit.Types.Properties(typ, false) // don't make
	vk := typ.Kind()
	// fmt.Printf("vv val %v: typ: %v nptyp: %v kind: %v\n", it, typ.String(), nptyp.String(), vk)

	if nptyp == reflect.TypeOf(gi.IconName("")) {
		vv := IconValueView{}
		vv.Init(&vv)
		return &vv
	}
	if nptyp == reflect.TypeOf(gi.FontName("")) {
		vv := FontValueView{}
		vv.Init(&vv)
		return &vv
	}
	if nptyp == reflect.TypeOf(gi.FileName("")) {
		vv := FileValueView{}
		vv.Init(&vv)
		return &vv
	}
	if nptyp == reflect.TypeOf(gi.KeyMapName("")) {
		vv := KeyMapValueView{}
		vv.Init(&vv)
		return &vv
	}
	if nptyp == reflect.TypeOf(gi.KeyChord("")) {
		vv := KeyChordValueView{}
		vv.Init(&vv)
		return &vv
	}

	switch {
	case vk >= reflect.Int && vk <= reflect.Uint64:
		if kit.Enums.TypeRegistered(nptyp) { // todo: bitfield
			vv := EnumValueView{}
			vv.Init(&vv)
			return &vv
		} else if _, ok := it.(fmt.Stringer); ok { // use stringer
			vv := ValueViewBase{}
			vv.Init(&vv)
			return &vv
		} else {
			vv := IntValueView{}
			vv.Init(&vv)
			return &vv
		}
	case nptyp == reflect.TypeOf(time.Time{}): // todo: could do better..
		vv := ValueViewBase{}
		vv.Init(&vv)
		return &vv
	case nptyp == reflect.TypeOf(FileTime{}): // todo: could do better..
		vv := ValueViewBase{}
		vv.Init(&vv)
		return &vv
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
		if kit.IfaceIsNil(it) {
			return nil
		}
		v := reflect.ValueOf(it)
		if !kit.ValueIsZero(v) {
			return ToValueView(v.Elem().Interface())
		}
	case nptyp == ki.KiT_Signal:
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
		sz = kit.MapStructElsN(it)
		if sz > 0 && sz <= MapInlineLen {
			vv := MapInlineValueView{}
			vv.Init(&vv)
			return &vv
		} else {
			vv := MapValueView{}
			vv.Init(&vv)
			return &vv
		}
	case vk == reflect.Struct:
		// note: we need to handle these here b/c cannot define new methods for gi types
		if nptyp == gi.KiT_Color {
			vv := ColorValueView{}
			vv.Init(&vv)
			return &vv
		}
		inline := false
		if typrops != nil {
			inprop, ok := typrops["inline"]
			if ok {
				inline, ok = kit.ToBool(inprop)
			}
		}
		nfld := kit.AllFieldsN(typ)
		if inline || nfld <= StructInlineLen {
			vv := StructInlineValueView{}
			vv.Init(&vv)
			return &vv
		} else {
			vv := StructValueView{}
			vv.Init(&vv)
			return &vv
		}
	case vk == reflect.Interface:
		fmt.Printf("interface kind: %v %v %v\n", nptyp, nptyp.Name(), nptyp.String())
		switch {
		case nptyp == reflect.TypeOf((*reflect.Type)(nil)).Elem():
			vv := TypeValueView{}
			vv.Init(&vv)
			return &vv
		}
	}
	// fallback.
	vv := ValueViewBase{}
	vv.Init(&vv)
	return &vv
}

// FieldToValueView returns the appropriate ValueView for given field on a
// struct -- attempts to get the FieldValueViewer interface, and falls back on
// ToValueView otherwise, using field value (fval)
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

	if kig, ok := it.(ki.Ki); ok {
		typ := reflect.TypeOf(fval)
		if typ != nil {
			nptyp := kit.NonPtrType(typ)
			vk := nptyp.Kind()

			ft := kig.FieldTag(field, "view")
			switch ft {
			case "no-inline":
				if vk == reflect.Map {
					vv := MapValueView{}
					vv.Init(&vv)
					return &vv
				}
			}
		}
	}

	// fallback
	return ToValueView(fval)
}

// ValueView is an interface for managing the GUI representation of values
// (e.g., fields, map values, slice values) in Views (StructView, MapView,
// etc).  The different types of ValueView are for different Kinds of values
// (bool, float, etc) -- which can have different Kinds of owners.  The
// ValueVuewBase class supports all the basic fields for managing the owner
// kinds.
type ValueView interface {
	ki.Ki

	// AsValueViewBase gives access to the basic data fields so that the
	// interface doesn't need to provide accessors for them.
	AsValueViewBase() *ValueViewBase

	// SetStructValue sets the value, owner and field information for a struct field.
	SetStructValue(val reflect.Value, owner interface{}, field *reflect.StructField, tmpSave ValueView)

	// SetMapKey sets the key value and owner for a map key.
	SetMapKey(val reflect.Value, owner interface{}, tmpSave ValueView)

	// SetMapValue sets the value, owner and map key information for a map
	// element -- needs pointer to ValueView representation of key to track
	// current key value.
	SetMapValue(val reflect.Value, owner interface{}, key interface{}, keyView ValueView, tmpSave ValueView)

	// SetSliceValue sets the value, owner and index information for a slice element.
	SetSliceValue(val reflect.Value, owner interface{}, idx int, tmpSave ValueView)

	// SetStandaloneValue sets the value for a singleton standalone value
	// (e.g., for arg values).
	SetStandaloneValue(val reflect.Value)

	// OwnerKind returns the reflect.Kind of the owner: Struct, Map, or Slice
	// (or Invalid for standalone values such as args).
	OwnerKind() reflect.Kind

	// IsInactive returns whether the value is inactive -- e.g., Map owners
	// have Inactive values, and some fields can be marked as Inactive using a
	// struct tag.
	IsInactive() bool

	// WidgetType returns an appropriate type of widget to represent the
	// current value.
	WidgetType() reflect.Type

	// UpdateWidget updates the widget representation to reflect the current
	// value.  Must first check for a nil widget -- can be called in a
	// no-widget context (e.g., for single-argument values with actions).
	UpdateWidget()

	// ConfigWidget configures a widget of WidgetType for representing the
	// value, including setting up the signal connections to set the value
	// when the user edits it (values are always set immediately when the
	// widget is updated).
	ConfigWidget(widg gi.Node2D)

	// HasAction returns true if this value has an associated action, such as
	// pulling up a dialog or chooser for this value.  Activate method will
	// trigger this action.
	HasAction() bool

	// Activate triggers any action associated with this value, such as
	// pulling up a dialog or chooser for this value.  This is called by
	// default for single-argument methods that have value representations
	// with actions.  The viewport provides a context for opening other
	// windows, and the receiver and dlgFunc should receive the DialogSig for
	// the relevant dialog, or a pass-on call thereof, including the
	// DialogAccepted or Canceled signal, so that the caller can execute its
	// own actions based on the user hitting Ok or Cancel.
	Activate(vp *gi.Viewport2D, recv ki.Ki, dlgFunc ki.RecvFunc)

	// Val returns the reflect.Value representation for this item.
	Val() reflect.Value

	// SetValue assigns given value to this item (if not Inactive), using
	// Ki.SetField for Ki types and kit.SetRobust otherwise -- emits a ViewSig
	// signal when set.
	SetValue(val interface{}) bool

	// SetTags sets tags for this valueview, for non-struct values, to
	// influence interface for this value -- see
	// https://github.com/goki/gi/wiki/Tags for valid options.  Adds to
	// existing tags if some are already set.
	SetTags(tags map[string]string)

	// SetTag sets given tag to given value for this valueview, for non-struct
	// values, to influence interface for this value -- see
	// https://github.com/goki/gi/wiki/Tags for valid options.
	SetTag(tag, value string)

	// Tag returns value for given tag -- looks first at tags set by
	// SetTag(s) methods, and then at field tags if this is a field in a
	// struct -- returns false if tag was not set.
	Tag(tag string) (string, bool)

	// SaveTmp saves a temporary copy of a struct to a map -- map values must
	// be explicitly re-saved and cannot be directly written to by the value
	// elements -- each ValueView has a pointer to any parent ValueView that
	// might need to be saved after SetValue -- SaveTmp called automatically
	// in SetValue but other cases that use something different need to call
	// it explicitly.
	SaveTmp()
}

// note: could have a more efficient way to represent the different owner type
// data (Key vs. Field vs. Idx), instead of just having everything for
// everything.  However, ValueView itself gets customized for different target
// value types, and those are orthogonal to the owner type, so need a separate
// ValueViewOwner class that encodes these options more efficiently -- but
// that introduces another struct alloc and pointer -- not clear if it is
// worth it..

// ValueViewBase provides the basis for implementations of the ValueView
// interface, representing values in the interface -- it implements a generic
// TextField representation of the string value, and provides the generic
// fallback for everything that doesn't provide a specific ValueViewer type.
type ValueViewBase struct {
	ki.Node
	ViewSig   ki.Signal            `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update -- data is the value that was set"`
	Value     reflect.Value        `desc:"the reflect.Value representation of the value"`
	OwnKind   reflect.Kind         `desc:"kind of owner that we have -- reflect.Struct, .Map, .Slice are supported"`
	IsMapKey  bool                 `desc:"for OwnKind = Map, this value represents the Key -- otherwise the Value"`
	Owner     interface{}          `desc:"the object that owns this value, either a struct, slice, or map, if non-nil -- if a Ki Node, then SetField is used to set value, to provide proper updating"`
	Field     *reflect.StructField `desc:"if Owner is a struct, this is the reflect.StructField associated with the value"`
	Tags      map[string]string    `desc:"set of tags that can be set to customize interface for different types of values -- only source for non-structfield values"`
	Key       interface{}          `desc:"if Owner is a map, and this is a value, this is the key for this value in the map"`
	KeyView   ValueView            `desc:"if Owner is a map, and this is a value, this is the value view representing the key -- its value has the *current* value of the key, which can be edited"`
	Idx       int                  `desc:"if Owner is a slice, this is the index for the value in the slice"`
	WidgetTyp reflect.Type         `desc:"type of widget to create -- cached during WidgetType method -- chosen based on the ValueView type and reflect.Value type -- see ValueViewer interface"`
	Widget    gi.Node2D            `desc:"the widget used to display and edit the value in the interface -- this is created for us externally and we cache it during ConfigWidget"`
	TmpSave   ValueView            `desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
}

var KiT_ValueViewBase = kit.Types.AddType(&ValueViewBase{}, ValueViewBaseProps)

var ValueViewBaseProps = ki.Props{
	"base-type": true,
}

func (vv *ValueViewBase) AsValueViewBase() *ValueViewBase {
	return vv
}

func (vv *ValueViewBase) SetStructValue(val reflect.Value, owner interface{}, field *reflect.StructField, tmpSave ValueView) {
	vv.OwnKind = reflect.Struct
	vv.Value = val
	vv.Owner = owner
	vv.Field = field
	vv.TmpSave = tmpSave
	vv.SetName(field.Name)
}

func (vv *ValueViewBase) SetMapKey(key reflect.Value, owner interface{}, tmpSave ValueView) {
	vv.OwnKind = reflect.Map
	vv.IsMapKey = true
	vv.Value = key
	vv.Owner = owner
	vv.TmpSave = tmpSave
	vv.SetName(kit.ToString(key.Interface()))
}

func (vv *ValueViewBase) SetMapValue(val reflect.Value, owner interface{}, key interface{}, keyView ValueView, tmpSave ValueView) {
	vv.OwnKind = reflect.Map
	vv.Value = val
	vv.Owner = owner
	vv.Key = key
	vv.KeyView = keyView
	vv.TmpSave = tmpSave
	vv.SetName(kit.ToString(key))
}

func (vv *ValueViewBase) SetSliceValue(val reflect.Value, owner interface{}, idx int, tmpSave ValueView) {
	vv.OwnKind = reflect.Slice
	vv.Value = val
	vv.Owner = owner
	vv.Idx = idx
	vv.TmpSave = tmpSave
	vv.SetName(fmt.Sprintf("%v", idx))
}

func (vv *ValueViewBase) SetStandaloneValue(val reflect.Value) {
	vv.OwnKind = reflect.Invalid
	vv.Value = val
}

// we have this one accessor b/c it is more useful for outside consumers vs. internal usage
func (vv *ValueViewBase) OwnerKind() reflect.Kind {
	return vv.OwnKind
}

func (vv *ValueViewBase) IsInactive() bool {
	if vv.OwnKind == reflect.Struct {
		if _, ok := vv.Tag("inactive"); ok {
			return true
		}
	}
	npv := kit.NonPtrValue(vv.Value)
	if npv.Kind() == reflect.Interface && kit.ValueIsZero(npv) {
		return true
	}
	return false
}

func (vv *ValueViewBase) WidgetType() reflect.Type {
	vv.WidgetTyp = gi.KiT_TextField
	return vv.WidgetTyp
}

func (vv *ValueViewBase) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	tf := vv.Widget.(*gi.TextField)
	npv := kit.NonPtrValue(vv.Value)
	// fmt.Printf("vvb val: %v  type: %v  kind: %v\n", npv.Interface(), npv.Type().String(), npv.Kind())
	if npv.Kind() == reflect.Interface && kit.ValueIsZero(npv) {
		tf.SetText("nil")
	} else {
		txt := kit.ToString(vv.Value.Interface())
		tf.SetText(txt)
	}
}

func (vv *ValueViewBase) ConfigWidget(widg gi.Node2D) {
	vv.Widget = widg
	tf := vv.Widget.(*gi.TextField)
	tf.SetStretchMaxWidth()
	tf.Tooltip, _ = vv.Tag("desc")
	tf.SetProp("min-width", units.NewValue(16, units.Ch))
	if widthtag, ok := vv.Tag("width"); ok {
		width, err := strconv.ParseFloat(widthtag, 32)
		if err == nil {
			tf.SetMinPrefWidth(units.NewValue(float32(width), units.Ch))
		}
	}
	if maxwidthtag, ok := vv.Tag("max-width"); ok {
		width, err := strconv.ParseFloat(maxwidthtag, 32)
		if err == nil {
			tf.SetProp("max-width", units.NewValue(float32(width), units.Ch))
		}
	}

	bitflag.SetState(tf.Flags(), vv.IsInactive(), int(gi.Inactive))
	tf.TextFieldSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) {
			vvv, _ := recv.Embed(KiT_ValueViewBase).(*ValueViewBase)
			tf := send.(*gi.TextField)
			if vvv.SetValue(tf.Text()) {
				vvv.UpdateWidget() // always update after setting value..
			}
		}
	})
	vv.UpdateWidget()
}

func (vv *ValueViewBase) HasAction() bool {
	return false
}

func (vv *ValueViewBase) Activate(vp *gi.Viewport2D, recv ki.Ki, fun ki.RecvFunc) {
}

func (vv *ValueViewBase) Val() reflect.Value {
	return vv.Value
}

func (vv *ValueViewBase) SetValue(val interface{}) bool {
	if vv.This.(ValueView).IsInactive() {
		return false
	}
	rval := false
	if vv.Owner != nil {
		switch vv.OwnKind {
		case reflect.Struct:
			if kiv, ok := vv.Owner.(ki.Ki); ok {
				rval = kiv.SetField(vv.Field.Name, val)

			} else {
				rval = kit.SetRobust(kit.PtrValue(vv.Value).Interface(), val)
			}
		case reflect.Map:
			ov := kit.NonPtrValue(reflect.ValueOf(vv.Owner))
			if vv.IsMapKey {
				nv := reflect.ValueOf(val)                // new key value
				cv := ov.MapIndex(vv.Value)               // get current value
				ov.SetMapIndex(vv.Value, reflect.Value{}) // delete old key
				ov.SetMapIndex(nv, cv)                    // set new key to current value
				vv.Value = nv                             // update value to new key
				rval = true
			} else {
				vv.Value = reflect.ValueOf(val)
				if vv.KeyView != nil {
					ck := vv.KeyView.Val() // current key value
					ov.SetMapIndex(ck, vv.Value)
				} else { // static, key not editable?
					ov.SetMapIndex(reflect.ValueOf(vv.Key), vv.Value)
				}
				rval = true
			}
		case reflect.Slice:
			rval = kit.SetRobust(kit.PtrValue(vv.Value).Interface(), val)
		}
	} else {
		rval = kit.SetRobust(kit.PtrValue(vv.Value).Interface(), val)
	}
	if rval {
		vv.This.(ValueView).SaveTmp()
	}
	// fmt.Printf("value view: %T sending for setting val %v\n", vv.This, val)
	vv.ViewSig.Emit(vv.This, 0, nil)
	return rval
}

func (vv *ValueViewBase) SaveTmp() {
	if vv.TmpSave == nil {
		return
	}
	if vv.TmpSave == vv.This.(ValueView) {
		// if we are a map value, of a struct value, we save our value
		if vv.Owner != nil && vv.OwnKind == reflect.Map && !vv.IsMapKey {
			if kit.NonPtrValue(vv.Value).Kind() == reflect.Struct {
				ov := kit.NonPtrValue(reflect.ValueOf(vv.Owner))
				if vv.KeyView != nil {
					ck := vv.KeyView.Val()
					ov.SetMapIndex(ck, kit.NonPtrValue(vv.Value))
					// fmt.Printf("save tmp of struct value in key: %v\n", ck.Interface())
				} else {
					ov.SetMapIndex(reflect.ValueOf(vv.Key), kit.NonPtrValue(vv.Value))
					// fmt.Printf("save tmp of struct value in key: %v\n", vv.Key)
				}
			}
		}
	} else {
		vv.TmpSave.SaveTmp()
	}
}

func (vv *ValueViewBase) CreateTempIfNotPtr() bool {
	if vv.Value.Kind() != reflect.Ptr { // we create a temp variable -- SaveTmp will save it!
		vv.TmpSave = vv.This.(ValueView) // we are it!
		vtyp := reflect.TypeOf(vv.Value.Interface())
		vtp := reflect.New(vtyp)
		// fmt.Printf("vtyp: %v %v %v, vtp: %v %v %T\n", vtyp, vtyp.Name(), vtyp.String(), vtp, vtp.Type(), vtp.Interface())
		kit.SetRobust(vtp.Interface(), vv.Value.Interface())
		vv.Value = vtp // use this instead
		return true
	}
	return false
}

func (vv *ValueViewBase) SetTags(tags map[string]string) {
	if vv.Tags == nil {
		vv.Tags = make(map[string]string, len(tags))
	}
	for tag, val := range tags {
		vv.Tags[tag] = val
	}
}

func (vv *ValueViewBase) SetTag(tag, value string) {
	if vv.Tags == nil {
		vv.Tags = make(map[string]string, 10)
	}
	vv.Tags[tag] = value
}

func (vv *ValueViewBase) Tag(tag string) (string, bool) {
	if vv.Tags != nil {
		if tv, ok := vv.Tags[tag]; ok {
			return tv, ok
		}
	}
	if !(vv.Owner != nil && vv.OwnKind == reflect.Struct) {
		return "", false
	}
	return vv.Field.Tag.Lookup(tag)
}

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
	txt := fmt.Sprintf("%T", npv.Interface())
	ac.SetText(txt)
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
			sv.ViewSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				vv, _ := recv.Embed(KiT_SliceValueView).(*SliceValueView)
				vv.UpdateWidget()
				vv.ViewSig.Emit(vv.This, 0, nil)
			})
		}
	}
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
		vvv, _ := recv.Embed(KiT_BoolValueView).(*BoolValueView)
		cbb := vvv.Widget.(*gi.CheckBox)
		if vvv.SetValue(cbb.IsChecked()) {
			vvv.UpdateWidget() // always update after setting value..
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
		min, err := strconv.ParseFloat(mintag, 32)
		if err == nil {
			sb.SetMin(float32(min))
		}
	}
	if maxtag, ok := vv.Tag("max"); ok {
		max, err := strconv.ParseFloat(maxtag, 32)
		if err == nil {
			sb.SetMax(float32(max))
		}
	}
	if steptag, ok := vv.Tag("step"); ok {
		step, err := strconv.ParseFloat(steptag, 32)
		if err == nil {
			sb.Step = float32(step)
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
		min, err := strconv.ParseFloat(mintag, 32)
		if err == nil {
			sb.HasMin = true
			sb.Min = float32(min)
		}
	}
	if maxtag, ok := vv.Tag("max"); ok {
		max, err := strconv.ParseFloat(maxtag, 32)
		if err == nil {
			sb.HasMax = true
			sb.Max = float32(max)
		}
	}
	if steptag, ok := vv.Tag("step"); ok {
		step, err := strconv.ParseFloat(steptag, 32)
		if err == nil {
			sb.Step = float32(step)
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
// ViewIFace

// giv.ViewIFace is THE implementation of the gi.ViewIFace interface
type ViewIFace struct {
}

func (vi *ViewIFace) CtxtMenuView(val interface{}, inactive bool, vp *gi.Viewport2D, menu *gi.Menu) bool {
	return CtxtMenuView(val, inactive, vp, menu)
}

func (vi *ViewIFace) GoGiEditor(obj ki.Ki) {
	GoGiEditorDialog(obj)
}

func (vi *ViewIFace) PrefsView(prefs *gi.Preferences) {
	PrefsView(prefs)
}
