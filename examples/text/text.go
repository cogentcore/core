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

	// gi.Layout2DTrace = true

	oswin.TheApp.SetName("text")
	oswin.TheApp.SetAbout(`This is a demo of the text rendering in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	win := gi.NewWindow2D("gogi-text-test", "GoGi Text Test", width, height, true) // true = pixel sizes

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	// style sheet
	var css = ki.Props{
		"kbd": ki.Props{
			"color": "blue",
		},
	}
	vp.CSS = css

	mfr := win.SetMainFrame()

	// trow := mfr.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	// trow.Lay = gi.LayoutHoriz
	// trow.SetStretchMaxWidth()

	// 	trow.AddNewChild(gi.KiT_Stretch, "str1")
	// 	title := trow.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	// 	hdrText := `This is a <b>test</b> of the
	// <span style="color:red">various</span> <i>GoGi</i> Text elements<br>
	// <large>Shortcuts: <kbd>Ctrl+Alt+P</kbd> = Preferences,
	// <kbd>Ctrl+Alt+E</kbd> = Editor, <kbd>Ctrl/Cmd +/-</kbd> = zoom</large><br>
	// Other styles: <u>underlining</u> and <abbr>abbr dotted uline</abbr> and <strike>strikethrough</strike><br>
	// <q>and</q> <mark>marked text</mark> and <span style="text-decoration:overline">overline</span>
	// and Sub<sub>script</sub> and Super<sup>script</sup>`
	// 	title.Text = hdrText
	// 	// title.Text = "header" // use this to test word wrapping
	// 	title.SetProp("text-align", gi.AlignRight)
	// 	title.SetProp("vertical-align", gi.AlignTop)
	// 	title.SetProp("font-family", "Times New Roman, serif")
	// 	title.SetProp("font-size", "x-large")
	// 	// title.SetProp("letter-spacing", 2)
	// 	title.SetProp("line-height", 1.5)

	// 	rtxt := trow.AddNewChild(gi.KiT_Label, "rtxt").(*gi.Label)
	// 	rtxt.Text = "this is to test right margin"

	// 	mfr.AddNewChild(gi.KiT_Space, "spc")

	wrlab := mfr.AddNewChild(gi.KiT_Label, "wrlab").(*gi.Label)
	wrlab.SetProp("word-wrap", true)
	wrlab.SetProp("width", "20em")
	wrlab.SetProp("max-width", -1)
	wrlab.SetProp("line-height", 1.2)
	wrlab.SetProp("para-spacing", "1ex")
	wrlab.SetProp("text-indent", "20px")
	wrlab.Text = `<p>Word <u>wrapping</u> should be <span style="color:red">enabled in this label</span> -- this is a test to see if it is.  Usually people use some kind of obscure latin text here -- not really sure why <u>because nobody reads latin anymore,</u> at least nobody I know.  Anyway, let's see how this works.  Also, it should be interesting to determine how word wrapping works with styling -- <large>the styles should properly wrap across the lines</large>.  In addition, there is the question of <b>how built-in breaks interface</b> with the auto-line breaks, and furthermore the question of paragraph breaks versus simple br line breaks.</p>
<p>One major question is the extent to which <a href="https://en.wikipedia.org/wiki/Line_wrap_and_word_wrap">word wrapping</a> can be made sensitive to the overall size of the containing element -- this is easy when setting a direct fixed width, but word wrapping should also occur as the user resizes the window.</p>
It appears that the <b>end</b> of one paragraph implies the start of a new one, even if you do <i>not</i> insert a <code>p</code> tag.
`

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

	button1.SetIcon("computer")
	button1.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received button signal: %v from button: %v\n", gi.ButtonSignals(sig), send.Name())
		if sig == int64(gi.ButtonClicked) { // note: 3 diff ButtonSig sig's possible -- important to check
			// vp.Win.Quit()
			gi.StringPromptDialog(vp, "Enter value here..", "Button1 Dialog", "This is a string prompt dialog!  Various specific types of dialogs are available.", nil, win.This,
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
	button2.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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

	slider1.SliderSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received slider signal: %v from slider: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
	})

	slider2.SliderSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	scrollbar1.SliderSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	scrollbar2.SliderSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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
	edit1.TextFieldSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received line edit signal: %v from edit: %v with data: %v\n", gi.TextFieldSignals(sig), send.Name(), data)
	})
	// edit1.SetProp("inactive", true)

	sb := txrow.AddNewChild(gi.KiT_SpinBox, "spin").(*gi.SpinBox)
	sb.Defaults()
	sb.HasMin = true
	sb.Min = 0.0
	sb.SpinBoxSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("SpinBox %v value changed: %v\n", send.Name(), data)
	})

	cb := txrow.AddNewChild(gi.KiT_ComboBox, "combo").(*gi.ComboBox)
	cb.ItemsFromTypes(kit.Types.AllImplementersOf(reflect.TypeOf((*gi.Node2D)(nil)).Elem(), false), true, true, 50)
	cb.ComboSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("ComboBox %v selected index: %v data: %v\n", send.Name(), sig, data)
	})

	// mfr.AddNewChild(gi.KiT_Space, "aspc")

	etxt := mfr.AddNewChild(gi.KiT_Label, "etxt").(*gi.Label)
	etxt.Text = "this is to test bottom after word wrapped text"

	// mfr.AddNewChild(gi.KiT_Stretch, "str")

	// // main menu
	// appnm := oswin.TheApp.Name()
	// mmen := win.MainMenu
	// mmen.ConfigMenus([]string{appnm, "Edit", "Window"})

	// amen := win.MainMenu.KnownChildByName(appnm, 0).(*gi.Action)
	// amen.Menu = make(gi.Menu, 0, 10)
	// amen.Menu.AddAppMenu(win)

	// emen := win.MainMenu.KnownChildByName("Edit", 1).(*gi.Action)
	// emen.Menu = make(gi.Menu, 0, 10)
	// emen.Menu.AddCopyCutPaste(win, true)

	// win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
	// 	go oswin.TheApp.Quit() // once main window is closed, quit
	// })

	// win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
