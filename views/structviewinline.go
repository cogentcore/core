// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"fmt"
	"reflect"
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

// StructViewInline represents a struct within a single line of
// field labels and value widgets. This is typically used for smaller structs.
type StructViewInline struct {
	core.Frame

	// Struct is the pointer to the struct that we are viewing.
	Struct any

	// StructValue is the [Value] associated with this struct view, if any.
	StructValue Value `set:"-"`

	// ViewPath is a record of parent view names that have led up to this view.
	// It is displayed as extra contextual information in view dialogs.
	ViewPath string

	// isShouldShower is whether the struct implements [core.ShouldShower], which results
	// in additional updating being done at certain points.
	isShouldShower bool
}

func (sv *StructViewInline) WidgetValue() any { return &sv.Struct }

func (sv *StructViewInline) OnInit() {
	sv.Frame.OnInit()
	sv.SetStyles()
}

func (sv *StructViewInline) SetStyles() {
	sv.Style(func(s *styles.Style) {
		s.Grow.Set(0, 0)
	})
}

func (sv *StructViewInline) Config(c *core.Config) {
	if reflectx.AnyIsNil(sv.Struct) {
		return
	}

	sc := true
	if len(NoSentenceCaseFor) > 0 {
		sc = !NoSentenceCaseForType(types.TypeNameValue(sv.Struct))
	}

	sval := reflectx.OnePointerUnderlyingValue(reflect.ValueOf(sv.Struct)).Interface()

	reflectx.WalkValueFlatFields(sval, func(fval any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		// todo: check tags, skip various etc
		vwtag := field.Tag.Get("view")
		if vwtag == "-" {
			return true
		}
		if reflectx.AnyIsNil(fieldVal) {
			// fmt.Println("field is nil:", fnm)
			return true
		}
		if ss, ok := sval.(core.ShouldShower); ok {
			sv.isShouldShower = true
			if !ss.ShouldShow(field.Name) {
				return true
			}
		}
		fnm := field.Name
		flab := fnm
		if sc {
			flab = strcase.ToSentence(fnm)
		}
		labnm := fmt.Sprintf("label-%v", fnm)
		valnm := fmt.Sprintf("value-%v", fnm)
		readOnlyTag := field.Tag.Get("edit") == "-"
		ttip := "" // TODO:
		def, hasDef := field.Tag.Lookup("default")

		core.Configure(c, labnm, func() *core.Text {
			w := core.NewText()
			w.Style(func(s *styles.Style) {
				s.SetTextWrap(false)
				s.Align.Self = styles.Center // was Start
				s.Min.X.Em(2)
			})
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
			return w
		}, func(w *core.Text) {
			w.SetText(flab)
		})

		core.Configure(c, valnm, func() core.Value {
			w := core.NewValue(fieldVal.Interface(), string(field.Tag))
			wb := w.AsWidget()
			// vvp := fieldVal.Addr()
			// vv.SetStructValue(vvp, sv.Struct, &field, sv.ViewPath)
			if !sv.IsReadOnly() && !readOnlyTag {
				wb.OnChange(func(e events.Event) {
					// sv.UpdateFieldAction()
					// if !reflectx.KindIsBasic(reflectx.NonPointerValue(vv.Val()).Kind()) {
					// 	if updater, ok := sv.Struct.(core.Updater); ok {
					// 		// fmt.Printf("updating: %v kind: %v\n", updater, vvv.Value.Kind())
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
		return true
	})
}
