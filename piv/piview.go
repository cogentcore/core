// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package piv provides the PiView object for the full GUI view of the
// interactive parser (pi) system.
package piv

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/gide/gide"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/pi"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/parse"
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
	MainTabsIdx
)

// PiView provides the interactive GUI view for constructing and testing the
// lexer and parser
type PiView struct {
	gi.Frame
	Parser   pi.Parser   `desc:"the parser we are viewing"`
	Prefs    ProjPrefs   `desc:"project preferences -- this IS the project file"`
	Changed  bool        `json:"-" desc:"has the root changed?  we receive update signals from root for changes"`
	TestBuf  giv.TextBuf `json:"-" desc:"test file buffer"`
	LexBuf   giv.TextBuf `json:"-" desc:"buffer of lexified tokens"`
	ParseBuf giv.TextBuf `json:"-" desc:"buffer of parse info"`
	KeySeq1  key.Chord   `desc:"first key in sequence if needs2 key pressed"`
}

var KiT_PiView = kit.Types.AddType(&PiView{}, PiViewProps)

// InitView initializes the viewer / editor
func (pv *PiView) InitView() {
	parse.GuiActive = true
	pv.Parser.Init()
	mods, updt := pv.StdConfig()
	if !mods {
		updt = pv.UpdateStart()
	}
	pv.ConfigSplitView()
	pv.ConfigStatusBar()
	pv.ConfigToolbar()
	pv.UpdateEnd(updt)
}

// IsEmpty returns true if current project is empty
func (pv *PiView) IsEmpty() bool {
	return (!pv.Parser.Lexer.HasChildren() && !pv.Parser.Parser.HasChildren())
}

// OpenRecent opens a recently-used project
func (pv *PiView) OpenRecent(filename gi.FileName) {
	pv.OpenProj(filename)
}

// OpenProj opens lexer and parser rules to current filename, in a standard JSON-formatted file
// if current is not empty, opens in a new window
func (pv *PiView) OpenProj(filename gi.FileName) *PiView {
	if !pv.IsEmpty() {
		_, nprj := NewPiView()
		nprj.OpenProj(filename)
		return nprj
	}
	pv.Prefs.OpenJSON(filename)
	pv.InitView()
	pv.ApplyPrefs()
	SavedPaths.AddPath(string(filename), gi.Prefs.SavedPathsMax)
	SavePaths()
	return pv
}

// NewProj makes a new project in a new window
func (pv *PiView) NewProj() (*gi.Window, *PiView) {
	return NewPiView()
}

// SaveProj saves project prefs to current filename, in a standard JSON-formatted file
// also saves the current parser
func (pv *PiView) SaveProj() {
	if pv.Prefs.ProjFile == "" {
		return
	}
	pv.SaveParser()
	pv.GetPrefs()
	pv.Prefs.SaveJSON(pv.Prefs.ProjFile)
	pv.Changed = false
	pv.SetStatus(fmt.Sprintf("Project Saved to: %v", pv.Prefs.ProjFile))
	pv.UpdateSig() // notify our editor
}

// SaveProjAs saves lexer and parser rules to current filename, in a standard JSON-formatted file
// also saves the current parser
func (pv *PiView) SaveProjAs(filename gi.FileName) {
	SavedPaths.AddPath(string(filename), gi.Prefs.SavedPathsMax)
	SavePaths()
	pv.SaveParser()
	pv.GetPrefs()
	pv.Prefs.SaveJSON(filename)
	pv.Changed = false
	pv.SetStatus(fmt.Sprintf("Project Saved to: %v", pv.Prefs.ProjFile))
	pv.UpdateSig() // notify our editor
}

// ApplyPrefs applies project-level prefs (e.g., after opening)
func (pv *PiView) ApplyPrefs() {
	parse.Trace = pv.Prefs.TraceOpts
	if pv.Prefs.ParserFile != "" {
		pv.OpenParser(pv.Prefs.ParserFile)
	}
	if pv.Prefs.TestFile != "" {
		pv.OpenTest(pv.Prefs.TestFile)
	}
}

// GetPrefs gets the current values of things for prefs
func (pv *PiView) GetPrefs() {
	pv.Prefs.TraceOpts = parse.Trace
}

/////////////////////////////////////////////////////////////////////////
//  other IO

// OpenParser opens lexer and parser rules to current filename, in a standard JSON-formatted file
func (pv *PiView) OpenParser(filename gi.FileName) {
	pv.Parser.OpenJSON(string(filename))
	pv.Prefs.ParserFile = filename
	pv.InitView()
}

// SaveParser saves lexer and parser rules to current filename, in a standard JSON-formatted file
func (pv *PiView) SaveParser() {
	if pv.Prefs.ParserFile == "" {
		return
	}
	pv.Parser.SaveJSON(string(pv.Prefs.ParserFile))

	ext := filepath.Ext(string(pv.Prefs.ParserFile))
	pigfn := strings.TrimSuffix(string(pv.Prefs.ParserFile), ext) + ".pig"
	pv.Parser.SaveGrammar(pigfn)

	pv.Changed = false
	pv.SetStatus(fmt.Sprintf("Parser Saved to: %v", pv.Prefs.ParserFile))
	pv.UpdateSig() // notify our editor
}

// SaveParserAs saves lexer and parser rules to current filename, in a standard JSON-formatted file
func (pv *PiView) SaveParserAs(filename gi.FileName) {
	pv.Parser.SaveJSON(string(filename))

	ext := filepath.Ext(string(pv.Prefs.ParserFile))
	pigfn := strings.TrimSuffix(string(pv.Prefs.ParserFile), ext) + ".pig"
	pv.Parser.SaveGrammar(pigfn)

	pv.Changed = false
	pv.Prefs.ParserFile = filename
	pv.SetStatus(fmt.Sprintf("Parser Saved to: %v", pv.Prefs.ParserFile))
	pv.UpdateSig() // notify our editor
}

// OpenTest opens test file
func (pv *PiView) OpenTest(filename gi.FileName) {
	pv.TestBuf.OpenFile(filename)
	pv.Prefs.TestFile = filename
}

// SaveTestAs saves the test file as..
func (pv *PiView) SaveTestAs(filename gi.FileName) {
	pv.TestBuf.EditDone()
	pv.TestBuf.SaveFile(filename)
	pv.Prefs.TestFile = filename
	pv.SetStatus(fmt.Sprintf("TestFile Saved to: %v", pv.Prefs.TestFile))
}

// SetStatus updates the statusbar label with given message, along with other status info
func (pv *PiView) SetStatus(msg string) {
	sb := pv.StatusBar()
	if sb == nil {
		return
	}
	// pv.UpdtMu.Lock()
	// defer pv.UpdtMu.Unlock()

	updt := sb.UpdateStart()
	lbl := pv.StatusLabel()
	fnm := ""
	ln := 0
	ch := 0
	if tv, ok := pv.TestTextView(); ok {
		ln = tv.CursorPos.Ln + 1
		ch = tv.CursorPos.Ch
		if tv.ISearch.On {
			msg = fmt.Sprintf("\tISearch: %v (n=%v)\t%v", tv.ISearch.Find, len(tv.ISearch.Matches), msg)
		}
		if tv.QReplace.On {
			msg = fmt.Sprintf("\tQReplace: %v -> %v (n=%v)\t%v", tv.QReplace.Find, tv.QReplace.Replace, len(tv.QReplace.Matches), msg)
		}
	}

	str := fmt.Sprintf("%v\t<b>%v:</b>\t(%v,%v)\t%v", pv.Nm, fnm, ln, ch, msg)
	lbl.SetText(str)
	sb.UpdateEnd(updt)
}

////////////////////////////////////////////////////////////////////////////////////////
//  Lexing

// LexInit initializes / restarts lexing process for current test file
func (pv *PiView) LexInit() {
	fs := &pv.TestBuf.PiState
	fs.SetSrc(&pv.TestBuf.Lines, string(pv.TestBuf.Filename))
	// pv.Hi.SetParser(&pv.Parser)
	pv.Parser.LexInit(fs)
	if fs.LexHasErrs() {
		gi.PromptDialog(pv.Viewport, gi.DlgOpts{Title: "Lex Error",
			Prompt: "The Lexer validation has errors\n" + fs.LexErrString()}, true, false, nil, nil)
	}
	pv.UpdtLexBuf()
}

// LexStopped tells the user why the lexer stopped
func (pv *PiView) LexStopped() {
	fs := &pv.TestBuf.PiState
	if fs.LexAtEnd() {
		pv.SetStatus("The Lexer is now at the end of available text")
	} else {
		errs := fs.LexErrString()
		if errs != "" {
			pv.SetStatus("Lexer Errors!")
			gi.PromptDialog(pv.Viewport, gi.DlgOpts{Title: "Lex Error",
				Prompt: "The Lexer has stopped due to errors\n" + errs}, true, false, nil, nil)
		} else {
			pv.SetStatus("Lexer Missing Rules!")
			gi.PromptDialog(pv.Viewport, gi.DlgOpts{Title: "Lex Error",
				Prompt: "The Lexer has stopped because it cannot process the source at this point:<br>\n" + fs.LexNextSrcLine()}, true, false, nil, nil)
		}
	}
}

// LexNext does next step of lexing
func (pv *PiView) LexNext() *lex.Rule {
	fs := &pv.TestBuf.PiState
	mrule := pv.Parser.LexNext(fs)
	if mrule == nil {
		pv.LexStopped()
	} else {
		pv.SetStatus(mrule.Nm + ": " + fs.LexLineString())
		pv.SelectLexRule(mrule)
	}
	pv.UpdtLexBuf()
	return mrule
}

// LexLine does next line of lexing
func (pv *PiView) LexNextLine() *lex.Rule {
	fs := &pv.TestBuf.PiState
	mrule := pv.Parser.LexNextLine(fs)
	if mrule == nil && fs.LexHasErrs() {
		pv.LexStopped()
	} else if mrule != nil {
		pv.SetStatus(mrule.Nm + ": " + fs.LexLineString())
		pv.SelectLexRule(mrule)
	}
	pv.UpdtLexBuf()
	return mrule
}

// LexAll does all remaining lexing until end or error -- if animate is true, then
// it updates the display -- otherwise proceeds silently
func (pv *PiView) LexAll(animate bool) {
	fs := &pv.TestBuf.PiState
	ntok := 0
	for {
		mrule := pv.Parser.LexNext(fs)
		if mrule == nil {
			if !fs.LexAtEnd() {
				pv.LexStopped()
			}
			break
		}
		if animate {
			nntok := len(fs.LexState.Lex)
			if nntok != ntok {
				pv.SetStatus(mrule.Nm + ": " + fs.LexLineString())
				pv.SelectLexRule(mrule)
				ntok = nntok
			}
		}
	}
	pv.UpdtLexBuf()
}

// SelectLexRule selects given lex rule in Lexer
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

// UpdtLexBuf sets the LexBuf to current lex content
func (pv *PiView) UpdtLexBuf() {
	fs := &pv.TestBuf.PiState
	txt := fs.Src.LexTagSrc()
	pv.LexBuf.SetText([]byte(txt))
	pv.TestBuf.HiTags = fs.Src.Lexs
	pv.TestBuf.MarkupFromTags()
	pv.TestBuf.Refresh()
}

////////////////////////////////////////////////////////////////////////////////////////
//  PassTwo

// EditPassTwo shows the PassTwo settings to edit -- does nest depth and finds the EOS end-of-statements
func (pv *PiView) EditPassTwo() {
	sv := pv.StructView()
	if sv != nil {
		sv.SetStruct(&pv.Parser.PassTwo, nil)
	}
}

// PassTwo does the second pass after lexing, per current settings
func (pv *PiView) PassTwo() {
	fs := &pv.TestBuf.PiState
	pv.Parser.DoPassTwo(fs)
	if fs.PassTwoHasErrs() {
		gi.PromptDialog(pv.Viewport, gi.DlgOpts{Title: "PassTwo Error",
			Prompt: "The PassTwo had the following errors\n" + fs.PassTwoErrString()}, true, false, nil, nil)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  Parsing

// EditTrace shows the parse.Trace options for detailed tracing output
func (pv *PiView) EditTrace() {
	sv := pv.StructView()
	if sv != nil {
		sv.SetStruct(&parse.Trace, nil)
	}
}

// ParseInit initializes / restarts lexing process for current test file
func (pv *PiView) ParseInit() {
	fs := &pv.TestBuf.PiState
	gide.TheConsole.Buf.New(0)
	pv.LexInit()
	pv.Parser.LexAll(fs)
	pv.Parser.ParserInit(fs)
	pv.UpdtLexBuf()
	if fs.ParseHasErrs() {
		gi.PromptDialog(pv.Viewport, gi.DlgOpts{Title: "Parse Error",
			Prompt: "The Parser validation has errors\n" + fs.ParseErrString()}, true, false, nil, nil)
	}
}

// ParseStopped tells the user why the lexer stopped
func (pv *PiView) ParseStopped() {
	fs := &pv.TestBuf.PiState
	if fs.ParseAtEnd() {
		pv.SetStatus("The Parser is now at the end of available text")
	} else {
		errs := fs.ParseErrString()
		if errs != "" {
			pv.SetStatus("Parse Error!")
			gi.PromptDialog(pv.Viewport, gi.DlgOpts{Title: "Parse Error",
				Prompt: "The Parser has stopped due to errors<br>\n" + errs}, true, false, nil, nil)
		} else {
			pv.SetStatus("Parse Missing Rules!")
			gi.PromptDialog(pv.Viewport, gi.DlgOpts{Title: "Parse Error",
				Prompt: "The Parser has stopped because it cannot process the source at this point:<br>\n" + fs.ParseNextSrcLine()}, true, false, nil, nil)
		}
	}
}

// ParseNext does next step of lexing
func (pv *PiView) ParseNext() *parse.Rule {
	fs := &pv.TestBuf.PiState
	at := pv.AstTree()
	updt := at.UpdateStart()
	mrule := pv.Parser.ParseNext(fs)
	at.UpdateEnd(updt)
	at.OpenAll()
	pv.AstTreeToEnd()
	pv.UpdtParseBuf()
	if mrule == nil {
		pv.ParseStopped()
	} else {
		// pv.SelectParseRule(mrule) // not that informative
		if fs.ParseHasErrs() { // can have errs even when matching..
			pv.ParseStopped()
		}
	}
	return mrule
}

// ParseAll does all remaining lexing until end or error
func (pv *PiView) ParseAll() {
	fs := &pv.TestBuf.PiState
	at := pv.AstTree()
	updt := at.UpdateStart()
	for {
		mrule := pv.Parser.ParseNext(fs)
		if mrule == nil {
			break
		}
	}
	at.UpdateEnd(updt)
	at.OpenAll()
	pv.AstTreeToEnd()
	pv.UpdtParseBuf()
	if !fs.ParseAtEnd() {
		pv.ParseStopped()
	}
}

// SelectParseRule selects given lex rule in Parser
func (pv *PiView) SelectParseRule(rule *parse.Rule) {
	lt := pv.ParseTree()
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

// AstTreeToEnd
func (pv *PiView) AstTreeToEnd() {
	lt := pv.AstTree()
	lt.MoveEndAction(mouse.SelectOne)
}

// UpdtParseBuf sets the ParseBuf to current parse rule output
func (pv *PiView) UpdtParseBuf() {
	fs := &pv.TestBuf.PiState
	txt := fs.ParseRuleString(parse.Trace.FullStackOut)
	pv.ParseBuf.SetText([]byte(txt))
}

//////////////////////////////////////////////////////////////////////////////////////
//   Panels

// CurPanel returns the splitter panel that currently has keyboard focus
func (pv *PiView) CurPanel() int {
	sv := pv.SplitView()
	if sv == nil {
		return -1
	}
	for i, ski := range sv.Kids {
		_, sk := gi.KiToNode2D(ski)
		if sk.ContainsFocus() {
			return i
		}
	}
	return -1 // nobody
}

// FocusOnPanel moves keyboard focus to given panel -- returns false if nothing at that tab
func (pv *PiView) FocusOnPanel(panel int) bool {
	sv := pv.SplitView()
	if sv == nil {
		return false
	}
	win := pv.ParentWindow()
	ski := sv.Kids[panel]
	win.FocusNext(ski)
	return true
}

// FocusNextPanel moves the keyboard focus to the next panel to the right
func (pv *PiView) FocusNextPanel() {
	sv := pv.SplitView()
	if sv == nil {
		return
	}
	cp := pv.CurPanel()
	cp++
	np := len(sv.Kids)
	if cp >= np {
		cp = 0
	}
	for sv.Splits[cp] <= 0.01 {
		cp++
		if cp >= np {
			cp = 0
		}
	}
	pv.FocusOnPanel(cp)
}

// FocusPrevPanel moves the keyboard focus to the previous panel to the left
func (pv *PiView) FocusPrevPanel() {
	sv := pv.SplitView()
	if sv == nil {
		return
	}
	cp := pv.CurPanel()
	cp--
	np := len(sv.Kids)
	if cp < 0 {
		cp = np - 1
	}
	for sv.Splits[cp] <= 0.01 {
		cp--
		if cp < 0 {
			cp = np - 1
		}
	}
	pv.FocusOnPanel(cp)
}

//////////////////////////////////////////////////////////////////////////////////////
//   Tabs

// MainTabByName returns a MainTabs (first set of tabs) tab with given name,
// and its index -- returns false if not found
func (pv *PiView) MainTabByName(label string) (gi.Node2D, int, bool) {
	tv := pv.MainTabs()
	return tv.TabByName(label)
}

// SelectMainTabByName Selects given main tab, and returns all of its contents as well.
func (pv *PiView) SelectMainTabByName(label string) (gi.Node2D, int, bool) {
	tv := pv.MainTabs()
	widg, idx, ok := pv.MainTabByName(label)
	if ok {
		tv.SelectTabIndex(idx)
	}
	return widg, idx, ok
}

// FindOrMakeMainTab returns a MainTabs (first set of tabs) tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with widget of given type.  if sel, then select it.  returns widget and tab index.
func (pv *PiView) FindOrMakeMainTab(label string, typ reflect.Type, sel bool) (gi.Node2D, int) {
	tv := pv.MainTabs()
	widg, idx, ok := pv.MainTabByName(label)
	if ok {
		if sel {
			tv.SelectTabIndex(idx)
		}
		return widg, idx
	}
	widg, idx = tv.AddNewTab(typ, label)
	if sel {
		tv.SelectTabIndex(idx)
	}
	return widg, idx
}

// ConfigTextView configures text view
func (pv *PiView) ConfigTextView(ly *gi.Layout, out bool) *giv.TextView {
	ly.Lay = gi.LayoutVert
	ly.SetStretchMaxWidth()
	ly.SetStretchMaxHeight()
	ly.SetMinPrefWidth(units.NewValue(20, units.Ch))
	ly.SetMinPrefHeight(units.NewValue(10, units.Ch))
	var tv *giv.TextView
	if ly.HasChildren() {
		tv = ly.KnownChild(0).Embed(giv.KiT_TextView).(*giv.TextView)
	} else {
		tv = ly.AddNewChild(giv.KiT_TextView, ly.Nm).(*giv.TextView)
	}

	if gide.Prefs.Editor.WordWrap {
		tv.SetProp("white-space", gi.WhiteSpacePreWrap)
	} else {
		tv.SetProp("white-space", gi.WhiteSpacePre)
	}
	tv.SetProp("tab-size", 4)
	tv.SetProp("font-family", gide.Prefs.FontFamily)
	if out {
		tv.SetInactive()
	}
	return tv
}

// FindOrMakeMainTabTextView returns a MainTabs (first set of tabs) tab with given
// name, first by looking for an existing one, and if not found, making a new
// one with a Layout and then a TextView in it.  if sel, then select it.
// returns widget and tab index.
func (pv *PiView) FindOrMakeMainTabTextView(label string, sel bool, out bool) (*giv.TextView, int) {
	lyk, idx := pv.FindOrMakeMainTab(label, gi.KiT_Layout, sel)
	ly := lyk.Embed(gi.KiT_Layout).(*gi.Layout)
	tv := pv.ConfigTextView(ly, out)
	return tv, idx
}

// MainTabTextViewByName returns the textview for given main tab, if it exists
func (pv *PiView) MainTabTextViewByName(tabnm string) (*giv.TextView, bool) {
	lyk, _, got := pv.MainTabByName(tabnm)
	if !got {
		return nil, false
	}
	ctv := lyk.KnownChild(0).Embed(giv.KiT_TextView).(*giv.TextView)
	return ctv, true
}

// TextTextView returns the textview for TestBuf TextView
func (pv *PiView) TestTextView() (*giv.TextView, bool) {
	return pv.MainTabTextViewByName("TestText")
}

// OpenConsoleTab opens a main tab displaying console output (stdout, stderr)
func (pv *PiView) OpenConsoleTab() {
	ctv, _ := pv.FindOrMakeMainTabTextView("Console", true, true)
	ctv.SetInactive()
	ctv.SetProp("white-space", gi.WhiteSpacePre) // no word wrap
	if ctv.Buf == nil || ctv.Buf != gide.TheConsole.Buf {
		ctv.SetBuf(gide.TheConsole.Buf)
		gide.TheConsole.Buf.TextBufSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			pve, _ := recv.Embed(KiT_PiView).(*PiView)
			pve.SelectMainTabByName("Console")
		})
	}
}

// OpenTestTextTab opens a main tab displaying test text
func (pv *PiView) OpenTestTextTab() {
	ctv, _ := pv.FindOrMakeMainTabTextView("TestText", true, false)
	if ctv.Buf == nil || ctv.Buf != &pv.TestBuf {
		ctv.SetBuf(&pv.TestBuf)
	}
}

// OpenLexTab opens a main tab displaying lexer output
func (pv *PiView) OpenLexTab() {
	ctv, _ := pv.FindOrMakeMainTabTextView("LexOut", true, true)
	if ctv.Buf == nil || ctv.Buf != &pv.LexBuf {
		ctv.SetBuf(&pv.LexBuf)
	}
}

// OpenParseTab opens a main tab displaying parser output
func (pv *PiView) OpenParseTab() {
	ctv, _ := pv.FindOrMakeMainTabTextView("ParseOut", true, true)
	if ctv.Buf == nil || ctv.Buf != &pv.ParseBuf {
		ctv.SetBuf(&pv.ParseBuf)
	}
}

//////////////////////////////////////////////////////////////////////////////////////
//   GUI configs

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (pv *PiView) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_SplitView, "splitview")
	config.Add(gi.KiT_Frame, "statusbar")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (pv *PiView) StdConfig() (mods, updt bool) {
	pv.Lay = gi.LayoutVert
	pv.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := pv.StdFrameConfig()
	mods, updt = pv.ConfigChildren(config, false)
	return
}

// SplitView returns the main SplitView
func (pv *PiView) SplitView() *gi.SplitView {
	idx, ok := pv.Children().IndexByName("splitview", 4)
	if !ok {
		return nil
	}
	return pv.KnownChild(idx).(*gi.SplitView)
}

// LexTree returns the lex rules tree view
func (pv *PiView) LexTree() *giv.TreeView {
	split := pv.SplitView()
	if split != nil {
		tv := split.KnownChild(LexRulesIdx).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// ParseTree returns the parse rules tree view
func (pv *PiView) ParseTree() *giv.TreeView {
	split := pv.SplitView()
	if split != nil {
		tv := split.KnownChild(ParseRulesIdx).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// AstTree returns the Ast output tree view
func (pv *PiView) AstTree() *giv.TreeView {
	split := pv.SplitView()
	if split != nil {
		tv := split.KnownChild(AstOutIdx).KnownChild(0).(*giv.TreeView)
		return tv
	}
	return nil
}

// StructView returns the StructView for editing rules
func (pv *PiView) StructView() *giv.StructView {
	split := pv.SplitView()
	if split != nil {
		return split.KnownChild(StructViewIdx).(*giv.StructView)
	}
	return nil
}

// MainTabs returns the main TabView
func (pv *PiView) MainTabs() *gi.TabView {
	split := pv.SplitView()
	if split != nil {
		tv := split.KnownChild(MainTabsIdx).Embed(gi.KiT_TabView).(*gi.TabView)
		return tv
	}
	return nil
}

// StatusBar returns the statusbar widget
func (pv *PiView) StatusBar() *gi.Frame {
	tbi, ok := pv.ChildByName("statusbar", 2)
	if !ok {
		return nil
	}
	return tbi.(*gi.Frame)
}

// StatusLabel returns the statusbar label widget
func (pv *PiView) StatusLabel() *gi.Label {
	sb := pv.StatusBar()
	if sb != nil {
		return sb.KnownChild(0).Embed(gi.KiT_Label).(*gi.Label)
	}
	return nil
}

// ConfigStatusBar configures statusbar with label
func (pv *PiView) ConfigStatusBar() {
	sb := pv.StatusBar()
	if sb == nil || sb.HasChildren() {
		return
	}
	sb.SetStretchMaxWidth()
	sb.SetMinPrefHeight(units.NewValue(1.2, units.Em))
	sb.SetProp("overflow", "hidden") // no scrollbars!
	sb.SetProp("margin", 0)
	sb.SetProp("padding", 0)
	lbl := sb.AddNewChild(gi.KiT_Label, "sb-lbl").(*gi.Label)
	lbl.SetStretchMaxWidth()
	lbl.SetMinPrefHeight(units.NewValue(1, units.Em))
	lbl.SetProp("vertical-align", gi.AlignTop)
	lbl.SetProp("margin", 0)
	lbl.SetProp("padding", 0)
	lbl.SetProp("tab-size", 4)
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
	config.Add(gi.KiT_TabView, "main-tabs")
	return config
}

// ConfigSplitView configures the SplitView.
func (pv *PiView) ConfigSplitView() {
	fs := &pv.TestBuf.PiState
	split := pv.SplitView()
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
		astt.SetRootNode(&fs.Ast)

		pv.TestBuf.SetHiStyle(gide.Prefs.HiStyle)
		gide.Prefs.Editor.ConfigTextBuf(&pv.TestBuf.Opts)

		pv.LexBuf.SetHiStyle(gide.Prefs.HiStyle)
		gide.Prefs.Editor.ConfigTextBuf(&pv.LexBuf.Opts)

		pv.ParseBuf.SetHiStyle(gide.Prefs.HiStyle)
		gide.Prefs.Editor.ConfigTextBuf(&pv.ParseBuf.Opts)

		split.SetSplits(.15, .15, .2, .15, .35)
		split.UpdateEnd(updt)

		pv.OpenConsoleTab()
		pv.OpenTestTextTab()
		pv.OpenLexTab()
		pv.OpenParseTab()

	} else {
		pv.LexTree().SetRootNode(&pv.Parser.Lexer)
		pv.LexTree().Open()
		pv.ParseTree().SetRootNode(&pv.Parser.Parser)
		pv.ParseTree().Open()
		pv.AstTree().SetRootNode(&fs.Ast)
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

	pv.ParseTree().TreeViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
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

	pv.AstTree().TreeViewSig.Connect(pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
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

func (ge *PiView) PiViewKeys(kt *key.ChordEvent) {
	var kf gide.KeyFuns
	kc := kt.Chord()
	if gi.KeyEventTrace {
		fmt.Printf("PiView KeyInput: %v\n", ge.PathUnique())
	}
	// gkf := gi.KeyFun(kc)
	if ge.KeySeq1 != "" {
		kf = gide.KeyFun(ge.KeySeq1, kc)
		seqstr := string(ge.KeySeq1) + " " + string(kc)
		if kf == gide.KeyFunNil || kc == "Escape" {
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun sequence: %v aborted\n", seqstr)
			}
			ge.SetStatus(seqstr + " -- aborted")
			kt.SetProcessed() // abort key sequence, don't send esc to anyone else
			ge.KeySeq1 = ""
			return
		}
		ge.SetStatus(seqstr)
		ge.KeySeq1 = ""
		// gkf = gi.KeyFunNil // override!
	} else {
		kf = gide.KeyFun(kc, "")
		if kf == gide.KeyFunNeeds2 {
			kt.SetProcessed()
			ge.KeySeq1 = kt.Chord()
			ge.SetStatus(string(ge.KeySeq1))
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun sequence needs 2 after: %v\n", ge.KeySeq1)
			}
			return
		} else if kf != gide.KeyFunNil {
			if gi.KeyEventTrace {
				fmt.Printf("gide.KeyFun got in one: %v = %v\n", ge.KeySeq1, kf)
			}
			// gkf = gi.KeyFunNil // override!
		}
	}

	// switch gkf {
	// case gi.KeyFunFind:
	// 	kt.SetProcessed()
	// 	tv := ge.ActiveTextView()
	// 	if tv.HasSelection() {
	// 		ge.Prefs.Find.Find = string(tv.Selection().ToBytes())
	// 	}
	// 	giv.CallMethod(ge, "Find", ge.Viewport)
	// }
	// if kt.IsProcessed() {
	// 	return
	// }
	switch kf {
	case gide.KeyFunNextPanel:
		kt.SetProcessed()
		ge.FocusNextPanel()
	case gide.KeyFunPrevPanel:
		kt.SetProcessed()
		ge.FocusPrevPanel()
	case gide.KeyFunFileOpen:
		kt.SetProcessed()
		giv.CallMethod(ge, "OpenTest", ge.Viewport)
	// case gide.KeyFunBufSelect:
	// 	kt.SetProcessed()
	// 	ge.SelectOpenNode()
	// case gide.KeyFunBufClone:
	// 	kt.SetProcessed()
	// 	ge.CloneActiveView()
	case gide.KeyFunBufSave:
		kt.SetProcessed()
		giv.CallMethod(ge, "SaveTestAs", ge.Viewport)
	case gide.KeyFunBufSaveAs:
		kt.SetProcessed()
		giv.CallMethod(ge, "SaveActiveViewAs", ge.Viewport)
		// case gide.KeyFunBufClose:
		// 	kt.SetProcessed()
		// 	ge.CloseActiveView()
		// case gide.KeyFunExecCmd:
		// 	kt.SetProcessed()
		// 	giv.CallMethod(ge, "ExecCmd", ge.Viewport)
		// case gide.KeyFunCommentOut:
		// 	kt.SetProcessed()
		// 	ge.CommentOut()
		// case gide.KeyFunIndent:
		// 	kt.SetProcessed()
		// 	ge.Indent()
		// case gide.KeyFunSetSplit:
		// 	kt.SetProcessed()
		// 	giv.CallMethod(ge, "SplitsSetView", ge.Viewport)
		// case gide.KeyFunBuildProj:
		// 	kt.SetProcessed()
		// 	ge.Build()
		// case gide.KeyFunRunProj:
		// 	kt.SetProcessed()
		// 	ge.Run()
	}
}

func (ge *PiView) KeyChordEvent() {
	// need hipri to prevent 2-seq guys from being captured by others
	ge.ConnectEvent(oswin.KeyChordEvent, gi.HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		gee := recv.Embed(KiT_PiView).(*PiView)
		kt := d.(*key.ChordEvent)
		gee.PiViewKeys(kt)
	})
}

func (ge *PiView) ConnectEvents2D() {
	if ge.HasAnyScroll() {
		ge.LayoutScrollEvents()
	}
	ge.KeyChordEvent()
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
		{"SaveProj", ki.Props{
			"shortcut": gi.KeyFunMenuSave,
			"label":    "Save Project",
			"desc":     "Save GoPi project file to standard JSON-formatted file",
			"updtfunc": giv.ActionUpdateFunc(func(pvi interface{}, act *gi.Action) {
				pv := pvi.(*PiView)
				act.SetActiveState( /* pv.Changed && */ pv.Prefs.ProjFile != "")
			}),
		}},
		{"sep-parse", ki.BlankProp{}},
		{"OpenParser", ki.Props{
			"label": "Open Parser...",
			"icon":  "file-open",
			"desc":  "Open lexer and parser rules from standard JSON-formatted file",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Prefs.ParserFile",
					"ext":           ".pi",
				}},
			},
		}},
		{"SaveParser", ki.Props{
			"icon": "file-save",
			"desc": "Save lexer and parser rules from file standard JSON-formatted file",
			"updtfunc": giv.ActionUpdateFunc(func(pvi interface{}, act *gi.Action) {
				pv := pvi.(*PiView)
				act.SetActiveStateUpdt( /* pv.Changed && */ pv.Prefs.ParserFile != "")
			}),
		}},
		{"SaveParserAs", ki.Props{
			"label": "Save Parser As...",
			"icon":  "file-save",
			"desc":  "Save As lexer and parser rules from file standard JSON-formatted file",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Prefs.ParserFile",
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
					"default-field": "Prefs.TestFile",
				}},
			},
		}},
		{"SaveTestAs", ki.Props{
			"label": "Save Test As",
			"icon":  "file-save",
			"desc":  "Save current test file as",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Prefs.TestFile",
				}},
			},
		}},
		{"sep-lex", ki.BlankProp{}},
		{"LexInit", ki.Props{
			"icon": "update",
			"desc": "Init / restart lexer",
		}},
		{"LexNext", ki.Props{
			"icon": "play",
			"desc": "do next single step of lexing",
		}},
		{"LexNextLine", ki.Props{
			"icon": "play",
			"desc": "do next line of lexing",
		}},
		{"LexAll", ki.Props{
			"icon": "fast-fwd",
			"desc": "do all remaining lexing",
			"Args": ki.PropSlice{
				{"Animate", ki.Props{}},
			},
		}},
		{"sep-passtwo", ki.BlankProp{}},
		{"EditPassTwo", ki.Props{
			"icon": "edit",
			"desc": "edit the settings of the PassTwo -- second pass after lexing",
		}},
		{"PassTwo", ki.Props{
			"icon": "play",
			"desc": "perform second pass after lexing -- computes nesting depth globally and finds EOS tokens",
		}},
		{"sep-parse", ki.BlankProp{}},
		{"EditTrace", ki.Props{
			"icon": "edit",
			"desc": "edit the parse tracing options for seeing how the parsing process is working",
		}},
		{"ParseInit", ki.Props{
			"icon": "update",
			"desc": "initialize parser -- this also performs lexing, PassTwo, assuming that is all working",
		}},
		{"ParseNext", ki.Props{
			"icon": "play",
			"desc": "do next step of parsing",
		}},
		{"ParseAll", ki.Props{
			"icon": "fast-fwd",
			"desc": "do remaining parsing",
		}},
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"OpenRecent", ki.Props{
				"submenu": &SavedPaths,
				"Args": ki.PropSlice{
					{"File Name", ki.Props{}},
				},
			}},
			{"OpenProj", ki.Props{
				"shortcut": gi.KeyFunMenuOpen,
				"label":    "Open Project...",
				"desc":     "open a GoPi project that has full settings",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Prefs.ProjFile",
						"ext":           ".pip",
					}},
				},
			}},
			{"NewProj", ki.Props{
				"shortcut": gi.KeyFunMenuNew,
				"label":    "New Project...",
				"desc":     "create a new project",
			}},
			{"SaveProj", ki.Props{
				"shortcut": gi.KeyFunMenuSave,
				"label":    "Save Project",
				"desc":     "Save GoPi project file to standard JSON-formatted file",
				"updtfunc": giv.ActionUpdateFunc(func(pvi interface{}, act *gi.Action) {
					pv := pvi.(*PiView)
					act.SetActiveState( /* pv.Changed && */ pv.Prefs.ProjFile != "")
				}),
			}},
			{"SaveProjAs", ki.Props{
				"shortcut": gi.KeyFunMenuSaveAs,
				"label":    "Save Project As...",
				"desc":     "Save GoPi project to file standard JSON-formatted file",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Prefs.ProjFile",
						"ext":           ".pip",
					}},
				},
			}},
			{"sep-parse", ki.BlankProp{}},
			{"OpenParser", ki.Props{
				"shortcut": gi.KeyFunMenuOpenAlt1,
				"label":    "Open Parser...",
				"desc":     "Open lexer and parser rules from standard JSON-formatted file",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Prefs.ParserFile",
						"ext":           ".pi",
					}},
				},
			}},
			{"SaveParser", ki.Props{
				"shortcut": gi.KeyFunMenuSaveAlt,
				"desc":     "Save lexer and parser rules to file standard JSON-formatted file",
				"updtfunc": giv.ActionUpdateFunc(func(pvi interface{}, act *gi.Action) {
					pv := pvi.(*PiView)
					act.SetActiveState( /* pv.Changed && */ pv.Prefs.ParserFile != "")
				}),
			}},
			{"SaveParserAs", ki.Props{
				"label": "Save Parser As...",
				"desc":  "Save As lexer and parser rules to file standard JSON-formatted file",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Prefs.ParserFile",
						"ext":           ".pi",
					}},
				},
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
			{"OpenConsoleTab", ki.Props{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
}

//////////////////////////////////////////////////////////////////////////////////////
//   Project window

// CloseWindowReq is called when user tries to close window -- we
// automatically save the project if it already exists (no harm), and prompt
// to save open files -- if this returns true, then it is OK to close --
// otherwise not
func (pv *PiView) CloseWindowReq() bool {
	if !pv.Changed {
		return true
	}
	gi.ChoiceDialog(pv.Viewport, gi.DlgOpts{Title: "Close Project: There are Unsaved Changes",
		Prompt: fmt.Sprintf("In Project: %v There are <b>unsaved changes</b> -- do you want to save or cancel closing this project and review?", pv.Nm)},
		[]string{"Cancel", "Save Proj", "Close Without Saving"},
		pv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			switch sig {
			case 0:
				// do nothing, will have returned false already
			case 1:
				pv.SaveProj()
			case 2:
				pv.ParentWindow().OSWin.Close() // will not be prompted again!
			}
		})
	return false // not yet
}

// QuitReq is called when user tries to quit the app -- we go through all open
// main windows and look for gide windows and call their CloseWindowReq
// functions!
func QuitReq() bool {
	for _, win := range gi.MainWindows {
		if !strings.HasPrefix(win.Nm, "Pie") {
			continue
		}
		mfr, ok := win.MainWidget()
		if !ok {
			continue
		}
		gek, ok := mfr.ChildByName("piview", 0)
		if !ok {
			continue
		}
		ge := gek.Embed(KiT_PiView).(*PiView)
		if !ge.CloseWindowReq() {
			return false
		}
	}
	return true
}

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
