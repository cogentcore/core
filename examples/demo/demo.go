// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate

import (
	"embed"
	"fmt"
	"image"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
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
	"cogentcore.org/core/tree"
)

//go:embed demo.go
var demoFile embed.FS

func main() {
	b := core.NewBody("Cogent Core Demo")
	ts := core.NewTabs(b)

	home(ts)
	widgets(ts)
	collections(ts)
	valueBinding(ts)
	makeStyles(ts)

	b.RunMainWindow()
}

func home(ts *core.Tabs) {
	tab, _ := ts.NewTab("Home")
	tab.Styler(func(s *styles.Style) {
		s.CenterAll()
	})

	errors.Log(core.NewSVG(tab).ReadString(core.AppIcon))

	core.NewText(tab).SetType(core.TextDisplayLarge).SetText("The Cogent Core Demo")
	core.NewText(tab).SetType(core.TextTitleLarge).SetText(`A <b>demonstration</b> of the <i>various</i> features of the <a href="https://cogentcore.org/core">Cogent Core</a> 2D and 3D Go GUI <u>framework</u>`)
}

func widgets(ts *core.Tabs) {
	wts := core.NewTabs(ts.NewTab("Widgets"))

	text(wts)
	buttons(wts)
	inputs(wts)
	sliders(wts)
	dialogs(wts)
	textEditors(wts)
}

func text(ts *core.Tabs) {
	tab, _ := ts.NewTab("Text")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Text")
	core.NewText(tab).SetText("Cogent Core provides fully customizable text elements that can be styled in any way you want. Also, there are pre-configured style types for text that allow you to easily create common text types.")

	for _, typ := range core.TextTypesValues() {
		s := strcase.ToSentence(typ.String())
		core.NewText(tab).SetType(typ).SetText(s)
	}
}

func makeRow(parent core.Widget) *core.Frame {
	row := core.NewFrame(parent)
	row.Styler(func(s *styles.Style) {
		s.Wrap = true
		s.Align.Items = styles.Center
	})
	return row
}

func buttons(ts *core.Tabs) {
	tab, _ := ts.NewTab("Buttons")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Buttons")

	core.NewText(tab).SetText("Cogent Core provides customizable buttons that support various events and can be styled in any way you want. Also, there are pre-configured style types for buttons that allow you to achieve common functionality with ease. All buttons support any combination of text, an icon, and an indicator.")

	rowm := makeRow(tab)
	rowti := makeRow(tab)
	rowt := makeRow(tab)
	rowi := makeRow(tab)

	menu := func(m *core.Scene) {
		m1 := core.NewButton(m).SetText("Menu Item 1").SetIcon(icons.Save).SetShortcut("Control+Shift+1")
		m1.SetTooltip("A standard menu item with an icon")
		m1.OnClick(func(e events.Event) {
			fmt.Println("Clicked on menu item 1")
		})

		m2 := core.NewButton(m).SetText("Menu Item 2").SetIcon(icons.Open)
		m2.SetTooltip("A menu item with an icon and a sub menu")

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

		core.NewButton(rowm).SetType(typ).SetText(s).SetIcon(ics[typ]).SetMenu(menu).
			SetTooltip(art + sl + " menu button with text and an icon")

		b := core.NewButton(rowti).SetType(typ).SetText(s).SetIcon(ics[typ+6]).
			SetTooltip("A " + sl + " button with text and an icon")
		b.OnClick(func(e events.Event) {
			fmt.Println("Got click event on", b.Name)
		})

		bt := core.NewButton(rowt).SetType(typ).SetText(s).
			SetTooltip("A " + sl + " button with text")
		bt.OnClick(func(e events.Event) {
			fmt.Println("Got click event on", bt.Name)
		})

		bi := core.NewButton(rowi).SetType(typ).SetIcon(ics[typ+12]).
			SetTooltip("A " + sl + " button with an icon")
		bi.OnClick(func(e events.Event) {
			fmt.Println("Got click event on", bi.Name)
		})
	}
}

func inputs(ts *core.Tabs) {
	tab, _ := ts.NewTab("Inputs")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Inputs")
	core.NewText(tab).SetText("Cogent Core provides various customizable input widgets that cover all common uses. Various events can be bound to inputs, and their data can easily be fetched and used wherever needed. There are also pre-configured style types for most inputs that allow you to easily switch among common styling patterns.")

	core.NewTextField(tab).SetPlaceholder("Text field")
	core.NewTextField(tab).AddClearButton().SetLeadingIcon(icons.Search)
	core.NewTextField(tab).SetType(core.TextFieldOutlined).SetTypePassword().SetPlaceholder("Password")
	core.NewTextField(tab).SetText("Text field with relatively long initial text")

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
		core.SwitchItem{Value: "Switch 1", Tooltip: "The first switch"},
		core.SwitchItem{Value: "Switch 2", Tooltip: "The second switch"},
		core.SwitchItem{Value: "Switch 3", Tooltip: "The third switch"})

	core.NewSwitches(tab).SetType(core.SwitchChip).SetStrings("Chip 1", "Chip 2", "Chip 3")
	core.NewSwitches(tab).SetType(core.SwitchCheckbox).SetStrings("Checkbox 1", "Checkbox 2", "Checkbox 3")
	cs := core.NewSwitches(tab).SetType(core.SwitchCheckbox).SetStrings("Indeterminate 1", "Indeterminate 2", "Indeterminate 3")
	cs.SetOnChildAdded(func(n tree.Node) {
		core.AsWidget(n).SetState(true, states.Indeterminate)
	})

	core.NewSwitches(tab).SetType(core.SwitchRadioButton).SetMutex(true).SetStrings("Radio Button 1", "Radio Button 2", "Radio Button 3")
	rs := core.NewSwitches(tab).SetType(core.SwitchRadioButton).SetMutex(true).SetStrings("Indeterminate 1", "Indeterminate 2", "Indeterminate 3")
	rs.SetOnChildAdded(func(n tree.Node) {
		core.AsWidget(n).SetState(true, states.Indeterminate)
	})

	core.NewSwitches(tab).SetType(core.SwitchSegmentedButton).SetMutex(true).SetStrings("Segmented Button 1", "Segmented Button 2", "Segmented Button 3")
}

func sliders(ts *core.Tabs) {
	tab, _ := ts.NewTab("Sliders")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Sliders and meters")
	core.NewText(tab).SetText("Cogent Core provides interactive sliders and customizable meters, allowing you to edit and display bounded numbers.")

	core.NewSlider(tab)
	core.NewSlider(tab).SetValue(0.7).SetState(true, states.Disabled)

	csliders := core.NewFrame(tab)

	core.NewSlider(csliders).SetValue(0.3).Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	core.NewSlider(csliders).SetValue(0.2).SetState(true, states.Disabled).Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})

	core.NewMeter(tab).SetType(core.MeterCircle).SetValue(0.7).SetText("70%")
	core.NewMeter(tab).SetType(core.MeterSemicircle).SetValue(0.7).SetText("70%")
	core.NewMeter(tab).SetValue(0.7)
	core.NewMeter(tab).SetValue(0.7).Styler(func(s *styles.Style) {
		s.Direction = styles.Column
	})
}

func textEditors(ts *core.Tabs) {
	tab, _ := ts.NewTab("Text editors")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Text editors")
	core.NewText(tab).SetText("Cogent Core provides powerful text editors that support advanced code editing features, like syntax highlighting, completion, undo and redo, copy and paste, rectangular selection, and word, line, and page based navigation, selection, and deletion.")

	sp := core.NewSplits(tab)

	errors.Log(texteditor.NewEditor(sp).Buffer.OpenFS(demoFile, "demo.go"))
	texteditor.NewEditor(sp).Buffer.SetLanguage(fileinfo.Svg).SetString(core.AppIcon)
}

func valueBinding(ts *core.Tabs) {
	tab, _ := ts.NewTab("Value binding")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Value binding")
	core.NewText(tab).SetText("Cogent Core provides the value binding system, which allows you to instantly bind Go values to interactive widgets with just a single simple line of code.")

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

	theme := core.ThemeLight
	core.Bind(&theme, core.NewSwitches(tab)).OnChange(func(e events.Event) {
		fmt.Println("The theme is now", theme)
	})

	var state states.States
	state.SetFlag(true, states.Hovered)
	state.SetFlag(true, states.Dragging)
	core.Bind(&state, core.NewSwitches(tab)).OnChange(func(e events.Event) {
		fmt.Println("The state is now", state)
	})

	color := colors.Orange
	core.Bind(&color, core.NewColorButton(tab)).OnChange(func(e events.Event) {
		fmt.Println("The color is now", color)
	})

	colorMap := core.ColorMapName("ColdHot")
	core.Bind(&colorMap, core.NewColorMapButton(tab)).OnChange(func(e events.Event) {
		fmt.Println("The color map is now", colorMap)
	})

	t := time.Now()
	core.Bind(&t, core.NewTimeInput(tab)).OnChange(func(e events.Event) {
		fmt.Println("The time is now", t)
	})

	duration := 5 * time.Minute
	core.Bind(&duration, core.NewDurationInput(tab)).OnChange(func(e events.Event) {
		fmt.Println("The duration is now", duration)
	})

	file := core.Filename("demo.go")
	core.Bind(&file, core.NewFileButton(tab)).OnChange(func(e events.Event) {
		fmt.Println("The file is now", file)
	})

	font := core.AppearanceSettings.Font
	core.Bind(&font, core.NewFontButton(tab)).OnChange(func(e events.Event) {
		fmt.Println("The font is now", font)
	})

	core.Bind(hello, core.NewFuncButton(tab)).SetShowReturn(true)
	core.Bind(styles.NewStyle, core.NewFuncButton(tab)).SetConfirm(true).SetShowReturn(true)

	core.NewButton(tab).SetText("Inspector").OnClick(func(e events.Event) {
		core.InspectorWindow(ts.Scene)
	})
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

func collections(ts *core.Tabs) {
	tab, _ := ts.NewTab("Collections")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Collections")
	core.NewText(tab).SetText("Cogent Core provides powerful collection widgets that allow you to easily view and edit complex data types like structs, maps, and slices, allowing you to easily create widgets like lists, tables, and forms.")

	vts := core.NewTabs(tab)

	str := testStruct{
		Name:      "Go",
		Condition: 2,
		Value:     3.1415,
		Vector:    math32.Vec2(5, 7),
		Inline:    inlineStruct{Value: 3},
		Condition2: tableStruct{
			Age:   22,
			Score: 44.4,
			Name:  "foo",
			File:  "core.go",
		},
		Table: make([]tableStruct, 2),
		List:  []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7},
	}

	ftab, _ := vts.NewTab("Forms")
	core.NewForm(ftab).SetStruct(&str)

	sl := make([]string, 50)
	for i := 0; i < len(sl); i++ {
		sl[i] = fmt.Sprintf("element %d", i)
	}
	sl[10] = "this is a particularly long value"
	ltab, _ := vts.NewTab("Lists")
	core.NewList(ltab).SetSlice(&sl)

	mp := map[string]string{}

	mp["Go"] = "Elegant, fast, and easy-to-use"
	mp["Python"] = "Slow and duck-typed"
	mp["C++"] = "Hard to use and slow to compile"

	ktab, _ := vts.NewTab("Keyed lists")
	core.NewKeyedList(ktab).SetMap(&mp)

	tbl := make([]*tableStruct, 50)
	for i := range tbl {
		ts := &tableStruct{Age: i, Score: float32(i) / 10}
		tbl[i] = ts
	}
	tbl[0].Name = "this is a particularly long field"
	ttab, _ := vts.NewTab("Tables")
	core.NewTable(ttab).SetSlice(&tbl)

	sp := core.NewSplits(vts.NewTab("Trees")).SetSplits(0.3, 0.7)
	tr := core.NewTree(core.NewFrame(sp)).SetText("Root")
	makeTree(tr, 0)

	sv := core.NewForm(sp).SetStruct(tr)

	tr.OnSelect(func(e events.Event) {
		if len(tr.SelectedNodes) > 0 {
			sv.SetStruct(tr.SelectedNodes[0]).Update()
		}
	})
}

func makeTree(tr *core.Tree, round int) {
	if round > 2 {
		return
	}
	for i := range 3 {
		n := core.NewTree(tr).SetText("Child " + strconv.Itoa(i))
		makeTree(n, round+1)
	}
}

type tableStruct struct { //types:add

	// an icon
	Icon icons.Icon

	// an integer field
	Age int `default:"2"`

	// a float field
	Score float32

	// a string field
	Name string

	// a file
	File core.Filename
}

type inlineStruct struct { //types:add

	// click to show next
	On bool

	// this is now showing
	ShowMe string

	// a condition
	Condition int

	// if On && Condition == 0
	Condition1 string

	// if On && Condition <= 1
	Condition2 tableStruct

	// a value
	Value float32
}

func (il *inlineStruct) ShouldDisplay(field string) bool {
	switch field {
	case "ShowMe", "Condition":
		return il.On
	case "Condition1":
		return il.On && il.Condition == 0
	case "Condition2":
		return il.On && il.Condition <= 1
	}
	return true
}

type testStruct struct { //types:add

	// An enum value
	Enum core.ButtonTypes

	// a string
	Name string `default:"Go" width:"50"`

	// click to show next
	ShowNext bool

	// this is now showing
	ShowMe string

	// inline struct
	Inline inlineStruct `display:"inline"`

	// a condition
	Condition int

	// if Condition == 0
	Condition1 string

	// if Condition >= 0
	Condition2 tableStruct

	// a value
	Value float32

	// a vector
	Vector math32.Vector2

	// a slice of structs
	Table []tableStruct

	// a slice of floats
	List []float32

	// a file
	File core.Filename
}

func (ts *testStruct) ShouldDisplay(field string) bool {
	switch field {
	case "Name":
		return ts.Enum <= core.ButtonElevated
	case "ShowMe":
		return ts.ShowNext
	case "Condition1":
		return ts.Condition == 0
	case "Condition2":
		return ts.Condition >= 0
	}
	return true
}

func dialogs(ts *core.Tabs) {
	tab, _ := ts.NewTab("Dialogs")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Dialogs, snackbars, and windows")
	core.NewText(tab).SetText("Cogent Core provides completely customizable dialogs, snackbars, and windows that allow you to easily display, obtain, and organize information.")

	core.NewText(tab).SetType(core.TextHeadlineSmall).SetText("Dialogs")
	drow := makeRow(tab)

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
		d := core.NewBody("Confirm")
		core.NewText(d).SetType(core.TextSupporting).SetText("Send message?")
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar).OnClick(func(e events.Event) {
				core.MessageSnackbar(cd, "Dialog canceled")
			})
			d.AddOK(bar).OnClick(func(e events.Event) {
				core.MessageSnackbar(cd, "Dialog accepted")
			})
		})
		d.RunDialog(cd)
	})

	td := core.NewButton(drow).SetText("Input")
	td.OnClick(func(e events.Event) {
		d := core.NewBody("Input")
		core.NewText(d).SetType(core.TextSupporting).SetText("What is your name?")
		tf := core.NewTextField(d)
		d.AddBottomBar(func(bar *core.Frame) {
			d.AddCancel(bar)
			d.AddOK(bar).OnClick(func(e events.Event) {
				core.MessageSnackbar(td, "Your name is "+tf.Text())
			})
		})
		d.RunDialog(td)
	})

	fd := core.NewButton(drow).SetText("Full window")
	u := &core.User{}
	fd.OnClick(func(e events.Event) {
		d := core.NewBody("Full window dialog")
		core.NewText(d).SetType(core.TextSupporting).SetText("Edit your information")
		core.NewForm(d).SetStruct(u).OnInput(func(e events.Event) {
			fmt.Println("Got input event")
		})
		d.OnClose(func(e events.Event) {
			fmt.Println("Your information is:", u)
		})
		d.RunFullDialog(td)
	})

	nd := core.NewButton(drow).SetText("New window")
	nd.OnClick(func(e events.Event) {
		d := core.NewBody("New window dialog")
		core.NewText(d).SetType(core.TextSupporting).SetText("This dialog opens in a new window on multi-window platforms")
		d.RunWindowDialog(nd)
	})

	core.NewText(tab).SetType(core.TextHeadlineSmall).SetText("Snackbars")
	srow := makeRow(tab)

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
			}).AddSnackbarIcon(icons.Close).RunSnackbar(cs)
	})

	core.NewText(tab).SetType(core.TextHeadlineSmall).SetText("Windows")
	wrow := makeRow(tab)

	nw := core.NewButton(wrow).SetText("New window")
	nw.OnClick(func(e events.Event) {
		d := core.NewBody("New window")
		core.NewText(d).SetType(core.TextHeadlineSmall).SetText("New window")
		core.NewText(d).SetType(core.TextSupporting).SetText("A standalone window that opens in a new window on multi-window platforms")
		d.RunWindow()
	})

	fw := core.NewButton(wrow).SetText("Full window")
	fw.OnClick(func(e events.Event) {
		d := core.NewBody("Full window")
		core.NewText(d).SetType(core.TextSupporting).SetText("A standalone window that opens in the same system window")
		d.NewWindow().SetNewWindow(false).SetDisplayTitle(true).Run()
	})

	rw := core.NewButton(wrow).SetText("Resize to content")
	rw.SetTooltip("Resizes this window to fit the current content")
	rw.OnClick(func(e events.Event) {
		wrow.Scene.ResizeToContent(image.Pt(0, 40)) // note: size is not correct due to wrapping? #1307
	})
}

func makeStyles(ts *core.Tabs) {
	tab, _ := ts.NewTab("Styles")

	core.NewText(tab).SetType(core.TextHeadlineLarge).SetText("Styles and layouts")
	core.NewText(tab).SetText("Cogent Core provides a fully customizable styling and layout system that allows you to easily control the position, size, and appearance of all widgets. You can edit the style properties of the outer frame below.")

	// same as docs advanced styling demo
	sp := core.NewSplits(tab)
	fm := core.NewForm(sp)
	fr := core.NewFrame(core.NewFrame(sp)) // can not control layout when directly in splits
	fr.Styler(func(s *styles.Style) {
		s.Background = colors.Scheme.Select.Container
		s.Grow.Set(1, 1)
	})
	fr.Style() // must style immediately to get correct default values
	fm.SetStruct(&fr.Styles)
	fm.OnChange(func(e events.Event) {
		fr.OverrideStyle = true
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
		core.NewFrame(fr).Styler(func(s *styles.Style) {
			s.Min.Set(units.Px(sz.X), units.Px(sz.Y))
			s.Background = colors.Scheme.Primary.Base
		})
	}
}
