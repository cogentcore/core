// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/styles"
)

func main() {
	b := gi.NewBody("Cogent Core Widgets Demo")

	b.AddAppBar(func(tb *gi.Toolbar) { // put first in top app bar
		gi.NewButton(tb).SetText("Button 1").
			OnClick(func(e events.Event) {
				fmt.Println("AppBar Button 1")
				gi.NewBody().AddSnackbarText("Something went wrong!").
					AddSnackbarButton("Try again", func(e events.Event) {
						fmt.Println("got snackbar try again event")
					}).
					AddSnackbarIcon(icons.Close, func(e events.Event) {
						fmt.Println("got snackbar close icon event")
					}).NewSnackbar(tb).Run()
			})
		gi.NewButton(tb).SetText("Button 2").
			OnClick(func(e events.Event) {
				fmt.Println("AppBar Button 2")
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
		<span style="color:red">various</span> <a href="https://cogentcore.org/core/gi">Cogent Core</a> <i>Widgets</i><br>
		<small>Shortcuts: <kbd>` + string(prsc) + `</kbd> = Preferences,
		<kbd>` + string(giedsc) + `</kbd> = Editor, <kbd>Ctrl/Cmd +/-</kbd> = zoom</small><br>
		See <a href="https://cogentcore.org/core/gi/blob/master/examples/widgets/README.md">README</a> for detailed info and things to try.`).
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

	b1 := gi.NewButton(brow).SetIcon(icons.OpenInNew).SetTooltip("press this <i>button</i> to pop up a dialog box")

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
		SetTooltip("This button will open the Cogent Core GUI editor where you can edit this very GUI and see it update dynamically as you change things")
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
		fmt.Println("toggled", toggle.IsChecked())
	})

	mb := gi.NewButton(brow).SetText("Menu Button")
	mb.SetTooltip("Press this button to pull up a nested menu of buttons")

	mb.Menu = func(m *gi.Scene) {
		m1 := gi.NewButton(m).SetText("Menu Item 1").SetIcon(icons.Save).SetShortcut("Shift+Control+1").
			SetTooltip("A standard menu item with an icon")
		m1.OnClick(func(e events.Event) {
			fmt.Println("Clicked on menu item 1")
		})

		m2 := gi.NewButton(m).SetText("Menu Item 2").SetIcon(icons.Open).
			SetTooltip("A menu item with an icon and a sub menu")

		m2.Menu = func(m *gi.Scene) {
			sm2 := gi.NewButton(m).SetText("Sub Menu Item 2").SetIcon(icons.InstallDesktop).
				SetTooltip("A sub menu item with an icon")
			sm2.OnClick(func(e events.Event) {
				fmt.Println("Clicked on sub menu item 2")
			})
		}

		gi.NewSeparator(m)

		m3 := gi.NewButton(m).SetText("Menu Item 3").SetIcon(icons.Favorite).SetShortcut("Control+3").
			SetTooltip("A standard menu item with an icon, below a separator")
		m3.OnClick(func(e events.Event) {
			fmt.Println("Clicked on menu item 3")
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

	slider0 := gi.NewSlider(srow).SetValue(0.5).SetSnap(true).SetIcon(icons.RadioButtonChecked)
	slider0.OnChange(func(e events.Event) {
		fmt.Println("slider0", slider0.Value)
	})

	slider1 := gi.NewSlider(srow).SetValue(0.5).SetThumbSize(mat32.V2(1, 4))
	slider1.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	slider1.OnInput(func(e events.Event) {
		fmt.Println("slider1", slider1.Value)
	})

	scroll0 := gi.NewSlider(srow).SetType(gi.SliderScrollbar).
		SetVisiblePct(0.25).SetValue(0.25).SetStep(0.05).SetSnap(true)
	scroll0.OnInput(func(e events.Event) {
		fmt.Println("scroll0", scroll0.Value)
	})

	scroll1 := gi.NewSlider(srow).SetType(gi.SliderScrollbar).SetVisiblePct(.01).
		SetValue(0).SetMax(3000).SetStep(1).SetPageStep(10)
	scroll1.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
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
		SetTypes(gti.AllEmbeddersOf(gi.WidgetBaseType), true, true)
	// ItemsFromEnum(gi.ButtonTypesN, true, 50)
	ch.OnChange(func(e events.Event) {
		fmt.Printf("Chooser selected index: %d data: %v\n", ch.CurIndex, ch.CurVal)
	})

	b.RunMainWindow()
}
