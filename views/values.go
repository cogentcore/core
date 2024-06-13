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
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// ListButton represents a slice or array value with a button that opens a [List].
type ListButton struct {
	core.Button
	Slice any
}

func (lb *ListButton) WidgetValue() any { return &lb.Slice }

func (lb *ListButton) Init() {
	lb.Button.Init()
	lb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	lb.Updater(func() {
		lb.SetText(labels.FriendlySliceLabel(reflect.ValueOf(lb.Slice)))
	})
	core.InitValueButton(lb, true, func(d *core.Body) {
		up := reflectx.Underlying(reflect.ValueOf(lb.Slice))
		if up.Type().Kind() != reflect.Array && reflectx.NonPointerType(reflectx.SliceElementType(lb.Slice)).Kind() == reflect.Struct {
			tb := NewTable(d).SetSlice(lb.Slice)
			tb.SetValueTitle(lb.ValueTitle).SetReadOnly(lb.IsReadOnly())
			d.AddAppBar(tb.MakeToolbar)
		} else {
			sv := NewList(d).SetSlice(lb.Slice)
			sv.SetValueTitle(lb.ValueTitle).SetReadOnly(lb.IsReadOnly())
			d.AddAppBar(sv.MakeToolbar)
		}
	})
}

// FormButton represents a struct value with a button that opens a [Form].
type FormButton struct {
	core.Button
	Struct any
}

func (fb *FormButton) WidgetValue() any { return &fb.Struct }

func (fb *FormButton) Init() {
	fb.Button.Init()
	fb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	fb.Updater(func() {
		fb.SetText(labels.FriendlyStructLabel(reflect.ValueOf(fb.Struct)))
	})
	core.InitValueButton(fb, true, func(d *core.Body) {
		fm := NewForm(d).SetStruct(fb.Struct)
		fm.SetValueTitle(fb.ValueTitle).SetReadOnly(fb.IsReadOnly())
		if tb, ok := fb.Struct.(core.ToolbarMaker); ok {
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
		kv := NewKeyValueTable(d).SetMap(mb.Map)
		kv.SetValueTitle(mb.ValueTitle).SetReadOnly(mb.IsReadOnly())
		d.AddAppBar(kv.MakeToolbar)
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
			path = tb.Tree.AsTree().String()
		}
		tb.SetText(path)
	})
	core.InitValueButton(tb, true, func(d *core.Body) {
		InspectorView(d, tb.Tree)
	})
}

// TypeChooser represents a [types.Type] value with a chooser.
type TypeChooser struct {
	core.Chooser
}

func (tc *TypeChooser) Init() {
	tc.Chooser.Init()
	typEmbeds := core.WidgetBaseType
	// if tetag, ok := tc.Tag("type-embeds"); ok { // TODO(config)
	// 	typ := types.TypeByName(tetag)
	// 	if typ != nil {
	// 		typEmbeds = typ
	// 	}
	// }

	tl := types.AllEmbeddersOf(typEmbeds)
	tc.SetTypes(tl...)
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
		sv := NewList(d)
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
		tb := NewTable(d)
		tb.SetSlice(&fi).SetSelectedField("Name").SetSelectedValue(fb.Text).BindSelect(&si)
		tb.SetStyleFunc(func(w core.Widget, s *styles.Style, row, col int) {
			if col != 4 {
				return
			}
			s.Font.Family = fi[row].Name
			s.Font.Stretch = fi[row].Stretch
			s.Font.Weight = fi[row].Weight
			s.Font.Style = fi[row].Style
			s.Font.Size.Pt(18)
		})
		tb.OnChange(func(e events.Event) {
			fb.Text = fi[si].Name
		})
	})
}
