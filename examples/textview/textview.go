// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
)

// var samplefile gi.FileName = "sample.go"
var samplefile gi.FileName = "../../README.md"

func main() { gimain.Run(app) }

func app() {
	// gi.LayoutTrace = true
	// gi.EventTrace = true

	goosi.ZoomFactor = 2

	gi.SetAppName("textview")
	gi.SetAppAbout(`This is a demo of the textview.View in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	sc := gi.NewScene("gogi-textview-test").SetTitle("GoGi textview.View Test")

	trow := gi.NewLayout(sc, "trow").SetLayout(gi.LayoutHoriz)
	trow.SetStretchMaxWidth()

	hdrText := `This is a <b>test</b> of the textview.View`
	title := gi.NewLabel(trow, "title").SetText(hdrText).SetType(gi.LabelHeadlineSmall)
	title.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpaceNowrap
		s.Text.Align = styles.AlignCenter
		s.Text.AlignV = styles.AlignTop
	})

	splt := gi.NewSplits(sc, "split-view")
	splt.SetSplits(.5, .5)
	// these are all inherited so we can put them at the top "editor panel" level
	splt.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		s.Text.TabSize = 4
		s.Font.Family = string(gi.Prefs.MonoFont)
		// s.Text.LineHeight = units.Dot(1.1)
	})

	// generally need to put text view within its own layout for scrolling

	txed1 := texteditor.NewView(splt, "textview-1")
	txed1.Style(func(s *styles.Style) {
		s.SetStretchMax()
		s.SetMinPrefWidth(units.Ch(20))
		s.SetMinPrefHeight(units.Ch(10))
	})

	txed2 := texteditor.NewView(splt, "textview-2")
	txed2.Style(func(s *styles.Style) {
		s.SetStretchMax()
		s.SetMinPrefWidth(units.Ch(20))
		s.SetMinPrefHeight(units.Ch(10))
	})

	txbuf := texteditor.NewBuf()
	txed1.SetBuf(txbuf)
	txed2.SetBuf(txbuf)

	txbuf.Hi.Lang = "Markdown" // "Go" // "Markdown"
	txbuf.Open(samplefile)

	gi.NewWindow(sc).Run().Wait()
}
