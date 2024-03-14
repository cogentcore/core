// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

//go:generate core generate

import (
	"fmt"
	"log/slog"
	"reflect"
	"strconv"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/states"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
)

// NewValue makes and returns a new [Value] from the given value and creates
// the widget for it with the given parent and optional tags (only the first
// argument is used). It is the main way that end-user code should interact
// with [Value]s. The given value needs to be a pointer for it to be modified.
//
// NewValue is not appropriate for internal code configuring
// non-solo values (for example, in StructView), but it should be fine
// for end-user code.
func NewValue(par ki.Ki, val any, tags ...string) Value {
	t := ""
	if len(tags) > 0 {
		t = tags[0]
	}
	v := ToValue(val, t)
	v.SetSoloValue(reflect.ValueOf(val))
	w := par.NewChild(v.WidgetType()).(gi.Widget)
	Config(v, w)
	return v
}

// A Value is a bridge between Go values like strings, integers, and structs
// and GUI widgets. It allows you to represent simple and complicated values
// of any kind with automatic, editable, and user-friendly widgets. The most
// common pathway for making [Value]s is [NewValue].
type Value interface {
	fmt.Stringer

	// AsValueData gives access to the basic data fields so that the
	// interface doesn't need to provide accessors for them.
	AsValueData() *ValueData

	// AsWidget returns the widget associated with the value.
	AsWidget() gi.Widget

	// AsWidgetBase returns the widget base associated with the value.
	AsWidgetBase() *gi.WidgetBase

	// WidgetType returns the type of widget associated with this value.
	WidgetType() *gti.Type

	// Config configures the widget to represent the value, including setting up
	// the OnChange event listener to set the value when the user edits it
	// (values are always set immediately when the widget is updated).
	// You should typically call the global [Config] function instead of this;
	// this is the method that values implement, which is called in the global
	// Config helper function.
	Config()

	// Update updates the widget representation to reflect the current value.
	Update()

	// Name returns the name of the value
	Name() string

	// SetName sets the name of the value
	SetName(name string)

	// Label returns the label for the value
	Label() string

	// SetLabel sets the label for the value
	SetLabel(label string) *ValueData

	// Doc returns the documentation for the value
	Doc() string

	// SetDoc sets the documentation for the value
	SetDoc(doc string) *ValueData

	// Is checks if flag is set, using atomic, safe for concurrent access
	Is(f enums.BitFlag) bool

	// SetFlag sets the given flag(s) to given state
	// using atomic, safe for concurrent access
	SetFlag(on bool, f ...enums.BitFlag)

	// SetStructValue sets the value, owner and field information for a struct field.
	SetStructValue(val reflect.Value, owner any, field *reflect.StructField, tmpSave Value, viewPath string)

	// SetMapKey sets the key value and owner for a map key.
	SetMapKey(val reflect.Value, owner any, tmpSave Value)

	// SetMapValue sets the value, owner and map key information for a map
	// element -- needs pointer to Value representation of key to track
	// current key value.
	SetMapValue(val reflect.Value, owner any, key any, keyView Value, tmpSave Value, viewPath string)

	// SetSliceValue sets the value, owner and index information for a slice element.
	SetSliceValue(val reflect.Value, owner any, idx int, tmpSave Value, viewPath string)

	// SetSoloValue sets the value for a singleton standalone value
	// (e.g., for arg values).
	SetSoloValue(val reflect.Value)

	// OwnerKind returns the reflect.Kind of the owner: Struct, Map, or Slice
	// (or Invalid for standalone values such as args).
	OwnerKind() reflect.Kind

	// IsReadOnly returns whether the value is ReadOnly, which prevents modification
	// of the underlying Value.  Can be flagged by container views, or
	// Map owners have ReadOnly values, and fields can be marked
	// as ReadOnly using a struct tag.
	IsReadOnly() bool

	// SetReadOnly marks this value as ReadOnly or not
	SetReadOnly(ro bool)

	// Val returns the reflect.Value representation for this item.
	Val() reflect.Value

	// SetValue assigns given value to this item (if not ReadOnly), using
	// Ki.SetField for Ki types and laser.SetRobust otherwise -- emits a ViewSig
	// signal when set.
	SetValue(val any) bool

	// SendChange sends events.Change event to all listeners registered on this view.
	// This is the primary notification event for all Value elements.
	// It takes an optional original event to base the event on.
	SendChange(orig ...events.Event)

	// OnChange registers given listener function for Change events on Value.
	// This is the primary notification event for all Value elements.
	OnChange(fun func(e events.Event))

	// SetTags sets tags for this valueview, for non-struct values, to
	// influence interface for this value -- see
	// https://cogentcore.org/core/wiki/Tags for valid options.  Adds to
	// existing tags if some are already set.
	SetTags(tags map[string]string)

	// SetTag sets given tag to given value for this valueview, for non-struct
	// values, to influence interface for this value -- see
	// https://cogentcore.org/core/wiki/Tags for valid options.
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
	// elements -- each Value has a pointer to any parent Value that
	// might need to be saved after SetValue -- SaveTmp called automatically
	// in SetValue but other cases that use something different need to call
	// it explicitly.
	SaveTmp()
}

// ConfigDialoger is an optional interface that [Value]s may implement to
// indicate that they have a dialog associated with them that is configured
// with the ConfigDialog method. The dialog body itself is constructed and run
// using [OpenDialog].
type ConfigDialoger interface {
	// ConfigDialog adds content to the given dialog body for this value.
	// The bool return is false if the value does not use this method
	// (e.g., for simple menu choosers).
	// The returned function is an optional closure to be called
	// in the Ok case, for cases where extra logic is required.
	ConfigDialog(d *gi.Body) (bool, func())
}

// OpenDialoger is an optional interface that [Value]s may implement to
// indicate that they have a dialog associated with them that is created,
// configured, and run with the OpenDialog method. This method typically
// calls a separate ConfigDialog method. If the [Value] does not implement
// [OpenDialoger] but does implement [ConfigDialoger], then [OpenDialogBase]
// will be used to create and run the dialog, and [ConfigDialoger.ConfigDialog]
// will be used to configure it.
type OpenDialoger interface {
	// OpenDialog opens the dialog for this Value.
	// Given function closure is called for the Ok action, after value
	// has been updated, if using the dialog as part of another control flow.
	// Note that some cases just pop up a menu chooser, not a full dialog.
	OpenDialog(ctx gi.Widget, fun func())
}

// ValueBase is the base type that all [Value] objects extend. It contains both
// [ValueData] and a generically parameterized [gi.Widget].
type ValueBase[W gi.Widget] struct {
	ValueData

	// Widget is the GUI widget used to display and edit the value in the GUI.
	Widget W
}

func (v *ValueBase[W]) AsWidget() gi.Widget {
	return v.Widget
}

func (v *ValueBase[W]) AsWidgetBase() *gi.WidgetBase {
	return v.Widget.AsWidget()
}

func (v *ValueBase[W]) WidgetType() *gti.Type {
	var w W
	return w.KiType()
}

// Config configures the given [gi.Widget] to represent the given [Value].
func Config(v Value, w gi.Widget) {
	if w == v.AsWidget() {
		v.Update()
		return
	}
	ConfigBase(v, w)
	v.Config()
	v.Update()
}

// ConfigBase does the base configuration for the given [gi.Widget]
// to represent the given [Value].
func ConfigBase(v Value, w gi.Widget) {
	w.AsWidget().SetTooltip(v.Doc())
	w.SetState(v.IsReadOnly(), states.ReadOnly) // do right away
	w.Style(func(s *styles.Style) {
		s.SetState(v.IsReadOnly(), states.ReadOnly) // and in style
		if tv, ok := v.Tag("width"); ok {
			v, err := laser.ToFloat32(tv)
			if err == nil {
				s.Min.X.Ch(v)
			}
		}
		if tv, ok := v.Tag("max-width"); ok {
			v, err := laser.ToFloat32(tv)
			if err == nil {
				if v < 0 {
					s.Grow.X = 1 // support legacy
				} else {
					s.Max.X.Ch(v)
				}
			}
		}
		if tv, ok := v.Tag("height"); ok {
			v, err := laser.ToFloat32(tv)
			if err == nil {
				s.Min.Y.Em(v)
			}
		}
		if tv, ok := v.Tag("max-height"); ok {
			v, err := laser.ToFloat32(tv)
			if err == nil {
				if v < 0 {
					s.Grow.Y = 1
				} else {
					s.Max.Y.Em(v)
				}
			}
		}
		if tv, ok := v.Tag("grow"); ok {
			v, err := laser.ToFloat32(tv)
			if err == nil {
				s.Grow.X = v
			}
		}
		if tv, ok := v.Tag("grow-y"); ok {
			v, err := laser.ToFloat32(tv)
			if err == nil {
				s.Grow.Y = v
			}
		}
	})
}

// OpenDialog opens any applicable dialog for the given value in the
// context of the given widget. It first tries [OpenDialoger], then
// [ConfigDialoger] with [OpenDialogBase]. If both of those fail, it
// returns false.
func OpenDialog(v Value, ctx gi.Widget, fun func()) bool {
	if od, ok := v.(OpenDialoger); ok {
		od.OpenDialog(ctx, fun)
		return true
	}
	if cd, ok := v.(ConfigDialoger); ok {
		OpenDialogBase(v, cd, ctx, fun)
		return true
	}
	return false
}

// OpenDialogBase is a helper for [OpenDialog] for cases that
// do not implement [OpenDialoger] but do implement [ConfigDialoger]
// to configure the dialog contents.
func OpenDialogBase(v Value, cd ConfigDialoger, ctx gi.Widget, fun func()) {
	vd := v.AsValueData()
	title, _, _ := vd.GetTitle()
	opv := laser.OnePtrUnderlyingValue(vd.Value)
	if !opv.IsValid() {
		return
	}
	obj := opv.Interface()
	if gi.RecycleDialog(obj) {
		return
	}
	d := gi.NewBody().AddTitle(title).AddText(v.Doc())
	ok, okfun := cd.ConfigDialog(d)
	if !ok {
		return
	}

	// if we don't have anything specific for ok events,
	// we just register an OnClose event and skip the
	// OK and Cancel buttons
	if okfun == nil && fun == nil {
		d.OnClose(func(e events.Event) {
			v.Update()
			v.SendChange()
		})
	} else {
		// otherwise, we have to make the bottom bar
		d.AddBottomBar(func(pw gi.Widget) {
			d.AddCancel(pw)
			d.AddOk(pw).OnClick(func(e events.Event) {
				if okfun != nil {
					okfun()
				}
				v.Update()
				v.SendChange()
				if fun != nil {
					fun()
				}
			})
		})
	}

	ds := d.NewFullDialog(ctx)
	if v.Is(ValueDialogNewWindow) {
		ds.SetNewWindow(true)
	}
	ds.Run()
}

func (v *ValueBase[W]) SetValue(val any) bool {
	if v.IsReadOnly() {
		return false
	}
	var err error
	wasSet := false
	if v.Owner != nil {
		switch v.OwnKind {
		case reflect.Struct:
			err = laser.SetRobust(laser.PtrValue(v.Value).Interface(), val)
			wasSet = true
		case reflect.Map:
			wasSet, err = v.SetValueMap(val)
		case reflect.Slice:
			err = laser.SetRobust(laser.PtrValue(v.Value).Interface(), val)
		}
		if updtr, ok := v.Owner.(gi.Updater); ok {
			// fmt.Printf("updating: %v\n", updtr)
			updtr.Update()
		}
	} else {
		err = laser.SetRobust(laser.PtrValue(v.Value).Interface(), val)
		wasSet = true
	}
	if wasSet {
		v.SaveTmp()
	}
	// fmt.Printf("value view: %T sending for setting val %v\n", v.This(), val)
	v.SendChange()
	if err != nil {
		// todo: snackbar for error?
		slog.Error("giv.SetValue error", "type", v.Value.Type(), "err", err)
	}
	return wasSet
}

func (v *ValueBase[W]) SetValueMap(val any) (bool, error) {
	ov := laser.NonPtrValue(reflect.ValueOf(v.Owner))
	wasSet := false
	var err error
	if v.Is(ValueMapKey) {
		nv := laser.NonPtrValue(reflect.ValueOf(val)) // new key value
		kv := laser.NonPtrValue(v.Value)
		cv := ov.MapIndex(kv)    // get current value
		curnv := ov.MapIndex(nv) // see if new value there already
		if val != kv.Interface() && curnv.IsValid() && !curnv.IsZero() {
			// actually new key and current exists
			d := gi.NewBody().AddTitle("Map key conflict").
				AddText(fmt.Sprintf("The map key value: %v already exists in the map; are you sure you want to overwrite the current value?", val))
			d.AddBottomBar(func(pw gi.Widget) {
				d.AddCancel(pw).SetText("Cancel change")
				d.AddOk(pw).SetText("Overwrite").OnClick(func(e events.Event) {
					cv := ov.MapIndex(kv)               // get current value
					ov.SetMapIndex(kv, reflect.Value{}) // delete old key
					ov.SetMapIndex(nv, cv)              // set new key to current value
					v.Value = nv                        // update value to new key
					v.SaveTmp()
					v.SendChange()
				})
			})
			d.NewDialog(v.Widget).Run()
			return false, nil // abort this action right now
		}
		ov.SetMapIndex(kv, reflect.Value{}) // delete old key
		ov.SetMapIndex(nv, cv)              // set new key to current value
		v.Value = nv                        // update value to new key
		wasSet = true
	} else {
		v.Value = laser.NonPtrValue(reflect.ValueOf(val))
		if v.KeyView != nil {
			ck := laser.NonPtrValue(v.KeyView.Val())                  // current key value
			wasSet = laser.SetMapRobust(ov, ck, reflect.ValueOf(val)) // todo: error
		} else { // static, key not editable?
			wasSet = laser.SetMapRobust(ov, laser.NonPtrValue(reflect.ValueOf(v.Key)), v.Value) // todo: error
		}
		// wasSet = true
	}
	return wasSet, err
}

// note: could have a more efficient way to represent the different owner type
// data (Key vs. Field vs. Idx), instead of just having everything for
// everything.  However, Value itself gets customized for different target
// value types, and those are orthogonal to the owner type, so need a separate
// ValueOwner class that encodes these options more efficiently -- but
// that introduces another struct alloc and pointer -- not clear if it is
// worth it..

// ValueData contains the base data common to all [Value] objects.
// [Value] objects should extend [ValueBase], not ValueData.
type ValueData struct {

	// Nm is locally-unique name of Value
	Nm string

	// SavedLabel is the label for the Value
	SavedLabel string

	// SavedDoc is the saved documentation for the Value, if any
	// (only valid if [ValueHasSaveDoc] is true)
	SavedDoc string

	// Flags are atomic bit flags for Value state
	Flags ValueFlags

	// the reflect.Value representation of the value
	Value reflect.Value `set:"-"`

	// kind of owner that we have -- reflect.Struct, .Map, .Slice are supported
	OwnKind reflect.Kind

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// the object that owns this value, either a struct, slice, or map, if non-nil -- if a Ki Node, then SetField is used to set value, to provide proper updating
	Owner any

	// if Owner is a struct, this is the reflect.StructField associated with the value
	Field *reflect.StructField

	// set of tags that can be set to customize interface for different types of values -- only source for non-structfield values
	Tags map[string]string `set:"-"`

	// if Owner is a map, and this is a value, this is the key for this value in the map
	Key any `set:"-" edit:"-"`

	// if Owner is a map, and this is a value, this is the value view representing the key -- its value has the *current* value of the key, which can be edited
	KeyView Value `set:"-" edit:"-"`

	// if Owner is a slice, this is the index for the value in the slice
	Idx int `set:"-" edit:"-"`

	// Listeners are event listener functions for processing events on this widget.
	// type specific Listeners are added in OnInit when the widget is initialized.
	Listeners events.Listeners `set:"-" view:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `set:"-" view:"-"`
}

// ValueFlags for Value bool state
type ValueFlags int64 //enums:bitflag -trim-prefix Value

const (
	// ValueReadOnly flagged after first configuration
	ValueReadOnly ValueFlags = iota

	// ValueMapKey for OwnKind = Map, this value represents the Key -- otherwise the Value
	ValueMapKey

	// ValueHasSavedLabel is whether the value has a saved version of its
	// label, which can be set either automatically or explicitly
	ValueHasSavedLabel

	// ValueHasSavedDoc is whether the value has a saved version of its
	// documentation, which can be set either automatically or explicitly
	ValueHasSavedDoc

	// ValueDialogNewWindow indicates that the dialog should be opened with
	// in a new window, instead of a typical FullWindow in same current window.
	// this is triggered by holding down any modifier key while clicking on a
	// button that opens the window.
	ValueDialogNewWindow
)

func (v *ValueData) AsValueData() *ValueData {
	return v
}

func (v *ValueData) Name() string {
	return v.Nm
}

func (v *ValueData) SetName(name string) {
	v.Nm = name
}

func (v *ValueData) Label() string {
	if v.Is(ValueHasSavedLabel) {
		return v.SavedLabel
	}

	lbl := ""
	lbltag, has := v.Tag("label")

	// whether to sentence case
	sc := true
	if v.Owner != nil && len(NoSentenceCaseFor) > 0 {
		sc = !NoSentenceCaseForType(gti.TypeNameObj(v.Owner))
	}

	switch {
	case has:
		lbl = lbltag
	case v.Field != nil:
		lbl = v.Field.Name
		if sc {
			lbl = strcase.ToSentence(lbl)
		}
	default:
		lbl = v.Nm
		if sc {
			lbl = strcase.ToSentence(lbl)
		}
	}

	v.SavedLabel = lbl
	v.SetFlag(true, ValueHasSavedLabel)
	return v.SavedLabel
}

func (v *ValueData) SetLabel(label string) *ValueData {
	v.SavedLabel = label
	v.SetFlag(true, ValueHasSavedLabel)
	return v
}

func (v *ValueData) Doc() string {
	if v.Is(ValueHasSavedDoc) {
		return v.SavedDoc
	}
	doc, _ := gti.GetDoc(v.Value, reflect.ValueOf(v.Owner), v.Field, v.Label())
	v.SavedDoc = doc
	v.SetFlag(true, ValueHasSavedDoc)
	return v.SavedDoc
}

func (v *ValueData) SetDoc(doc string) *ValueData {
	v.SavedDoc = doc
	v.SetFlag(true, ValueHasSavedDoc)
	return v
}

func (v *ValueData) String() string {
	return v.Nm + ": " + v.Value.String()
}

// Is checks if flag is set, using atomic, safe for concurrent access
func (v *ValueData) Is(f enums.BitFlag) bool {
	return v.Flags.HasFlag(f)
}

// SetFlag sets the given flag(s) to given state
// using atomic, safe for concurrent access
func (v *ValueData) SetFlag(on bool, f ...enums.BitFlag) {
	v.Flags.SetFlag(on, f...)
}

func (v *ValueData) SetReadOnly(ro bool) {
	v.SetFlag(ro, ValueReadOnly)
}

// JoinViewPath returns a view path composed of two elements,
// with a • path separator, handling the cases where either or
// both can be empty.
func JoinViewPath(a, b string) string {
	switch {
	case a == "" && b == "":
		return ""
	case a == "":
		return b
	case b == "":
		return a
	default:
		return a + " • " + b
	}
}

func (v *ValueData) SetStructValue(val reflect.Value, owner any, field *reflect.StructField, tmpSave Value, viewPath string) {
	v.OwnKind = reflect.Struct
	v.Value = val
	v.Owner = owner
	v.Field = field
	v.TmpSave = tmpSave
	v.ViewPath = viewPath
	v.SetName(field.Name)
}

func (v *ValueData) SetMapKey(key reflect.Value, owner any, tmpSave Value) {
	v.OwnKind = reflect.Map
	v.SetFlag(true, ValueMapKey)
	v.Value = key
	v.Owner = owner
	v.TmpSave = tmpSave
	v.SetName(laser.ToString(key.Interface()))
}

func (v *ValueData) SetMapValue(val reflect.Value, owner any, key any, keyView Value, tmpSave Value, viewPath string) {
	v.OwnKind = reflect.Map
	v.Value = val
	v.Owner = owner
	v.Key = key
	v.KeyView = keyView
	v.TmpSave = tmpSave
	keystr := laser.ToString(key)
	v.ViewPath = JoinViewPath(viewPath, keystr)
	v.SetName(keystr)
}

func (v *ValueData) SetSliceValue(val reflect.Value, owner any, idx int, tmpSave Value, viewPath string) {
	v.OwnKind = reflect.Slice
	v.Value = val
	v.Owner = owner
	v.Idx = idx
	v.TmpSave = tmpSave
	idxstr := fmt.Sprintf("%v", idx)
	vpath := viewPath + "[" + idxstr + "]"
	if v.Owner != nil {
		if lblr, ok := v.Owner.(gi.SliceLabeler); ok {
			slbl := lblr.ElemLabel(idx)
			if slbl != "" {
				vpath = JoinViewPath(viewPath, slbl)
			}
		}
	}
	v.ViewPath = vpath
	v.SetName(idxstr)
}

// SetSoloValue sets the value for a singleton standalone value
// (e.g., for arg values).
func (v *ValueData) SetSoloValue(val reflect.Value) {
	v.OwnKind = reflect.Invalid
	// we must ensure that it is a pointer value so that it has
	// an underlying value that updates when changes occur
	v.Value = laser.PtrValue(val)
}

// SetSoloValueIface sets the value for a singleton standalone value
// using an interface for the value -- you must pass a pointer.
// for now, this cannot be a method because gopy doesn't find the
// key comment below that tells it what to do with the interface
// gopy:interface=handle
func SetSoloValueIface(v *ValueData, val any) {
	v.OwnKind = reflect.Invalid
	v.Value = reflect.ValueOf(val)
}

// OwnerKind we have this one accessor b/c it is more useful for outside consumers vs. internal usage
func (v *ValueData) OwnerKind() reflect.Kind {
	return v.OwnKind
}

func (v *ValueData) IsReadOnly() bool {
	if v.Is(ValueReadOnly) {
		return true
	}
	if v.OwnKind == reflect.Struct {
		if et, has := v.Tag("edit"); has && et == "-" {
			v.SetReadOnly(true) // cache
			return true
		}
	}
	npv := laser.NonPtrValue(v.Value)
	if npv.Kind() == reflect.Interface && npv.IsZero() {
		v.SetReadOnly(true) // cache
		return true
	}
	return false
}

func (v *ValueData) HasDialog() bool {
	return false
}

func (v *ValueData) OpenDialog(ctx gi.Widget, fun func()) {
}

func (v *ValueData) ConfigDialog(d *gi.Body) (bool, func()) {
	return false, nil
}

func (v *ValueData) Val() reflect.Value {
	return v.Value
}

// OnChange registers given listener function for Change events on Value.
// This is the primary notification event for all Value elements.
func (v *ValueData) OnChange(fun func(e events.Event)) {
	v.On(events.Change, fun)
}

// On adds an event listener function for the given event type
func (v *ValueData) On(etype events.Types, fun func(e events.Event)) {
	v.Listeners.Add(etype, fun)
}

// SendChange sends events.Change event to all listeners registered on this view.
// This is the primary notification event for all Value elements. It takes
// an optional original event to base the event on.
func (v *ValueData) SendChange(orig ...events.Event) {
	v.Send(events.Change, orig...)
}

// Send sends an NEW event of given type to this widget,
// optionally starting from values in the given original event
// (recommended to include where possible).
// Do NOT send an existing event using this method if you
// want the Handled state to persist throughout the call chain;
// call HandleEvent directly for any existing events.
func (v *ValueData) Send(typ events.Types, orig ...events.Event) {
	var e events.Event
	if len(orig) > 0 && orig[0] != nil {
		e = orig[0].Clone()
		e.AsBase().Typ = typ
	} else {
		e = &events.Base{Typ: typ}
	}
	v.HandleEvent(e)
}

// HandleEvent sends the given event to all Listeners for that event type.
// It also checks if the State has changed and calls ApplyStyle if so.
// If more significant Config level changes are needed due to an event,
// the event handler must do this itself.
func (v *ValueData) HandleEvent(ev events.Event) {
	if gi.DebugSettings.EventTrace {
		fmt.Println("Event to Value:", v.String(), ev.String())
	}
	v.Listeners.Call(ev)
}

func (v *ValueData) SaveTmp() {
	if v.TmpSave == nil {
		return
	}
	if v.TmpSave.AsValueData() == v {
		// if we are a map value, of a struct value, we save our value
		if v.Owner != nil && v.OwnKind == reflect.Map && !v.Is(ValueMapKey) {
			if laser.NonPtrValue(v.Value).Kind() == reflect.Struct {
				ov := laser.NonPtrValue(reflect.ValueOf(v.Owner))
				if v.KeyView != nil {
					ck := laser.NonPtrValue(v.KeyView.Val())
					laser.SetMapRobust(ov, ck, laser.NonPtrValue(v.Value))
				} else {
					laser.SetMapRobust(ov, laser.NonPtrValue(reflect.ValueOf(v.Key)), laser.NonPtrValue(v.Value))
					// fmt.Printf("save tmp of struct value in key: %v\n", v.Key)
				}
			}
		}
	} else {
		v.TmpSave.SaveTmp()
	}
}

func (v *ValueData) SetTags(tags map[string]string) {
	if v.Tags == nil {
		v.Tags = make(map[string]string, len(tags))
	}
	for tag, val := range tags {
		v.Tags[tag] = val
	}
}

func (v *ValueData) SetTag(tag, value string) {
	if v.Tags == nil {
		v.Tags = make(map[string]string, 10)
	}
	v.Tags[tag] = value
}

func (v *ValueData) Tag(tag string) (string, bool) {
	if v.Tags != nil {
		if tv, ok := v.Tags[tag]; ok {
			return tv, ok
		}
	}
	if !(v.Owner != nil && v.OwnKind == reflect.Struct) {
		return "", false
	}
	return v.Field.Tag.Lookup(tag)
}

func (v *ValueData) AllTags() map[string]string {
	rvt := make(map[string]string)
	if v.Tags != nil {
		for key, val := range v.Tags {
			rvt[key] = val
		}
	}
	if !(v.Owner != nil && v.OwnKind == reflect.Struct) {
		return rvt
	}
	smap := laser.StructTags(v.Field.Tag)
	for key, val := range smap {
		rvt[key] = val
	}
	return rvt
}

// OwnerLabel returns some extra info about the owner of this value view
// which is useful to put in title of our object
func (v *ValueData) OwnerLabel() string {
	if v.Owner == nil {
		return ""
	}
	switch v.OwnKind {
	case reflect.Struct:
		return strcase.ToSentence(v.Field.Name)
	case reflect.Map:
		kystr := ""
		if v.Is(ValueMapKey) {
			kv := laser.NonPtrValue(v.Value)
			kystr = laser.ToString(kv.Interface())
		} else {
			if v.KeyView != nil {
				ck := laser.NonPtrValue(v.KeyView.Val()) // current key value
				kystr = laser.ToString(ck.Interface())
			} else {
				kystr = laser.ToString(v.Key)
			}
		}
		if kystr != "" {
			return kystr
		}
	case reflect.Slice:
		if lblr, ok := v.Owner.(gi.SliceLabeler); ok {
			slbl := lblr.ElemLabel(v.Idx)
			if slbl != "" {
				return slbl
			}
		}
		return strconv.Itoa(v.Idx)
	}
	return ""
}

// GetTitle returns a title for this item suitable for a window title etc,
// based on the underlying value type name, owner label, and ViewPath.
// newPath returns just what should be added to a ViewPath
// also includes zero value check reported in the isZero bool, which
// can be used for not proceeding in case of non-value-based types.
func (v *ValueData) GetTitle() (label, newPath string, isZero bool) {
	var npt reflect.Type
	if v.Value.IsZero() || laser.NonPtrValue(v.Value).IsZero() {
		npt = laser.NonPtrType(v.Value.Type())
		isZero = true
	} else {
		opv := laser.OnePtrUnderlyingValue(v.Value)
		npt = laser.NonPtrType(opv.Type())
	}
	newPath = laser.FriendlyTypeName(npt)
	olbl := v.OwnerLabel()
	if olbl != "" && olbl != newPath {
		label = olbl + " (" + newPath + ")"
	} else {
		label = newPath
	}
	if v.ViewPath != "" {
		label += " (" + v.ViewPath + ")"
	}
	return
}

// ConfigDialogWidget configures the widget for the given value to open the dialog for
// the given value when clicked and have the appropriate tooltip for that.
// If allowReadOnly is false, the dialog will not be opened if the value
// is read only.
func ConfigDialogWidget(v Value, allowReadOnly bool) {
	doc := v.Doc()
	tip := ""
	// windows are never new on mobile
	if !gi.TheApp.Platform().IsMobile() {
		tip += "[Shift: new window]"
		if doc != "" {
			tip += " "
		}
	}
	tip += doc
	v.AsWidgetBase().SetTooltip(tip)
	v.AsWidget().OnClick(func(e events.Event) {
		if allowReadOnly || !v.IsReadOnly() {
			v.SetFlag(e.HasAnyModifier(key.Shift), ValueDialogNewWindow)
			OpenDialog(v, v.AsWidget(), nil)
		}
	})
}
