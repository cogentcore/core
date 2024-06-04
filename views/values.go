// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"reflect"
	"time"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// SliceButton represents a slice or array value with a button.
type SliceButton struct {
	core.Button
	Slice any
}

func (sb *SliceButton) WidgetValue() any { return &sb.Slice }

func (sb *SliceButton) Init() {
	sb.Button.Init()
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

func (sb *StructButton) Init() {
	sb.Button.Init()
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

func (mb *MapButton) Init() {
	mb.Button.Init()
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

// TreeButton represents a [tree.Node] value with a button.
type TreeButton struct {
	core.Button
	Tree tree.Node
}

func (tb *TreeButton) WidgetValue() any { return &tb.Tree }

func (tb *TreeButton) Init() {
	tb.Button.Init()
	tb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	tb.Updater(func() {
		path := "None"
		if tb.Tree != nil {
			path = tb.Tree.AsTreeNode().String()
		}
		tb.SetText(path)
	})
	core.InitValueButton(tb, true, func(d *core.Body) {
		InspectorView(d, tb.Tree)
	})
}

// IconButton represents an [icons.Icon] with a [core.Button] that opens
// a dialog for selecting the icon.
type IconButton struct {
	core.Button
}

func (ib *IconButton) WidgetValue() any { return &ib.Icon }

func (ib *IconButton) Init() { // TODO(config): view:"show-name"
	ib.Button.Init()
	ib.Updater(func() {
		if ib.IsReadOnly() {
			ib.SetType(core.ButtonText)
		} else {
			ib.SetType(core.ButtonTonal)
		}
		if ib.Icon.IsNil() {
			ib.Icon = icons.Blank
		}
	})
	core.InitValueButton(ib, false, func(d *core.Body) {
		d.SetTitle("Select an icon")
		si := 0
		all := icons.All()
		sv := NewSliceView(d)
		sv.SetSlice(&all).SetSelectedValue(ib.Icon).BindSelect(&si)
		sv.SetStyleFunc(func(w core.Widget, s *styles.Style, row int) {
			w.(*IconButton).SetText(strcase.ToSentence(string(all[row])))
		})
		sv.OnChange(func(e events.Event) {
			ib.Icon = icons.AllIcons[si]
		})
	})
}

// FontButton represents a [core.FontName] with a [core.Button] that opens
// a dialog for selecting the font family.
type FontButton struct {
	core.Button
}

func (fb *FontButton) WidgetValue() any { return &fb.Text }

func (fb *FontButton) Init() {
	fb.Button.Init()
	fb.SetType(core.ButtonTonal)
	core.InitValueButton(fb, false, func(d *core.Body) {
		d.SetTitle("Select a font family")
		si := 0
		fi := paint.FontLibrary.FontInfo
		tv := NewTableView(d)
		tv.SetSlice(&fi).SetSelectedField("Name").SetSelectedValue(fb.Text).BindSelect(&si)
		tv.SetStyleFunc(func(w core.Widget, s *styles.Style, row, col int) {
			if col != 4 {
				return
			}
			s.Font.Family = fi[row].Name
			s.Font.Stretch = fi[row].Stretch
			s.Font.Weight = fi[row].Weight
			s.Font.Style = fi[row].Style
			s.Font.Size.Pt(18)
		})
		tv.OnChange(func(e events.Event) {
			fb.Text = fi[si].Name
		})
	})
}

// TimeText represents a [time.Time] value with text
// that displays a standard date and time format.
type TimeText struct {
	core.Text
	Time time.Time
}

func (ft *TimeText) WidgetValue() any { return &ft.Time }

func (ft *TimeText) Init() {
	ft.Text.Init()
	ft.Updater(func() {
		ft.SetText(time.Time(ft.Time).Format("1/2/2006 " + core.SystemSettings.TimeFormat()))
	})
}
