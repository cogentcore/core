// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	"embed"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/views"
)

//go:embed demo.go
var demoFile embed.FS

func main() {
	b := core.NewBody("Cogent Core Demo")
	ts := core.NewTabs(b)

	home(ts)
	widgets(ts)
	makeViews(ts)
	values(ts)
	makeStyles(ts)

	b.RunMainWindow()
}

func home(ts *core.Tabs) {
	tab := ts.NewTab("Home")
	tab.Style(func(s *styles.Style) {
		s.CenterAll()
	})

	errors.Log(core.NewSVG(tab).ReadString(core.AppIcon))

	core.NewText(tab).SetType(core.TextDisplayLarge).SetText("The Cogent Core Demo")
	core.NewText(tab).SetType(core.TextTitleLarge).SetText(core.AppAbout)
}

func widgets(ts *core.Tabs) {
	wts := core.NewTabs(ts.NewTab("Widgets"))

	text(wts)
	buttons(wts)
	inputs(wts)
	sliders(wts)
	dialogs(wts)
	makeIcons(wts)
}

func text(ts *core.Tabs) {
	tab := ts.NewTab("Text")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Text")
	core.NewText(tab).SetText("Cogent Core provides fully customizable text elements that can be styled in any way you want. Also, there are pre-configured style types for text that allow you to easily create common text types.")

	for _, typ := range core.TextTypesValues() {
		s := strcase.ToSentence(typ.String())
		core.NewText(tab).SetType(typ).SetText(s)
	}
}

func buttons(ts *core.Tabs) {
	tab := ts.NewTab("Buttons")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Buttons")

	core.NewText(tab).SetText("Cogent Core provides customizable buttons that support various events and can be styled in any way you want. Also, there are pre-configured style types for buttons that allow you to achieve common functionality with ease. All buttons support any combination of text, an icon, and an indicator.")

	makeRow := func() core.Widget {
		return core.NewFrame(tab).Style(func(s *styles.Style) {
			s.Wrap = true
			s.Align.Items = styles.Center
		})
	}

	core.NewText(tab).SetType(core.TextHeadlineSmall).SetText("Standard buttons")
	brow := makeRow()
	browt := makeRow()
	browi := makeRow()

	core.NewText(tab).SetType(core.TextHeadlineSmall).SetText("Menu buttons")
	mbrow := makeRow()
	mbrowt := makeRow()
	mbrowi := makeRow()

	menu := func(m *core.Scene) {
		m1 := core.NewButton(m).SetText("Menu Item 1").SetIcon(icons.Save).SetShortcut("Control+Shift+1").
			SetTooltip("A standard menu item with an icon")
		m1.OnClick(func(e events.Event) {
			fmt.Println("Clicked on menu item 1")
		})

		m2 := core.NewButton(m).SetText("Menu Item 2").SetIcon(icons.Open).
			SetTooltip("A menu item with an icon and a sub menu")

		m2.Menu = func(m *core.Scene) {
			sm2 := core.NewButton(m).SetText("Sub Menu Item 2").SetIcon(icons.InstallDesktop).
				SetTooltip("A sub menu item with an icon")
			sm2.OnClick(func(e events.Event) {
				fmt.Println("Clicked on sub menu item 2")
			})
		}

		core.NewSeparator(m)

		m3 := core.NewButton(m).SetText("Menu Item 3").SetIcon(icons.Favorite).SetShortcut("Control+3").
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

	for _, typ := range core.ButtonTypesValues() {
		// not really a real button, so not worth including in demo
		if typ == core.ButtonMenu {
			continue
		}

		s := strings.TrimPrefix(typ.String(), "Button")
		sl := strings.ToLower(s)
		art := "A "
		if typ == core.ButtonElevated || typ == core.ButtonOutlined || typ == core.ButtonAction {
			art = "An "
		}

		b := core.NewButton(brow).SetType(typ).SetText(s).SetIcon(ics[typ]).
			SetTooltip("A standard " + sl + " button with text and an icon")
		b.OnClick(func(e events.Event) {
			fmt.Println("Got click event on", b.Nm)
		})

		bt := core.NewButton(browt).SetType(typ).SetText(s).
			SetTooltip("A standard " + sl + " button with text")
		bt.OnClick(func(e events.Event) {
			fmt.Println("Got click event on", bt.Nm)
		})

		bi := core.NewButton(browi).SetType(typ).SetIcon(ics[typ+5]).
			SetTooltip("A standard " + sl + " button with an icon")
		bi.OnClick(func(e events.Event) {
			fmt.Println("Got click event on", bi.Nm)
		})

		core.NewButton(mbrow).SetType(typ).SetText(s).SetIcon(ics[typ+10]).SetMenu(menu).
			SetTooltip(art + sl + " menu button with text and an icon")

		core.NewButton(mbrowt).SetType(typ).SetText(s).SetMenu(menu).
			SetTooltip(art + sl + " menu button with text")

		core.NewButton(mbrowi).SetType(typ).SetIcon(ics[typ+15]).SetMenu(menu).
			SetTooltip(art + sl + " menu button with an icon")
	}
}

func inputs(ts *core.Tabs) {
	tab := ts.NewTab("Inputs")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Inputs")
	core.NewText(tab).SetText("Cogent Core provides various customizable input widgets that cover all common uses. Various events can be bound to inputs, and their data can easily be fetched and used wherever needed. There are also pre-configured style types for most inputs that allow you to easily switch among common styling patterns.")

	core.NewTextField(tab).SetPlaceholder("Text field")
	core.NewTextField(tab).SetPlaceholder("Email").SetType(core.TextFieldOutlined).Style(func(s *styles.Style) {
		s.VirtualKeyboard = styles.KeyboardEmail
	})
	core.NewTextField(tab).SetPlaceholder("Phone number").AddClearButton().Style(func(s *styles.Style) {
		s.VirtualKeyboard = styles.KeyboardPhone
	})
	core.NewTextField(tab).SetPlaceholder("URL").SetType(core.TextFieldOutlined).AddClearButton().Style(func(s *styles.Style) {
		s.VirtualKeyboard = styles.KeyboardURL
	})
	core.NewTextField(tab).AddClearButton().SetLeadingIcon(icons.Search)
	core.NewTextField(tab).SetType(core.TextFieldOutlined).SetTypePassword().SetPlaceholder("Password")
	core.NewTextField(tab).SetText("Multiline textfield with a relatively long initial text").
		Style(func(s *styles.Style) {
			s.SetTextWrap(true)
		})

	spinners := core.NewFrame(tab)

	core.NewSpinner(spinners).SetStep(5).SetMin(-50).SetMax(100).SetValue(15)
	core.NewSpinner(spinners).SetFormat("%X").SetStep(1).SetMax(255).SetValue(44)

	choosers := core.NewFrame(tab)

	fruits := []core.ChooserItem{
		{Value: "Apple", Tooltip: "A round, edible fruit that typically has red skin"},
		{Value: "Apricot", Tooltip: "A stonefruit with a yellow or orange color"},
		{Value: "Blueberry", Tooltip: "A small blue or purple berry"},
		{Value: "Blackberry", Tooltip: "A small, edible, dark fruit"},
		{Value: "Peach", Tooltip: "A fruit with yellow or white flesh and a large seed"},
		{Value: "Strawberry", Tooltip: "A widely consumed small, red fruit"},
	}

	core.NewChooser(choosers).SetPlaceholder("Select a fruit").SetItems(fruits...).SetAllowNew(true)
	core.NewChooser(choosers).SetPlaceholder("Select a fruit").SetItems(fruits...).SetType(core.ChooserOutlined)
	core.NewChooser(tab).SetEditable(true).SetPlaceholder("Select or type a fruit").SetItems(fruits...).SetAllowNew(true)
	core.NewChooser(tab).SetEditable(true).SetPlaceholder("Select or type a fruit").SetItems(fruits...).SetType(core.ChooserOutlined)

	core.NewSwitch(tab).SetText("Toggle")

	core.NewSwitches(tab).SetItems(
		core.SwitchItem{"Switch 1", "A description for Switch 1"},
		core.SwitchItem{"Switch 2", "A description for Switch 2"},
		core.SwitchItem{"Switch 3", "A description for Switch 3"})

	core.NewSwitches(tab).SetType(core.SwitchChip).SetItems(
		core.SwitchItem{"Chip 1", "A description for Chip 1"},
		core.SwitchItem{"Chip 2", "A description for Chip 2"},
		core.SwitchItem{"Chip 3", "A description for Chip 3"})

	core.NewSwitches(tab).SetType(core.SwitchCheckbox).SetItems(
		core.SwitchItem{"Checkbox 1", "A description for Checkbox 1"},
		core.SwitchItem{"Checkbox 2", "A description for Checkbox 2"},
		core.SwitchItem{"Checkbox 3", "A description for Checkbox 3"})

	core.NewSwitches(tab).SetType(core.SwitchCheckbox).SetItems(
		core.SwitchItem{"Indeterminate 1", "A description for Indeterminate Checkbox 1"},
		core.SwitchItem{"Indeterminate 2", "A description for Indeterminate Checkbox 2"},
		core.SwitchItem{"Indeterminate 3", "A description for Indeterminate Checkbox 3"}).
		OnWidgetAdded(func(w core.Widget) {
			if sw, ok := w.(*core.Switch); ok {
				sw.SetState(true, states.Indeterminate)
			}
		})

	core.NewSwitches(tab).SetType(core.SwitchRadioButton).SetMutex(true).SetItems(
		core.SwitchItem{"Radio Button 1", "A description for Radio Button 1"},
		core.SwitchItem{"Radio Button 2", "A description for Radio Button 2"},
		core.SwitchItem{"Radio Button 3", "A description for Radio Button 3"})

	core.NewSwitches(tab).SetType(core.SwitchRadioButton).SetMutex(true).SetItems(
		core.SwitchItem{"Indeterminate 1", "A description for Indeterminate Radio Button 1"},
		core.SwitchItem{"Indeterminate 2", "A description for Indeterminate Radio Button 2"},
		core.SwitchItem{"Indeterminate 3", "A description for Indeterminate Radio Button 3"}).
		OnWidgetAdded(func(w core.Widget) {
			if sw, ok := w.(*core.Switch); ok {
				sw.SetState(true, states.Indeterminate)
			}
		})

	core.NewSwitches(tab).SetType(core.SwitchSegmentedButton).SetMutex(true).SetItems(
		core.SwitchItem{"Segmented Button 1", "A description for Segmented Button 1"},
		core.SwitchItem{"Segmented Button 2", "A description for Segmented Button 2"},
		core.SwitchItem{"Segmented Button 3", "A description for Segmented Button 3"})
}

func sliders(ts *core.Tabs) {
	tab := ts.NewTab("Sliders")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Sliders and meters")
	core.NewText(tab).SetText("Cogent Core provides interactive sliders and customizable meters, allowing you to edit and display bounded numbers.")

	core.NewSlider(tab)
	core.NewSlider(tab).SetValue(0.7).SetState(true, states.Disabled)

	csliders := core.NewFrame(tab)

	core.NewSlider(csliders).SetValue(0.3).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	core.NewSlider(csliders).SetValue(0.2).SetState(true, states.Disabled).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	core.NewMeter(tab).SetType(core.MeterCircle).SetValue(0.7).SetText("70%")
	core.NewMeter(tab).SetType(core.MeterSemicircle).SetValue(0.7).SetText("70%")
	core.NewMeter(tab).SetValue(0.7)
	core.NewMeter(tab).SetValue(0.7).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
}

func textEditors(ts *core.Tabs) {
	tab := ts.NewTab("Text editors")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Text editors")
	core.NewText(tab).SetText("Cogent Core provides powerful text editors that support advanced code editing features, like syntax highlighting, completion, undo and redo, copy and paste, rectangular selection, and word, line, and page based navigation, selection, and deletion.")

	sp := core.NewSplits(tab)

	errors.Log(texteditor.NewSoloEditor(sp).Buffer.OpenFS(demoFile, "demo.go"))
	texteditor.NewSoloEditor(sp).Buffer.SetLang("svg").SetTextString(core.AppIcon)
}

func makeIcons(ts *core.Tabs) {
	tab := ts.NewTab("Icons")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Icons")
	core.NewText(tab).SetText("Cogent Core provides more than 2,000 unique icons from the Material Symbols collection, allowing you to easily represent many things in a concise, visually pleasing, and language-independent way.")

	core.NewButton(tab).SetText("View icons").OnClick(func(e events.Event) {
		d := core.NewBody().AddTitle("Cogent Core Icons")
		grid := core.NewFrame(d)
		grid.Style(func(s *styles.Style) {
			s.Wrap = true
			s.Overflow.Y = styles.OverflowAuto
		})

		ics := icons.All()
		for _, ic := range ics {
			sic := string(ic)
			if strings.HasSuffix(sic, "-fill") {
				continue
			}
			vb := core.NewFrame(grid).Style(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Max.X.Em(15) // constraining width exactly gives nice grid-like appearance
				s.Min.X.Em(15)
			})
			core.NewIcon(vb).SetIcon(ic).Style(func(s *styles.Style) {
				s.Min.Set(units.Em(4))
			})
			core.NewText(vb).SetText(strcase.ToSentence(sic)).Style(func(s *styles.Style) {
				s.SetTextWrap(false)
			})
		}
		d.RunFullDialog(tab)
	})
}

func values(ts *core.Tabs) {
	tab := ts.NewTab("Values")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Values")
	core.NewText(tab).SetText("Cogent Core provides the value views system, which allows you to instantly turn Go values into interactive widgets bound to the original values with just a single simple line of code. For example, you can dynamically edit this very GUI right now by clicking the first button below.")

	name := "Gopher"
	core.Bind(&name, core.NewTextField(tab)).OnChange(func(e events.Event) {
		fmt.Println("Your name is now", name)
	})

	age := 35
	core.Bind(&age, core.NewSpinner(tab)).OnChange(func(e events.Event) {
		fmt.Println("Your age is now", age)
	})

	on := true
	core.Bind(&on, core.NewSwitch(tab)).OnChange(func(e events.Event) {
		fmt.Println("The switch is now", on)
	})

	align := core.ThemeLight
	core.Bind(&align, core.NewSwitches(tab)).OnChange(func(e events.Event) {
		fmt.Println("The alignment is now", align)
	})

	var state states.States
	state.SetFlag(true, states.Hovered)
	state.SetFlag(true, states.Dragging)
	core.Bind(&state, core.NewSwitches(tab)).OnChange(func(e events.Event) {
		fmt.Println("The state is now", state)
	})

	core.NewButton(tab).SetText("Inspector").OnClick(func(e events.Event) {
		views.InspectorWindow(ts.Scene)
	})

	views.NewValue(tab, colors.Orange)
	views.NewValue(tab, time.Now())
	views.NewValue(tab, 5*time.Minute)
	views.NewValue(tab, 500*time.Millisecond)
	views.NewValue(tab, core.Filename("demo.go"))
	views.NewValue(tab, core.AppearanceSettings.Font)
	views.NewValue(tab, views.ColorMapName("ColdHot"))
	views.NewFuncButton(tab, hello).SetShowReturn(true)
}

// Hello displays a greeting message and an age in weeks based on the given information.
func hello(firstName string, lastName string, age int, likesGo bool) (greeting string, weeksOld int) { //types:add
	weeksOld = age * 52
	greeting = "Hello, " + firstName + " " + lastName + "! "
	if likesGo {
		greeting += "I'm glad to hear that you like the best programming language!"
	} else {
		greeting += "You should reconsider what programming languages you like."
	}
	return
}

func makeViews(ts *core.Tabs) {
	tab := ts.NewTab("Views")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Views")
	core.NewText(tab).SetText("Cogent Core provides powerful views that allow you to easily view and edit complex data types like structs, maps, and slices, allowing you to easily create widgets like lists, tables, and forms.")

	vts := core.NewTabs(tab)

	str := testStruct{
		Name:   "Go",
		Cond:   2,
		Value:  3.1415,
		Vec:    math32.Vec2(5, 7),
		Inline: inlineStruct{Value: 3},
		Cond2: tableStruct{
			IntField:   22,
			FloatField: 44.4,
			StrField:   "fi",
			File:       "views.go",
		},
		Things: make([]tableStruct, 2),
		Stuff:  []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7},
	}

	views.NewStructView(vts.NewTab("Struct view")).SetStruct(&str)

	mp := map[string]any{}

	mp["Go"] = "Elegant, fast, and easy-to-use"
	mp["Python"] = true
	mp["C++"] = 4

	views.NewMapView(vts.NewTab("Map view")).SetMap(&mp)

	sl := make([]string, 15)
	for i := 0; i < len(sl); i++ {
		sl[i] = fmt.Sprintf("el: %v", i)
	}
	sl[10] = "this is a particularly long slice value"
	views.NewSliceView(vts.NewTab("Slice view")).SetSlice(&sl)

	tbl := make([]*tableStruct, 10)
	for i := range tbl {
		ts := &tableStruct{IntField: i, FloatField: float32(i) / 10}
		tbl[i] = ts
	}
	tbl[0].StrField = "this is a particularly long field"
	views.NewTableView(vts.NewTab("Table view")).SetSlice(&tbl)

	sp := core.NewSplits(vts.NewTab("Tree view")).SetSplits(0.3, 0.7)
	tv := views.NewTreeViewFrame(sp).SetText("Root")
	// fmt.Println("calling make tree")
	makeTree(tv, 0, 3, 5)
	tv.RootSetViewIndex()

	sv := views.NewStructView(sp)
	sv.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	sv.SetStruct(tv)

	tv.OnSelect(func(e events.Event) {
		if len(tv.SelectedNodes) > 0 {
			sv.SetStruct(tv.SelectedNodes[0])
			sv.Update()
		}
	})
	// fmt.Println(&tv, tv.Listeners)

	textEditors(vts)
}

func makeTree(tv *views.TreeView, iter, maxIter, maxKids int) {
	if iter > maxIter {
		return
	}
	n := rand.Intn(maxKids)
	if iter == 0 {
		n = maxKids
	}
	iter++
	tv.SetNChildren(n, views.TreeViewType, "Child ")
	for j := 0; j < n; j++ {
		kt := tv.Child(j).(*views.TreeView)
		makeTree(kt, iter, maxIter, maxKids)
	}
}

type tableStruct struct { //types:add

	// an icon
	Icon icons.Icon

	// an integer field
	IntField int `default:"2"`

	// a float field
	FloatField float32

	// a string field
	StrField string

	// a file
	File core.Filename
}

type inlineStruct struct { //types:add

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
	Value float32
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

type testStruct struct { //types:add

	// An enum value
	Enum core.ButtonTypes

	// a string
	Name string `default:"Go"`

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
	Value float32

	Vec math32.Vector2

	Things []tableStruct

	Stuff []float32

	// a file
	File core.Filename
}

func (ts *testStruct) ShouldShow(field string) bool {
	switch field {
	case "Name":
		return ts.Enum <= core.ButtonElevated
	case "ShowMe":
		return ts.ShowNext
	case "Cond1":
		return ts.Cond == 0
	case "Cond2":
		return ts.Cond >= 0
	}
	return true
}

func dialogs(ts *core.Tabs) {
	tab := ts.NewTab("Dialogs")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Dialogs, snackbars, and windows")
	core.NewText(tab).SetText("Cogent Core provides completely customizable dialogs, snackbars, and windows that allow you to easily display, obtain, and organize information.")

	makeRow := func() core.Widget {
		return core.NewFrame(tab).Style(func(s *styles.Style) {
			s.Wrap = true
			s.Align.Items = styles.Center
		})
	}

	core.NewText(tab).SetType(core.TextHeadlineSmall).SetText("Dialogs")
	drow := makeRow()

	md := core.NewButton(drow).SetText("Message")
	md.OnClick(func(e events.Event) {
		core.MessageDialog(md, "Something happened", "Message")
	})

	ed := core.NewButton(drow).SetText("Error")
	ed.OnClick(func(e events.Event) {
		core.ErrorDialog(ed, errors.New("invalid encoding format"), "Error loading file")
	})

	cd := core.NewButton(drow).SetText("Confirm")
	cd.OnClick(func(e events.Event) {
		d := core.NewBody().AddTitle("Confirm").AddText("Send message?")
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent).OnClick(func(e events.Event) {
				core.MessageSnackbar(cd, "Dialog canceled")
			})
			d.AddOK(parent).OnClick(func(e events.Event) {
				core.MessageSnackbar(cd, "Dialog accepted")
			})
		})
		d.RunDialog(cd)
	})

	td := core.NewButton(drow).SetText("Input")
	td.OnClick(func(e events.Event) {
		d := core.NewBody().AddTitle("Input").AddText("What is your name?")
		tf := core.NewTextField(d)
		d.AddBottomBar(func(parent core.Widget) {
			d.AddCancel(parent)
			d.AddOK(parent).OnClick(func(e events.Event) {
				core.MessageSnackbar(td, "Your name is "+tf.Text())
			})
		})
		d.RunDialog(td)
	})

	fd := core.NewButton(drow).SetText("Full window")
	u := &core.User{}
	fd.OnClick(func(e events.Event) {
		d := core.NewBody().AddTitle("Full window dialog").AddText("Edit your information")
		views.NewStructView(d).SetStruct(u).OnInput(func(e events.Event) {
			fmt.Println("Got input event")
		})
		d.OnClose(func(e events.Event) {
			fmt.Println("Your information is:", u)
		})
		d.RunFullDialog(td)
	})

	nd := core.NewButton(drow).SetText("New window")
	nd.OnClick(func(e events.Event) {
		core.NewBody().AddTitle("New window dialog").AddText("This dialog opens in a new window on multi-window platforms").RunWindowDialog(nd)
	})

	core.NewText(tab).SetType(core.TextHeadlineSmall).SetText("Snackbars")
	srow := makeRow()

	ms := core.NewButton(srow).SetText("Message")
	ms.OnClick(func(e events.Event) {
		core.MessageSnackbar(ms, "New messages loaded")
	})

	es := core.NewButton(srow).SetText("Error")
	es.OnClick(func(e events.Event) {
		core.ErrorSnackbar(es, errors.New("file not found"), "Error loading page")
	})

	cs := core.NewButton(srow).SetText("Custom")
	cs.OnClick(func(e events.Event) {
		core.NewBody().AddSnackbarText("Files updated").
			AddSnackbarButton("Refresh", func(e events.Event) {
				core.MessageSnackbar(cs, "Refreshed files")
			}).AddSnackbarIcon(icons.Close).NewSnackbar(cs).Run()
	})

	core.NewText(tab).SetType(core.TextHeadlineSmall).SetText("Windows")
	wrow := makeRow()

	nw := core.NewButton(wrow).SetText("New window")
	nw.OnClick(func(e events.Event) {
		core.NewBody().AddTitle("New window").AddText("A standalone window that opens in a new window on multi-window platforms").RunWindow()
	})

	fw := core.NewButton(wrow).SetText("Full window")
	fw.OnClick(func(e events.Event) {
		core.NewBody().AddTitle("Full window").AddText("A standalone window that opens in the same system window").NewWindow().SetNewWindow(false).Run()
	})
}

func makeStyles(ts *core.Tabs) {
	tab := ts.NewTab("Styles")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Styles and layouts")
	core.NewText(tab).SetText("Cogent Core provides a fully customizable styling and layout system that allows you to easily control the position, size, and appearance of all widgets. You can edit the style properties of the outer frame below.")

	sp := core.NewSplits(tab)

	sv := views.NewStructView(sp)

	fr := core.NewFrame(core.NewFrame(sp)) // can not control layout when directly in splits
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

	frameSizes := []math32.Vector2{
		{20, 100},
		{80, 20},
		{60, 80},
		{40, 120},
		{150, 100},
	}

	for _, sz := range frameSizes {
		core.NewFrame(fr).Style(func(s *styles.Style) {
			s.Min.Set(units.Px(sz.X), units.Px(sz.Y))
			s.Grow.Set(0, 0)
			s.Background = colors.C(colors.Scheme.Primary.Base)
		})
	}
}
