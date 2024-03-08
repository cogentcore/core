// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"reflect"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
)

// StructViewInline represents a struct as a single line widget,
// for smaller structs and those explicitly marked inline.
type StructViewInline struct {
	gi.Layout

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

	// IsShouldShower is whether the struct implements [gi.ShouldShower], which results
	// in additional updating being done at certain points.
	IsShouldShower bool `set:"-" json:"-" xml:"-" edit:"-"`
}

func (sv *StructViewInline) OnInit() {
	sv.Layout.OnInit()
	sv.SetStyles()
}

func (sv *StructViewInline) SetStyles() {
	sv.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
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

func (sv *StructViewInline) ConfigWidget() {
	sv.ConfigStruct()
}

// ConfigStruct configures the children for the current struct
func (sv *StructViewInline) ConfigStruct() bool {
	if laser.AnyIsNil(sv.Struct) {
		return false
	}
	config := ki.Config{}
	// note: widget re-use does not work due to all the closures
	sv.DeleteChildren()
	sv.FieldViews = make([]Value, 0)
	laser.FlatFieldsValueFunc(sv.Struct, func(fval any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		// todo: check tags, skip various etc
		vwtag := field.Tag.Get("view")
		if vwtag == "-" {
			return true
		}
		if ss, ok := sv.Struct.(gi.ShouldShower); ok {
			sv.IsShouldShower = true
			if !ss.ShouldShow(field.Name) {
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
	for i, vv := range sv.FieldViews {
		lbl := sv.Child(i * 2).(*gi.Label)
		lbl.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
			s.Align.Self = styles.Center // was Start
			s.Min.X.Em(2)
		})
		vvb := vv.AsValueBase()
		vvb.ViewPath = sv.ViewPath
		w, wb := gi.AsWidget(sv.Child((i * 2) + 1))
		hasDef, readOnlyTag := StructViewFieldTags(vv, lbl, w, sv.IsReadOnly()) // in structview.go
		if hasDef {
			lbl.Style(func(s *styles.Style) {
				dtag, _ := vv.Tag("default")
				isDef, _ := StructFieldIsDef(dtag, vv.Val().Interface(), laser.NonPtrValue(vv.Val()).Kind())
				dcr := "(Double click to reset to default) "
				if !isDef {
					s.Color = colors.C(colors.Scheme.Primary.Base)
					s.Cursor = cursors.Poof
					if !strings.HasPrefix(lbl.Tooltip, dcr) {
						lbl.Tooltip = dcr + lbl.Tooltip
					}
				} else {
					lbl.Tooltip = strings.TrimPrefix(lbl.Tooltip, dcr)
				}
			})
			lbl.OnDoubleClick(func(e events.Event) {
				dtag, _ := vv.Tag("default")
				isDef, _ := StructFieldIsDef(dtag, vv.Val().Interface(), laser.NonPtrValue(vv.Val()).Kind())
				if isDef {
					return
				}
				err := laser.SetFromDefaultTag(vv.Val(), dtag)
				if err != nil {
					gi.ErrorSnackbar(lbl, err, "Error setting default value")
				} else {
					vv.SendChange(e)
				}
			})
		}
		if wb.Prop("configured") == nil {
			wb.SetProp("configured", true)
			vv.ConfigWidget(w)
			vvb.AsWidgetBase().OnInput(sv.HandleEvent)
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
	if sv.IsShouldShower {
		sv.Update()
		sv.NeedsLayout(true)
	}
}
