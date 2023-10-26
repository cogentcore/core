// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"goki.dev/colors"
	"goki.dev/colors/matcolor"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/grows/jsons"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// GiEditor represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame with an optional title, and a button
// box at the bottom where methods can be invoked
type GiEditor struct {
	gi.Frame

	// root of tree being edited
	KiRoot ki.Ki

	// has the root changed via gui actions?  updated from treeview and structview for changes
	Changed bool `set:"-"`

	// current filename for saving / loading
	Filename gi.FileName
}

func (ge *GiEditor) OnInit() {
	ge.Style(func(s *styles.Style) {
		s.Color = colors.Scheme.OnBackground
		s.SetStretchMax()
		s.Margin.Set(units.Dp(8))
	})
	ge.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(ge.This()) {
		case "title":
			title := w.(*gi.Label)
			title.Type = gi.LabelHeadlineSmall
			title.Style(func(s *styles.Style) {
				s.SetStretchMaxWidth()
				s.AlignH = styles.AlignCenter
				s.AlignV = styles.AlignTop
			})
		}
	})
}

// Update updates the objects being edited (e.g., updating display changes)
func (ge *GiEditor) Update() { //gti:add
	if ge.KiRoot == nil {
		return
	}
	if w, ok := ge.KiRoot.(gi.Widget); ok {
		w.AsWidget().SetNeedsRender()
	}
}

// Save saves tree to current filename, in a standard JSON-formatted file
func (ge *GiEditor) Save() { //gti:add
	if ge.KiRoot == nil {
		return
	}
	if ge.Filename == "" {
		return
	}

	grr.Log0(jsons.Save(ge.KiRoot, string(ge.Filename)))
	ge.Changed = false
}

// SaveAs saves tree to given filename, in a standard JSON-formatted file
func (ge *GiEditor) SaveAs(filename gi.FileName) { //gti:add
	if ge.KiRoot == nil {
		return
	}
	grr.Log0(jsons.Save(ge.KiRoot, string(filename)))
	ge.Changed = false
	ge.Filename = filename
	ge.SetNeedsRender() // notify our editor
}

// Open opens tree from given filename, in a standard JSON-formatted file
func (ge *GiEditor) Open(filename gi.FileName) { //gti:add
	if ge.KiRoot == nil {
		return
	}
	grr.Log0(jsons.Open(ge.KiRoot, string(filename)))
	ge.Filename = filename
	ge.SetNeedsRender() // notify our editor
}

// EditColorScheme pulls up a window to edit the current color scheme
func (ge *GiEditor) EditColorScheme() { //gti:add
	if gi.ActivateExistingMainWindow(&colors.Schemes) {
		return
	}

	sc := gi.NewScene("gogi-color-scheme")
	sc.Title = "GoGi Color Scheme"
	sc.Lay = gi.LayoutVert
	sc.Data = &colors.Schemes

	key := &matcolor.Key{
		Primary:        colors.FromRGB(123, 135, 122),
		Secondary:      colors.FromRGB(106, 196, 178),
		Tertiary:       colors.FromRGB(106, 196, 178),
		Error:          colors.FromRGB(219, 46, 37),
		Neutral:        colors.FromRGB(133, 131, 121),
		NeutralVariant: colors.FromRGB(107, 106, 101),
	}
	p := matcolor.NewPalette(key)
	schemes := matcolor.NewSchemes(p)

	kv := NewStructView(sc, "kv")
	kv.SetStruct(key)
	kv.SetStretchMax()

	split := gi.NewSplits(sc, "split")
	split.Dim = mat32.X

	svl := NewStructView(split, "svl")
	svl.SetStruct(&schemes.Light)
	svl.SetStretchMax()

	svd := NewStructView(split, "svd")
	svd.SetStruct(&schemes.Dark)
	svd.SetStretchMax()

	kv.OnChange(func(e events.Event) {
		p = matcolor.NewPalette(key)
		schemes = matcolor.NewSchemes(p)
		colors.Schemes = schemes
		gi.Prefs.UpdateAll()
		svl.UpdateFields()
		svd.UpdateFields()
	})

	gi.NewWindow(sc).Run()
}

// ToggleSelectionMode toggles the editor between selection mode or not
func (ge *GiEditor) ToggleSelectionMode() { //gti:add
	return
	// TODO(kai/sel): implement
	// if win, ok := ge.KiRoot.(*gi.RenderWin); ok {
	// 	if !win.HasFlag(WinSelectionMode) && win.SelectedWidgetChan == nil {
	// 		win.SelectedWidgetChan = make(chan *gi.WidgetBase)
	// 	}
	// 	win.SetFlag(!win.HasFlag(WinSelectionMode), WinSelectionMode)
	// }
}

// SetRoot sets the source root and ensures everything is configured
func (ge *GiEditor) SetRoot(root ki.Ki) {
	updt := false
	if ge.KiRoot != root {
		updt = ge.UpdateStart()
		ge.KiRoot = root
		// ge.GetAllUpdates(root)
	}
	ge.Config(ge.Sc)
	ge.UpdateEnd(updt)
}

// // GetAllUpdates connects to all nodes in the tree to receive notification of changes
// func (ge *GiEditor) GetAllUpdates(root ki.Ki) {
// 	ge.KiRoot.WalkPre(func(k ki.Ki) bool {
// 		k.NodeSignal().Connect(ge.This(), func(recv, send ki.Ki, sig int64, data any) {
// 			gee := recv.Embed(TypeGiEditor).(*GiEditor)
// 			if !gee.Changed {
// 				fmt.Printf("GiEditor: Tree changed with signal: %v\n", ki.NodeSignals(sig))
// 				gee.Changed = true
// 			}
// 		})
// 		return ki.Continue
// 	})
// }

// Config configures the widget
func (ge *GiEditor) ConfigWidget(sc *gi.Scene) {
	if ge.KiRoot == nil {
		return
	}
	ge.Lay = gi.LayoutVert
	ge.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := ki.Config{}
	config.Add(gi.LabelType, "title")
	config.Add(gi.ToolbarType, "toolbar")
	config.Add(gi.SplitsType, "splits")
	mods, updt := ge.ConfigChildren(config)
	ge.SetTitle(fmt.Sprintf("GoGi Editor of Ki Node Tree: %v", ge.KiRoot.Name()))
	ge.ConfigSplits()
	ge.ConfigToolbar()
	if mods {
		ge.UpdateEnd(updt)
	}
}

// SetTitle sets the optional title and updates the Title label
func (ge *GiEditor) SetTitle(title string) {
	lab := ge.TitleWidget()
	lab.Text = title
}

// Title returns the title label widget, and its index, within frame
func (ge *GiEditor) TitleWidget() *gi.Label {
	return ge.ChildByName("title", 0).(*gi.Label)
}

// Splits returns the main Splits
func (ge *GiEditor) Splits() *gi.Splits {
	return ge.ChildByName("splits", 2).(*gi.Splits)
}

// TreeView returns the main TreeSyncView
func (ge *GiEditor) TreeView() *TreeView {
	return ge.Splits().Child(0).Child(0).(*TreeView)
}

// StructView returns the main StructView
func (ge *GiEditor) StructView() *StructView {
	return ge.Splits().Child(1).(*StructView)
}

// ToolbarWidget returns the toolbar widget
func (ge *GiEditor) ToolbarWidget() *gi.Toolbar {
	return ge.ChildByName("toolbar", 1).(*gi.Toolbar)
}

// ConfigToolbar adds a GiEditor toolbar.
func (ge *GiEditor) ConfigToolbar() {
	tb := ge.ToolbarWidget()
	if tb != nil && tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	gi.ToolbarFor(ge, tb)
}

// ConfigSplits configures the Splits.
func (ge *GiEditor) ConfigSplits() {
	if ge.KiRoot == nil {
		return
	}
	split := ge.Splits()
	// split.Dim = mat32.Y
	split.Dim = mat32.X

	if len(split.Kids) == 0 {
		tvfr := gi.NewFrame(split, "tvfr").SetLayout(gi.LayoutHoriz)
		tv := NewTreeView(tvfr, "tv")
		sv := NewStructView(split, "sv")
		tv.OnSelect(func(e events.Event) {
			if len(tv.SelectedNodes) > 0 {
				sv.SetStruct(tv.SelectedNodes[0].SyncNode)
			}
		})
		split.SetSplits(.3, .7)
	}
	tv := ge.TreeView()
	tv.SyncRootNode(ge.KiRoot)
	sv := ge.StructView()
	sv.SetStruct(ge.KiRoot)
}

func (ge *GiEditor) SetChanged() {
	ge.Changed = true
	// ge.Toolbar().UpdateButtons() // nil safe
}

func (ge *GiEditor) Render(sc *gi.Scene) {
	// ge.Toolbar().UpdateButtons()
	// if win := ge.ParentRenderWin(); win != nil {
	// 	if !win.Is(WinResizing) {
	// 		win.MainMenuUpdateActives()
	// 	}
	// }
	ge.Frame.Render(sc)
}

func (ge *GiEditor) Toolbar(tb *gi.Toolbar) {
	up := NewFuncButton(tb, ge.Update).SetIcon(icons.Refresh)
	up.SetUpdateFunc(func() {
		up.SetEnabled(ge.Changed)
	})
	sel := NewFuncButton(tb, ge.ToggleSelectionMode).SetText("Select Element").SetIcon(icons.ArrowSelectorTool)
	sel.SetUpdateFunc(func() {
		sc, ok := ge.KiRoot.(*gi.Scene)
		sc.SetEnabled(ok)
		if !ok {
			return
		}
		// TODO(kai/sel): check if has flag
	})
	gi.NewSeparator(tb)
	op := NewFuncButton(tb, ge.Open).SetIcon(icons.FileOpen).SetShortcutKey(gi.KeyFunMenuOpen)
	op.Args[0].SetValue(ge.Filename)
	op.Args[0].SetTag("ext", ".json")
	save := NewFuncButton(tb, ge.Save).SetShortcutKey(gi.KeyFunMenuSave)
	save.SetUpdateFunc(func() {
		save.SetEnabledUpdt(ge.Changed && ge.Filename != "")
	})
	sa := NewFuncButton(tb, ge.SaveAs).SetShortcutKey(gi.KeyFunMenuSaveAs)
	sa.Args[0].SetValue(ge.Filename)
	sa.Args[0].SetTag("ext", ".json")
	gi.NewSeparator(tb)
	NewFuncButton(tb, ge.EditColorScheme).SetIcon(icons.Colors)
}

func (ge *GiEditor) MenuBar(mb *gi.MenuBar) {
	NewFuncButton(mb, ge.Update)
}

var GiEditorProps = ki.Props{
	"Toolbar": ki.PropSlice{
		{"Update", ki.Props{
			"icon": icons.Refresh,
			"updtfunc": ActionUpdateFunc(func(gei any, act *gi.Button) {
				ge := gei.(*GiEditor)
				act.SetEnabledUpdt(ge.Changed)
			}),
		}},
		{"sep-sel", ki.BlankProp{}},
		{"ToggleSelectionMode", ki.Props{
			"icon": icons.ArrowSelectorTool,
			"desc": "Select an element in the window to edit it",
			"updtfunc": ActionUpdateFunc(func(gei any, act *gi.Button) {
				ge := gei.(*GiEditor)
				_ = ge
				// win, ok := ge.KiRoot.(*gi.RenderWin) // todo
				// ok := true
				// act.SetEnabledStateUpdt(ok)
				// if ok {
				// 	if win.HasFlag(WinSelectionMode) {
				// 		act.SetText("Disable Selection")
				// 	} else {
				// 		act.SetText("Enable Selection")
				// 	}
				// }
			}),
		}},
		{"sep-file", ki.BlankProp{}},
		{"Open", ki.Props{
			"label": "Open",
			"icon":  icons.FileOpen,
			"desc":  "Open a json-formatted Ki tree structure",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".json",
				}},
			},
		}},
		{"Save", ki.Props{
			"icon": icons.Save,
			"desc": "Save json-formatted Ki tree structure to existing filename",
			"updtfunc": ActionUpdateFunc(func(gei any, act *gi.Button) {
				ge := gei.(*GiEditor)
				act.SetEnabledUpdt(ge.Changed && ge.Filename != "")
			}),
		}},
		{"SaveAs", ki.Props{
			"label": "Save As...",
			"icon":  icons.SaveAs,
			"desc":  "Save as a json-formatted Ki tree structure",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".json",
				}},
			},
		}},
		{"sep-color", ki.BlankProp{}},
		{"EditColorScheme", ki.Props{
			"label": "Edit Color Scheme",
			"icon":  icons.Colors,
			"desc":  "View and edit the current color scheme",
		}},
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"Update", ki.Props{
				"updtfunc": ActionUpdateFunc(func(gei any, act *gi.Button) {
					ge := gei.(*GiEditor)
					act.SetEnabled(ge.Changed)
				}),
			}},
			{"sep-file", ki.BlankProp{}},
			{"Open", ki.Props{
				"shortcut": gi.KeyFunMenuOpen,
				"desc":     "Open a json-formatted Ki tree structure",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Filename",
						"ext":           ".json",
					}},
				},
			}},
			{"Save", ki.Props{
				"shortcut": gi.KeyFunMenuSave,
				"desc":     "Save json-formatted Ki tree structure to existing filename",
				"updtfunc": ActionUpdateFunc(func(gei any, act *gi.Button) {
					ge := gei.(*GiEditor)
					act.SetEnabled(ge.Changed && ge.Filename != "")
				}),
			}},
			{"SaveAs", ki.Props{
				"shortcut": gi.KeyFunMenuSaveAs,
				"label":    "Save As...",
				"desc":     "Save as a json-formatted Ki tree structure",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Filename",
						"ext":           ".json",
					}},
				},
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close RenderWin", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste Dupe"},
		{"RenderWin", "RenderWins"},
	},
}

// GoGiEditorDialog opens an interactive editor of the given Ki tree, at its
// root, returns GiEditor and window
func GoGiEditorDialog(obj ki.Ki) {
	if gi.ActivateExistingMainWindow(obj) {
		return
	}
	sc := gi.NewScene("gogi-editor")
	sc.Title = "GoGi Editor"
	if obj != nil {
		sc.Nm += "-" + obj.Name()
		sc.Title += ": " + obj.Name()
	}
	sc.Lay = gi.LayoutVert

	ge := NewGiEditor(sc, "editor")
	ge.SetRoot(obj)

	// mmen := win.MainMenu
	// MainMenuView(ge, win, mmen)

	tb := ge.ToolbarWidget()
	tb.UpdateButtons()

	// ge.SelectionLoop()

	/*
		inClosePrompt := false
		win.RenderWin.SetCloseReqFunc(func(w goosi.RenderWin) {
			if !ge.Changed {
				win.Close()
				return
			}
			if inClosePrompt {
				return
			}
			inClosePrompt = true
			gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
				Prompt: "Do you want to save your changes?  If so, Cancel and then Save"},
				[]string{"Close Without Saving", "Cancel"}, func(dlg *gi.Dialog) {
					switch sig {
					case 0:
						win.Close()
					case 1:
						// default is to do nothing, i.e., cancel
						inClosePrompt = false
					}
				})
		})
	*/

	gi.NewWindow(sc).Run()
}

// SelectionLoop, if [KiRoot] is a [gi.RenderWin], runs a loop in a separate goroutine
// that listens to the [RenderWin.SelectedWidgetChan] channel and selects selected elements.
func (ge *GiEditor) SelectionLoop() {
	/*
		if win, ok := ge.KiRoot.(*gi.RenderWin); ok {
			go func() {
				if win.SelectedWidgetChan == nil {
					win.SelectedWidgetChan = make(chan *gi.WidgetBase)
				}
				for {
					sw := <-win.SelectedWidgetChan
					tv := ge.TreeView().FindSrcNode(sw.This())
					if tv == nil {
						log.Printf("GiEditor on %v: tree view source node missing for", sw)
					} else {
						// TODO: make quicker
						wupdt := tv.RootView.TopUpdateStart()
						updt := tv.RootView.UpdateStart()

						tv.RootView.CloseAll()
						tv.RootView.UnselectAll()
						tv.OpenParents()
						tv.SelectAction(events.SelectOne)
						tv.ScrollToMe()

						tv.RootView.UpdateEnd(updt)
						tv.RootView.TopUpdateEnd(wupdt)
					}
				}
			}()
		}
	*/
}
