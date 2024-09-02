// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package highlighting

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
)

func init() {
	core.AddValueType[core.HighlightingName, Button]()
}

// Button represents a [core.HighlightingName] with a button.
type Button struct {
	core.Button
	HighlightingName string
}

func (hb *Button) WidgetValue() any { return &hb.HighlightingName }

func (hb *Button) Init() {
	hb.Button.Init()
	hb.SetType(core.ButtonTonal).SetIcon(icons.Brush)
	hb.Updater(func() {
		hb.SetText(hb.HighlightingName)
	})
	core.InitValueButton(hb, false, func(d *core.Body) {
		d.SetTitle("Select a syntax highlighting style")
		si := 0
		ls := core.NewList(d).SetSlice(&StyleNames).SetSelectedValue(hb.HighlightingName).BindSelect(&si)
		ls.OnChange(func(e events.Event) {
			hb.HighlightingName = StyleNames[si]
		})
	})
}

// Editor opens an editor of highlighting styles.
func Editor(st *Styles) {
	if core.RecycleMainWindow(st) {
		return
	}

	d := core.NewBody("Highlighting styles").SetData(st)
	core.NewText(d).SetType(core.TextSupporting).SetText("View standard to see the builtin styles, from which you can add and customize by saving ones from the standard and then loading them into a custom file to modify.")
	kl := core.NewKeyedList(d).SetMap(st)
	StylesChanged = false
	kl.OnChange(func(e events.Event) {
		StylesChanged = true
	})
	d.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.FuncButton) {
				w.SetFunc(st.OpenJSON).SetText("Open from file").SetIcon(icons.Open)
				w.Args[0].SetTag(`extension:".highlighting"`)
			})
			tree.Add(p, func(w *core.FuncButton) {
				w.SetFunc(st.SaveJSON).SetText("Save from file").SetIcon(icons.Save)
				w.Args[0].SetTag(`extension:".highlighting"`)
			})
			tree.Add(p, func(w *core.FuncButton) {
				w.SetFunc(st.ViewStandard).SetIcon(icons.Visibility)
			})
			tree.Add(p, func(w *core.Separator) {})
			kl.MakeToolbar(p)
		})
	})
	d.RunWindow() // note: no context here so not dialog
}
