// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/token"
	"reflect"

	"github.com/goki/gi"
	"github.com/goki/gi/complete"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768

	// turn these on to see a traces of various stages of processing..
	// gi.Update2DTrace = true
	// gi.Render2DTrace = true
	// gi.Layout2DTrace = true
	// ki.SignalTrace = true

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	oswin.TheApp.SetName("widgets")
	oswin.TheApp.SetAbout(`This is a demo of the main widgets and general functionality of the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>.
<p>The <a href="https://github.com/goki/gi/blob/master/examples/widgets/README.md">README</a> page for this example app has lots of further info.</p>`)

	win := gi.NewWindow2D("gogi-widgets-demo", "GoGi Widgets Demo", width, height, true) // true = pixel sizes

	icnm := "widget-wedge-down"

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	// style sheet
	var css = ki.Props{
		"button": ki.Props{
			"background-color": gi.Prefs.Colors.Control, // gi.Color{255, 240, 240, 255},
		},
		"#combo": ki.Props{
			"background-color": gi.Color{240, 255, 240, 255},
		},
		".hslides": ki.Props{
			"background-color": gi.Color{240, 225, 255, 255},
		},
		"kbd": ki.Props{
			"color": "blue",
		},
	}
	vp.CSS = css

	mfr := win.SetMainFrame()
	mfr.SetProp("spacing", units.NewValue(1, units.Ex))
	// mfr.SetProp("background-color", "linear-gradient(to top, red, lighter-80)")
	// mfr.SetProp("background-color", "linear-gradient(to right, red, orange, yellow, green, blue, indigo, violet)")
	// mfr.SetProp("background-color", "linear-gradient(to right, rgba(255,0,0,0), rgba(255,0,0,1))")
	// mfr.SetProp("background-color", "radial-gradient(red, lighter-80)")

	trow := mfr.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutHoriz
	trow.SetStretchMaxWidth()

	giedsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunGoGiEditor)
	prsc := gi.ActiveKeyMap.ChordForFun(gi.KeyFunPrefs)

	title := trow.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	title.Text = `This is a <b>demonstration</b> of the
<span style="color:red">various</span> <a href="https://github.com/goki/gi">GoGi</a> <i>Widgets</i><br>
<large>Shortcuts: <kbd>` + string(prsc) + `</kbd> = Preferences,
<kbd>` + string(giedsc) + `</kbd> = Editor, <kbd>Ctrl/Cmd +/-</kbd> = zoom</large><br>
See <a href="https://github.com/goki/gi/blob/master/examples/widgets/README.md">README</a> for detailed info and things to try.`
	title.SetProp("white-space", gi.WhiteSpaceNormal) // wrap
	title.SetProp("text-align", gi.AlignCenter)       // note: this also sets horizontal-align, which controls the "box" that the text is rendered in..
	title.SetProp("vertical-align", gi.AlignCenter)
	title.SetProp("font-family", "Times New Roman, serif")
	title.SetProp("font-size", "x-large")
	// title.SetProp("letter-spacing", 2)
	title.SetProp("line-height", 1.5)
	title.SetStretchMaxWidth()
	title.SetStretchMaxHeight()

	//////////////////////////////////////////
	//      Buttons

	mfr.AddNewChild(gi.KiT_Space, "blspc")
	blrow := mfr.AddNewChild(gi.KiT_Layout, "blrow").(*gi.Layout)
	blab := blrow.AddNewChild(gi.KiT_Label, "blab").(*gi.Label)
	blab.Text = "Buttons:"
	blab.Selectable = true

	brow := mfr.AddNewChild(gi.KiT_Layout, "brow").(*gi.Layout)
	brow.Lay = gi.LayoutHoriz
	brow.SetProp("spacing", units.NewValue(2, units.Ex))

	brow.SetProp("horizontal-align", gi.AlignLeft)
	// brow.SetProp("horizontal-align", gi.AlignJustify)
	brow.SetStretchMaxWidth()

	button1 := brow.AddNewChild(gi.KiT_Button, "button1").(*gi.Button)
	button1.SetProp("#icon", ki.Props{ // note: must come before SetIcon
		"width":  units.NewValue(1.5, units.Em),
		"height": units.NewValue(1.5, units.Em),
	})
	button1.Tooltip = "press this <i>button</i> to pop up a dialog box"

	button1.SetIcon(icnm)
	button1.ButtonSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received button signal: %v from button: %v\n", gi.ButtonSignals(sig), send.Name())
		if sig == int64(gi.ButtonClicked) { // note: 3 diff ButtonSig sig's possible -- important to check
			// vp.Win.Quit()
			gi.StringPromptDialog(vp, "", "Enter value here..",
				gi.DlgOpts{Title: "Button1 Dialog", Prompt: "This is a string prompt dialog!  Various specific types of dialogs are available."},
				rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					dlg := send.(*gi.Dialog)
					if sig == int64(gi.DialogAccepted) {
						val := gi.StringPromptDialogValue(dlg)
						fmt.Printf("got string value: %v\n", val)
					}
				})
		}
	})

	button2 := brow.AddNewChild(gi.KiT_Button, "button2").(*gi.Button)
	button2.SetText("Open GoGiEditor")
	// button2.SetProp("background-color", "#EDF")
	button2.Tooltip = "This button will open the GoGi GUI editor where you can edit this very GUI and see it update dynamically as you change things"
	button2.ButtonSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received button signal: %v from button: %v\n", gi.ButtonSignals(sig), send.Name())
		if sig == int64(gi.ButtonClicked) {
			giv.GoGiEditorDialog(vp)
		}
	})

	checkbox := brow.AddNewChild(gi.KiT_CheckBox, "checkbox").(*gi.CheckBox)
	checkbox.Text = "Toggle"

	// note: receiver for menut items with shortcuts must be a Node2D or Window
	mb1 := brow.AddNewChild(gi.KiT_MenuButton, "menubutton1").(*gi.MenuButton)
	mb1.SetText("Menu Button")
	mb1.Menu.AddAction(gi.ActOpts{Label: "Menu Item 1", Shortcut: "Shift+Control+1", Data: 1},
		win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mi2 := mb1.Menu.AddAction(gi.ActOpts{Label: "Menu Item 2", Data: 2}, nil, nil)

	mi2.Menu.AddAction(gi.ActOpts{Label: "Sub Menu Item 2", Data: 2.1},
		win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	mb1.Menu.AddSeparator("sep1")

	mb1.Menu.AddAction(gi.ActOpts{Label: "Menu Item 3", Shortcut: "Control+3", Data: 3},
		win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("Received menu action data: %v from menu action: %v\n", data, send.Name())
		})

	//////////////////////////////////////////
	//      Sliders

	mfr.AddNewChild(gi.KiT_Space, "slspc")
	slrow := mfr.AddNewChild(gi.KiT_Layout, "slrow").(*gi.Layout)
	slab := slrow.AddNewChild(gi.KiT_Label, "slab").(*gi.Label)
	slab.Text = "Sliders:"

	srow := mfr.AddNewChild(gi.KiT_Layout, "srow").(*gi.Layout)
	srow.Lay = gi.LayoutHoriz
	srow.SetProp("spacing", units.NewValue(2, units.Ex))
	srow.SetProp("horizontal-align", "left")
	srow.SetStretchMaxWidth()

	slider1 := srow.AddNewChild(gi.KiT_Slider, "slider1").(*gi.Slider)
	slider1.Dim = gi.X
	slider1.Class = "hslides"
	slider1.Defaults()
	slider1.SetMinPrefWidth(units.NewValue(20, units.Em))
	slider1.SetMinPrefHeight(units.NewValue(2, units.Em))
	slider1.SetValue(0.5)
	slider1.Snap = true
	slider1.Tracking = true
	slider1.Icon = gi.IconName("widget-circlebutton-on")

	slider2 := srow.AddNewChild(gi.KiT_Slider, "slider2").(*gi.Slider)
	slider2.Dim = gi.Y
	slider2.Defaults()
	slider2.SetMinPrefHeight(units.NewValue(10, units.Em))
	slider2.SetMinPrefWidth(units.NewValue(1, units.Em))
	slider2.SetStretchMaxHeight()
	slider2.SetValue(0.5)

	slider1.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received slider signal: %v from slider: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
	})

	slider2.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received slider signal: %v from slider: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
	})

	scrollbar1 := srow.AddNewChild(gi.KiT_ScrollBar, "scrollbar1").(*gi.ScrollBar)
	scrollbar1.Dim = gi.X
	scrollbar1.Class = "hslides"
	scrollbar1.Defaults()
	scrollbar1.SetMinPrefWidth(units.NewValue(20, units.Em))
	scrollbar1.SetMinPrefHeight(units.NewValue(1, units.Em))
	scrollbar1.SetThumbValue(0.25)
	scrollbar1.SetValue(0.25)
	// scrollbar1.Snap = true
	scrollbar1.Tracking = true
	scrollbar1.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received scrollbar signal: %v from scrollbar: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
	})

	scrollbar2 := srow.AddNewChild(gi.KiT_ScrollBar, "scrollbar2").(*gi.ScrollBar)
	scrollbar2.Dim = gi.Y
	scrollbar2.Defaults()
	scrollbar2.SetMinPrefHeight(units.NewValue(10, units.Em))
	scrollbar2.SetMinPrefWidth(units.NewValue(1, units.Em))
	scrollbar2.SetStretchMaxHeight()
	scrollbar2.SetThumbValue(0.1)
	scrollbar2.SetValue(0.5)
	scrollbar2.SliderSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received scrollbar signal: %v from scrollbar: %v with data: %v\n", gi.SliderSignals(sig), send.Name(), data)
	})

	//////////////////////////////////////////
	//      Text Widgets

	mfr.AddNewChild(gi.KiT_Space, "tlspc")
	txlrow := mfr.AddNewChild(gi.KiT_Layout, "txlrow").(*gi.Layout)
	txlab := txlrow.AddNewChild(gi.KiT_Label, "txlab").(*gi.Label)
	txlab.Text = "Text Widgets:"
	txrow := mfr.AddNewChild(gi.KiT_Layout, "txrow").(*gi.Layout)
	txrow.Lay = gi.LayoutHoriz
	txrow.SetProp("spacing", units.NewValue(2, units.Ex))
	// txrow.SetProp("horizontal-align", gi.AlignJustify)
	txrow.SetStretchMaxWidth()

	edit1 := txrow.AddNewChild(gi.KiT_TextField, "edit1").(*gi.TextField)
	edit1.Placeholder = "Enter text here..."
	// edit1.SetText("Edit this text")
	edit1.SetProp("min-width", "20em")
	edit1.SetCompleter(nil, Complete, CompleteEdit) // gets us word demo completion
	edit1.TextFieldSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("Received line edit signal: %v from edit: %v with data: %v\n", gi.TextFieldSignals(sig), send.Name(), data)
	})
	// edit1.SetProp("inactive", true)

	sb := txrow.AddNewChild(gi.KiT_SpinBox, "spin").(*gi.SpinBox)
	sb.Defaults()
	sb.HasMin = true
	sb.Min = 0.0
	sb.SpinBoxSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("SpinBox %v value changed: %v\n", send.Name(), data)
	})

	cb := txrow.AddNewChild(gi.KiT_ComboBox, "combo").(*gi.ComboBox)
	cb.ItemsFromTypes(kit.Types.AllImplementersOf(reflect.TypeOf((*gi.Node2D)(nil)).Elem(), false), true, true, 50)
	cb.ComboSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("ComboBox %v selected index: %v data: %v\n", send.Name(), sig, data)
	})

	//////////////////////////////////////////
	//      Main Menu

	appnm := oswin.TheApp.Name()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "File", "Edit", "Window"})

	amen := win.MainMenu.KnownChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	// note: Command in shortcuts is automatically translated into Control for
	// Linux, Windows or Meta for MacOS
	fmen := win.MainMenu.KnownChildByName("File", 0).(*gi.Action)
	fmen.Menu = make(gi.Menu, 0, 10)
	fmen.Menu.AddAction(gi.ActOpts{Label: "New", Shortcut: "Command+N"},
		rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("File:New menu action triggered\n")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Open", Shortcut: "Command+O"},
		rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("File:Open menu action triggered\n")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Save", Shortcut: "Command+S"},
		rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("File:Save menu action triggered\n")
		})
	fmen.Menu.AddAction(gi.ActOpts{Label: "Save As..", Shortcut: "Shift+Command+S"},
		rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			fmt.Printf("File:SaveAs menu action triggered\n")
		})
	fmen.Menu.AddSeparator("csep")
	fmen.Menu.AddAction(gi.ActOpts{Label: "Close Window", Shortcut: "Command+W"},
		win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			win.OSWin.CloseReq()
		})

	emen := win.MainMenu.KnownChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	inQuitPrompt := false
	oswin.TheApp.SetQuitReqFunc(func() {
		if !inQuitPrompt {
			inQuitPrompt = true
			gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Quit?",
				Prompt: "Are you <i>sure</i> you want to quit?"}, true, true,
				win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.DialogAccepted) {
						oswin.TheApp.Quit()
					} else {
						inQuitPrompt = false
					}
				})
		}
	})

	oswin.TheApp.SetQuitCleanFunc(func() {
		fmt.Printf("Doing final Quit cleanup here..\n")
	})

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !inClosePrompt {
			inClosePrompt = true
			gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Close Window?",
				Prompt: "Are you <i>sure</i> you want to close the window?  This will Quit the App as well."}, true, true,
				win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.DialogAccepted) {
						oswin.TheApp.Quit()
					} else {
						inClosePrompt = false
					}
				})
		}
	})

	win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
		fmt.Printf("Doing final Close cleanup here..\n")
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()

	// note: may eventually get down here on a well-behaved quit, but better
	// to handle cleanup above using QuitCleanFunc, which happens before all
	// windows are closed etc
	fmt.Printf("main loop ended\n")
}

func Complete(text string, pos token.Position) (matches complete.Completions, seed string) {
	seed = complete.SeedWhiteSpace(text)
	possibles := complete.MatchSeedString(words, seed)
	for _, p := range possibles {
		m := complete.Completion{Text: p, Icon: ""}
		matches = append(matches, m)
	}
	return matches, seed
}

func CompleteEdit(text string, cursorPos int, selection string, seed string) (s string, delta int) {
	s, delta = complete.EditWord(text, cursorPos, selection, seed)
	return s, delta
}

var words = []string{"a", "able", "about", "above", "act", "add", "afraid", "after", "again", "against", "age", "ago", "agree", "air", "all",
	"allow", "also", "always", "am", "among", "an", "and", "anger", "animal", "answer", "any", "appear", "apple", "are",
	"area", "arm", "arrange", "arrive", "art", "as", "ask", "at", "atom", "baby", "back", "bad", "ball", "band", "bank",
	"bar", "base", "basic", "bat", "be", "bear", "beat", "beauty", "bed", "been", "before", "began", "begin", "behind",
	"believe", "bell", "best", "better", "between", "big", "bird", "bit", "black", "block", "blood", "blow", "blue",
	"board", "boat", "body", "bone", "book", "born", "both", "bottom", "bought", "box", "boy", "branch", "bread", "break",
	"bright", "bring", "broad", "broke", "brother", "brought", "brown", "build", "burn", "busy", "but", "buy", "by", "call",
	"came", "camp", "can", "capital", "captain", "car", "card", "care", "carry", "case", "cat", "catch", "caught", "cause",
	"cell", "cent", "center", "century", "certain", "chair", "chance", "change", "character", "charge", "chart", "check",
	"chick", "chief", "child", "children", "choose", "chord", "circle", "city", "claim", "class", "clean", "clear", "climb",
	"clock", "close", "clothe", "cloud", "coast", "coat", "cold", "collect", "colony", "color", "column", "come", "common",
	"company", "compare", "complete", "condition", "connect", "consider", "consonant", "contain", "continent", "continue",
	"control", "cook", "cool", "copy", "corn", "corner", "correct", "cost", "cotton", "could", "count", "country", "course",
	"cover", "cow", "crease", "create", "crop", "cross", "crowd", "cry", "current", "cut", "dad", "dance", "danger", "dark",
	"day", "dead", "deal", "dear", "death", "decide", "decimal", "deep", "degree", "depend", "describe", "desert", "design",
	"determine", "develop", "dictionary", "did", "die", "differ", "difficult", "direct", "discuss", "distant", "divide",
	"division", "do", "doctor", "does", "dog", "dollar", "don't", "done", "door", "double", "down", "draw", "dream",
	"dress", "drink", "drive", "drop", "dry", "duck", "during", "each", "ear", "early", "earth", "ease", "east", "eat",
	"edge", "effect", "egg", "eight", "either", "electric", "element", "else", "end", "enemy", "energy", "engine", "enough",
	"enter", "equal", "equate", "especially", "even", "evening", "event", "ever", "every", "exact", "example", "except",
	"excite", "exercise", "expect", "experience", "experiment", "eye", "face", "fact", "fair", "fall", "family", "famous",
	"far", "farm", "fast", "fat", "father", "favor", "fear", "feed", "feel", "feet", "fell", "felt", "few", "field", "fig",
	"fight", "figure", "fill", "final", "find", "fine", "finger", "finish", "fire", "first", "fish", "fit", "five", "flat",
	"floor", "flow", "flower", "fly", "follow", "food", "foot", "for", "force", "forest", "form", "forward", "found",
	"four", "fraction", "free", "fresh", "friend", "from", "front", "fruit", "full", "fun", "game", "garden", "gas",
	"gather", "gave", "general", "gentle", "get", "girl", "give", "glad", "glass", "go", "gold", "gone", "good", "got",
	"govern", "grand", "grass", "gray", "great", "green", "grew", "ground", "group", "grow", "guess", "guide", "gun", "had",
	"hair", "half", "hand", "happen", "happy", "hard", "has", "hat", "have", "he", "head", "hear", "heard", "heart", "heat",
	"heavy", "held", "help", "her", "here", "high", "hill", "him", "his", "history", "hit", "hold", "hole", "home", "hope",
	"horse", "hot", "hot", "hour", "house", "how", "huge", "human", "hundred", "hunt", "hurry", "I", "ice", "idea", "if",
	"imagine", "in", "inch", "include", "indicate", "industry", "insect", "instant", "instrument", "interest", "invent",
	"iron", "is", "island", "it", "job", "join", "joy", "jump", "just", "keep", "kept", "key", "kill", "kind", "king",
	"knew", "know", "lady", "lake", "land", "language", "large", "last", "late", "laugh", "law", "lay", "lead", "learn",
	"least", "leave", "led", "left", "leg", "length", "less", "let", "letter", "level", "lie", "life", "lift", "light",
	"like", "line", "liquid", "list", "listen", "little", "live", "locate", "log", "lone", "long", "look", "lost", "lot",
	"loud", "love", "low", "machine", "made", "magnet", "main", "major", "make", "man", "many", "map", "mark", "market",
	"mass", "master", "match", "material", "matter", "may", "me", "mean", "meant", "measure", "meat", "meet", "melody",
	"men", "metal", "method", "middle", "might", "mile", "milk", "million", "mind", "mine", "minute", "miss", "mix",
	"modern", "molecule", "moment", "money", "month", "moon", "more", "morning", "most", "mother", "motion", "mount",
	"mountain", "mouth", "move", "much", "multiply", "music", "must", "my", "name", "nation", "natural", "nature", "near",
	"necessary", "neck", "need", "neighbor", "never", "new", "next", "night", "nine", "no", "noise", "noon", "nor", "north",
	"nose", "note", "nothing", "notice", "noun", "now", "number", "numeral", "object", "observe", "occur", "ocean", "of",
	"off", "offer", "office", "often", "oh", "oil", "old", "on", "once", "one", "only", "open", "operate", "opposite", "or",
	"order", "organ", "original", "other", "our", "out", "over", "own", "oxygen", "page", "paint", "pair", "paper",
	"paragraph", "parent", "part", "particular", "party", "pass", "past", "path", "pattern", "pay", "people", "perhaps",
	"period", "person", "phrase", "pick", "picture", "piece", "pitch", "place", "plain", "plan", "plane", "planet", "plant",
	"play", "please", "plural", "poem", "point", "poor", "populate", "port", "pose", "position", "possible", "post",
	"pound", "power", "practice", "prepare", "present", "press", "pretty", "print", "probable", "problem", "process",
	"produce", "product", "proper", "property", "protect", "prove", "provide", "pull", "push", "put", "quart", "question",
	"quick", "quiet", "quite", "quotient", "race", "radio", "rail", "rain", "raise", "ran", "range", "rather", "reach",
	"read", "ready", "real", "reason", "receive", "record", "red", "region", "remember", "repeat", "reply", "represent",
	"require", "rest", "result", "rich", "ride", "right", "ring", "rise", "river", "road", "rock", "roll", "room", "root",
	"rope", "rose", "round", "row", "rub", "rule", "run", "safe", "said", "sail", "salt", "same", "sand", "sat", "save",
	"saw", "say", "scale", "school", "science", "score", "sea", "search", "season", "seat", "second", "section", "see",
	"seed", "seem", "segment", "select", "self", "sell", "send", "sense", "sent", "sentence", "separate", "serve", "set",
	"settle", "seven", "several", "shall", "shape", "share", "sharp", "she", "sheet", "shell", "shine", "ship", "shoe",
	"shop", "shore", "short", "should", "shoulder", "shout", "show", "side", "sight", "sign", "silent", "silver", "similar",
	"simple", "since", "sing", "single", "sister", "sit", "six", "size", "skill", "skin", "sky", "slave", "sleep", "slip",
	"slow", "small", "smell", "smile", "snow", "so", "soft", "soil", "soldier", "solution", "solve", "some", "son", "song",
	"soon", "sound", "south", "space", "speak", "special", "speech", "speed", "spell", "spend", "spoke", "spot", "spread",
	"spring", "square", "stand", "star", "start", "state", "station", "stay", "stead", "steam", "steel", "step", "stick",
	"still", "stone", "stood", "stop", "store", "story", "straight", "strange", "stream", "street", "stretch", "string",
	"strong", "student", "study", "subject", "substance", "subtract", "success", "such", "sudden", "suffix", "sugar",
	"suggest", "suit", "summer", "sun", "supply", "support", "sure", "surface", "surprise", "swim", "syllable", "symbol",
	"system", "table", "tail", "take", "talk", "tall", "teach", "team", "teeth", "tell", "temperature", "ten", "term",
	"test", "than", "thank", "that", "the", "their", "them", "then", "there", "these", "they", "thick", "thin", "thing",
	"think", "third", "this", "those", "though", "thought", "thousand", "three", "through", "throw", "thus", "tie", "time",
	"tiny", "tire", "to", "together", "told", "tone", "too", "took", "tool", "top", "total", "touch", "toward", "town",
	"track", "trade", "train", "travel", "tree", "triangle", "trip", "trouble", "truck", "true", "try", "tube", "turn",
	"twenty", "two", "type", "under", "unit", "until", "up", "us", "use", "usual", "valley", "value", "vary", "verb",
	"very", "view", "village", "visit", "voice", "vowel", "wait", "walk", "wall", "want", "war", "warm", "was", "wash",
	"watch", "water", "wave", "way", "we", "wear", "weather", "week", "weight", "well", "went", "were", "west", "what",
	"wheel", "when", "where", "whether", "which", "while", "white", "who", "whole", "whose", "why", "wide", "wife", "wild",
	"will", "win", "wind", "window", "wing", "winter", "wire", "wish", "with", "woman", "women", "won't", "wonder", "wood",
	"word", "work", "world", "would", "write", "written", "wrong", "wrote", "yard", "year", "yellow", "yes", "yet", "you",
	"young", "your"}
