// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package greasi

import (
	"github.com/iancoleman/strcase"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/grease"
	"goki.dev/grog"
)

// GUI starts the GUI for the given Grease app, which must be passed as
// a pointer. It should typically not be called by end-user code; see [Run].

func GUI[T any](opts *grease.Options, cfg T, cmds ...*grease.Cmd[T]) {
	gimain.Run(func() {
		App(opts, cfg, cmds...)
	})
}

// App does runs the GUI. It should be called on the main thread.
// It should typically not be called by end-user code; see [Run].
func App[T any](opts *grease.Options, cfg T, cmds ...*grease.Cmd[T]) {
	gi.SetAppName(opts.AppName)
	gi.SetAppAbout(opts.AppAbout)

	sc := gi.StageScene(opts.AppName).SetTitle(opts.AppTitle)

	tb := gi.NewToolBar(sc)
	for _, cmd := range cmds {
		cmd := cmd
		if cmd.Name == "gui" { // we are already in GUI so that command is irrelevant
			continue
		}
		tb.AddButton(gi.ActOpts{
			Name:    cmd.Name,
			Label:   strcase.ToCamel(cmd.Name),
			Tooltip: cmd.Doc,
		}, func(bt *gi.Button) {
			err := cmd.Func(cfg)
			if err != nil {
				grog.PrintlnError(err)
			}
		})
	}

	sv := giv.NewStructView(sc)
	sv.SetStruct(cfg)

	gi.NewWindow(sc).Run().Wait()

	// TODO: add main menu
	/*

		// Main Menu

		appnm := gi.AppName()
		mmen := win.MainMenu
		mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

		amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Button)
		amen.Menu.AddAppMenu(win)

		// TODO: finish these functions
		fmen := win.MainMenu.ChildByName("File", 0).(*gi.Button)
		fmen.Menu.AddAction(gi.ActOpts{Label: "New", ShortcutKey: gi.KeyFunMenuNew},
			fmen.This(), func(recv, send ki.Ki, sig int64, data any) {

			})
		fmen.Menu.AddAction(gi.ActOpts{Label: "Open", ShortcutKey: gi.KeyFunMenuOpen},
			fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
			})
		fmen.Menu.AddAction(gi.ActOpts{Label: "Save", ShortcutKey: gi.KeyFunMenuSave},
			fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
				if grease.ConfigFile != "" {
					grease.Save(cfg, grease.ConfigFile)
				}
			})
		fmen.Menu.AddAction(gi.ActOpts{Label: "Save As..", ShortcutKey: gi.KeyFunMenuSaveAs},
			fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
			})
		fmen.Menu.AddSeparator("csep")
		fmen.Menu.AddAction(gi.ActOpts{Label: "Close Window", ShortcutKey: gi.KeyFunWinClose},
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				win.CloseReq()
			})

		emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Button)
		emen.Menu.AddCopyCutPaste(win)

		vp.UpdateEndNoSig(updt)
		win.StartEventLoop()
	*/
}
