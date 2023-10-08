// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/gti"
	"goki.dev/icons"
	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

func app() {
	// turn these on to see a traces of various stages of processing..
	// gi.UpdateTrace = true
	// gi.RenderTrace = true
	// gi.LayoutTrace = true
	// gi.WinEventTrace = true
	// gi.WinRenderTrace = true
	// gi.EventTrace = true
	// gi.KeyEventTrace = true
	// events.TraceEventCompression = true
	// events.TraceWindowPaint = true

	// goosi.ZoomFactor = 2

	gi.SetAppName("widgets")
	gi.SetAppAbout(`This is a demo of the main widgets and general functionality of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://goki.dev/gi/v2/blob/master/examples/widgets/README.md">README</a> page for this example app has lots of further info.</p>`)

	scene := gi.StageScene("widgets").SetTitle("GoGi Widgets Demo")

	tbar := gi.NewToolBar(scene, "tbar").SetStretchMaxWidth().(*gi.ToolBar)
	tbar.AddAction(gi.ActOpts{Label: "Action 1", Data: 1}, func(act *gi.Action) {
		fmt.Println("Toolbar Action 1")
	})

	trow := gi.NewLayout(scene, "trow").
		SetLayout(gi.LayoutHoriz).SetStretchMaxWidth()

	giedsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunGoGiEditor)
	prsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunPrefs)

	gi.NewLabel(trow, "title").SetText(
		`This is a <b>demonstration</b> of the
<span style="color:red">various</span> <a href="https://goki.dev/gi/v2">GoGi</a> <i>Widgets</i><br>
<small>Shortcuts: <kbd>` + string(prsc) + `</kbd> = Preferences,
<kbd>` + string(giedsc) + `</kbd> = Editor, <kbd>Ctrl/Cmd +/-</kbd> = zoom</small><br>
See <a href="https://goki.dev/gi/v2/blob/master/examples/widgets/README.md">README</a> for detailed info and things to try.`).
		SetType(gi.LabelHeadlineSmall).
		SetStretchMax().
		AddStyles(func(s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpaceNormal
			s.Text.Align = styles.AlignCenter
			s.Text.AlignV = styles.AlignCenter
			s.Font.Family = "Times New Roman, serif"
			// s.Font.Size = units.Dp(24) // todo: "x-large"?
			// s.Text.LineHeight = units.Em(1.5)
		})

	//////////////////////////////////////////
	//      Buttons

	gi.NewSpace(scene, "blspc")
	gi.NewLabel(gi.NewLayout(scene, "blrow").SetLayout(gi.LayoutHoriz), "blab").
		SetText("Buttons:").SetSelectable()

	brow := gi.NewLayout(scene, "brow").
		SetLayout(gi.LayoutHoriz).SetSpacing(units.Em(1))

	b1 := gi.NewButton(brow, "button1").
		SetIcon(icons.OpenInNew).
		SetTooltip("press this <i>button</i> to pop up a dialog box").
		AddStyles(func(s *styles.Style) {
			s.Width = units.Em(1.5)
			s.Height = units.Em(1.5)
		}).(*gi.Button)

	b1.On(events.Click, func(e events.Event) {
		fmt.Printf("Button1 clicked\n")
		gi.NewDialog(gi.StageScene("dlg"), b1).
			Title("Test Dialog").Prompt("This is a prompt").
			Modal(true).NewWindow(true).OkCancel().Run()

		// gi.StringPromptDialog(vp, "", "Enter value here..",
		// 	gi.DlgOpts{Title: "Button1 Dialog", Prompt: "This is a string prompt dialog!  Various specific types of dialogs are available."},
		// 	rec.This(), func(recv, send ki.Ki, sig int64, data any) {
		// 		dlg := send.(*gi.Dialog)
		// 		if sig == int64(gi.DialogAccepted) {
		// 			val := gi.StringPromptDialogValue(dlg)
		// 			fmt.Printf("got string value: %v\n", val)
		// 		}
		// 	})
	})

	button2 := gi.NewButton(brow, "button2").
		SetText("Open GoGiEditor").
		SetTooltip("This button will open the GoGi GUI editor where you can edit this very GUI and see it update dynamically as you change things").(*gi.Button)
	_ = button2
	button2.On(events.Click, func(e events.Event) {
		// gi.PromptDialog(button2, gi.DlgOpts{Title: "Look Ok?", Prompt: "Does this look ok?", Ok: true, Cancel: true}, button2, func(dlg *gi.DialogStage) {
		// 	fmt.Println("dialog looks OK:", dlg.Accepted)
		// }).Run()
		// gi.ChoiceDialog(button2, gi.DlgOpts{Title: "Which One?", Prompt: "What is your choice?"}, []string{"Ok", "Option1", "Option2", "Cancel"}, func(dlg *gi.DialogStage) {
		// 	fmt.Println("choice option:", dlg.Data.(int), "accepted:", dlg.Accepted)
		// }).Run()
		gi.StringPromptDialog(button2, gi.DlgOpts{Title: "What is it?", Prompt: "Please enter your response:", Ok: true, Cancel: true}, "", "Enter string here...", func(dlg *gi.DialogStage) {
			fmt.Println("string entered:", dlg.Data.(string), "accepted:", dlg.Accepted)
		}).Run()
	})

	checkbox := gi.NewCheckBox(brow, "checkbox").
		SetText("Toggle")
	checkbox.On(events.Click, func(e events.Event) {
		fmt.Println("toggled", checkbox.StateIs(states.Checked))
	})

	_ = checkbox

	mb1 := gi.NewButton(brow, "menubutton1").
		SetText("Menu Button").(*gi.Button)

	mb1.Menu.AddAction(gi.ActOpts{Label: "Menu Item 1", Shortcut: "Shift+Control+1", Data: 1}, func(act *gi.Action) {
		fmt.Println(act.Name(), act.Data)
	})
	mi2 := mb1.Menu.AddAction(gi.ActOpts{Label: "Menu Item 2", Data: 2}, nil)
	_ = mi2
	mi2.Menu.AddAction(gi.ActOpts{Label: "Sub Menu Item 2", Data: 2.1}, func(act *gi.Action) {
		fmt.Println(act.Text, act.Data)
	})
	mb1.Menu.AddSeparator("sep1")
	mb1.Menu.AddAction(gi.ActOpts{Label: "Menu Item 3", Shortcut: "Control+3", Data: 3}, func(act *gi.Action) {
		fmt.Println(act.Text, act.Data)
	})

	//////////////////////////////////////////
	//      Sliders

	gi.NewSpace(scene, "slspc")
	gi.NewLabel(gi.NewLayout(scene, "slrow").SetLayout(gi.LayoutHoriz), "slab").
		SetText("Sliders:")

	srow := gi.NewLayout(scene, "srow").SetLayout(gi.LayoutHoriz).
		SetSpacing(units.Ex(2)).
		SetStretchMaxWidth().
		AddStyles(func(s *styles.Style) {
			s.AlignH = styles.AlignLeft
		})

	slider1 := gi.NewSlider(srow, "slider1").
		SetDim(mat32.X).
		SetValue(0.5).
		SetSnap(true).
		SetTracking(true).
		SetIcon(icons.RadioButtonChecked)
	slider1.SetMinPrefWidth(units.Em(20)).SetMinPrefHeight(units.Em(2)).
		On(events.Change, func(e events.Event) {
			fmt.Println("slider1", slider1.Value)
		})

	slider2 := gi.NewSlider(srow, "slider2").
		SetDim(mat32.Y).
		SetTracking(true).
		SetValue(0.5)
	slider2.SetStretchMaxHeight().SetMinPrefHeight(units.Em(10)).SetMinPrefWidth(units.Em(1)).
		On(events.Change, func(e events.Event) {
			fmt.Println("slider2", slider2.Value)
		})

	scrollbar1 := gi.NewScrollBar(srow, "scrollbar1").
		SetDim(mat32.X).
		SetThumbValue(0.25).
		SetValue(0.25).
		SetSnap(true).
		SetTracking(true)
	scrollbar1.SetMinPrefWidth(units.Em(20)).SetMinPrefHeight(units.Em(1)).
		On(events.Change, func(e events.Event) {
			fmt.Println("scroll1", scrollbar1.Value)
		})

	scrollbar2 := gi.NewScrollBar(srow, "scrollbar2").
		SetDim(mat32.Y).
		SetThumbValue(10).
		SetValue(0).
		SetMax(3000).
		SetTracking(true).
		SetStep(1).
		SetPageStep(10)
	scrollbar2.SetMinPrefHeight(units.Em(10)).SetMinPrefWidth(units.Em(1)).SetStretchMaxHeight().
		On(events.Change, func(e events.Event) {
			fmt.Println("scroll2", scrollbar2.Value)
		})

	//////////////////////////////////////////
	//      Text Widgets

	gi.NewSpace(scene, "tlspc")
	gi.NewLabel(gi.NewLayout(scene, "txlrow").SetLayout(gi.LayoutHoriz), "txlab").
		SetText("Text Widgets:")

	txrow := gi.NewLayout(scene, "txrow").SetLayout(gi.LayoutHoriz).
		SetSpacing(units.Ex(2)).
		SetStretchMaxWidth()

	edit1 := gi.NewTextField(txrow, "edit1").
		SetPlaceholder("Enter text here...").
		AddClearAction().
		// SetTypePassword().
		AddStyles(func(s *styles.Style) {
			s.SetMinPrefWidth(units.Em(20))
		}).(*gi.TextField)
	edit1.On(events.Change, func(e events.Event) {
		fmt.Println("Text:", edit1.Text())
	})

	sb := gi.NewSpinBox(txrow, "spin")
	sb.SetMax(255)
	sb.Step = 1
	sb.Format = "%#X"
	sb.SetMin(0)
	// sb.SpinBoxSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
	// 	fmt.Printf("SpinBox %v value changed: %v\n", send.Name(), data)
	// })

	cb := gi.NewComboBox(txrow, "combo").
		ItemsFromTypes(gti.AllEmbeddersOf(gi.WidgetBaseType), true, true, 50)
	// ItemsFromEnum(gi.ButtonTypesN, true, 50)
	cb.On(events.Change, func(e events.Event) {
		fmt.Printf("ComboBox selected index: %d data: %v\n", cb.CurIndex, cb.CurVal)
	})

	//////////////////////////////////////////
	//      Main Menu

	/*  todo:

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "RenderWin"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)

	// note: use KeyFunMenu* for standard shortcuts
	// Command in shortcuts is automatically translated into Control for
	// Linux, RenderWins or Meta for MacOS
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
	fmen.Menu.AddAction(gi.ActOpts{Label: "Close RenderWin", ShortcutKey: gi.KeyFunWinClose},
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
	win.SetCloseReqFunc(func(w *gi.RenderWin) {
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Close RenderWin?",
			Prompt: "Are you <i>sure</i> you want to close the window?  This will Quit the App as well."}, gi.AddOk, gi.AddCancel,
			win.This(), func(recv, send ki.Ki, sig int64, data any) {
				if sig == int64(gi.DialogAccepted) {
					gi.Quit()
				} else {
					inClosePrompt = false
				}
			})
	})

	win.SetCloseCleanFunc(func(w *gi.RenderWin) {
		fmt.Printf("Doing final Close cleanup here..\n")
	})

	win.MainMenuUpdated()
	*/

	gi.NewWindow(scene).Run().Wait()
}
