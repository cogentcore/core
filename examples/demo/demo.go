// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"
	"time"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

func app() {
	gi.SetAppName("gogi-demo")
	gi.SetAppAbout("The GoGi Demo demonstrates the various features of the GoGi 2D and 3D Go GUI framework.")

	sc := gi.NewScene("gogi-demo").SetTitle("GoGi Demo")

	ts := gi.NewTabs(sc)
	ts.NewTabButton = true

	makeHome(ts)
	makeText(ts)
	makeButtons(ts)
	makeInputs(ts)
	makeLayouts(ts)

	gi.NewWindow(sc).Run().Wait()
}

func makeHome(ts *gi.Tabs) {
	home := ts.NewTab("Home").SetLayout(gi.LayoutVert)

	gi.NewLabel(home).SetType(gi.LabelHeadlineLarge).SetText("The GoGi Demo")

	gi.NewLabel(home).SetType(gi.LabelBodyLarge).SetText("A demonstration of the <i>various</i> features of the <u>GoGi</u> 2D and 3D Go GUI <b>framework.</b>")

	pbar := gi.NewProgressBar(home)
	pbar.Start(100)
	go func() {
		for {
			if pbar.ProgCur >= pbar.ProgMax {
				pbar.Start(100)
			}
			time.Sleep(100 * time.Millisecond)
			pbar.ProgStep()
		}
	}()

	clr := colors.Tan
	giv.NewValue(home, &clr)

	giv.NewFuncButton(home, hello)

	img := gi.NewImage(home)
	err := img.OpenImage("gopher.png", 300, 300)
	if err != nil {
		fmt.Println("error loading gopher image:", err)
	}
}

// Hello displays a greeting message and an age in weeks based on the given information.
func hello(firstName string, lastName string, age int, likesGo bool) (greeting string, weeksOld int) { //gti:add
	weeksOld = age * 52
	greeting = "Hello, " + firstName + " " + lastName + "! "
	if likesGo {
		greeting += "I'm glad to here that you like the best programming language!"
	} else {
		greeting += "You should reconsider what programming languages you like."
	}
	return
}

func makeText(ts *gi.Tabs) {
	text := ts.NewTab("Text")
	text.Lay = gi.LayoutVert
	text.Style(func(s *styles.Style) {
		s.Spacing.Ex(1)
		s.Padding.Set(units.Dp(8))
		s.SetStretchMax()
	})

	gi.NewLabel(text).SetType(gi.LabelHeadlineLarge).SetText("Text")
	gi.NewLabel(text).SetText(
		`GoGi provides fully customizable text elements that can be styled in any way you want. Also, there are pre-configured style types for text that allow you to easily create common text types.`)

	for _, typ := range gi.LabelTypesValues() {
		s := strings.TrimPrefix(typ.String(), "Label")
		gi.NewLabel(text, "label"+s).SetType(typ).SetText(s)
	}
}

func makeButtons(ts *gi.Tabs) {
	buttons := ts.NewTab("Buttons")
	buttons.Lay = gi.LayoutVert
	buttons.Style(func(s *styles.Style) {
		s.Spacing.Ex(1)
		s.Padding.Set(units.Dp(8))
		s.SetStretchMax()
	})

	gi.NewLabel(buttons).SetType(gi.LabelHeadlineLarge).SetText("Buttons")

	gi.NewLabel(buttons, "bdesc").SetText(
		`GoGi provides customizable buttons that support various events and can be styled in any way you want. Also, there are pre-configured style types for buttons that allow you to achieve common functionality with ease. All buttons support any combination of a label, icon, and indicator.`)

	gi.NewLabel(buttons).SetType(gi.LabelHeadlineSmall).SetText("Standard Buttons")

	brow := gi.NewLayout(buttons, "brow").SetLayout(gi.LayoutHorizFlow)
	brow.Style(func(s *styles.Style) {
		s.Spacing.Em(1)
		s.SetStretchMaxWidth()
	})

	browt := gi.NewLayout(buttons, "browt").SetLayout(gi.LayoutHorizFlow)
	browt.Style(func(s *styles.Style) {
		s.Spacing.Em(1)
		s.SetStretchMaxWidth()
	})

	browi := gi.NewLayout(buttons, "browi").SetLayout(gi.LayoutHorizFlow)
	browi.Style(func(s *styles.Style) {
		s.Spacing.Em(1)
		s.SetStretchMaxWidth()
	})

	mbtitle := gi.NewLabel(buttons, "mbtitle").SetText("Menu Buttons")
	mbtitle.Type = gi.LabelHeadlineSmall

	mbrow := gi.NewLayout(buttons, "mbrow").SetLayout(gi.LayoutHorizFlow)
	mbrow.Style(func(s *styles.Style) {
		s.Spacing.Em(1)
		s.SetStretchMaxWidth()
	})

	mbrowt := gi.NewLayout(buttons, "mbrowt").SetLayout(gi.LayoutHorizFlow)
	mbrowt.Style(func(s *styles.Style) {
		s.Spacing.Em(1)
		s.SetStretchMaxWidth()
	})

	mbrowi := gi.NewLayout(buttons, "mbrowi").SetLayout(gi.LayoutHorizFlow)
	mbrowi.Style(func(s *styles.Style) {
		s.Spacing.Em(1)
		s.SetStretchMaxWidth()
	})

	menu := func(m *gi.Scene) {
		m1 := gi.NewButton(m).SetText("Menu Item 1").SetIcon(icons.Save).SetShortcut("Shift+Control+1").SetData(1).
			SetTooltip("A standard menu item with an icon")
		m1.OnClick(func(e events.Event) {
			fmt.Println("Received menu action with data", m1.Data)
		})

		m2 := gi.NewButton(m).SetText("Menu Item 2").SetIcon(icons.Open).SetData(2).
			SetTooltip("A menu item with an icon and a sub menu")

		m2.Menu = func(m *gi.Scene) {
			sm2 := gi.NewButton(m).SetText("Sub Menu Item 2").SetIcon(icons.InstallDesktop).SetData(2.1).
				SetTooltip("A sub menu item with an icon")
			sm2.OnClick(func(e events.Event) {
				fmt.Println("Received menu action with data", sm2.Data)
			})
		}

		gi.NewSeparator(m)

		m3 := gi.NewButton(m).SetText("Menu Item 3").SetIcon(icons.Favorite).SetShortcut("Control+3").SetData(3).
			SetTooltip("A standard menu item with an icon, below a separator")
		m3.OnClick(func(e events.Event) {
			fmt.Println("Received menu action with data", m3.Data)
		})
	}

	ics := []icons.Icon{
		icons.Search, icons.Home, icons.Close, icons.Done, icons.Favorite, icons.PlayArrow,
		icons.Add, icons.Delete, icons.ArrowBack, icons.Info, icons.Refresh, icons.VideoCall,
		icons.Menu, icons.Settings, icons.AccountCircle, icons.Download, icons.Sort, icons.DateRange,
		icons.Undo, icons.OpenInFull, icons.IosShare, icons.LibraryAdd, icons.OpenWith,
	}

	for _, typ := range gi.ButtonTypesValues() {
		// not really a real button, so not worth including in demo
		if typ == gi.ButtonMenu {
			continue
		}

		s := strings.TrimPrefix(typ.String(), "Button")
		sl := strings.ToLower(s)
		art := "A "
		if typ == gi.ButtonElevated || typ == gi.ButtonOutlined || typ == gi.ButtonAction {
			art = "An "
		}

		b := gi.NewButton(brow, "button"+s).SetType(typ).SetText(s).SetIcon(ics[typ]).
			SetTooltip("A standard " + sl + " button with a label and icon")
		b.OnClick(func(e events.Event) {
			fmt.Println("Got click event on", b.Nm)
		})

		bt := gi.NewButton(browt, "buttonText"+s).SetType(typ).SetText(s).
			SetTooltip("A standard " + sl + " button with a label")
		bt.OnClick(func(e events.Event) {
			fmt.Println("Got click event on", bt.Nm)
		})

		bi := gi.NewButton(browi, "buttonIcon"+s).SetType(typ).SetIcon(ics[typ+5]).
			SetTooltip("A standard " + sl + " button with an icon")
		bi.OnClick(func(e events.Event) {
			fmt.Println("Got click event on", bi.Nm)
		})

		gi.NewButton(mbrow, "menuButton"+s).SetType(typ).SetText(s).SetIcon(ics[typ+10]).SetMenu(menu).
			SetTooltip(art + sl + " menu button with a label and icon")

		gi.NewButton(mbrowt, "menuButtonText"+s).SetType(typ).SetText(s).SetMenu(menu).
			SetTooltip(art + sl + " menu button with a label")

		gi.NewButton(mbrowi, "menuButtonIcon"+s).SetType(typ).SetIcon(ics[typ+15]).SetMenu(menu).
			SetTooltip(art + sl + " menu button with an icon")
	}
}

func makeInputs(ts *gi.Tabs) {
	inputs := ts.NewTab("Inputs")
	inputs.Lay = gi.LayoutVert
	inputs.Style(func(s *styles.Style) {
		s.Spacing.Ex(1)
		s.Padding.Set(units.Dp(8))
		s.SetStretchMax()
	})

	gi.NewLabel(inputs).SetType(gi.LabelHeadlineLarge).SetText("Inputs")

	gi.NewLabel(inputs).SetType(gi.LabelBodyLarge).SetText(
		`GoGi provides various customizable input widgets that cover all common uses. Various events can be bound to inputs, and their data can easily be fetched and used wherever needed. There are also pre-configured style types for most inputs that allow you to easily switch among common styling patterns.`)

	gi.NewTextField(inputs).SetType(gi.TextFieldFilled).SetPlaceholder("Filled")
	gi.NewTextField(inputs).SetType(gi.TextFieldOutlined).SetPlaceholder("Outlined")
	gi.NewTextField(inputs).SetType(gi.TextFieldFilled).AddClearButton()
	gi.NewTextField(inputs).SetType(gi.TextFieldOutlined).AddClearButton()
	gi.NewTextField(inputs).SetType(gi.TextFieldFilled).AddClearButton().SetLeadingIcon(icons.Search)
	gi.NewTextField(inputs).SetType(gi.TextFieldOutlined).AddClearButton().SetLeadingIcon(icons.Search)
	gi.NewTextField(inputs).SetType(gi.TextFieldFilled).SetTypePassword().SetPlaceholder("Password")
	gi.NewTextField(inputs).SetType(gi.TextFieldOutlined).SetTypePassword().SetPlaceholder("Password")

	spinners := gi.NewLayout(inputs, "spinners").SetLayout(gi.LayoutHoriz)

	gi.NewSpinner(spinners).SetStep(5).SetMin(-50).SetMax(100).SetValue(15)
	gi.NewSpinner(spinners).SetFormat("%#X").SetStep(1).SetMax(255).SetValue(44)

	choosers := gi.NewLayout(inputs, "choosers").SetLayout(gi.LayoutHoriz)
	choosers.Style(func(s *styles.Style) {
		s.Spacing.Em(0.5)
	})

	fruits := []any{"Apple", "Apricot", "Blueberry", "Blackberry", "Peach", "Strawberry"}
	fruitDescs := []string{
		"A round, edible fruit that typically has red skin",
		"A stonefruit with a yellow or orange color",
		"A small blue or purple berry",
		"A small, edible, dark fruit",
		"A fruit with yellow or white flesh and a large seed",
		"A widely consumed small, red fruit",
	}

	gi.NewChooser(choosers).SetPlaceholder("Select a fruit").SetItems(fruits).SetTooltips(fruitDescs)
	gi.NewChooser(choosers).SetPlaceholder("Select a fruit").SetItems(fruits).SetTooltips(fruitDescs).SetType(gi.ChooserOutlined)
	gi.NewChooser(inputs).SetEditable(true).SetPlaceholder("Select or type a fruit").SetItems(fruits).SetTooltips(fruitDescs)
	gi.NewChooser(inputs).SetEditable(true).SetPlaceholder("Select or type a fruit").SetItems(fruits).SetTooltips(fruitDescs).SetType(gi.ChooserOutlined)

	gi.NewSwitch(inputs).SetText("Toggle")

	gi.NewSwitches(inputs).SetItems([]string{"Switch 1", "Switch 2", "Switch 3"}).
		SetTooltips([]string{"A description for Switch 1", "A description for Switch 2", "A description for Switch 3"})

	gi.NewSwitches(inputs).SetType(gi.SwitchChip).SetItems([]string{"Chip 1", "Chip 2", "Chip 3"}).
		SetTooltips([]string{"A description for Chip 1", "A description for Chip 2", "A description for Chip 3"})

	gi.NewSwitches(inputs).SetType(gi.SwitchCheckbox).SetItems([]string{"Checkbox 1", "Checkbox 2", "Checkbox 3"}).
		SetTooltips([]string{"A description for Checkbox 1", "A description for Checkbox 2", "A description for Checkbox 3"})

	gi.NewSwitches(inputs).SetType(gi.SwitchRadioButton).SetMutex(true).SetItems([]string{"Radio Button 1", "Radio Button 2", "Radio Button 3"}).
		SetTooltips([]string{"A description for Radio Button 1", "A description for Radio Button 2", "A description for Radio Button 3"})

	gi.NewSlider(inputs).SetDim(mat32.X).SetValue(0.5).SetTracking(true)
	gi.NewSlider(inputs).SetDim(mat32.X).SetValue(0.7).SetState(true, states.Disabled)

	sliderys := gi.NewLayout(inputs, "sliderys").SetLayout(gi.LayoutHorizFlow)

	gi.NewSlider(sliderys).SetDim(mat32.Y).SetValue(0.3)
	gi.NewSlider(sliderys).SetDim(mat32.Y).SetValue(0.2).SetState(true, states.Disabled)

	// tbuf := &giv.TextBuf{}
	// tbuf.InitName(tbuf, "tbuf")
	// tbuf.SetText([]byte("A keyboard-navigable, multi-line\ntext editor with support for\ncompletion and syntax highlighting"))

	// tview := giv.NewTextView(inputs, "tview")
	// tview.SetBuf(tbuf)
	// tview.Style(func(s *styles.Style) {
	// 	s.MaxWidth.SetDp(500)
	// 	s.MaxHeight.SetDp(300)
	// })
}

func makeLayouts(ts *gi.Tabs) {
	layouts := ts.NewTab("Layouts")
	layouts.Lay = gi.LayoutVert
	layouts.Style(func(s *styles.Style) {
		s.Spacing.Ex(1)
		s.Padding.Set(units.Dp(8))
		s.SetStretchMax()
	})

	// vw := gi.NewLabel(layouts, "vw", "50vw")
	// vw.Style(func(s *styles.Style) {
	// 	s.Width = units.Vw(50)
	// 	s.BackgroundColor.SetSolid(colors.Scheme.Primary.Base)
	// 	s.Color = colors.Scheme.Primary.On
	// })

	// pw := gi.NewLabel(layouts, "pw", "50pw")
	// pw.Style(func(s *styles.Style) {
	// 	s.Width = units.Pw(50)
	// 	s.BackgroundColor.SetSolid(colors.Scheme.Primary.Container)
	// 	s.Color = colors.Scheme.Primary.OnContainer
	// })

	sv := gi.NewSplits(layouts).SetDim(mat32.X)

	left := gi.NewFrame(sv).SetLayout(gi.LayoutVert).Style(func(s *styles.Style) {
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
	})
	gi.NewLabel(left).SetType(gi.LabelHeadlineMedium).SetText("Left")
	right := gi.NewFrame(sv).SetLayout(gi.LayoutVert).Style(func(s *styles.Style) {
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
	})
	gi.NewLabel(right).SetType(gi.LabelHeadlineMedium).SetText("Right")
}

// func doRenderWinSetup(win *gi.RenderWin, vp *gi.Scene) {
// 	// Main Menu

// 	appnm := gi.AppName()
// 	mmen := win.MainMenu
// 	mmen.ConfigMenus([]string{appnm, "File", "Edit", "RenderWin"})

// 	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Button)
// 	amen.Menu.AddAppMenu(win)

// 	fmen := win.MainMenu.ChildByName("File", 0).(*gi.Button)
// 	fmen.Menu.AddAction(gi.ActOpts{Label: "New", ShortcutKey: keyfun.MenuNew},
// 		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
// 			fmt.Println("File:New menu action triggered")
// 		})
// 	fmen.Menu.AddAction(gi.ActOpts{Label: "Open", ShortcutKey: keyfun.MenuOpen},
// 		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
// 			fmt.Println("File:Open menu action triggered")
// 		})
// 	fmen.Menu.AddAction(gi.ActOpts{Label: "Save", ShortcutKey: keyfun.MenuSave},
// 		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
// 			fmt.Println("File:Save menu action triggered")
// 		})
// 	fmen.Menu.AddAction(gi.ActOpts{Label: "Save As..", ShortcutKey: keyfun.MenuSaveAs},
// 		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
// 			fmt.Println("File:SaveAs menu action triggered")
// 		})
// 	fmen.Menu.AddSeparator("csep")
// 	fmen.Menu.AddAction(gi.ActOpts{Label: "Close RenderWin", ShortcutKey: keyfun.WinClose},
// 		win.This(), func(recv, send ki.Ki, sig int64, data any) {
// 			win.CloseReq()
// 		})
// 	inQuitPrompt := false
// 	gi.SetQuitReqFunc(func() {
// 		if inQuitPrompt {
// 			return
// 		}
// 		inQuitPrompt = true
// 		gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Quit?",
// 			Prompt: "Are you <i>sure</i> you want to quit?", Ok: true, Cancel: true}, func(dlg *gi.Dialog) {
// 			if dlg.Accepted {
// 				gi.Quit()
// 			} else {
// 				inQuitPrompt = false
// 			}
// 		})
// 	})

// 	gi.SetQuitCleanFunc(func() {
// 		fmt.Printf("Doing final Quit cleanup here..\n")
// 	})

// 	inClosePrompt := false
// 	win.SetCloseReqFunc(func(w *gi.RenderWin) {
// 		if inClosePrompt {
// 			return
// 		}
// 		inClosePrompt = true
// 		gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Close RenderWin?",
// 			Prompt: "Are you <i>sure</i> you want to close the window?  This will Quit the App as well.", Ok: true, Cancel: true}, func(dlg *gi.Dialog) {
// 			if dlg.Accepted {
// 				gi.Quit()
// 			} else {
// 				inClosePrompt = false
// 			}
// 		})
// 	})

// 	// win.SetCloseCleanFunc(func(w *gi.RenderWin) {
// 	// 	fmt.Printf("Doing final Close cleanup here..\n")
// 	// })

// 	win.MainMenuUpdated()
// }
