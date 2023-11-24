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

	b := gi.NewBody("widgets").SetTitle("GoGi Widgets Demo")

	// gi.DefaultTopAppBar = nil // turns it off

	b.AddTopBar(func(pw gi.Widget) {
		tb := b.DefaultTopAppBar(pw)
		gi.NewButton(tb).SetText("Button 1").SetData(1).
			OnClick(func(e events.Event) {
				fmt.Println("TopAppBar Button 1")
				gi.NewSnackbar(tb).AddSnackbarText("Something went wrong!").
					AddSnackbarButton("Try again", func(e events.Event) {
						fmt.Println("got snackbar try again event")
					}).
					AddSnackbarIcon(icons.Close, func(e events.Event) {
						fmt.Println("got snackbar close icon event")
					}).Stage.Run()
			})
		gi.NewButton(tb).SetText("Button 2").SetData(2).
			OnClick(func(e events.Event) {
				fmt.Println("TopAppBar Button 2")
			})
	})

	trow := gi.NewLayout(b, "trow")
	trow.Style(func(s *styles.Style) {
		s.Align.Items = styles.Center
		s.Align.Content = styles.Center
	})

	giedsc := keyfun.ChordFor(keyfun.Inspector)
	prsc := keyfun.ChordFor(keyfun.Prefs)

	gi.NewLabel(trow, "title").SetText(
		`This is a <b>demonstration</b> of the
		<span style="color:red">various</span> <a href="https://goki.dev/gi/v2">GoGi</a> <i>Widgets</i><br>
		<small>Shortcuts: <kbd>` + string(prsc) + `</kbd> = Preferences,
		<kbd>` + string(giedsc) + `</kbd> = Editor, <kbd>Ctrl/Cmd +/-</kbd> = zoom</small><br>
		See <a href="https://goki.dev/gi/v2/blob/master/examples/widgets/README.md">README</a> for detailed info and things to try.`).
		SetType(gi.LabelHeadlineSmall).
		Style(func(s *styles.Style) {
			s.Text.Align = styles.Center
			// s.Text.AlignV = styles.Center
			s.Font.Family = "Times New Roman, serif"
		})

	//////////////////////////////////////////
	//      Buttons

	gi.NewSpace(b)
	gi.NewLabel(b).SetText("Buttons:")

	brow := gi.NewLayout(b, "brow").
		Style(func(s *styles.Style) {
			s.Gap.X.Em(1)
		})

	b1 := gi.NewButton(brow).SetIcon(icons.OpenInNew).SetTooltip("press this <i>button</i> to pop up a dialog box").
		Style(func(s *styles.Style) {
			s.Min.X.Em(1.5)
			s.Min.Y.Em(1.5)
		})

	b1.OnClick(func(e events.Event) {
		fmt.Printf("Button1 clicked\n")
		b := gi.NewBody().AddTitle("Test Dialog").AddText("This is a prompt")
		b.AddBottomBar(func(pw gi.Widget) {
			b.AddCancel(pw)
			b.AddOk(pw)
		})
		b.NewDialog(b1).Run()
	})

	button2 := gi.NewButton(brow).SetText("Open Inspector").
		SetTooltip("This button will open the GoGi GUI editor where you can edit this very GUI and see it update dynamically as you change things")
	button2.OnClick(func(e events.Event) {
		txt := ""
		d := gi.NewBody().AddTitle("What is it?").AddText("Please enter your response:")
		giv.NewValue(d, &txt).AsWidget().(*gi.TextField).SetPlaceholder("Enter string here...")
		d.AddBottomBar(func(pw gi.Widget) {
			d.AddCancel(pw)
			d.AddOk(pw).OnClick(func(e events.Event) {
				fmt.Println("dialog accepted; string entered:", txt)
			})
		})
		d.NewDialog(button2).Run()
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

	gi.NewSpace(b)
	gi.NewLabel(b).SetText("Sliders:")

	srow := gi.NewLayout(b).
		Style(func(s *styles.Style) {
			s.Align.Items = styles.Center
			s.Gap.X.Ex(2)
		})

	slider0 := gi.NewSlider(srow).SetDim(mat32.X).SetValue(0.5).SetSnap(true).SetIcon(icons.RadioButtonChecked)
	slider0.OnChange(func(e events.Event) {
		fmt.Println("slider0", slider0.Value)
	})

	slider1 := gi.NewSlider(srow).SetDim(mat32.Y).SetValue(0.5).SetThumbSize(mat32.NewVec2(1, 4))
	slider1.OnInput(func(e events.Event) {
		fmt.Println("slider1", slider1.Value)
	})

	scroll0 := gi.NewSlider(srow).SetType(gi.SliderScrollbar).SetDim(mat32.X).
		SetVisiblePct(0.25).SetValue(0.25).SetStep(0.05).SetSnap(true)
	scroll0.OnInput(func(e events.Event) {
		fmt.Println("scroll0", scroll0.Value)
	})

	scroll1 := gi.NewSlider(srow).SetType(gi.SliderScrollbar).SetDim(mat32.Y).SetVisiblePct(.01).
		SetValue(0).SetMax(3000).SetStep(1).SetPageStep(10)
	scroll1.OnInput(func(e events.Event) {
		fmt.Println("scroll1", scroll1.Value)
	})

	//////////////////////////////////////////
	//      Text Widgets

	gi.NewLabel(b).SetText("Text Widgets:")

	txrow := gi.NewLayout(b).
		Style(func(s *styles.Style) {
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

	b.NewWindow().Run().Wait()
}
