// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"reflect"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/reflectx"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// StructViewInline represents a struct within a single line of
// field labels and value widgets. This is typically used for smaller structs.
type StructViewInline struct {
	core.Layout

	// Struct is the pointer to the struct that we are viewing.
	Struct any `set:"-"`

	// StructValue is the [Value] associated with this struct view, if any.
	StructValue Value `set:"-"`

	// Values are [Value] representations of the struct fields values.
	Values []Value `json:"-" xml:"-"`

	// ViewPath is a record of parent view names that have led up to this view.
	// It is displayed as extra contextual information in view dialogs.
	ViewPath string

	// isShouldShower is whether the struct implements [core.ShouldShower], which results
	// in additional updating being done at certain points.
	isShouldShower bool
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

func (sv *StructViewInline) Config() {
	if reflectx.AnyIsNil(sv.Struct) {
		return
	}
	config := tree.Config{}
	// note: widget re-use does not work due to all the closures
	sv.DeleteChildren()
	sv.Values = make([]Value, 0)
	reflectx.WalkValueFlatFields(sv.Struct, func(fval any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		// todo: check tags, skip various etc
		vwtag := field.Tag.Get("view")
		if vwtag == "-" {
			return true
		}
		if ss, ok := sv.Struct.(core.ShouldShower); ok {
			sv.isShouldShower = true
			if !ss.ShouldShow(field.Name) {
				return true
			}
		}
		vv := FieldToValue(sv.Struct, field.Name, fval)
		if vv == nil { // shouldn't happen
			return true
		}
		vvp := fieldVal.Addr()
		vv.SetStructValue(vvp, sv.Struct, &field, sv.ViewPath)
		vtyp := vv.WidgetType()
		// todo: other things with view tag..
		labnm := "label-" + field.Name
		valnm := "value-" + field.Name
		config.Add(core.TextType, labnm)
		config.Add(vtyp, valnm) // todo: extend to diff types using interface..
		sv.Values = append(sv.Values, vv)
		return true
	})
	sv.ConfigChildren(config)
	for i, vv := range sv.Values {
		lbl := sv.Child(i * 2).(*core.Text)
		lbl.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
			s.Align.Self = styles.Center // was Start
			s.Min.X.Em(2)
		})
		vv.AsValueData().ViewPath = sv.ViewPath
		w, _ := core.AsWidget(sv.Child((i * 2) + 1))
		hasDef, readOnlyTag := StructViewFieldTags(vv, lbl, w, sv.IsReadOnly()) // in structview.go
		if hasDef {
			lbl.Style(func(s *styles.Style) {
				dtag, _ := vv.Tag("default")
				isDef, _ := StructFieldIsDef(dtag, vv.Val().Interface(), reflectx.NonPointerValue(vv.Val()).Kind())
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
				isDef, _ := StructFieldIsDef(dtag, vv.Val().Interface(), reflectx.NonPointerValue(vv.Val()).Kind())
				if isDef {
					return
				}
				e.SetHandled()
				err := reflectx.SetFromDefaultTag(vv.Val(), dtag)
				if err != nil {
					core.ErrorSnackbar(lbl, err, "Error setting default value")
				} else {
					vv.Update()
					vv.SendChange(e)
				}
			})
		}
		Config(vv, w)
		vv.AsWidgetBase().OnInput(sv.HandleEvent)
		if !sv.IsReadOnly() && !readOnlyTag {
			vv.OnChange(func(e events.Event) {
				sv.UpdateFieldAction()
				if !reflectx.KindIsBasic(reflectx.NonPointerValue(vv.Val()).Kind()) {
					if updtr, ok := sv.Struct.(core.Updater); ok {
						// fmt.Printf("updating: %v kind: %v\n", updtr, vvv.Value.Kind())
						updtr.Update()
					}
				}
				if hasDef {
					lbl.Update()
				}
				sv.SendChange(e)
			})
		}
	}
}

func (sv *StructViewInline) UpdateFields() {
	for _, vv := range sv.Values {
		vv.Update()
	}
	sv.NeedsLayout()
}

func (sv *StructViewInline) UpdateFieldAction() {
	if sv.isShouldShower {
		sv.Update()
	}
}
