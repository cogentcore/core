// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strconv"
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
	Struct any `set:"-"`

	// StructValue is the Value for the struct itself
	// if this was created within the Value framework.
	// Otherwise, it is nil.
	StructValue Value `set:"-"`

	// ViewPath is a record of parent view names that have led up to this view.
	// It is displayed as extra contextual information in view dialogs.
	ViewPath string

	// isShouldShower is whether the struct implements [core.ShouldShower], which results
	// in additional updating being done at certain points.
	isShouldShower bool
}

func (sv *StructView) OnInit() {
	sv.Frame.OnInit()
	sv.SetStyles()
}

func (sv *StructView) SetStyles() {
	sv.Style(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Grow.Set(0, 0)
		if sv.SizeClass() == core.SizeCompact {
			s.Columns = 1
		} else {
			s.Columns = 2
		}
	})
}

// SetStruct sets the source struct that we are viewing -- rebuilds the
// children to represent this struct
func (sv *StructView) SetStruct(st any) *StructView {
	sv.Struct = st
	sv.Update()
	return sv
}

func (sv *StructView) Config(c *core.Config) {
	if reflectx.AnyIsNil(sv.Struct) {
		return
	}

	sc := true
	if len(NoSentenceCaseFor) > 0 {
		sc = !NoSentenceCaseForType(types.TypeNameValue(sv.Struct))
	}

	dupeFields := map[string]bool{} // todo: build this into basic config

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
		if fieldVal == nil || reflectx.AnyIsNil(fieldVal) {
			// fmt.Println("field is nil:", fnm)
			return
		}
		if _, exists := dupeFields[fnm]; exists {
			slog.Error("StructView: duplicate field name:", "name:", fnm)
			return
		} else {
			dupeFields[fnm] = true
		}
		flab := fnm
		if sc {
			flab = strcase.ToSentence(fnm)
		}
		labnm := fmt.Sprintf("label-%v", fnm)
		valnm := fmt.Sprintf("value-%v", fnm)
		readOnlyTag := field.Tag.Get("edit") == "-"

		core.Configure(c, labnm, func() *core.Text {
			w := core.NewText()
			w.Style(func(s *styles.Style) {
				s.SetTextWrap(false)
			})
			// w.Tooltip = vv.Doc()
			// vv.AsValueData().ViewPath = sv.ViewPath
			def, hasDef := field.Tag.Lookup("default")
			// hasDef, readOnlyTag := StructViewFieldTags(vv, lbl, w, sv.IsReadOnly())
			if hasDef {
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
			return w
		}, func(w *core.Text) {
			w.SetText(flab)
		})

		core.Configure(c, valnm, func() core.Value {
			w := core.NewValue(fieldVal)
			wb := w.AsWidget()
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
				})
			}
			return w
		}, func(w core.Value) {
			w.AsWidget().SetReadOnly(sv.IsReadOnly() || readOnlyTag)
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

/////////////////////////////////////////////////////////////////////////
//  Tag parsing

// StructViewFieldTags processes the tags for a field in a struct view, setting
// the properties on the label or widget appropriately
// returns true if there were any "default" default tags -- if so, needs updating
func StructViewFieldTags(vv Value, lbl *core.Text, w core.Widget, isReadOnly bool) (hasDef, readOnlyTag bool) {
	lbl.Text = vv.Label()
	if et, has := vv.Tag("edit"); has && et == "-" {
		readOnlyTag = true
		w.AsWidget().SetReadOnly(true)
	} else {
		if isReadOnly {
			w.AsWidget().SetReadOnly(true)
			vv.SetTag("edit", "-")
		}
	}
	defStr, hasDef := vv.Tag("default")
	if hasDef {
		lbl.Tooltip = "(Default: " + defStr + ") " + vv.Doc()
	} else {
		lbl.Tooltip = vv.Doc()
	}
	return
}

// StructFieldIsDef processses "default" tag for default value(s) of field
// defs = default values as strings as either comma-separated list of valid values
// or low:high value range (only for int or float numeric types)
// valPtr = pointer to value
// returns true if value is default, and string to add to tooltip for default values.
// Uses JSON format for composite types (struct, slice, map), replacing " with '
// so it is easier to use in def tag.
func StructFieldIsDef(defs string, valPtr any, kind reflect.Kind) (bool, string) {
	defStr := "(Default: " + defs + ")"
	if kind >= reflect.Int && kind <= reflect.Complex128 && strings.Contains(defs, ":") {
		dtags := strings.Split(defs, ":")
		lo, _ := strconv.ParseFloat(dtags[0], 64)
		hi, _ := strconv.ParseFloat(dtags[1], 64)
		vf, err := reflectx.ToFloat(valPtr)
		if err != nil {
			slog.Error("views.StructFieldIsDef: error parsing struct field numerical range def tag", "type", reflectx.NonPointerType(reflect.TypeOf(valPtr)), "def", defs, "err", err)
			return true, defStr
		}
		return lo <= vf && vf <= hi, defStr
	}
	v := reflectx.NonPointerValue(reflect.ValueOf(valPtr))
	dtags := strings.Split(defs, ",")
	if strings.ContainsAny(defs, "{[") { // complex type, so don't split on commas
		dtags = []string{defs}
	}
	for _, def := range dtags {
		def = reflectx.FormatDefault(def)
		if def == "" {
			if v.IsZero() {
				return true, defStr
			}
			continue
		}
		dv := reflect.New(v.Type())
		err := reflectx.SetRobust(dv.Interface(), def)
		if err != nil {
			slog.Error("views.StructFieldIsDef: error parsing struct field def tag", "type", v.Type(), "def", def, "err", err)
			return true, defStr
		}
		if reflect.DeepEqual(v.Interface(), dv.Elem().Interface()) {
			return true, defStr
		}
	}
	return false, defStr
}
