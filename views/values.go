// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/errors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// This file contains the standard [Value]s built into views.

// StringValue represents any value with a text field.
type StringValue struct {
	ValueBase[*core.TextField]
}

func (v *StringValue) Config() {
	if vtag, _ := v.Tag("view"); vtag == "password" {
		v.Widget.SetTypePassword()
	}
	if vl, ok := v.Value.Interface().(core.Validator); ok {
		v.Widget.SetValidator(vl.Validate)
	}
	if fv, ok := v.Owner.(core.FieldValidator); ok {
		v.Widget.SetValidator(func() error {
			return fv.ValidateField(v.Field.Name)
		})
	}

	v.Widget.OnFinal(events.Change, func(e events.Event) {
		if v.SetValue(v.Widget.Text()) {
			v.Update()
		}
	})
}

func (v *StringValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	if npv.Kind() == reflect.Interface && npv.IsZero() {
		v.Widget.SetText("None")
	} else {
		txt := laser.ToString(v.Value.Interface())
		v.Widget.SetText(txt)
	}
}

// BoolValue represents a bool value with a switch.
type BoolValue struct {
	ValueBase[*core.Switch]
}

func (v *BoolValue) Config() {
	v.Widget.OnChange(func(e events.Event) {
		v.SetValue(v.Widget.IsChecked())
	})
}

func (v *BoolValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	bv, err := laser.ToBool(npv.Interface())
	if errors.Log(err) == nil {
		v.Widget.SetChecked(bv)
	}
}

// NumberValue represents an integer or float value with a spinner.
type NumberValue struct {
	ValueBase[*core.Spinner]
}

func (v *NumberValue) Config() {
	vk := laser.NonPtrType(v.Value.Type()).Kind()
	if vk >= reflect.Int && vk <= reflect.Uintptr {
		v.Widget.SetStep(1).SetEnforceStep(true)
	}
	if vk >= reflect.Uint && vk <= reflect.Uintptr {
		v.Widget.SetMin(0)
	}
	if min, ok := v.Tag("min"); ok {
		minv, err := laser.ToFloat32(min)
		if errors.Log(err) == nil {
			v.Widget.SetMin(minv)
		}
	}
	if max, ok := v.Tag("max"); ok {
		maxv, err := laser.ToFloat32(max)
		if errors.Log(err) == nil {
			v.Widget.SetMax(maxv)
		}
	}
	if step, ok := v.Tag("step"); ok {
		step, err := laser.ToFloat32(step)
		if errors.Log(err) == nil {
			v.Widget.SetStep(step)
		}
	}
	if format, ok := v.Tag("format"); ok {
		v.Widget.SetFormat(format)
	}
	v.Widget.OnChange(func(e events.Event) {
		v.SetValue(v.Widget.Value)
	})
}

func (v *NumberValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	fv, err := laser.ToFloat32(npv.Interface())
	if errors.Log(err) == nil {
		v.Widget.SetValue(fv)
	}
}

// SliderValue represents an integer or float value with a slider.
type SliderValue struct {
	ValueBase[*core.Slider]
}

func (v *SliderValue) Config() {
	vk := laser.NonPtrType(v.Value.Type()).Kind()
	if vk >= reflect.Int && vk <= reflect.Uintptr {
		v.Widget.SetStep(1).SetEnforceStep(true).SetMax(100)
	}
	if min, ok := v.Tag("min"); ok {
		minv, err := laser.ToFloat32(min)
		if errors.Log(err) == nil {
			v.Widget.SetMin(minv)
		}
	}
	if max, ok := v.Tag("max"); ok {
		maxv, err := laser.ToFloat32(max)
		if errors.Log(err) == nil {
			v.Widget.SetMax(maxv)
		}
	}
	if step, ok := v.Tag("step"); ok {
		stepv, err := laser.ToFloat32(step)
		if errors.Log(err) == nil {
			v.Widget.SetStep(stepv)
		}
	}
	v.Widget.OnChange(func(e events.Event) {
		v.SetValue(v.Widget.Value)
	})
}

func (v *SliderValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	fv, err := laser.ToFloat32(npv.Interface())
	if errors.Log(err) == nil {
		v.Widget.SetValue(fv)
	}
}

// StructValue represents a struct value with a button.
type StructValue struct {
	ValueBase[*core.Button]
}

func (v *StructValue) Config() {
	v.Widget.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogWidget(v, true)
}

func (v *StructValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	if v.Value.IsZero() {
		v.Widget.SetText("None")
	} else {
		opv := laser.OnePtrUnderlyingValue(v.Value)
		if lbler, ok := opv.Interface().(core.Labeler); ok {
			v.Widget.SetText(lbler.Label())
		} else {
			v.Widget.SetText(laser.FriendlyTypeName(npv.Type()))
		}
	}
	v.Widget.Update()
}

func (v *StructValue) ConfigDialog(d *core.Body) (bool, func()) {
	if v.Value.IsZero() {
		return false, nil
	}
	opv := laser.OnePtrUnderlyingValue(v.Value)
	str := opv.Interface()
	NewStructView(d).SetStruct(str).SetViewPath(v.ViewPath).
		SetReadOnly(v.IsReadOnly())
	if tb, ok := str.(core.Toolbarer); ok {
		d.AddAppBar(tb.ConfigToolbar)
	}
	return true, nil
}

// StructInlineValue represents a struct value with a [StructViewInline].
type StructInlineValue struct {
	ValueBase[*StructViewInline]
}

func (v *StructInlineValue) Config() {
	v.Widget.StructValue = v
	v.Widget.ViewPath = v.ViewPath
	v.Widget.SetStruct(v.Value.Interface())
	v.Widget.OnChange(func(e events.Event) {
		v.SendChange(e)
	})
}

func (v *StructInlineValue) Update() {
	v.Widget.SetStruct(v.Value.Interface())
}

// SliceValue represents a slice or array value with a button.
type SliceValue struct {
	ValueBase[*core.Button]
}

func (v *SliceValue) Config() {
	v.Widget.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogWidget(v, true)
}

func (v *SliceValue) Update() {
	npv := laser.OnePtrUnderlyingValue(v.Value).Elem()
	txt := ""
	if !npv.IsValid() {
		txt = "None"
	} else {
		if npv.Kind() == reflect.Array || !npv.IsNil() {
			bnm := laser.FriendlyTypeName(laser.SliceElType(v.Value.Interface()))
			if strings.HasSuffix(bnm, "s") {
				txt = strcase.ToSentence(fmt.Sprintf("%d lists of %s", npv.Len(), bnm))
			} else {
				txt = strcase.ToSentence(fmt.Sprintf("%d %ss", npv.Len(), bnm))
			}
		} else {
			txt = "None"
		}
	}
	v.Widget.SetText(txt).Update()
}

func (v *SliceValue) ConfigDialog(d *core.Body) (bool, func()) {
	npv := laser.NonPtrValue(v.Value)
	if v.Value.IsZero() || npv.IsZero() {
		return false, nil
	}
	vvp := laser.OnePtrValue(v.Value)
	if vvp.Kind() != reflect.Ptr {
		slog.Error("views.SliceValue: Cannot view unadressable (non-pointer) slices", "type", v.Value.Type())
		return false, nil
	}
	slci := vvp.Interface()
	if npv.Kind() != reflect.Array && laser.NonPtrType(laser.SliceElType(v.Value.Interface())).Kind() == reflect.Struct {
		tv := NewTableView(d).SetSlice(slci).SetViewPath(v.ViewPath)
		tv.SetReadOnly(v.IsReadOnly())
		d.AddAppBar(tv.ConfigToolbar)
	} else {
		sv := NewSliceView(d).SetSlice(slci).SetViewPath(v.ViewPath)
		sv.SetReadOnly(v.IsReadOnly())
		d.AddAppBar(sv.ConfigToolbar)
	}
	return true, nil
}

// SliceInlineValue represents a slice or array value with a [SliceViewInline].
type SliceInlineValue struct {
	ValueBase[*SliceViewInline]
}

func (v *SliceInlineValue) Config() {
	v.Widget.SliceValue = v
	v.Widget.ViewPath = v.ViewPath
	v.Widget.SetSlice(v.Value.Interface())
	v.Widget.OnChange(func(e events.Event) {
		v.SendChange(e)
	})
}

func (v *SliceInlineValue) Update() {
	csl := v.Value.Interface()
	newslc := false
	if reflect.TypeOf(v.Value).Kind() != reflect.Pointer { // prevent crash on non-comparable
		newslc = true
	} else {
		newslc = v.Widget.Slice != csl
	}
	if newslc {
		v.Widget.SetSlice(csl)
	} else {
		v.Widget.Update()
	}
}

// MapValue represents a map value with a button.
type MapValue struct {
	ValueBase[*core.Button]
}

func (v *MapValue) Config() {
	v.Widget.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogWidget(v, true)
}

func (v *MapValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	mpi := v.Value.Interface()
	txt := ""
	if !npv.IsValid() || npv.IsNil() {
		txt = "None"
	} else {
		bnm := laser.FriendlyTypeName(laser.MapValueType(mpi))
		if strings.HasSuffix(bnm, "s") {
			txt = strcase.ToSentence(fmt.Sprintf("%d lists of %s", npv.Len(), bnm))
		} else {
			txt = strcase.ToSentence(fmt.Sprintf("%d %ss", npv.Len(), bnm))
		}
	}
	v.Widget.SetText(txt).Update()
}

func (v *MapValue) ConfigDialog(d *core.Body) (bool, func()) {
	if v.Value.IsZero() || laser.NonPtrValue(v.Value).IsZero() {
		return false, nil
	}
	mpi := v.Value.Interface()
	mv := NewMapView(d).SetMap(mpi)
	mv.SetViewPath(v.ViewPath).SetReadOnly(v.IsReadOnly())
	d.AddAppBar(mv.ConfigToolbar)
	return true, nil
}

// MapInlineValue represents a map value with a [MapViewInline].
type MapInlineValue struct {
	ValueBase[*MapViewInline]
}

func (v *MapInlineValue) Config() {
	v.Widget.MapValue = v
	v.Widget.ViewPath = v.ViewPath
	v.Widget.SetMap(v.Value.Interface())
	v.Widget.OnChange(func(e events.Event) {
		v.SendChange(e)
	})
}

func (v *MapInlineValue) Update() {
	cmp := v.Value.Interface()
	if v.Widget.Map != cmp {
		v.Widget.SetMap(cmp)
	} else {
		v.Widget.UpdateValues()
	}
}

// KiValue represents a [tree.Node] value with a button.
type KiValue struct {
	ValueBase[*core.Button]
}

func (v *KiValue) Config() {
	v.Widget.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogWidget(v, true)
}

func (v *KiValue) Update() {
	path := "None"
	k := v.KiValue()
	if k != nil && k.This() != nil {
		path = k.AsTreeNode().String()
	}
	v.Widget.SetText(path).Update()
}

func (v *KiValue) ConfigDialog(d *core.Body) (bool, func()) {
	k := v.KiValue()
	if k == nil {
		return false, nil
	}
	InspectorView(d, k)
	return true, nil
}

// KiValue returns the actual underlying [tree.Node] value, or nil.
func (vv *KiValue) KiValue() tree.Node {
	if !vv.Value.IsValid() || vv.Value.IsNil() {
		return nil
	}
	npv := laser.NonPtrValue(vv.Value)
	if npv.Kind() == reflect.Interface {
		return npv.Interface().(tree.Node)
	}
	opv := laser.OnePtrValue(vv.Value)
	if opv.IsNil() {
		return nil
	}
	return opv.Interface().(tree.Node)
}

// EnumValue represents an [enums.Enum] value with a chooser.
type EnumValue struct {
	ValueBase[*core.Chooser]
}

func (v *EnumValue) Config() {
	e := laser.OnePtrUnderlyingValue(v.Value).Interface().(enums.Enum)
	v.Widget.SetEnum(e)
	v.Widget.OnChange(func(e events.Event) {
		v.SetValue(v.Widget.CurrentItem.Value)
	})
}

func (v *EnumValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	v.Widget.SetCurrentValue(npv.Interface())
}

// BitFlagValue represents an [enums.BitFlag] value with chip switches.
type BitFlagValue struct {
	ValueBase[*core.Switches]
}

func (v *BitFlagValue) Config() {
	v.Widget.SetType(core.SwitchChip).SetEnum(v.EnumValue())
	v.Widget.OnChange(func(e events.Event) {
		v.Widget.UpdateBitFlag(v.EnumValue())
	})
}

func (v *BitFlagValue) Update() {
	v.Widget.UpdateFromBitFlag(v.EnumValue())
}

// EnumValue returns the underlying [enums.BitFlagSetter] value.
func (v *BitFlagValue) EnumValue() enums.BitFlagSetter {
	// special case to use [tree.Node.FlagType] if we are the Flags field
	if v.Field != nil && v.Field.Name == "Flags" {
		if k, ok := v.Owner.(tree.Node); ok {
			return k.FlagType()
		}
	}
	e := v.Value.Interface().(enums.BitFlagSetter)
	return e
}

// TypeValue represents a [types.Type] value with a chooser.
type TypeValue struct {
	ValueBase[*core.Chooser]
}

func (v *TypeValue) Config() {
	typEmbeds := core.WidgetBaseType
	if tetag, ok := v.Tag("type-embeds"); ok {
		typ := types.TypeByName(tetag)
		if typ != nil {
			typEmbeds = typ
		}
	}

	tl := types.AllEmbeddersOf(typEmbeds)
	v.Widget.SetTypes(tl...)
	v.Widget.OnChange(func(e events.Event) {
		tval := v.Widget.CurrentItem.Value.(*types.Type)
		v.SetValue(tval)
	})
}

func (v *TypeValue) Update() {
	opv := laser.OnePtrValue(v.Value)
	typ := opv.Interface().(*types.Type)
	v.Widget.SetCurrentValue(typ)
}

// ByteSliceValue represents a slice of bytes with a text field.
type ByteSliceValue struct {
	ValueBase[*core.TextField]
}

func (v *ByteSliceValue) Config() {
	v.Widget.OnChange(func(e events.Event) {
		v.SetValue(v.Widget.Text())
	})
}

func (v *ByteSliceValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	bv := npv.Interface().([]byte)
	v.Widget.SetText(string(bv))
}

// RuneSliceValue represents a slice of runes with a text field.
type RuneSliceValue struct {
	ValueBase[*core.TextField]
}

func (v *RuneSliceValue) Config() {
	v.Widget.OnChange(func(e events.Event) {
		v.SetValue(v.Widget.Text())
	})
}

func (v *RuneSliceValue) Update() {
	npv := laser.NonPtrValue(v.Value)
	rv := npv.Interface().([]rune)
	v.Widget.SetText(string(rv))
}

// NilValue represents a nil value with a label that has text "None".
type NilValue struct {
	ValueBase[*core.Label]
}

func (v *NilValue) Config() {
	v.Widget.SetText("None")
}

func (vv *NilValue) Update() {}

// IconValue represents an [icons.Icon] value with a button.
type IconValue struct {
	ValueBase[*core.Button]
}

func (v *IconValue) Config() {
	v.Widget.SetType(core.ButtonTonal)
	ConfigDialogWidget(v, false)
}

func (v *IconValue) Update() {
	txt := laser.ToString(v.Value.Interface())
	if icons.Icon(txt).IsNil() {
		v.Widget.SetIcon(icons.Blank)
	} else {
		v.Widget.SetIcon(icons.Icon(txt))
	}
	if view, ok := v.Tag("view"); ok {
		if strings.Contains(view, "show-name") {
			if txt == "" {
				txt = "None"
			}
			v.Widget.SetText(strcase.ToSentence(txt))
		}
	}
	v.Widget.Update()
}

func (v *IconValue) ConfigDialog(d *core.Body) (bool, func()) {
	d.SetTitle("Select an icon")
	si := 0
	ics := icons.All()
	cur := icons.Icon(laser.ToString(v.Value.Interface()))
	NewSliceView(d).SetStyleFunc(func(w core.Widget, s *styles.Style, row int) {
		w.(*core.Button).SetText(strcase.ToSentence(string(ics[row])))
	}).SetSlice(&ics).SetSelectedValue(cur).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			ic := icons.AllIcons[si]
			v.SetValue(ic)
			v.Update()
		}
	}
}

// FontValue represents a [core.FontName] value with a button.
type FontValue struct {
	ValueBase[*core.Button]
}

func (v *FontValue) Config() {
	v.Widget.SetType(core.ButtonTonal)
	v.Widget.Style(func(s *styles.Style) {
		// TODO(kai): fix this not working (probably due to medium font weight)
		s.Font.Family = laser.ToString(v.Value.Interface())
	})
	ConfigDialogWidget(v, false)
}

func (v *FontValue) Update() {
	txt := laser.ToString(v.Value.Interface())
	v.Widget.SetText(txt).Update()
}

func (v *FontValue) ConfigDialog(d *core.Body) (bool, func()) {
	d.SetTitle("Select a font")
	si := 0
	fi := paint.FontLibrary.FontInfo
	cur := core.FontName(laser.ToString(v.Value.Interface()))
	NewTableView(d).SetStyleFunc(func(w core.Widget, s *styles.Style, row, col int) {
		if col != 4 {
			return
		}
		s.Font.Family = fi[row].Name
		s.Font.Stretch = fi[row].Stretch
		s.Font.Weight = fi[row].Weight
		s.Font.Style = fi[row].Style
		s.Font.Size.Pt(18)
	}).SetSlice(&fi).SetSelectedValue(cur).SetSelectedField("Name").BindSelect(&si)

	return true, func() {
		if si >= 0 {
			fi := paint.FontLibrary.FontInfo[si]
			v.SetValue(fi.Name)
			v.Update()
		}
	}
}

// FileValue represents a [core.Filename] value with a button.
type FileValue struct {
	ValueBase[*core.Button]
}

func (v *FileValue) Config() {
	v.Widget.SetType(core.ButtonTonal).SetIcon(icons.File)
	ConfigDialogWidget(v, false)
}

func (v *FileValue) Update() {
	txt := laser.ToString(v.Value.Interface())
	if txt == "" {
		txt = "(click to open file chooser)"
	}
	v.Widget.SetText(txt).Update()
}

func (v *FileValue) ConfigDialog(d *core.Body) (bool, func()) {
	v.SetFlag(true, ValueDialogNewWindow) // default to new window on supported platforms
	cur := laser.ToString(v.Value.Interface())
	ext, _ := v.Tag("ext")
	fv := NewFileView(d).SetFilename(cur, ext)
	d.AddAppBar(fv.ConfigToolbar)
	return true, func() {
		cur = fv.SelectedFile()
		v.SetValue(cur)
		v.Update()
	}
}

// FuncValue represents a function value with a [FuncButton].
type FuncValue struct {
	ValueBase[*FuncButton]
}

func (v *FuncValue) Config() {
	v.Widget.SetType(core.ButtonTonal)
}

func (v *FuncValue) Update() {
	fun := laser.NonPtrValue(v.Value).Interface()
	// if someone is viewing an arbitrary function, there is a good chance
	// that it is not added to gti (and that is out of their control)
	// (eg: in the inspector), so we do not warn on unadded functions.
	v.Widget.SetWarnUnadded(false).SetFunc(fun)
}
