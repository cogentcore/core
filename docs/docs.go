// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate -webcore content

import (
	"embed"
	"io/fs"

	"cogentcore.org/core/coredom"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/webcore"
)

//go:embed content
var content embed.FS

//go:embed icon.svg
var icon []byte

//go:embed image.png
var myImage embed.FS

func main() {
	b := gi.NewBody("Cogent Core Docs")
	pg := webcore.NewPage(b).SetSource(grr.Log1(fs.Sub(content, "content")))
	pg.Context.WikilinkResolver = coredom.PkgGoDevWikilink("cogentcore.org/core")
	b.AddAppBar(pg.AppBar)

	coredom.ElementHandlers["home-header"] = func(ctx *coredom.Context) bool {
		ly := gi.NewLayout(ctx.BlockParent).Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Justify.Content = styles.Center
			s.Align.Content = styles.Center
			s.Align.Items = styles.Center
			s.Text.Align = styles.Center
		})
		grr.Log(gi.NewSVG(ly).ReadBytes(icon))
		gi.NewLabel(ly).SetType(gi.LabelDisplayLarge).SetText("Cogent Core")
		return true
	}

	b.RunMainWindow()
}
