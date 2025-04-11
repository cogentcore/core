// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

// StyleMenuScene configures the default styles
// for the given pop-up menu frame with the given parent.
// It should be called on menu frames when they are created.
func StyleMenuScene(msc *Scene) {
	msc.Styler(func(s *styles.Style) {
		s.Grow.Set(0, 0)
		s.Padding.Set(units.Dp(2))
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Background = colors.Scheme.SurfaceContainer
		s.BoxShadow = styles.BoxShadow2()
		s.Gap.Zero()
	})
	msc.SetOnChildAdded(func(n tree.Node) {
		if bt := AsButton(n); bt != nil {
			bt.Type = ButtonMenu
			bt.OnKeyChord(func(e events.Event) {
				kf := keymap.Of(e.KeyChord())
				switch kf {
				case keymap.MoveRight:
					if bt.openMenu(e) {
						e.SetHandled()
					}
				case keymap.MoveLeft:
					// need to be able to use arrow keys to navigate in completer
					if msc.Stage.Type != CompleterStage {
						msc.Stage.ClosePopup()
						e.SetHandled()
					}
				}
			})
			return
		}
		if sp, ok := n.(*Separator); ok {
			sp.Styler(func(s *styles.Style) {
				s.Direction = styles.Row
			})
		}
	})
}

// newMenuScene constructs a [Scene] for displaying a menu, using the
// given menu constructor function. If no name is provided, it defaults
// to "menu".  If no menu items added, returns nil.
func newMenuScene(menu func(m *Scene), name ...string) *Scene {
	nm := "menu"
	if len(name) > 0 {
		nm = name[0] + "-menu"
	}
	msc := NewScene(nm)
	StyleMenuScene(msc)
	menu(msc)
	if !msc.HasChildren() {
		return nil
	}

	hasSelected := false
	msc.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if cw == msc {
			return tree.Continue
		}
		if bt := AsButton(cw); bt != nil {
			if bt.Menu == nil {
				bt.handleClickDismissMenu()
			}
		}
		if !hasSelected && cwb.StateIs(states.Selected) {
			// fmt.Println("start focus sel:", cwb)
			msc.Events.SetStartFocus(cwb)
			hasSelected = true
		}
		return tree.Continue
	})
	if !hasSelected && msc.HasChildren() {
		// fmt.Println("start focus first:", msc.Child(0).(Widget))
		msc.Events.SetStartFocus(msc.Child(0).(Widget))
	}
	return msc
}

// NewMenuStage returns a new Menu stage with given scene contents,
// in connection with given widget, which provides key context
// for constructing the menu, at given RenderWindow position
// (e.g., use ContextMenuPos or WinPos method on ctx Widget).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use Run call at the end to start the Stage running.
func NewMenuStage(sc *Scene, ctx Widget, pos image.Point) *Stage {
	if sc == nil || !sc.HasChildren() {
		return nil
	}
	st := NewPopupStage(MenuStage, sc, ctx)
	if pos != (image.Point{}) {
		st.Pos = pos
	}
	return st
}

// NewMenu returns a new menu stage based on the given menu constructor
// function, in connection with given widget, which provides key context
// for constructing the menu at given RenderWindow position
// (e.g., use ContextMenuPos or WinPos method on ctx Widget).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use Run call at the end to start the Stage running.
func NewMenu(menu func(m *Scene), ctx Widget, pos image.Point) *Stage {
	return NewMenuStage(newMenuScene(menu, ctx.AsTree().Name), ctx, pos)
}

// AddContextMenu adds the given context menu to [WidgetBase.ContextMenus].
// It is the main way that code should modify a widget's context menus.
// Context menu functions are run in reverse order, and separators are
// automatically added between each context menu function. [Scene.ContextMenus]
// apply to all widgets in the scene.
func (wb *WidgetBase) AddContextMenu(menu func(m *Scene)) {
	wb.ContextMenus = append(wb.ContextMenus, menu)
}

// applyContextMenus adds the [WidgetBase.ContextMenus] and [Scene.ContextMenus]
// to the given menu scene in reverse order. It also adds separators between each
// context menu function.
func (wb *WidgetBase) applyContextMenus(m *Scene) {
	do := func(cms []func(m *Scene)) {
		for i := len(cms) - 1; i >= 0; i-- {
			if m.NumChildren() > 0 {
				NewSeparator(m)
			}
			cms[i](m)
		}
	}
	do(wb.ContextMenus)
	if wb.This != wb.Scene {
		do(wb.Scene.ContextMenus)
	}
}

// ContextMenuPos returns the default position for the context menu
// upper left corner.  The event will be from a mouse ContextMenu
// event if non-nil: should handle both cases.
func (wb *WidgetBase) ContextMenuPos(e events.Event) image.Point {
	if e != nil {
		return e.WindowPos()
	}
	return wb.winPos(.5, .5) // center
}

func (wb *WidgetBase) handleWidgetContextMenu() {
	wb.On(events.ContextMenu, func(e events.Event) {
		wi := wb.This.(Widget)
		wi.ShowContextMenu(e)
	})
}

func (wb *WidgetBase) ShowContextMenu(e events.Event) {
	e.SetHandled() // always
	if wb == nil || wb.This == nil {
		return
	}
	wi := wb.This.(Widget)
	nm := NewMenu(wi.AsWidget().applyContextMenus, wi, wi.ContextMenuPos(e))
	if nm == nil { // no items
		return
	}
	nm.Run()
}

// NewMenuFromStrings constructs a new menu from given list of strings,
// calling the given function with the index of the selected string.
// if string == sel, that menu item is selected initially.
func NewMenuFromStrings(strs []string, sel string, fun func(idx int)) *Scene {
	return newMenuScene(func(m *Scene) {
		for i, s := range strs {
			b := NewButton(m).SetText(s)
			b.OnClick(func(e events.Event) {
				fun(i)
			})
			if s == sel {
				b.SetSelected(true)
			}
		}
	})
}

var (
	// webCanInstall is whether the app can be installed on the web platform
	webCanInstall bool

	// webInstall installs the app on the web platform
	webInstall func()
)

// MenuSearcher is an interface that [Widget]s can implement
// to customize the items of the menu search chooser created
// by the default [Scene] context menu in [Scene.MenuSearchDialog].
type MenuSearcher interface {
	MenuSearch(items *[]ChooserItem)
}

// standardContextMenu adds standard context menu items for the [Scene].
func (sc *Scene) standardContextMenu(m *Scene) { //types:add
	msdesc := "Search for menu buttons and other app actions"
	NewButton(m).SetText("Menu search").SetIcon(icons.Search).SetKey(keymap.Menu).SetTooltip(msdesc).OnClick(func(e events.Event) {
		sc.MenuSearchDialog("Menu search", msdesc)
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
		d.AddOKOnly().NewDialog(sc).SetDisplayTitle(false).Run()
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
			InspectorWindow(sc)
		})

	// No window menu on mobile platforms
	if TheApp.Platform().IsMobile() && TheApp.Platform() != system.Web {
		return
	}
	NewButton(m).SetText("Window").SetMenu(func(m *Scene) {
		if sc.IsFullscreen() {
			NewButton(m).SetText("Exit fullscreen").SetIcon(icons.Fullscreen).OnClick(func(e events.Event) {
				sc.SetFullscreen(false)
			})
		} else {
			NewButton(m).SetText("Fullscreen").SetIcon(icons.Fullscreen).OnClick(func(e events.Event) {
				sc.SetFullscreen(true)
			})
		}
		// Only do fullscreen on web
		if TheApp.Platform() == system.Web {
			return
		}
		NewButton(m).SetText("Focus next").SetIcon(icons.CenterFocusStrong).
			SetKey(keymap.WinFocusNext).OnClick(func(e events.Event) {
			AllRenderWindows.focusNext()
		})
		NewButton(m).SetText("Minimize").SetIcon(icons.Minimize).
			OnClick(func(e events.Event) {
				win := sc.RenderWindow()
				if win != nil {
					win.minimize()
				}
			})
		NewSeparator(m)
		NewButton(m).SetText("Close window").SetIcon(icons.Close).SetKey(keymap.WinClose).
			OnClick(func(e events.Event) {
				win := sc.RenderWindow()
				if win != nil {
					win.closeReq()
				}
			})
		quit := NewButton(m).SetText("Quit").SetIcon(icons.Close).SetShortcut("Command+Q")
		quit.OnClick(func(e events.Event) {
			go TheApp.QuitReq()
		})
	})
}

// MenuSearchDialog runs the menu search dialog for the scene with
// the given title and description text. It includes scenes, toolbar buttons,
// and [MenuSearcher]s.
func (sc *Scene) MenuSearchDialog(title, text string) {
	d := NewBody(title)
	NewText(d).SetType(TextSupporting).SetText(text)
	w := NewChooser(d).SetEditable(true).SetIcon(icons.Search)
	w.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	w.AddItemsFunc(func() {
		for _, rw := range AllRenderWindows {
			for _, kv := range rw.mains.stack.Order {
				st := kv.Value
				// we do not include ourself
				if st == sc.Stage || st == w.Scene.Stage {
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
		addButtonItems(&w.Items, sc, "")
		tmps := NewScene()
		sc.applyContextMenus(tmps)
		addButtonItems(&w.Items, tmps, "")
	})
	w.OnFinal(events.Change, func(e events.Event) {
		d.Close()
	})
	d.AddBottomBar(func(bar *Frame) {
		d.AddCancel(bar)
	})
	d.RunDialog(sc)
}

// addButtonItems adds to the given items all of the buttons under
// the given parent. It navigates through button menus to find other
// buttons using a recursive approach that updates path with context
// about the original button menu. Consumers of this function should
// typically set path to "".
func addButtonItems(items *[]ChooserItem, parent tree.Node, path string) {
	parent.AsTree().WalkDown(func(n tree.Node) bool {
		if ms, ok := n.(MenuSearcher); ok {
			ms.MenuSearch(items)
		}
		bt := AsButton(n)
		if bt == nil || bt.IsDisabled() {
			return tree.Continue
		}
		_, isTb := bt.Parent.(*Toolbar)
		_, isSc := bt.Parent.(*Scene)
		if !isTb && !isSc {
			return tree.Continue
		}
		if bt.Text == "Menu search" {
			return tree.Continue
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
			return tree.Continue
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
		return tree.Continue
	})
}
