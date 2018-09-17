// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/goki/gi"
	"github.com/goki/gi/complete"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"go/token"
	"os/exec"
	"sort"
)

var samplefile gi.FileName = "sample.in"
var txbuf *giv.TextBuf

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768

	// gi.Layout2DTrace = true

	oswin.TheApp.SetName("textview")
	oswin.TheApp.SetAbout(`This is a demo of the TextView in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	win := gi.NewWindow2D("gogi-textview-test", "GoGi TextView Test", width, height, true) // true = pixel sizes

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	// // style sheet
	// var css = ki.Props{
	// 	"kbd": ki.Props{
	// 		"color": "blue",
	// 	},
	// }
	// vp.CSS = css

	mfr := win.SetMainFrame()

	trow := mfr.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutHoriz
	trow.SetStretchMaxWidth()

	title := trow.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	hdrText := `This is a <b>test</b> of the TextView`
	title.Text = hdrText
	title.SetProp("text-align", gi.AlignCenter)
	title.SetProp("vertical-align", gi.AlignTop)
	title.SetProp("font-size", "x-large")

	splt := mfr.AddNewChild(gi.KiT_SplitView, "split-view").(*gi.SplitView)
	splt.SetSplits(.5, .5)
	// these are all inherited so we can put them at the top "editor panel" level
	splt.SetProp("white-space", gi.WhiteSpacePreWrap)
	splt.SetProp("tab-size", 4)
	splt.SetProp("font-family", "Go Mono")
	splt.SetProp("line-height", 1.1)

	// generally need to put text view within its own layout for scrolling
	txly1 := splt.AddNewChild(gi.KiT_Layout, "view-layout-1").(*gi.Layout)
	txly1.SetStretchMaxWidth()
	txly1.SetStretchMaxHeight()
	txly1.SetMinPrefWidth(units.NewValue(20, units.Ch))
	txly1.SetMinPrefHeight(units.NewValue(10, units.Ch))

	txed1 := txly1.AddNewChild(giv.KiT_TextView, "textview-1").(*giv.TextView)
	txed1.HiStyle = "emacs"
	txed1.Opts.LineNos = true
	txed1.SetCompleter(nil, CompleteGocode, CompleteEdit)

	// generally need to put text view within its own layout for scrolling
	txly2 := splt.AddNewChild(gi.KiT_Layout, "view-layout-2").(*gi.Layout)
	txly2.SetStretchMaxWidth()
	txly2.SetStretchMaxHeight()
	txly2.SetMinPrefWidth(units.NewValue(20, units.Ch))
	txly2.SetMinPrefHeight(units.NewValue(10, units.Ch))

	txed2 := txly2.AddNewChild(giv.KiT_TextView, "textview-2").(*giv.TextView)
	txed2.HiStyle = "emacs"

	txbuf = giv.NewTextBuf()
	txed1.SetBuf(txbuf)
	txed2.SetBuf(txbuf)

	txbuf.Open(samplefile)
	txbuf.HiLang = "Go"

	// main menu
	appnm := oswin.TheApp.Name()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "Edit", "Window"})

	amen := win.MainMenu.KnownChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.KnownChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	win.OSWin.SetCloseCleanFunc(func(w oswin.Window) {
		go oswin.TheApp.Quit() // once main window is closed, quit
	})

	win.MainMenuUpdated()
	vp.UpdateEndNoSig(updt)

	// todo: find a place for this code perhaps in SetCompleter
	cmd := exec.Command("gocode", "close")
	defer cmd.Run()

	win.StartEventLoop()
}

// CompleteGocode uses github.com/mdempsky/gocode to do code completion
func CompleteGocode(text string, pos token.Position) (matches complete.Completions, seed string) {
	pos.Filename = string(samplefile)
	seed = complete.SeedGolang(text)

	textbytes := make([]byte, 0, txbuf.NLines*40)
	for _, lr := range txbuf.Lines {
		textbytes = append(textbytes, []byte(string(lr))...)
		textbytes = append(textbytes, '\n')
	}
	results := complete.GetCompletions(textbytes, pos)

	sort.Slice(results, func(i, j int) bool {
		if results[i].Text < results[j].Text {
			return true
		}
		if results[i].Text > results[j].Text {
			return false
		}
		return results[i].Text < results[j].Text
	})

	matches = complete.MatchSeedCompletion(results, seed)
	return matches, seed
}

// CompleteEdit uses the selected completion to edit the text
func CompleteEdit(text string, cursorPos int, selection string, seed string) (s string, delta int) {
	s, delta = complete.EditWord(text, cursorPos, selection, seed)
	return s, delta
}

// CompleteGo is not being used - it calls a new code completer that was started but is not
// under development at this time
func CompleteGo(text string, pos token.Position) (matches complete.Completions, seed string) {
	pos.Filename = string(samplefile)
	textbytes := make([]byte, 0, txbuf.NLines*40)
	for _, lr := range txbuf.Lines {
		textbytes = append(textbytes, []byte(string(lr))...)
		textbytes = append(textbytes, '\n')
	}
	results, seed := complete.CompleteGo(textbytes, pos)
	if len(seed) > 0 {
		results = complete.MatchSeedString(results, seed)
	}
	for _, lv := range results {
		m := complete.Completion{Text: lv}
		matches = append(matches, m)
	}
	return matches, seed
}
