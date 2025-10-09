// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"reflect"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
	"golang.org/x/exp/maps"
)

// ListButton represents a slice or array value with a button that opens a [List].
type ListButton struct {
	Button
	Slice any
}

func (lb *ListButton) WidgetValue() any { return &lb.Slice }

func (lb *ListButton) Init() {
	lb.Button.Init()
	lb.SetType(ButtonTonal).SetIcon(icons.Edit)
	lb.Updater(func() {
		lb.SetText(labels.FriendlySliceLabel(reflect.ValueOf(lb.Slice)))
	})
	InitValueButton(lb, true, func(d *Body) {
		up := reflectx.Underlying(reflect.ValueOf(lb.Slice))
		if up.Type().Kind() != reflect.Array && reflectx.NonPointerType(reflectx.SliceElementType(lb.Slice)).Kind() == reflect.Struct {
			tb := NewTable(d).SetSlice(lb.Slice)
			tb.SetValueTitle(lb.ValueTitle).SetReadOnly(lb.IsReadOnly())
			d.AddTopBar(func(bar *Frame) {
				NewToolbar(bar).Maker(tb.MakeToolbar)
			})
		} else {
			sv := NewList(d).SetSlice(lb.Slice)
			sv.SetValueTitle(lb.ValueTitle).SetReadOnly(lb.IsReadOnly())
			d.AddTopBar(func(bar *Frame) {
				NewToolbar(bar).Maker(sv.MakeToolbar)
			})
		}
	})
}

// FormButton represents a struct value with a button that opens a [Form].
type FormButton struct {
	Button
	Struct any
}

func (fb *FormButton) WidgetValue() any { return &fb.Struct }

func (fb *FormButton) Init() {
	fb.Button.Init()
	fb.SetType(ButtonTonal).SetIcon(icons.Edit)
	fb.Updater(func() {
		fb.SetText(labels.FriendlyStructLabel(reflect.ValueOf(fb.Struct)))
	})
	InitValueButton(fb, true, func(d *Body) {
		fm := NewForm(d).SetStruct(fb.Struct)
		fm.SetValueTitle(fb.ValueTitle).SetReadOnly(fb.IsReadOnly())
		if tb, ok := fb.Struct.(ToolbarMaker); ok {
			d.AddTopBar(func(bar *Frame) {
				NewToolbar(bar).Maker(tb.MakeToolbar)
			})
		}
	})
}

// KeyedListButton represents a map value with a button that opens a [KeyedList].
type KeyedListButton struct {
	Button
	Map any
}

func (kb *KeyedListButton) WidgetValue() any { return &kb.Map }

func (kb *KeyedListButton) Init() {
	kb.Button.Init()
	kb.SetType(ButtonTonal).SetIcon(icons.Edit)
	kb.Updater(func() {
		kb.SetText(labels.FriendlyMapLabel(reflect.ValueOf(kb.Map)))
	})
	InitValueButton(kb, true, func(d *Body) {
		kl := NewKeyedList(d).SetMap(kb.Map)
		kl.SetValueTitle(kb.ValueTitle).SetReadOnly(kb.IsReadOnly())
		d.AddTopBar(func(bar *Frame) {
			NewToolbar(bar).Maker(kl.MakeToolbar)
		})
	})
}

// TreeButton represents a [tree.Node] value with a button.
type TreeButton struct {
	Button
	Tree tree.Node
}

func (tb *TreeButton) WidgetValue() any { return &tb.Tree }

func (tb *TreeButton) Init() {
	tb.Button.Init()
	tb.SetType(ButtonTonal).SetIcon(icons.Edit)
	tb.Updater(func() {
		path := "None"
		if !reflectx.UnderlyingPointer(reflect.ValueOf(tb.Tree)).IsNil() {
			path = tb.Tree.AsTree().String()
		}
		tb.SetText(path)
	})
	InitValueButton(tb, true, func(d *Body) {
		if !reflectx.UnderlyingPointer(reflect.ValueOf(tb.Tree)).IsNil() {
			makeInspector(d, tb.Tree)
		}
	})
}

func (tb *TreeButton) WidgetTooltip(pos image.Point) (string, image.Point) {
	if reflectx.UnderlyingPointer(reflect.ValueOf(tb.Tree)).IsNil() {
		return tb.Tooltip, tb.DefaultTooltipPos()
	}
	tpa := "(" + tb.Tree.AsTree().Path() + ")"
	if tb.Tooltip == "" {
		return tpa, tb.DefaultTooltipPos()
	}
	return tpa + " " + tb.Tooltip, tb.DefaultTooltipPos()
}

// TypeChooser represents a [types.Type] value with a chooser.
type TypeChooser struct {
	Chooser
}

func (tc *TypeChooser) Init() {
	tc.Chooser.Init()
	tc.SetTypes(maps.Values(types.Types)...)
}

// IconButton represents an [icons.Icon] with a [Button] that opens
// a dialog for selecting the icon.
type IconButton struct {
	Button
}

func (ib *IconButton) WidgetValue() any { return &ib.Icon }

func (ib *IconButton) Init() {
	ib.Button.Init()
	ib.Updater(func() {
		if !ib.Icon.IsSet() {
			ib.SetText("Select an icon")
		} else {
			ib.SetText("")
		}
		if ib.IsReadOnly() {
			ib.SetType(ButtonText)
			if !ib.Icon.IsSet() {
				ib.SetText("").SetIcon(icons.Blank)
			}
		} else {
			ib.SetType(ButtonTonal)
		}
	})
	InitValueButton(ib, false, func(d *Body) {
		d.SetTitle("Select an icon")
		si := 0
		used := icons.Used()
		ls := NewList(d)
		ls.SetSlice(&used).SetSelectedValue(ib.Icon).BindSelect(&si)
		ls.OnChange(func(e events.Event) {
			ib.Icon = used[si]
		})
	})
}

// FontName is used to specify a font family name.
// It results in a [FontButton] [Value].
type FontName = rich.FontName

// FontButton represents a [FontName] with a [Button] that opens
// a dialog for selecting the font family.
type FontButton struct {
	Button
}

func (fb *FontButton) WidgetValue() any { return &fb.Text }

func (fb *FontButton) Init() {
	fb.Button.Init()
	fb.SetType(ButtonTonal)
	fb.Updater(func() {
		if fb.Text == "" {
			fb.SetText("(default)")
		}
	})
	InitValueButton(fb, false, func(d *Body) {
		d.SetTitle("Select a font family")
		si := 0
		fi := fonts.Families(fb.Scene.TextShaper().FontList())
		tb := NewTable(d)
		tb.SetSlice(&fi).SetSelectedField("Family").SetSelectedValue(fb.Text).BindSelect(&si)
		tb.SetTableStyler(func(w Widget, s *styles.Style, row, col int) {
			if col != 1 {
				return
			}
			s.Font.CustomFont = rich.FontName(fi[row].Family)
			s.Font.Family = rich.Custom
			s.Font.Size.Dp(24)
		})
		tb.OnChange(func(e events.Event) {
			fb.Text = fi[si].Family
		})
	})
}

// HighlightingName is a highlighting style name.
type HighlightingName = highlighting.HighlightingName

// HighlightingButton represents a [HighlightingName] with a button.
type HighlightingButton struct {
	Button
	HighlightingName string
}

func (hb *HighlightingButton) WidgetValue() any { return &hb.HighlightingName }

func (hb *HighlightingButton) Init() {
	hb.Button.Init()
	hb.SetType(ButtonTonal).SetIcon(icons.Brush)
	hb.Updater(func() {
		hb.SetText(hb.HighlightingName)
	})
	InitValueButton(hb, false, func(d *Body) {
		d.SetTitle("Select a syntax highlighting style")
		si := 0
		ls := NewList(d).SetSlice(&highlighting.StyleNames).SetSelectedValue(hb.HighlightingName).BindSelect(&si)
		ls.OnChange(func(e events.Event) {
			hb.HighlightingName = highlighting.StyleNames[si]
		})
	})
}

// Editor opens an editor of highlighting styles.
func HighlightingEditor(st *highlighting.Styles) {
	if RecycleMainWindow(st) {
		return
	}

	d := NewBody("Highlighting styles").SetData(st)
	NewText(d).SetType(TextSupporting).SetText("View standard to see the builtin styles, from which you can add and customize by saving ones from the standard and then loading them into a custom file to modify.")
	kl := NewKeyedList(d).SetMap(st)
	highlighting.StylesChanged = false
	kl.OnChange(func(e events.Event) {
		highlighting.StylesChanged = true
	})
	d.AddTopBar(func(bar *Frame) {
		NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *FuncButton) {
				w.SetFunc(st.OpenJSON).SetText("Open from file").SetIcon(icons.Open)
				w.Args[0].SetTag(`extension:".highlighting"`)
			})
			tree.Add(p, func(w *FuncButton) {
				w.SetFunc(st.SaveJSON).SetText("Save from file").SetIcon(icons.Save)
				w.Args[0].SetTag(`extension:".highlighting"`)
			})
			tree.Add(p, func(w *Button) {
				w.SetText("View standard").SetIcon(icons.Visibility).OnClick(func(e events.Event) {
					HighlightingEditor(&highlighting.StandardStyles)
				})
			})
			tree.Add(p, func(w *Separator) {})
			kl.MakeToolbar(p)
		})
	})
	d.RunWindow() // note: no context here so not dialog
}
