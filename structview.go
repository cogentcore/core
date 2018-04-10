// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"reflect"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//  StructView -- a widget that graphically represents / manipulates a struct
//  as a property editor

// signals that structview can send
type StructViewSignals int64

const (
	// struct field was edited -- data is the field name
	StructFieldEdited StructViewSignals = iota
	StructViewSignalsN
)

//go:generate stringer -type=TreeViewSignals

// todo: sub-editor panels with shared menubutton panel at bottom.. not clear that that is necc -- probably better to have each sub-panel fully separate

// StructView represents a struct, creating a property editor of the fields -- constructs Children widgets to show the field names and editor fields for each field, within an overall frame with an optional title, and a button box at the bottom where methods can be invoked
type StructView struct {
	Frame
	Struct        interface{} `desc:"the struct that we are a view onto"`
	StructViewSig ki.Signal   `json:"-" desc:"signal for StructView -- see StructViewSignals for the types"`
	Title         string      `desc:"title / prompt to show above the editor fields"`
	Fields        []ValueView `desc:"ValueView representations of the fields"`
}

var KiT_StructView = kit.Types.AddType(&StructView{}, nil)

// Note: the overall strategy here is similar to Dialog, where we provide lots
// of flexible configuration elements that can be easily extended and modified

// SetStruct sets the source struct that we are viewing -- rebuilds the children to represent this struct
func (sv *StructView) SetStruct(st interface{}) {
	sv.UpdateStart()
	sv.Struct = st
	sv.UpdateFromStruct()
	sv.UpdateEnd()
}

var StructViewProps = map[string]interface{}{
	"#frame": map[string]interface{}{
		"border-width":        units.NewValue(2, units.Px),
		"margin":              units.NewValue(8, units.Px),
		"padding":             units.NewValue(4, units.Px),
		"box-shadow.h-offset": units.NewValue(4, units.Px),
		"box-shadow.v-offset": units.NewValue(4, units.Px),
		"box-shadow.blur":     units.NewValue(4, units.Px),
		"box-shadow.color":    "#CCC",
	},
	"#title": map[string]interface{}{
		// todo: add "bigger" font
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignCenter,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
	"#prompt": map[string]interface{}{
		"max-width":        units.NewValue(-1, units.Px),
		"text-align":       AlignLeft,
		"vertical-align":   AlignTop,
		"background-color": "none",
	},
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
	config.Add(KiT_Label, "Title")
	config.Add(KiT_Space, "TitleSpace")
	config.Add(KiT_Layout, "StructGrid")
	config.Add(KiT_Space, "GridSpace")
	config.Add(KiT_Layout, "ButtonBox")
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
	idx := sv.ChildIndexByName("Title", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Label), idx
}

// StructGrid returns the StructGrid grid layout widget, which contains all the fields and values, and its index, within frame -- nil, -1 if not found
func (sv *StructView) StructGrid() (*Layout, int) {
	idx := sv.ChildIndexByName("StructGrid", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (sv *StructView) ButtonBox() (*Layout, int) {
	idx := sv.ChildIndexByName("ButtonBox", 0)
	if idx < 0 {
		return nil, -1
	}
	return sv.Child(idx).(*Layout), idx
}

// ConfigStructGrid configures the StructGrid for the current struct
func (sv *StructView) ConfigStructGrid() {
	if sv.Struct == nil {
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
		jstag := field.Tag.Get("json")
		xmtag := field.Tag.Get("xml")
		if jstag == "-" || xmtag == "-" { // skip non-saved elements -- todo: options for what to skip
			return true
		}
		vwtag := field.Tag.Get("view")
		if vwtag == "-" {
			return true
		}
		vv := FieldToValueView(sv.Struct, field.Name, fval)
		if vv == nil { // shouldn't happen
			return true
		}
		vvp := fieldVal.Addr()
		vv.SetStructValue(vvp, sv.Struct, &field)
		vtyp := vv.WidgetType()
		// todo: other things with view tag..
		labnm := fmt.Sprintf("Lbl%v", field.Name)
		valnm := fmt.Sprintf("Val%v", field.Name)
		config.Add(KiT_Label, labnm)
		config.Add(vtyp, valnm) // todo: extend to diff types using interface..
		sv.Fields = append(sv.Fields, vv)
		return true
	})
	updt := sg.ConfigChildren(config, false)
	if updt {
		bitflag.Set(&sv.NodeFlags, int(NodeFlagFullReRender))
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
		vv.ConfigWidget(widg)
	}
}

func (sv *StructView) UpdateFromStruct() {
	sv.StdConfig()
	typ := kit.NonPtrType(reflect.TypeOf(sv.Struct))
	sv.SetTitle(fmt.Sprintf("Properties of %v", typ.Name()))
	sv.ConfigStructGrid()
}

func (sv *StructView) Render2D() {
	bitflag.Clear(&sv.NodeFlags, int(NodeFlagFullReRender))
	sv.Frame.Render2D()
}

// todo: see notes on treeview
func (sv *StructView) ReRender2D() (node Node2D, layout bool) {
	if bitflag.Has(sv.NodeFlags, int(NodeFlagFullReRender)) {
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
