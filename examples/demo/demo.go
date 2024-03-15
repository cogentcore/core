// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/states"
	"cogentcore.org/core/strcase"
	"cogentcore.org/core/styles"
)

//go:embed icon.svg
var icon []byte

func main() {
	gi.TheApp.SetIconBytes(icon)

	b := gi.NewBody("Cogent Core Demo")
	ts := gi.NewTabs(b)

	home(ts)
	text(ts)
	buttons(ts)
	inputs(ts)
	layouts(ts)
	dialogs(ts)
	values(ts)
	views(ts)
	other(ts)

	b.RunMainWindow()
}

func home(ts *gi.Tabs) {
	tab := ts.NewTab("Home")
	tab.Style(func(s *styles.Style) {
		s.Justify.Content = styles.Center
		s.Align.Content = styles.Center
		s.Align.Items = styles.Center
		s.Text.Align = styles.Center
	})

	grr.Log(gi.NewSVG(tab).ReadBytes(icon))

	gi.NewLabel(tab).SetType(gi.LabelDisplayLarge).SetText("The Cogent Core Demo")

	gi.NewLabel(tab).SetType(gi.LabelTitleLarge).SetText(`A <b>demonstration</b> of the <i>various</i> features of the <a href="https://cogentcore.org/core">Cogent Core</a> 2D and 3D Go GUI <u>framework</u>`)
}

func text(ts *gi.Tabs) {
	tab := ts.NewTab("Text")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Text")
	gi.NewLabel(tab).SetText(
		`Cogent Core provides fully customizable text elements that can be styled in any way you want. Also, there are pre-configured style types for text that allow you to easily create common text types.`)

	for _, typ := range gi.LabelTypesValues() {
		s := strcase.ToSentence(typ.String())
		gi.NewLabel(tab, "label"+typ.String()).SetType(typ).SetText(s)
	}
}

func buttons(ts *gi.Tabs) {
	tab := ts.NewTab("Buttons")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Buttons")

	gi.NewLabel(tab).SetText(
		`Cogent Core provides customizable buttons that support various events and can be styled in any way you want. Also, there are pre-configured style types for buttons that allow you to achieve common functionality with ease. All buttons support any combination of a label, icon, and indicator.`)

	makeRow := func() gi.Widget {
		return gi.NewLayout(tab).Style(func(s *styles.Style) {
			s.Wrap = true
			s.Align.Items = styles.Center
		})
	}

	gi.NewLabel(tab).SetType(gi.LabelHeadlineSmall).SetText("Standard buttons")
	brow := makeRow()
	browt := makeRow()
	browi := makeRow()

	gi.NewLabel(tab).SetType(gi.LabelHeadlineSmall).SetText("Menu buttons")
	mbrow := makeRow()
	mbrowt := makeRow()
	mbrowi := makeRow()

	menu := func(m *gi.Scene) {
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

func inputs(ts *gi.Tabs) {
	tab := ts.NewTab("Inputs")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Inputs")
	gi.NewLabel(tab).SetText(
		`Cogent Core provides various customizable input widgets that cover all common uses. Various events can be bound to inputs, and their data can easily be fetched and used wherever needed. There are also pre-configured style types for most inputs that allow you to easily switch among common styling patterns.`)

	gi.NewTextField(tab).SetPlaceholder("Text field")
	gi.NewTextField(tab).SetPlaceholder("Email").SetType(gi.TextFieldOutlined).Style(func(s *styles.Style) {
		s.VirtualKeyboard = styles.KeyboardEmail
	})
	gi.NewTextField(tab).SetPlaceholder("Phone number").AddClearButton().Style(func(s *styles.Style) {
		s.VirtualKeyboard = styles.KeyboardPhone
	})
	gi.NewTextField(tab).SetPlaceholder("URL").SetType(gi.TextFieldOutlined).AddClearButton().Style(func(s *styles.Style) {
		s.VirtualKeyboard = styles.KeyboardURL
	})
	gi.NewTextField(tab).AddClearButton().SetLeadingIcon(icons.Search)
	gi.NewTextField(tab).SetType(gi.TextFieldOutlined).SetTypePassword().SetPlaceholder("Password")
	gi.NewTextField(tab).SetText("Multiline textfield with a relatively long initial text").
		Style(func(s *styles.Style) {
			s.SetTextWrap(true)
		})

	spinners := gi.NewLayout(tab, "spinners")

	gi.NewSpinner(spinners).SetStep(5).SetMin(-50).SetMax(100).SetValue(15)
	gi.NewSpinner(spinners).SetFormat("%X").SetStep(1).SetMax(255).SetValue(44)

	choosers := gi.NewLayout(tab, "choosers")

	fruits := []gi.ChooserItem{
		{Value: "Apple", Tooltip: "A round, edible fruit that typically has red skin"},
		{Value: "Apricot", Tooltip: "A stonefruit with a yellow or orange color"},
		{Value: "Blueberry", Tooltip: "A small blue or purple berry"},
		{Value: "Blackberry", Tooltip: "A small, edible, dark fruit"},
		{Value: "Peach", Tooltip: "A fruit with yellow or white flesh and a large seed"},
		{Value: "Strawberry", Tooltip: "A widely consumed small, red fruit"},
	}

	gi.NewChooser(choosers).SetPlaceholder("Select a fruit").SetItems(fruits...).SetAllowNew(true)
	gi.NewChooser(choosers).SetPlaceholder("Select a fruit").SetItems(fruits...).SetType(gi.ChooserOutlined)
	gi.NewChooser(tab).SetEditable(true).SetPlaceholder("Select or type a fruit").SetItems(fruits...).SetAllowNew(true)
	gi.NewChooser(tab).SetEditable(true).SetPlaceholder("Select or type a fruit").SetItems(fruits...).SetType(gi.ChooserOutlined)

	gi.NewSwitch(tab).SetText("Toggle")

	gi.NewSwitches(tab).SetItems("Switch 1", "Switch 2", "Switch 3").
		SetTooltips("A description for Switch 1", "A description for Switch 2", "A description for Switch 3")

	gi.NewSwitches(tab).SetType(gi.SwitchChip).SetItems("Chip 1", "Chip 2", "Chip 3").
		SetTooltips("A description for Chip 1", "A description for Chip 2", "A description for Chip 3")

	gi.NewSwitches(tab).SetType(gi.SwitchCheckbox).SetItems("Checkbox 1", "Checkbox 2", "Checkbox 3").
		SetTooltips("A description for Checkbox 1", "A description for Checkbox 2", "A description for Checkbox 3")

	gi.NewSwitches(tab).SetType(gi.SwitchCheckbox).SetItems("Indeterminate 1", "Indeterminate 2", "Indeterminate 3").
		SetTooltips("A description for Checkbox 1", "A description for Checkbox 2", "A description for Checkbox 3").
		OnWidgetAdded(func(w gi.Widget) {
			if sw, ok := w.(*gi.Switch); ok {
				sw.SetState(true, states.Indeterminate)
			}
		})

	gi.NewSwitches(tab).SetType(gi.SwitchRadioButton).SetMutex(true).SetItems("Radio Button 1", "Radio Button 2", "Radio Button 3").
		SetTooltips("A description for Radio Button 1", "A description for Radio Button 2", "A description for Radio Button 3")

	gi.NewSwitches(tab).SetType(gi.SwitchRadioButton).SetMutex(true).SetItems("Indeterminate 1", "Indeterminate 2", "Indeterminate 3").
		SetTooltips("A description for Radio Button 1", "A description for Radio Button 2", "A description for Radio Button 3").
		OnWidgetAdded(func(w gi.Widget) {
			if sw, ok := w.(*gi.Switch); ok {
				sw.SetState(true, states.Indeterminate)
			}
		})

	gi.NewSwitches(tab).SetType(gi.SwitchSegmentedButton).SetMutex(true).SetItems("Segmented Button 1", "Segmented Button 2", "Segmented Button 3").
		SetTooltips("A description for Segmented Button 1", "A description for Segmented Button 2", "A description for Segmented Button 3")

	gi.NewSlider(tab).SetValue(0.5)
	gi.NewSlider(tab).SetValue(0.7).SetState(true, states.Disabled)

	colsliders := gi.NewLayout(tab)

	gi.NewSlider(colsliders).SetValue(0.3).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	gi.NewSlider(colsliders).SetValue(0.2).SetState(true, states.Disabled).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
}

func layouts(ts *gi.Tabs) {
	tab := ts.NewTab("Layouts")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Layout")
	gi.NewLabel(tab).SetText(
		`Cogent Core provides various adaptable layout types that allow you to easily organize content so that it is easy to use, customize, and understand.`)

	// vw := gi.NewLabel(layouts, "vw", "50vw")
	// vw.Style(func(s *styles.Style) {
	// 	s.Min.X.Vw(50)
	// 	s.BackgroundColor.SetSolid(colors.Scheme.Primary.Base)
	// 	s.Color = colors.C(colors.Scheme.Primary.On)
	// })

	// pw := gi.NewLabel(layouts, "pw", "50pw")
	// pw.Style(func(s *styles.Style) {
	// 	s.Min.X.Pw(50)
	// 	s.BackgroundColor.SetSolid(colors.Scheme.Primary.Container)
	// 	s.Color = colors.C(colors.Scheme.Primary.OnContainer)
	// })

	sv := gi.NewSplits(tab)

	left := gi.NewFrame(sv).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
	})
	gi.NewLabel(left).SetType(gi.LabelHeadlineMedium).SetText("Left")
	right := gi.NewFrame(sv).Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
	})
	gi.NewLabel(right).SetType(gi.LabelHeadlineMedium).SetText("Right")
}

func dialogs(ts *gi.Tabs) {
	tab := ts.NewTab("Dialogs")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Dialogs, snackbars, and windows")
	gi.NewLabel(tab).SetText(
		`Cogent Core provides completely customizable dialogs, snackbars, and windows that allow you to easily display, obtain, and organize information.`)

	makeRow := func() gi.Widget {
		return gi.NewLayout(tab).Style(func(s *styles.Style) {
			s.Wrap = true
			s.Align.Items = styles.Center
		})
	}

	gi.NewLabel(tab).SetType(gi.LabelHeadlineSmall).SetText("Dialogs")
	drow := makeRow()

	md := gi.NewButton(drow).SetText("Message")
	md.OnClick(func(e events.Event) {
		gi.MessageDialog(md, "Something happened", "Message")
	})

	ed := gi.NewButton(drow).SetText("Error")
	ed.OnClick(func(e events.Event) {
		gi.ErrorDialog(ed, errors.New("invalid encoding format"), "Error loading file")
	})

	cd := gi.NewButton(drow).SetText("Confirm")
	cd.OnClick(func(e events.Event) {
		d := gi.NewBody().AddTitle("Confirm").AddText("Send message?")
		d.AddBottomBar(func(pw gi.Widget) {
			d.AddCancel(pw).OnClick(func(e events.Event) {
				fmt.Println("Dialog canceled")
			})
			d.AddOk(pw).OnClick(func(e events.Event) {
				fmt.Println("Dialog accepted")
			})
		})
		d.NewDialog(cd).Run()
	})

	td := gi.NewButton(drow).SetText("Input")
	td.OnClick(func(e events.Event) {
		d := gi.NewBody().AddTitle("Input").AddText("What is your name?")
		tf := gi.NewTextField(d)
		d.AddBottomBar(func(pw gi.Widget) {
			d.AddCancel(pw)
			d.AddOk(pw).OnClick(func(e events.Event) {
				fmt.Println("Your name is", tf.Text())
			})
		})
		d.NewDialog(td).Run()
	})

	fd := gi.NewButton(drow).SetText("Full window")
	u := &gi.User{}
	fd.OnClick(func(e events.Event) {
		d := gi.NewBody().AddTitle("Full window dialog").AddText("Edit your information")
		giv.NewStructView(d).SetStruct(u).OnInput(func(e events.Event) {
			fmt.Println("Got input event")
		})
		d.OnClose(func(e events.Event) {
			fmt.Println("Your information is:", u)
		})
		d.NewFullDialog(td).Run()
	})

	nd := gi.NewButton(drow).SetText("New window")
	nd.OnClick(func(e events.Event) {
		gi.NewBody().AddTitle("New window dialog").AddText("This dialog opens in a new window on multi-window platforms").NewDialog(nd).SetNewWindow(true).Run()
	})

	gi.NewLabel(tab).SetType(gi.LabelHeadlineSmall).SetText("Snackbars")
	srow := makeRow()

	ms := gi.NewButton(srow).SetText("Message")
	ms.OnClick(func(e events.Event) {
		gi.MessageSnackbar(ms, "New messages loaded")
	})

	es := gi.NewButton(srow).SetText("Error")
	es.OnClick(func(e events.Event) {
		gi.ErrorSnackbar(es, errors.New("file not found"), "Error loading page")
	})

	cs := gi.NewButton(srow).SetText("Custom")
	cs.OnClick(func(e events.Event) {
		gi.NewBody().AddSnackbarText("Files updated").
			AddSnackbarButton("Refresh", func(e events.Event) {
				fmt.Println("Refreshed files")
			}).AddSnackbarIcon(icons.Close).NewSnackbar(cs).Run()
	})

	gi.NewLabel(tab).SetType(gi.LabelHeadlineSmall).SetText("Windows")
	wrow := makeRow()

	nw := gi.NewButton(wrow).SetText("New window")
	nw.OnClick(func(e events.Event) {
		gi.NewBody().AddTitle("New window").AddText("A standalone window that opens in a new window on multi-window platforms").NewWindow().Run()
	})

	fw := gi.NewButton(wrow).SetText("Full window")
	fw.OnClick(func(e events.Event) {
		gi.NewBody().AddTitle("Full window").AddText("A standalone window that opens in the same system window").NewWindow().SetNewWindow(false).Run()
	})
}

func values(ts *gi.Tabs) {
	tab := ts.NewTab("Values")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Values")
	gi.NewLabel(tab).SetText(
		`Cogent Core provides the giv value system, which allows you to instantly turn Go values and functions into type-specific widgets bound to the original values. This powerful system means that you can automatically turn backend data structures into GUI apps with just a single simple line of code. For example, you can dynamically edit this very GUI right now by clicking the first button below.`)

	gi.NewButton(tab).SetText("Inspector").OnClick(func(e events.Event) {
		giv.InspectorWindow(ts.Scene)
	})

	giv.NewValue(tab, colors.Orange)
	giv.NewValue(tab, time.Now())
	giv.NewValue(tab, 5*time.Minute)
	giv.NewValue(tab, 500*time.Millisecond).SetTags(map[string]string{"min": "10ms", "max": "10s", "step": "10ms"})
	giv.NewValue(tab, gi.Filename("demo.go"))
	giv.NewValue(tab, gi.AppearanceSettings.FontFamily)
	giv.NewValue(tab, giv.ColorMapName("ColdHot"))
	giv.NewFuncButton(tab, hello).SetShowReturn(true)
}

// Hello displays a greeting message and an age in weeks based on the given information.
func hello(firstName string, lastName string, age int, likesGo bool) (greeting string, weeksOld int) { //gti:add
	weeksOld = age * 52
	greeting = "Hello, " + firstName + " " + lastName + "! "
	if likesGo {
		greeting += "I'm glad to hear that you like the best programming language!"
	} else {
		greeting += "You should reconsider what programming languages you like."
	}
	return
}

func views(ts *gi.Tabs) {
	tab := ts.NewTab("Views")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Views")
	gi.NewLabel(tab).SetText(
		`Cogent Core provides powerful views that allow you to easily view and edit complex data types like structs, maps, and slices, allowing you to easily create widgets like lists, tables, and forms.`)

	vts := gi.NewTabs(tab)

	str := testStruct{
		Name:   "happy",
		Cond:   2,
		Val:    3.1415,
		Vec:    mat32.V2(5, 7),
		Inline: inlineStruct{Val: 3},
		Cond2: tableStruct{
			IntField:   22,
			FloatField: 44.4,
			StrField:   "fi",
			File:       "views.go",
		},
		Things: make([]tableStruct, 2),
		Stuff:  make([]float32, 3),
	}

	giv.NewStructView(vts.NewTab("Struct view")).SetStruct(&str)

	mp := map[string]string{}

	mp["mapkey1"] = "whatever"
	mp["mapkey2"] = "testing"
	mp["mapkey3"] = "boring"

	giv.NewMapView(vts.NewTab("Map view")).SetMap(&mp)

	sl := make([]string, 20)

	for i := 0; i < len(sl); i++ {
		sl[i] = fmt.Sprintf("el: %v", i)
	}
	sl[10] = "this is a particularly long slice value"

	giv.NewSliceView(vts.NewTab("Slice view")).SetSlice(&sl)

	tbl := make([]*tableStruct, 100)

	for i := range tbl {
		ts := &tableStruct{IntField: i, FloatField: float32(i) / 10}
		tbl[i] = ts
	}

	tbl[0].StrField = "this is a particularly long field"

	giv.NewTableView(vts.NewTab("Table view")).SetSlice(&tbl)
}

type tableStruct struct { //gti:add

	// an icon
	Icon icons.Icon

	// an integer field
	IntField int `default:"2"`

	// a float field
	FloatField float32

	// a string field
	StrField string

	// a file
	File gi.Filename
}

type inlineStruct struct { //gti:add

	// click to show next
	On bool

	// can u see me?
	ShowMe string

	// a conditional
	Cond int

	// On and Cond=0
	Cond1 string

	// if Cond=0
	Cond2 tableStruct

	// a value
	Val float32
}

func (il *inlineStruct) ShouldShow(field string) bool {
	switch field {
	case "ShowMe", "Cond":
		return il.On
	case "Cond1":
		return il.On && il.Cond == 0
	case "Cond2":
		return il.On && il.Cond <= 1
	}
	return true
}

type testStruct struct { //gti:add

	// An enum value
	Enum gi.ButtonTypes

	// a string
	Name string

	// click to show next
	ShowNext bool

	// can u see me?
	ShowMe string

	// how about that
	Inline inlineStruct `view:"inline"`

	// a conditional
	Cond int

	// if Cond=0
	Cond1 string

	// if Cond>=0
	Cond2 tableStruct

	// a value
	Val float32

	Vec mat32.Vec2

	Things []tableStruct

	Stuff []float32

	// a file
	File gi.Filename
}

func (ts *testStruct) ShouldShow(field string) bool {
	switch field {
	case "Name":
		return ts.Enum <= gi.ButtonElevated
	case "ShowMe":
		return ts.ShowNext
	case "Cond1":
		return ts.Cond == 0
	case "Cond2":
		return ts.Cond >= 0
	}
	return true
}

func other(ts *gi.Tabs) {
	tab := ts.NewTab("Other")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Other")
	gi.NewLabel(tab).SetText(`Other features of the Cogent Core framework`)

	gi.NewMeter(tab).SetType(gi.MeterCircle).SetValue(0.7).SetText("70%")
	gi.NewMeter(tab).SetType(gi.MeterSemicircle).SetValue(0.7).SetText("70%")
	gi.NewMeter(tab).SetValue(0.7)
	gi.NewMeter(tab).SetValue(0.7).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
}
