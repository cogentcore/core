// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
)

// PrefsEditor opens an editor of user preferences
func PrefsEditor(p *gi.Preferences) {
	width := 800
	height := 600
	win := gi.NewWindow2D("gogi-prefs", "GoGi Preferences", width, height, true)

	if p.StdKeyMapName == "" {
		p.StdKeyMapName = gi.StdKeyMapName(gi.DefaultKeyMap)
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	vp.Fill = true

	vlay := vp.AddNewChild(gi.KiT_Frame, "vlay").(*gi.Frame)
	vlay.Lay = gi.LayoutCol

	trow := vlay.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutRow
	trow.SetStretchMaxWidth()

	spc := vlay.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	trow.AddNewChild(gi.KiT_Stretch, "str1")
	title := trow.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.Text = "GoGi Preferences"
	title.SetStretchMaxWidth()
	trow.AddNewChild(gi.KiT_Stretch, "str2")

	sv := vlay.AddNewChild(KiT_StructView, "sv").(*StructView)
	sv.SetStruct(p, nil)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	bspc := vlay.AddNewChild(gi.KiT_Space, "ButSpc").(*gi.Space)
	bspc.SetFixedHeight(units.NewValue(1.0, units.Em))

	brow := vlay.AddNewChild(gi.KiT_Layout, "brow").(*gi.Layout)
	brow.Lay = gi.LayoutRow
	brow.SetProp("horizontal-align", "center")
	brow.SetStretchMaxWidth()

	up := brow.AddNewChild(gi.KiT_Button, "update").(*gi.Button)
	up.SetText("Update")
	up.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			p.Update()
		}
	})

	savej := brow.AddNewChild(gi.KiT_Button, "savejson").(*gi.Button)
	savej.SetText("Save")
	savej.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			p.Save()
		}
	})

	loadj := brow.AddNewChild(gi.KiT_Button, "loadjson").(*gi.Button)
	loadj.SetText("Load")
	loadj.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			p.Load()
		}
	})

	stdmap := brow.AddNewChild(gi.KiT_Button, "stdmap").(*gi.Button)
	stdmap.SetText("Std KeyMap")
	stdmap.Tooltip = "select a standard KeyMap -- copies map into CustomKeyMap, and you can customize from there by editing CustomKeyMap"
	stdmap.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			mapName := p.StdKeyMapName
			_, initRow := gi.StdKeyMapByName(mapName)
			SliceViewSelectDialog(vp, &gi.StdKeyMapNames, "Select a Standard KeyMap", "Can then customize from there", initRow, nil, stdmap.This,
				func(recv, send ki.Ki, sig int64, data interface{}) {
					svv, _ := send.(*SliceView)
					si := svv.SelectedIdx
					if si >= 0 {
						mapName = gi.StdKeyMapNames[si]
					}
				},
				func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.DialogAccepted) {
						p.StdKeyMapName = mapName
						km, _ := gi.StdKeyMapByName(mapName)
						if km != nil {
							p.SetKeyMap(km)
							sv.UpdateFields()
						}
					}
				})
		}
	})

	scrinfo := brow.AddNewChild(gi.KiT_Button, "scrinfo").(*gi.Button)
	scrinfo.SetText("Screen Info")
	scrinfo.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(gi.ButtonClicked) {
			p.ScreenInfo()
		}
	})

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}
