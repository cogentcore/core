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
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
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
			tree.Add(p, func(w *core.Button) {
				w.SetText("PDF").SetIcon(icons.PictureAsPdf)
				w.OnClick(func(e events.Event) {
					ct.PagePDF("")
				})
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

var home *core.Frame

func makeBlock[T tree.NodeValue](title, txt string, graphic func(w *T), url ...string) {
	if len(url) > 0 {
		title = `<a target="_blank" href="` + url[0] + `">` + title + `</a>`
	}
	tree.AddChildAt(home, title, func(w *core.Frame) {
		w.Styler(func(s *styles.Style) {
			s.Gap.Set(units.Em(1))
			s.Grow.Set(1, 0)
			if home.SizeClass() == core.SizeCompact {
				s.Direction = styles.Column
			}
		})
		w.Maker(func(p *tree.Plan) {
			graphicFirst := w.IndexInParent()%2 != 0 && w.SizeClass() != core.SizeCompact
			if graphicFirst {
				tree.Add(p, graphic)
			}
			tree.Add(p, func(w *core.Frame) {
				w.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
					s.Text.Align = text.Start
					s.Grow.Set(1, 1)
				})
				tree.AddChild(w, func(w *core.Text) {
					w.SetType(core.TextHeadlineLarge).SetText(title)
					w.Styler(func(s *styles.Style) {
						s.Font.Weight = rich.Bold
						s.Color = colors.Scheme.Primary.Base
					})
				})
				tree.AddChild(w, func(w *core.Text) {
					w.SetType(core.TextTitleLarge).SetText(txt)
				})
			})
			if !graphicFirst {
				tree.Add(p, graphic)
			}
		})
	})
}

func homePage(ctx *htmlcore.Context) bool {
	home = core.NewFrame(ctx.BlockParent)
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
		w.SetType(core.TextHeadlineMedium).SetText("A cross-platform framework for building powerful, fast, elegant 2D and 3D apps")
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

	initIcon := func(w *core.Icon) *core.Icon {
		w.Styler(func(s *styles.Style) {
			s.IconSize.Set(units.Dp(256))
			s.Color = colors.Scheme.Primary.Base
		})
		return w
	}

	makeBlock("CODE ONCE, RUN EVERYWHERE (CORE)", "With Cogent Core, you can write your app once and it will run on macOS, Windows, Linux, iOS, Android, and the web, automatically scaling to any screen. Instead of struggling with platform-specific code in multiple languages, you can write and maintain a single Go codebase.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Devices)
	})

	makeBlock("EFFORTLESS ELEGANCE", "Cogent Core is built on Go, a high-level language designed for building elegant, readable, and scalable code with type safety and a robust design that doesn't get in your way. Cogent Core makes it easy to get started with cross-platform app development in just two commands and three lines of simple code.", func(w *textcore.Editor) {
		w.Lines.SetLanguage(fileinfo.Go).SetString(`b := core.NewBody()
core.NewButton(b).SetText("Hello, World!")
b.RunMainWindow()`)
		w.SetReadOnly(true)
		w.Lines.Settings.LineNumbers = false
		w.Styler(func(s *styles.Style) {
			if w.SizeClass() != core.SizeCompact {
				s.Min.X.Em(20)
			}
		})
	})

	makeBlock("COMPLETELY CUSTOMIZABLE", "Cogent Core allows developers and users to customize apps to fit their needs and preferences through a robust styling system and powerful color settings.", func(w *core.Form) {
		w.SetStruct(core.AppearanceSettings)
		w.OnChange(func(e events.Event) {
			core.UpdateSettings(w, core.AppearanceSettings)
		})

	})

	makeBlock("POWERFUL FEATURES", "Cogent Core supports text editors, video players, interactive 3D graphics, customizable data plots, Markdown and HTML rendering, SVG and canvas vector graphics, and automatic views of any Go data structure for data binding and app inspection.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.ScatterPlot)
	})

	makeBlock("OPTIMIZED EXPERIENCE", "Cogent Core has editable, interactive, example-based documentation, video tutorials, command line tools, and support from the developers.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.PlayCircle)
	})

	makeBlock("EXTREMELY FAST", "Cogent Core is powered by WebGPU, a modern, cross-platform, high-performance graphics framework that allows apps to run at high speeds. Apps compile to machine code, allowing them to run without overhead.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Bolt)
	})

	makeBlock("FREE AND OPEN SOURCE", "Cogent Core is completely free and open source under the permissive BSD-3 License, allowing you to use Cogent Core for any purpose, commercially or personally.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Code)
	})

	makeBlock("USED AROUND THE WORLD", "Over seven years of development, Cogent Core has been used and tested by developers and scientists around the world for various use cases. Cogent Core is an advanced framework used to power everything from end-user apps to scientific research.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.GlobeAsia)
	})

	tree.AddChild(home, func(w *core.Text) {
		w.SetType(core.TextDisplaySmall).SetText("<b>What can Cogent Core do?</b>")
	})

	makeBlock("COGENT CODE", "Cogent Code is a Go IDE with support for syntax highlighting, code completion, symbol lookup, building and debugging, version control, keyboard shortcuts, and many other features.", func(w *core.SVG) {
		errors.Log(w.OpenFS(resources, "code-icon.svg"))
	}, "https://cogentcore.org/cogent/code")

	makeBlock("COGENT CANVAS", "Cogent Canvas is a vector graphics editor with support for shapes, paths, curves, text, images, gradients, groups, alignment, styling, importing, exporting, undo, redo, and various other features.", func(w *core.SVG) {
		errors.Log(w.OpenFS(resources, "canvas-icon.svg"))
	}, "https://cogentcore.org/cogent/canvas")

	makeBlock("COGENT LAB", "Cogent Lab is an extensible math, data science, and statistics platform and language.", func(w *core.SVG) {
		errors.Log(w.OpenFS(resources, "numbers-icon.svg"))
	}, "https://cogentcore.org/lab")

	makeBlock("COGENT MAIL", "Cogent Mail is a customizable email client with built-in Markdown support, automatic mail filtering, and keyboard shortcuts for mail filing.", func(w *core.SVG) {
		errors.Log(w.OpenFS(resources, "mail-icon.svg"))
	}, "https://github.com/cogentcore/cogent/tree/main/mail")

	makeBlock("COGENT CRAFT", "Cogent Craft is a 3D modeling app with support for creating, loading, and editing 3D object files using an interactive WYSIWYG editor.", func(w *core.SVG) {
		errors.Log(w.OpenFS(resources, "craft-icon.svg"))
	}, "https://github.com/cogentcore/cogent/tree/main/craft")

	makeBlock("EMERGENT", "Emergent is a collection of biologically based 3D neural network models of the brain that power ongoing research in computational cognitive neuroscience.", func(w *core.SVG) {
		errors.Log(w.OpenFS(resources, "emergent-icon.svg"))
	}, "https://emersim.org")

	// makeBlock("WELD", "WELD is a set of 3D computational models of a new approach to quantum physics based on the de Broglie-Bohm pilot wave theory.", func(w *core.Image) {
	// 	errors.Log(w.OpenFS(resources, "weld-icon.png"))
	// 	w.Styler(func(s *styles.Style) {
	// 		s.Min.Set(units.Dp(256))
	// 	})
	// }, "https://github.com/WaveELD/WELDBook/blob/main/textmd/ch01_intro.md")

	tree.AddChild(home, func(w *core.Text) {
		w.SetType(core.TextDisplaySmall).SetText("<b>Why Cogent Core instead of something else?</b>")
	})

	makeBlock("THE PROBLEM", "After using other frameworks built on HTML and Qt for years to make apps ranging from simple tools to complex scientific models, we realized that we were spending more time dealing with excessive boilerplate, browser inconsistencies, and dependency management issues than actual app development.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Problem)
	})

	makeBlock("THE SOLUTION", "We decided to make a framework that would allow us to focus on app content and logic by providing a consistent API that automatically handles cross-platform support, user customization, and app packaging and deployment.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Lightbulb)
	})

	makeBlock("THE RESULT", "Instead of constantly jumping through hoops to create a consistent, easy-to-use, cross-platform app, you can now take advantage of a powerful featureset on all platforms and simplify your development experience.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Check)
	})

	tree.AddChild(home, func(w *core.Button) {
		ctx.LinkButton(w, "basics")
		w.SetText("Get started")
	})

	return true
}
