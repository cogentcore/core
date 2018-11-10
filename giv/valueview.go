// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/goki/gi"
	"github.com/goki/gi/histyle"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

func init() {
	gi.TheViewIFace = &ViewIFace{}
}

// MapInlineLen is the number of map elements at or below which an inline
// representation of the map will be presented -- more convenient for small
// #'s of props
var MapInlineLen = 3

// StructInlineLen is the number of elemental struct fields at or below which an inline
// representation of the struct will be presented -- more convenient for small structs
var StructInlineLen = 6

// SliceInlineLen is the number of slice elements below which inline will be used
var SliceInlineLen = 6

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
// falls back on default Kind-based options.  tags are optional tags, e.g.,
// from the field in a struct, that control the view properties -- see the gi wiki
// for details on supported tags -- these are NOT set for the view element, only
// used for options that affect what kind of view to create.
// See FieldToValueView for version that takes into account the properties of the owner.
func ToValueView(it interface{}, tags string) ValueView {
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
	if nptyp == reflect.TypeOf(key.Chord("")) {
		vv := KeyChordValueView{}
		vv.Init(&vv)
		return &vv
	}
	if nptyp == reflect.TypeOf(histyle.StyleName("")) {
		vv := HiStyleValueView{}
		vv.Init(&vv)
		return &vv
	}

	forceInline := false
	forceNoInline := false

	tprops := kit.Types.Properties(typ, false) // don't make
	if tprops != nil {
		if inprop, ok := kit.TypeProp(*tprops, "inline"); ok {
			forceInline, ok = kit.ToBool(inprop)
		}
		if inprop, ok := kit.TypeProp(*tprops, "no-inline"); ok {
			forceNoInline, ok = kit.ToBool(inprop)
		}
	}

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
			return ToValueView(v.Elem().Interface(), tags)
		}
	case nptyp == ki.KiT_Signal:
		return nil
	case vk == reflect.Array:
		fallthrough
	case vk == reflect.Slice:
		v := reflect.ValueOf(it)
		sz := v.Len()
		eltyp := kit.SliceElType(it)
		if _, ok := it.([]byte); ok {
			vv := ByteSliceValueView{}
			vv.Init(&vv)
			return &vv
		}
		if _, ok := it.([]rune); ok {
			vv := RuneSliceValueView{}
			vv.Init(&vv)
			return &vv
		}
		isstru := (kit.NonPtrType(eltyp).Kind() == reflect.Struct)
		if !forceNoInline && (forceInline || (!isstru && sz <= SliceInlineLen && !ki.IsKi(eltyp))) {
			vv := SliceInlineValueView{}
			vv.Init(&vv)
			return &vv
		} else {
			vv := SliceValueView{}
			vv.Init(&vv)
			return &vv
		}
	case vk == reflect.Map:
		v := reflect.ValueOf(it)
		sz := v.Len()
		sz = kit.MapStructElsN(it)
		if !forceNoInline && (forceInline || sz <= MapInlineLen) {
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
		nfld := kit.AllFieldsN(typ)
		if !forceNoInline && (forceInline || nfld <= StructInlineLen) {
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
		return ToValueView(fval, "")
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

	typ := reflect.TypeOf(it)
	nptyp := kit.NonPtrType(typ)
	ftyp, ok := nptyp.FieldByName(field)
	if ok {
		return ToValueView(fval, string(ftyp.Tag))
	}
	return ToValueView(fval, "")
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

	// AllTags returns all the tags for this value view, from structfield or set
	// specifically using SetTag* methods
	AllTags() map[string]string

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

func (vv *ValueViewBase) HasAction() bool {
	return false
}

func (vv *ValueViewBase) Activate(vp *gi.Viewport2D, recv ki.Ki, fun ki.RecvFunc) {
}

func (vv *ValueViewBase) Val() reflect.Value {
	return vv.Value
}

func (vv *ValueViewBase) SetValue(val interface{}) bool {
	if vv.This().(ValueView).IsInactive() {
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
				nv := kit.NonPtrValue(reflect.ValueOf(val)) // new key value
				kv := kit.NonPtrValue(vv.Value)
				cv := ov.MapIndex(kv)    // get current value
				curnv := ov.MapIndex(nv) // see if new value there already
				if val != kv.Interface() && !kit.ValueIsZero(curnv) {
					var vp *gi.Viewport2D
					if vv.Widget != nil {
						widg := vv.Widget.AsNode2D()
						vp = widg.Viewport
					}
					// actually new key and current exists
					gi.ChoiceDialog(vp,
						gi.DlgOpts{Title: "Map Key Conflict", Prompt: fmt.Sprintf("The map key value: %v already exists in the map -- are you sure you want to overwrite the current value?", val)},
						[]string{"Cancel Change", "Overwrite"},
						vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
							switch sig {
							case 0:
								if vp != nil {
									vp.FullRender2DTree()
								}
							case 1:
								cv := ov.MapIndex(kv)               // get current value
								ov.SetMapIndex(kv, reflect.Value{}) // delete old key
								ov.SetMapIndex(nv, cv)              // set new key to current value
								vv.Value = nv                       // update value to new key
								vv.This().(ValueView).SaveTmp()
								vv.ViewSig.Emit(vv.This(), 0, nil)
								if vp != nil {
									vp.FullRender2DTree()
								}
							}
						})
					return false // abort this action right now
				}
				ov.SetMapIndex(kv, reflect.Value{}) // delete old key
				ov.SetMapIndex(nv, cv)              // set new key to current value
				vv.Value = nv                       // update value to new key
				rval = true
			} else {
				vv.Value = reflect.ValueOf(val)
				if vv.KeyView != nil {
					ck := kit.NonPtrValue(vv.KeyView.Val()) // current key value
					ov.SetMapIndex(ck, vv.Value)
				} else { // static, key not editable?
					ov.SetMapIndex(kit.NonPtrValue(reflect.ValueOf(vv.Key)), vv.Value)
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
		vv.This().(ValueView).SaveTmp()
	}
	// fmt.Printf("value view: %T sending for setting val %v\n", vv.This(), val)
	vv.ViewSig.Emit(vv.This(), 0, nil)
	return rval
}

func (vv *ValueViewBase) SaveTmp() {
	if vv.TmpSave == nil {
		return
	}
	if vv.TmpSave == vv.This().(ValueView) {
		// if we are a map value, of a struct value, we save our value
		if vv.Owner != nil && vv.OwnKind == reflect.Map && !vv.IsMapKey {
			if kit.NonPtrValue(vv.Value).Kind() == reflect.Struct {
				ov := kit.NonPtrValue(reflect.ValueOf(vv.Owner))
				if vv.KeyView != nil {
					ck := kit.NonPtrValue(vv.KeyView.Val())
					ov.SetMapIndex(ck, kit.NonPtrValue(vv.Value))
					// fmt.Printf("save tmp of struct value in key: %v\n", ck.Interface())
				} else {
					ov.SetMapIndex(kit.NonPtrValue(reflect.ValueOf(vv.Key)), kit.NonPtrValue(vv.Value))
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
		vv.TmpSave = vv.This().(ValueView) // we are it!
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

func (vv *ValueViewBase) AllTags() map[string]string {
	rvt := make(map[string]string)
	if vv.Tags != nil {
		for key, val := range vv.Tags {
			rvt[key] = val
		}
	}
	if !(vv.Owner != nil && vv.OwnKind == reflect.Struct) {
		return rvt
	}
	smap := kit.StructTags(vv.Field.Tag)
	for key, val := range smap {
		rvt[key] = val
	}
	return rvt
}

// OwnerLabel returns some extra info about the owner of this value view
// which is useful to put in title of our object
func (vv *ValueViewBase) OwnerLabel() string {
	switch vv.OwnKind {
	case reflect.Struct:
		olbl := gi.ToLabeler(vv.Owner)
		if olbl != "" {
			return olbl + "." + vv.Field.Name
		}
		return vv.Field.Name
	case reflect.Map:
		if vv.IsMapKey {
			kv := kit.NonPtrValue(vv.Value)
			return kit.ToString(kv.Interface())
		} else {
			if vv.KeyView != nil {
				ck := kit.NonPtrValue(vv.KeyView.Val()) // current key value
				return kit.ToString(ck.Interface())
			} else {
				return kit.ToString(vv.Key)
			}
		}
	case reflect.Slice:
		return kit.ToString(kit.PtrValue(vv.Value).Interface())
	}
	return ""
}

////////////////////////////////////////////////////////////////////////////////////////
//   Base Widget Functions -- these are typically redefined in ValueView subtypes

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
	tf.SetInactiveState(vv.This().(ValueView).IsInactive())
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
	if completetag, ok := vv.Tag("complete"); ok {
		in := []reflect.Value{reflect.ValueOf(tf)}
		in = append(in, reflect.ValueOf(completetag)) // pass tag value - object may doing completion on multiple fields
		cmpfv := reflect.ValueOf(vv.Owner).MethodByName("SetCompleter")
		if kit.ValueIsZero(cmpfv) {
			log.Printf("giv.ValueViewBase: programmer error -- SetCompleter method not found in type: %T\n", vv.Owner)
		} else {
			cmpfv.Call(in)
		}
	}

	tf.TextFieldSig.ConnectOnly(vv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.TextFieldDone) || sig == int64(gi.TextFieldDeFocused) {
			vvv, _ := recv.Embed(KiT_ValueViewBase).(*ValueViewBase)
			tf := send.(*gi.TextField)
			if vvv.SetValue(tf.Text()) {
				vvv.UpdateWidget() // always update after setting value..
			}
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

func (vi *ViewIFace) KeyMapsView(maps *gi.KeyMaps) {
	KeyMapsView(maps)
}

func (vi *ViewIFace) PrefsDetView(prefs *gi.PrefsDetailed) {
	PrefsDetView(prefs)
}

func (vi *ViewIFace) HiStylesView(styles interface{}) {
	HiStylesView(styles.(*histyle.Styles))
}

func (vi *ViewIFace) PrefsDetDefaults(pf *gi.PrefsDetailed) {
	pf.TextViewClipHistMax = TextViewClipHistMax
	pf.MapInlineLen = MapInlineLen
	pf.StructInlineLen = StructInlineLen
	pf.SliceInlineLen = SliceInlineLen
}

func (vi *ViewIFace) PrefsDetApply(pf *gi.PrefsDetailed) {
	TextViewClipHistMax = pf.TextViewClipHistMax
	MapInlineLen = pf.MapInlineLen
	StructInlineLen = pf.StructInlineLen
	SliceInlineLen = pf.SliceInlineLen
}

func (vi *ViewIFace) PrefsDbgView(prefs *gi.PrefsDebug) {
	PrefsDbgView(prefs)
}
