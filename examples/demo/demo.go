// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	_ "embed"
	"errors"
	"fmt"
	"math/rand"
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
	"cogentcore.org/core/units"
)

//go:embed icon.svg
var appIcon []byte

func main() {
	gi.TheApp.SetIconBytes(appIcon)

	b := gi.NewBody("Cogent Core Demo")
	ts := gi.NewTabs(b)

	home(ts)
	widgets(ts)
	views(ts)
	values(ts)
	style(ts)

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

	grr.Log(gi.NewSVG(tab).ReadBytes(appIcon))

	gi.NewLabel(tab).SetType(gi.LabelDisplayLarge).SetText("The Cogent Core Demo")

	gi.NewLabel(tab).SetType(gi.LabelTitleLarge).SetText(`A <b>demonstration</b> of the <i>various</i> features of the <a href="https://cogentcore.org/core">Cogent Core</a> 2D and 3D Go GUI <u>framework</u>`)
}

func widgets(ts *gi.Tabs) {
	wts := gi.NewTabs(ts.NewTab("Widgets"))

	text(wts)
	buttons(wts)
	inputs(wts)
	sliders(wts)
	dialogs(wts)
	icon(wts)
}

func text(ts *gi.Tabs) {
	tab := ts.NewTab("Text")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Text")
	gi.NewLabel(tab).SetText("Cogent Core provides fully customizable text elements that can be styled in any way you want. Also, there are pre-configured style types for text that allow you to easily create common text types.")

	for _, typ := range gi.LabelTypesValues() {
		s := strcase.ToSentence(typ.String())
		gi.NewLabel(tab, "label"+typ.String()).SetType(typ).SetText(s)
	}
}

func buttons(ts *gi.Tabs) {
	tab := ts.NewTab("Buttons")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Buttons")

	gi.NewLabel(tab).SetText("Cogent Core provides customizable buttons that support various events and can be styled in any way you want. Also, there are pre-configured style types for buttons that allow you to achieve common functionality with ease. All buttons support any combination of a label, icon, and indicator.")

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
	gi.NewLabel(tab).SetText("Cogent Core provides various customizable input widgets that cover all common uses. Various events can be bound to inputs, and their data can easily be fetched and used wherever needed. There are also pre-configured style types for most inputs that allow you to easily switch among common styling patterns.")

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

	gi.NewSwitches(tab).SetItems(
		gi.SwitchItem{"Switch 1", "A description for Switch 1"},
		gi.SwitchItem{"Switch 2", "A description for Switch 2"},
		gi.SwitchItem{"Switch 3", "A description for Switch 3"})

	gi.NewSwitches(tab).SetType(gi.SwitchChip).SetItems(
		gi.SwitchItem{"Chip 1", "A description for Chip 1"},
		gi.SwitchItem{"Chip 2", "A description for Chip 2"},
		gi.SwitchItem{"Chip 3", "A description for Chip 3"})

	gi.NewSwitches(tab).SetType(gi.SwitchCheckbox).SetItems(
		gi.SwitchItem{"Checkbox 1", "A description for Checkbox 1"},
		gi.SwitchItem{"Checkbox 2", "A description for Checkbox 2"},
		gi.SwitchItem{"Checkbox 3", "A description for Checkbox 3"})

	gi.NewSwitches(tab).SetType(gi.SwitchCheckbox).SetItems(
		gi.SwitchItem{"Indeterminate 1", "A description for Indeterminate Checkbox 1"},
		gi.SwitchItem{"Indeterminate 2", "A description for Indeterminate Checkbox 2"},
		gi.SwitchItem{"Indeterminate 3", "A description for Indeterminate Checkbox 3"}).
		OnWidgetAdded(func(w gi.Widget) {
			if sw, ok := w.(*gi.Switch); ok {
				sw.SetState(true, states.Indeterminate)
			}
		})

	gi.NewSwitches(tab).SetType(gi.SwitchRadioButton).SetMutex(true).SetItems(
		gi.SwitchItem{"Radio Button 1", "A description for Radio Button 1"},
		gi.SwitchItem{"Radio Button 2", "A description for Radio Button 2"},
		gi.SwitchItem{"Radio Button 3", "A description for Radio Button 3"})

	gi.NewSwitches(tab).SetType(gi.SwitchRadioButton).SetMutex(true).SetItems(
		gi.SwitchItem{"Indeterminate 1", "A description for Indeterminate Radio Button 1"},
		gi.SwitchItem{"Indeterminate 2", "A description for Indeterminate Radio Button 2"},
		gi.SwitchItem{"Indeterminate 3", "A description for Indeterminate Radio Button 3"}).
		OnWidgetAdded(func(w gi.Widget) {
			if sw, ok := w.(*gi.Switch); ok {
				sw.SetState(true, states.Indeterminate)
			}
		})

	gi.NewSwitches(tab).SetType(gi.SwitchSegmentedButton).SetMutex(true).SetItems(
		gi.SwitchItem{"Segmented Button 1", "A description for Segmented Button 1"},
		gi.SwitchItem{"Segmented Button 2", "A description for Segmented Button 2"},
		gi.SwitchItem{"Segmented Button 3", "A description for Segmented Button 3"})
}

func sliders(ts *gi.Tabs) {
	tab := ts.NewTab("Sliders")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Sliders and meters")
	gi.NewLabel(tab).SetText("Cogent Core provides interactive sliders and customizable meters, allowing you to edit and display bounded numbers.")

	gi.NewSlider(tab)
	gi.NewSlider(tab).SetValue(0.7).SetState(true, states.Disabled)

	csliders := gi.NewLayout(tab)

	gi.NewSlider(csliders).SetValue(0.3).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	gi.NewSlider(csliders).SetValue(0.2).SetState(true, states.Disabled).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	gi.NewMeter(tab).SetType(gi.MeterCircle).SetValue(0.7).SetText("70%")
	gi.NewMeter(tab).SetType(gi.MeterSemicircle).SetValue(0.7).SetText("70%")
	gi.NewMeter(tab).SetValue(0.7)
	gi.NewMeter(tab).SetValue(0.7).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
}

func icon(ts *gi.Tabs) {
	tab := ts.NewTab("Icons")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Icons")
	gi.NewLabel(tab).SetText("Cogent Core provides more than 2,000 unique icons from the Material Symbols collection, allowing you to easily represent many things in a concise, visually pleasing, and language-independent way.")

	gi.NewButton(tab).SetText("View icons").OnClick(func(e events.Event) {
		d := gi.NewBody().AddTitle("Cogent Core Icons")
		grid := gi.NewFrame(d)
		grid.Style(func(s *styles.Style) {
			s.Wrap = true
			s.Overflow.Y = styles.OverflowAuto
		})

		icnms := icons.All()
		for _, ic := range icnms {
			icnm := string(ic)
			if strings.HasSuffix(icnm, "-fill") {
				continue
			}
			vb := gi.NewLayout(grid, icnm).Style(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Max.X.Em(15) // constraining width exactly gives nice grid-like appearance
				s.Min.X.Em(15)
			})
			gi.NewIcon(vb, icnm).SetIcon(icons.Icon(icnm)).Style(func(s *styles.Style) {
				s.Min.Set(units.Em(4))
			})
			gi.NewLabel(vb, icnm).SetText(strcase.ToSentence(icnm)).Style(func(s *styles.Style) {
				s.SetTextWrap(false)
			})
		}
		d.NewFullDialog(tab).Run()
	})
}

func values(ts *gi.Tabs) {
	tab := ts.NewTab("Values")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Values")
	gi.NewLabel(tab).SetText("Cogent Core provides the giv value system, which allows you to instantly turn Go values into interactive widgets bound to the original values with just a single simple line of code. For example, you can dynamically edit this very GUI right now by clicking the first button below.")

	gi.NewButton(tab).SetText("Inspector").OnClick(func(e events.Event) {
		giv.InspectorWindow(ts.Scene)
	})

	giv.NewValue(tab, colors.Orange)
	giv.NewValue(tab, time.Now())
	giv.NewValue(tab, 5*time.Minute)
	giv.NewValue(tab, 500*time.Millisecond)
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
	gi.NewLabel(tab).SetText("Cogent Core provides powerful views that allow you to easily view and edit complex data types like structs, maps, and slices, allowing you to easily create widgets like lists, tables, and forms.")

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

	sp := gi.NewSplits(vts.NewTab("Tree view"))

	tv := giv.NewTreeViewFrame(sp).SetText("Root")
	makeTree(tv, 0, 3, 5)

	sv := giv.NewStructView(sp)
	sv.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	sv.SetStruct(tv)

	tv.OnSelect(func(e events.Event) {
		if len(tv.SelectedNodes) > 0 {
			sv.SetStruct(tv.SelectedNodes[0])
		}
	})

	sp.SetSplits(0.3, 0.7)
}

func makeTree(tv *giv.TreeView, iter, maxIter, maxKids int) {
	if iter > maxIter {
		return
	}
	n := rand.Intn(maxKids)
	if iter == 0 {
		n = maxKids
	}
	iter++
	tv.SetNChildren(n, giv.TreeViewType, "Child ")
	for j := 0; j < n; j++ {
		kt := tv.Child(j).(*giv.TreeView)
		makeTree(kt, iter, maxIter, maxKids)
	}
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

func dialogs(ts *gi.Tabs) {
	tab := ts.NewTab("Dialogs")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Dialogs, snackbars, and windows")
	gi.NewLabel(tab).SetText("Cogent Core provides completely customizable dialogs, snackbars, and windows that allow you to easily display, obtain, and organize information.")

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
				gi.MessageSnackbar(cd, "Dialog canceled")
			})
			d.AddOk(pw).OnClick(func(e events.Event) {
				gi.MessageSnackbar(cd, "Dialog accepted")
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
				gi.MessageSnackbar(td, "Your name is "+tf.Text())
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
				gi.MessageSnackbar(cs, "Refreshed files")
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

func style(ts *gi.Tabs) {
	tab := ts.NewTab("Styles")

	gi.NewLabel(tab).SetType(gi.LabelHeadlineLarge).SetText("Styles and layouts")
	gi.NewLabel(tab).SetText("Cogent Core provides a fully customizable styling and layout system that allows you to easily control the position, size, and appearance of all widgets. You can edit the style properties of the outer frame below.")

	sp := gi.NewSplits(tab)

	sv := giv.NewStructView(sp)

	fr := gi.NewFrame(gi.NewFrame(sp)) // can not control layout when directly in splits
	sv.SetStruct(&fr.Styles)

	fr.Style(func(s *styles.Style) {
		s.Background = colors.C(colors.Scheme.Select.Container)
	})

	fr.OnShow(func(e events.Event) {
		fr.OverrideStyle = true
	})

	sv.OnChange(func(e events.Event) {
		fr.Update()
	})

	frameSizes := []mat32.Vec2{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}

	for _, sz := range frameSizes {
		gi.NewFrame(fr).Style(func(s *styles.Style) {
			s.Min.Set(units.Px(sz.X), units.Px(sz.Y))
			s.Grow.Set(0, 0)
			s.Background = colors.C(colors.Scheme.Primary.Base)
		})
	}
}
