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
// [Form]s into "Sentence case" for types whose full,
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

// structField represents the values of one struct field being viewed.
type structField struct {
	path          string
	field         reflect.StructField
	value, parent reflect.Value
}

// Form represents a struct with rows of field names and editable values.
type Form struct {
	core.Frame

	// Struct is the pointer to the struct that we are viewing.
	Struct any

	// Inline is whether to display the form in one line.
	Inline bool

	// structFields are the fields of the current struct.
	structFields []*structField

	// isShouldShower is whether the struct implements [core.ShouldShower], which results
	// in additional updating being done at certain points.
	isShouldShower bool
}

func (fm *Form) WidgetValue() any { return &fm.Struct }

func (fm *Form) getStructFields() {
	var fields []*structField

	shouldShow := func(parent reflect.Value, field reflect.StructField) bool {
		if field.Tag.Get("view") == "-" {
			return false
		}
		if ss, ok := reflectx.UnderlyingPointer(parent).Interface().(core.ShouldShower); ok {
			fm.isShouldShower = true
			if !ss.ShouldShow(field.Name) {
				return false
			}
		}
		return true
	}

	reflectx.WalkFields(reflectx.Underlying(reflect.ValueOf(fm.Struct)),
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
	fm.structFields = fields
}

func (fm *Form) Init() {
	fm.Frame.Init()
	fm.Styler(func(s *styles.Style) {
		s.Align.Items = styles.Center
		if fm.Inline {
			return
		}
		s.Display = styles.Grid
		if fm.SizeClass() == core.SizeCompact {
			s.Columns = 1
		} else {
			s.Columns = 2
		}
	})

	fm.Maker(func(p *core.Plan) {
		if reflectx.AnyIsNil(fm.Struct) {
			return
		}

		fm.getStructFields()

		sc := true
		if len(NoSentenceCaseFor) > 0 {
			sc = !NoSentenceCaseForType(types.TypeNameValue(fm.Struct))
		}

		for i, f := range fm.structFields {
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
				w.Styler(func(s *styles.Style) {
					s.SetTextWrap(false)
				})
				doc, _ := types.GetDoc(f.value, f.parent, f.field, label)
				w.SetTooltip(doc)
				if hasDef {
					w.SetTooltip("(Default: " + def + ") " + w.Tooltip)
					var isDef bool
					w.Styler(func(s *styles.Style) {
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
							valueWidget.AsWidget().Update()
							valueWidget.AsWidget().SendChange(e)
						}
					})
				}
				w.Updater(func() {
					w.SetText(label)
				})
			})

			core.AddNew(p, valnm, func() core.Value {
				return core.NewValue(reflectx.UnderlyingPointer(f.value).Interface(), f.field.Tag)
			}, func(w core.Value) {
				valueWidget = w
				wb := w.AsWidget()
				doc, _ := types.GetDoc(f.value, f.parent, f.field, label)
				// InitValueButton may set starting wb.Tooltip in Init
				if wb.Tooltip == "" {
					wb.SetTooltip(doc)
				} else if doc == "" {
					wb.SetTooltip(wb.Tooltip)
				} else {
					wb.SetTooltip(wb.Tooltip + " " + doc)
				}
				if hasDef {
					wb.SetTooltip("(Default: " + def + ") " + wb.Tooltip)
				}
				wb.OnInput(func(e events.Event) {
					fm.Send(events.Input, e)
					if f.field.Tag.Get("immediate") == "+" {
						wb.SendChange(e)
					}
				})
				if !fm.IsReadOnly() && !readOnlyTag {
					wb.OnChange(func(e events.Event) {
						fm.SendChange(e)
						if hasDef {
							labelWidget.Update()
						}
						if fm.isShouldShower {
							fm.Update()
						}
					})
				}
				wb.Updater(func() {
					wb.SetReadOnly(fm.IsReadOnly() || readOnlyTag)
					if i < len(fm.structFields) {
						core.Bind(reflectx.UnderlyingPointer(fm.structFields[i].value).Interface(), w)
						vc := core.JoinValueTitle(fm.ValueTitle, label)
						if vc != wb.ValueTitle {
							wb.ValueTitle = vc + " (" + wb.ValueTitle + ")"
						}
					}
				})
			})
		}
	})
}
