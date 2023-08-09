// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/gist/colors"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/icons"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/mat32"
	"github.com/iancoleman/strcase"
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

	tv := gi.AddNewTabView(mfr, "tv")
	// tv.NoDeleteTabs = true
	tv.NewTabButton = true

	makeHome(tv)
	makeText(tv)
	makeButtons(win, tv)
	makeInputs(tv)
	makeLayouts(tv)
	doWindowSetup(win, vp)

	vp.UpdateEndNoSig(updt)
	win.StartEventLoop()
}

func makeHome(tv *gi.TabView) {
	home := tv.AddNewTab(gi.TypeFrame, "Home").(*gi.Frame)
	home.Lay = gi.LayoutVert
	home.AddStyleFunc(gi.StyleFuncFinal, func() {
		home.Spacing.SetEx(1)
		home.Style.Padding.Set(units.Px(8))
		home.Style.MaxWidth.SetPx(-1)
		home.Style.MaxHeight.SetPx(-1)
	})

	title := gi.AddNewLabel(home, "title", "The GoGi Demo")
	title.Type = gi.LabelHeadlineLarge

	desc := gi.AddNewLabel(home, "desc", "A demonstration of the <i>various</i> features of the <u>GoGi</u> 2D and 3D Go GUI <b>framework.</b>")
	desc.Type = gi.LabelBodyLarge

	pbar := gi.AddNewProgressBar(home, "pbar")
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

	bmap := gi.AddNewBitmap(home, "bmap")
	err := bmap.OpenImage("gopher.png", 300, 300)
	if err != nil {
		fmt.Println("error loading gopher image:", err)
	}
}

func makeText(tv *gi.TabView) {
	text := tv.AddNewTab(gi.TypeFrame, "Text").(*gi.Frame)
	text.Lay = gi.LayoutVert
	text.AddStyleFunc(gi.StyleFuncFinal, func() {
		text.Spacing.SetEx(1)
		text.Style.Padding.Set(units.Px(8))
		text.Style.MaxWidth.SetPx(-1)
		text.Style.MaxHeight.SetPx(-1)
	})

	ttitle := gi.AddNewLabel(text, "ttitle", "Text")
	ttitle.Type = gi.LabelHeadlineLarge

	tdesc := gi.AddNewLabel(text, "tdesc",
		`GoGi provides fully customizable text elements that can be styled in any way you want. Also, there are pre-configured style types for text that allow you to easily create common text types.`,
	)
	tdesc.Type = gi.LabelBodyLarge

	for typ := gi.LabelTypes(0); typ < gi.LabelTypesN; typ++ {
		s := strcase.ToCamel(strings.TrimPrefix(typ.String(), "Label"))
		label := gi.AddNewLabel(text, "label"+s, s)
		label.Type = typ
	}

}

func makeButtons(win *gi.Window, tv *gi.TabView) {
	buttons := tv.AddNewTab(gi.TypeFrame, "Buttons").(*gi.Frame)
	buttons.Lay = gi.LayoutVert
	buttons.AddStyleFunc(gi.StyleFuncFinal, func() {
		buttons.Spacing.SetEx(1)
		buttons.Style.Padding.Set(units.Px(8))
		buttons.Style.MaxWidth.SetPx(-1)
		buttons.Style.MaxHeight.SetPx(-1)
	})

	btitle := gi.AddNewLabel(buttons, "btitle", "Buttons")
	btitle.Type = gi.LabelHeadlineLarge

	bdesc := gi.AddNewLabel(buttons, "bdesc",
		`GoGi provides customizable buttons that support various events and can be styled in any way you want. Also, there are pre-configured style types for buttons that allow you to achieve common functionality with ease. All buttons support any combination of a label, icon, and indicator.`,
	)
	bdesc.Type = gi.LabelBodyLarge

	sbdesc := gi.AddNewLabel(buttons, "bdesc", "Standard Buttons")
	sbdesc.Type = gi.LabelHeadlineSmall

	brow := gi.AddNewLayout(buttons, "brow", gi.LayoutHorizFlow)
	brow.AddStyleFunc(gi.StyleFuncFinal, func() {
		brow.Spacing.SetEx(1)
		brow.Style.MaxWidth.SetPx(-1)
	})

	bfill := gi.AddNewButton(brow, "buttonFilled")
	bfill.Text = "Filled"
	bfill.Type = gi.ButtonFilled
	bfill.Icon = icons.InstallDesktop

	bton := gi.AddNewButton(brow, "buttonTonal")
	bton.Text = "Tonal"
	bton.Type = gi.ButtonTonal
	bton.Icon = icons.Settings

	belv := gi.AddNewButton(brow, "buttonElevated")
	belv.Text = "Elevated"
	belv.Type = gi.ButtonElevated
	belv.Icon = icons.Star

	bout := gi.AddNewButton(brow, "buttonOutlined")
	bout.Text = "Outlined"
	bout.Type = gi.ButtonOutlined
	bout.Icon = icons.Close

	btxt := gi.AddNewButton(brow, "buttonText")
	btxt.Text = "Text"
	btxt.Type = gi.ButtonText
	btxt.Icon = icons.Home

	// browto := gi.AddNewLayout(buttons, "browTextOnly", gi.LayoutHorizFlow)
	// browto.AddStyleFunc(gi.StyleFuncFinal, func() {
	// 	browto.Spacing.SetEx(1)
	// 	browto.Style.MaxWidth.SetPx(-1)
	// })

	// bprito := gi.AddNewButton(browto, "buttonPrimaryTextOnly")
	// bprito.Text = "Primary Button"
	// bprito.Type = gi.ButtonFilled

	// bsecto := gi.AddNewButton(browto, "buttonSecondaryTextOnly")
	// bsecto.Text = "Secondary Button"
	// bsecto.Type = gi.ButtonTonal

	// bdefto := gi.AddNewButton(browto, "buttonDefaultTextOnly")
	// bdefto.Text = "Default Button"

	// browio := gi.AddNewLayout(buttons, "browIconOnly", gi.LayoutHorizFlow)
	// browio.AddStyleFunc(gi.StyleFuncFinal, func() {
	// 	browio.Spacing.SetEx(1)
	// 	browio.Style.MaxWidth.SetPx(-1)
	// })

	// bpriio := gi.AddNewButton(browio, "buttonPrimaryTextOnly")
	// bpriio.Icon = icons.Send
	// bpriio.Type = gi.ButtonFilled

	// bsecio := gi.AddNewButton(browio, "buttonSecondaryTextOnly")
	// bsecio.Icon = icons.Info
	// bsecio.Type = gi.ButtonTonal

	// bdefio := gi.AddNewButton(browio, "buttonDefaultTextOnly")
	// bdefio.Icon = icons.AccountCircle

	// bidesc := gi.AddNewLabel(buttons, "bidesc", "Inactive Standard Buttons")
	// bidesc.Type = gi.LabelH3

	// browi := gi.AddNewLayout(buttons, "browi", gi.LayoutHorizFlow)
	// browi.AddStyleFunc(gi.StyleFuncFinal, func() {
	// 	browi.Spacing.SetEx(1)
	// 	browi.Style.MaxWidth.SetPx(-1)
	// })

	// bprii := gi.AddNewButton(browi, "buttonPrimaryInactive")
	// bprii.Text = "Inactive Primary Button"
	// bprii.Type = gi.ButtonFilled
	// bprii.Icon = icons.OpenInNew
	// bprii.SetInactive()

	// bseci := gi.AddNewButton(browi, "buttonSecondaryInactive")
	// bseci.Text = "Inactive Secondary Button"
	// bseci.Type = gi.ButtonTonal
	// bseci.Icon = icons.Settings
	// bseci.SetInactive()

	// bdefi := gi.AddNewButton(browi, "buttonDefaultInactive")
	// bdefi.Text = "Inactive Default Button"
	// bdefi.SetInactive()

	mbdesc := gi.AddNewLabel(buttons, "mbdesc", "Menu Buttons")
	mbdesc.Type = gi.LabelHeadlineSmall

	mbrow := gi.AddNewLayout(buttons, "mbrow", gi.LayoutHorizFlow)
	mbrow.AddStyleFunc(gi.StyleFuncFinal, func() {
		mbrow.Spacing.SetEx(1)
		mbrow.Style.MaxWidth.SetPx(-1)
	})

	mbfill := gi.AddNewMenuButton(mbrow, "menuButtonFilled")
	mbfill.Text = "Filled Menu Button"
	mbfill.Type = gi.MenuButtonFilled
	mbfill.Menu.AddAction(gi.ActOpts{Label: "Menu Item 1", Shortcut: "Shift+Control+1", Data: 1},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mi2 := mbfill.Menu.AddAction(gi.ActOpts{Label: "Menu Item 2", Data: 2}, nil, nil)

	mi2.Menu.AddAction(gi.ActOpts{Label: "Sub Menu Item 2", Data: 2.1},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mbfill.Menu.AddSeparator("sep1")

	mbfill.Menu.AddAction(gi.ActOpts{Label: "Menu Item 3", Shortcut: "Control+3", Data: 3},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mbout := gi.AddNewMenuButton(mbrow, "menuButtonOutlined")
	mbout.Text = "Outlined Menu Button"
	mbout.Type = gi.MenuButtonOutlined
	mbout.Menu.AddAction(gi.ActOpts{Label: "Menu Item 1", Shortcut: "Shift+Control+1", Data: 1},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mi2 = mbout.Menu.AddAction(gi.ActOpts{Label: "Menu Item 2", Data: 2}, nil, nil)

	mi2.Menu.AddAction(gi.ActOpts{Label: "Sub Menu Item 2", Data: 2.1},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mbout.Menu.AddSeparator("sep1")

	mbout.Menu.AddAction(gi.ActOpts{Label: "Menu Item 3", Shortcut: "Control+3", Data: 3},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mbtxt := gi.AddNewMenuButton(mbrow, "menuButtonText")
	mbtxt.Text = "Text Menu Button"
	mbtxt.Type = gi.MenuButtonText
	mbtxt.Menu.AddAction(gi.ActOpts{Label: "Menu Item 1", Shortcut: "Shift+Control+1", Data: 1},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mi2 = mbtxt.Menu.AddAction(gi.ActOpts{Label: "Menu Item 2", Data: 2}, nil, nil)

	mi2.Menu.AddAction(gi.ActOpts{Label: "Sub Menu Item 2", Data: 2.1},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mbtxt.Menu.AddSeparator("sep1")

	mbtxt.Menu.AddAction(gi.ActOpts{Label: "Menu Item 3", Shortcut: "Control+3", Data: 3},
		win.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})
}

func makeInputs(tv *gi.TabView) {
	inputs := tv.AddNewTab(gi.TypeFrame, "Inputs").(*gi.Frame)
	inputs.Lay = gi.LayoutVert
	inputs.AddStyleFunc(gi.StyleFuncFinal, func() {
		inputs.Spacing.SetEx(1)
		inputs.Style.Padding.Set(units.Px(8))
		inputs.Style.MaxWidth.SetPx(-1)
		inputs.Style.MaxHeight.SetPx(-1)
	})

	ititle := gi.AddNewLabel(inputs, "ititle", "Inputs")
	ititle.Type = gi.LabelHeadlineLarge

	idesc := gi.AddNewLabel(inputs, "idesc",
		`GoGi provides various customizable input widgets that cover all common uses. Various events can be bound to inputs, and their data can easily be fetched and used wherever needed. There are also pre-configured style types for most inputs that allow you to easily switch among common styling patterns.`,
	)
	idesc.Type = gi.LabelBodyLarge

	tff := gi.AddNewTextField(inputs, "tff")
	tff.Placeholder = "Filled Text Field"
	tff.Type = gi.TextFieldFilled

	tfo := gi.AddNewTextField(inputs, "tfo")
	tfo.Placeholder = "Outlined Text Field"
	tfo.Type = gi.TextFieldOutlined

	tffc := gi.AddNewTextField(inputs, "tffc")
	tffc.Placeholder = "Filled Text Field"
	tffc.Type = gi.TextFieldFilled
	tffc.AddClearAction()

	tfoc := gi.AddNewTextField(inputs, "tfoc")
	tfoc.Placeholder = "Outlined Text Field"
	tfoc.Type = gi.TextFieldOutlined
	tfoc.AddClearAction()

	tffcs := gi.AddNewTextField(inputs, "tffcs")
	tffcs.Placeholder = "Filled Text Field"
	tffcs.Type = gi.TextFieldFilled
	tffcs.AddClearAction()
	tffcs.LeadingIcon = icons.Search

	tfocs := gi.AddNewTextField(inputs, "tfocs")
	tfocs.Placeholder = "Outlined Text Field"
	tfocs.Type = gi.TextFieldOutlined
	tfocs.AddClearAction()
	tfocs.LeadingIcon = icons.Search

	tffp := gi.AddNewTextField(inputs, "tffp")
	tffp.Placeholder = "Password Text Field"
	tffp.Type = gi.TextFieldFilled
	tffp.SetTypePassword()

	tfop := gi.AddNewTextField(inputs, "tfop")
	tfop.Placeholder = "Password Text Field"
	tfop.Type = gi.TextFieldOutlined
	tfop.SetTypePassword()

	sboxes := gi.AddNewLayout(inputs, "sboxes", gi.LayoutHoriz)

	sbox := gi.AddNewSpinBox(sboxes, "sbox")
	sbox.Value = 15
	sbox.Step = 5
	sbox.SetMin(-50)
	sbox.SetMax(100)

	sboxh := gi.AddNewSpinBox(sboxes, "sboxh")
	sboxh.Format = "%#X"
	sboxh.Value = 44
	sboxh.Step = 1
	sboxh.SetMax(255)

	cbox := gi.AddNewComboBox(inputs, "cbox")
	cbox.Text = "Select an option"
	cbox.Items = []any{"Option 1", "Option 2", "Option 3"}
	cbox.Tooltips = []string{"A description for Option 1", "A description for Option 2", "A description for Option 3"}

	cboxe := gi.AddNewComboBox(inputs, "cboxe")
	cboxe.Editable = true
	cboxe.Text = "Select or type an option"
	cboxe.Items = []any{"Option 1", "Option 2", "Option 3"}
	cboxe.Tooltips = []string{"A description for Option 1", "A description for Option 2", "A description for Option 3"}

	clr := colors.Blue

	colorvv := giv.ToValueView(&clr, "")
	colorvv.SetSoloValue(reflect.ValueOf(&clr))
	cvvw := inputs.AddNewChild(colorvv.WidgetType(), "cvvw").(gi.Node2D)
	colorvv.ConfigWidget(cvvw)

	svi := giv.AddNewStructView(inputs, "svi")
	svi.Struct = &gi.ColorScheme
	svi.AddStyleFunc(gi.StyleFuncDefault, func() {
		svi.Style.MaxWidth.SetPx(-1)
		svi.Style.MaxHeight.SetPx(-1)
	})

	sliderx := gi.AddNewSlider(inputs, "sliderx")
	sliderx.Dim = mat32.X
	sliderx.Value = 0.5

	sliderxi := gi.AddNewSlider(inputs, "sliderxi")
	sliderxi.Dim = mat32.X
	sliderxi.Value = 0.7
	sliderxi.SetInactive()

	sliderys := gi.AddNewLayout(inputs, "sliderys", gi.LayoutHorizFlow)

	slidery := gi.AddNewSlider(sliderys, "slidery")
	slidery.Dim = mat32.Y
	slidery.Value = 0.3

	slideryi := gi.AddNewSlider(sliderys, "slideryi")
	slideryi.Dim = mat32.Y
	slideryi.Value = 0.2
	slideryi.SetInactive()

	bbox := gi.AddNewButtonBox(inputs, "bbox")
	bbox.Items = []string{"Checkbox 1", "Checkbox 2", "Checkbox 3"}
	bbox.Tooltips = []string{"A description for Checkbox 1", "A description for Checkbox 2", "A description for Checkbox 3"}

	bboxr := gi.AddNewButtonBox(inputs, "bboxr")
	bboxr.Items = []string{"Radio Button 1", "Radio Button 2", "Radio Button 3"}
	bboxr.Tooltips = []string{"A description for Radio Button 1", "A description for Radio Button 2", "A description for Radio Button 3"}
	bboxr.Mutex = true

	tbuf := &giv.TextBuf{}
	tbuf.InitName(tbuf, "tbuf")
	tbuf.SetText([]byte("A keyboard-navigable, multi-line\ntext editor with support for\ncompletion and syntax highlighting"))

	tview := giv.AddNewTextView(inputs, "tview")
	tview.SetBuf(tbuf)
	tview.AddStyleFunc(gi.StyleFuncFinal, func() {
		tview.Style.MaxWidth.SetPx(500)
		tview.Style.MaxHeight.SetPx(300)
	})
}

func makeLayouts(tv *gi.TabView) {
	layouts := tv.AddNewTab(gi.TypeFrame, "Layouts").(*gi.Frame)
	layouts.Lay = gi.LayoutVert
	layouts.AddStyleFunc(gi.StyleFuncFinal, func() {
		layouts.Spacing.SetEx(1)
		layouts.Style.Padding.Set(units.Px(8))
		layouts.Style.MaxWidth.SetPx(-1)
		layouts.Style.MaxHeight.SetPx(-1)
	})

	sv := gi.AddNewSplitView(layouts, "sv")
	sv.Dim = mat32.X

	left := gi.AddNewFrame(sv, "left", gi.LayoutVert)

	leftTitle := gi.AddNewLabel(left, "leftTitle", "Left")
	leftTitle.Type = gi.LabelHeadlineMedium

	right := gi.AddNewFrame(sv, "right", gi.LayoutVert)

	rightTitle := gi.AddNewLabel(right, "rightTitle", "Right")
	rightTitle.Type = gi.LabelHeadlineMedium

}

func doWindowSetup(win *gi.Window, vp *gi.Viewport2D) {
	// Main Menu

	appnm := gi.AppName()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu.AddAppMenu(win)

	fmen := win.MainMenu.ChildByName("File", 0).(*gi.Action)
	fmen.Menu.AddAction(gi.ActOpts{Label: "New", ShortcutKey: gi.KeyFunMenuNew},
		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:New menu action triggered\n")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Open", ShortcutKey: gi.KeyFunMenuOpen},
		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:Open menu action triggered\n")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Save", ShortcutKey: gi.KeyFunMenuSave},
		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
			fmt.Printf("File:Save menu action triggered\n")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Save As..", ShortcutKey: gi.KeyFunMenuSaveAs},
		fmen.This(), func(recv, send ki.Ki, sig int64, data any) {
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
}
