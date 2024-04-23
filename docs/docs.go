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
	"cogentcore.org/core/icons"
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

	htmlview.ElementHandlers["home-header"] = homeHeader
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

func homeHeader(ctx *htmlview.Context) bool {
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
		x := func(uc *units.Context) float32 {
			return min(uc.Dp(612), uc.Vw(90))
		}
		s.Min.Set(units.Custom(x), units.Custom(func(uc *units.Context) float32 {
			return x(uc) * (128.0 / 612.0)
		}))
	})
	core.NewText(ly).SetType(core.TextHeadlineMedium).SetText("A cross-platform framework for building powerful, fast, and cogent 2D and 3D apps")

	blocks := core.NewLayout(ly).Style(func(s *styles.Style) {
		s.Display = styles.Grid
		s.Columns = 2
		s.Gap.Set(units.Em(1))
	})

	homeTextBlock(blocks, "CODE ONCE,\nRUN EVERYWHERE", "With Cogent Core, you can write your app once and it will instantly run on macOS, Windows, Linux, iOS, Android, and the Web, automatically scaling to any device.")
	core.NewIcon(blocks).SetIcon(icons.Devices).Style(func(s *styles.Style) {
		s.Min.Set(units.Pw(50))
	})

	core.NewIcon(blocks).SetIcon(icons.DeployedCode).Style(func(s *styles.Style) {
		s.Min.Set(units.Pw(50))
	})
	homeTextBlock(blocks, "EFFORTLESS ELEGANCE", "With the power of Go, Cogent Core allows you to easily write simple, elegant, and readable code with full type safety and a robust design that never gets in your way.")
	return true
}

func homeTextBlock(parent core.Widget, title, text string) {
	block := core.NewLayout(parent).Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Text.Align = styles.Start
		s.Max.X.Pw(50)
	})
	core.NewText(block).SetType(core.TextHeadlineLarge).SetText(title).Style(func(s *styles.Style) {
		s.Font.Weight = styles.WeightBold
	})
	core.NewText(block).SetType(core.TextTitleLarge).SetText(text)
}
