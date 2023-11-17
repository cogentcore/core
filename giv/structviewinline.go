// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// StructViewInline represents a struct as a single line widget,
// for smaller structs and those explicitly marked inline.
type StructViewInline struct {
	gi.Frame

	// the struct that we are a view onto
	Struct any `set:"-"`

	// Value for the struct itself, if this was created within value view framework -- otherwise nil
	StructValView Value

	// if true add an edit action button at the end -- other users of this widget can then configure that -- it is called 'edit-action'
	AddButton bool

	// Value representations of the fields
	FieldViews []Value `json:"-" xml:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `json:"-" xml:"-" view:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// if true, some fields have default values -- update labels when values change
	HasDefs bool `json:"-" xml:"-" edit:"-"`

	// if true, some fields have viewif conditional view tags -- update after..
	HasViewIfs bool `json:"-" xml:"-" edit:"-"`
}

func (sv *StructViewInline) OnInit() {
	sv.StructViewInlineStyles()
}

func (sv *StructViewInline) StructViewInlineStyles() {
	sv.Style(func(s *styles.Style) {
		s.Align.Y = styles.AlignStart
	})
}

// SetStruct sets the source struct that we are viewing -- rebuilds the
// children to represent this struct
func (sv *StructViewInline) SetStruct(st any) *StructViewInline {
	if sv.Struct != st {
		sv.Struct = st
		sv.Update()
	}
	return sv
}

func (sv *StructViewInline) ConfigWidget(sc *gi.Scene) {
	sv.ConfigStruct(sc)
}

// ConfigStruct configures the children for the current struct
func (sv *StructViewInline) ConfigStruct(sc *gi.Scene) bool {
	if laser.AnyIsNil(sv.Struct) {
		return false
	}
	config := ki.Config{}
	// note: widget re-use does not work due to all the closures
	sv.DeleteChildren(ki.DestroyKids)
	sv.FieldViews = make([]Value, 0)
	laser.FlatFieldsValueFunc(sv.Struct, func(fval any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		// todo: check tags, skip various etc
		vwtag := field.Tag.Get("view")
		if vwtag == "-" {
			return true
		}
		viewif := field.Tag.Get("viewif")
		if viewif != "" {
			sv.HasViewIfs = true
			if !StructViewIf(viewif, field, sv.Struct) {
				return true
			}
		}
		vv := FieldToValue(sv.Struct, field.Name, fval)
		if vv == nil { // shouldn't happen
			return true
		}
		vvp := fieldVal.Addr()
		vv.SetStructValue(vvp, sv.Struct, &field, sv.TmpSave, sv.ViewPath)
		vtyp := vv.WidgetType()
		// todo: other things with view tag..
		labnm := "label-" + field.Name
		valnm := "value-" + field.Name
		config.Add(gi.LabelType, labnm)
		config.Add(vtyp, valnm) // todo: extend to diff types using interface..
		sv.FieldViews = append(sv.FieldViews, vv)
		return true
	})
	if sv.AddButton {
		config.Add(gi.ButtonType, "edit-action")
	}
	mods, updt := sv.ConfigChildren(config)
	if !mods {
		updt = sv.UpdateStart()
	}
	sv.HasDefs = false
	for i, vv := range sv.FieldViews {
		lbl := sv.Child(i * 2).(*gi.Label)
		lbl.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
			s.Align.Y = styles.AlignStart
			s.Min.X.Em(2)
		})
		vvb := vv.AsValueBase()
		vvb.ViewPath = sv.ViewPath
		w, wb := gi.AsWidget(sv.Child((i * 2) + 1))
		hasDef, readOnlyTag := StructViewFieldTags(vv, lbl, w, sv.IsReadOnly()) // in structview.go
		if hasDef {
			sv.HasDefs = true
		}
		if wb.Class == "" {
			wb.Class = "configed"
			vv.ConfigWidget(w, sc)
		} else {
			vvb.Widget = w
			vv.UpdateWidget()
		}
		if !sv.IsReadOnly() && !readOnlyTag {
			vvb.OnChange(func(e events.Event) {
				sv.UpdateFieldAction()
				if !laser.KindIsBasic(laser.NonPtrValue(vvb.Value).Kind()) {
					if updtr, ok := sv.Struct.(gi.Updater); ok {
						// fmt.Printf("updating: %v kind: %v\n", updtr, vvv.Value.Kind())
						updtr.Update()
					}
				}
				sv.SendChange(e)
			})
		}
	}
	sv.UpdateEnd(updt)
	return updt
}

func (sv *StructViewInline) UpdateFields() {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
		vv.UpdateWidget()
	}
	sv.UpdateEndLayout(updt)
}

func (sv *StructViewInline) UpdateFieldAction() {
	if sv.HasViewIfs {
		fmt.Println("did view if update")
		sv.Update()
		sv.SetNeedsLayout()
	} else if sv.HasDefs {
		updt := sv.UpdateStart()
		for i, vv := range sv.FieldViews {
			lbl := sv.Child(i * 2).(*gi.Label)
			StructViewFieldDefTag(vv, lbl)
		}
		sv.UpdateEndRender(updt)
	}
}

// func (sv *StructViewInline) SizeUp(sc *gi.Scene) {
// 	updt := sv.ConfigStruct(sc)
// 	if updt {
// 		sv.ApplyStyleTree(sc)
// 	}
// 	sv.Frame.SizeUp(sc)
// }
