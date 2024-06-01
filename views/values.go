// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"reflect"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
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
	core.InitValueButton(sb, true, func(d *core.Body) {
		up := reflectx.Underlying(reflect.ValueOf(sb.Slice))
		if up.Type().Kind() != reflect.Array && reflectx.NonPointerType(reflectx.SliceElementType(sb.Slice)).Kind() == reflect.Struct {
			tv := NewTableView(d).SetSlice(sb.Slice)
			tv.SetValueTitle(sb.ValueTitle).SetReadOnly(sb.IsReadOnly())
			d.AddAppBar(tv.MakeToolbar)
		} else {
			sv := NewSliceView(d).SetSlice(sb.Slice)
			sv.SetValueTitle(sb.ValueTitle).SetReadOnly(sb.IsReadOnly())
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
	core.InitValueButton(sb, true, func(d *core.Body) {
		sv := NewStructView(d).SetStruct(sb.Struct)
		sv.SetValueTitle(sb.ValueTitle).SetReadOnly(sb.IsReadOnly())
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
	core.InitValueButton(mb, true, func(d *core.Body) {
		mv := NewMapView(d).SetMap(mb.Map)
		mv.SetValueTitle(mb.ValueTitle).SetReadOnly(mb.IsReadOnly())
		d.AddAppBar(mv.MakeToolbar)
	})
}

// IconButton represents an [icons.Icon] with a [core.Button] that opens
// a dialog for selecting the icon.
type IconButton struct {
	core.Button
}

func (ib *IconButton) WidgetValue() any { return &ib.Icon }

func (ib *IconButton) OnInit() {
	ib.Button.OnInit()
	ib.SetType(core.ButtonTonal)
	core.InitValueButton(ib, false, func(d *core.Body) {
		d.SetTitle("Select an icon")
		si := 0
		all := icons.All()
		sv := NewSliceView(d)
		sv.SetSlice(&all).SetSelectedValue(ib.Icon).BindSelect(&si)
		sv.SetStyleFunc(func(w core.Widget, s *styles.Style, row int) {
			w.(*IconButton).SetText(strcase.ToSentence(string(all[row])))
		})
		sv.OnFinal(events.Select, func(e events.Event) {
			ib.Icon = icons.AllIcons[si]
		})
	})
}
