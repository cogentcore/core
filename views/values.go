// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"log/slog"
	"reflect"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
)

// This file contains the standard [Value]s built into views.

// SliceButton represents a slice or array value with a button.
type SliceButton struct {
	core.Button
	Slice any
}

func (sb *SliceButton) WidgetValue() any { return &sb.Slice }

func (sb *SliceButton) OnInit() {
	sb.Button.OnInit()
	sb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogValue(sb, true)
}

func (sb *SliceButton) Make(p *core.Plan) {
	sb.SetText(labels.FriendlySliceLabel(reflect.ValueOf(sb.Slice)))
	sb.Button.Make(p)
}

func (sb *SliceButton) ConfigDialog(d *core.Body) (bool, func()) {
	npv := reflectx.NonPointerValue(reflect.ValueOf(sb.Slice))
	if reflectx.AnyIsNil(sb.Slice) || npv.IsZero() {
		return false, nil
	}
	vvp := reflectx.UnderlyingPointer(reflect.ValueOf(sb.Slice))
	if vvp.Kind() != reflect.Pointer {
		slog.Error("views.SliceButton: Cannot view unadressable (non-pointer) slices", "type", npv.Type())
		return false, nil
	}
	slci := vvp.Interface()
	if npv.Kind() != reflect.Array && reflectx.NonPointerType(reflectx.SliceElementType(sb.Slice)).Kind() == reflect.Struct {
		tv := NewTableView(d).SetSlice(slci).SetViewPath(sb.ValueContext)
		tv.SetReadOnly(sb.IsReadOnly())
		d.AddAppBar(tv.MakeToolbar)
	} else {
		sv := NewSliceView(d).SetSlice(slci).SetViewPath(sb.ValueContext)
		sv.SetReadOnly(sb.IsReadOnly())
		d.AddAppBar(sv.MakeToolbar)
	}
	return true, nil
}

// StructButton represents a slice or array value with a button.
type StructButton struct {
	core.Button
	Struct any
}

func (sb *StructButton) WidgetValue() any { return &sb.Struct }

func (sb *StructButton) OnInit() {
	sb.Button.OnInit()
	sb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogValue(sb, true)
}

func (sb *StructButton) Make(p *core.Plan) {
	sb.SetReadOnly(false)
	sb.SetText(labels.FriendlyStructLabel(reflect.ValueOf(sb.Struct)))
	sb.Button.Make(p)
}

func (sb *StructButton) ConfigDialog(d *core.Body) (bool, func()) {
	if reflectx.AnyIsNil(sb.Struct) {
		return false, nil
	}
	opv := reflectx.UnderlyingPointer(reflect.ValueOf(sb.Struct))
	str := opv.Interface()
	NewStructView(d).SetStruct(str).SetViewPath(sb.ValueContext).
		SetReadOnly(sb.IsReadOnly())
	if tb, ok := str.(core.ToolbarMaker); ok {
		d.AddAppBar(tb.MakeToolbar)
	}
	return true, nil
}

// MapButton represents a slice or array value with a button.
type MapButton struct {
	core.Button
	Map any
}

func (sb *MapButton) WidgetValue() any { return &sb.Map }

func (sb *MapButton) OnInit() {
	sb.Button.OnInit()
	sb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	ConfigDialogValue(sb, true)
}

func (sb *MapButton) Make(p *core.Plan) {
	sb.SetText(labels.FriendlyMapLabel(reflect.ValueOf(sb.Map)))
	sb.Button.Make(p)
}

func (sb *MapButton) ConfigDialog(d *core.Body) (bool, func()) {
	if reflectx.AnyIsNil(sb.Map) || reflectx.NonPointerValue(reflect.ValueOf(sb.Map)).IsZero() {
		return false, nil
	}
	mpi := sb.Map
	mv := NewMapView(d).SetMap(mpi)
	mv.SetViewPath(sb.ValueContext).SetReadOnly(sb.IsReadOnly())
	d.AddAppBar(mv.MakeToolbar)
	return true, nil
}
