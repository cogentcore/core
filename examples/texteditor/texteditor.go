// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/gi"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
)

var samplefile gi.Filename = "texteditor.go"

// var samplefile gi.Filename = "../../Makefile"

// var samplefile gi.Filename = "../../README.md"

func main() {
	b := gi.NewAppBody("Cogent Core Text Editor Demo")
	b.App().About = `This is a demo of the texteditor.Editor in the <b>Cogent Core</b> graphical interface system, within the <b>Goki</b> tree framework.  See <a href="https://github.com/goki">Cogent Core on GitHub</a>`

	splt := gi.NewSplits(b, "split-view")
	splt.SetSplits(.5, .5)
	// these are all inherited so we can put them at the top "editor panel" level
	splt.Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpacePreWrap
		s.Text.TabSize = 4
		s.Font.Family = string(gi.AppearanceSettings.MonoFont)
	})

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
	txbuf.Hi.Lang = "Go"
	txbuf.Open(samplefile)
	// pr := txbuf.Hi.PiLang.Parser()
	// giv.InspectorDialog(&pr.Lexer)

	b.RunMainWindow()
}
