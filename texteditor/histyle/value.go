// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package histyle

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/laser"
)

func init() {
	giv.ValueMapAdd(gi.HiStyleName(""), func() giv.Value {
		return &Value{}
	})
}

// Value presents a button for selecting a highlight styling method
type Value struct {
	giv.ValueBase
}

func (vv *Value) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *Value) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	bt.SetText(txt)
}

func (vv *Value) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	giv.ConfigDialogWidget(vv, bt, false)
	vv.UpdateWidget()
}

func (vv *Value) HasDialog() bool { return true }
func (vv *Value) OpenDialog(ctx gi.Widget, fun func()) {
	giv.OpenValueDialog(vv, ctx, fun, "Select a syntax highlighting style")
}

func (vv *Value) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	giv.NewSliceView(d).SetSlice(&StyleNames).SetSelVal(cur).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			hs := StyleNames[si]
			vv.SetValue(hs)
			vv.UpdateWidget()
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
