// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/texteditor/histyle"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
)

////////////////////////////////////////////////////////////////////////////////////////
//  HiStyleValue

// HiStyleValue presents an action for displaying a mat32.Y and selecting
// from styles
type HiStyleValue struct {
	ValueBase
}

func (vv *HiStyleValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *HiStyleValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	bt.SetText(txt)
}

func (vv *HiStyleValue) ConfigWidget(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config()
	bt.OnClick(func(e events.Event) {
		if !vv.IsReadOnly() {
			vv.OpenDialog(vv.Widget, nil)
		}
	})
	vv.UpdateWidget()
}

func (vv *HiStyleValue) HasDialog() bool                      { return true }
func (vv *HiStyleValue) OpenDialog(ctx gi.Widget, fun func()) { OpenValueDialog(vv, ctx, fun) }

func (vv *HiStyleValue) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	NewSliceView(d).SetSlice(&histyle.StyleNames).SetSelVal(cur).BindSelectDialog(d.Sc, &si)
	return true, func() {
		if si >= 0 {
			hs := histyle.StyleNames[si]
			vv.SetValue(hs)
			vv.UpdateWidget()
		}
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//  HiStylesView

// HiStylesView opens a view of highlighting styles
func HiStylesView(st *histyle.Styles) {
	if gi.ActivateExistingMainWindow(st) {
		return
	}

	d := gi.NewBody("hi-styles")
	d.AddTitle("Highlighting Styles: use ViewStd to see builtin ones -- can add and customize -- save ones from standard and load into custom to modify standards.")
	mv := NewMapView(d).SetMap(st)
	histyle.StylesChanged = false
	mv.OnChange(func(e events.Event) {
		histyle.StylesChanged = true
	})
	d.Sc.Data = st                   // todo: still needed?
	d.AddTopBar(func(pw gi.Widget) { // todo: if?
		tb := d.DefaultTopAppBar(pw)
		oj := NewFuncButton(tb, st.OpenJSON).SetText("Open from file").SetIcon(icons.Open)
		oj.Args[0].SetTag(".ext", ".histy")
		sj := NewFuncButton(tb, st.SaveJSON).SetText("Save from file").SetIcon(icons.Save)
		sj.Args[0].SetTag(".ext", ".histy")
		gi.NewSeparator(tb)
		mv.MapDefaultTopAppBar(tb)
	})
	d.NewWindow().Run() // note: no context here so not dialog
}
