// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command docs provides documentation of Cogent Core,
// hosted at https://cogentcore.org/core.
package main

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/yaegicore"
	"cogentcore.org/core/yaegicore/coresymbols"
)

//go:embed content
var econtent embed.FS

//go:embed *.svg name.png weld-icon.png
var resources embed.FS

//go:embed image.png
var myImage embed.FS

//go:embed icon.svg
var mySVG embed.FS

//go:embed file.go
var myFile embed.FS

const defaultPlaygroundCode = `package main

func main() {
	b := core.NewBody()
	core.NewButton(b).SetText("Hello, World!")
	b.RunMainWindow()
}`

func main() {
	b := core.NewBody("Cogent Core Docs")
	ct := content.NewContent(b).SetContent(econtent)
	ctx := ct.Context
	content.OfflineURL = "https://cogentcore.org/core"
	ctx.AddWikilinkHandler(htmlcore.GoDocWikilink("doc", "cogentcore.org/core"))
	b.AddTopBar(func(bar *core.Frame) {
		tb := core.NewToolbar(bar)
		tb.Maker(ct.MakeToolbar)
		tb.Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				ctx.LinkButton(w, "playground")
				w.SetText("Playground").SetIcon(icons.PlayCircle)
			})
			tree.Add(p, func(w *core.Button) {
				ctx.LinkButton(w, "https://youtube.com/@CogentCore")
				w.SetText("Videos").SetIcon(icons.VideoLibrary)
			})
			tree.Add(p, func(w *core.Button) {
				ctx.LinkButton(w, "https://cogentcore.org/blog")
				w.SetText("Blog").SetIcon(icons.RssFeed)
			})
			tree.Add(p, func(w *core.Button) {
				ctx.LinkButton(w, "https://github.com/cogentcore/core")
				w.SetText("GitHub").SetIcon(icons.GitHub)
			})
			tree.Add(p, func(w *core.Button) {
				ctx.LinkButton(w, "https://cogentcore.org/community")
				w.SetText("Community").SetIcon(icons.Forum)
			})
			tree.Add(p, func(w *core.Button) {
				ctx.LinkButton(w, "https://github.com/sponsors/cogentcore")
				w.SetText("Sponsor").SetIcon(icons.Favorite)
			})
		})
	})

	coresymbols.Symbols["."]["econtent"] = reflect.ValueOf(econtent)
	coresymbols.Symbols["."]["myImage"] = reflect.ValueOf(myImage)
	coresymbols.Symbols["."]["mySVG"] = reflect.ValueOf(mySVG)
	coresymbols.Symbols["."]["myFile"] = reflect.ValueOf(myFile)

	ctx.ElementHandlers["home-page"] = homePage
	ctx.ElementHandlers["core-playground"] = func(ctx *htmlcore.Context) bool {
		splits := core.NewSplits(ctx.BlockParent)
		ed := textcore.NewEditor(splits)
		playgroundFile := filepath.Join(core.TheApp.AppDataDir(), "playground.go")
		err := ed.Lines.Open(playgroundFile)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				err := os.WriteFile(playgroundFile, []byte(defaultPlaygroundCode), 0666)
				core.ErrorSnackbar(ed, err, "Error creating code file")
				if err == nil {
					err := ed.Lines.Open(playgroundFile)
					core.ErrorSnackbar(ed, err, "Error loading code")
				}
			} else {
				core.ErrorSnackbar(ed, err, "Error loading code")
			}
		}
		ed.OnChange(func(e events.Event) {
			core.ErrorSnackbar(ed, ed.SaveQuiet(), "Error saving code")
		})
		parent := core.NewFrame(splits)
		yaegicore.BindTextEditor(ed, parent, "Go")
		return true
	}
	ctx.ElementHandlers["style-demo"] = func(ctx *htmlcore.Context) bool {
		// same as demo styles tab
		sp := core.NewSplits(ctx.BlockParent)
		sp.Styler(func(s *styles.Style) {
			s.Min.Y.Em(40)
		})
		fm := core.NewForm(sp)
		fr := core.NewFrame(core.NewFrame(sp)) // can not control layout when directly in splits
		fr.Styler(func(s *styles.Style) {
			s.Background = colors.Scheme.Select.Container
			s.Grow.Set(1, 1)
		})
		fr.Style() // must style immediately to get correct default values
		fm.SetStruct(&fr.Styles)
		fm.OnChange(func(e events.Event) {
			fr.OverrideStyle = true
			fr.Update()
		})
		frameSizes := []math32.Vector2{
			{20, 100},
			{80, 20},
			{60, 80},
			{40, 120},
			{150, 100},
		}
		for _, sz := range frameSizes {
			core.NewFrame(fr).Styler(func(s *styles.Style) {
				s.Min.Set(units.Dp(sz.X), units.Dp(sz.Y))
				s.Background = colors.Scheme.Primary.Base
			})
		}
		return true
	}

	b.RunMainWindow()
}

func homePage(ctx *htmlcore.Context) bool {
	home := core.NewFrame(ctx.BlockParent)
	home.Styler(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Grow.Set(1, 1)
		s.CenterAll()
	})
	home.OnShow(func(e events.Event) {
		home.Update() // TODO: temporary workaround for #1037
	})

	tree.AddChild(home, func(w *core.SVG) {
		errors.Log(w.ReadString(core.AppIcon))
	})
	tree.AddChild(home, func(w *core.Image) {
		errors.Log(w.OpenFS(resources, "name.png"))
		w.Styler(func(s *styles.Style) {
			s.Min.X.SetCustom(func(uc *units.Context) float32 {
				return min(uc.Dp(612), uc.Vw(80))
			})
		})
	})
	tree.AddChild(home, func(w *core.Text) {
		w.SetType(core.TextHeadlineMedium).SetText("Interactive documentation for Cogent Core, built using the framework and deployed with WebAssembly")
	})
	tree.AddChild(home, func(w *core.Frame) {
		tree.AddChild(w, func(w *core.Button) {
			ctx.LinkButton(w, "basics")
			w.SetText("Get started")
		})
		tree.AddChild(w, func(w *core.Button) {
			ctx.LinkButton(w, "install")
			w.SetText("Install").SetType(core.ButtonTonal)
		})
	})

	return true
}
