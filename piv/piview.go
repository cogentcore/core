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
	"github.com/goki/pi/lex"
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
func (pv *PiView) InitView() {
	pv.Parser.Init()
	mods, updt := pv.StdConfig()
	if !mods {
		updt = pv.UpdateStart()
	}
	pv.ConfigSplitView()
	pv.ConfigToolbar()
	pv.UpdateEnd(updt)
}

// Save saves lexer and parser rules to current filename, in a standard JSON-formatted file
func (pv *PiView) Save() {
	if pv.Filename == "" {
		return
	}
	pv.Parser.SaveJSON(string(pv.Filename))
	pv.Changed = false
	pv.UpdateSig() // notify our editor
}

// SaveAs saves lexer and parser rules to current filename, in a standard JSON-formatted file
func (pv *PiView) SaveAs(filename gi.FileName) {
	pv.Parser.SaveJSON(string(filename))
	pv.Changed = false
	pv.Filename = filename
	pv.UpdateSig() // notify our editor
}

// Open opens lexer and parser rules to current filename, in a standard JSON-formatted file
func (pv *PiView) Open(filename gi.FileName) {
	pv.Parser.OpenJSON(string(filename))
	pv.Filename = filename
	pv.InitView()
}

// OpenTest opens test file
func (pv *PiView) OpenTest(filename gi.FileName) {
	pv.Buf.OpenFile(filename)
	pv.TestFile = filename
}

// SaveTestAs saves the test file as..
func (pv *PiView) SaveTestAs(filename gi.FileName) {
	pv.Buf.EditDone()
	pv.Buf.SaveFile(filename)
	pv.TestFile = filename
}

// LexInit initializes / restarts lexing process for current test file
func (pv *PiView) LexInit() {
	pv.Parser.SetSrc(pv.Buf.Lines, string(pv.Buf.Filename))
}

// LexStopped tells the user why the lexer stopped
func (pv *PiView) LexStopped() {
	if pv.Parser.LexAtEnd() {
		gi.PromptDialog(pv.Viewport, gi.DlgOpts{Title: "Lex At End",
			Prompt: "The Lexer is now at the end of available text"}, true, false, nil, nil)
	} else {
		gi.PromptDialog(pv.Viewport, gi.DlgOpts{Title: "Lex Error",
			Prompt: "The Lexer has stopped due to errors\n" + pv.Parser.LexState.Errs.AllString()}, true, false, nil, nil)
	}
}

// LexNext does next step of lexing
func (pv *PiView) LexNext() *lex.Rule {
	mrule := pv.Parser.LexNext()
	if mrule == nil {
		pv.LexStopped()
	} else {
		pv.LexLine().SetText(mrule.Nm + ": " + pv.Parser.LexLineOut())
		pv.SelectLexRule(mrule)
	}
	return mrule
}

// LexAll does all remaining lexing until end or error
func (pv *PiView) LexAll() {
	ntok := 0
	for {
		mrule := pv.Parser.LexNext()
		if mrule == nil {
			pv.LexStopped()
			break
		}
		nntok := len(pv.Parser.LexState.Lex)
		if nntok != ntok {
			pv.LexLine().SetText(mrule.Nm + ": " + pv.Parser.LexLineOut())
			pv.SelectLexRule(mrule)
			ntok = nntok
		}
	}
}

func (pv *PiView) SelectLexRule(rule *lex.Rule) {
	lt := pv.LexTree()
	lt.UnselectAll()
	lt.FuncDownMeFirst(0, lt.This(), func(k ki.Ki, level int, d interface{}) bool {
		lnt := k.Embed(giv.KiT_TreeView)
		if lnt == nil {
			return true
		}
		ln := lnt.(*giv.TreeView)
		if ln.SrcNode.Ptr == rule.This() {
			ln.Select()
			return false
		}
		return true
	})
}

//////////////////////////////////////////////////////////////////////////////////////
//   GUI configs

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (pv *PiView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_Label, "lex-line")
	config.Add(gi.KiT_Label, "parse-line")
	config.Add(gi.KiT_SplitView, "splitview")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (pv *PiView) StdConfig() (mods, updt bool) {
	pv.Lay = gi.LayoutVert
	pv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := pv.StdFrameConfig()
	mods, updt = pv.ConfigChildren(config, false)
	if mods {
		ll := pv.LexLine()
		ll.SetStretchMaxWidth()
		ll.Redrawable = true
		pl := pv.ParseLine()
		pl.SetStretchMaxWidth()
		pl.Redrawable = true
	}
	return
}

// LexLine returns the lex line label
func (pv *PiView) LexLine() *gi.Label {
	idx, ok := pv.Children().IndexByName("lex-line", 2)
	if !ok {
		return nil
	}
	return pv.KnownChild(idx).(*gi.Label)
}

// ParseLine returns the parse line label
func (pv *PiView) ParseLine() *gi.Label {
	idx, ok := pv.Children().IndexByName("parse-line", 3)
	if !ok {
		return nil
	}
	return pv.KnownChild(idx).(*gi.Label)
}

// SplitView returns the main SplitView
func (pv *PiView) SplitView() (*gi.SplitView, int) {
	idx, ok := pv.Children().IndexByName("splitview", 4)
	if !ok {
		return nil, -1
	}
	return pv.KnownChild(idx).(*gi.SplitView), idx
}

// LexTree returns the lex rules tree view
func (pv *PiView) LexTree() *giv.TreeView {
	split, _ := pv.SplitView()
	if split != nil {
		tv := split.KnownChild(LexRulesIdx).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// ParseTree returns the parse rules tree view
func (pv *PiView) ParseTree() *giv.TreeView {
	split, _ := pv.SplitView()
	if split != nil {
		tv := split.KnownChild(ParseRulesIdx).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// AstTree returns the Ast output tree view
func (pv *PiView) AstTree() *giv.TreeView {
	split, _ := pv.SplitView()
	if split != nil {
		tv := split.KnownChild(AstOutIdx).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// StructView returns the StructView for editing rules
func (pv *PiView) StructView() *giv.StructView {
	split, _ := pv.SplitView()
	if split != nil {
		return split.KnownChild(StructViewIdx).(*giv.StructView)
	}
	return nil
}

// ToolBar returns the toolbar widget
func (pv *PiView) ToolBar() *gi.ToolBar {
	idx, ok := pv.Children().IndexByName("toolbar", 0)
	if !ok {
		return nil
	}
	return pv.KnownChild(idx).(*gi.ToolBar)
}

// ConfigToolbar adds a PiView toolbar.
func (pv *PiView) ConfigToolbar() {
	tb := pv.ToolBar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	giv.ToolBarView(pv, pv.Viewport, tb)
}

// SplitViewConfig returns a TypeAndNameList for configuring the SplitView
func (pv *PiView) SplitViewConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Frame, "lex-tree-fr")
	config.Add(gi.KiT_Frame, "parse-tree-fr")
	config.Add(giv.KiT_StructView, "struct-view")
	config.Add(gi.KiT_Frame, "ast-tree-fr")
	config.Add(gi.KiT_Layout, "textview-lay")
	return config
}

// ConfigSplitView configures the SplitView.
func (pv *PiView) ConfigSplitView() {
	split, _ := pv.SplitView()
	if split == nil {
		return
	}
	split.Dim = gi.X

	split.SetProp("white-space", gi.WhiteSpacePreWrap)
	split.SetProp("tab-size", 4)

	config := pv.SplitViewConfig()
	mods, updt := split.ConfigChildren(config, true)
	if mods {
		lxfr := split.KnownChild(LexRulesIdx).(*gi.Frame)
		lxt := lxfr.AddNewChild(giv.KiT_TreeView, "lex-tree").(*giv.TreeView)
		lxt.SetRootNode(&pv.Parser.Lexer)

		prfr := split.KnownChild(ParseRulesIdx).(*gi.Frame)
		prt := prfr.AddNewChild(giv.KiT_TreeView, "parse-tree").(*giv.TreeView)
		prt.SetRootNode(&pv.Parser.Parser)

		astfr := split.KnownChild(AstOutIdx).(*gi.Frame)
		astt := astfr.AddNewChild(giv.KiT_TreeView, "ast-tree").(*giv.TreeView)
		astt.SetRootNode(&pv.Parser.Ast)

		txly := split.KnownChild(TextViewIdx).(*gi.Layout)
		txly.SetStretchMaxWidth()
		txly.SetStretchMaxHeight()
		txly.SetMinPrefWidth(units.NewValue(20, units.Ch))
		txly.SetMinPrefHeight(units.NewValue(10, units.Ch))

		txed := txly.AddNewChild(giv.KiT_TextView, "textview").(*giv.TextView)
		txed.Viewport = pv.Viewport
		txed.SetBuf(&pv.Buf)
		pv.Buf.SetHiStyle("emacs")
		pv.Buf.Opts.LineNos = true
		pv.Buf.Opts.TabSize = 4
		txed.SetProp("white-space", gi.WhiteSpacePreWrap)
		txed.SetProp("tab-size", 4)
		txed.SetProp("font-family", "Go Mono")

		split.SetSplits(.15, .15, .25, .15, .3)
		split.UpdateEnd(updt)
	} else {
		pv.LexTree().SetRootNode(&pv.Parser.Lexer)
		pv.LexTree().Open()
		pv.ParseTree().SetRootNode(&pv.Parser.Parser)
		pv.ParseTree().Open()
		pv.AstTree().SetRootNode(&pv.Parser.Ast)
		pv.AstTree().Open()
		pv.StructView().SetStruct(&pv.Parser.Lexer, nil)
	}

	pv.LexTree().TreeViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
		if data == nil {
			return
		}
		tvn, _ := data.(ki.Ki).Embed(giv.KiT_TreeView).(*giv.TreeView)
		pvb, _ := recv.Embed(KiT_PiView).(*PiView)
		switch sig {
		case int64(giv.TreeViewSelected):
			pvb.ViewNode(tvn)
		case int64(giv.TreeViewChanged):
			pvb.SetChanged()
		}
	})

}

// ViewNode sets the StructView view to src node for given treeview
func (pv *PiView) ViewNode(tv *giv.TreeView) {
	sv := pv.StructView()
	if sv != nil {
		sv.SetStruct(tv.SrcNode.Ptr, nil)
	}
}

func (pv *PiView) SetChanged() {
	pv.Changed = true
	pv.ToolBar().UpdateActions() // nil safe
}

func (pv *PiView) FileNodeOpened(fn *giv.FileNode, tvn *giv.FileTreeView) {
	if fn.IsDir() {
		if !fn.IsOpen() {
			tvn.SetOpen()
			fn.OpenDir()
		}
	}
}

func (pv *PiView) FileNodeClosed(fn *giv.FileNode, tvn *giv.FileTreeView) {
	if fn.IsDir() {
		if fn.IsOpen() {
			fn.CloseDir()
		}
	}
}

func (pv *PiView) Render2D() {
	pv.ToolBar().UpdateActions()
	if win := pv.ParentWindow(); win != nil {
		if !win.IsResizing() {
			win.MainMenuUpdateActives()
		}
	}
	pv.Frame.Render2D()
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
					"ext":           ".pi",
				}},
			},
		}},
		{"Save", ki.Props{
			"icon": "file-save",
			"desc": "Save lexer and parser rules from file standard JSON-formatted file",
			"updtfunc": giv.ActionUpdateFunc(func(pvi interface{}, act *gi.Action) {
				pv := pvi.(*PiView)
				act.SetActiveStateUpdt( /* pv.Changed && */ pv.Filename != "")
			}),
		}},
		{"SaveAs", ki.Props{
			"label": "Save As...",
			"icon":  "file-save",
			"desc":  "Save As lexer and parser rules from file standard JSON-formatted file",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".pi",
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
		{"LexAll", ki.Props{
			"icon": "fast-fwd",
			"desc": "do all remaining lexing",
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
						"ext":           ".pi",
					}},
				},
			}},
			{"Save", ki.Props{
				"shortcut": gi.KeyFunMenuSave,
				"desc":     "Save lexer and parser rules from file standard JSON-formatted file",
				"updtfunc": giv.ActionUpdateFunc(func(pvi interface{}, act *gi.Action) {
					pv := pvi.(*PiView)
					act.SetActiveState( /* pv.Changed && */ pv.Filename != "")
				}),
			}},
			{"SaveAs", ki.Props{
				"shortcut": gi.KeyFunMenuSaveAs,
				"label":    "Save As...",
				"desc":     "Save As lexer and parser rules from file standard JSON-formatted file",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Filename",
						"ext":           ".pi",
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

	pv := mfr.AddNewChild(KiT_PiView, "piview").(*PiView)
	pv.Viewport = vp

	mmen := win.MainMenu
	giv.MainMenuView(pv, win, mmen)

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !inClosePrompt {
			inClosePrompt = true
			if pv.Changed {
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

	pv.InitView()

	vp.UpdateEndNoSig(updt)

	win.GoStartEventLoop()
	return win, pv
}
