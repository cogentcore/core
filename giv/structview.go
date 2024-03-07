// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/states"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
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

// StructView represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame.
type StructView struct {
	gi.Frame

	// the struct that we are a view onto
	Struct any `set:"-"`

	// Value for the struct itself, if this was created within value view framework -- otherwise nil
	StructValView Value `set:"-"`

	// has the value of any field changed?  updated by the ViewSig signals from fields
	Changed bool `set:"-"`

	// Value representations of the fields
	FieldViews []Value `set:"-" json:"-" xml:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `json:"-" xml:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// IsShouldShower is whether the struct implements [gi.ShouldShower], which results
	// in additional updating being done at certain points.
	IsShouldShower bool `set:"-" json:"-" xml:"-" edit:"-"`

	// extra tags by field name -- from type properties
	TypeFieldTags map[string]string `set:"-" json:"-" xml:"-" edit:"-"`
}

func (sv *StructView) OnInit() {
	sv.Frame.OnInit()
	sv.SetStyles()
}

func (sv *StructView) SetStyles() {
	sv.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(0, 0)
	})
	sv.OnWidgetAdded(func(w gi.Widget) {
		pfrom := w.PathFrom(sv)
		switch {
		case pfrom == "struct-grid":
			w.Style(func(s *styles.Style) {
				s.Display = styles.Grid
				s.Grow.Set(0, 0)
				if sv.SizeClass() == gi.SizeCompact {
					s.Columns = 1
				} else {
					s.Columns = 2
				}
			})
		}
	})
}

// SetStruct sets the source struct that we are viewing -- rebuilds the
// children to represent this struct
func (sv *StructView) SetStruct(st any) *StructView {
	if sv.Struct != st {
		sv.Changed = false
	}
	sv.Struct = st
	sv.Update()
	sv.SetNeedsLayout(true)
	return sv
}

// UpdateFields updates each of the value-view widgets for the fields --
// called by the ViewSig update
func (sv *StructView) UpdateFields() {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
		// we do not update focused elements to prevent panics
		if wb := vv.AsWidgetBase(); wb != nil {
			if wb.StateIs(states.Focused) {
				continue
			}
		}
		vv.UpdateWidget()
	}
	sv.UpdateEndRender(updt)
}

// UpdateField updates the value-view widget for the named field
func (sv *StructView) UpdateField(field string) {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
		if vv.Name() == field {
			vv.UpdateWidget()
			break
		}
	}
	sv.UpdateEndRender(updt)
}

// ConfigWidget configures the view
func (sv *StructView) ConfigWidget() {
	if ks, ok := sv.Struct.(ki.Ki); ok {
		if ks == nil || ks.This() == nil {
			return
		}
	}
	if sv.HasChildren() {
		sv.ConfigStructGrid()
		return
	}
	updt := sv.UpdateStart()
	gi.NewFrame(sv, "struct-grid")
	sv.ConfigStructGrid()
	sv.UpdateEndLayout(updt)
}

// IsConfiged returns true if the widget is fully configured
func (sv *StructView) IsConfiged() bool {
	return len(sv.Kids) != 0
}

// StructGrid returns the grid layout widget, which contains all the fields and values
func (sv *StructView) StructGrid() *gi.Frame {
	return sv.ChildByName("struct-grid", 2).(*gi.Frame)
}

// FieldTags returns the integrated tags for this field
func (sv *StructView) FieldTags(fld reflect.StructField) reflect.StructTag {
	if sv.TypeFieldTags == nil {
		return fld.Tag
	}
	ft, has := sv.TypeFieldTags[fld.Name]
	if !has {
		return fld.Tag
	}
	return fld.Tag + " " + reflect.StructTag(ft)
}

// ConfigStructGrid configures the StructGrid for the current struct.
// returns true if any fields changed.
func (sv *StructView) ConfigStructGrid() bool {
	if laser.AnyIsNil(sv.Struct) {
		return false
	}
	sc := true
	if len(NoSentenceCaseFor) > 0 {
		sc = !NoSentenceCaseForType(gti.TypeNameObj(sv.Struct))
	}
	sg := sv.StructGrid()
	// note: widget re-use does not work due to all the closures
	sg.DeleteChildren()
	config := ki.Config{}
	dupeFields := map[string]bool{}
	sv.FieldViews = make([]Value, 0)

	shouldShow := func(field reflect.StructField, stru any) bool {
		ftags := sv.FieldTags(field)
		vwtag := ftags.Get("view")
		if vwtag == "-" {
			return false
		}
		if ss, ok := stru.(gi.ShouldShower); ok {
			sv.IsShouldShower = true
			if !ss.ShouldShow(field.Name) {
				return false
			}
		}
		return true
	}

	laser.FlatFieldsValueFuncIf(sv.Struct,
		func(stru any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			return shouldShow(field, sv.Struct)
		},
		func(fval any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			// todo: check tags, skip various etc
			ftags := sv.FieldTags(field)
			vwtag := ftags.Get("view")
			if !shouldShow(field, sv.Struct) {
				return true
			}
			if vwtag == "add-fields" && field.Type.Kind() == reflect.Struct {
				fvalp := fieldVal.Addr().Interface()
				laser.FlatFieldsValueFuncIf(fvalp,
					func(stru any, typ reflect.Type, sfield reflect.StructField, fieldVal reflect.Value) bool {
						return shouldShow(sfield, fvalp)
					},
					func(sfval any, styp reflect.Type, sfield reflect.StructField, sfieldVal reflect.Value) bool {
						if !shouldShow(sfield, fvalp) {
							return true
						}
						svv := FieldToValue(fvalp, sfield.Name, sfval)
						if svv == nil { // shouldn't happen
							return true
						}
						svvp := sfieldVal.Addr()
						svv.SetStructValue(svvp, fvalp, &sfield, sv.TmpSave, sv.ViewPath)

						svtyp := svv.WidgetType()
						// todo: other things with view tag..
						fnm := field.Name + " • " + sfield.Name
						if _, exists := dupeFields[fnm]; exists {
							slog.Error("StructView: duplicate field name:", "name:", fnm)
						} else {
							dupeFields[fnm] = true
						}
						if sc {
							svv.SetLabel(strcase.ToSentence(fnm))
						} else {
							svv.SetLabel(fnm)
						}
						labnm := fmt.Sprintf("label-%v", fnm)
						valnm := fmt.Sprintf("value-%v", fnm)
						config.Add(gi.LabelType, labnm)
						config.Add(svtyp, valnm) // todo: extend to diff types using interface..
						sv.FieldViews = append(sv.FieldViews, svv)
						return true
					})
				return true
			}
			vv := FieldToValue(sv.Struct, field.Name, fval)
			if vv == nil { // shouldn't happen
				return true
			}
			if _, exists := dupeFields[field.Name]; exists {
				slog.Error("StructView: duplicate field name:", "name:", field.Name)
			} else {
				dupeFields[field.Name] = true
			}
			vvp := fieldVal.Addr()
			vv.SetStructValue(vvp, sv.Struct, &field, sv.TmpSave, sv.ViewPath)
			vtyp := vv.WidgetType()
			// todo: other things with view tag..
			labnm := fmt.Sprintf("label-%v", field.Name)
			valnm := fmt.Sprintf("value-%v", field.Name)
			config.Add(gi.LabelType, labnm)
			config.Add(vtyp, valnm) // todo: extend to diff types using interface..
			sv.FieldViews = append(sv.FieldViews, vv)
			return true
		})
	mods, updt := sg.ConfigChildren(config) // fields could be non-unique with labels..
	if !mods {
		updt = sg.UpdateStart()
	}
	for i, vv := range sv.FieldViews {
		lbl := sg.Child(i * 2).(*gi.Label)
		lbl.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
		lbl.Tooltip = vv.Doc()
		vvb := vv.AsValueBase()
		vvb.ViewPath = sv.ViewPath
		w, wb := gi.AsWidget(sg.Child((i * 2) + 1))
		hasDef, readOnlyTag := StructViewFieldTags(vv, lbl, w, sv.IsReadOnly())
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
		if w.KiType() != vv.WidgetType() {
			slog.Error("StructView: Widget Type is not the proper type.  This usually means there are duplicate field names (including across embedded types", "field:", lbl.Text, "is:", w.KiType().Name, "should be:", vv.WidgetType().Name)
			break
		}
		if wb.Prop("configured") == nil {
			wb.SetProp("configured", true)
			vv.ConfigWidget(w)
			vvb.AsWidgetBase().OnInput(func(e events.Event) {
				if tag, _ := vv.Tag("immediate"); tag == "+" {
					wb.SendChange(e)
					sv.SendChange(e)
				}
				sv.Send(events.Input, e)
			})
		} else {
			vvb.Widget = w
			vv.UpdateWidget()
		}
		if !sv.IsReadOnly() && !readOnlyTag {
			vvb.OnChange(func(e events.Event) {
				sv.UpdateFieldAction()
				// note: updating vv here is redundant -- relevant field will have already updated
				sv.Changed = true
				if !laser.KindIsBasic(laser.NonPtrValue(vvb.Value).Kind()) {
					if updtr, ok := sv.Struct.(gi.Updater); ok {
						updtr.Update()
					}
				}
				sv.SendChange(e)
			})
		}
	}
	sg.UpdateEnd(updt)
	return updt
}

func (sv *StructView) UpdateFieldAction() {
	if !sv.IsConfiged() {
		return
	}
	if sv.IsShouldShower {
		sv.Update()
		sv.SetNeedsLayout(true)
	}
}

/////////////////////////////////////////////////////////////////////////
//  Tag parsing

// StructViewFieldTags processes the tags for a field in a struct view, setting
// the properties on the label or widget appropriately
// returns true if there were any "default" default tags -- if so, needs updating
func StructViewFieldTags(vv Value, lbl *gi.Label, w gi.Widget, isReadOnly bool) (hasDef, readOnlyTag bool) {
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
		vf, err := laser.ToFloat(valPtr)
		if err != nil {
			slog.Error("giv.StructFieldIsDef: error parsing struct field numerical range def tag", "type", laser.NonPtrType(reflect.TypeOf(valPtr)), "def", defs, "err", err)
			return true, defStr
		}
		return lo <= vf && vf <= hi, defStr
	}
	v := laser.NonPtrValue(reflect.ValueOf(valPtr))
	dtags := strings.Split(defs, ",")
	if strings.ContainsAny(defs, "{[") { // complex type, no split on commas
		dtags = []string{defs}
	}
	for _, def := range dtags {
		def = laser.FormatDefault(def)
		if def == "" {
			if v.IsZero() {
				return true, defStr
			}
			continue
		}
		dv := reflect.New(v.Type())
		err := laser.SetRobust(dv.Interface(), def)
		if err != nil {
			slog.Error("giv.StructFieldIsDef: error parsing struct field def tag", "type", v.Type(), "def", def, "err", err)
			return true, defStr
		}
		if reflect.DeepEqual(v.Interface(), dv.Elem().Interface()) {
			return true, defStr
		}
	}
	return false, defStr
}

// StructFieldVals represents field values in a struct, at multiple
// levels of depth potentially (represented by the Path field)
// used for StructNonDefFields for example.
type StructFieldVals struct {

	// path of field.field parent fields to this field
	Path string

	// type information for field
	Field reflect.StructField

	// value of field (as a pointer)
	Val reflect.Value

	// def tag information for default values
	Defs string
}

// StructNonDefFields processses "default" tag for default value(s)
// of fields in given struct and starting path, and returns all
// fields not at their default values.
// See also StructNoDefFieldsStr for a string representation of this information.
// Uses laser.FlatFieldsValueFunc to get all embedded fields.
// Uses a recursive strategy -- any fields that are themselves structs are
// expanded, and the field name represented by dots path separators.
func StructNonDefFields(structPtr any, path string) []StructFieldVals {
	var flds []StructFieldVals
	laser.FlatFieldsValueFunc(structPtr, func(fval any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
		vvp := fieldVal.Addr()
		dtag, got := field.Tag.Lookup("default")
		if field.Type.Kind() == reflect.Struct && (!got || dtag == "") {
			spath := path
			if path != "" {
				spath += "."
			}
			spath += field.Name
			subs := StructNonDefFields(vvp.Interface(), spath)
			if len(subs) > 0 {
				flds = append(flds, subs...)
			}
			return true
		}
		if !got {
			return true
		}
		def, defStr := StructFieldIsDef(dtag, vvp.Interface(), field.Type.Kind())
		if def {
			return true
		}
		flds = append(flds, StructFieldVals{Path: path, Field: field, Val: vvp, Defs: defStr})
		return true
	})
	return flds
}

// StructNonDefFieldsStr processses "default" tag for default value(s) of fields in
// given struct, and returns a string of all those not at their default values,
// in format: Path.Field: val // default value(s)
// Uses a recursive strategy -- any fields that are themselves structs are
// expanded, and the field name represented by dots path separators.
func StructNonDefFieldsStr(structPtr any, path string) string {
	flds := StructNonDefFields(structPtr, path)
	if len(flds) == 0 {
		return ""
	}
	str := ""
	for _, fld := range flds {
		pth := fld.Path
		fnm := fld.Field.Name
		val := laser.ToStringPrec(fld.Val.Interface(), 6)
		dfs := fld.Defs
		if len(pth) > 0 {
			fnm = pth + "." + fnm
		}
		str += fmt.Sprintf("%s: %s // %s<br>\n", fnm, val, dfs)
	}
	return str
}

// StructViewDialog opens a dialog (optionally in a new, separate window)
// for viewing / editing the given struct object, in the context of given ctx widget
func StructViewDialog(ctx gi.Widget, stru any, title string, newWindow bool) {
	d := gi.NewBody().AddTitle(title)
	NewStructView(d).SetStruct(stru)
	if tb, ok := stru.(gi.Toolbarer); ok {
		d.AddAppBar(tb.ConfigToolbar)
	}
	ds := d.NewFullDialog(ctx)
	if newWindow {
		ds.SetNewWindow(true)
	}
	ds.Run()
}
