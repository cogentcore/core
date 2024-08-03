// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

var (
	// TheApp is the current [App]; only one is ever in effect.
	TheApp = &App{App: system.TheApp}

	// AppAbout is the about information for the current app.
	// It is set by a linker flag in the core command line tool.
	AppAbout string

	// AppIcon is the svg icon for the current app.
	// It is set by a linker flag in the core command line tool.
	// It defaults to [icons.CogentCore] otherwise.
	AppIcon string = string(icons.CogentCore)
)

// App represents a Cogent Core app. It extends [system.App] to provide both system-level
// and high-level data and functions to do with the currently running application. The
// single instance of it is [TheApp], which embeds [system.TheApp].
type App struct { //types:add -setters
	system.App `set:"-"`

	// SceneInit is a function called on every newly created [Scene].
	// This can be used to set global configuration and styling for all
	// widgets in conjunction with [Scene.WidgetInit].
	SceneInit func(sc *Scene) `edit:"-"`
}

// appIconImagesCache is a cached version of [appIconImages].
var appIconImagesCache []image.Image

// appIconImages returns a slice of images of sizes 16x16, 32x32, and 48x48
// rendered from [AppIcon]. It returns nil if [AppIcon] is "" or if there is
// an error. It automatically logs any errors. It caches the result for future
// calls.
func appIconImages() []image.Image {
	if appIconImagesCache != nil {
		return appIconImagesCache
	}
	if AppIcon == "" {
		return nil
	}

	res := make([]image.Image, 3)

	sv := svg.NewSVG(16, 16)
	sv.Color = colors.Uniform(colors.FromRGB(66, 133, 244)) // Google Blue (#4285f4)
	err := sv.ReadXML(strings.NewReader(AppIcon))
	if errors.Log(err) != nil {
		return nil
	}

	sv.Render()
	res[0] = sv.Pixels

	sv.Resize(image.Pt(32, 32))
	sv.Render()
	res[1] = sv.Pixels

	sv.Resize(image.Pt(48, 48))
	sv.Render()
	res[2] = sv.Pixels
	appIconImagesCache = res
	return res
}

// makeAppBar configures a new top app bar in the given parent.
// It adds a back navigation button and an app chooser,
// followed by standard overflow menu items.
func makeAppBar(parent Widget) {
	tb := NewToolbar(parent)
	tb.Maker(makeStandardAppBar)
	if len(tb.Scene.AppBars) > 0 {
		tb.Makers.Normal = append(tb.Makers.Normal, tb.Scene.AppBars...)
	}
	tb.AddOverflowMenu(tb.standardOverflowMenu)
}

// makeStandardAppBar adds standard items to start of an app bar [tree.Plan].
func makeStandardAppBar(p *tree.Plan) {
	tree.AddAt(p, "back", func(w *Button) {
		w.SetIcon(icons.ArrowBack).SetKey(keymap.HistPrev).SetTooltip("Back")
		w.OnClick(func(e events.Event) {
			if slen := w.Scene.Stage.Mains.stack.Len(); slen > 1 {
				if w.Scene.Stage.CloseOnBack {
					w.Scene.Close()
				} else {
					w.Scene.Stage.Mains.stack.ValueByIndex(slen - 2).raise()
				}
				return
			}
			if wlen := len(AllRenderWindows); wlen > 1 {
				if w.Scene.Stage.CloseOnBack {
					currentRenderWindow.closeReq()
				}
				AllRenderWindows[wlen-2].Raise()
			}
		})
		// TODO(kai/abc): app bar back button disabling
		// bt.FirstStyler(func(s *styles.Style) {
		// 	if tb.Scene.Stage.Mains == nil {
		// 		return
		// 	}
		// 	s.SetState(tb.Scene.Stage.Mains.Stack.Len() <= 1 && len(AllRenderWins) <= 1, states.Disabled)
		// })
	})
}

var (
	// webCanInstall is whether the app can be installed on the web platform
	webCanInstall bool

	// webInstall installs the app on the web platform
	webInstall func()
)

// note: StandardOverflowMenu must be a method on toolbar to get context scene

// standardOverflowMenu adds standard overflow menu items for an app bar.
func (tb *Toolbar) standardOverflowMenu(m *Scene) { //types:add
	NewButton(m).SetText("Search").SetIcon(icons.Search).SetKey(keymap.Menu).SetTooltip("Search the menus").OnClick(func(e events.Event) {
		d := NewBody().AddTitle("Search")
		w := NewChooser(d).SetEditable(true).SetIcon(icons.Search)
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
		})
		w.AddItemsFunc(func() {
			for _, rw := range AllRenderWindows {
				for _, kv := range rw.mains.stack.Order {
					st := kv.Value
					// we do not include ourself
					if st == w.Scene.Stage {
						continue
					}
					w.Items = append(w.Items, ChooserItem{
						Text:    st.Title,
						Icon:    icons.Toolbar,
						Tooltip: "Show " + st.Title,
						Func:    st.raise,
					})
				}
			}
		})
		w.AddItemsFunc(func() {
			addButtonItems(&w.Items, tb, "")
		})
		w.OnFinal(events.Change, func(e events.Event) {
			d.Close()
		})
		d.AddBottomBar(func(parent Widget) {
			d.AddCancel(parent)
		})
		d.RunDialog(tb)
	})
	NewButton(m).SetText("About").SetIcon(icons.Info).OnClick(func(e events.Event) {
		d := NewBody(TheApp.Name())
		d.Styler(func(s *styles.Style) {
			s.CenterAll()
		})
		NewText(d).SetType(TextHeadlineLarge).SetText(TheApp.Name())
		if AppIcon != "" {
			errors.Log(NewSVG(d).ReadString(AppIcon))
		}
		if AppAbout != "" {
			NewText(d).SetText(AppAbout)
		}
		NewText(d).SetText("App version: " + system.AppVersion)
		NewText(d).SetText("Core version: " + system.CoreVersion)
		d.AddOKOnly().RunDialog(tb)
	})
	NewFuncButton(m).SetFunc(SettingsWindow).SetText("Settings").SetIcon(icons.Settings).SetShortcut("Command+,")
	if webCanInstall {
		icon := icons.InstallDesktop
		if TheApp.SystemPlatform().IsMobile() {
			icon = icons.InstallMobile
		}
		NewFuncButton(m).SetFunc(webInstall).SetText("Install").SetIcon(icon).SetTooltip("Install this app to your device as a Progressive Web App (PWA)")
	}
	NewButton(m).SetText("Inspect").SetIcon(icons.Edit).SetShortcut("Command+Shift+I").
		SetTooltip("Developer tools for inspecting the content of the app").
		OnClick(func(e events.Event) {
			InspectorWindow(tb.Scene)
		})
	NewButton(m).SetText("Edit").SetMenu(func(m *Scene) {
		// todo: these need to actually do something -- currently just show keyboard shortcut
		NewButton(m).SetText("Copy").SetIcon(icons.Copy).SetKey(keymap.Copy)
		NewButton(m).SetText("Cut").SetIcon(icons.Cut).SetKey(keymap.Cut)
		NewButton(m).SetText("Paste").SetIcon(icons.Paste).SetKey(keymap.Paste)
	})

	// no window menu on single-window platforms
	if TheApp.Platform().IsMobile() {
		return
	}
	NewButton(m).SetText("Window").SetMenu(func(m *Scene) {
		NewButton(m).SetText("Focus next").SetIcon(icons.CenterFocusStrong).
			SetKey(keymap.WinFocusNext).OnClick(func(e events.Event) {
			AllRenderWindows.focusNext()
		})
		NewButton(m).SetText("Minimize").SetIcon(icons.Minimize).
			OnClick(func(e events.Event) {
				win := tb.Scene.RenderWindow()
				if win != nil {
					win.minimize()
				}
			})
		NewSeparator(m)
		NewButton(m).SetText("Close window").SetIcon(icons.Close).SetKey(keymap.WinClose).
			OnClick(func(e events.Event) {
				win := tb.Scene.RenderWindow()
				if win != nil {
					win.closeReq()
				}
			})
		quit := NewButton(m).SetText("Quit").SetIcon(icons.Close).SetShortcut("Command+Q")
		quit.OnClick(func(e events.Event) {
			go TheApp.QuitReq()
		})
		quit.SetName("quit-app")
		NewSeparator(m)
		for _, w := range mainRenderWindows {
			if w != nil {
				NewButton(m).SetText(w.title).OnClick(func(e events.Event) {
					w.Raise()
				})
			}
		}
		if len(dialogRenderWindows) > 0 {
			NewSeparator(m)
			for _, w := range dialogRenderWindows {
				if w != nil {
					NewButton(m).SetText(w.title).OnClick(func(e events.Event) {
						w.Raise()
					})
				}
			}
		}
	})
}

// addButtonItems adds to the given items all of the buttons under
// the given parent. It navigates through button menus to find other
// buttons using a recursive approach that updates path with context
// about the original button menu. Consumers of this function should
// typically set path to "".
func addButtonItems(items *[]ChooserItem, parent tree.Node, path string) {
	for _, kid := range parent.AsTree().Children {
		bt := AsButton(kid)
		if bt == nil || bt.IsDisabled() {
			continue
		}
		label := bt.Text
		if label == "" {
			label = bt.Tooltip
		}
		if bt.HasMenu() {
			tmps := NewScene()
			bt.Menu(tmps)
			npath := path
			if npath != "" {
				npath += " > "
			}
			if bt.Name != "overflow-menu" {
				npath += label
			}
			addButtonItems(items, tmps, npath)
			continue
		}
		if path != "" {
			label = path + " > " + label
		}
		*items = append(*items, ChooserItem{
			Text:    label,
			Icon:    bt.Icon,
			Tooltip: bt.Tooltip,
			Func: func() {
				bt.Send(events.Click)
			},
		})
		// after the quit button, there are the render wins,
		// which we do not want to show here as we are already
		// showing the stages
		if bt.Name == "quit-app" {
			break
		}
	}
}
