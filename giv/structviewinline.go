// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// StructViewInline represents a struct as a single line widget, for smaller
// structs and those explicitly marked inline in the kit type registry type
// properties -- constructs widgets in Parts to show the field names and
// editor fields for each field
type StructViewInline struct {
	gi.PartsWidgetBase
	Struct        interface{} `desc:"the struct that we are a view onto"`
	StructValView ValueView   `desc:"ValueView for the struct itself, if this was created within value view framework -- otherwise nil"`
	AddAction     bool        `desc:"if true add an edit action button at the end -- other users of this widget can then configure that -- it is called 'edit-action'"`
	FieldViews    []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave       ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig       ki.Signal   `json:"-" xml:"-" view:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
	ViewPath      string      `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`
	HasDefs       bool        `json:"-" xml:"-" view:"inactive" desc:"if true, some fields have default values -- update labels when values change"`
}

var KiT_StructViewInline = kit.Types.AddType(&StructViewInline{}, StructViewInlineProps)

func (sv *StructViewInline) Disconnect() {
	sv.PartsWidgetBase.Disconnect()
	sv.ViewSig.DisconnectAll()
}

// SetStruct sets the source struct that we are viewing -- rebuilds the
// children to represent this struct
func (sv *StructViewInline) SetStruct(st interface{}) {
	updt := false
	if sv.Struct != st {
		updt = sv.UpdateStart()
		sv.Struct = st
		if k, ok := st.(ki.Ki); ok {
			k.NodeSignal().Connect(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				svv, _ := recv.Embed(KiT_StructViewInline).(*StructViewInline)
				svv.UpdateFields() // this never gets called, per below!
				// fmt.Printf("struct view inline ki update values\n")
				svv.ViewSig.Emit(svv.This(), 0, k)
			})
		}
	}
	sv.ConfigParts()
	sv.UpdateEnd(updt)
}

var StructViewInlineProps = ki.Props{
	"EnumType:Flag": gi.KiT_NodeFlags,
}

// ConfigParts configures Parts for the current struct
func (sv *StructViewInline) ConfigParts() {
	if kit.IfaceIsNil(sv.Struct) {
		return
	}
	sv.Parts.Lay = gi.LayoutHoriz
	config := kit.TypeAndNameList{}
	// always start fresh!
	sv.FieldViews = make([]ValueView, 0)
	kit.FlatFieldsValueFunc(sv.Struct, func(fval interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		// todo: check tags, skip various etc
		vwtag := field.Tag.Get("view")
		if vwtag == "-" {
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
	if sv.AddAction {
		config.Add(gi.KiT_Action, "edit-action")
	}
	mods, updt := sv.Parts.ConfigChildren(config)
	if !mods {
		updt = sv.Parts.UpdateStart()
	}
	sv.HasDefs = false
	for i, vv := range sv.FieldViews {
		lbl := sv.Parts.Child(i * 2).(*gi.Label)
		vvb := vv.AsValueViewBase()
		vvb.ViewPath = sv.ViewPath
		lbl.Redrawable = true
		lbl.SetProp("horizontal-align", gist.AlignLeft)
		widg := sv.Parts.Child((i * 2) + 1).(gi.Node2D)
		hasDef, inactTag := StructViewFieldTags(vv, lbl, widg, sv.IsInactive()) // in structview.go
		if hasDef {
			sv.HasDefs = true
		}
		vv.ConfigWidget(widg)
		if !sv.IsInactive() && !inactTag {
			vvb.ViewSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				svv, _ := recv.Embed(KiT_StructViewInline).(*StructViewInline)
				svv.UpdateDefaults()
				// note: updating here is redundant
				svv.ViewSig.Emit(svv.This(), 0, nil)
			})
		}
	}
	sv.Parts.UpdateEnd(updt)
}

func (sv *StructViewInline) UpdateFields() {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
}

func (sv *StructViewInline) UpdateDefaults() {
	if !sv.HasDefs {
		return
	}
	updt := sv.UpdateStart()
	sv.SetFullReRender() // key to regen
	for i, vv := range sv.FieldViews {
		lbl := sv.Parts.Child(i * 2).(*gi.Label)
		StructViewFieldDefTag(vv, lbl)
	}
	sv.UpdateEnd(updt)
}

func (sv *StructViewInline) Render2D() {
	if sv.FullReRenderIfNeeded() {
		return
	}
	if sv.PushBounds() {
		sv.Render2DParts()
		sv.Render2DChildren()
		sv.PopBounds()
	}
}
