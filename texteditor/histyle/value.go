// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package histyle

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/laser"
)

func init() {
	giv.AddValue(gi.HiStyleName(""), func() giv.Value { return &Value{} })
}

// Value represents a [gi.HiStyleName] with a button.
type Value struct {
	giv.ValueBase[*gi.Button]
}

func (v *Value) Config() {
	v.Widget.SetType(gi.ButtonTonal).SetIcon(icons.Brush)
	giv.ConfigDialogWidget(v, false)
}

func (v *Value) Update() {
	txt := laser.ToString(v.Value.Interface())
	v.Widget.SetText(txt).Update()
}

// TODO(dtl): Select a syntax highlighting style
func (v *Value) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	cur := laser.ToString(v.Value.Interface())
	giv.NewSliceView(d).SetSlice(&StyleNames).SetSelVal(cur).BindSelect(&si)
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
	if gi.ActivateExistingMainWindow(st) {
		return
	}

	d := gi.NewBody("hi-styles").SetData(st)
	d.AddTitle("Highlighting Styles: use ViewStd to see builtin ones -- can add and customize -- save ones from standard and load into custom to modify standards.")
	mv := giv.NewMapView(d).SetMap(st)
	StylesChanged = false
	mv.OnChange(func(e events.Event) {
		StylesChanged = true
	})
	d.AddAppBar(func(tb *gi.Toolbar) {
		oj := giv.NewFuncButton(tb, st.OpenJSON).SetText("Open from file").SetIcon(icons.Open)
		oj.Args[0].SetTag(".ext", ".histy")
		sj := giv.NewFuncButton(tb, st.SaveJSON).SetText("Save from file").SetIcon(icons.Save)
		sj.Args[0].SetTag(".ext", ".histy")
		gi.NewSeparator(tb)
		mv.ConfigToolbar(tb)
	})
	d.NewWindow().Run() // note: no context here so not dialog
}
