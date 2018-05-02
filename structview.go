// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"reflect"

	"github.com/goki/goki/gi/units"
	"github.com/goki/goki/ki"
	"github.com/goki/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  StructView

// todo: sub-editor panels with shared menubutton panel at bottom.. not clear that that is necc -- probably better to have each sub-panel fully separate

// StructView represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame with an optional title, and a button
// box at the bottom where methods can be invoked
type StructView struct {
	Frame
	Struct     interface{} `desc:"the struct that we are a view onto"`
	Title      string      `desc:"title / prompt to show above the editor fields"`
	FieldViews []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave    ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig    ki.Signal   `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
}

var KiT_StructView = kit.Types.AddType(&StructView{}, StructViewProps)

func (n *StructView) New() ki.Ki { return &StructView{} }

var StructViewProps = ki.Props{
	"background-color": &Prefs.BackgroundColor,
	"#title": ki.Props{
		"max-width":      units.NewValue(-1, units.Px),
		"text-align":     AlignCenter,
		"vertical-align": AlignTop,
	},
}

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetStruct sets the source struct that we are viewing -- rebuilds the children to represent this struct
func (sv *StructView) SetStruct(st interface{}, tmpSave ValueView) {
	updt := false
	if sv.Struct != st {
		updt = sv.UpdateStart()
		sv.Struct = st
		if k, ok := st.(ki.Ki); ok {
			k.NodeSignal().Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				svv, _ := recv.EmbeddedStruct(KiT_StructView).(*StructView)
				svv.UpdateFields()
				svv.ViewSig.Emit(svv.This, 0, nil)
			})
		}
	}
	sv.TmpSave = tmpSave
	sv.UpdateFromStruct()
	sv.UpdateEnd(updt)
}

// SetFrame configures view as a frame
func (sv *StructView) SetFrame() {
	sv.Lay = LayoutCol
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (sv *StructView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	config.Add(KiT_Label, "title")
	config.Add(KiT_Space, "title-space")
	config.Add(KiT_Layout, "struct-grid")
	config.Add(KiT_Space, "grid-space")
	config.Add(KiT_Layout, "buttons")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (sv *StructView) StdConfig() (mods, updt bool) {
	sv.SetFrame()
	config := sv.StdFrameConfig()
	mods, updt = sv.ConfigChildren(config, false)
	return
}

// SetTitle sets the title and updates the Title label
func (sv *StructView) SetTitle(title string) {
	sv.Title = title
	lab, _ := sv.TitleWidget()
	if lab != nil {
		lab.Text = title
	}
}

// Title returns the title label widget, and its index, within frame -- nil,
// -1 if not found
func (sv *StructView) TitleWidget() (*Label, int) {
	idx := sv.ChildIndexByName("title", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Label), idx
}

// StructGrid returns the grid layout widget, which contains all the fields
// and values, and its index, within frame -- nil, -1 if not found
func (sv *StructView) StructGrid() (*Layout, int) {
	idx := sv.ChildIndexByName("struct-grid", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame
// -- nil, -1 if not found
func (sv *StructView) ButtonBox() (*Layout, int) {
	idx := sv.ChildIndexByName("buttons", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ConfigStructGrid configures the StructGrid for the current struct
func (sv *StructView) ConfigStructGrid() {
	if kit.IfaceIsNil(sv.Struct) {
		return
	}
	sg, _ := sv.StructGrid()
	if sg == nil {
		return
	}
	sg.Lay = LayoutGrid
	sg.SetProp("columns", 2)
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// always start fresh!
	sv.FieldViews = make([]ValueView, 0)
	kit.FlatFieldsValueFun(sv.Struct, func(fval interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
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
		vv.SetStructValue(vvp, sv.Struct, &field, sv.TmpSave)
		vtyp := vv.WidgetType()
		// todo: other things with view tag..
		labnm := fmt.Sprintf("label-%v", field.Name)
		valnm := fmt.Sprintf("value-%v", field.Name)
		config.Add(KiT_Label, labnm)
		config.Add(vtyp, valnm) // todo: extend to diff types using interface..
		sv.FieldViews = append(sv.FieldViews, vv)
		return true
	})
	mods, updt := sg.ConfigChildren(config, false)
	if mods {
		sv.SetFullReRender()
	} else {
		updt = sg.UpdateStart()
	}
	for i, vv := range sv.FieldViews {
		lbl := sg.Child(i * 2).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		vvb := vv.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.EmbeddedStruct(KiT_StructView).(*StructView)
			// note: updating here is redundant -- relevant field will have already updated
			svv.ViewSig.Emit(svv.This, 0, nil)
		})
		lbltag := vvb.Field.Tag.Get("label")
		if lbltag != "" {
			lbl.Text = lbltag
		} else {
			lbl.Text = vvb.Field.Name
		}
		widg := sg.Child((i * 2) + 1).(Node2D)
		widg.SetProp("vertical-align", AlignMiddle)
		vv.ConfigWidget(widg)
	}
	sg.UpdateEnd(updt)
}

func (sv *StructView) UpdateFromStruct() {
	mods, updt := sv.StdConfig()
	typ := kit.NonPtrType(reflect.TypeOf(sv.Struct))
	sv.SetTitle(fmt.Sprintf("%v Fields", typ.Name()))
	sv.ConfigStructGrid()
	if mods {
		sv.UpdateEnd(updt)
	}
}

func (sv *StructView) UpdateFields() {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
}

func (sv *StructView) Style2D() {
	sv.Frame.Style2D()
	sv.UpdateFromStruct()
}

func (sv *StructView) Render2D() {
	sv.ClearFullReRender()
	sv.Frame.Render2D()
}

func (sv *StructView) ReRender2D() (node Node2D, layout bool) {
	if sv.NeedsFullReRender() {
		node = nil
		layout = false
	} else {
		node = sv.This.(Node2D)
		layout = true
	}
	return
}

// check for interface implementation
var _ Node2D = &StructView{}

////////////////////////////////////////////////////////////////////////////////////////
//  StructViewInline

// StructViewInline represents a struct as a single line widget, for smaller
// structs and those explicitly marked inline in the kit type registry type
// properties -- constructs widgets in Parts to show the field names and
// editor fields for each field
type StructViewInline struct {
	WidgetBase
	Struct     interface{} `desc:"the struct that we are a view onto"`
	AddAction  bool        `desc:"if true add an ... action button at the end -- other users of this widget can then configure that -- it is called 'extra-action'"`
	FieldViews []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`
	TmpSave    ValueView   `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
	ViewSig    ki.Signal   `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`
}

var KiT_StructViewInline = kit.Types.AddType(&StructViewInline{}, StructViewInlineProps)

func (n *StructViewInline) New() ki.Ki { return &StructViewInline{} }

// SetStruct sets the source struct that we are viewing -- rebuilds the children to represent this struct
func (sv *StructViewInline) SetStruct(st interface{}, tmpSave ValueView) {
	updt := false
	if sv.Struct != st {
		updt = sv.UpdateStart()
		sv.Struct = st
		if k, ok := st.(ki.Ki); ok {
			k.NodeSignal().Connect(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				svv, _ := recv.EmbeddedStruct(KiT_StructViewInline).(*StructViewInline)
				svv.UpdateFields()
				fmt.Printf("struct view inline ki update values\n")
				svv.ViewSig.Emit(svv.This, 0, k)
			})
		}
	}
	sv.TmpSave = tmpSave
	sv.UpdateFromStruct()
	sv.UpdateEnd(updt)
}

var StructViewInlineProps = ki.Props{
	"min-width": units.NewValue(20, units.Ex),
}

// ConfigParts configures Parts for the current struct
func (sv *StructViewInline) ConfigParts() {
	if kit.IfaceIsNil(sv.Struct) {
		return
	}
	sv.Parts.Lay = LayoutRow
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// always start fresh!
	sv.FieldViews = make([]ValueView, 0)
	kit.FlatFieldsValueFun(sv.Struct, func(fval interface{}, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
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
		vv.SetStructValue(vvp, sv.Struct, &field, sv.TmpSave)
		vtyp := vv.WidgetType()
		// todo: other things with view tag..
		labnm := fmt.Sprintf("label-%v", field.Name)
		valnm := fmt.Sprintf("value-%v", field.Name)
		config.Add(KiT_Label, labnm)
		config.Add(vtyp, valnm) // todo: extend to diff types using interface..
		sv.FieldViews = append(sv.FieldViews, vv)
		return true
	})
	if sv.AddAction {
		config.Add(KiT_Action, "edit-action")
	}
	mods, updt := sv.Parts.ConfigChildren(config, false)
	if !mods {
		updt = sv.Parts.UpdateStart()
	}
	for i, vv := range sv.FieldViews {
		lbl := sv.Parts.Child(i * 2).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		vvb := vv.AsValueViewBase()
		vvb.ViewSig.ConnectOnly(sv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			svv, _ := recv.EmbeddedStruct(KiT_StructViewInline).(*StructViewInline)
			// note: updating here is redundant
			svv.ViewSig.Emit(svv.This, 0, nil)
		})
		lbltag := vvb.Field.Tag.Get("label")
		if lbltag != "" {
			lbl.Text = lbltag
		} else {
			lbl.Text = vvb.Field.Name
		}
		widg := sv.Parts.Child((i * 2) + 1).(Node2D)
		widg.SetProp("vertical-align", AlignMiddle)
		vv.ConfigWidget(widg)
	}
	sv.Parts.UpdateEnd(updt)
}

func (sv *StructViewInline) UpdateFromStruct() {
	sv.ConfigParts()
}

func (sv *StructViewInline) UpdateFields() {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
}

func (sv *StructViewInline) Render2D() {
	if sv.PushBounds() {
		sv.Render2DParts()
		sv.Render2DChildren()
		sv.PopBounds()
	}
}

func (sv *StructViewInline) ReRender2D() (node Node2D, layout bool) {
	node = sv.This.(Node2D)
	layout = true
	return
}

// check for interface implementation
var _ Node2D = &StructViewInline{}
