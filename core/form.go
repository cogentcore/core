// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// Form represents a struct with rows of field names and editable values.
type Form struct {
	Frame

	// Struct is the pointer to the struct that we are viewing.
	Struct any

	// Inline is whether to display the form in one line.
	Inline bool

	// structFields are the fields of the current struct.
	structFields []*structField

	// isShouldDisplayer is whether the struct implements [ShouldDisplayer], which results
	// in additional updating being done at certain points.
	isShouldDisplayer bool
}

// structField represents the values of one struct field being viewed.
type structField struct {
	path          string
	field         reflect.StructField
	value, parent reflect.Value
}

// NoSentenceCaseFor indicates to not transform field names in
// [Form]s into "Sentence case" for types whose full,
// package-path-qualified name contains any of these strings.
// For example, this can be used to disable sentence casing
// for types with scientific abbreviations in field names,
// which are more readable when not sentence cased. However,
// this should not be needed in most circumstances.
var NoSentenceCaseFor []string

// noSentenceCaseForType returns whether the given fully
// package-path-qualified name contains anything in the
// [NoSentenceCaseFor] list.
func noSentenceCaseForType(tnm string) bool {
	return slices.ContainsFunc(NoSentenceCaseFor, func(s string) bool {
		return strings.Contains(tnm, s)
	})
}

// ShouldDisplayer is an interface that determines whether a named field
// should be displayed in [Form].
type ShouldDisplayer interface {

	// ShouldDisplay returns whether the given named field should be displayed.
	ShouldDisplay(field string) bool
}

func (fm *Form) WidgetValue() any { return &fm.Struct }

func (fm *Form) getStructFields() {
	var fields []*structField

	shouldShow := func(parent reflect.Value, field reflect.StructField) bool {
		if field.Tag.Get("display") == "-" {
			return false
		}
		if ss, ok := reflectx.UnderlyingPointer(parent).Interface().(ShouldDisplayer); ok {
			fm.isShouldDisplayer = true
			if !ss.ShouldDisplay(field.Name) {
				return false
			}
		}
		return true
	}

	reflectx.WalkFields(reflectx.Underlying(reflect.ValueOf(fm.Struct)),
		func(parent reflect.Value, field reflect.StructField, value reflect.Value) bool {
			return shouldShow(parent, field)
		},
		func(parent reflect.Value, parentField *reflect.StructField, field reflect.StructField, value reflect.Value) {
			if field.Tag.Get("display") == "add-fields" && field.Type.Kind() == reflect.Struct {
				reflectx.WalkFields(value,
					func(parent reflect.Value, sfield reflect.StructField, value reflect.Value) bool {
						return shouldShow(parent, sfield)
					},
					func(parent reflect.Value, parentField *reflect.StructField, sfield reflect.StructField, value reflect.Value) {
						// if our parent field is read only, we must also be
						if field.Tag.Get("edit") == "-" && sfield.Tag.Get("edit") == "" {
							sfield.Tag += ` edit:"-"`
						}
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
		if fm.SizeClass() == SizeCompact {
			s.Columns = 1
		} else {
			s.Columns = 2
		}
	})

	fm.Maker(func(p *tree.Plan) {
		if reflectx.AnyIsNil(fm.Struct) {
			return
		}

		fm.getStructFields()

		sc := true
		if len(NoSentenceCaseFor) > 0 {
			sc = !noSentenceCaseForType(types.TypeNameValue(fm.Struct))
		}

		for i, f := range fm.structFields {
			label := f.path
			if sc {
				label = strcase.ToSentence(label)
			}
			if lt, ok := f.field.Tag.Lookup("label"); ok {
				label = lt
			}
			labnm := fmt.Sprintf("label-%s", f.path)
			// we must have a different name for different types
			// so that the widget can be re-made for a new type
			typnm := reflectx.ShortTypeName(f.field.Type)
			// we must have a different name for invalid values
			// so that the widget can be re-made for valid values
			if !reflectx.Underlying(f.value).IsValid() {
				typnm = "invalid"
			}
			// We must have a different name for different indexes so that the index
			// is always guaranteed to be accurate, which is required since we use it
			// as the ground truth everywhere. The index could otherwise become invalid,
			// such as when a ShouldDisplayer condition is newly satisfied
			// (see https://github.com/cogentcore/core/issues/1096).
			valnm := fmt.Sprintf("value-%s-%s-%d", f.path, typnm, i)
			readOnlyTag := f.field.Tag.Get("edit") == "-"
			def, hasDef := f.field.Tag.Lookup("default")

			var labelWidget *Text
			var valueWidget Value

			tree.AddAt(p, labnm, func(w *Text) {
				labelWidget = w
				w.Styler(func(s *styles.Style) {
					s.SetTextWrap(false)
				})
				// TODO: technically we should recompute doc, readOnlyTag,
				// def, hasDef, etc every time, as this is not fully robust
				// (see https://github.com/cogentcore/core/issues/1098).
				doc, _ := types.GetDoc(f.value, f.parent, f.field, label)
				w.SetTooltip(doc)
				if hasDef {
					w.SetTooltip("(Default: " + def + ") " + w.Tooltip)
					var isDef bool
					w.Styler(func(s *styles.Style) {
						f := fm.structFields[i]
						isDef = reflectx.ValueIsDefault(f.value, def)
						dcr := "(Double click to reset to default) "
						if !isDef {
							s.Color = colors.Scheme.Primary.Base
							s.Cursor = cursors.Poof
							if !strings.HasPrefix(w.Tooltip, dcr) {
								w.SetTooltip(dcr + w.Tooltip)
							}
						} else {
							w.SetTooltip(strings.TrimPrefix(w.Tooltip, dcr))
						}
					})
					w.OnDoubleClick(func(e events.Event) {
						f := fm.structFields[i]
						if isDef {
							return
						}
						e.SetHandled()
						err := reflectx.SetFromDefaultTag(f.value, def)
						if err != nil {
							ErrorSnackbar(w, err, "Error setting default value")
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

			tree.AddNew(p, valnm, func() Value {
				return NewValue(reflectx.UnderlyingPointer(f.value).Interface(), f.field.Tag)
			}, func(w Value) {
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
					f := fm.structFields[i]
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
						if fm.isShouldDisplayer {
							fm.Update()
						}
					})
				}
				wb.Updater(func() {
					wb.SetReadOnly(fm.IsReadOnly() || readOnlyTag)
					if i < len(fm.structFields) {
						f := fm.structFields[i]
						Bind(reflectx.UnderlyingPointer(f.value).Interface(), w)
						vc := joinValueTitle(fm.ValueTitle, label)
						if vc != wb.ValueTitle {
							wb.ValueTitle = vc + " (" + wb.ValueTitle + ")"
						}
					}
				})
			})
		}
	})
}
