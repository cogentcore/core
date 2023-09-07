// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package greasi

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/grease"
	"goki.dev/ki/v2/ki"
)

// GUI starts the GUI for the given
// Grease app, which must be passed as
// a pointer.
func GUI(app any) {
	gimain.Main(func() {
		mainrun(app)
	})
}

func mainrun(app any) {
	gi.SetAppName(grease.AppName)
	gi.SetAppAbout(grease.AppAbout)

	win := gi.NewMainWindow(grease.AppName, grease.AppTitle, 1024, 768)
	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	mfr := win.SetMainFrame()

	sv := giv.AddNewStructView(mfr, "sv")
	sv.Viewport = vp
	sv.SetStruct(app)

	// Main Menu

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)

	fmen := win.MainMenu.ChildByName("File", 0).(*gi.Action)
	fmen.Menu.AddAction(gi.ActOpts{Label: "New", ShortcutKey: gi.KeyFunMenuNew},
		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Println("File:New menu action triggered")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Open", ShortcutKey: gi.KeyFunMenuOpen},
		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Println("File:Open menu action triggered")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Save", ShortcutKey: gi.KeyFunMenuSave},
		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Println("File:Save menu action triggered")
			grease.Save(app, grease.ConfigFile)
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Save As..", ShortcutKey: gi.KeyFunMenuSaveAs},
		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Println("File:SaveAs menu action triggered")
		})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddAction(gi.ActOpts{Label: "Close Window", ShortcutKey: gi.KeyFunWinClose},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			win.CloseReq()
		})

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu.AddCopyCutPaste(win)

	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}
