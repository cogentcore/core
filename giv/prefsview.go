// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
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
	tbar.SetStretchMaxWidth()

	spc := mfr.AddNewChild(gi.KiT_Space, "spc1").(*gi.Space)
	spc.SetFixedHeight(units.NewValue(1.0, units.Em))

	sv := mfr.AddNewChild(KiT_StructView, "sv").(*StructView)
	sv.SetStruct(p, nil)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	bspc := mfr.AddNewChild(gi.KiT_Space, "ButSpc").(*gi.Space)
	bspc.SetFixedHeight(units.NewValue(1.0, units.Em))

	up := tbar.AddNewChild(gi.KiT_Action, "update").(*gi.Action)
	up.SetText("Update")
	up.Tooltip = "Update all windows with current prefs settings"
	up.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		p.Update()
	})

	savej := tbar.AddNewChild(gi.KiT_Action, "savejson").(*gi.Action)
	savej.SetText("Save")
	savej.Tooltip = "Save current prefs to prefs.json persistent prefs file in standard config prefs location for platform"
	savej.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		p.Save()
	})

	loadj := tbar.AddNewChild(gi.KiT_Action, "loadjson").(*gi.Action)
	loadj.SetText("Load")
	loadj.Tooltip = "Load saved prefs from prefs.json persistent prefs -- done automatically at startup"
	loadj.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		p.Load()
		vp.UpdateSig()
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
	scrinfo.Tooltip = "display information about all the currently-available screens -- can set per-screen preferences using name of screen"
	scrinfo.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		scinfo := p.ScreenInfo()
		fmt.Println(scinfo)
		gi.PromptDialog(win.Viewport, "Screen Info", scinfo, true, false, nil, nil, nil)
	})

	// main menu
	appnm := oswin.TheApp.Name()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.KnownChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	fmen := win.MainMenu.KnownChildByName("File", 0).(*gi.Action)
	fmen.Menu = make(gi.Menu, 0, 10)
	fmen.Menu.AddMenuText("Update", "Command+U", win.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		p.Update()
	})
	fmen.Menu.AddMenuText("Load", "Command+O", win.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		p.Load()
		vp.UpdateSig()
	})
	fmen.Menu.AddMenuText("Save", "Command+S", win.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		p.Save()
	})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddMenuText("Close Window", "Command+W", win.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		win.OSWin.Close()
	})

	emen := win.MainMenu.KnownChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win, false)

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}
