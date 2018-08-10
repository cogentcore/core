// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
)

// PrefsEditor opens an editor of user preferences
func PrefsEditor(p *gi.Preferences) {
	width := 800
	height := 800
	win := gi.NewWindow2D("gogi-prefs", "GoGi Preferences", width, height, true)

	if p.StdKeyMapName == "" {
		p.StdKeyMapName = gi.StdKeyMapName(gi.DefaultKeyMap)
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	tbar := mfr.AddNewChild(gi.KiT_ToolBar, "tbar").(*gi.ToolBar)
	tbar.Lay = gi.LayoutHoriz
	tbar.SetProp("horizontal-align", "center")
	tbar.SetStretchMaxWidth()

	trow := mfr.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutHoriz
	trow.SetStretchMaxWidth()

	spc := mfr.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	trow.AddNewChild(gi.KiT_Stretch, "str1")
	title := trow.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.Text = "GoGi Preferences"
	title.SetStretchMaxWidth()
	trow.AddNewChild(gi.KiT_Stretch, "str2")

	sv := mfr.AddNewChild(KiT_StructView, "sv").(*StructView)
	sv.SetStruct(p, nil)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	bspc := mfr.AddNewChild(gi.KiT_Space, "ButSpc").(*gi.Space)
	bspc.SetFixedHeight(units.NewValue(1.0, units.Em))

	up := tbar.AddNewChild(gi.KiT_Action, "update").(*gi.Action)
	up.SetText("Update")
	up.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		p.Update()
	})

	savej := tbar.AddNewChild(gi.KiT_Action, "savejson").(*gi.Action)
	savej.SetText("Save")
	savej.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		p.Save()
	})

	loadj := tbar.AddNewChild(gi.KiT_Action, "loadjson").(*gi.Action)
	loadj.SetText("Load")
	loadj.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		p.Load()
	})

	stdmap := tbar.AddNewChild(gi.KiT_Action, "stdmap").(*gi.Action)
	stdmap.SetText("Std KeyMap")
	stdmap.Tooltip = "select a standard KeyMap -- copies map into CustomKeyMap, and you can customize from there by editing CustomKeyMap"
	stdmap.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	})

	scrinfo := tbar.AddNewChild(gi.KiT_Action, "scrinfo").(*gi.Action)
	scrinfo.SetText("Screen Info")
	scrinfo.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		scinfo := p.ScreenInfo()
		fmt.Println(scinfo)
		gi.PromptDialog(win.Viewport, "Screen Info", scinfo, true, false, nil, nil, nil)
	})

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}
