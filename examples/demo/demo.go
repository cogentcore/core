// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"
	"fmt"
	"strings"
	"time"

	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/glop/sentence"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

func app() {
	b := gi.NewAppBody("gogi-demo").SetTitle("GoGi Demo")
	b.App().About = "The GoGi Demo demonstrates the various features of the GoGi 2D and 3D Go GUI framework."

	ts := gi.NewTabs(b)
	ts.DeleteTabButtons = false

	makeHome(ts)
	makeText(ts)
	makeButtons(ts)
	makeInputs(ts)
	makeLayouts(ts)
	makeValues(ts)

	b.NewWindow().Run().Wait()
}

//go:embed .goki/icons/512.png
var giLogo embed.FS

func makeHome(ts *gi.Tabs) {
	home := ts.NewTab("Home")
	home.Style(func(s *styles.Style) {
		s.Justify.Content = styles.Center
		s.Align.Content = styles.Center
		s.Align.Items = styles.Center
		s.Text.Align = styles.Center
	})

	img := gi.NewImage(home)
	grr.Log(img.OpenImageFS(giLogo, ".goki/icons/512.png"))
	img.Style(func(s *styles.Style) {
		s.Min.Set(units.Dp(256))
	})

	gi.NewLabel(home).SetType(gi.LabelDisplayLarge).SetText("The GoGi Demo")

	gi.NewLabel(home).SetType(gi.LabelTitleLarge).SetText(`A <b>demonstration</b> of the <i>various</i> features of the <a href="https://goki.dev/gi">GoGi</a> 2D and 3D Go GUI <u>framework</u>`)
}

func makeText(ts *gi.Tabs) {
	text := ts.NewTab("Text")

	gi.NewLabel(text).SetType(gi.LabelHeadlineLarge).SetText("Text")
	gi.NewLabel(text).SetText(
		`GoGi provides fully customizable text elements that can be styled in any way you want. Also, there are pre-configured style types for text that allow you to easily create common text types.`)

	for _, typ := range gi.LabelTypesValues() {
		s := sentence.Case(typ.String())
		gi.NewLabel(text, "label"+typ.String()).SetType(typ).SetText(s)
	}
}

func makeButtons(ts *gi.Tabs) {
	buttons := ts.NewTab("Buttons")

	gi.NewLabel(buttons).SetType(gi.LabelHeadlineLarge).SetText("Buttons")

	gi.NewLabel(buttons).SetText(
		`GoGi provides customizable buttons that support various events and can be styled in any way you want. Also, there are pre-configured style types for buttons that allow you to achieve common functionality with ease. All buttons support any combination of a label, icon, and indicator.`)

	makeRow := func() gi.Widget {
		return gi.NewLayout(buttons).Style(func(s *styles.Style) {
			s.Wrap = true
			s.Align.Items = styles.Center
		})
	}

	gi.NewLabel(buttons).SetType(gi.LabelHeadlineSmall).SetText("Standard buttons")
	brow := makeRow()
	browt := makeRow()
	browi := makeRow()

	gi.NewLabel(buttons).SetType(gi.LabelHeadlineSmall).SetText("Menu buttons")
	mbrow := makeRow()
	mbrowt := makeRow()
	mbrowi := makeRow()

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
		b.SetState(true, states.Disabled)

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

	gi.NewLabel(inputs).SetType(gi.LabelHeadlineLarge).SetText("Inputs")

	gi.NewLabel(inputs).SetType(gi.LabelBodyLarge).SetText(
		`GoGi provides various customizable input widgets that cover all common uses. Various events can be bound to inputs, and their data can easily be fetched and used wherever needed. There are also pre-configured style types for most inputs that allow you to easily switch among common styling patterns.`)

	gi.NewTextField(inputs).SetPlaceholder("Filled")
	gi.NewTextField(inputs).SetType(gi.TextFieldOutlined).SetPlaceholder("Outlined")
	gi.NewTextField(inputs).AddClearButton()
	gi.NewTextField(inputs).SetType(gi.TextFieldOutlined).AddClearButton()
	gi.NewTextField(inputs).AddClearButton().SetLeadingIcon(icons.Search)
	gi.NewTextField(inputs).SetType(gi.TextFieldOutlined).AddClearButton().SetLeadingIcon(icons.Search)
	gi.NewTextField(inputs).SetTypePassword().SetPlaceholder("Password")
	gi.NewTextField(inputs).SetType(gi.TextFieldOutlined).SetTypePassword().SetPlaceholder("Password")

	gi.NewLabeledTextField(inputs).SetLabel("Labeled text field").SetHintText("Hint text")

	spinners := gi.NewLayout(inputs, "spinners")

	gi.NewSpinner(spinners).SetStep(5).SetMin(-50).SetMax(100).SetValue(15)
	gi.NewSpinner(spinners).SetFormat("%#X").SetStep(1).SetMax(255).SetValue(44)

	choosers := gi.NewLayout(inputs, "choosers")

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

	gi.NewSwitches(inputs).SetType(gi.SwitchSegmentedButton).SetMutex(true).SetItems([]string{"Segmented Button 1", "Segmented Button 2", "Segmented Button 3"}).
		SetTooltips([]string{"A description for Segmented Button 1", "A description for Segmented Button 2", "A description for Segmented Button 3"})

	gi.NewSlider(inputs).SetDim(mat32.X).SetValue(0.5)
	gi.NewSlider(inputs).SetDim(mat32.X).SetValue(0.7).SetState(true, states.Disabled)

	sliderys := gi.NewLayout(inputs, "sliderys")

	gi.NewSlider(sliderys).SetDim(mat32.Y).SetValue(0.3)
	gi.NewSlider(sliderys).SetDim(mat32.Y).SetValue(0.2).SetState(true, states.Disabled)

	tb := texteditor.NewBuf()
	tb.NewBuf(0)
	tb.SetText([]byte("A keyboard-navigable, multi-line\ntext editor with support for\ncompletion and syntax highlighting"))
	texteditor.NewEditor(inputs).SetBuf(tb)
}

func makeLayouts(ts *gi.Tabs) {
	layouts := ts.NewTab("Layouts")

	gi.NewLabel(layouts).SetType(gi.LabelHeadlineLarge).SetText("Layout")

	gi.NewLabel(layouts).SetType(gi.LabelBodyLarge).SetText(
		`GoGi provides various adaptable layout types that allow you to easily organize content so that it is easy to use, customize, and understand.`)

	// vw := gi.NewLabel(layouts, "vw", "50vw")
	// vw.Style(func(s *styles.Style) {
	// 	s.Min.X.Vw(50)
	// 	s.BackgroundColor.SetSolid(colors.Scheme.Primary.Base)
	// 	s.Color = colors.Scheme.Primary.On
	// })

	// pw := gi.NewLabel(layouts, "pw", "50pw")
	// pw.Style(func(s *styles.Style) {
	// 	s.Min.X.Pw(50)
	// 	s.BackgroundColor.SetSolid(colors.Scheme.Primary.Container)
	// 	s.Color = colors.Scheme.Primary.OnContainer
	// })

	sv := gi.NewSplits(layouts).SetDim(mat32.X)

	left := gi.NewFrame(sv).Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
	})
	gi.NewLabel(left).SetType(gi.LabelHeadlineMedium).SetText("Left")
	right := gi.NewFrame(sv).Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
	})
	gi.NewLabel(right).SetType(gi.LabelHeadlineMedium).SetText("Right")
}

func makeValues(ts *gi.Tabs) {
	values := ts.NewTab("Values")

	gi.NewLabel(values).SetType(gi.LabelHeadlineLarge).SetText("Values")

	gi.NewLabel(values).SetType(gi.LabelBodyLarge).SetText(
		`GoGi provides the giv value system, which allows you to instantly turn Go values and functions into type-specific widgets bound to the original values. This powerful system means that you can automatically turn backend data structures into GUI apps with just a single simple line of code. For example, you can dynamically edit this very GUI right now by clicking the first button below.`)

	gi.NewButton(values).SetText("Inspector").OnClick(func(e events.Event) {
		giv.InspectorDialog(ts.Sc)
	})

	giv.NewValue(values, colors.Orange)
	giv.NewValue(values, time.Now())
	giv.NewValue(values, gi.FileName("demo.go"))
	giv.NewValue(values, giv.ColorMapName("ColdHot"))
	giv.NewValue(values, hello)
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
