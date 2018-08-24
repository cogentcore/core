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
	spc.SetFixedHeight(units.NewValue(0.5, units.Em))

	sv := mfr.AddNewChild(KiT_StructView, "sv").(*StructView)
	sv.SetStruct(p, nil)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	stdmap := tbar.AddNewChild(gi.KiT_Action, "stdmap").(*gi.Action)
	stdmap.SetText("Std KeyMap")
	stdmap.Tooltip = "select a standard KeyMap -- copies map into CustomKeyMap, and you can customize from there by editing CustomKeyMap"
	stdmap.ActionSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		mapName := p.StdKeyMapName
		_, initRow := gi.StdKeyMapByName(mapName)
		SliceViewSelectDialog(vp, &gi.StdKeyMapNames, mapName, "Select a Standard KeyMap", "Can then customize from there", initRow, nil, stdmap.This,
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
			}, nil)
	})

	mmen := win.MainMenu
	MainMenuView(p, win, mmen)

	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		gi.ChoiceDialog(vp, "Save Prefs Before Closing?", "Do you want to save any changes to preferences before closing?", []string{"Save and Close", "Discard and Close", "Cancel"}, nil, win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			switch sig {
			case 0:
				p.Save()
				fmt.Println("Preferences Saved to prefs.json")
				w.Close()
			case 1:
				w.Close()
			case 2:
				// default is to do nothing, i.e., cancel
			}
		})
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}
