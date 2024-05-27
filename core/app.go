// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
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
	// It defaults to [icons.DefaultAppIcon] otherwise.
	AppIcon string = icons.DefaultAppIcon
)

// App represents a Cogent Core app. It extends [system.App] to provide both system-level
// and high-level data and functions to do with the currently running application. The
// single instance of it is [TheApp], which embeds [system.TheApp].
type App struct { //types:add -setters
	system.App `set:"-"`

	// AppBarConfig is the function that configures the AppBar,
	// typically put in the [Scene.Bars.Top] (i.e., a TopAppBar).
	// It is set to StdAppBarConfig by default, which makes the
	// standard AppBar behavior. If this is nil, then no AppBar
	// will be created by default.
	AppBarConfig func(parent Widget)

	// SceneConfig is the function called on every newly created [core.Scene]
	// to configure it, if it is non-nil. This can be used to set global
	// configuration and styling for all widgets using the OnWidgetAdded
	// method.
	SceneConfig func(sc *Scene)
}

// AppIconImagesCache is a cached version of [AppIconImages].
var AppIconImagesCache []image.Image

// AppIconImages returns a slice of images of sizes 16x16, 32x32, and 48x48
// rendered from [AppIcon]. It returns nil if [AppIcon] is "" or if there is
// an error. It automatically logs any errors. It caches the result for future
// calls in [AppIconImagesCache].
func AppIconImages() []image.Image {
	if AppIconImagesCache != nil {
		return AppIconImagesCache
	}
	if AppIcon == "" {
		return nil
	}

	res := make([]image.Image, 3)

	sv := svg.NewSVG(16, 16)
	sv.Color = colors.C(colors.FromRGB(66, 133, 244)) // Google Blue (#4285f4)
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
	AppIconImagesCache = res
	return res
}

//////////////////////////////////////////////////////////////////////////////
//		AppBar

// StandardAppBarConfig is the standard impl for a [App.AppBarConfig].
// It adds a Back navigation buttons and the AppChooser,
// followed by the [Widget.MakeToolbar] for the current FullWindow
// Scene being viewed, along with [StandardOverflowMenu] items.
// and calls AddDefaultOverflowMenu to provide default menu items,
// which will appear below any other OverflowMenu items added.
func StandardAppBarConfig(parent Widget) {
	tb := RecycleToolbar(parent)
	tb.Maker(StandardAppBarStart)
	if len(tb.Scene.AppBars) > 0 {
		tb.Maker(tb.Scene.AppBars...)
	}
	StandardOverflowMenu(tb) // todo -- need a config option for this
}

// StandardAppBarStart adds standard items to start of an AppBar:
// [StandardAppBarBack] and [StandardAppBarChooser]
func StandardAppBarStart(p *Plan) {
	StandardAppBarBack(p)
	StandardAppBarChooser(p)
}

// StandardAppBarBack adds a back button
func StandardAppBarBack(p *Plan) {
	AddAt(p, "back", func(w *Button) {
		w.SetIcon(icons.ArrowBack).SetTooltip("Back").SetKey(keymap.HistPrev)
		w.OnClick(func(e events.Event) {
			if slen := w.Scene.Stage.Mains.Stack.Len(); slen > 1 {
				if w.Scene.Stage.CloseOnBack {
					w.Scene.Close()
				} else {
					w.Scene.Stage.Mains.Stack.ValueByIndex(slen - 2).Raise()
				}
				return
			}
			if wlen := len(AllRenderWindows); wlen > 1 {
				if w.Scene.Stage.CloseOnBack {
					CurrentRenderWindow.CloseReq()
				}
				AllRenderWindows[wlen-2].Raise()
			}
		})
		// TODO(kai/abc): app bar back button disabling
		// bt.StyleFirst(func(s *styles.Style) {
		// 	if tb.Scene.Stage.Mains == nil {
		// 		return
		// 	}
		// 	s.SetState(tb.Scene.Stage.Mains.Stack.Len() <= 1 && len(AllRenderWins) <= 1, states.Disabled)
		// })
	})
}

// StandardAppBarChooser adds a standard app chooser using [ConfigAppChooser].
func StandardAppBarChooser(p *Plan) {
	AddAt(p, "app-chooser", ConfigAppChooser)
}

// StandardOverflowMenu adds the standard overflow menu function.
func StandardOverflowMenu(tb *Toolbar) {
	tb.OverflowMenus = append(tb.OverflowMenus, tb.StandardOverflowMenu)
}

var (
	// webCanInstall is whether the app can be installed on the web platform
	webCanInstall bool

	// webInstall installs the app on the web platform
	webInstall func()
)

// note: StandardOverflowMenu must be a method on toolbar to get context scene

// StandardOverflowMenu adds standard overflow menu items.
func (tb *Toolbar) StandardOverflowMenu(m *Scene) { //types:add
	NewButton(m).SetText("About").SetIcon(icons.Info).OnClick(func(e events.Event) {
		d := NewBody(TheApp.Name())
		d.Style(func(s *styles.Style) {
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
		d.AddOKOnly().RunDialog(m)
	})
	if SettingsWindow != nil {
		NewButton(m).SetText("Settings").SetIcon(icons.Settings).SetShortcut("Command+,").
			OnClick(func(e events.Event) {
				SettingsWindow()
			})
	}
	if webCanInstall {
		icon := icons.InstallDesktop
		if TheApp.SystemPlatform().IsMobile() {
			icon = icons.InstallMobile
		}
		NewButton(m).SetText("Install").SetIcon(icon).SetTooltip("Install this app to your device as a Progressive Web App (PWA)").OnClick(func(e events.Event) {
			webInstall()
		})
	}
	if InspectorWindow != nil {
		NewButton(m).SetText("Inspect").SetIcon(icons.Edit).SetTooltip("Developer tools for inspecting the content of the app").SetShortcut("Command+Shift+I").
			OnClick(func(e events.Event) {
				InspectorWindow(tb.Scene)
			})
	}
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
			AllRenderWindows.FocusNext()
		})
		NewButton(m).SetText("Minimize").SetIcon(icons.Minimize).
			OnClick(func(e events.Event) {
				win := tb.Scene.RenderWindow()
				if win != nil {
					win.Minimize()
				}
			})
		NewSeparator(m)
		NewButton(m).SetText("Close window").SetIcon(icons.Close).SetKey(keymap.WinClose).
			OnClick(func(e events.Event) {
				win := tb.Scene.RenderWindow()
				if win != nil {
					win.CloseReq()
				}
			})
		quit := NewButton(m).SetText("Quit").SetIcon(icons.Close).SetShortcut("Command+Q").
			OnClick(func(e events.Event) {
				go TheApp.QuitReq()
			})
		quit.SetName("quit-app")
		NewSeparator(m)
		for _, w := range MainRenderWindows {
			if w != nil {
				NewButton(m).SetText(w.Title).OnClick(func(e events.Event) {
					w.Raise()
				})
			}
		}
		if len(DialogRenderWindows) > 0 {
			NewSeparator(m)
			for _, w := range DialogRenderWindows {
				if w != nil {
					NewButton(m).SetText(w.Title).OnClick(func(e events.Event) {
						w.Raise()
					})
				}
			}
		}
	})
}

//////////////////////////////////////////////////////////////////////////////
//		AppChooser

// ConfigAppChooser configures the given [Chooser] to give access
// to all app resources, such as open scenes and buttons in the
// given toolbar. This chooser is typically placed at the start
// of the AppBar. You can extend the resources available for access
// in the app chooser using [Chooser.AddItemsFunc] and [ChooserItem.Func].
func ConfigAppChooser(ch *Chooser) {
	ch.SetEditable(true).SetType(ChooserOutlined).SetIcon(icons.Search)
	if TheApp.SystemPlatform().IsMobile() {
		ch.SetPlaceholder("Search")
	} else {
		ch.SetPlaceholder(fmt.Sprintf("Search (%s)", keymap.Menu.Label()))
	}

	ch.OnWidgetAdded(func(w Widget) { // TODO(config)
		if w.PathFrom(ch) == "text-field" {
			tf := w.(*TextField)
			w.Style(func(s *styles.Style) {
				s.Background = colors.C(colors.Scheme.SurfaceContainerHighest)
				if !s.Is(states.Focused) && tf.Error == nil {
					s.Border = styles.Border{}
				}
				s.Border.Radius = styles.BorderRadiusFull
				s.Min.X.SetCustom(func(uc *units.Context) float32 {
					return min(uc.Ch(40), uc.Vw(80)-uc.Ch(20))
				})
				s.Max.X = s.Min.X
			})
		}
	})

	ch.AddItemsFunc(func() {
		for _, rw := range AllRenderWindows {
			for _, kv := range rw.Mains.Stack.Order {
				st := kv.Value
				// we do not include ourself
				if st == ch.Scene.Stage {
					continue
				}
				ch.Items = append(ch.Items, ChooserItem{
					Text:    st.Title,
					Icon:    icons.Toolbar,
					Tooltip: "Show " + st.Title,
					Func:    st.Raise,
				})
			}
		}
	})
	// todo: need tb
	// ch.AddItemsFunc(func() {
	// 	AddButtonItems(&ch.Items, tb, "")
	// })
	ch.OnFinal(events.Change, func(e events.Event) {
		// we must never have a chooser label so that it
		// always displays the search placeholder
		ch.CurrentIndex = -1
		ch.CurrentItem = ChooserItem{}
		ch.ShowCurrentItem()
	})
}

// AddButtonItems adds to the given items all of the buttons under
// the given parent. It navigates through button menus to find other
// buttons using a recursive approach that updates path with context
// about the original button menu. Consumers of this function should
// typically set path to "".
func AddButtonItems(items *[]ChooserItem, parent tree.Node, path string) {
	for _, kid := range *parent.Children() {
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
			if bt.Name() != "overflow-menu" {
				npath += label
			}
			AddButtonItems(items, tmps, npath)
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
		if bt.Name() == "quit-app" {
			break
		}
	}
}
