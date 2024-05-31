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
	sb.Updater(func() {
		sb.SetText(labels.FriendlySliceLabel(reflect.ValueOf(sb.Slice)))
	})
	InitValueButton(sb, true, func(d *core.Body) {
		up := reflectx.Underlying(reflect.ValueOf(sb.Slice))
		if up.Type().Kind() != reflect.Array && reflectx.NonPointerType(reflectx.SliceElementType(sb.Slice)).Kind() == reflect.Struct {
			tv := NewTableView(d).SetSlice(sb.Slice)
			tv.SetValueContext(sb.ValueContext).SetReadOnly(sb.IsReadOnly())
			d.AddAppBar(tv.MakeToolbar)
		} else {
			sv := NewSliceView(d).SetSlice(sb.Slice)
			sv.SetValueContext(sb.ValueContext).SetReadOnly(sb.IsReadOnly())
			d.AddAppBar(sv.MakeToolbar)
		}
	})
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
	sb.Updater(func() {
		sb.SetText(labels.FriendlyStructLabel(reflect.ValueOf(sb.Struct)))
	})
	InitValueButton(sb, true, func(d *core.Body) {
		sv := NewStructView(d).SetStruct(sb.Struct)
		sv.SetValueContext(sb.ValueContext).SetReadOnly(sb.IsReadOnly())
		if tb, ok := sb.Struct.(core.ToolbarMaker); ok {
			d.AddAppBar(tb.MakeToolbar)
		}
	})
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
	mb.Updater(func() {
		mb.SetText(labels.FriendlyMapLabel(reflect.ValueOf(mb.Map)))
	})
	InitValueButton(mb, true, func(d *core.Body) {
		mv := NewMapView(d).SetMap(mb.Map)
		mv.SetValueContext(mb.ValueContext).SetReadOnly(mb.IsReadOnly())
		d.AddAppBar(mv.MakeToolbar)
	})
}
