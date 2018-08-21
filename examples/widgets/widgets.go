// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"reflect"

	"github.com/goki/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768

	// turn these on to see a traces of various stages of processing..
	// gi.Update2DTrace = true
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true
	// ki.SignalTrace = true

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	oswin.TheApp.SetName("widgets")
	oswin.TheApp.SetAbout(`This is a demo of the main widgets and general functionality of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://github.com/goki/gi/blob/master/examples/widgets/README.md">README</a> page for this example app has lots of further info.</p>`)

	win := gi.NewWindow2D("gogi-widgets-demo", "GoGi Widgets Demo", width, height, true) // true = pixel sizes

	icnm := "widget-wedge-down"

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	// style sheet
	var css = ki.Props{
		"button": ki.Props{
			"background-color": gi.Color{255, 240, 240, 255},
		},
		"#combo": ki.Props{
			"background-color": gi.Color{240, 255, 240, 255},
		},
		".hslides": ki.Props{
			"background-color": gi.Color{240, 225, 255, 255},
		},
		"kbd": ki.Props{
			"color": "blue",
		},
	}
	vp.CSS = css

	mfr := win.SetMainFrame()
	mfr.SetProp("spacing", units.NewValue(1, units.Px))
	// mfr.SetProp("background-color", "linear-gradient(to top, red, lighter-80)")
	// mfr.SetProp("background-color", "linear-gradient(to right, red, orange, yellow, green, blue, indigo, violet)")
	// mfr.SetProp("background-color", "linear-gradient(to right, rgba(255,0,0,0), rgba(255,0,0,1))")
	// mfr.SetProp("background-color", "radial-gradient(red, lighter-80)")

	trow := mfr.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutHoriz
	trow.SetStretchMaxWidth()

	giedsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunGoGiEditor)
	prsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunPrefs)

	trow.AddNewChild(gi.KiT_Stretch, "str1")
	title := trow.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.Text = `This is a <b>demonstration</b> of the
<span style="color:red">various</span> <a href="https://github.com/goki/gi">GoGi</a> <i>Widgets</i><br>
<large>Shortcuts: <kbd>` + prsc + `</kbd> = Preferences,
<kbd>` + giedsc + `</kbd> = Editor, <kbd>Ctrl/Cmd +/-</kbd> = zoom</large><br>
See <a href="https://github.com/goki/gi/blob/master/examples/widgets/README.md">README</a> for detailed info and things to try.`
	title.SetProp("text-align", gi.AlignCenter)
	title.SetProp("vertical-align", gi.AlignTop)
	title.SetProp("font-family", "Times New Roman, serif")
	title.SetProp("font-size", "x-large")
	// title.SetProp("letter-spacing", 2)
	title.SetProp("line-height", 1.5)
	trow.AddNewChild(gi.KiT_Stretch, "str2")

	//////////////////////////////////////////
	//      Buttons

	mfr.AddNewChild(gi.KiT_Space, "blspc")
	blrow := mfr.AddNewChild(gi.KiT_Layout, "blrow").(*gi.Layout)
	blab := blrow.AddNewChild(gi.KiT_Label, "blab").(*gi.Label)
	blab.SetProp("font-family", "Arial Unicode")
	blab.Text = "Buttons:"
	blab.Selectable = true

	brow := mfr.AddNewChild(gi.KiT_Layout, "brow").(*gi.Layout)
	brow.Lay = gi.LayoutHoriz
	brow.SetProp("spacing", units.NewValue(2, units.Ex))

	brow.SetProp("horizontal-align", gi.AlignLeft)
	// brow.SetProp("horizontal-align", gi.AlignJustify)
	brow.SetStretchMaxWidth()

	button1 := brow.AddNewChild(gi.KiT_Button, "button1").(*gi.Button)
	button1.SetProp("#icon", ki.Props{ // note: must come before SetIcon
		"width":  units.NewValue(1.5, units.Em),
		"height": units.NewValue(1.5, units.Em),
	})
	button1.Tooltip = "press this <i>button</i> to pop up a dialog box"

	button1.SetIcon(icnm)
	button1.ButtonSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received button signal: %v from button: %v\n", gi.ButtonSignals(sig), send.Name())
		if sig == int64(gi.ButtonClicked) { // note: 3 diff ButtonSig sig's possible -- important to check
			// vp.Win.Quit()
			gi.StringPromptDialog(vp, "Enter value here..", "Button1 Dialog", "This is a string prompt dialog!  Various specific types of dialogs are available.", nil, rec.This,
				func(recv, send ki.Ki, sig int64, data interface{}) {
					dlg := send.(*gi.Dialog)
					if sig == int64(gi.DialogAccepted) {
						val := gi.StringPromptDialogValue(dlg)
						fmt.Printf("got string value: %v\n", val)
					}
				})
		}
	})

	button2 := brow.AddNewChild(gi.KiT_Button, "button2").(*gi.Button)
	button2.SetText("Open GoGiEditor")
	// button2.SetProp("background-color", "#EDF")
	button2.Tooltip = "This button will open the GoGi GUI editor where you can edit this very GUI and see it update dynamically as you change things"
	button2.ButtonSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received button signal: %v from button: %v\n", gi.ButtonSignals(sig), send.Name())
		if sig == int64(gi.ButtonClicked) {
			giv.GoGiEditor(vp)
		}
	})

	checkbox := brow.AddNewChild(gi.KiT_CheckBox, "checkbox").(*gi.CheckBox)
	checkbox.Text = "Toggle"

	// note: receiver for menut items with shortcuts must be a Node2D or Window
	mb1 := brow.AddNewChild(gi.KiT_MenuButton, "menubutton1").(*gi.MenuButton)
	mb1.SetText("Menu Button")
	mb1.Menu.AddMenuText("Menu Item 1", "Shift+Control+1", win.This, 1, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
	})

	mi2 := mb1.Menu.AddMenuText("Menu Item 2", "", nil, 2, nil)

	mi2.Menu.AddMenuText("Sub Menu Item 2", "", win.This, 2.1, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
	})

	mb1.Menu.AddSeparator("sep1")

	mb1.Menu.AddMenuText("Menu Item 3", "Control+3", win.This, 3, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
	})

	//////////////////////////////////////////
	//      Sliders

	mfr.AddNewChild(gi.KiT_Space, "slspc")
	slrow := mfr.AddNewChild(gi.KiT_Layout, "slrow").(*gi.Layout)
	slab := slrow.AddNewChild(gi.KiT_Label, "slab").(*gi.Label)
	slab.Text = "Sliders:"

	srow := mfr.AddNewChild(gi.KiT_Layout, "srow").(*gi.Layout)
	srow.Lay = gi.LayoutHoriz
	srow.SetProp("spacing", units.NewValue(2, units.Ex))
	srow.SetProp("horizontal-align", "left")
	srow.SetStretchMaxWidth()

	slider1 := srow.AddNewChild(gi.KiT_Slider, "slider1").(*gi.Slider)
	slider1.Dim = gi.X
	slider1.Class = "hslides"
	slider1.Defaults()
	slider1.SetMinPrefWidth(units.NewValue(20, units.Em))
	slider1.SetMinPrefHeight(units.NewValue(2, units.Em))
	slider1.SetValue(0.5)
	slider1.Snap = true
	slider1.Tracking = true
	slider1.Icon = gi.IconName("widget-circlebutton-on")

	slider2 := srow.AddNewChild(gi.KiT_Slider, "slider2").(*gi.Slider)
	slider2.Dim = gi.Y
	slider2.Defaults()
	slider2.SetMinPrefHeight(units.NewValue(10, units.Em))
	slider2.SetMinPrefWidth(units.NewValue(1, units.Em))
	slider2.SetStretchMaxHeight()
	slider2.SetValue(0.5)

	slider1.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received slider signal: %v from slider: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
	})

	slider2.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received slider signal: %v from slider: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
	})

	scrollbar1 := srow.AddNewChild(gi.KiT_ScrollBar, "scrollbar1").(*gi.ScrollBar)
	scrollbar1.Dim = gi.X
	scrollbar1.Class = "hslides"
	scrollbar1.Defaults()
	scrollbar1.SetMinPrefWidth(units.NewValue(20, units.Em))
	scrollbar1.SetMinPrefHeight(units.NewValue(1, units.Em))
	scrollbar1.SetThumbValue(0.25)
	scrollbar1.SetValue(0.25)
	// scrollbar1.Snap = true
	scrollbar1.Tracking = true
	scrollbar1.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received scrollbar signal: %v from scrollbar: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
	})

	scrollbar2 := srow.AddNewChild(gi.KiT_ScrollBar, "scrollbar2").(*gi.ScrollBar)
	scrollbar2.Dim = gi.Y
	scrollbar2.Defaults()
	scrollbar2.SetMinPrefHeight(units.NewValue(10, units.Em))
	scrollbar2.SetMinPrefWidth(units.NewValue(1, units.Em))
	scrollbar2.SetStretchMaxHeight()
	scrollbar2.SetThumbValue(0.1)
	scrollbar2.SetValue(0.5)
	scrollbar2.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received scrollbar signal: %v from scrollbar: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
	})

	//////////////////////////////////////////
	//      Text Widgets

	mfr.AddNewChild(gi.KiT_Space, "tlspc")
	txlrow := mfr.AddNewChild(gi.KiT_Layout, "txlrow").(*gi.Layout)
	txlab := txlrow.AddNewChild(gi.KiT_Label, "txlab").(*gi.Label)
	txlab.Text = "Text Widgets:"
	txrow := mfr.AddNewChild(gi.KiT_Layout, "txrow").(*gi.Layout)
	txrow.Lay = gi.LayoutHoriz
	txrow.SetProp("spacing", units.NewValue(2, units.Ex))
	// txrow.SetProp("horizontal-align", gi.AlignJustify)
	txrow.SetStretchMaxWidth()

	edit1 := txrow.AddNewChild(gi.KiT_TextField, "edit1").(*gi.TextField)
	edit1.Placeholder = "Enter text here..."
	// edit1.SetText("Edit this text")
	edit1.SetProp("min-width", "20em")
	edit1.SetCompleter(nil, nil) // gets us word demo completion
	edit1.TextFieldSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received line edit signal: %v from edit: %v with data: %v\n", gi.TextFieldSignals(sig), send.Name(), data)
	})
	// edit1.SetProp("inactive", true)

	sb := txrow.AddNewChild(gi.KiT_SpinBox, "spin").(*gi.SpinBox)
	sb.Defaults()
	sb.HasMin = true
	sb.Min = 0.0
	sb.SpinBoxSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("SpinBox %v value changed: %v\n", send.Name(), data)
	})

	cb := txrow.AddNewChild(gi.KiT_ComboBox, "combo").(*gi.ComboBox)
	cb.ItemsFromTypes(kit.Types.AllImplementersOf(reflect.TypeOf((*gi.Node2D)(nil)).Elem(), false), true, true, 50)
	cb.ComboSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("ComboBox %v selected index: %v data: %v\n", send.Name(), sig, data)
	})

	//////////////////////////////////////////
	//      Main Menu

	appnm := oswin.TheApp.Name()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.KnownChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	// note: Command in shortcuts is automatically translated into Control for
	// Linux, Windows or Meta for MacOS
	fmen := win.MainMenu.KnownChildByName("File", 0).(*gi.Action)
	fmen.Menu = make(gi.Menu, 0, 10)
	fmen.Menu.AddMenuText("New", "Command+N", rec.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("File:New menu action triggered\n")
	})
	fmen.Menu.AddMenuText("Open", "Command+O", rec.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("File:Open menu action triggered\n")
	})
	fmen.Menu.AddMenuText("Save", "Command+S", rec.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("File:Save menu action triggered\n")
	})
	fmen.Menu.AddMenuText("Save As..", "Shift+Command+S", rec.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("File:SaveAs menu action triggered\n")
	})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddMenuText("Close Window", "Command+W", win.This, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		win.OSWin.CloseReq()
	})

	emen := win.MainMenu.KnownChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win, false)

	oswin.TheApp.SetQuitReqFunc(func() {
		gi.PromptDialog(vp, "Really Quit?", "Are you <i>sure</i> you want to quit?", true, true, nil, win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				oswin.TheApp.Quit()
			}
		})
	})

	oswin.TheApp.SetQuitCleanFunc(func() {
		fmt.Printf("Doing final Quit cleanup here..\n")
	})

	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		gi.PromptDialog(vp, "Really Close Window?", "Are you <i>sure</i> you want to close the window?  This will Quit the App as well.", true, true, nil, win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				oswin.TheApp.Quit()
			}
		})
	})

	win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
		fmt.Printf("Doing final Close cleanup here..\n")
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()

	// note: never gets here..
	fmt.Printf("ending\n")
}
