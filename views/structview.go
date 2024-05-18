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

	// isShouldShower is whether the struct implements [core.ShouldShower], which results
	// in additional updating being done at certain points.
	isShouldShower bool
}

func (sv *StructView) WidgetValue() any { return &sv.Struct }

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
}

func (sv *StructView) Config(c *core.Config) {
	if reflectx.AnyIsNil(sv.Struct) {
		return
	}

	sc := true
	if len(NoSentenceCaseFor) > 0 {
		sc = !NoSentenceCaseForType(types.TypeNameValue(sv.Struct))
	}

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

	addField := func(field reflect.StructField, value reflect.Value, name string) {
		label := name
		if sc {
			label = strcase.ToSentence(name)
		}
		labnm := fmt.Sprintf("label-%v", name)
		valnm := fmt.Sprintf("value-%v", name)
		readOnlyTag := field.Tag.Get("edit") == "-"
		ttip := "" // TODO(config)
		def, hasDef := field.Tag.Lookup("default")

		core.Configure(c, labnm, func(w *core.Text) {
			w.Style(func(s *styles.Style) {
				s.SetTextWrap(false)
			})
			// w.Tooltip = vv.Doc()
			// vv.AsValueData().ViewPath = sv.ViewPath
			if hasDef {
				ttip = "(Default: " + def + ") " + ttip
				var isDef bool
				w.Style(func(s *styles.Style) {
					isDef = reflectx.ValueIsDefault(value, def)
					dcr := "(Double click to reset to default) "
					if !isDef {
						s.Color = colors.C(colors.Scheme.Primary.Base)
						s.Cursor = cursors.Poof
						if !strings.HasPrefix(w.Tooltip, dcr) {
							w.Tooltip = dcr + w.Tooltip
						}
					} else {
						w.Tooltip = strings.TrimPrefix(w.Tooltip, dcr)
					}
				})
				w.OnDoubleClick(func(e events.Event) {
					fmt.Println(label, "doubleclick", isDef)
					if isDef {
						return
					}
					e.SetHandled()
					err := reflectx.SetFromDefaultTag(value, def)
					if err != nil {
						core.ErrorSnackbar(w, err, "Error setting default value")
					} else {
						w.Update() // TODO: critically this needs to be the value widget, and the text separately
						sv.SendChange(e)
					}
				})
			}
		}, func(w *core.Text) {
			w.SetText(label)
		})

		core.ConfigureNew(c, valnm, func() core.Value {
			w := core.NewValue(reflectx.UnderlyingPointer(value).Interface(), field.Tag)
			wb := w.AsWidget()
			// vv.AsValueData().ViewPath = sv.ViewPath
			// svv := FieldToValue(fvalp, sfield.Name, sfval)
			// if svv == nil { // shouldn't happen
			// 	return true
			// }
			// svvp := sfieldVal.Addr()
			// svv.SetStructValue(svvp, fvalp, &sfield, sv.ViewPath)
			// todo: other things with view tag..
			// vv.AsWidgetBase().OnInput(func(e events.Event) {
			// 	if tag, _ := vv.Tag("immediate"); tag == "+" {
			// 		wb.SendChange(e)
			// 		sv.SendChange(e)
			// 	}
			// 	sv.Send(events.Input, e)
			// })
			if !sv.IsReadOnly() && !readOnlyTag {
				wb.OnChange(func(e events.Event) {
					// sv.UpdateFieldAction()
					// note: updating vv here is redundant -- relevant field will have already updated
					// if !reflectx.KindIsBasic(reflectx.NonPointerValue(vv.Val()).Kind()) {
					// 	if updater, ok := sv.Struct.(core.Updater); ok {
					// 		updater.Update()
					// 	}
					// }
					// if hasDef {
					// 	lbl.Update()
					// }
					sv.SendChange(e)
					// sv.Update()
				})
			}
			return w
		}, func(w core.Value) {
			w.AsWidget().SetReadOnly(sv.IsReadOnly() || readOnlyTag)
			core.Bind(reflectx.UnderlyingPointer(value).Interface(), w)
		})
	}

	reflectx.WalkFlatFields(reflectx.Underlying(reflect.ValueOf(sv.Struct)),
		func(parent reflect.Value, field reflect.StructField, value reflect.Value) bool {
			return shouldShow(parent, field)
		},
		func(parent reflect.Value, field reflect.StructField, value reflect.Value) bool {
			if field.Tag.Get("view") == "add-fields" && field.Type.Kind() == reflect.Struct {
				reflectx.WalkFlatFields(value,
					func(parent reflect.Value, sfield reflect.StructField, value reflect.Value) bool {
						return shouldShow(parent, sfield)
					},
					func(parent reflect.Value, sfield reflect.StructField, value reflect.Value) bool {
						addField(sfield, value, field.Name+" • "+sfield.Name)
						return true
					})
				return true
			}
			addField(field, value, field.Name)
			return true
		})
}
