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
		if sv.Inline {
			s.Align.Items = styles.Center
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

	shouldShow := func(field reflect.StructField, stru any) bool {
		ftags := field.Tag
		vwtag := ftags.Get("view")
		if vwtag == "-" {
			return false
		}
		if ss, ok := stru.(core.ShouldShower); ok {
			sv.isShouldShower = true
			if !ss.ShouldShow(field.Name) {
				return false
			}
		}
		return true
	}

	addField := func(c *core.Config, structVal, fieldVal any, field reflect.StructField, fnm string) {
		if fieldVal == nil {
			return
		}
		flab := fnm
		if sc {
			flab = strcase.ToSentence(fnm)
		}
		labnm := fmt.Sprintf("label-%v", fnm)
		valnm := fmt.Sprintf("value-%v", fnm)
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
					isDef = reflectx.ValueIsDefault(reflect.ValueOf(fieldVal), def)
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
					fmt.Println(fnm, "doubleclick", isDef)
					if isDef {
						return
					}
					e.SetHandled()
					err := reflectx.SetFromDefaultTag(reflect.ValueOf(fieldVal), def)
					if err != nil {
						core.ErrorSnackbar(w, err, "Error setting default value")
					} else {
						w.Update() // TODO: critically this needs to be the value widget, and the text separately
						sv.SendChange(e)
					}
				})
			}
		}, func(w *core.Text) {
			w.SetText(flab)
		})

		core.ConfigureNew(c, valnm, func() core.Value {
			w := core.NewValue(fieldVal, field.Tag)
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
					fmt.Println("updating", sv)
					// sv.Update()
				})
			}
			return w
		}, func(w core.Value) {
			w.AsWidget().SetReadOnly(sv.IsReadOnly() || readOnlyTag)
			core.Bind(fieldVal, w)
		})
	}

	reflectx.WalkValueFlatFieldsIf(sv.Struct,
		func(stru any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			return shouldShow(field, sv.Struct)
		},
		func(fval any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			// todo: check tags, skip various etc
			ftags := field.Tag
			vwtag := ftags.Get("view")
			if !shouldShow(field, sv.Struct) {
				return true
			}
			if vwtag == "add-fields" && field.Type.Kind() == reflect.Struct {
				fvalp := fieldVal.Addr().Interface()
				reflectx.WalkValueFlatFieldsIf(fvalp,
					func(stru any, typ reflect.Type, sfield reflect.StructField, fieldVal reflect.Value) bool {
						return shouldShow(sfield, fvalp)
					},
					func(sfval any, styp reflect.Type, sfield reflect.StructField, sfieldVal reflect.Value) bool {
						if !shouldShow(sfield, fvalp) {
							return true
						}
						fnm := field.Name + " • " + sfield.Name
						addField(c, fvalp, sfval, sfield, fnm)
						return true
					})
				return true
			}
			addField(c, sv.Struct, fval, field, field.Name)
			return true
		})
}
