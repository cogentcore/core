// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki"
)

// PrefsView opens a view of user preferences
func PrefsView(p *gi.Preferences) {
	winm := "gogi-prefs"
	if w, ok := gi.MainWindows.FindName(winm); ok {
		w.OSWin.Raise()
		return
	}

	width := 800
	height := 800
	win := gi.NewWindow2D(winm, "GoGi Preferences", width, height, true)

	if p.StdKeyMapName == "" {
		p.StdKeyMapName = gi.KeyMapName(gi.StdKeyMapName(gi.DefaultKeyMap))
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	sv := mfr.AddNewChild(KiT_StructView, "sv").(*StructView)
	sv.Viewport = vp
	sv.SetStruct(p, nil)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	mmen := win.MainMenu
	MainMenuView(p, win, mmen)

	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Prefs Before Closing?",
			Prompt: "Do you want to save any changes to preferences before closing?"},
			[]string{"Save and Close", "Discard and Close", "Cancel"},
			win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
