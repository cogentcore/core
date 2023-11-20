// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/styles"
)

// var samplefile gi.FileName = "sample.go"
var samplefile gi.FileName = "../../Makefile"

// var samplefile gi.FileName = "../../README.md"

func main() { gimain.Run(app) }

func app() {
	// gi.LayoutTrace = true
	// gi.EventTrace = true

	gi.SetAppName("texteditor")
	gi.SetAppAbout(`This is a demo of the texteditor.Editor in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki">GoKi on GitHub</a>`)

	gi.DefaultTopAppBar = nil

	sc := gi.NewScene("texteditor-test").SetTitle("GoGi texteditor.Editor Test")

	// hdrText := `This is a <b>test</b> of the texteditor.Editor`
	// title := gi.NewLabel(sc, "title").SetText(hdrText).SetType(gi.LabelHeadlineSmall)
	// title.Style(func(s *styles.Style) {
	// 	s.Text.WhiteSpace = styles.WhiteSpaceNowrap
	// 	s.Text.Align = styles.Center
	// 	s.Text.AlignV = styles.Start
	// })
	//
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

	txed1 := texteditor.NewEditor(splt, "texteditor-1")
	txed1.Style(func(s *styles.Style) {
		s.Min.X.Ch(20)
		s.Min.Y.Ch(10)
	})

	txed2 := texteditor.NewEditor(splt, "texteditor-2")
	txed2.Style(func(s *styles.Style) {
		s.Min.X.Ch(20)
		s.Min.Y.Ch(10)
	})

	txbuf := texteditor.NewBuf()
	txed1.SetBuf(txbuf)
	txed2.SetBuf(txbuf)

	// txbuf.Hi.Lang = "Markdown" // "Makefile" // "Go" // "Markdown"
	txbuf.Hi.Lang = "Makefile"
	txbuf.Open(samplefile)
	// giv.InspectorDialog(&txbuf.Hi.PiLang.Parser().Parser) .Lexer //

	gi.NewWindow(sc).Run().Wait()
}
