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
	"cogentcore.org/core/views"
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
		s.CenterAll()
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
	})

	homeTextBlock(blocks, "CODE ONCE,\nRUN EVERYWHERE", "With Cogent Core, you can write your app once and it will instantly run on macOS, Windows, Linux, iOS, Android, and the Web, automatically scaling to any screen. Instead of struggling with platform-specific code in a multitude of languages, you can easily write and maintain a single pure Go codebase.")
	core.NewIcon(blocks).SetIcon(icons.Devices).Style(func(s *styles.Style) {
		s.Min.Set(units.Pw(40))
	})

	// we get the code example contained within the md file
	ctx.Node = ctx.Node.FirstChild.NextSibling
	ctx.BlockParent = core.NewLayout(blocks).Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	htmlview.HandleElement(ctx)

	homeTextBlock(blocks, "EFFORTLESS ELEGANCE", "Cogent Core is built on Go, a high-level language designed for building elegant, readable, and scalable code with full type safety and a robust design that never gets in your way. Cogent Core makes it easy to get started with cross-platform app development in just two commands and seven lines of simple code.")

	homeTextBlock(blocks, "COMPLETELY CUSTOMIZABLE", "Cogent Core allows developers and users to fully customize apps to fit their unique needs and preferences through a robust styling system and a powerful color system that allow developers and users to instantly customize every aspect of the appearance and behavior of an app.")
	views.NewStructView(blocks).SetStruct(core.AppearanceSettings).OnChange(func(e events.Event) {
		core.UpdateSettings(blocks, core.AppearanceSettings)
	})
	return true
}

func homeTextBlock(parent core.Widget, title, text string) {
	block := core.NewLayout(parent).Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Text.Align = styles.Start
		s.Min.X.Pw(50)
		s.Grow.Set(0, 0)
	})
	core.NewText(block).SetType(core.TextHeadlineLarge).SetText(title).Style(func(s *styles.Style) {
		s.Font.Weight = styles.WeightBold
	})
	core.NewText(block).SetType(core.TextTitleLarge).SetText(text)
}
