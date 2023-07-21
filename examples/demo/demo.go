// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/icons"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
)

func main() {
	gimain.Main(mainrun)
}

func mainrun() {
	gi.SetAppName("gogi-demo")
	gi.SetAppAbout("The GoGi Demo demonstrates the various features of the GoGi 2D and 3D Go GUI framework.")

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	win := gi.NewMainWindow("gogi-demo", "The GoGi Demo", 1024, 768)
	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.StyleFunc = func() {
		mfr.Spacing.SetEx(1)
		mfr.Style.Layout.Padding.Set(units.Px(8))
	}

	title := gi.AddNewLabel(mfr, "title", "The GoGi Demo")
	title.Type = gi.LabelH1

	desc := gi.AddNewLabel(mfr, "desc", "A demonstration of the <i>various</i> features of the <u>GoGi</u> 2D and 3D Go GUI <b>framework.</b>")
	desc.Type = gi.LabelP

	bdesc := gi.AddNewLabel(mfr, "bdesc", "Buttons")
	bdesc.Type = gi.LabelH3

	brow := gi.AddNewLayout(mfr, "brow", gi.LayoutHoriz)
	brow.StyleFunc = func() {
		brow.Spacing.SetEx(1)
		brow.Style.Layout.MaxWidth.SetPx(-1)
	}

	bpri := gi.AddNewButton(brow, "buttonPrimary")
	bpri.Text = "Primary Button"
	bpri.Type = gi.ButtonPrimary
	bpri.Icon = icons.FastForward

	bsec := gi.AddNewButton(brow, "buttonSecondary")
	bsec.Text = "Secondary Button"
	bsec.Type = gi.ButtonSecondary
	bsec.Icon = icons.Settings

	bdef := gi.AddNewButton(brow, "buttonDefault")
	bdef.Text = "Default Button"

	idesc := gi.AddNewLabel(mfr, "idesc", "Inputs")
	idesc.Type = gi.LabelH3

	irow := gi.AddNewLayout(mfr, "irow", gi.LayoutHorizFlow)
	irow.StyleFunc = func() {
		irow.Spacing.SetEx(1)
		irow.Style.Layout.MaxWidth.SetPx(-1)
	}

	check := gi.AddNewCheckBox(irow, "check")
	check.Text = "Checkbox"

	tfield := gi.AddNewTextField(irow, "tfield")
	tfield.Placeholder = "Text Field"

	sbox := gi.AddNewSpinBox(irow, "sbox")
	sbox.Value = 0.5

	cbox := gi.AddNewComboBox(irow, "cbox")
	cbox.Text = "Select an option"
	cbox.Items = []any{"Option 1", "Option 2", "Option 3"}

	// tview := giv.AddNewTextView(mfr, "tview")
	// tview.Placeholder = "Text View"

	// Main Menu

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)

	fmen := win.MainMenu.ChildByName("File", 0).(*gi.Action)
	fmen.Menu.AddAction(gi.ActOpts{Label: "New", ShortcutKey: gi.KeyFunMenuNew},
		rec.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:New menu action triggered\n")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Open", ShortcutKey: gi.KeyFunMenuOpen},
		rec.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:Open menu action triggered\n")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Save", ShortcutKey: gi.KeyFunMenuSave},
		rec.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:Save menu action triggered\n")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Save As..", ShortcutKey: gi.KeyFunMenuSaveAs},
		rec.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:SaveAs menu action triggered\n")
		})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddAction(gi.ActOpts{Label: "Close Window", ShortcutKey: gi.KeyFunWinClose},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			win.CloseReq()
		})

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu.AddCopyCutPaste(win)

	inQuitPrompt := false
	gi.SetQuitReqFunc(func() {
		if inQuitPrompt {
			return
		}
		inQuitPrompt = true
		gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Quit?",
			Prompt: "Are you <i>sure</i> you want to quit?"}, gi.AddOk, gi.AddCancel,
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				if sig == int64(gi.DialogAccepted) {
					gi.Quit()
				} else {
					inQuitPrompt = false
				}
			})
	})

	gi.SetQuitCleanFunc(func() {
		fmt.Printf("Doing final Quit cleanup here..\n")
	})

	inClosePrompt := false
	win.SetCloseReqFunc(func(w *gi.Window) {
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Close Window?",
			Prompt: "Are you <i>sure</i> you want to close the window?  This will Quit the App as well."}, gi.AddOk, gi.AddCancel,
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				if sig == int64(gi.DialogAccepted) {
					gi.Quit()
				} else {
					inClosePrompt = false
				}
			})
	})

	win.SetCloseCleanFunc(func(w *gi.Window) {
		fmt.Printf("Doing final Close cleanup here..\n")
	})

	win.MainMenuUpdated()
	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}
