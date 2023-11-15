// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
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

	gi.SetAppName("widgets")
	gi.SetAppAbout(`This is a demo of the main widgets and general functionality of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://goki.dev/gi/v2/blob/master/examples/widgets/README.md">README</a> page for this example app has lots of further info.</p>`)

	sc := gi.NewScene("widgets").SetTitle("GoGi Widgets Demo")

	// gi.DefaultTopAppBar = nil // turns it off

	sc.TopAppBar = func(tb *gi.TopAppBar) {
		if gi.DefaultTopAppBar != nil {
			gi.DefaultTopAppBar(tb)
		}
		gi.NewButton(tb).SetText("Button 1").SetData(1).
			OnClick(func(e events.Event) {
				fmt.Println("TopAppBar Button 1")
				gi.NewSnackbar(tb).Text("Something went wrong!").
					Button("Try again", func(e events.Event) {
						fmt.Println("got snackbar try again event")
					}).
					Icon(icons.Close, func(e events.Event) {
						fmt.Println("got snackbar close icon event")
					}).Run()
			})
		gi.NewButton(tb).SetText("Button 2").SetData(2).
			OnClick(func(e events.Event) {
				fmt.Println("TopAppBar Button 2")
			})
	}

	trow := gi.NewLayout(sc, "trow")
	trow.Style(func(s *styles.Style) {
		s.SetMainAxis(mat32.X)
		s.Align.X = styles.AlignCenter
	})

	giedsc := keyfun.ChordFor(keyfun.GoGiEditor)
	prsc := keyfun.ChordFor(keyfun.Prefs)

	gi.NewLabel(trow, "title").SetText(
		`This is a <b>demonstration</b> of the
		<span style="color:red">various</span> <a href="https://goki.dev/gi/v2">GoGi</a> <i>Widgets</i><br>
		<small>Shortcuts: <kbd>` + string(prsc) + `</kbd> = Preferences,
		<kbd>` + string(giedsc) + `</kbd> = Editor, <kbd>Ctrl/Cmd +/-</kbd> = zoom</small><br>
		See <a href="https://goki.dev/gi/v2/blob/master/examples/widgets/README.md">README</a> for detailed info and things to try.`).
		SetType(gi.LabelHeadlineSmall).
		Style(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Text.Align = styles.AlignCenter
			// s.Text.AlignV = styles.AlignCenter
			s.Font.Family = "Times New Roman, serif"
		})

	//////////////////////////////////////////
	//      Buttons

	gi.NewSpace(sc)
	gi.NewLabel(sc).SetText("Buttons:")

	brow := gi.NewLayout(sc, "brow").
		Style(func(s *styles.Style) {
			s.SetMainAxis(mat32.X)
			s.Gap.X.Em(1)
		})

	b1 := gi.NewButton(brow).SetIcon(icons.OpenInNew).SetTooltip("press this <i>button</i> to pop up a dialog box").
		Style(func(s *styles.Style) {
			s.Min.X.Em(1.5)
			s.Min.Y.Em(1.5)
		})

	b1.OnClick(func(e events.Event) {
		fmt.Printf("Button1 clicked\n")
		gi.NewDialog(b1).Title("Test Dialog").Prompt("This is a prompt").
			Modal(true).NewWindow(true).Cancel().Ok().Run()

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

	button2 := gi.NewButton(brow).SetText("Open GoGiEditor").
		SetTooltip("This button will open the GoGi GUI editor where you can edit this very GUI and see it update dynamically as you change things")
	button2.OnClick(func(e events.Event) {
		txt := ""
		d := gi.NewDialog(button2).Title("What is it?").Prompt("Please enter your response:")
		giv.NewValue(d, &txt).AsWidget().(*gi.TextField).SetPlaceholder("Enter string here...")
		d.Cancel().Ok().OnAccept(func(e events.Event) {
			fmt.Println("dialog accepted; string entered:", txt)
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

		m2 := gi.NewButton(m).SetText("Menu Item 2").SetIcon(icons.Open).SetData(2)
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

	gi.NewSpace(sc)
	gi.NewLabel(sc).SetText("Sliders:")

	srow := gi.NewLayout(sc).
		Style(func(s *styles.Style) {
			s.SetMainAxis(mat32.X)
			s.Align.Y = styles.AlignCenter
			s.Gap.X.Ex(2)
		})

	slider0 := gi.NewSlider(srow).SetDim(mat32.X).SetValue(0.5).
		SetSnap(true).SetTracking(false).SetIcon(icons.RadioButtonChecked)
	slider0.OnChange(func(e events.Event) {
		fmt.Println("slider0", slider0.Value)
	})
	slider0.Style(func(s *styles.Style) {
		s.Align.Y = styles.AlignCenter
	})

	slider1 := gi.NewSlider(srow).SetDim(mat32.Y).
		SetTracking(true).SetValue(0.5).SetThumbSize(mat32.NewVec2(1, 4))
	slider1.OnChange(func(e events.Event) {
		fmt.Println("slider1", slider1.Value)
	})

	scroll0 := gi.NewSlider(srow).SetType(gi.SliderScrollbar).SetDim(mat32.X).
		SetVisiblePct(0.25).SetValue(0.25).SetTracking(true).SetStep(0.05).SetSnap(true)
	scroll0.OnChange(func(e events.Event) {
		fmt.Println("scroll0", scroll0.Value)
	})
	scroll0.Style(func(s *styles.Style) {
		s.Align.Y = styles.AlignCenter
	})

	scroll1 := gi.NewSlider(srow).SetType(gi.SliderScrollbar).SetDim(mat32.Y).
		SetVisiblePct(.01).SetValue(0).SetMax(3000).
		SetTracking(true).SetStep(1).SetPageStep(10)
	scroll1.OnChange(func(e events.Event) {
		fmt.Println("scroll1", scroll1.Value)
	})

	//////////////////////////////////////////
	//      Text Widgets

	gi.NewSpace(sc)
	gi.NewLabel(sc).SetText("Text Widgets:")

	txrow := gi.NewLayout(sc).
		Style(func(s *styles.Style) {
			s.SetMainAxis(mat32.X)
			s.Gap.X.Ex(2)
		})

	edit1 := gi.NewTextField(txrow, "edit1").SetPlaceholder("Enter text here...").AddClearButton()
	edit1.OnChange(func(e events.Event) {
		fmt.Println("Text:", edit1.Text())
	})
	edit1.Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})

	sb := gi.NewSpinner(txrow).SetMax(1000).SetMin(-1000).SetStep(5)
	sb.OnChange(func(e events.Event) {
		fmt.Println("spinbox value changed to", sb.Value)
	})

	ch := gi.NewChooser(txrow).SetType(gi.ChooserOutlined).SetEditable(true).
		SetTypes(gti.AllEmbeddersOf(gi.WidgetBaseType), true, true, 50)
	// ItemsFromEnum(gi.ButtonTypesN, true, 50)
	ch.OnChange(func(e events.Event) {
		fmt.Printf("Chooser selected index: %d data: %v\n", ch.CurIndex, ch.CurVal)
	})

	/*  todo:
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
