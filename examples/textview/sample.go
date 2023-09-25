// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

package main

import (
	"io/ioutil"
	"log"
	"os"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/girl/units"
	"goki.dev/goosi"
)

func main() {
	gimain.Main(mainrun)
}

func mainrun() {
	width := 1024
	height := 768

	// gi.LayoutTrace = true

	goosi.TheApp.SetName("text")
	goosi.TheApp.SetAbout(`This is a demo of the TextEdit in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	win := gi.NewMainRenderWin("gogi-textedit-test", "GoGi TextEdit Test", width, height)

	vp := win.WinScene()
	updt := vp.UpdateStart()

	// // style sheet
	// var css = ki.Props{
	// 	"kbd": ki.Props{
	// 		"color": "blue",
	// 	},
	// }
	// vp.CSS = css

	mfr := win.SetMainFrame()

	trow := gi.NewLayout(mfr, "trow", gi.LayoutHoriz)
	trow.SetStretchMaxWidth()

	hdrText := `This is a <b>test</b> of the TextEdit`
	title := gi.NewLabel(trow, "title", hdrText)
	title.SetProp("text-align", gi.AlignCenter)
	title.SetProp("vertical-align", gi.AlignTop)
	title.SetProp("font-size", "x-large")

	txed := giv.NewTextEdit(mfr, "textedit")
	// txed.SetProp("word-wrap", true)
	txed.SetProp("max-width", -1)
	txed.SetProp("min-width", units.NewCh(80))
	txed.SetProp("min-height", units.NewCh(40))
	// txed.SetProp("line-height", 1.2)
	// txed.SetProp("para-spacing", "1ex")
	// txed.SetProp("text-indent", "20px")
	txed.HiLang = "Go"
	txed.HiStyle = "emacs"

	fp, err := os.Open("sample.in")
	if err != nil {
		log.Println(err)
		// return err
	}
	b, err := ioutil.ReadAll(fp)
	txed.Txt = string(b)
	fp.Close()

	// main menu
	appnm := goosi.TheApp.Name()
	mmen := win.MainMenu
	mmen.ConfigMenus([]string{appnm, "Edit", "RenderWin"})

	amen := win.MainMenu.ChildByName(appnm, 0).(*gi.Action)
	amen.Menu = make(gi.Menu, 0, 10)
	amen.Menu.AddAppMenu(win)

	emen := win.MainMenu.ChildByName("Edit", 1).(*gi.Action)
	emen.Menu = make(gi.Menu, 0, 10)
	emen.Menu.AddCopyCutPaste(win)

	win.RenderWin.SetCloseCleanFunc(func(w goosi.RenderWin) {
		go goosi.TheApp.Quit() // once main window is closed, quit
	})

	win.MainMenuUpdated()

	vp.UpdateEndNoSig(updt)

	win.StartEventLoop()
}
