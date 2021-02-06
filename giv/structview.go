// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// StructView represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame.
// Automatically has a toolbar with Struct ToolBar props if defined
// set prop toolbar = false to turn off
type StructView struct {
	gi.Frame
	Struct        interface{}       `desc:"the struct that we are a view onto"`
	StructValView ValueView         `desc:"ValueView for the struct itself, if this was created within value view framework -- otherwise nil"`
	Changed       bool              `desc:"has the value of any field changed?  updated by the ViewSig signals from fields"`
	ChangeFlag    *reflect.Value    `json:"-" xml:"-" desc:"ValueView for a field marked with changeflag struct tag, which must be a bool type, which is updated when changes are registered in field values."`
	FieldViews    []ValueView       `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave       ValueView         `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig       ki.Signal         `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	ViewPath      string            `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`
	ToolbarStru   interface{}       `desc:"the struct that we successfully set a toolbar for"`
	HasDefs       bool              `json:"-" xml:"-" view:"inactive" desc:"if true, some fields have default values -- update labels when values change"`
	TypeFieldTags map[string]string `json:"-" xml:"-" view:"inactive" desc:"extra tags by field name -- from type properties"`
}

var KiT_StructView = kit.Types.AddType(&StructView{}, StructViewProps)

// AddNewStructView adds a new structview to given parent node, with given name.
func AddNewStructView(parent ki.Ki, name string) *StructView {
	return parent.AddNewChild(KiT_StructView, name).(*StructView)
}

func (sv *StructView) Disconnect() {
	sv.Frame.Disconnect()
	sv.ViewSig.DisconnectAll()
}

var StructViewProps = ki.Props{
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
}

// SetStruct sets the source struct that we are viewing -- rebuilds the
// children to represent this struct
func (sv *StructView) SetStruct(st interface{}) {
	updt := false
	if sv.Struct != st {
		sv.Changed = false
		updt = sv.UpdateStart()
		sv.SetFullReRender()
		if sv.Struct != nil {
			if k, ok := sv.Struct.(ki.Ki); ok {
				k.NodeSignal().Disconnect(sv.This())
			}
		}
		sv.Struct = st
		tp := kit.Types.Properties(kit.NonPtrType(reflect.TypeOf(sv.Struct)), false)
		if tp != nil {
			if sfp, has := ki.SubTypeProps(*tp, "StructViewFields"); has {
				sv.TypeFieldTags = make(map[string]string)
				for k, v := range sfp {
					vs := kit.ToString(v)
					sv.TypeFieldTags[k] = vs
				}
			}
		}
		if k, ok := st.(ki.Ki); ok {
			k.NodeSignal().Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				// todo: check for delete??
				svv, _ := recv.Embed(KiT_StructView).(*StructView)
				svv.UpdateFields()
				svv.ViewSig.Emit(svv.This(), 0, nil)
			})
		}
	}
	sv.Config()
	sv.UpdateEnd(updt)
}

// UpdateFields updates each of the value-view widgets for the fields --
// called by the ViewSig update
func (sv *StructView) UpdateFields() {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
}

// UpdateField updates the value-view widget for the named field
func (sv *StructView) UpdateField(field string) {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
		if vv.Name() == field {
			vv.UpdateWidget()
			break
		}
	}
	sv.UpdateEnd(updt)
}

// Config configures the view
func (sv *StructView) Config() {
	if ks, ok := sv.Struct.(ki.Ki); ok {
		if ks.IsDeleted() || ks.IsDestroyed() {
			return
		}
	}
	sv.Lay = gi.LayoutVert
	sv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_Frame, "struct-grid")
	mods, updt := sv.ConfigChildren(config)
	sv.ConfigStructGrid()
	sv.ConfigToolbar()
	if mods {
		sv.UpdateEnd(updt)
	}
}

// IsConfiged returns true if the widget is fully configured
func (sv *StructView) IsConfiged() bool {
	if len(sv.Kids) == 0 {
		return false
	}
	return true
}

// StructGrid returns the grid layout widget, which contains all the fields and values
func (sv *StructView) StructGrid() *gi.Frame {
	return sv.ChildByName("struct-grid", 2).(*gi.Frame)
}

// ToolBar returns the toolbar widget
func (sv *StructView) ToolBar() *gi.ToolBar {
	return sv.ChildByName("toolbar", 1).(*gi.ToolBar)
}

// ConfigToolbar adds a toolbar based on the methview ToolBarView function, if
// one has been defined for this struct type through its registered type
// properties.
func (sv *StructView) ConfigToolbar() {
	if kit.IfaceIsNil(sv.Struct) {
		return
	}
	if sv.ToolbarStru == sv.Struct {
		return
	}
	if pv, ok := sv.PropInherit("toolbar", ki.NoInherit, ki.TypeProps); ok {
		pvb, _ := kit.ToBool(pv)
		if !pvb {
			sv.ToolbarStru = sv.Struct
			return
		}
	}
	tb := sv.ToolBar()
	tb.SetStretchMaxWidth()
	svtp := kit.NonPtrType(reflect.TypeOf(sv.Struct))
	ttip := "update this StructView (not any other views that might be present) to show current state of this struct of type: " + svtp.String()
	if len(*tb.Children()) == 0 {
		tb.AddAction(gi.ActOpts{Label: "UpdtView", Icon: "update", Tooltip: ttip},
			sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				svv := recv.Embed(KiT_StructView).(*StructView)
				svv.UpdateFields()
			})
	} else {
		act := tb.Child(0).(*gi.Action)
		act.Tooltip = ttip
	}
	ndef := 1 // number of default actions
	sz := len(*tb.Children())
	if sz > ndef {
		for i := sz - 1; i >= ndef; i-- {
			tb.DeleteChildAtIndex(i, ki.DestroyKids)
		}
	}
	if HasToolBarView(sv.Struct) {
		ToolBarView(sv.Struct, sv.Viewport, tb)
		tb.SetFullReRender()
	}
	sv.ToolbarStru = sv.Struct
}

// FieldTags returns the integrated tags for this field
func (sv *StructView) FieldTags(fld reflect.StructField) reflect.StructTag {
	if sv.TypeFieldTags == nil {
		return fld.Tag
	}
	ft, has := sv.TypeFieldTags[fld.Name]
	if !has {
		return fld.Tag
	}
	return fld.Tag + " " + reflect.StructTag(ft)
}

// ConfigStructGrid configures the StructGrid for the current struct
func (sv *StructView) ConfigStructGrid() {
	if kit.IfaceIsNil(sv.Struct) {
		return
	}
	sg := sv.StructGrid()
	sg.Lay = gi.LayoutGrid
	sg.Stripes = gi.RowStripes
	// setting a pref here is key for giving it a scrollbar in larger context
	sg.SetMinPrefHeight(units.NewEm(1.5))
	sg.SetMinPrefWidth(units.NewEm(10))
	sg.SetStretchMax()                          // for this to work, ALL layers above need it too
	sg.SetProp("overflow", gist.OverflowScroll) // this still gives it true size during PrefSize
	sg.SetProp("columns", 2)
	config := kit.TypeAndNameList{}
	// always start fresh!
	sv.FieldViews = make([]ValueView, 0)
	kit.FlatFieldsValueFunc(sv.Struct, func(fval interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		// todo: check tags, skip various etc
		ftags := sv.FieldTags(field)
		_, got := ftags.Lookup("changeflag")
		if got {
			if field.Type.Kind() == reflect.Bool {
				sv.ChangeFlag = &fieldVal
			}
		}
		vwtag := ftags.Get("view")
		if vwtag == "-" {
			return true
		}
		if vwtag == "add-fields" && field.Type.Kind() == reflect.Struct {
			fvalp := fieldVal.Addr().Interface()
			kit.FlatFieldsValueFunc(fvalp, func(sfval interface{}, styp reflect.Type, sfield reflect.StructField, sfieldVal reflect.Value) bool {
				svwtag := sfield.Tag.Get("view")
				if svwtag == "-" {
					return true
				}
				svv := FieldToValueView(fvalp, sfield.Name, sfval)
				if svv == nil { // shouldn't happen
					return true
				}
				svvp := sfieldVal.Addr()
				svv.SetStructValue(svvp, fvalp, &sfield, sv.TmpSave, sv.ViewPath)

				svtyp := svv.WidgetType()
				// todo: other things with view tag..
				fnm := field.Name + "." + sfield.Name
				svv.SetTag("label", fnm)
				labnm := fmt.Sprintf("label-%v", fnm)
				valnm := fmt.Sprintf("value-%v", fnm)
				config.Add(gi.KiT_Label, labnm)
				config.Add(svtyp, valnm) // todo: extend to diff types using interface..
				sv.FieldViews = append(sv.FieldViews, svv)
				return true
			})
			return true
		}
		vv := FieldToValueView(sv.Struct, field.Name, fval)
		if vv == nil { // shouldn't happen
			return true
		}
		vvp := fieldVal.Addr()
		vv.SetStructValue(vvp, sv.Struct, &field, sv.TmpSave, sv.ViewPath)
		vtyp := vv.WidgetType()
		// todo: other things with view tag..
		labnm := fmt.Sprintf("label-%v", field.Name)
		valnm := fmt.Sprintf("value-%v", field.Name)
		config.Add(gi.KiT_Label, labnm)
		config.Add(vtyp, valnm) // todo: extend to diff types using interface..
		sv.FieldViews = append(sv.FieldViews, vv)
		return true
	})
	mods, updt := sg.ConfigChildren(config) // fields could be non-unique with labels..
	if mods {
		sg.SetFullReRender()
	} else {
		updt = sg.UpdateStart()
	}
	sv.HasDefs = false
	for i, vv := range sv.FieldViews {
		lbl := sg.Child(i * 2).(*gi.Label)
		vvb := vv.AsValueViewBase()
		vvb.ViewPath = sv.ViewPath
		lbl.Redrawable = true
		widg := sg.Child((i * 2) + 1).(gi.Node2D)
		widg.SetProp("horizontal-align", gist.AlignLeft)
		hasDef, inactTag := StructViewFieldTags(vv, lbl, widg, sv.IsInactive())
		if hasDef {
			sv.HasDefs = true
		}
		vv.ConfigWidget(widg)
		if !sv.IsInactive() && !inactTag {
			vvb.ViewSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				svv := recv.Embed(KiT_StructView).(*StructView)
				svv.UpdateDefaults()
				// note: updating vv here is redundant -- relevant field will have already updated
				svv.Changed = true
				if svv.ChangeFlag != nil {
					svv.ChangeFlag.SetBool(true)
				}
				vvv := send.(ValueView).AsValueViewBase()
				if !kit.KindIsBasic(kit.NonPtrValue(vvv.Value).Kind()) {
					if updtr, ok := svv.Struct.(gi.Updater); ok {
						// fmt.Printf("updating: %v kind: %v\n", updtr, vvv.Value.Kind())
						updtr.Update()
					}
				}
				tb := svv.ToolBar()
				if tb != nil {
					tb.UpdateActions()
				}
				svv.ViewSig.Emit(svv.This(), 0, nil)
				// vvv, _ := send.Embed(KiT_ValueViewBase).(*ValueViewBase)
				// fmt.Printf("sview got edit from vv %v field: %v\n", vvv.Nm, vvv.Field.Name)
			})
		}
	}
	sg.UpdateEnd(updt)
}

func (sv *StructView) Style2D() {
	mvp := sv.ViewportSafe()
	if mvp != nil && mvp.IsDoingFullRender() {
		sv.Config()
	}
	sv.Frame.Style2D()
}

func (sv *StructView) UpdateDefaults() {
	if !sv.HasDefs || !sv.IsConfiged() {
		return
	}
	sg := sv.StructGrid()
	updt := sg.UpdateStart()
	for i, vv := range sv.FieldViews {
		lbl := sg.Child(i * 2).(*gi.Label)
		StructViewFieldDefTag(vv, lbl)
	}
	sg.UpdateEnd(updt)
}

func (sv *StructView) Render2D() {
	if sv.IsConfiged() {
		sv.ToolBar().UpdateActions()
	}
	if win := sv.ParentWindow(); win != nil {
		if !win.IsResizing() {
			win.MainMenuUpdateActives()
		}
	}
	sv.Frame.Render2D()
}

/////////////////////////////////////////////////////////////////////////
//  Tag parsing

// StructViewFieldTags processes the tags for a field in a struct view, setting
// the properties on the label or widget appropriately
// returns true if there were any "def" default tags -- if so, needs updating
func StructViewFieldTags(vv ValueView, lbl *gi.Label, widg gi.Node2D, isInact bool) (hasDef, inactTag bool) {
	vvb := vv.AsValueViewBase()
	if lbltag, has := vv.Tag("label"); has {
		lbl.Text = lbltag
	} else {
		lbl.Text = vvb.Field.Name
	}
	if _, has := vv.Tag("inactive"); has {
		inactTag = true
		widg.AsNode2D().SetInactive()
	} else {
		if isInact {
			widg.AsNode2D().SetInactive()
			vv.SetTag("inactive", "true")
		}
	}
	defStr := ""
	hasDef, _, defStr = StructViewFieldDefTag(vv, lbl)
	if ttip, has := vv.Tag("desc"); has {
		lbl.Tooltip = defStr + ttip
	}
	return
}

// StructViewFieldDefTag processes the "def" tag for default values -- can be
// called multiple times for updating as values change.
// returns true if value is default, and string to add to tooltip for default vals
func StructViewFieldDefTag(vv ValueView, lbl *gi.Label) (hasDef bool, isDef bool, defStr string) {
	if dtag, has := vv.Tag("def"); has {
		hasDef = true
		isDef, defStr = StructFieldIsDef(dtag, vv.Val().Interface())
		if isDef {
			lbl.CurBgColor = gi.Prefs.Colors.Background
		} else {
			lbl.CurBgColor = gi.Prefs.Colors.Highlight
		}
		return
	}
	return
}

// StructFieldIsDef processses "def" tag for default value(s) of field
// defs = default values as strings as either comma-separated list of valid values
// or low:high value range (only for int or float numeric types)
// valPtr = pointer to value
// returns true if value is default, and string to add to tooltip for default values
func StructFieldIsDef(defs string, valPtr interface{}) (bool, string) {
	defStr := "[Def: " + defs + "] "
	def := false
	if strings.Contains(defs, ":") {
		dtags := strings.Split(defs, ":")
		lo, _ := strconv.ParseFloat(dtags[0], 64)
		hi, _ := strconv.ParseFloat(dtags[1], 64)
		switch fv := valPtr.(type) {
		case *float32:
			if lo <= float64(*fv) && float64(*fv) <= hi {
				def = true
			}
		case *float64:
			if lo <= *fv && *fv <= hi {
				def = true
			}
		case *int32:
			if lo <= float64(*fv) && float64(*fv) <= hi {
				def = true
			}
		case *int64:
			if lo <= float64(*fv) && float64(*fv) <= hi {
				def = true
			}
		case *int:
			if lo <= float64(*fv) && float64(*fv) <= hi {
				def = true
			}
		}
	} else {
		val := kit.ToStringPrec(valPtr, 6)
		dtags := strings.Split(defs, ",")
		for _, dv := range dtags {
			if dv == strings.TrimSpace(val) {
				def = true
				break
			}
		}
	}
	return def, defStr
}

// StructFieldVals represents field values in a struct, at multiple
// levels of depth potentially (represented by the Path field)
// used for StructNonDefFields for example.
type StructFieldVals struct {
	Path  string              `desc:"path of field.field parent fields to this field"`
	Field reflect.StructField `desc:"type information for field"`
	Val   reflect.Value       `desc:"value of field (as a pointer)"`
	Defs  string              `desc:"def tag information for default values"`
}

// StructNonDefFields processses "def" tag for default value(s)
// of fields in given struct and starting path, and returns all
// fields not at their default values.
// See also StructNoDefFieldsStr for a string representation of this information.
// Uses kit.FlatFieldsValueFunc to get all embedded fields.
// Uses a recursive strategy -- any fields that are themselves structs are
// expanded, and the field name represented by dots path separators.
func StructNonDefFields(structPtr interface{}, path string) []StructFieldVals {
	var flds []StructFieldVals
	kit.FlatFieldsValueFunc(structPtr, func(fval interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		vvp := fieldVal.Addr()
		if field.Type.Kind() == reflect.Struct {
			spath := path
			if path != "" {
				spath += "."
			}
			spath += field.Name
			subs := StructNonDefFields(vvp.Interface(), spath)
			if len(subs) > 0 {
				flds = append(flds, subs...)
			}
			return true
		}
		dtag, got := field.Tag.Lookup("def")
		if !got {
			return true
		}
		def, defStr := StructFieldIsDef(dtag, vvp.Interface())
		if def {
			return true
		}
		flds = append(flds, StructFieldVals{Path: path, Field: field, Val: vvp, Defs: defStr})
		return true
	})
	return flds
}

// StructNonDefFieldsStr processses "def" tag for default value(s) of fields in
// given struct, and returns a string of all those not at their default values,
// in format: Path.Field: val // default value(s)
// Uses a recursive strategy -- any fields that are themselves structs are
// expanded, and the field name represented by dots path separators.
func StructNonDefFieldsStr(structPtr interface{}, path string) string {
	flds := StructNonDefFields(structPtr, path)
	if len(flds) == 0 {
		return ""
	}
	str := ""
	for _, fld := range flds {
		pth := fld.Path
		fnm := fld.Field.Name
		val := kit.ToStringPrec(fld.Val.Interface(), 6)
		dfs := fld.Defs
		if len(pth) > 0 {
			fnm = pth + "." + fnm
		}
		str += fmt.Sprintf("%s: %s // %s<br>\n", fnm, val, dfs)
	}
	return str
}
