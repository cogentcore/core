// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/types"
)

// NoSentenceCaseFor indicates to not transform field names in
// [StructView]s into "Sentence case" for types whose full,
// package-path-qualified name contains any of these strings.
// For example, this can be used to disable sentence casing
// for types with scientific abbreviations in field names,
// which are more readable when not sentence cased. However,
// this should not be needed in most circumstances.
var NoSentenceCaseFor []string

// NoSentenceCaseForType returns whether the given fully
// package-path-qualified name contains anything in the
// [NoSentenceCaseFor] list.
func NoSentenceCaseForType(tnm string) bool {
	return slices.ContainsFunc(NoSentenceCaseFor, func(s string) bool {
		return strings.Contains(tnm, s)
	})
}

// structField represents the values of one field being viewed
type structField struct {
	path string

	field reflect.StructField

	value, parent reflect.Value
}

// StructView represents a struct with rows of field names and editable values.
type StructView struct {
	core.Frame

	// Struct is the pointer to the struct that we are viewing.
	Struct any

	// Inline is whether to display the struct in one line.
	Inline bool

	// ViewPath is a record of parent view names that have led up to this view.
	// It is displayed as extra contextual information in view dialogs.
	ViewPath string

	// structFields are the fields of the current struct.
	structFields []*structField

	// isShouldShower is whether the struct implements [core.ShouldShower], which results
	// in additional updating being done at certain points.
	isShouldShower bool
}

func (sv *StructView) WidgetValue() any { return &sv.Struct }

func (sv *StructView) getStructFields() {
	var fields []*structField

	shouldShow := func(parent reflect.Value, field reflect.StructField) bool {
		if field.Tag.Get("view") == "-" {
			return false
		}
		if ss, ok := reflectx.UnderlyingPointer(parent).Interface().(core.ShouldShower); ok {
			sv.isShouldShower = true
			if !ss.ShouldShow(field.Name) {
				return false
			}
		}
		return true
	}

	reflectx.WalkFields(reflectx.Underlying(reflect.ValueOf(sv.Struct)),
		func(parent reflect.Value, field reflect.StructField, value reflect.Value) bool {
			return shouldShow(parent, field)
		},
		func(parent reflect.Value, field reflect.StructField, value reflect.Value) {
			if field.Tag.Get("view") == "add-fields" && field.Type.Kind() == reflect.Struct {
				reflectx.WalkFields(value,
					func(parent reflect.Value, sfield reflect.StructField, value reflect.Value) bool {
						return shouldShow(parent, sfield)
					},
					func(parent reflect.Value, sfield reflect.StructField, value reflect.Value) {
						fields = append(fields, &structField{path: field.Name + " • " + sfield.Name, field: sfield, value: value, parent: parent})
					})
			} else {
				fields = append(fields, &structField{path: field.Name, field: field, value: value, parent: parent})
			}
		})
	sv.structFields = fields
}

func (sv *StructView) OnInit() {
	sv.Frame.OnInit()
	sv.Style(func(s *styles.Style) {
		s.Align.Items = styles.Center
		if sv.Inline {
			return
		}
		s.Display = styles.Grid
		if sv.SizeClass() == core.SizeCompact {
			s.Columns = 1
		} else {
			s.Columns = 2
		}
	})

	sv.Maker(func(p *core.Plan) {
		if reflectx.AnyIsNil(sv.Struct) {
			return
		}

		sv.getStructFields()

		sc := true
		if len(NoSentenceCaseFor) > 0 {
			sc = !NoSentenceCaseForType(types.TypeNameValue(sv.Struct))
		}

		for i, f := range sv.structFields {
			label := f.path
			if sc {
				label = strcase.ToSentence(label)
			}
			labnm := fmt.Sprintf("label-%s", f.path)
			valnm := fmt.Sprintf("value-%s-%s", f.path, reflectx.ShortTypeName(f.field.Type))
			readOnlyTag := f.field.Tag.Get("edit") == "-"
			def, hasDef := f.field.Tag.Lookup("default")

			var labelWidget *core.Text
			var valueWidget core.Value

			core.AddAt(p, labnm, func(w *core.Text) {
				labelWidget = w
				w.Style(func(s *styles.Style) {
					s.SetTextWrap(false)
				})
				doc, _ := types.GetDoc(f.value, f.parent, f.field, label)
				w.SetTooltip(doc)
				if hasDef {
					w.SetTooltip("(Default: " + def + ") " + w.Tooltip)
					var isDef bool
					w.Style(func(s *styles.Style) {
						isDef = reflectx.ValueIsDefault(f.value, def)
						dcr := "(Double click to reset to default) "
						if !isDef {
							s.Color = colors.C(colors.Scheme.Primary.Base)
							s.Cursor = cursors.Poof
							if !strings.HasPrefix(w.Tooltip, dcr) {
								w.SetTooltip(dcr + w.Tooltip)
							}
						} else {
							w.SetTooltip(strings.TrimPrefix(w.Tooltip, dcr))
						}
					})
					w.OnDoubleClick(func(e events.Event) {
						if isDef {
							return
						}
						e.SetHandled()
						err := reflectx.SetFromDefaultTag(f.value, def)
						if err != nil {
							core.ErrorSnackbar(w, err, "Error setting default value")
						} else {
							w.Update()
							valueWidget.Update()
							valueWidget.AsWidget().SendChange(e)
						}
					})
				}
				w.Builder(func() {
					w.SetText(label)
				})
			})

			core.AddNew(p, valnm, func() core.Value {
				return core.NewValue(reflectx.UnderlyingPointer(f.value).Interface(), f.field.Tag)
			}, func(w core.Value) {
				valueWidget = w
				wb := w.AsWidget()
				doc, _ := types.GetDoc(f.value, f.parent, f.field, label)
				if wb.Tooltip == "" {
					wb.SetTooltip(doc)
				} else { // InitValueButton may set starting tooltip in OnInit
					wb.SetTooltip(wb.Tooltip + " " + doc)
				}
				if hasDef {
					wb.SetTooltip("(Default: " + def + ") " + wb.Tooltip)
				}
				wb.OnInput(func(e events.Event) {
					sv.Send(events.Input, e)
					if f.field.Tag.Get("immediate") == "+" {
						wb.SendChange(e)
					}
				})
				if !sv.IsReadOnly() && !readOnlyTag {
					wb.OnChange(func(e events.Event) {
						sv.SendChange(e)
						if hasDef {
							labelWidget.Update()
						}
						if sv.isShouldShower {
							sv.Update()
						}
					})
				}
				wb.Builder(func() {
					wb.SetReadOnly(sv.IsReadOnly() || readOnlyTag)
					if i < len(sv.structFields) {
						core.Bind(reflectx.UnderlyingPointer(sv.structFields[i].value).Interface(), w)
					}
				})
			})
		}
	})
}
