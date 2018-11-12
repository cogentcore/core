// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
)

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	width := 1024
	height := 768

	// gi.Layout2DTrace = true

	oswin.TheApp.SetName("label")
	oswin.TheApp.SetAbout(`This is a demo of the text rendering using labels in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	win := gi.NewWindow2D("gogi-label-test", "GoGi Label Test", width, height, true) // true = pixel sizes

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	// style sheet
	var css = ki.Props{
		"kbd": ki.Props{
			"color": "blue",
		},
	}
	vp.CSS = css

	mfr := win.SetMainFrame()

	trow := mfr.AddNewChild(gi.KiT_Layout, "trow").(*gi.Layout)
	trow.Lay = gi.LayoutHoriz
	trow.SetStretchMaxWidth()

	title := trow.AddNewChild(gi.KiT_Label, "title").(*gi.Label)
	hdrText := `This is a <b>test</b> of the
	 <span style="color:red">various</span> <i>GoGi</i> Text elements<br>
	 <large>Shortcuts: <kbd>Ctrl+Alt+P</kbd> = Preferences,
	 <kbd>Ctrl+Alt+E</kbd> = Editor, <kbd>Ctrl/Cmd +/-</kbd> = zoom</large><br>
	 Other styles: <u>underlining</u> and <abbr>abbr dotted uline</abbr> and <strike>strikethrough</strike><br>
	 <q>and</q> <mark>marked text</mark> and <span style="text-decoration:overline">overline</span>
	 and Sub<sub>script</sub> and Super<sup>script</sup>`
	title.Text = hdrText
	// title.Text = "header" // use this to test word wrapping
	title.SetProp("white-space", gi.WhiteSpaceNormal)
	title.SetProp("text-align", gi.AlignRight)
	title.SetProp("vertical-align", gi.AlignTop)
	title.SetProp("font-family", "Times New Roman, serif")
	title.SetProp("font-size", "x-large")
	// title.SetProp("letter-spacing", 2)
	title.SetProp("line-height", 1.5)

	rtxt := trow.AddNewChild(gi.KiT_Label, "rtxt").(*gi.Label)
	rtxt.Text = "this is to test right margin"

	wrlab := mfr.AddNewChild(gi.KiT_Label, "wrlab").(*gi.Label)
	wrlab.SetProp("white-space", gi.WhiteSpaceNormal)
	wrlab.SetProp("width", "20em")
	wrlab.SetProp("max-width", -1)
	wrlab.SetProp("line-height", 1.2)
	wrlab.SetProp("para-spacing", "1ex")
	wrlab.SetProp("text-indent", "20px")
	wrlab.Text = `<p>Word <u>wrapping</u> should be <span style="color:red">enabled in this label</span> -- this is a test to see if it is.  Usually people use some kind of obscure latin text here -- not really sure why <u>because nobody reads latin anymore,</u> at least nobody I know.  Anyway, let's see how this works.  Also, it should be interesting to determine how word wrapping works with styling -- <large>the styles should properly wrap across the lines</large>.  In addition, there is the question of <b>how built-in breaks interface</b> with the auto-line breaks, and furthermore the question of paragraph breaks versus simple br line breaks.</p>
<p>One major question is the extent to which <a href="https://en.wikipedia.org/wiki/Line_wrap_and_word_wrap">word wrapping</a> can be made sensitive to the overall size of the containing element -- this is easy when setting a direct fixed width, but word wrapping should also occur as the user resizes the window.</p>
It appears that the <b>end</b> of one paragraph implies the start of a new one, even if you do <i>not</i> insert a <code>p</code> tag.
`

	// mfr.AddNewChild(gi.KiT_Space, "aspc")

	etxt := mfr.AddNewChild(gi.KiT_Label, "etxt").(*gi.Label)
	etxt.Text = "this is to test bottom after word wrapped text"

	str := mfr.AddNewChild(gi.KiT_Stretch, "str").(*gi.Stretch)
	str.SetMinPrefHeight(units.NewValue(5, units.Em))

	etxt2 := mfr.AddNewChild(gi.KiT_Label, "etxt2").(*gi.Label)
	etxt2.Text = "this is after final stretch"

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

	win.StartEventLoop()
}
