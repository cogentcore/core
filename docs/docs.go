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
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/pages"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/yaegicore"
	"cogentcore.org/core/yaegicore/symbols"
)

//go:embed content
var content embed.FS

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
	pg := pages.NewPage(b).SetContent(content)
	htmlcore.WikilinkBaseURL = "cogentcore.org/core"
	b.AddAppBar(pg.MakeToolbar)
	b.AddAppBar(func(p *tree.Plan) {
		tree.Add(p, func(w *core.Button) {
			w.SetText("Setup").SetIcon(icons.Download)
			w.OnClick(func(e events.Event) {
				pg.Context.OpenURL("/setup")
			})
		})
		tree.Add(p, func(w *core.Button) {
			w.SetText("Playground").SetIcon(icons.PlayCircle)
			w.OnClick(func(e events.Event) {
				pg.Context.OpenURL("/playground")
			})
		})
		tree.Add(p, func(w *core.Button) {
			w.SetText("Videos").SetIcon(icons.VideoLibrary)
			w.OnClick(func(e events.Event) {
				pg.Context.OpenURL("https://youtube.com/@CogentCore")
			})
		})
		tree.Add(p, func(w *core.Button) {
			w.SetText("Blog").SetIcon(icons.RssFeed)
			w.OnClick(func(e events.Event) {
				pg.Context.OpenURL("https://cogentcore.org/blog")
			})
		})
		tree.Add(p, func(w *core.Button) {
			w.SetText("GitHub").SetIcon(icons.GitHub)
			w.OnClick(func(e events.Event) {
				pg.Context.OpenURL("https://github.com/cogentcore/core")
			})
		})
		tree.Add(p, func(w *core.Button) {
			w.SetText("Sponsor").SetIcon(icons.Favorite)
			w.OnClick(func(e events.Event) {
				pg.Context.OpenURL("https://github.com/sponsors/cogentcore")
			})
		})
		tree.Add(p, func(w *core.Button) {
			w.SetText("Contact").SetIcon(icons.Mail)
			w.OnClick(func(e events.Event) {
				core.MessageDialog(w, "contact@cogentcore.org", "Contact us via email at")
			})
		})
	})

	symbols.Symbols["."]["content"] = reflect.ValueOf(content)
	symbols.Symbols["."]["myImage"] = reflect.ValueOf(myImage)
	symbols.Symbols["."]["mySVG"] = reflect.ValueOf(mySVG)
	symbols.Symbols["."]["myFile"] = reflect.ValueOf(myFile)

	htmlcore.ElementHandlers["home-page"] = homePage
	htmlcore.ElementHandlers["core-playground"] = func(ctx *htmlcore.Context) bool {
		splits := core.NewSplits(ctx.BlockParent)
		ed := texteditor.NewEditor(splits)
		playgroundFile := filepath.Join(core.TheApp.AppDataDir(), "playground.go")
		err := ed.Buffer.Open(core.Filename(playgroundFile))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				err := os.WriteFile(playgroundFile, []byte(defaultPlaygroundCode), 0666)
				core.ErrorSnackbar(ed, err, "Error creating code file")
				if err == nil {
					err := ed.Buffer.Open(core.Filename(playgroundFile))
					core.ErrorSnackbar(ed, err, "Error loading code")
				}
			} else {
				core.ErrorSnackbar(ed, err, "Error loading code")
			}
		}
		ed.OnChange(func(e events.Event) {
			core.ErrorSnackbar(ed, ed.Buffer.Save(), "Error saving code")
		})
		parent := core.NewFrame(splits)
		yaegicore.BindTextEditor(ed, parent)
		return true
	}
	htmlcore.ElementHandlers["style-demo"] = func(ctx *htmlcore.Context) bool {
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
				s.Min.Set(units.Px(sz.X), units.Px(sz.Y))
				s.Background = colors.Scheme.Primary.Base
			})
		}
		return true
	}

	b.RunMainWindow()
}

var home *core.Frame

func makeBlock[T tree.NodeValue](title, text string, graphic func(w *T), url ...string) {
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
					s.Text.Align = styles.Start
					s.Grow.Set(1, 1)
				})
				tree.AddChild(w, func(w *core.Text) {
					w.SetType(core.TextHeadlineLarge).SetText(title)
					w.Styler(func(s *styles.Style) {
						s.Font.Weight = styles.WeightBold
						s.Color = colors.Scheme.Primary.Base
					})
				})
				tree.AddChild(w, func(w *core.Text) {
					w.SetType(core.TextTitleLarge).SetText(text)
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
	tree.AddChild(home, func(w *core.Button) {
		w.SetText("Get started")
		w.OnClick(func(e events.Event) {
			ctx.OpenURL("basics")
		})
	})

	initIcon := func(w *core.Icon) *core.Icon {
		w.Styler(func(s *styles.Style) {
			s.Min.Set(units.Dp(256))
			s.Color = colors.Scheme.Primary.Base
		})
		return w
	}

	makeBlock("CODE ONCE, RUN EVERYWHERE (CORE)", "With Cogent Core, you can write your app once and it will instantly run on macOS, Windows, Linux, iOS, Android, and the web, automatically scaling to any screen. Instead of struggling with platform-specific code in a multitude of languages, you can easily write and maintain a single Go codebase.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Devices)
	})

	makeBlock("EFFORTLESS ELEGANCE", "Cogent Core is built on Go, a high-level language designed for building elegant, readable, and scalable code with full type safety and a robust design that never gets in your way. Cogent Core makes it easy to get started with cross-platform app development in just two commands and seven lines of simple code.", func(w *texteditor.Editor) {
		w.Buffer.SetLanguage(fileinfo.Go).SetString(`b := core.NewBody()
core.NewButton(b).SetText("Hello, World!")
b.RunMainWindow()`)
		w.SetReadOnly(true)
		w.Buffer.Options.LineNumbers = false
		w.Styler(func(s *styles.Style) {
			if w.SizeClass() != core.SizeCompact {
				s.Min.X.Em(20)
			}
		})
	})

	makeBlock("COMPLETELY CUSTOMIZABLE", "Cogent Core allows developers and users to fully customize apps to fit their unique needs and preferences through a robust styling system and a powerful color system that allow developers and users to instantly customize every aspect of the appearance and behavior of an app.", func(w *core.Form) {
		w.SetStruct(core.AppearanceSettings)
		w.OnChange(func(e events.Event) {
			core.UpdateSettings(w, core.AppearanceSettings)
		})

	})

	makeBlock("POWERFUL FEATURES", "Cogent Core comes with a powerful set of advanced features that allow you to make almost anything, including fully featured text editors, video and audio players, interactive 3D graphics, customizable data plots, Markdown and HTML rendering, SVG and canvas vector graphics, and automatic views of any Go data structure for instant data binding and advanced app inspection.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.ScatterPlot)
	})

	makeBlock("OPTIMIZED EXPERIENCE", "Every part of your development experience is guided by a comprehensive set of editable interactive example-based documentation, in-depth video tutorials, easy-to-use command line tools specialized for Cogent Core, and active support and development from the Cogent Core developers.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.PlayCircle)
	})

	makeBlock("EXTREMELY FAST", "Cogent Core is powered by Vulkan, a modern, cross-platform, high-performance graphics framework that allows apps to run on all platforms at extremely fast speeds. All Cogent Core apps compile to machine code, allowing them to run without any overhead.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Bolt)
	})

	makeBlock("FREE AND OPEN SOURCE", "Cogent Core is completely free and open source under the permissive BSD-3 License, allowing you to use Cogent Core for any purpose, commercially or personally. We believe that software works best when everyone can use it.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Code)
	})

	makeBlock("USED AROUND THE WORLD", "Over six years of development, Cogent Core has been used and thoroughly tested by developers and scientists around the world for a wide variety of use cases. Cogent Core is an advanced framework actively used to power everything from end-user apps to scientific research.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.GlobeAsia)
	})

	tree.AddChild(home, func(w *core.Text) {
		w.SetType(core.TextDisplaySmall).SetText("<b>What can Cogent Core do?</b>")
	})

	makeBlock("COGENT CODE", "Cogent Code is a fully featured Go IDE with support for syntax highlighting, code completion, symbol lookup, building and debugging, version control, keyboard shortcuts, and many other features.", func(w *core.SVG) {
		errors.Log(w.OpenFS(resources, "code-icon.svg"))
	}, "https://cogentcore.org/cogent/code")

	makeBlock("COGENT CANVAS", "Cogent Canvas is a powerful vector graphics editor with complete support for shapes, paths, curves, text, images, gradients, groups, alignment, styling, importing, exporting, undo, redo, and various other features.", func(w *core.SVG) {
		errors.Log(w.OpenFS(resources, "canvas-icon.svg"))
	}, "https://cogentcore.org/cogent/canvas")

	makeBlock("COGENT NUMBERS", "Cogent Numbers is a highly extensible math, data science, and statistics platform that combines the power of programming with the convenience of spreadsheets and graphing calculators.", func(w *core.SVG) {
		errors.Log(w.OpenFS(resources, "numbers-icon.svg"))
	}, "https://github.com/cogentcore/cogent/tree/main/numbers")

	makeBlock("COGENT MAIL", "Cogent Mail is a customizable email client with built-in Markdown support, automatic mail filtering, and an extensive set of keyboard shortcuts for advanced mail filing.", func(w *core.SVG) {
		errors.Log(w.OpenFS(resources, "mail-icon.svg"))
	}, "https://github.com/cogentcore/cogent/tree/main/mail")

	makeBlock("COGENT CRAFT", "Cogent Craft is a powerful 3D modeling app with support for creating, loading, and editing 3D object files using an interactive WYSIWYG editor.", func(w *core.SVG) {
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

	makeBlock("THE SOLUTION", "We decided to make a framework that would allow us to focus on app content and logic by providing a consistent and elegant API that automatically handles cross-platform support, user customization, and app packaging and deployment.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Lightbulb)
	})

	makeBlock("THE RESULT", "Instead of constantly jumping through hoops to create a consistent, easy-to-use, cross-platform app, you can now take advantage of a powerful featureset on all platforms and vastly simplify your development experience.", func(w *core.Icon) {
		initIcon(w).SetIcon(icons.Check)
	})

	tree.AddChild(home, func(w *core.Button) {
		w.SetText("Get started")
		w.OnClick(func(e events.Event) {
			ctx.OpenURL("basics")
		})
	})

	return true
}
