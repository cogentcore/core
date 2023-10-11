// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/textview"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
)

var samplefile gi.FileName = "sample.go"

func main() { gimain.Run(app) }

func app() {
	// gi.LayoutTrace = true

	gi.SetAppName("textview")
	gi.SetAppAbout(`This is a demo of the textview.View in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	sc := gi.StageScene("gogi-textview-test").SetTitle("GoGi textview.View Test")

	trow := gi.NewLayout(sc, "trow").SetLayout(gi.LayoutHoriz)
	trow.SetStretchMaxWidth()

	hdrText := `This is a <b>test</b> of the textview.View`
	title := gi.NewLabel(trow, "title").SetText(hdrText).SetType(gi.LabelHeadlineSmall)
	title.AddStyles(func(s *styles.Style) {
		s.Text.Align = styles.AlignCenter
		s.Text.AlignV = styles.AlignTop
	})

	/*
		splt := gi.NewSplits(sc, "split-view")
		splt.SetSplits(.5, .5)
		// these are all inherited so we can put them at the top "editor panel" level
		splt.SetProp("white-space", styles.WhiteSpacePreWrap)
		splt.SetProp("tab-size", 4)
		splt.SetProp("font-family", gi.Prefs.MonoFont)
		splt.SetProp("line-height", 1.1)
	*/

	// generally need to put text view within its own layout for scrolling
	txly1 := gi.NewLayout(splt, "view-layout-1")
	txly1.AddStyles(func(s *styles.Style) {
		s.SetStretchMaxWidth()
		s.SetStretchMaxHeight()
		s.SetMinPrefWidth(units.Ch(20))
		s.SetMinPrefHeight(units.Ch(10))
	})

	txed1 := textview.NewView(txly1, "textview-1")
	_ = txed1

	/*
		// generally need to put text view within its own layout for scrolling
		txly2 := splt.NewChild(gi.LayoutType, "view-layout-2").(*gi.Layout)
		txly2.SetStretchMaxWidth()
		txly2.SetStretchMaxHeight()
		txly2.SetMinPrefWidth(units.Ch(20))
		txly2.SetMinPrefHeight(units.Ch(10))

		txed2 := textview.NewView(txly1, "textview-2")
	*/

	txbuf := textview.NewBuf()
	txed1.SetBuf(txbuf)
	// txed2.SetBuf(txbuf)

	txbuf.Hi.Lang = "Go"
	txbuf.Open(samplefile)

	// // main menu
	// appnm := gi.AppName()
	// mmen := win.MainMenu
	// mmen.ConfigMenus([]string{appnm, "Edit", "RenderWin"})
	//
	// amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Button)
	// amen.Menu = make(gi.MenuStage, 0, 10)
	// amen.Menu.AddAppMenu(win)
	//
	// emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Button)
	// emen.Menu = make(gi.MenuStage, 0, 10)
	// emen.Menu.AddCopyCutPaste(win)
	//
	// win.SetCloseCleanFunc(func(w *gi.RenderWin) {
	// 	go gi.Quit() // once main window is closed, quit
	// })

	gi.NewWindow(sc).Run().Wait()
}
