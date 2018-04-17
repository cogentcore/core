// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"reflect"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  StructView

// todo: sub-editor panels with shared menubutton panel at bottom.. not clear that that is necc -- probably better to have each sub-panel fully separate

// StructView represents a struct, creating a property editor of the fields -- constructs Children widgets to show the field names and editor fields for each field, within an overall frame with an optional title, and a button box at the bottom where methods can be invoked
type StructView struct {
	Frame
	Struct  interface{} `desc:"the struct that we are a view onto"`
	Title   string      `desc:"title / prompt to show above the editor fields"`
	Fields  []ValueView `desc:"ValueView representations of the fields"`
	TmpSave ValueView   `desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
}

var KiT_StructView = kit.Types.AddType(&StructView{}, StructViewProps)

var StructViewProps = ki.Props{
	"#frame": ki.Props{
		"border-width":        units.NewValue(2, units.Px),
		"margin":              units.NewValue(8, units.Px),
		"padding":             units.NewValue(4, units.Px),
		"box-shadow.h-offset": units.NewValue(4, units.Px),
		"box-shadow.v-offset": units.NewValue(4, units.Px),
		"box-shadow.blur":     units.NewValue(4, units.Px),
		"box-shadow.color":    "#CCC",
	},
	"#title": ki.Props{
		// todo: add "bigger" font
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignCenter,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
	"#prompt": ki.Props{
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignLeft,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
}

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetStruct sets the source struct that we are viewing -- rebuilds the children to represent this struct
func (sv *StructView) SetStruct(st interface{}, tmpSave ValueView) {
	sv.UpdateStart()
	sv.Struct = st
	sv.TmpSave = tmpSave
	sv.UpdateFromStruct()
	sv.UpdateEnd()
}

// SetFrame configures view as a frame
func (sv *StructView) SetFrame() {
	sv.Lay = LayoutCol
	sv.PartStyleProps(sv, StructViewProps)
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

// StdConfig configures a standard setup of the overall Frame
func (sv *StructView) StdConfig() {
	sv.SetFrame()
	config := sv.StdFrameConfig()
	sv.ConfigChildren(config, false)
}

// SetTitle sets the title and updates the Title label
func (sv *StructView) SetTitle(title string) {
	sv.Title = title
	lab, _ := sv.TitleWidget()
	if lab != nil {
		lab.Text = title
	}
}

// Title returns the title label widget, and its index, within frame -- nil, -1 if not found
func (sv *StructView) TitleWidget() (*Label, int) {
	idx := sv.ChildIndexByName("title", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Label), idx
}

// StructGrid returns the StructGrid grid layout widget, which contains all the fields and values, and its index, within frame -- nil, -1 if not found
func (sv *StructView) StructGrid() (*Layout, int) {
	idx := sv.ChildIndexByName("struct-grid", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (sv *StructView) ButtonBox() (*Layout, int) {
	idx := sv.ChildIndexByName("buttons", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ConfigStructGrid configures the StructGrid for the current struct
func (sv *StructView) ConfigStructGrid() {
	if kit.IsNil(sv.Struct) {
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
	sv.Fields = make([]ValueView, 0)
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
		sv.Fields = append(sv.Fields, vv)
		return true
	})
	updt := sg.ConfigChildren(config, false)
	if updt {
		sv.SetFullReRender()
	}
	for i, vv := range sv.Fields {
		lbl := sg.Child(i * 2).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		vvb := vv.AsValueViewBase()
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
}

func (sv *StructView) UpdateFromStruct() {
	sv.StdConfig()
	typ := kit.NonPtrType(reflect.TypeOf(sv.Struct))
	sv.SetTitle(fmt.Sprintf("%v Fields", typ.Name()))
	sv.ConfigStructGrid()
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

// StructViewInline represents a struct as a single line widget, for smaller structs and those explicitly marked inline in the kit type registry type properties -- constructs widgets in Parts to show the field names and editor fields for each field
type StructViewInline struct {
	WidgetBase
	Struct        interface{} `desc:"the struct that we are a view onto"`
	StructViewSig ki.Signal   `json:"-" desc:"signal for StructView -- see StructViewSignals for the types"`
	Fields        []ValueView `desc:"ValueView representations of the fields"`
	TmpSave       ValueView   `desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`
}

var KiT_StructViewInline = kit.Types.AddType(&StructViewInline{}, nil)

// SetStruct sets the source struct that we are viewing -- rebuilds the children to represent this struct
func (sv *StructViewInline) SetStruct(st interface{}, tmpSave ValueView) {
	sv.UpdateStart()
	sv.Struct = st
	sv.TmpSave = tmpSave
	sv.UpdateFromStruct()
	sv.UpdateEnd()
}

var StructViewInlineProps = ki.Props{}

// ConfigParts configures Parts for the current struct
func (sv *StructViewInline) ConfigParts() {
	if kit.IsNil(sv.Struct) {
		return
	}
	sv.UpdateStart()
	sv.Parts.Lay = LayoutRow
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	// always start fresh!
	sv.Fields = make([]ValueView, 0)
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
		sv.Fields = append(sv.Fields, vv)
		return true
	})
	sv.Parts.ConfigChildren(config, false)
	for i, vv := range sv.Fields {
		lbl := sv.Parts.Child(i * 2).(*Label)
		lbl.SetProp("vertical-align", AlignMiddle)
		vvb := vv.AsValueViewBase()
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
	sv.UpdateEnd()
}

func (sv *StructViewInline) UpdateFromStruct() {
	sv.ConfigParts()
}

func (sv *StructViewInline) Render2D() {
	if sv.PushBounds() {
		sv.Render2DParts()
		sv.Render2DChildren()
		sv.PopBounds()
	}
}

// todo: see notes on treeview
func (sv *StructViewInline) ReRender2D() (node Node2D, layout bool) {
	node = sv.This.(Node2D)
	layout = true
	return
}

// check for interface implementation
var _ Node2D = &StructViewInline{}
