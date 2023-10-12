// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/goosi/events"
)

// TODO: make base simplified preferences view, improve organization of information, and maybe add titles

// PrefsView opens a view of user preferences
func PrefsView(pf *gi.Preferences) {
	if gi.ActivateExistingMainWindow(pf) {
		return
	}
	sc := gi.StageScene("gogi-prefs")
	sc.Title = "GoGi Preferences"
	sc.Lay = gi.LayoutVert
	sc.Data = pf

	sv := NewStructView(sc, "sv")
	sv.SetStruct(pf)
	sv.SetStretchMax()
	sv.OnChange(func(e events.Event) {
		pf.Apply()
		pf.Save()
	})

	/*
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
				func(dlg *gi.Dialog) {
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
	*/

	gi.NewWindow(sc).Run()
}

// PrefsDetView opens a view of user detailed preferences
func PrefsDetView(pf *gi.PrefsDetailed) {
	if gi.ActivateExistingMainWindow(pf) {
		return
	}

	sc := gi.StageScene("gogi-prefs-det")
	sc.Title = "GoGi Detailed Preferences"
	sc.Lay = gi.LayoutVert
	sc.Data = pf

	sv := NewStructView(sc, "sv")
	sv.SetStruct(pf)
	sv.SetStretchMax()

	/*
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
				func(dlg *gi.Dialog) {
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
	*/

	gi.NewWindow(sc).Run()
}

// PrefsDbgView opens a view of user debugging preferences
func PrefsDbgView(pf *gi.PrefsDebug) {
	if gi.ActivateExistingMainWindow(pf) {
		return
	}
	sc := gi.StageScene("gogi-prefs-dbg")
	sc.Title = "GoGi Debugging Preferences"
	sc.Lay = gi.LayoutVert
	sc.Data = pf

	sv := NewStructView(sc, "sv")
	sv.SetStruct(pf)
	sv.SetStretchMax()

	// mmen := win.MainMenu
	// MainMenuView(pf, win, mmen)
	// win.MainMenuUpdated()

	gi.NewWindow(sc).Run()
}
