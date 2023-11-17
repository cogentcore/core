// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/ast"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/glop/bools"
	"goki.dev/glop/sentencecase"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
	"goki.dev/laser"
	"goki.dev/mat32/v2"
)

// StructView represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame.
type StructView struct {
	gi.Frame

	// the struct that we are a view onto
	Struct any `set:"-"`

	// Value for the struct itself, if this was created within value view framework -- otherwise nil
	StructValView Value

	// has the value of any field changed?  updated by the ViewSig signals from fields
	Changed bool `set:"-"`

	// Value for a field marked with changeflag struct tag, which must be a bool type, which is updated when changes are registered in field values.
	ChangeFlag *reflect.Value `json:"-" xml:"-"`

	// Value representations of the fields
	FieldViews []Value `json:"-" xml:"-"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave Value `json:"-" xml:"-"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string

	// if true, some fields have default values -- update labels when values change
	HasDefs bool `json:"-" xml:"-" edit:"-"`

	// if true, some fields have viewif conditional view tags -- update after..
	HasViewIfs bool `json:"-" xml:"-" edit:"-"`

	// extra tags by field name -- from type properties
	TypeFieldTags map[string]string `json:"-" xml:"-" edit:"-"`
}

func (sv *StructView) OnInit() {
	sv.Style(func(s *styles.Style) {
		s.SetMainAxis(mat32.Y)
		s.Grow.Set(1, 1)
	})
	sv.OnWidgetAdded(func(w gi.Widget) {
		pfrom := w.PathFrom(sv)
		switch {
		case pfrom == "struct-grid":
			sg := w.(*gi.Frame)
			sg.Stripes = gi.RowStripes
			w.Style(func(s *styles.Style) {
				s.Display = styles.DisplayGrid
				s.Columns = 2
				s.Overflow.Set(styles.OverflowAuto)
				s.Grow.Set(1, 1)
				// s.Min.X.Em(20)
				// s.Min.Y.Em(10)
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
	return sv
}

// UpdateFields updates each of the value-view widgets for the fields --
// called by the ViewSig update
func (sv *StructView) UpdateFields() {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
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

// Config configures the view
func (sv *StructView) ConfigWidget(sc *gi.Scene) {
	if ks, ok := sv.Struct.(ki.Ki); ok {
		if ks == nil || ks.This() == nil || ks.Is(ki.Deleted) {
			return
		}
	}
	if sv.HasChildren() {
		sv.ConfigStructGrid(sc)
		return
	}
	updt := sv.UpdateStart()
	gi.NewFrame(sv, "struct-grid")
	sv.ConfigStructGrid(sc)
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
func (sv *StructView) ConfigStructGrid(sc *gi.Scene) bool {
	if laser.AnyIsNil(sv.Struct) {
		return false
	}
	sg := sv.StructGrid()
	// note: widget re-use does not work due to all the closures
	sg.DeleteChildren(ki.DestroyKids)
	config := ki.Config{}
	dupeFields := map[string]bool{}
	sv.FieldViews = make([]Value, 0)
	laser.FlatFieldsValueFuncIf(sv.Struct,
		func(stru any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			ftags := sv.FieldTags(field)
			vwtag := ftags.Get("view")
			if vwtag == "-" {
				return false
			}
			viewif := field.Tag.Get("viewif")
			if viewif != "" {
				sv.HasViewIfs = true
				if !StructViewIf(viewif, field, sv.Struct) {
					return false
				}
			}
			return true
		},
		func(fval any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
			// todo: check tags, skip various etc
			ftags := sv.FieldTags(field)
			_, got := ftags.Lookup("changeflag")
			if got {
				if field.Type.Kind() == reflect.Bool {
					sv.ChangeFlag = &fieldVal
				}
			}
			vwtag := ftags.Get("view")
			if vwtag == "-" {
				return true
			}
			viewif := field.Tag.Get("viewif")
			if viewif != "" {
				sv.HasViewIfs = true
				if !StructViewIf(viewif, field, sv.Struct) {
					return true
				}
			}
			if vwtag == "add-fields" && field.Type.Kind() == reflect.Struct {
				fvalp := fieldVal.Addr().Interface()
				laser.FlatFieldsValueFuncIf(fvalp,
					func(stru any, typ reflect.Type, sfield reflect.StructField, fieldVal reflect.Value) bool {
						svwtag := sfield.Tag.Get("view")
						if svwtag == "-" {
							return false
						}
						viewif := sfield.Tag.Get("viewif")
						if viewif != "" {
							sv.HasViewIfs = true
							if !StructViewIf(viewif, sfield, fvalp) {
								return false
							}
						}
						return true
					},
					func(sfval any, styp reflect.Type, sfield reflect.StructField, sfieldVal reflect.Value) bool {
						svwtag := sfield.Tag.Get("view")
						if svwtag == "-" {
							return true
						}
						viewif := sfield.Tag.Get("viewif")
						if viewif != "" {
							sv.HasViewIfs = true
							if !StructViewIf(viewif, sfield, fvalp) {
								return true
							}
						}
						svv := FieldToValue(fvalp, sfield.Name, sfval)
						if svv == nil { // shouldn't happen
							return true
						}
						svvp := sfieldVal.Addr()
						svv.SetStructValue(svvp, fvalp, &sfield, sv.TmpSave, sv.ViewPath)

						svtyp := svv.WidgetType()
						// todo: other things with view tag..
						fnm := field.Name + sfield.Name
						if _, exists := dupeFields[fnm]; exists {
							slog.Error("StructView: duplicate field name:", "name:", fnm)
						} else {
							dupeFields[fnm] = true
						}
						// TODO(kai): how should we format this label?
						svv.SetLabel(sentencecase.Of(fnm))
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
	sv.HasDefs = false
	for i, vv := range sv.FieldViews {
		lbl := sg.Child(i * 2).(*gi.Label)
		lbl.Style(func(s *styles.Style) {
			s.SetTextWrap(false)
		})
		vvb := vv.AsValueBase()
		vvb.ViewPath = sv.ViewPath
		w, wb := gi.AsWidget(sg.Child((i * 2) + 1))
		hasDef, readOnlyTag := StructViewFieldTags(vv, lbl, w, sv.IsReadOnly())
		if hasDef {
			sv.HasDefs = true
		}
		if w.KiType() != vv.WidgetType() {
			slog.Error("StructView: Widget Type is not the proper type.  This usually means there are duplicate field names (including across embedded types", "field:", lbl.Text, "is:", w.KiType().Name, "should be:", vv.WidgetType().Name)
			break
		}
		if wb.Class == "" {
			wb.Class = "configed"
			vv.ConfigWidget(w, sc)
		} else {
			vvb.Widget = w
			vv.UpdateWidget()
		}
		if !sv.IsReadOnly() && !readOnlyTag {
			vvb.OnChange(func(e events.Event) {
				sv.UpdateFieldAction()
				// note: updating vv here is redundant -- relevant field will have already updated
				sv.Changed = true
				if sv.ChangeFlag != nil {
					sv.ChangeFlag.SetBool(true)
				}
				if !laser.KindIsBasic(laser.NonPtrValue(vvb.Value).Kind()) {
					if updtr, ok := sv.Struct.(gi.Updater); ok {
						// fmt.Printf("updating: %v kind: %v\n", updtr, vvv.Value.Kind())
						updtr.Update()
					}
				}
				sv.SendChange(e)
				// vvv, _ := send.Embed(TypeValueBase).(*ValueBase)
				// fmt.Printf("sview got edit from vv %v field: %v\n", vvv.Nm, vvv.Field.Name)
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
	if sv.HasViewIfs {
		sv.Update()
	} else if sv.HasDefs {
		sg := sv.StructGrid()
		updt := sg.UpdateStart()
		for i, vv := range sv.FieldViews {
			lbl := sg.Child(i * 2).(*gi.Label)
			StructViewFieldDefTag(vv, lbl)
		}
		sg.UpdateEndRender(updt)
	}
}

/////////////////////////////////////////////////////////////////////////
//  Tag parsing

// StructViewFieldTags processes the tags for a field in a struct view, setting
// the properties on the label or widget appropriately
// returns true if there were any "def" default tags -- if so, needs updating
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
	defStr := ""
	hasDef, _, defStr = StructViewFieldDefTag(vv, lbl)
	lbl.Tooltip = defStr + vv.Doc()
	return
}

// StructViewFieldDefTag processes the "def" tag for default values -- can be
// called multiple times for updating as values change.
// returns true if value is default, and string to add to tooltip for default vals
func StructViewFieldDefTag(vv Value, lbl *gi.Label) (hasDef bool, isDef bool, defStr string) {
	// todo
	// if dtag, has := vv.Tag("def"); has {
	// 	hasDef = true
	// 	isDef, defStr = StructFieldIsDef(dtag, vv.Val().Interface(), laser.NonPtrValue(vv.Val()).Kind())
	// 	if isDef {
	// 		lbl.CurBackgroundColor = gi.Prefs.Colors.Background
	// 	} else {
	// 		lbl.CurBackgroundColor = gi.Prefs.Colors.Highlight
	// 	}
	// 	return
	// }
	return
}

// StructFieldIsDef processses "def" tag for default value(s) of field
// defs = default values as strings as either comma-separated list of valid values
// or low:high value range (only for int or float numeric types)
// valPtr = pointer to value
// returns true if value is default, and string to add to tooltip for default values.
// Uses JSON format for composite types (struct, slice, map), replacing " with '
// so it is easier to use in def tag.
func StructFieldIsDef(defs string, valPtr any, kind reflect.Kind) (bool, string) {
	defStr := "[Def: " + defs + "] "
	def := false
	switch {
	case kind == reflect.Struct || kind == reflect.Slice || kind == reflect.Map:
		jb, _ := json.Marshal(valPtr)
		jstr := string(jb)
		// fmt.Println(jstr, defs)
		if defs == jstr {
			def = true
		} else {
			jstr = strings.ReplaceAll(jstr, `"`, `'`)
			if defs == jstr {
				def = true
			}
		}
	case strings.Contains(defs, ":"):
		dtags := strings.Split(defs, ":")
		lo, _ := strconv.ParseFloat(dtags[0], 64)
		hi, _ := strconv.ParseFloat(dtags[1], 64)
		switch fv := valPtr.(type) {
		case *float32:
			if lo <= float64(*fv) && float64(*fv) <= hi {
				def = true
			}
		case *float64:
			if lo <= *fv && *fv <= hi {
				def = true
			}
		case *int32:
			if lo <= float64(*fv) && float64(*fv) <= hi {
				def = true
			}
		case *int64:
			if lo <= float64(*fv) && float64(*fv) <= hi {
				def = true
			}
		case *int:
			if lo <= float64(*fv) && float64(*fv) <= hi {
				def = true
			}
		}
	default:
		val := laser.ToStringPrec(valPtr, 6)
		if strings.HasPrefix(val, "&") {
			val = val[1:]
		}
		dtags := strings.Split(defs, ",")
		for _, dv := range dtags {
			if dv == strings.TrimSpace(val) {
				def = true
				break
			}
		}
	}
	return def, defStr
}

type viewifPatcher struct{}

var (
	replaceEqualsRegexp = regexp.MustCompile(`([^\!\=\<\>])(=)([^\!\=\<\>])`)
	stringer            = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	booler              = reflect.TypeOf((*bools.Booler)(nil)).Elem()
	// slboolv             = reflect.TypeOf((*slbool.Bool)(nil)).Elem()
)

func (p *viewifPatcher) Visit(node *ast.Node) {
	switch x := (*node).(type) {
	case *ast.IdentifierNode:
		lt := x.Type()
		if lt == nil {
			return
		}
		if lt.Implements(booler) {
			ast.Patch(node, &ast.CallNode{
				Callee: &ast.MemberNode{
					Node:     *node,
					Property: &ast.StringNode{Value: "Bool"},
				},
			})
		}
	case *ast.BinaryNode:
		lid, lok := x.Left.(*ast.IdentifierNode)
		rid, rok := x.Right.(*ast.IdentifierNode)
		rars, rrok := x.Right.(*ast.ArrayNode)
		switch {
		case lok && rok && (x.Operator == "==" || x.Operator == "!="):
			lt := lid.Type()
			if lt == nil {
				return
			}
			switch {
			case lt.Implements(stringer):
				ast.Patch(node, &ast.BinaryNode{
					Operator: x.Operator,
					Left: &ast.CallNode{
						Callee: &ast.MemberNode{
							Node:     x.Left,
							Property: &ast.StringNode{Value: "String"},
						},
					},
					Right: &ast.StringNode{
						Value: rid.Value,
					},
				})
			}
		case lok && rrok && (x.Operator == "=="):
			var strs []ast.Node
			for _, on := range rars.Nodes {
				strs = append(strs, &ast.StringNode{Value: on.(*ast.IdentifierNode).Value})
			}
			ast.Patch(node, &ast.BinaryNode{
				Operator: "in",
				Left: &ast.CallNode{
					Callee: &ast.MemberNode{
						Node:     x.Left,
						Property: &ast.StringNode{Value: "String"},
					},
				},
				Right: &ast.ArrayNode{
					Nodes: strs,
				},
			})
		}
	}
}

// StructViewIf parses given `viewif:"expr"` expression and returns
// true if should be visible, false if not.
// Prints an error if the expression is not parsed properly
// or does not evaluate to a boolean.
func StructViewIf(viewif string, field reflect.StructField, stru any) bool {
	// replace = -> == without screwing up existing ==, !=, >=, <=
	viewif = replaceEqualsRegexp.ReplaceAllString(viewif, "$1==$3")

	program, err := expr.Compile(viewif, expr.Env(stru), expr.Patch(&viewifPatcher{}), expr.AllowUndefinedVariables())
	if err != nil {
		if gi.StructViewIfDebug {
			fmt.Printf("giv.StructView viewif tag on field %s: syntax error: `%s`: %s\n", field.Name, viewif, err)
		}
		return true
	}
	val, err := expr.Run(program, stru)
	if err != nil {
		if gi.StructViewIfDebug {
			fmt.Printf("giv.StructView viewif tag on field %s: run error: `%s`: %s\n", field.Name, viewif, err)
		}
		return true
	}
	if err != nil {
		if gi.StructViewIfDebug {
			fmt.Printf("giv.StructView viewif tag on field %s: syntax error: `%s`: %s\n", field.Name, viewif, err)
		}
		return true // visible by default
	}
	// fmt.Printf("fld: %s  viewif: %s  val: %t  %+v\n", field.Name, viewif, val, val)
	switch x := val.(type) {
	case bool:
		return x
	case *bool:
		return *x
	}
	if gi.StructViewIfDebug {
		fmt.Printf("giv.StructView viewif tag on field %s: didn't evaluate to a boolean: `%s`: type: %t val: %+v\n", field.Name, viewif, val, val)
	}
	return true
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

// StructNonDefFields processses "def" tag for default value(s)
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
		dtag, got := field.Tag.Lookup("def")
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

// StructNonDefFieldsStr processses "def" tag for default value(s) of fields in
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
