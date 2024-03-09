// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
)

// KeyMapsView opens a view of a key maps table
func KeyMapsView(km *keyfun.Maps) {
	if gi.ActivateExistingMainWindow(km) {
		return
	}
	d := gi.NewBody("Key maps").SetData(km)
	d.AddTitle("Available key maps: duplicate an existing map (using context menu) as starting point for creating a custom map")
	tv := NewTableView(d).SetSlice(km)
	keyfun.AvailMapsChanged = false
	tv.OnChange(func(e events.Event) {
		keyfun.AvailMapsChanged = true
	})

	d.Scene.Data = km // todo: needed?
	d.AddAppBar(func(tb *gi.Toolbar) {
		NewFuncButton(tb, km.SavePrefs).SetText("Save to preferences").SetIcon(icons.Save).SetKey(keyfun.Save).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(keyfun.AvailMapsChanged && km == &keyfun.AvailMaps) })
		oj := NewFuncButton(tb, km.Open).SetText("Open from file").SetIcon(icons.Open).SetKey(keyfun.Open)
		oj.Args[0].SetTag("ext", ".json")
		sj := NewFuncButton(tb, km.Save).SetText("Save to file").SetIcon(icons.SaveAs).SetKey(keyfun.SaveAs)
		sj.Args[0].SetTag("ext", ".json")
		gi.NewSeparator(tb)
		NewFuncButton(tb, ViewStdKeyMaps).SetConfirm(true).
			SetText("View standard").SetIcon(icons.Visibility).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(km != &keyfun.StdMaps) })

		NewFuncButton(tb, km.RevertToStd).SetConfirm(true).
			SetText("Revert to standard").SetIcon(icons.DeviceReset).
			StyleFirst(func(s *styles.Style) { s.SetEnabled(km != &keyfun.StdMaps) })
		NewFuncButton(tb, km.MarkdownDoc).SetIcon(icons.Document).
			SetShowReturn(true).SetShowReturnAsDialog(true)
		tb.AddOverflowMenu(func(m *gi.Scene) {
			NewFuncButton(m, km.OpenSettings).SetIcon(icons.Open).SetKey(keyfun.OpenAlt1)
		})
	})

	d.NewWindow().Run()
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
	bt.SetText(txt).Update()
}

func (vv *KeyMapValue) Config(w gi.Widget) {
	if vv.Widget == w {
		vv.UpdateWidget()
		return
	}
	vv.Widget = w
	vv.StdConfig(w)
	bt := vv.Widget.(*gi.Button)
	bt.SetType(gi.ButtonTonal)
	ConfigDialogWidget(vv, bt, false)
	vv.UpdateWidget()
}

func (vv *KeyMapValue) HasDialog() bool { return true }
func (vv *KeyMapValue) OpenDialog(ctx gi.Widget, fun func()) {
	OpenValueDialog(vv, ctx, fun, "Select a key map")
}

func (vv *KeyMapValue) ConfigDialog(d *gi.Body) (bool, func()) {
	si := 0
	cur := laser.ToString(vv.Value.Interface())
	_, curRow, _ := keyfun.AvailMaps.MapByName(keyfun.MapName(cur))
	NewTableView(d).SetSlice(&keyfun.AvailMaps).SetSelIdx(curRow).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			km := keyfun.AvailMaps[si]
			vv.SetValue(keyfun.MapName(km.Name))
			vv.UpdateWidget()
		}
	}
}
