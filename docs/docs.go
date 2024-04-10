// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

//go:generate core generate -webcore content

import (
	"embed"
	"io/fs"

	"cogentcore.org/core/core"
	"cogentcore.org/core/coredom"
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

//go:embed icon.svg
var mySVG embed.FS

//go:embed file.go
var myFile embed.FS

func main() {
	b := core.NewBody("Cogent Core Docs")
	pg := webcore.NewPage(b).SetSource(grr.Log1(fs.Sub(content, "content")))
	pg.Context.WikilinkResolver = coredom.PkgGoDevWikilink("cogentcore.org/core")
	b.AddAppBar(pg.AppBar)

	coredom.ElementHandlers["home-header"] = func(ctx *coredom.Context) bool {
		ly := core.NewLayout(ctx.BlockParent).Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Justify.Content = styles.Center
			s.Align.Content = styles.Center
			s.Align.Items = styles.Center
			s.Text.Align = styles.Center
		})
		grr.Log(core.NewSVG(ly).ReadBytes(icon))
		core.NewLabel(ly).SetType(core.LabelDisplayLarge).SetText("Cogent Core")
		return true
	}

	b.RunMainWindow()
}
