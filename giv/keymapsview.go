// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/laser"
)

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *keyfun.Maps) {
	if gi.ActivateExistingMainWindow(km) {
		return
	}
	sc := gi.NewScene("gogi-key-maps")
	sc.Title = "Available Key Maps: Duplicate an existing map (using Ctxt Menu) as starting point for creating a custom map"
	sc.Data = km

	title := gi.NewLabel(sc, "title").SetText(sc.Title).SetType(gi.LabelHeadlineSmall)
	title.Style(func(s *styles.Style) {
		s.Min.X.Ch(30) // need for wrap
		s.Grow.Set(1, 0)
		s.Text.WhiteSpace = styles.WhiteSpaceNormal // wrap
	})

	tv := NewTableView(sc).SetSlice(km)
	tv.SetStretchMax()

	keyfun.AvailMapsChanged = false
	tv.OnChange(func(e events.Event) {
		keyfun.AvailMapsChanged = true
	})

	sc.TopAppBar = func(tb *gi.TopAppBar) {
		gi.DefaultTopAppBar(tb)

		sp := NewFuncButton(tb, km.SavePrefs).SetText("Save to preferences").SetIcon(icons.Save).SetKey(keyfun.Save)
		sp.SetUpdateFunc(func() {
			sp.SetEnabled(keyfun.AvailMapsChanged && km == &keyfun.AvailMaps)
		})
		oj := NewFuncButton(tb, km.OpenJSON).SetText("Open from file").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".json")
		sj := NewFuncButton(tb, km.SaveJSON).SetText("Save to file").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".json")
		gi.NewSeparator(tb)
		vs := NewFuncButton(tb, ViewStdKeyMaps).SetConfirm(true).SetText("View standard").SetIcon(icons.Visibility)
		vs.SetUpdateFunc(func() {
			vs.SetEnabledUpdt(km != &keyfun.StdMaps)
		})
		rs := NewFuncButton(tb, km.RevertToStd).SetConfirm(true).SetText("Revert to standard").SetIcon(icons.DeviceReset)
		rs.SetUpdateFunc(func() {
			rs.SetEnabledUpdt(km != &keyfun.StdMaps)
		})
		tb.AddOverflowMenu(func(m *gi.Scene) {
			NewFuncButton(m, km.OpenPrefs).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	}

	gi.NewWindow(sc).Run()
}

// ViewStdKeyMaps shows the standard maps that are compiled into the program and have
// all the lastest key functions bound to standard values.  Useful for
// comparing against custom maps.
func ViewStdKeyMaps() { //gti:add
	KeyMapsView(&keyfun.StdMaps)
}

////////////////////////////////////////////////////////////////////////////////////////
//  KeyMapValue

// KeyMapValue presents an action for displaying a KeyMapName and selecting
// from chooser
type KeyMapValue struct {
	ValueBase
}

func (vv *KeyMapValue) WidgetType() *gti.Type {
	vv.WidgetTyp = gi.ButtonType
	return vv.WidgetTyp
}

func (vv *KeyMapValue) UpdateWidget() {
	if vv.Widget == nil {
		return
	}
	bt := vv.Widget.(*gi.Button)
	txt := laser.ToString(vv.Value.Interface())
	bt.SetText(txt)
}

func (vv *KeyMapValue) ConfigWidget(w gi.Widget, sc *gi.Scene) {
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
		vv.OpenDialog(bt, nil)
	})
	vv.UpdateWidget()
}

func (vv *KeyMapValue) HasDialog() bool {
	return true
}

func (vv *KeyMapValue) OpenDialog(ctx gi.Widget, fun func(d *gi.Dialog)) {
	if vv.IsReadOnly() {
		return
	}
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := keyfun.AvailMaps.MapByName(keyfun.MapName(cur))
	d := gi.NewDialog(ctx).Title("Select a key map").Prompt(vv.Doc()).FullWindow(true)
	NewTableView(d).SetSlice(&keyfun.AvailMaps).SetSelIdx(curRow).BindSelectDialog(d, &si)
	d.OnAccept(func(e events.Event) {
		if si >= 0 {
			km := keyfun.AvailMaps[si]
			vv.SetValue(km.Name)
			vv.UpdateWidget()
		}
		if fun != nil {
			fun(d)
		}
	}).Run()
}
