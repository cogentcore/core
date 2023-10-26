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
	"goki.dev/goosi"
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

	goosi.ZoomFactor = 2

	gi.SetAppName("widgets")
	gi.SetAppAbout(`This is a demo of the main widgets and general functionality of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://goki.dev/gi/v2/blob/master/examples/widgets/README.md">README</a> page for this example app has lots of further info.</p>`)

	sc := gi.NewScene("widgets").SetTitle("GoGi Widgets Demo")

	tb := gi.NewToolbar(sc, "tbar")
	tb.SetStretchMaxWidth()
	gi.NewButton(tb).SetText("Button 1").SetData(1).
		OnClick(func(e events.Event) {
			fmt.Println("Toolbar Button 1")
			gi.NewSnackbar(tb, gi.SnackbarOpts{
				Text:   "Something went wrong!",
				Button: "Try again",
				ButtonOnClick: func(bt *gi.Button) {
					fmt.Println("got snackbar try again event")
				},
				Icon: icons.Close,
				IconOnClick: func(bt *gi.Button) {
					fmt.Println("got snackbar close icon event")
				},
			}).Run()
		})
	gi.NewButton(tb).SetText("Button 2").SetData(2).
		OnClick(func(e events.Event) {
			fmt.Println("Toolbar Button 2")
		})

	trow := gi.NewLayout(sc, "trow").
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
		Style(func(s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpaceNormal
			s.Text.Align = styles.AlignCenter
			s.Text.AlignV = styles.AlignCenter
			s.Font.Family = "Times New Roman, serif"
			// s.Font.Size = units.Dp(24) // todo: "x-large"?
			// s.Text.LineHeight = units.Em(1.5)
		})

	//////////////////////////////////////////
	//      Buttons

	gi.NewSpace(sc, "blspc")
	gi.NewLabel(sc, "blab").
		SetText("Buttons:")

	brow := gi.NewLayout(sc, "brow").
		SetLayout(gi.LayoutHoriz).SetSpacing(units.Em(1))

	b1 := gi.NewButton(brow, "button1").
		SetIcon(icons.OpenInNew).
		SetTooltip("press this <i>button</i> to pop up a dialog box").
		Style(func(s *styles.Style) {
			s.Width = units.Em(1.5)
			s.Height = units.Em(1.5)
		}).(*gi.Button)

	b1.OnClick(func(e events.Event) {
		fmt.Printf("Button1 clicked\n")
		gi.NewDialog(gi.NewScene("dlg"), b1).
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
		SetText("Open GoGiEditor")
	button2.SetTooltip("This button will open the GoGi GUI editor where you can edit this very GUI and see it update dynamically as you change things")
	button2.OnClick(func(e events.Event) {
		// gi.PromptDialog(button2, gi.DlgOpts{Title: "Look Ok?", Prompt: "Does this look ok?", Ok: true, Cancel: true}, button2, func(dlg *gi.DialogStage) {
		// 	fmt.Println("dialog looks OK:", dlg.Accepted)
		// }).Run()
		// gi.ChoiceDialog(button2, gi.DlgOpts{Title: "Which One?", Prompt: "What is your choice?"}, []string{"Ok", "Option1", "Option2", "Cancel"}, func(dlg *gi.DialogStage) {
		// 	fmt.Println("choice option:", dlg.Data.(int), "accepted:", dlg.Accepted)
		// }).Run()
		gi.StringPromptDialog(button2, gi.DlgOpts{Title: "What is it?", Prompt: "Please enter your response:", Ok: true, Cancel: true}, "", "Enter string here...", func(dlg *gi.Dialog) {
			fmt.Println("string entered:", dlg.Data.(string), "accepted:", dlg.Accepted)
		}).Run()
	})

	toggle := gi.NewSwitch(brow).SetText("Toggle")
	toggle.OnChange(func(e events.Event) {
		fmt.Println("toggled", toggle.StateIs(states.Checked))
	})

	mb := gi.NewButton(brow).SetText("Menu Button")
	mb.SetTooltip("Press this button to pull up a nested menu of buttons")

	mb.Menu = func(m *gi.Scene) {
		m1 := gi.NewButton(m).SetText("Menu Item 1").SetIcon(icons.Save).SetShortcut("Shift+Control+1").SetData(1)
		m1.SetTooltip("A standard menu item with an icon").
			OnClick(func(e events.Event) {
				fmt.Println("Received menu action with data", m1.Data)
			})

		m2 := gi.NewButton(m).SetText("Menu Item 2").SetIcon(icons.FileOpen).SetData(2)
		m2.SetTooltip("A menu item with an icon and a sub menu")

		m2.Menu = func(m *gi.Scene) {
			sm2 := gi.NewButton(m).SetText("Sub Menu Item 2").SetIcon(icons.InstallDesktop).SetData(2.1)
			sm2.SetTooltip("A sub menu item with an icon").
				OnClick(func(e events.Event) {
					fmt.Println("Received menu action with data", sm2.Data)
				})
		}

		gi.NewSeparator(m)

		m3 := gi.NewButton(m).SetText("Menu Item 3").SetIcon(icons.Favorite).SetShortcut("Control+3").SetData(3)
		m3.SetTooltip("A standard menu item with an icon, below a separator").
			OnClick(func(e events.Event) {
				fmt.Println("Received menu action with data", m3.Data)
			})
	}

	//////////////////////////////////////////
	//      Sliders

	gi.NewSpace(sc, "slspc")
	gi.NewLabel(gi.NewLayout(sc, "slrow").SetLayout(gi.LayoutHoriz), "slab").
		SetText("Sliders:")

	srow := gi.NewLayout(sc, "srow").SetLayout(gi.LayoutHoriz).
		SetSpacing(units.Ex(2)).
		SetStretchMaxWidth().
		Style(func(s *styles.Style) {
			s.AlignH = styles.AlignLeft
		})

	slider0 := gi.NewSlider(srow).
		SetDim(mat32.X).
		SetValue(0.5).
		SetSnap(true).
		SetTracking(true).
		SetIcon(icons.RadioButtonChecked)
	slider0.OnChange(func(e events.Event) {
		fmt.Println("slider0", slider0.Value)
	})

	slider1 := gi.NewSlider(srow).
		SetDim(mat32.Y).
		SetTracking(true).
		SetValue(0.5)
	slider1.OnChange(func(e events.Event) {
		fmt.Println("slider1", slider1.Value)
	})

	scroll0 := gi.NewSlider(srow).
		SetType(gi.SliderScrollbar).
		SetDim(mat32.X).
		SetThumbValue(0.25).
		SetValue(0.25).
		SetSnap(true).
		SetTracking(true)
	scroll0.Style(func(s *styles.Style) {
		s.MaxHeight.SetDp(12)
	})
	scroll0.OnChange(func(e events.Event) {
		fmt.Println("scroll0", scroll0.Value)
	})

	scroll1 := gi.NewSlider(srow).
		SetType(gi.SliderScrollbar).
		SetDim(mat32.Y).
		SetThumbValue(10).
		SetValue(0).
		SetMax(3000).
		SetTracking(true).
		SetStep(1).
		SetPageStep(10)
	scroll1.Style(func(s *styles.Style) {
		s.MaxWidth = units.Dp(16)
	})
	scroll1.OnChange(func(e events.Event) {
		fmt.Println("scroll1", scroll1.Value)
	})

	//////////////////////////////////////////
	//      Text Widgets

	gi.NewSpace(sc, "tlspc")
	gi.NewLabel(gi.NewLayout(sc, "txlrow").SetLayout(gi.LayoutHoriz), "txlab").
		SetText("Text Widgets:")

	txrow := gi.NewLayout(sc, "txrow").SetLayout(gi.LayoutHoriz).
		SetSpacing(units.Ex(2)).
		SetStretchMaxWidth()

	edit1 := gi.NewTextField(txrow, "edit1").
		SetPlaceholder("Enter text here...").
		AddClearButton().
		// SetTypePassword().
		Style(func(s *styles.Style) {
			s.SetMinPrefWidth(units.Em(20))
		}).(*gi.TextField)
	edit1.OnChange(func(e events.Event) {
		fmt.Println("Text:", edit1.Text())
	})

	sb := gi.NewSpinner(txrow).SetMax(1000).SetMin(-1000).SetStep(5)
	sb.OnChange(func(e events.Event) {
		fmt.Println("spinbox value changed to", sb.Value)
	})

	ch := gi.NewChooser(txrow).SetType(gi.ChooserOutlined).SetEditable(true).
		ItemsFromTypes(gti.AllEmbeddersOf(gi.WidgetBaseType), true, true, 50)
	// ItemsFromEnum(gi.ButtonTypesN, true, 50)
	ch.OnChange(func(e events.Event) {
		fmt.Printf("ComboBox selected index: %d data: %v\n", ch.CurIndex, ch.CurVal)
	})

	//////////////////////////////////////////
	//      Main Menu

	/*  todo:

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "RenderWin"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Button)
	amen.Menu.AddAppMenu(win)

	// note: use KeyFunMenu* for standard shortcuts
	// Command in shortcuts is automatically translated into Control for
	// Linux, Windows or Meta for MacOS
	fmen := win.MainMenu.ChildByName("File", 0).(*gi.Button)
	fmen.Menu.AddButton(gi.ActOpts{Label: "New", ShortcutKey: gi.KeyFunMenuNew},
		rec.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:New menu button triggered\n")
		})
	fmen.Menu.AddButton(gi.ActOpts{Label: "Open", ShortcutKey: gi.KeyFunMenuOpen},
		rec.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:Open menu button triggered\n")
		})
	fmen.Menu.AddButton(gi.ActOpts{Label: "Save", ShortcutKey: gi.KeyFunMenuSave},
		rec.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:Save menu button triggered\n")
		})
	fmen.Menu.AddButton(gi.ActOpts{Label: "Save As..", ShortcutKey: gi.KeyFunMenuSaveAs},
		rec.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:SaveAs menu button triggered\n")
		})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddButton(gi.ActOpts{Label: "Close RenderWin", ShortcutKey: gi.KeyFunWinClose},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			win.CloseReq()
		})

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Button)
	emen.Menu.AddCopyCutPaste(win)

	inQuitPrompt := false
	gi.SetQuitReqFunc(func() {
		if inQuitPrompt {
			return
		}
		inQuitPrompt = true
		gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Quit?",
			Prompt: "Are you <i>sure</i> you want to quit?"}, Ok: true, Cancel: true,
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
			Prompt: "Are you <i>sure</i> you want to close the window?  This will Quit the App as well."}, Ok: true, Cancel: true,
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

	gi.NewWindow(sc).Run().Wait()
}
