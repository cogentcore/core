// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command docs provides documentation of Cogent Core,
// hosted at https://cogentcore.org/core.
package main

//go:generate core generate -pages content

import (
	"embed"
	"io/fs"

	"cogentcore.org/core/core"
	"cogentcore.org/core/errors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlview"
	"cogentcore.org/core/pages"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

//go:embed content
var content embed.FS

//go:embed icon.svg
var icon []byte

//go:embed name.png
var name embed.FS

//go:embed image.png
var myImage embed.FS

//go:embed icon.svg
var mySVG embed.FS

//go:embed file.go
var myFile embed.FS

func main() {
	b := core.NewBody("Cogent Core Docs")
	pg := pages.NewPage(b).SetSource(errors.Log1(fs.Sub(content, "content")))
	pg.Context.WikilinkResolver = htmlview.PkgGoDevWikilink("cogentcore.org/core")
	b.AddAppBar(pg.AppBar)

	htmlview.ElementHandlers["home-header"] = func(ctx *htmlview.Context) bool {
		ly := core.NewLayout(ctx.BlockParent).Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Justify.Content = styles.Center
			s.Align.Content = styles.Center
			s.Align.Items = styles.Center
			s.Text.Align = styles.Center
		})
		errors.Log(core.NewSVG(ly).ReadBytes(icon))
		img := core.NewImage(ly)
		errors.Log(img.OpenFS(name, "name.png"))
		img.Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(612), units.Dp(128))
		})
		return true
	}
	htmlview.ElementHandlers["get-started"] = func(ctx *htmlview.Context) bool {
		core.NewButton(ctx.BlockParent).SetText("Get Started").OnClick(func(e events.Event) {
			pg.OpenURL("/getting-started", true)
		}).Style(func(s *styles.Style) {
			s.Align.Self = styles.Center
		})
		return true
	}

	b.RunMainWindow()
}
