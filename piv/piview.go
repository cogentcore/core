// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package piv provides the PiView object for the full GUI view of the
// interactive parser (pi) system.
package piv

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi"
)

// todo:
//
// need to have textview using our style tokens so we can see the thing getting styled
// directly from the lexer as it proceeds!  replace histyle tags with tokens.Tokens
//
// save all existing chroma styles, then load back in -- names should then transfer!
// need to just add background style name

// These are then the fixed indices of the different elements in the splitview
const (
	LexRulesIdx = iota
	ParseRulesIdx
	StructViewIdx
	AstOutIdx
	TextViewIdx
)

// PiView provides the interactive GUI view for constructing and testing the
// lexer and parser
type PiView struct {
	gi.Frame
	Parser   pi.Parser   `desc:"the parser we are viewing"`
	TestFile gi.FileName `desc:"the file for testing"`
	Filename gi.FileName `desc:"filename for saving parser"`
	Changed  bool        `json:"-" desc:"has the root changed?  we receive update signals from root for changes"`
	Buf      giv.TextBuf `json:"-" desc:"test file buffer"`
}

var KiT_PiView = kit.Types.AddType(&PiView{}, PiViewProps)

// InitView initializes the viewer / editor
func (pi *PiView) InitView() {
	pi.Parser.Init()
	mods, updt := pi.StdConfig()
	if !mods {
		updt = pi.UpdateStart()
	}
	pi.ConfigSplitView()
	pi.ConfigToolbar()
	pi.UpdateEnd(updt)
}

// Save saves lexer and parser rules to current filename, in a standard JSON-formatted file
func (pi *PiView) Save() {
	if pi.Filename == "" {
		return
	}
	pi.Parser.SaveJSON(string(pi.Filename))
	pi.Changed = false
	pi.UpdateSig() // notify our editor
}

// SaveAs saves lexer and parser rules to current filename, in a standard JSON-formatted file
func (pi *PiView) SaveAs(filename gi.FileName) {
	pi.Parser.SaveJSON(string(filename))
	pi.Changed = false
	pi.Filename = filename
	pi.UpdateSig() // notify our editor
}

// Open opens lexer and parser rules to current filename, in a standard JSON-formatted file
func (pi *PiView) Open(filename gi.FileName) {
	pi.Parser.OpenJSON(string(filename))
	pi.Filename = filename
	pi.InitView()
}

// OpenTest opens test file
func (pi *PiView) OpenTest(filename gi.FileName) {
	pi.Buf.OpenFile(filename)
	pi.TestFile = filename
}

// SaveTestAs saves the test file as..
func (pi *PiView) SaveTestAs(filename gi.FileName) {
	pi.Buf.SaveFile(filename)
	pi.TestFile = filename
}

// LexInit initializes / restarts lexing process for current test file
func (pi *PiView) LexInit() {
	pi.Parser.SetSrc(pi.Buf.Lines)
}

// LexNext does next step of lexing
func (pi *PiView) LexNext() {
	ok := pi.Parser.LexNext()
	if !ok {
		gi.PromptDialog(pi.Viewport, gi.DlgOpts{Title: "Lex at End",
			Prompt: "The Lexer is now at the end of available text"}, true, false, nil, nil)
	} else {
		pi.LexLine().SetText(pi.Parser.LexLineOut())
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//   GUI configs

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (pi *PiView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_Label, "lex-line")
	config.Add(gi.KiT_Label, "parse-line")
	config.Add(gi.KiT_SplitView, "splitview")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (pi *PiView) StdConfig() (mods, updt bool) {
	pi.Lay = gi.LayoutVert
	pi.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := pi.StdFrameConfig()
	mods, updt = pi.ConfigChildren(config, false)
	if mods {
		ll := pi.LexLine()
		ll.SetStretchMaxWidth()
		ll.Redrawable = true
		pl := pi.ParseLine()
		pl.SetStretchMaxWidth()
		pl.Redrawable = true
	}
	return
}

// LexLine returns the lex line label
func (pi *PiView) LexLine() *gi.Label {
	idx, ok := pi.Children().IndexByName("lex-line", 2)
	if !ok {
		return nil
	}
	return pi.KnownChild(idx).(*gi.Label)
}

// ParseLine returns the parse line label
func (pi *PiView) ParseLine() *gi.Label {
	idx, ok := pi.Children().IndexByName("parse-line", 3)
	if !ok {
		return nil
	}
	return pi.KnownChild(idx).(*gi.Label)
}

// SplitView returns the main SplitView
func (pi *PiView) SplitView() (*gi.SplitView, int) {
	idx, ok := pi.Children().IndexByName("splitview", 4)
	if !ok {
		return nil, -1
	}
	return pi.KnownChild(idx).(*gi.SplitView), idx
}

// LexTree returns the lex rules tree view
func (pi *PiView) LexTree() *giv.TreeView {
	split, _ := pi.SplitView()
	if split != nil {
		tv := split.KnownChild(LexRulesIdx).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// ParseTree returns the parse rules tree view
func (pi *PiView) ParseTree() *giv.TreeView {
	split, _ := pi.SplitView()
	if split != nil {
		tv := split.KnownChild(ParseRulesIdx).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// AstTree returns the Ast output tree view
func (pi *PiView) AstTree() *giv.TreeView {
	split, _ := pi.SplitView()
	if split != nil {
		tv := split.KnownChild(AstOutIdx).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// StructView returns the StructView for editing rules
func (pi *PiView) StructView() *giv.StructView {
	split, _ := pi.SplitView()
	if split != nil {
		return split.KnownChild(StructViewIdx).(*giv.StructView)
	}
	return nil
}

// ToolBar returns the toolbar widget
func (pi *PiView) ToolBar() *gi.ToolBar {
	idx, ok := pi.Children().IndexByName("toolbar", 0)
	if !ok {
		return nil
	}
	return pi.KnownChild(idx).(*gi.ToolBar)
}

// ConfigToolbar adds a PiView toolbar.
func (pi *PiView) ConfigToolbar() {
	tb := pi.ToolBar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	giv.ToolBarView(pi, pi.Viewport, tb)
}

// SplitViewConfig returns a TypeAndNameList for configuring the SplitView
func (pi *PiView) SplitViewConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Frame, "lex-tree-fr")
	config.Add(gi.KiT_Frame, "parse-tree-fr")
	config.Add(giv.KiT_StructView, "struct-view")
	config.Add(gi.KiT_Frame, "ast-tree-fr")
	config.Add(gi.KiT_Layout, "textview-lay")
	return config
}

// ConfigSplitView configures the SplitView.
func (pi *PiView) ConfigSplitView() {
	split, _ := pi.SplitView()
	if split == nil {
		return
	}
	split.Dim = gi.X

	split.SetProp("white-space", gi.WhiteSpacePreWrap)
	split.SetProp("tab-size", 4)

	config := pi.SplitViewConfig()
	mods, updt := split.ConfigChildren(config, true)
	if mods {
		lxfr := split.KnownChild(LexRulesIdx).(*gi.Frame)
		lxt := lxfr.AddNewChild(giv.KiT_TreeView, "lex-tree").(*giv.TreeView)
		lxt.SetRootNode(&pi.Parser.Lexer)

		prfr := split.KnownChild(ParseRulesIdx).(*gi.Frame)
		prt := prfr.AddNewChild(giv.KiT_TreeView, "parse-tree").(*giv.TreeView)
		prt.SetRootNode(&pi.Parser.Parser)

		astfr := split.KnownChild(AstOutIdx).(*gi.Frame)
		astt := astfr.AddNewChild(giv.KiT_TreeView, "ast-tree").(*giv.TreeView)
		astt.SetRootNode(&pi.Parser.Ast)

		txly := split.KnownChild(TextViewIdx).(*gi.Layout)
		txly.SetStretchMaxWidth()
		txly.SetStretchMaxHeight()
		txly.SetMinPrefWidth(units.NewValue(20, units.Ch))
		txly.SetMinPrefHeight(units.NewValue(10, units.Ch))

		txed := txly.AddNewChild(giv.KiT_TextView, "textview").(*giv.TextView)
		txed.Viewport = pi.Viewport
		txed.SetBuf(&pi.Buf)

		split.SetSplits(.15, .15, .25, .15, .3)
		split.UpdateEnd(updt)
	} else {
		pi.LexTree().SetRootNode(&pi.Parser.Lexer)
		pi.LexTree().Open()
		pi.ParseTree().SetRootNode(&pi.Parser.Parser)
		pi.ParseTree().Open()
		pi.AstTree().SetRootNode(&pi.Parser.Ast)
		pi.AstTree().Open()
		pi.StructView().SetStruct(&pi.Parser.Lexer, nil)
	}

	pi.LexTree().TreeViewSig.Connect(pi.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if data == nil {
			return
		}
		tvn, _ := data.(ki.Ki).Embed(giv.KiT_TreeView).(*giv.TreeView)
		pib, _ := recv.Embed(KiT_PiView).(*PiView)
		switch sig {
		case int64(giv.TreeViewSelected):
			pib.ViewNode(tvn)
		case int64(giv.TreeViewChanged):
			pib.SetChanged()
		}
	})

}

// ViewNode sets the StructView view to src node for given treeview
func (pi *PiView) ViewNode(tv *giv.TreeView) {
	sv := pi.StructView()
	if sv != nil {
		sv.SetStruct(tv.SrcNode.Ptr, nil)
	}
}

func (pi *PiView) SetChanged() {
	pi.Changed = true
	pi.ToolBar().UpdateActions() // nil safe
}

func (pi *PiView) FileNodeOpened(fn *giv.FileNode, tvn *giv.FileTreeView) {
	if fn.IsDir() {
		if !fn.IsOpen() {
			tvn.SetOpen()
			fn.OpenDir()
		}
	}
}

func (pi *PiView) FileNodeClosed(fn *giv.FileNode, tvn *giv.FileTreeView) {
	if fn.IsDir() {
		if fn.IsOpen() {
			fn.CloseDir()
		}
	}
}

func (pi *PiView) Render2D() {
	pi.ToolBar().UpdateActions()
	if win := pi.ParentWindow(); win != nil {
		if !win.IsResizing() {
			win.MainMenuUpdateActives()
		}
	}
	pi.Frame.Render2D()
}

var PiViewProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"#title": ki.Props{
		"max-width":        -1,
		"horizontal-align": gi.AlignCenter,
		"vertical-align":   gi.AlignTop,
	},
	"ToolBar": ki.PropSlice{
		{"Open", ki.Props{
			"label": "Open",
			"icon":  "file-open",
			"desc":  "Open lexer and parser rules from standard JSON-formatted file",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".json",
				}},
			},
		}},
		{"Save", ki.Props{
			"icon": "file-save",
			"desc": "Save lexer and parser rules from file standard JSON-formatted file",
			"updtfunc": giv.ActionUpdateFunc(func(pii interface{}, act *gi.Action) {
				pi := pii.(*PiView)
				act.SetActiveStateUpdt( /* pi.Changed && */ pi.Filename != "")
			}),
		}},
		{"SaveAs", ki.Props{
			"label": "Save As...",
			"icon":  "file-save",
			"desc":  "Save As lexer and parser rules from file standard JSON-formatted file",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".json",
				}},
			},
		}},
		{"sep-file", ki.BlankProp{}},
		{"OpenTest", ki.Props{
			"label": "Open Test",
			"icon":  "file-open",
			"desc":  "Open test file",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "TestFile",
				}},
			},
		}},
		{"SaveTestAs", ki.Props{
			"label": "Save Test As",
			"icon":  "file-save",
			"desc":  "Save current test file as",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "TestFile",
				}},
			},
		}},
		{"sep-ctrl", ki.BlankProp{}},
		{"LexInit", ki.Props{
			"icon": "update",
			"desc": "Init / restart lexer",
		}},
		{"LexNext", ki.Props{
			"icon": "play",
			"desc": "do next step of lexing",
		}},
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"Open", ki.Props{
				"shortcut": gi.KeyFunMenuOpen,
				"desc":     "Open lexer and parser rules from standard JSON-formatted file",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Filename",
						"ext":           ".json",
					}},
				},
			}},
			{"Save", ki.Props{
				"shortcut": gi.KeyFunMenuSave,
				"desc":     "Save lexer and parser rules from file standard JSON-formatted file",
				"updtfunc": giv.ActionUpdateFunc(func(pii interface{}, act *gi.Action) {
					pi := pii.(*PiView)
					act.SetActiveState( /* pi.Changed && */ pi.Filename != "")
				}),
			}},
			{"SaveAs", ki.Props{
				"shortcut": gi.KeyFunMenuSaveAs,
				"label":    "Save As...",
				"desc":     "Save As lexer and parser rules from file standard JSON-formatted file",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Filename",
						"ext":           ".json",
					}},
				},
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project window

// NewPiView creates a new PiView window
func NewPiView() (*gi.Window, *PiView) {
	winm := "Pie Interactive Parser Editor"

	width := 1280
	height := 720

	win := gi.NewWindow2D(winm, winm, width, height, true) // true = pixel sizes

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()

	pi := mfr.AddNewChild(KiT_PiView, "piview").(*PiView)
	pi.Viewport = vp

	mmen := win.MainMenu
	giv.MainMenuView(pi, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !inClosePrompt {
			inClosePrompt = true
			if pi.Changed {
				gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
					Prompt: "Do you want to save your changes?  If so, Cancel and then Save"},
					[]string{"Close Without Saving", "Cancel"},
					win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
						switch sig {
						case 0:
							w.Close()
						case 1:
							// default is to do nothing, i.e., cancel
						}
					})
			} else {
				w.Close()
			}
		}
	})

	inQuitPrompt := false
	oswin.TheApp.SetQuitReqFunc(func() {
		if !inQuitPrompt {
			inQuitPrompt = true
			gi.PromptDialog(vp, gi.DlgOpts{Title: "Really Quit?",
				Prompt: "Are you <i>sure</i> you want to quit?"}, true, true,
				win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
					if sig == int64(gi.DialogAccepted) {
						oswin.TheApp.Quit()
					} else {
						inQuitPrompt = false
					}
				})
		}
	})

	// win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
	// 	fmt.Printf("Doing final Close cleanup here..\n")
	// })

	win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
		if gi.MainWindows.Len() <= 1 {
			go oswin.TheApp.Quit() // once main window is closed, quit
		}
	})

	win.MainMenuUpdated()

	pi.InitView()

	vp.UpdateEndNoSig(updt)

	win.GoStartEventLoop()
	return win, pi
}
