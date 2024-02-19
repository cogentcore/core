// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/gi"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
)

//go:embed texteditor.go
var samplefile embed.FS

func main() {
	b := gi.NewBody("Cogent Core Text Editor Demo")

	// gi.DebugSettings.LayoutTrace = true

	// sp := gi.NewSplits(b)
	// sp.SetSplits(.5, .5)
	// these are all inherited so we can put them at the top "editor panel" level
	// sp.Style(func(s *styles.Style) {
	// 	s.Text.WhiteSpace = styles.WhiteSpacePreWrap
	// 	s.Text.TabSize = 4
	// 	s.Font.Family = string(gi.AppearanceSettings.MonoFont)
	// })

	te1 := texteditor.NewEditor(b)
	te1.Style(func(s *styles.Style) {
		s.Min.X.Ch(20)
		s.Min.Y.Ch(10)
	})
	// te2 := texteditor.NewEditor(sp)
	// te2.Style(func(s *styles.Style) {
	// 	s.Min.X.Ch(20)
	// 	s.Min.Y.Ch(10)
	// })

	tb := texteditor.NewBuf()
	te1.SetBuf(tb)
	// te2.SetBuf(tb)

	tb.Hi.Lang = "Go"
	tb.OpenFS(samplefile, "texteditor.go")

	b.RunMainWindow()
}
