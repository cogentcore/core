// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package histyle

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/laser"
)

func init() {
	giv.AddValue(core.HiStyleName(""), func() giv.Value { return &Value{} })
}

// Value represents a [core.HiStyleName] with a button.
type Value struct {
	giv.ValueBase[*core.Button]
}

func (v *Value) Config() {
	v.Widget.SetType(core.ButtonTonal).SetIcon(icons.Brush)
	giv.ConfigDialogWidget(v, false)
}

func (v *Value) Update() {
	txt := laser.ToString(v.Value.Interface())
	v.Widget.SetText(txt).Update()
}

func (v *Value) ConfigDialog(d *core.Body) (bool, func()) {
	d.SetTitle("Select a syntax highlighting style")
	si := 0
	cur := laser.ToString(v.Value.Interface())
	giv.NewSliceView(d).SetSlice(&StyleNames).SetSelectedValue(cur).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			hs := StyleNames[si]
			v.SetValue(hs)
			v.Update()
		}
	}
}

// View opens a view of highlighting styles
func View(st *Styles) {
	if core.ActivateExistingMainWindow(st) {
		return
	}

	d := core.NewBody("hi-styles").SetData(st)
	d.AddTitle("Highlighting Styles: use ViewStd to see builtin ones -- can add and customize -- save ones from standard and load into custom to modify standards.")
	mv := giv.NewMapView(d).SetMap(st)
	StylesChanged = false
	mv.OnChange(func(e events.Event) {
		StylesChanged = true
	})
	d.AddAppBar(func(tb *core.Toolbar) {
		oj := giv.NewFuncButton(tb, st.OpenJSON).SetText("Open from file").SetIcon(icons.Open)
		oj.Args[0].SetTag(".ext", ".histy")
		sj := giv.NewFuncButton(tb, st.SaveJSON).SetText("Save from file").SetIcon(icons.Save)
		sj.Args[0].SetTag(".ext", ".histy")
		core.NewSeparator(tb)
		mv.ConfigToolbar(tb)
	})
	d.NewWindow().Run() // note: no context here so not dialog
}
