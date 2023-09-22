// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"reflect"
	"strings"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/gicons"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
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
	vp := win.WinViewport()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	tv := gi.NewTabView(mfr, "tv")
	// tv.NoDeleteTabs = true
	tv.NewTabButton = true

	makeHome(tv)
	makeText(tv)
	makeButtons(win, tv)
	makeInputs(tv)
	makeLayouts(tv)
	makeFileTree(tv)
	doWindowSetup(win, vp)

	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}

func makeHome(tv *gi.TabView) {
	home := tv.NewTab(gi.FrameType, "Home").(*gi.Frame)
	home.Lay = gi.LayoutVert
	home.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		home.Spacing.SetEx(1)
		s.Padding.Set(units.Px(8))
		s.MaxWidth.SetPx(-1)
		s.MaxHeight.SetPx(-1)
	})

	title := gi.NewLabel(home, "title", "The GoGi Demo")
	title.Type = gi.LabelHeadlineLarge

	desc := gi.NewLabel(home, "desc", "A demonstration of the <i>various</i> features of the <u>GoGi</u> 2D and 3D Go GUI <b>framework.</b>")
	desc.Type = gi.LabelBodyLarge

	// pbar := gi.NewProgressBar(home, "pbar")
	// pbar.Start(100)
	// go func() {
	// 	for {
	// 		if pbar.ProgCur >= pbar.ProgMax {
	// 			pbar.Start(100)
	// 		}
	// 		time.Sleep(100 * time.Millisecond)
	// 		pbar.ProgStep()
	// 	}
	// }()

	// bt := gi.NewButton(home, "bt")
	// bt.Text = "Big Shadow"
	// bt.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
	// 	bt.Style.AddBoxShadow(
	// 		gist.Shadow{
	// 			HOffset: units.Px(20),
	// 			VOffset: units.Px(-10),
	// 			Blur:    units.Px(150),
	// 			Spread:  units.Px(150),
	// 			Color:   colors.Green,
	// 		},
	// 		gist.Shadow{
	// 			HOffset: units.Px(5),
	// 			VOffset: units.Px(30),
	// 			Blur:    units.Px(150),
	// 			Spread:  units.Px(100),
	// 			Color:   colors.Blue,
	// 		},
	// 		gist.Shadow{
	// 			HOffset: units.Px(20),
	// 			VOffset: units.Px(10),
	// 			Blur:    units.Px(100),
	// 			Spread:  units.Px(50),
	// 			Color:   colors.Purple,
	// 		},
	// 	)
	// })

	bmap := gi.NewBitmap(home, "bmap")
	err := bmap.OpenImage("gopher.png", 300, 300)
	if err != nil {
		fmt.Println("error loading gopher image:", err)
	}
}

func makeText(tv *gi.TabView) {
	text := tv.NewTab(gi.FrameType, "Text").(*gi.Frame)
	text.Lay = gi.LayoutVert
	text.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		text.Spacing.SetEx(1)
		s.Padding.Set(units.Px(8))
		s.MaxWidth.SetPx(-1)
		s.MaxHeight.SetPx(-1)
	})

	ttitle := gi.NewLabel(text, "ttitle", "Text")
	ttitle.Type = gi.LabelHeadlineLarge

	tdesc := gi.NewLabel(text, "tdesc",
		`GoGi provides fully customizable text elements that can be styled in any way you want. Also, there are pre-configured style types for text that allow you to easily create common text types.`,
	)
	tdesc.Type = gi.LabelBodyLarge

	for typ := gi.LabelTypes(0); typ < gi.LabelTypesN; typ++ {
		s := strings.TrimPrefix(typ.String(), "Label")
		label := gi.NewLabel(text, "label"+s, s)
		label.Type = typ
	}

}

func makeButtons(win *gi.Window, tv *gi.TabView) {
	buttons := tv.NewTab(gi.FrameType, "Buttons").(*gi.Frame)
	buttons.Lay = gi.LayoutVert
	buttons.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		buttons.Spacing.SetEx(1)
		s.Padding.Set(units.Px(8))
		s.MaxWidth.SetPx(-1)
		s.MaxHeight.SetPx(-1)
	})

	btitle := gi.NewLabel(buttons, "btitle", "Buttons")
	btitle.Type = gi.LabelHeadlineLarge

	bdesc := gi.NewLabel(buttons, "bdesc",
		`GoGi provides customizable buttons that support various events and can be styled in any way you want. Also, there are pre-configured style types for buttons that allow you to achieve common functionality with ease. All buttons support any combination of a label, icon, and indicator.`,
	)
	bdesc.Type = gi.LabelBodyLarge

	sbtitle := gi.NewLabel(buttons, "sbtitle", "Standard Buttons")
	sbtitle.Type = gi.LabelHeadlineSmall

	brow := gi.NewLayout(buttons, "brow", gi.LayoutHorizFlow)
	brow.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		brow.Spacing.SetEm(1)
		s.MaxWidth.SetPx(-1)
	})

	browt := gi.NewLayout(buttons, "browt", gi.LayoutHorizFlow)
	browt.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		browt.Spacing.SetEm(1)
		s.MaxWidth.SetPx(-1)
	})

	browi := gi.NewLayout(buttons, "browi", gi.LayoutHorizFlow)
	browi.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		browi.Spacing.SetEm(1)
		s.MaxWidth.SetPx(-1)
	})

	mbtitle := gi.NewLabel(buttons, "mbtitle", "Menu Buttons")
	mbtitle.Type = gi.LabelHeadlineSmall

	mbrow := gi.NewLayout(buttons, "mbrow", gi.LayoutHorizFlow)
	mbrow.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		mbrow.Spacing.SetEm(1)
		s.MaxWidth.SetPx(-1)
	})

	mbrowt := gi.NewLayout(buttons, "mbrowt", gi.LayoutHorizFlow)
	mbrowt.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		mbrowt.Spacing.SetEm(1)
		s.MaxWidth.SetPx(-1)
	})

	mbrowi := gi.NewLayout(buttons, "mbrowi", gi.LayoutHorizFlow)
	mbrowi.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		mbrowi.Spacing.SetEm(1)
		s.MaxWidth.SetPx(-1)
	})

	menu := gi.Menu{}

	menu.AddAction(gi.ActOpts{Label: "Menu Item 1", Icon: gicons.Save, Shortcut: "Shift+Control+1", Tooltip: "A standard menu item with an icon", Data: 1},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mi2 := menu.AddAction(gi.ActOpts{Label: "Menu Item 2", Icon: gicons.FileOpen, Tooltip: "A menu item with an icon and a sub menu", Data: 2}, nil, nil)

	mi2.Menu.AddAction(gi.ActOpts{Label: "Sub Menu Item 2", Icon: gicons.InstallDesktop, Tooltip: "A sub menu item with an icon", Data: 2.1},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	menu.AddSeparator("sep1")

	menu.AddAction(gi.ActOpts{Label: "Menu Item 3", Icon: gicons.Favorite, Shortcut: "Control+3", Tooltip: "A standard menu item with an icon, below a separator", Data: 3},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	ics := []gicons.Icon{
		gicons.Search, gicons.Home, gicons.Close, gicons.Done, gicons.Favorite,
		gicons.Add, gicons.Delete, gicons.ArrowBack, gicons.Info, gicons.Refresh,
		gicons.Menu, gicons.Settings, gicons.AccountCircle, gicons.Download, gicons.Sort,
		gicons.Undo, gicons.OpenInFull, gicons.IosShare, gicons.LibraryAdd, gicons.OpenWith,
	}

	for typ := gi.ButtonTypes(0); typ < gi.ButtonTypesN; typ++ {
		s := strings.TrimPrefix(typ.String(), "Button")
		sl := strings.ToLower(s)
		art := "A "
		if typ == gi.ButtonElevated || typ == gi.ButtonOutlined {
			art = "An "
		}

		b := gi.NewButton(brow, "button"+s)
		b.Text = s
		b.Icon = ics[typ]
		b.Type = typ
		b.Tooltip = "A standard " + sl + " button with a label and icon"
		b.OnClicked(func() {
			fmt.Println("Got click event on", b.Nm)
		})

		bt := gi.NewButton(browt, "buttonText"+s)
		bt.Text = s
		bt.Type = typ
		bt.Tooltip = "A standard " + sl + " button with a label"
		bt.OnClicked(func() {
			fmt.Println("Got click event on", bt.Nm)
		})

		bi := gi.NewButton(browi, "buttonIcon"+s)
		bi.Type = typ
		bi.Icon = ics[typ+5]
		bi.Tooltip = "A standard " + sl + " button with an icon"
		bi.OnClicked(func() {
			fmt.Println("Got click event on", bi.Nm)
		})

		mb := gi.NewButton(mbrow, "menuButton"+s)
		mb.Text = s
		mb.Icon = ics[typ+10]
		mb.Type = typ
		mb.Menu = menu
		mb.Tooltip = art + sl + " menu button with a label and icon"

		mbt := gi.NewButton(mbrowt, "menuButtonText"+s)
		mbt.Text = s
		mbt.Type = typ
		mbt.Menu = menu
		mbt.Tooltip = art + sl + " menu button with a label"

		mbi := gi.NewButton(mbrowi, "menuButtonIcon"+s)
		mbi.Icon = ics[typ+15]
		mbi.Type = typ
		mbi.Menu = menu
		mbi.Tooltip = art + sl + " menu button with an icon"
	}
}

func makeInputs(tv *gi.TabView) {
	inputs := tv.NewTab(gi.FrameType, "Inputs").(*gi.Frame)
	inputs.Lay = gi.LayoutVert
	inputs.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		inputs.Spacing.SetEx(1)
		s.Padding.Set(units.Px(8))
		s.MaxWidth.SetPx(-1)
		s.MaxHeight.SetPx(-1)
	})

	ititle := gi.NewLabel(inputs, "ititle", "Inputs")
	ititle.Type = gi.LabelHeadlineLarge

	idesc := gi.NewLabel(inputs, "idesc",
		`GoGi provides various customizable input widgets that cover all common uses. Various events can be bound to inputs, and their data can easily be fetched and used wherever needed. There are also pre-configured style types for most inputs that allow you to easily switch among common styling patterns.`,
	)
	idesc.Type = gi.LabelBodyLarge

	tff := gi.NewTextField(inputs, "tff")
	tff.Placeholder = "Filled Text Field"
	tff.Type = gi.TextFieldFilled

	tfo := gi.NewTextField(inputs, "tfo")
	tfo.Placeholder = "Outlined Text Field"
	tfo.Type = gi.TextFieldOutlined

	tffc := gi.NewTextField(inputs, "tffc")
	tffc.Placeholder = "Filled Text Field"
	tffc.Type = gi.TextFieldFilled
	tffc.AddClearAction()

	tfoc := gi.NewTextField(inputs, "tfoc")
	tfoc.Placeholder = "Outlined Text Field"
	tfoc.Type = gi.TextFieldOutlined
	tfoc.AddClearAction()

	tffcs := gi.NewTextField(inputs, "tffcs")
	tffcs.Placeholder = "Filled Text Field"
	tffcs.Type = gi.TextFieldFilled
	tffcs.AddClearAction()
	tffcs.LeadingIcon = gicons.Search

	tfocs := gi.NewTextField(inputs, "tfocs")
	tfocs.Placeholder = "Outlined Text Field"
	tfocs.Type = gi.TextFieldOutlined
	tfocs.AddClearAction()
	tfocs.LeadingIcon = gicons.Search

	tffp := gi.NewTextField(inputs, "tffp")
	tffp.Placeholder = "Password Text Field"
	tffp.Type = gi.TextFieldFilled
	tffp.SetTypePassword()

	tfop := gi.NewTextField(inputs, "tfop")
	tfop.Placeholder = "Password Text Field"
	tfop.Type = gi.TextFieldOutlined
	tfop.SetTypePassword()

	sboxes := gi.NewLayout(inputs, "sboxes", gi.LayoutHoriz)

	sbox := gi.NewSpinBox(sboxes, "sbox")
	sbox.Value = 15
	sbox.Step = 5
	sbox.SetMin(-50)
	sbox.SetMax(100)

	sboxh := gi.NewSpinBox(sboxes, "sboxh")
	sboxh.Format = "%#X"
	sboxh.Value = 44
	sboxh.Step = 1
	sboxh.SetMax(255)

	cboxes := gi.NewLayout(inputs, "cboxes", gi.LayoutHoriz)
	cboxes.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		cboxes.Spacing.SetEm(0.5)
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

	cbf := gi.NewComboBox(cboxes, "cbf")
	cbf.Text = "Select a fruit"
	cbf.Items = fruits
	cbf.Tooltips = fruitDescs

	cbo := gi.NewComboBox(cboxes, "cbo")
	cbo.Text = "Select a fruit"
	cbo.Items = fruits
	cbo.Tooltips = fruitDescs
	cbo.Type = gi.ComboBoxOutlined

	cbef := gi.NewComboBox(inputs, "cbef")
	cbef.Editable = true
	cbef.Placeholder = "Select or type a fruit"
	cbef.Items = fruits
	cbef.Tooltips = fruitDescs

	cbeo := gi.NewComboBox(inputs, "cbeo")
	cbeo.Editable = true
	cbeo.Placeholder = "Select or type a fruit"
	cbeo.Items = fruits
	cbeo.Tooltips = fruitDescs
	cbeo.Type = gi.ComboBoxOutlined

	sliderx := gi.NewSlider(inputs, "sliderx")
	sliderx.Dim = mat32.X
	sliderx.Value = 0.5

	sliderxi := gi.NewSlider(inputs, "sliderxi")
	sliderxi.Dim = mat32.X
	sliderxi.Value = 0.7
	sliderxi.SetDisabled()

	clr := colors.Tan

	colorvv := giv.ToValueView(&clr, "")
	colorvv.SetSoloValue(reflect.ValueOf(&clr))
	cvvw := inputs.NewChild(colorvv.WidgetType(), "cvvw").(gi.Node2D)
	colorvv.ConfigWidget(cvvw)

	sliderys := gi.NewLayout(inputs, "sliderys", gi.LayoutHorizFlow)

	slidery := gi.NewSlider(sliderys, "slidery")
	slidery.Dim = mat32.Y
	slidery.Value = 0.3

	slideryi := gi.NewSlider(sliderys, "slideryi")
	slideryi.Dim = mat32.Y
	slideryi.Value = 0.2
	slideryi.SetDisabled()

	bbox := gi.NewButtonBox(inputs, "bbox")
	bbox.Items = []string{"Checkbox 1", "Checkbox 2", "Checkbox 3"}
	bbox.Tooltips = []string{"A description for Checkbox 1", "A description for Checkbox 2", "A description for Checkbox 3"}

	bboxr := gi.NewButtonBox(inputs, "bboxr")
	bboxr.Items = []string{"Radio Button 1", "Radio Button 2", "Radio Button 3"}
	bboxr.Tooltips = []string{"A description for Radio Button 1", "A description for Radio Button 2", "A description for Radio Button 3"}
	bboxr.Mutex = true

	tbuf := &giv.TextBuf{}
	tbuf.InitName(tbuf, "tbuf")
	tbuf.SetText([]byte("A keyboard-navigable, multi-line\ntext editor with support for\ncompletion and syntax highlighting"))

	tview := giv.NewTextView(inputs, "tview")
	tview.SetBuf(tbuf)
	tview.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		s.MaxWidth.SetPx(500)
		s.MaxHeight.SetPx(300)
	})
}

func makeLayouts(tv *gi.TabView) {
	layouts := tv.NewTab(gi.FrameType, "Layouts").(*gi.Frame)
	layouts.Lay = gi.LayoutVert
	layouts.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		layouts.Spacing.SetEx(1)
		s.Padding.Set(units.Px(8))
		s.MaxWidth.SetPx(-1)
		s.MaxHeight.SetPx(-1)
	})

	vw := gi.NewLabel(layouts, "vw", "50vw")
	vw.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		s.Width = units.Vw(50)
		s.BackgroundColor.SetSolid(gi.ColorScheme.Primary)
		s.Color = gi.ColorScheme.OnPrimary
	})

	pw := gi.NewLabel(layouts, "pw", "50pw")
	pw.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
		s.Width = units.Pw(50)
		s.BackgroundColor.SetSolid(gi.ColorScheme.PrimaryContainer)
		s.Color = gi.ColorScheme.OnPrimaryContainer
	})

	// sv := gi.NewSplitView(layouts, "sv")
	// sv.Dim = mat32.X

	// left := gi.NewFrame(sv, "left", gi.LayoutVert)

	// leftTitle := gi.NewLabel(left, "leftTitle", "Left")
	// leftTitle.Type = gi.LabelHeadlineMedium

	// right := gi.NewFrame(sv, "right", gi.LayoutVert)

	// rightTitle := gi.NewLabel(right, "rightTitle", "Right")
	// rightTitle.Type = gi.LabelHeadlineMedium

}

func makeFileTree(tv *gi.TabView) {
	filetree := tv.NewTab(gi.FrameType, "File Tree").(*gi.Frame)
	filetree.Lay = gi.LayoutVert
}

func doWindowSetup(win *gi.Window, vp *gi.Viewport) {
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
}
