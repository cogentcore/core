// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/ast"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/glop/bools"
	"goki.dev/ki/v2"
)

// StructView represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame.
// Automatically has a toolbar with Struct ToolBar props if defined
// set prop toolbar = false to turn off
type StructView struct {
	gi.Frame

	// the struct that we are a view onto
	Struct any `desc:"the struct that we are a view onto"`

	// ValueView for the struct itself, if this was created within value view framework -- otherwise nil
	StructValView ValueView `desc:"ValueView for the struct itself, if this was created within value view framework -- otherwise nil"`

	// has the value of any field changed?  updated by the ViewSig signals from fields
	Changed bool `desc:"has the value of any field changed?  updated by the ViewSig signals from fields"`

	// ValueView for a field marked with changeflag struct tag, which must be a bool type, which is updated when changes are registered in field values.
	ChangeFlag *reflect.Value `json:"-" xml:"-" desc:"ValueView for a field marked with changeflag struct tag, which must be a bool type, which is updated when changes are registered in field values."`

	// ValueView representations of the fields
	FieldViews []ValueView `json:"-" xml:"-" desc:"ValueView representations of the fields"`

	// whether to show the toolbar or not
	ShowToolBar bool `desc:"whether to show the toolbar or not"`

	// value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent
	TmpSave ValueView `json:"-" xml:"-" desc:"value view that needs to have SaveTmp called on it whenever a change is made to one of the underlying values -- pass this down to any sub-views created from a parent"`

	// signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update
	ViewSig ki.Signal `json:"-" xml:"-" desc:"signal for valueview -- only one signal sent when a value has been set -- all related value views interconnect with each other to update when others update"`

	// a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows
	ViewPath string `desc:"a record of parent View names that have led up to this view -- displayed as extra contextual information in view dialog windows"`

	// the struct that we successfully set a toolbar for
	ToolbarStru any `desc:"the struct that we successfully set a toolbar for"`

	// if true, some fields have default values -- update labels when values change
	HasDefs bool `json:"-" xml:"-" inactive:"+" desc:"if true, some fields have default values -- update labels when values change"`

	// if true, some fields have viewif conditional view tags -- update after..
	HasViewIfs bool `json:"-" xml:"-" inactive:"+" desc:"if true, some fields have viewif conditional view tags -- update after.."`

	// extra tags by field name -- from type properties
	TypeFieldTags map[string]string `json:"-" xml:"-" inactive:"+" desc:"extra tags by field name -- from type properties"`
}

func (sv *StructView) OnInit() {
	sv.ShowToolBar = true
	sv.Lay = gi.LayoutVert
	sv.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		sv.Spacing = gi.StdDialogVSpaceUnits
		s.SetStretchMax()
	})
}

func (sv *StructView) OnChildAdded(child ki.Ki) {
	if w := gi.AsWidget(child); w != nil {
		switch w.Name() {
		case "toolbar":
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.SetStretchMaxWidth()
			})
		case "struct-grid":
			sg := child.(*gi.Frame)
			sg.Lay = gi.LayoutGrid
			sg.Stripes = gi.RowStripes
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				// setting a pref here is key for giving it a scrollbar in larger context
				s.SetMinPrefHeight(units.Em(1.5))
				s.SetMinPrefWidth(units.Em(10))
				s.SetStretchMax()                // for this to work, ALL layers above need it too
				s.Overflow = gist.OverflowScroll // this still gives it true size during PrefSize
				s.Columns = 2
			})
		}
		if w.Parent().Name() == "struct-grid" {
			w.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
				s.AlignH = gist.AlignLeft
			})
		}
	}
}

func (sv *StructView) Disconnect() {
	sv.Frame.Disconnect()
	sv.ViewSig.DisconnectAll()
}

var StructViewProps = ki.Props{
	ki.EnumTypeFlag: gi.TypeNodeFlags,
}

// SetStruct sets the source struct that we are viewing -- rebuilds the
// children to represent this struct
func (sv *StructView) SetStruct(st any) {
	updt := false
	if sv.Struct != st {
		sv.Changed = false
		updt = sv.UpdateStart()
		sv.SetFullReRender()
		if sv.Struct != nil {
			if k, ok := sv.Struct.(ki.Ki); ok {
				k.NodeSignal().Disconnect(sv.This())
			}
		}
		sv.Struct = st
		// tp := kit.Types.Properties(kit.NonPtrType(reflect.TypeOf(sv.Struct)), false)
		if tp != nil {
			if sfp, has := ki.SubTypeProps(*tp, "StructViewFields"); has {
				sv.TypeFieldTags = make(map[string]string)
				for k, v := range sfp {
					vs := laser.ToString(v)
					sv.TypeFieldTags[k] = vs
				}
			}
		}
		if k, ok := st.(ki.Ki); ok {
			k.NodeSignal().Connect(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
				// todo: check for delete??
				svv, _ := recv.Embed(TypeStructView).(*StructView)
				svv.UpdateFields()
				svv.ViewSig.Emit(svv.This(), 0, nil)
			})
		}
	}
	sv.Config()
	sv.UpdateEnd(updt)
}

// UpdateFields updates each of the value-view widgets for the fields --
// called by the ViewSig update
func (sv *StructView) UpdateFields() {
	updt := sv.UpdateStart()
	for _, vv := range sv.FieldViews {
		vv.UpdateWidget()
	}
	sv.UpdateEnd(updt)
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
	sv.UpdateEnd(updt)
}

// Config configures the view
func (sv *StructView) ConfigWidget(vp *Viewport) {
	if ks, ok := sv.Struct.(ki.Ki); ok {
		if ks.IsDeleted() || ks.IsDestroyed() {
			return
		}
	}
	config := ki.TypeAndNameList{}
	config.Add(gi.TypeToolBar, "toolbar")
	config.Add(gi.TypeFrame, "struct-grid")
	mods, updt := sv.ConfigChildren(config)
	sv.ConfigStructGrid()
	sv.ConfigToolbar()
	if mods {
		sv.UpdateEnd(updt)
	}
}

// IsConfiged returns true if the widget is fully configured
func (sv *StructView) IsConfiged() bool {
	if len(sv.Kids) == 0 {
		return false
	}
	return true
}

// StructGrid returns the grid layout widget, which contains all the fields and values
func (sv *StructView) StructGrid() *gi.Frame {
	return sv.ChildByName("struct-grid", 2).(*gi.Frame)
}

// ToolBar returns the toolbar widget
func (sv *StructView) ToolBar() *gi.ToolBar {
	return sv.ChildByName("toolbar", 1).(*gi.ToolBar)
}

// ConfigToolbar adds a toolbar based on the methview ToolBarView function, if
// one has been defined for this struct type through its registered type
// properties.
func (sv *StructView) ConfigToolbar() {
	if laser.IfaceIsNil(sv.Struct) {
		return
	}
	if sv.ToolbarStru == sv.Struct {
		return
	}
	if !sv.ShowToolBar {
		sv.ToolbarStru = sv.Struct
		return
	}
	tb := sv.ToolBar()
	svtp := laser.NonPtrType(reflect.TypeOf(sv.Struct))
	ttip := "update this StructView (not any other views that might be present) to show current state of this struct of type: " + svtp.String()
	if len(*tb.Children()) == 0 {
		tb.AddAction(gi.ActOpts{Label: "UpdtView", Icon: gicons.Refresh, Tooltip: ttip},
			sv.This(), func(recv, send ki.Ki, sig int64, data any) {
				svv := recv.Embed(TypeStructView).(*StructView)
				svv.UpdateFields()
			})
	} else {
		act := tb.Child(0).(*gi.Action)
		act.Tooltip = ttip
	}
	ndef := 1 // number of default actions
	sz := len(*tb.Children())
	if sz > ndef {
		for i := sz - 1; i >= ndef; i-- {
			tb.DeleteChildAtIndex(i, ki.DestroyKids)
		}
	}
	if HasToolBarView(sv.Struct) {
		ToolBarView(sv.Struct, sv.Viewport, tb)
		tb.SetFullReRender()
	}
	sv.ToolbarStru = sv.Struct
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

// ConfigStructGrid configures the StructGrid for the current struct
func (sv *StructView) ConfigStructGrid() {
	if laser.IfaceIsNil(sv.Struct) {
		return
	}
	sg := sv.StructGrid()
	config := ki.TypeAndNameList{}
	// always start fresh!
	sv.FieldViews = make([]ValueView, 0)
	laser.FlatFieldsValueFunc(sv.Struct, func(fval any, typ reflect.Type, field reflect.StructField, fieldVal reflect.Value) bool {
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
			laser.FlatFieldsValueFunc(fvalp, func(sfval any, styp reflect.Type, sfield reflect.StructField, sfieldVal reflect.Value) bool {
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
				svv := FieldToValueView(fvalp, sfield.Name, sfval)
				if svv == nil { // shouldn't happen
					return true
				}
				svvp := sfieldVal.Addr()
				svv.SetStructValue(svvp, fvalp, &sfield, sv.TmpSave, sv.ViewPath)

				svtyp := svv.WidgetType()
				// todo: other things with view tag..
				fnm := field.Name + "." + sfield.Name
				svv.SetTag("label", fnm)
				labnm := fmt.Sprintf("label-%v", fnm)
				valnm := fmt.Sprintf("value-%v", fnm)
				config.Add(gi.TypeLabel, labnm)
				config.Add(svtyp, valnm) // todo: extend to diff types using interface..
				sv.FieldViews = append(sv.FieldViews, svv)
				return true
			})
			return true
		}
		vv := FieldToValueView(sv.Struct, field.Name, fval)
		if vv == nil { // shouldn't happen
			return true
		}
		vvp := fieldVal.Addr()
		vv.SetStructValue(vvp, sv.Struct, &field, sv.TmpSave, sv.ViewPath)
		vtyp := vv.WidgetType()
		// todo: other things with view tag..
		labnm := fmt.Sprintf("label-%v", field.Name)
		valnm := fmt.Sprintf("value-%v", field.Name)
		config.Add(gi.TypeLabel, labnm)
		config.Add(vtyp, valnm) // todo: extend to diff types using interface..
		sv.FieldViews = append(sv.FieldViews, vv)
		return true
	})
	mods, updt := sg.ConfigChildren(config) // fields could be non-unique with labels..
	if mods {
		sg.SetFullReRender()
	} else {
		updt = sg.UpdateStart()
	}
	sv.HasDefs = false
	for i, vv := range sv.FieldViews {
		lbl := sg.Child(i * 2).(*gi.Label)
		vvb := vv.AsValueViewBase()
		vvb.ViewPath = sv.ViewPath
		lbl.Redrawable = true
		widg := sg.Child((i * 2) + 1).(gi.Node2D)
		hasDef, inactTag := StructViewFieldTags(vv, lbl, widg, sv.IsDisabled())
		if hasDef {
			sv.HasDefs = true
		}
		vv.ConfigWidget(widg)
		if !sv.IsDisabled() && !inactTag {
			vvb.ViewSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data any) {
				svv := recv.Embed(TypeStructView).(*StructView)
				svv.UpdateFieldAction()
				// note: updating vv here is redundant -- relevant field will have already updated
				svv.Changed = true
				if svv.ChangeFlag != nil {
					svv.ChangeFlag.SetBool(true)
				}
				vvv := send.(ValueView).AsValueViewBase()
				if !laser.KindIsBasic(laser.NonPtrValue(vvv.Value).Kind()) {
					if updtr, ok := svv.Struct.(gi.Updater); ok {
						// fmt.Printf("updating: %v kind: %v\n", updtr, vvv.Value.Kind())
						updtr.Update()
					}
				}
				tb := svv.ToolBar()
				if tb != nil {
					tb.UpdateActions()
				}
				svv.ViewSig.Emit(svv.This(), 0, nil)
				// vvv, _ := send.Embed(TypeValueViewBase).(*ValueViewBase)
				// fmt.Printf("sview got edit from vv %v field: %v\n", vvv.Nm, vvv.Field.Name)
			})
		}
	}
	sg.UpdateEnd(updt)
}

func (sv *StructView) SetStyle() {
	mvp := sv.Vp
	if mvp != nil && mvp.IsDoingFullRender() {
		sv.Config()
	}
	sv.Frame.SetStyle()
}

func (sv *StructView) UpdateFieldAction() {
	if !sv.IsConfiged() {
		return
	}
	if sv.HasViewIfs {
		sv.Config()
	} else if sv.HasDefs {
		sg := sv.StructGrid()
		updt := sg.UpdateStart()
		for i, vv := range sv.FieldViews {
			lbl := sg.Child(i * 2).(*gi.Label)
			StructViewFieldDefTag(vv, lbl)
		}
		sg.UpdateEnd(updt)
	}
}

func (sv *StructView) Render(vp *Viewport) {
	if sv.IsConfiged() {
		sv.ToolBar().UpdateActions()
	}
	if win := sv.ParentWindow(); win != nil {
		if !win.IsResizing() {
			win.MainMenuUpdateActives()
		}
	}
	sv.Frame.Render()
}

/////////////////////////////////////////////////////////////////////////
//  Tag parsing

// StructViewFieldTags processes the tags for a field in a struct view, setting
// the properties on the label or widget appropriately
// returns true if there were any "def" default tags -- if so, needs updating
func StructViewFieldTags(vv ValueView, lbl *gi.Label, widg gi.Node2D, isInact bool) (hasDef, inactTag bool) {
	vvb := vv.AsValueViewBase()
	if lbltag, has := vv.Tag("label"); has {
		lbl.Text = lbltag
	} else {
		lbl.Text = vvb.Field.Name
	}
	if _, has := vv.Tag("inactive"); has {
		inactTag = true
		widg.AsNode2D().SetDisabled()
	} else {
		if isInact {
			widg.AsNode2D().SetDisabled()
			vv.SetTag("inactive", "true")
		}
	}
	defStr := ""
	hasDef, _, defStr = StructViewFieldDefTag(vv, lbl)
	if ttip, has := vv.Tag("desc"); has {
		lbl.Tooltip = defStr + ttip
	}
	return
}

// StructViewFieldDefTag processes the "def" tag for default values -- can be
// called multiple times for updating as values change.
// returns true if value is default, and string to add to tooltip for default vals
func StructViewFieldDefTag(vv ValueView, lbl *gi.Label) (hasDef bool, isDef bool, defStr string) {
	if dtag, has := vv.Tag("def"); has {
		hasDef = true
		isDef, defStr = StructFieldIsDef(dtag, vv.Val().Interface(), laser.NonPtrValue(vv.Val()).Kind())
		if isDef {
			lbl.CurBackgroundColor = gi.Prefs.Colors.Background
		} else {
			lbl.CurBackgroundColor = gi.Prefs.Colors.Highlight
		}
		return
	}
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
	Path string `desc:"path of field.field parent fields to this field"`

	// type information for field
	Field reflect.StructField `desc:"type information for field"`

	// value of field (as a pointer)
	Val reflect.Value `desc:"value of field (as a pointer)"`

	// def tag information for default values
	Defs string `desc:"def tag information for default values"`
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
