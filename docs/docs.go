// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command docs provides documentation of Cogent Core,
// hosted at https://cogentcore.org/core.
package main

//go:generate core generate

import (
	"embed"
	"io/fs"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlview"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/pages"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/views"
)

//go:embed content
var content embed.FS

//go:embed name.png code-icon.svg mail-icon.svg emergent-icon.svg weld-icon.png
var resources embed.FS

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
	// b.AddAppBar(pg.AppBar)

	htmlview.ElementHandlers["home-page"] = homePage

	b.RunMainWindow()
}

func homePage(ctx *htmlview.Context) bool {
	frame := core.NewFrame(ctx.BlockParent).Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.CenterAll()
		// s.Background = gradient.NewLinear().AddStop(colors.Scheme.Primary.Container, 0).AddStop(colors.Scheme.Warn.Container, 1)
	})
	errors.Log(core.NewSVG(frame).ReadString(core.AppIcon))
	img := core.NewImage(frame)
	errors.Log(img.OpenFS(resources, "name.png"))
	img.Style(func(s *styles.Style) {
		x := func(uc *units.Context) float32 {
			return min(uc.Dp(612), uc.Pw(90))
		}
		s.Min.Set(units.Custom(x), units.Custom(func(uc *units.Context) float32 {
			return x(uc) * (128.0 / 612.0)
		}))
	})
	core.NewText(frame).SetType(core.TextHeadlineMedium).SetText("A cross-platform framework for building powerful, fast, and elegant 2D and 3D apps")

	core.NewButton(frame).SetText("Get started").OnClick(func(e events.Event) {
		ctx.OpenURL("getting-started")
	})

	makeBlock := func(title, text string, graphic func(parent core.Widget)) {
		block := core.NewFrame(frame).Style(func(s *styles.Style) {
			s.CenterAll()
			s.Gap.Set(units.Em(1))
			if frame.SizeClass() == core.SizeCompact {
				s.Direction = styles.Column
			}
		})

		graphicFirst := frame.NumChildren()%2 == 0
		if graphicFirst {
			graphic(block)
			block.Style(func(s *styles.Style) {
				// we dynamically swap the graphic and text block so that they
				// are ordered correctly based on our size class
				sc := block.SizeClass()
				wrongCompact := sc == core.SizeCompact && block.Child(1).Name() == "text-block"
				wrongNonCompact := sc != core.SizeCompact && block.Child(0).Name() == "text-block"
				if wrongCompact || wrongNonCompact {
					block.Kids.Move(0, 1)
				}
			})
		}

		textBlock := core.NewFrame(block).Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Text.Align = styles.Start
		})
		textBlock.SetName("text-block")
		core.NewText(textBlock).SetType(core.TextHeadlineLarge).SetText(title).Style(func(s *styles.Style) {
			s.Font.Weight = styles.WeightBold
			s.Color = colors.C(colors.Scheme.Primary.Base)
		})
		core.NewText(textBlock).SetType(core.TextTitleLarge).SetText(text)

		if !graphicFirst {
			graphic(block)
		}
	}

	makeBlock("CODE ONCE, RUN EVERYWHERE", "With Cogent Core, you can write your app once and it will instantly run on macOS, Windows, Linux, iOS, Android, and the Web, automatically scaling to any screen. Instead of struggling with platform-specific code in a multitude of languages, you can easily write and maintain a single pure Go codebase.", func(parent core.Widget) {
		core.NewIcon(parent).SetIcon(icons.Devices).Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(256))
		})
	})

	makeBlock("EFFORTLESS ELEGANCE", "Cogent Core is built on Go, a high-level language designed for building elegant, readable, and scalable code with full type safety and a robust design that never gets in your way. Cogent Core makes it easy to get started with cross-platform app development in just two commands and seven lines of simple code.", func(parent core.Widget) {
		// we get the code example contained within the md file
		ctx.Node = ctx.Node.FirstChild.NextSibling
		ctx.BlockParent = core.NewFrame(parent).Style(func(s *styles.Style) {
			s.Direction = styles.Column
		})
		htmlview.HandleElement(ctx)
	})

	makeBlock("COMPLETELY CUSTOMIZABLE", "Cogent Core allows developers and users to fully customize apps to fit their unique needs and preferences through a robust styling system and a powerful color system that allow developers and users to instantly customize every aspect of the appearance and behavior of an app.", func(parent core.Widget) {
		views.NewStructView(parent).SetStruct(core.AppearanceSettings).OnChange(func(e events.Event) {
			core.UpdateSettings(parent, core.AppearanceSettings)
		})

	})

	makeBlock("POWERFUL FEATURES", "Cogent Core comes with a powerful set of advanced features that allow you to make almost anything, including fully featured text editors, video and audio players, interactive 3D graphics, customizable data plots, Markdown and HTML rendering, SVG and canvas vector graphics, and automatic views of any Go data structure for instant data binding and advanced app inspection.", func(parent core.Widget) {
		texteditor.NewSoloEditor(parent).Buffer.SetLang("go").SetTextString(`package main

		func main() {
			fmt.Println("Hello, world!")
		}
		`)
	})

	makeBlock("OPTIMIZED EXPERIENCE", "Every part of your development experience is guided by a comprehensive set of interactive example-based documentation, in-depth video tutorials, easy-to-use command line tools specialized for Cogent Core, and active support and development from the Cogent Core developers.", func(parent core.Widget) {
		core.NewIcon(parent).SetIcon(icons.PlayCircle).Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(256))
		})
	})

	makeBlock("EXTREMELY FAST", "Cogent Core is powered by Vulkan, a modern, cross-platform, high-performance graphics framework that allows apps to run on all platforms at extremely fast speeds. All Cogent Core apps compile to machine code, allowing them to run without any overhead.", func(parent core.Widget) {
		core.NewIcon(parent).SetIcon(icons.Bolt).Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(256))
		})
	})

	makeBlock("FREE AND OPEN SOURCE", "Cogent Core is completely free and open source under the permissive BSD-3 License, allowing you to use Cogent Core for any purpose, commercially or personally. We believe that software works best when everyone can use it.", func(parent core.Widget) {
		core.NewIcon(parent).SetIcon(icons.Code).Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(256))
		})
	})

	makeBlock("USED AROUND THE WORLD", "Over six years of development, Cogent Core has been used and thoroughly tested by developers and scientists around the world for a wide variety of use cases. Cogent Core is a production-ready framework actively used to power everything from end-user apps to scientific research.", func(parent core.Widget) {
		core.NewIcon(parent).SetIcon(icons.GlobeAsia).Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(256))
		})
	})

	core.NewText(frame).SetType(core.TextDisplaySmall).SetText("<b>What can Cogent Core do?</b>")

	makeBlock("COGENT CODE", "Cogent Code is a fully featured Go IDE with support for syntax highlighting, code completion, symbol lookup, building and debugging, version control, keyboard shortcuts, and many other features.", func(parent core.Widget) {
		errors.Log(core.NewSVG(parent).OpenFS(resources, "code-icon.svg"))
	})

	makeBlock("COGENT VECTOR", "Cogent Vector is a powerful vector graphics editor with complete support for shapes, paths, curves, text, images, gradients, groups, alignment, styling, importing, exporting, undo, redo, and various other features.", func(parent core.Widget) {
		core.NewIcon(parent).SetIcon(icons.Polyline).Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(256))
		})
	})

	makeBlock("COGENT MAIL", "Cogent Mail is a customizable email client with built-in Markdown support and an extensive set of keyboard shortcuts for advanced mail filing.", func(parent core.Widget) {
		errors.Log(core.NewSVG(parent).OpenFS(resources, "mail-icon.svg"))
	})

	makeBlock("EMERGENT", "Emergent is a collection of biologically based 3D neural network models of the brain that power ongoing research in computational cognitive neuroscience.", func(parent core.Widget) {
		errors.Log(core.NewSVG(parent).OpenFS(resources, "emergent-icon.svg"))
	})

	makeBlock("WELD", "WELD is a set of 3D computational models of a new approach to quantum physics based on wave electrodynamics.", func(parent core.Widget) {
		errors.Log(core.NewImage(parent).OpenFS(resources, "weld-icon.png"))
	})

	core.NewText(frame).SetType(core.TextDisplaySmall).SetText("<b>Why Cogent Core instead of something else?</b>")

	makeBlock(`THE PROBLEM`, "After using other frameworks built on HTML and Qt for years to make various apps ranging from simple tools to complex scientific models, we realized that we were spending more time dealing with excessive boilerplate, browser inconsistencies, and dependency management issues than actual app development.", func(parent core.Widget) {
		core.NewIcon(parent).SetIcon(icons.Blank).Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(256))
		})
	})

	makeBlock(`THE SOLUTION`, "We decided to make a framework that would allow us to focus on app content and logic by providing a consistent and elegant API that automatically handles cross-platform support, user customization, and app packaging and deployment.", func(parent core.Widget) {
		core.NewIcon(parent).SetIcon(icons.Blank).Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(256))
		})
	})

	makeBlock(`THE RESULT`, "Instead of constantly jumping through hoops to create a consistent, easy-to-use, cross-platform app, you can now take advantage of a powerful featureset on all platforms and vastly simplify your development experience.", func(parent core.Widget) {
		core.NewIcon(parent).SetIcon(icons.Blank).Style(func(s *styles.Style) {
			s.Min.Set(units.Dp(256))
		})
	})

	core.NewButton(frame).SetText("Get started").OnClick(func(e events.Event) {
		ctx.OpenURL("getting-started")
	})

	return true
}
