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
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

func main() { gimain.Main(mainrun) }

func mainrun() {
	width := 1024
	height := 768

	// turn these on to see a traces of various stages of processing..
	// gi.Update2DTrace = true
	// gi.RenderTrace = true
	// gi.LayoutTrace = true
	// ki.SignalTrace = true
	// gi.WinEventTrace = true
	// gi.EventTrace = true
	// gi.KeyEventTrace = true

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	gi.SetAppName("widgets")
	gi.SetAppAbout(`This is a demo of the main widgets and general functionality of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://goki.dev/gi/v2/blob/master/examples/widgets/README.md">README</a> page for this example app has lots of further info.</p>`)

	scene := gi.NewScene("widgets").SetTitle("GoGi Widgets Demo")
	frame := &scene.Frame // todo: scene will be the frame

	trow := gi.NewLayout(frame, "trow").
		SetLayout(gi.LayoutHoriz).SetStretchMaxWidth()

	giedsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunGoGiEditor)
	prsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunPrefs)

	gi.NewLabel(trow, "title").SetText(
		`This is a <b>demonstration</b> of the
<span style="color:red">various</span> <a href="https://goki.dev/gi/v2">GoGi</a> <i>Widgets</i><br>
<large>Shortcuts: <kbd>` + string(prsc) + `</kbd> = Preferences,
<kbd>` + string(giedsc) + `</kbd> = Editor, <kbd>Ctrl/Cmd +/-</kbd> = zoom</large><br>
See <a href="https://goki.dev/gi/v2/blob/master/examples/widgets/README.md">README</a> for detailed info and things to try.`).
		SetStretchMax().
		SetStyle(func(w *gi.WidgetBase, s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpaceNormal
			s.Text.Align = styles.AlignCenter
			s.Text.AlignV = styles.AlignCenter
			s.Font.Family = "Times New Roman, serif"
			s.Font.Size = units.Pt(24) // todo: "x-large"?
			// s.Font.SetSize(styles.XLarge)
			s.Text.LineHeight = units.Em(1.5)
		})

	//////////////////////////////////////////
	//      Buttons

	gi.NewSpace(frame, "blspc")
	gi.NewLabel(gi.NewLayout(frame, "blrow").SetLayout(gi.LayoutHoriz), "blab").
		SetText("Buttons:").SetSelectable()

	brow := gi.NewLayout(frame, "brow").
		SetLayout(gi.LayoutHoriz).SetSpacing(units.Em(1))

	gi.NewButton(brow, "button1").
		SetIcon(icons.OpenInNew).
		SetTooltip("press this <i>button</i> to pop up a dialog box").
		SetStyle(func(w *gi.WidgetBase, s *styles.Style) {
			s.Width = units.Em(1.5)
			s.Height = units.Em(1.5)
		}).(*gi.Button).
		OnClicked(func() {
			fmt.Printf("Button1 clicked\n")
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

	// this is the "full strength" general purpose signaling framework
	button2.ButtonSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
		fmt.Printf("Received button signal: %v from button: %v\n", gi.ButtonSignals(sig), send.Name())
		if sig == int64(gi.ButtonClicked) {
			// giv.GoGiEditorDialog(win)
		}
	})

	// button2.StyleFunc = func() {
	// 	fmt.Println(button2.State)
	// 	switch button2.State {
	// 	case gi.ButtonHover:
	// 		button2.Style.BackgroundColor.SetSolid(styles.MustColorFromName("darkblue"))
	// 	default:
	// 		button2.Style.Border.Color.Set(styles.Transparent)
	// 		button2.Style.Border.Width.Set(units.Px(2))
	// 		button2.Style.Border.Radius.Set(units.Px(10))
	// 		button2.Style.BackgroundColor.SetSolid(styles.MustColorFromName("blue"))
	// 		button2.Style.Color.SetColor(styles.MustColorFromName("white"))
	// 		button2.Style.Padding.Set(units.Px(10), units.Px(5))
	// 		button2.Style.Height = units.Px(50)
	// 	}
	// }
	// button2.SetProp("border-color", styles.NewSides[styles.Color](styles.MustColorFromName("green"), styles.MustColorFromName("red"), styles.MustColorFromName("blue"), styles.MustColorFromName("orange")))
	// button2.SetProp("border-width", "2px 4px 6px 8px")
	// button2.SetProp("border-radius", "0 2 6 10")

	checkbox := gi.NewCheckBox(brow, "checkbox").
		SetText("Toggle").(*gi.CheckBox)

	// todo: need convenient OnToggled
	checkbox.ButtonSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.ButtonToggled) {
			fmt.Printf("Checkbox toggled: %v\n", checkbox.StateIs(states.Checked))
		}
	})

	// note: receiver for menu items with shortcuts must be a Node2D or RenderWin
	mb1 := gi.NewButton(brow, "menubutton1").
		SetText("Menu Button").(*gi.Button)
	mb1.Menu.AddAction(gi.ActOpts{Label: "Menu Item 1", Shortcut: "Shift+Control+1", Data: 1},
		mb1.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mi2 := mb1.Menu.AddAction(gi.ActOpts{Label: "Menu Item 2", Data: 2}, nil, nil)

	mi2.Menu.AddAction(gi.ActOpts{Label: "Sub Menu Item 2", Data: 2.1},
		mb1.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mb1.Menu.AddSeparator("sep1")

	mb1.Menu.AddAction(gi.ActOpts{Label: "Menu Item 3", Shortcut: "Control+3", Data: 3},
		mb1.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	//////////////////////////////////////////
	//      Sliders

	gi.NewSpace(frame, "slspc")
	gi.NewLabel(gi.NewLayout(frame, "slrow").SetLayout(gi.LayoutHoriz), "slab").
		SetText("Sliders:")

	srow := gi.NewLayout(frame, "srow").SetLayout(gi.LayoutHoriz).
		SetSpacing(units.Ex(2)).
		SetStretchMaxWidth().
		SetStyle(func(w *gi.WidgetBase, s *styles.Style) {
			s.AlignH = styles.AlignLeft
		})

	// todo: need Slider interface with Set methods
	slider1 := gi.NewSlider(srow, "slider1")
	slider1.Dim = mat32.X
	slider1.SetProp(":value", ki.Props{"background-color": "red"})
	slider1.SetMinPrefWidth(units.Em(20))
	slider1.SetMinPrefHeight(units.Em(2))
	slider1.SetValue(0.5)
	slider1.Snap = true
	slider1.Tracking = true
	slider1.Icon = icons.RadioButtonChecked

	slider2 := gi.NewSlider(srow, "slider2")
	slider2.Dim = mat32.Y
	slider2.SetMinPrefHeight(units.Em(10))
	slider2.SetMinPrefWidth(units.Em(1))
	slider2.SetStretchMaxHeight()
	slider2.SetValue(0.5)

	slider1.SliderSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig != int64(gi.SliderMoved) {
			fmt.Printf("Received slider signal: %v from slider: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
		}
	})

	slider2.SliderSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig != int64(gi.SliderMoved) {
			fmt.Printf("Received slider signal: %v from slider: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
		}
	})

	scrollbar1 := gi.NewScrollBar(srow, "scrollbar1")
	scrollbar1.Dim = mat32.X
	scrollbar1.Class = "hslides"
	scrollbar1.SetMinPrefWidth(units.Em(20))
	scrollbar1.SetMinPrefHeight(units.Em(1))
	scrollbar1.SetThumbValue(0.25)
	scrollbar1.SetValue(0.25)
	// scrollbar1.Snap = true
	// scrollbar1.Tracking = true
	scrollbar1.SliderSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig != int64(gi.SliderMoved) {
			fmt.Printf("Received scrollbar signal: %v from scrollbar: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
		}
	})

	scrollbar2 := gi.NewScrollBar(srow, "scrollbar2")
	scrollbar2.Dim = mat32.Y
	scrollbar2.SetMinPrefHeight(units.Em(10))
	scrollbar2.SetMinPrefWidth(units.Em(1))
	scrollbar2.SetStretchMaxHeight()
	scrollbar2.SetThumbValue(10)
	scrollbar2.SetValue(0)
	scrollbar2.Max = 3000
	scrollbar2.Tracking = true
	scrollbar2.Step = 1
	scrollbar2.PageStep = 10
	scrollbar2.SliderSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
		if sig == int64(gi.SliderValueChanged) { // typically this is the one you care about
			fmt.Printf("Received scrollbar signal: %v from scrollbar: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
		}
	})

	//////////////////////////////////////////
	//      Text Widgets

	gi.NewSpace(frame, "tlspc")
	gi.NewLabel(gi.NewLayout(frame, "txlrow").SetLayout(gi.LayoutHoriz), "txlab").
		SetText("Text Widgets:")

	txrow := gi.NewLayout(frame, "txrow").SetLayout(gi.LayoutHoriz).
		SetSpacing(units.Ex(2)).
		SetStretchMaxWidth()

	edit1 := gi.NewTextField(txrow, "edit1")
	edit1.Placeholder = "Enter text here..."
	// edit1.SetText("Edit this text")
	// edit1.SetProp("min-width", "20em")
	edit1.TextFieldSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
		estr := ""
		if rn, ok := data.([]rune); ok {
			estr = string(rn)
		} else if st, ok := data.(string); ok {
			estr = st
		}
		fmt.Printf("Received line edit signal: %v from edit: %v with data: %s\n", gi.TextFieldSignals(sig), send.Name(), estr)
	})
	// edit1.SetProp("inactive", true)

	sb := gi.NewSpinBox(txrow, "spin")
	sb.SetMax(255)
	sb.Step = 1
	sb.Format = "%#X"
	sb.SetMin(0)
	sb.SpinBoxSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
		fmt.Printf("SpinBox %v value changed: %v\n", send.Name(), data)
	})

	cb := gi.NewComboBox(txrow, "combo")
	// cb.ItemsFromTypes(kit.Types.AllImplementersOf(reflect.TypeOf((*gi.Node2D)(nil)).Elem(), false), true, true, 50)
	cb.ComboSig.Connect(rec.This(), func(recv, send ki.Ki, sig int64, data any) {
		fmt.Printf("ComboBox %v selected index: %v data: %v\n", send.Name(), sig, data)
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

	gi.NewWindow(scene).
		SetWidth(width).SetHeight(height).
		Run().Wait()
}
