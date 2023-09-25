// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/goosi"
	"goki.dev/ki/v2"
)

// TODO: make base simplified preferences view, improve organization of information, and maybe add titles

// PrefsView opens a view of user preferences
func PrefsView(pf *gi.Preferences) *gi.RenderWin {
	winm := "gogi-prefs"
	width := 1280
	height := 500
	win, recyc := gi.RecycleMainRenderWin(pf, winm, "GoGi Preferences", width, height)
	if recyc {
		return win
	}

	vp := win.WinScene()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	sv := NewStructView(mfr, "sv")
	sv.Scene = vp
	sv.SetStruct(pf)
	sv.SetStretchMax()

	mmen := win.MainMenu
	MainMenuView(pf, win, mmen)

	inClosePrompt := false
	win.RenderWin.SetCloseReqFunc(func(w goosi.RenderWin) {
		if !pf.Changed {
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Prefs Before Closing?",
			Prompt: "Do you want to save any changes to preferences before closing?"},
			[]string{"Cancel", "Discard and Close", "Save and Close"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				switch sig {
				case 0:
					inClosePrompt = false
					// default is to do nothing, i.e., cancel
				case 1:
					pf.Open() // if we don't do this, then it actually remains in edited state
					win.Close()
				case 2:
					pf.Save()
					fmt.Println("Preferences Saved to prefs.json")
					win.Close()
				}
			})
	})

	win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.RenderWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
	return win
}

// PrefsDetView opens a view of user detailed preferences
func PrefsDetView(pf *gi.PrefsDetailed) *gi.RenderWin {
	winm := "gogi-prefs-det"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainRenderWin(pf, winm, "GoGi Detailed Preferences", width, height)
	if recyc {
		return win
	}

	vp := win.WinScene()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	sv := NewStructView(mfr, "sv")
	sv.Scene = vp
	sv.SetStruct(pf)
	sv.SetStretchMax()

	mmen := win.MainMenu
	MainMenuView(pf, win, mmen)

	inClosePrompt := false
	win.RenderWin.SetCloseReqFunc(func(w goosi.RenderWin) {
		if !pf.Changed {
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Save Prefs Before Closing?",
			Prompt: "Do you want to save any changes to detailed preferences before closing?"},
			[]string{"Save and Close", "Discard and Close", "Cancel"},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				switch sig {
				case 0:
					pf.Save()
					fmt.Println("Preferences Saved to prefs_det.json")
					win.Close()
				case 1:
					pf.Open() // if we don't do this, then it actually remains in edited state
					win.Close()
				case 2:
					inClosePrompt = false
					// default is to do nothing, i.e., cancel
				}
			})
	})

	win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.RenderWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
	return win
}

// PrefsDbgView opens a view of user debugging preferences
func PrefsDbgView(pf *gi.PrefsDebug) *gi.RenderWin {
	winm := "gogi-prefs-dbg"
	width := 800
	height := 800
	win, recyc := gi.RecycleMainRenderWin(pf, winm, "GoGi Debugging Preferences", width, height)
	if recyc {
		return win
	}

	vp := win.WinScene()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	sv := NewStructView(mfr, "sv")
	sv.Scene = vp
	sv.SetStruct(pf)
	sv.SetStretchMax()

	// mmen := win.MainMenu
	// MainMenuView(pf, win, mmen)
	// win.MainMenuUpdated()

	if !win.HasGeomPrefs() { // resize to contents
		vpsz := vp.PrefSize(win.RenderWin.Screen().PixSize)
		win.SetSize(vpsz)
	}

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
	return win
}
