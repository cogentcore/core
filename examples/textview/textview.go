// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"go/token"
	"log"
	"os/exec"

	"github.com/goki/gi"
	"github.com/goki/gi/complete"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/giv"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
)

var samplefile gi.FileName = "sample.in"

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
	txed1.Opts.LineNos = true
	txed1.Opts.Completion = true
	txed1.SetCompleter(txed1, CompleteGo, CompleteGoEdit)

	// generally need to put text view within its own layout for scrolling
	txly2 := splt.AddNewChild(gi.KiT_Layout, "view-layout-2").(*gi.Layout)
	txly2.SetStretchMaxWidth()
	txly2.SetStretchMaxHeight()
	txly2.SetMinPrefWidth(units.NewValue(20, units.Ch))
	txly2.SetMinPrefHeight(units.NewValue(10, units.Ch))

	txed2 := txly2.AddNewChild(giv.KiT_TextView, "textview-2").(*giv.TextView)

	txbuf := giv.NewTextBuf()
	txed1.SetBuf(txbuf)
	txed2.SetBuf(txbuf)

	txbuf.Hi.Lang = "Go"
	txbuf.Hi.Style = "emacs"
	txbuf.Open(samplefile)

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

// Complete uses a combination of AST and github.com/mdempsky/gocode to do code completion
func CompleteGo(data interface{}, text string, pos token.Position) (matches complete.Completions, seed string) {
	var txbuf *giv.TextBuf
	switch t := data.(type) {
	case *giv.TextView:
		txbuf = t.Buf
	}
	if txbuf == nil {
		log.Printf("complete.Complete: txbuf is nil - can't do code completion\n")
		return
	}

	seed = complete.SeedGolang(text)

	textbytes := make([]byte, 0, txbuf.NLines*40)
	for _, lr := range txbuf.Lines {
		textbytes = append(textbytes, []byte(string(lr))...)
		textbytes = append(textbytes, '\n')
	}
	results := complete.CompleteGo(textbytes, pos)
	matches = complete.MatchSeedCompletion(results, seed)
	return matches, seed
}

// CompleteEdit uses the selected completion to edit the text
func CompleteGoEdit(data interface{}, text string, cursorPos int, selection string, seed string) (s string, delta int) {
	s, delta = complete.EditGoCode(text, cursorPos, selection, seed)
	return s, delta
}
