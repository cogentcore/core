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

func (vv *HiStyleValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfigWidget(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	bt.Config(sc)
	bt.OnClick(func(e events.Event) {
		vv.OpenDialog(bt)
	})
	vv.UpdateWidget()
}

func (vv *HiStyleValue) HasDialog() bool {
	return true
}

func (vv *HiStyleValue) OpenDialog(ctx gi.Widget) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	d := gi.NewBody().AddTitle("Select a HiStyle Highlighting Style").AddText(vv.Doc())
	sc := gi.NewScene(d)
	NewSliceView(d).SetSlice(&histyle.StyleNames).SetSelVal(cur).BindSelectDialog(sc, &si)
	sc.Footer.Add(func(par gi.Widget) {
		sc.AddCancel(par)
		sc.AddOk(par).OnClick(func(e events.Event) {
			if si >= 0 {
				hs := histyle.StyleNames[si]
				vv.SetValue(hs)
				vv.UpdateWidget()
			}
		})
	})
	gi.NewDialog(sc).SetContext(ctx).SetFullWindow(true).Run()
}

//////////////////////////////////////////////////////////////////////////////////////
//  HiStylesView

// HiStylesView opens a view of highlighting styles
func HiStylesView(st *histyle.Styles) {
	if gi.ActivateExistingMainWindow(st) {
		return
	}

	b := gi.NewBody("hi-styles")
	b.AddTitle("Highlighting Styles: use ViewStd to see builtin ones -- can add and customize -- save ones from standard and load into custom to modify standards.")
	mv := NewMapView(b).SetMap(st)
	histyle.StylesChanged = false
	mv.OnChange(func(e events.Event) {
		histyle.StylesChanged = true
	})
	sc := gi.NewScene(b)
	sc.Data = st
	sc.Header.Add(func(par gi.Widget) {
		tb := sc.TopAppBar(par)
		oj := NewFuncButton(tb, st.OpenJSON).SetText("Open from file").SetIcon(icons.Open)
		oj.Args[0].SetTag(".ext", ".histy")
		sj := NewFuncButton(tb, st.SaveJSON).SetText("Save from file").SetIcon(icons.Save)
		sj.Args[0].SetTag(".ext", ".histy")
		gi.NewSeparator(tb)
		mv.MapDefaultTopAppBar(tb)
	})
	gi.NewWindow(sc).Run() // todo: should be a dialog instead?
}
