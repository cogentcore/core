// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"reflect"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
)

// SliceButton represents a slice or array value with a button.
type SliceButton struct {
	core.Button
	Slice any
}

func (sb *SliceButton) WidgetValue() any { return &sb.Slice }

func (sb *SliceButton) OnInit() {
	sb.Button.OnInit()
	sb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	InitValueButton(sb, true)
	sb.Builder(func() {
		sb.SetText(labels.FriendlySliceLabel(reflect.ValueOf(sb.Slice)))
	})
}

func (sb *SliceButton) ConfigDialog(d *core.Body) (bool, func()) {
	up := reflectx.UnderlyingPointer(reflect.ValueOf(sb.Slice))
	if !up.IsValid() || up.IsZero() {
		return false, nil
	}
	upi := up.Interface()
	if up.Elem().Type().Kind() != reflect.Array && reflectx.NonPointerType(reflectx.SliceElementType(sb.Slice)).Kind() == reflect.Struct {
		tv := NewTableView(d).SetSlice(upi).SetViewPath(sb.ValueContext)
		tv.SetReadOnly(sb.IsReadOnly())
		d.AddAppBar(tv.MakeToolbar)
	} else {
		sv := NewSliceView(d).SetSlice(upi).SetViewPath(sb.ValueContext)
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
	InitValueButton(sb, true)
	sb.Builder(func() {
		sb.SetText(labels.FriendlyStructLabel(reflect.ValueOf(sb.Struct)))
	})
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

func (mb *MapButton) WidgetValue() any { return &mb.Map }

func (mb *MapButton) OnInit() {
	mb.Button.OnInit()
	mb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	InitValueButton(mb, true)
	mb.Builder(func() {
		mb.SetText(labels.FriendlyMapLabel(reflect.ValueOf(mb.Map)))
	})
}

func (mb *MapButton) ConfigDialog(d *core.Body) (bool, func()) {
	if reflectx.AnyIsNil(mb.Map) || reflectx.NonPointerValue(reflect.ValueOf(mb.Map)).IsZero() {
		return false, nil
	}
	mpi := mb.Map
	mv := NewMapView(d).SetMap(mpi)
	mv.SetViewPath(mb.ValueContext).SetReadOnly(mb.IsReadOnly())
	d.AddAppBar(mv.MakeToolbar)
	return true, nil
}
