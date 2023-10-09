// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
	"goki.dev/icons"
	"goki.dev/ki/v2"
)

////////////////////////////////////////////////////////////////////////////////////////
// MenuBar

// MenuBar is a Layout (typically LayoutHoriz) that renders a gradient
// background and has convenience methods for adding menus.
type MenuBar struct {
	Layout

	// is this the main menu bar for a window?  controls whether displayed on macOS
	MainMenu bool `desc:"is this the main menu bar for a window?  controls whether displayed on macOS"`

	// map of main menu items for callback from OS main menu (MacOS specific)
	OSMainMenus map[string]*Button `json:"-" xml:"-" desc:"map of main menu items for callback from OS main menu (MacOS specific)"`
}

func (mb *MenuBar) CopyFieldsFrom(frm any) {
	fr := frm.(*MenuBar)
	mb.Layout.CopyFieldsFrom(&fr.Layout)
	mb.MainMenu = fr.MainMenu
}

func (mb *MenuBar) OnInit() {
	mb.LayoutHandlers()
	mb.MenuBarStyles()
}

func (mb *MenuBar) MenuBarStyles() {
	mb.AddStyles(func(s *styles.Style) {
		s.MaxWidth.SetDp(-1)
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
		s.Color = colors.Scheme.OnSurface
	})
}

// MenuBarStdRender does the standard rendering of the bar
func (mb *MenuBar) MenuBarStdRender(sc *Scene) {
	rs, pc, st := mb.RenderLock(sc)
	pos := mb.LayState.Alloc.Pos
	sz := mb.LayState.Alloc.Size
	pc.FillBox(rs, pos, sz, &st.BackgroundColor)
	mb.RenderUnlock(rs)
}

func (mb *MenuBar) ShowMenuBar() bool {
	if len(mb.Kids) == 0 {
		return false
	}
	if mb.MainMenu {
		if goosi.TheApp.Platform() == goosi.MacOS && !LocalMainMenu {
			return false
		}
	}
	return true
}

func (mb *MenuBar) GetSize(sc *Scene, iter int) {
	if !mb.ShowMenuBar() {
		return
	}
	mb.Layout.GetSize(sc, iter)
}

func (mb *MenuBar) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	if !mb.ShowMenuBar() {
		return false
	}
	return mb.Layout.DoLayout(sc, parBBox, iter)
}

func (mb *MenuBar) Render(sc *Scene) {
	if !mb.ShowMenuBar() {
		return
	}
	if mb.PushBounds(sc) {
		mb.MenuBarStdRender(sc)
		mb.RenderScrolls(sc)
		mb.RenderChildren(sc)
		mb.PopBounds(sc)
	}
}

// UpdateButtons calls UpdateFunc on all buttons in menu -- individual menus
// are automatically updated just prior to menu popup
func (mb *MenuBar) UpdateButtons() {
	if mb == nil {
		return
	}
	for _, mi := range mb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			ac.UpdateButtons()
		}
	}
}

// SetShortcuts sets the shortcuts to window associated with Toolbar
// Called in SetTypeHandlers()
func (mb *MenuBar) SetShortcuts() {
	em := mb.EventMgr()
	if em == nil {
		return
	}
	for _, mi := range mb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			em.AddShortcut(ac.Shortcut, ac)
		}
	}
}

func (mb *MenuBar) Destroy() {
	mb.DeleteShortcuts()
	mb.Layout.Destroy()
}

// DeleteShortcuts deletes the shortcuts -- called when destroyed
func (mb *MenuBar) DeleteShortcuts() {
	em := mb.EventMgr()
	if em == nil {
		return
	}
	for _, mi := range mb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			em.DeleteShortcut(ac.Shortcut, ac)
		}
	}
}

// FindButtonByName finds an button on the menu, or any sub-menu, with given
// name (exact match) -- this is not the Text label but the Name of the
// element (for AddButton items, this is the same as Label or Icon (if Label
// is empty)) -- returns false if not found
func (m *MenuBar) FindButtonByName(name string) (*Button, bool) {
	if m == nil {
		return nil, false
	}
	for _, mi := range m.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			if ac.Name() == name {
				return ac, true
			}
			if ac.Menu != nil {
				if sac, ok := ac.Menu.FindButtonByName(name); ok {
					return sac, ok
				}
			}
		}
	}
	return nil, false
}

// ConfigMenus configures Button items as children of MenuBar with the given
// names, which function as the main menu panels for the menu bar (File, Edit,
// etc).  Access the resulting menus as .ChildByName("name").(*Button).
func (mb *MenuBar) ConfigMenus(menus []string) {
	if mb == nil {
		return
	}
	sz := len(menus)
	tnl := make(ki.Config, sz+1)
	typ := ButtonType // note: could pass in button type to make it more flexible, but..
	for i, m := range menus {
		tnl[i].Type = typ
		tnl[i].Name = m
	}
	tnl[sz].Type = StretchType
	tnl[sz].Name = "menstr"
	_, updt := mb.ConfigChildren(tnl)
	for i, m := range menus {
		mi := mb.Kids[i]
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			ac.SetText(m)
			ac.SetAsMenu()
		}
	}
	mb.UpdateEnd(updt)
}

/*

todo:

// MainMenuFunc is the callback function for OS-generated menu buttons.
func MainMenuFunc(owin goosi.Window, title string, tag int) {
	win, ok := owin.Parent().(*RenderWin)
	if !ok {
		return
	}
	mb := win.MainMenu
	if mb == nil {
		return
	}
	if mb.OSMainMenus == nil {
		return
	}
	ma, ok := mb.OSMainMenus[title]
	if !ok {
		return
	}
	// fmt.Printf("triggering OS main menu: %v\n", title)
	ma.Trigger()
}

// UpdateMainMenu updates the OS-specific, separate main menu of given window based
// on this MenuBar -- called by RenderWin.MainMenuUpdated.
func (mb *MenuBar) UpdateMainMenu(win *RenderWin) {
	osmm := win.GoosiWin.MainMenu()
	if osmm == nil { // no OS main menu
		return
	}

	mb.UpdateButtons()
	osmm.SetFunc(MainMenuFunc)

	mm := osmm.StartUpdate() // locks
	osmm.Reset(mm)
	mb.OSMainMenus = make(map[string]*Button, 100)
	for _, mi := range mb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			subm := osmm.AddSubMenu(mm, ac.Text)
			mb.SetMainMenuSub(osmm, subm, ac)
		}
	}
	osmm.EndUpdate(mm) // unlocks
}

// SetMainMenu sets this menu as the current OS-specific, separate main menu
// for given window -- only should be called in window.Focus event.
// Does nothing if menu is empty.
func (mb *MenuBar) SetMainMenu(win *RenderWin) {
	osmm := win.GoosiWin.MainMenu()
	if osmm == nil { // no OS main menu
		return
	}
	if len(mb.Kids) == 0 {
		return
	}

	if mb.OSMainMenus == nil {
		mb.UpdateMainMenu(win)
	}
	osmm.SetMenu()
}

// SetMainMenuSub iterates over sub-menus, adding items to overall main menu.
func (mb *MenuBar) SetMainMenuSub(osmm goosi.MainMenu, subm goosi.Menu, am *Button) {
	for i, mi := range am.Menu {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			if len(ac.Menu) > 0 {
				ssubm := osmm.AddSubMenu(subm, ac.Text)
				mb.SetMainMenuSub(osmm, ssubm, ac)
			} else {
				mid := osmm.AddItem(subm, ac.Text, string(ac.Shortcut), i, !ac.IsDisabled())
				mb.OSMainMenus[ac.Text] = ac
				ac.SetProp("__OSMainMenuItemID", mid)
			}
		} else if _, ok := mi.(*Separator); ok {
			osmm.AddSeparator(subm)
		}
	}
}

// MainMenuUpdateActives updates the active state of all menu items, based on
// active state of corresponding Buttons (button self-update functions are
// called via UpdateButtons) -- can be called by method of same name on
// RenderWin.
func (mb *MenuBar) MainMenuUpdateActives(win *RenderWin) {
	osmm := win.GoosiWin.MainMenu()
	if osmm == nil { // no OS main menu
		return
	}

	mb.UpdateButtons()
	if mb.OSMainMenus == nil {
		return
	}
	for _, ma := range mb.OSMainMenus {
		mid, err := ma.PropTry("__OSMainMenuItemID")
		if err != nil {
			continue
		}
		osmm.SetItemActive(mid.(goosi.MenuItem), !ma.IsDisabled()) // assuming this is threadsafe
	}
}

*/

////////////////////////////////////////////////////////////////////////////////////////
// ToolBar

// ToolBar is a [Frame] that is useful for holding [Button]s that do things.
//
//goki:embedder
type ToolBar struct {
	Frame
}

func (tb *ToolBar) CopyFieldsFrom(frm any) {
	fr := frm.(*ToolBar)
	tb.Frame.CopyFieldsFrom(&fr.Frame)
}

func (tb *ToolBar) OnInit() {
	tb.ToolBarStyles()
	tb.LayoutHandlers()
}

func (tb *ToolBar) ToolBarStyles() {
	tb.AddStyles(func(s *styles.Style) {
		s.MaxWidth.SetDp(-1)
		s.Border.Radius = styles.BorderRadiusFull
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
		s.Margin.Set(units.Dp(4 * Prefs.DensityMul()))
	})
}

// AddButton adds an button to the toolbar using given options, and connects
// the button signal to given receiver object and function, along with given
// data which is stored on the button and then passed in the button signal.
// Optional updateFunc is a function called prior to showing the menu to
// update the buttons (enabled or not typically).
func (tb *ToolBar) AddButton(opts ActOpts, fun func(act *Button)) *Button {
	nm := opts.Name
	if nm == "" {
		nm = opts.Label
	}
	if nm == "" {
		nm = string(opts.Icon)
	}
	ac := NewButton(tb, nm)
	ac.Text = opts.Label
	ac.Icon = icons.Icon(opts.Icon)
	ac.Tooltip = opts.Tooltip
	ac.Shortcut = key.Chord(opts.Shortcut).OSShortcut()
	if opts.ShortcutKey != KeyFunNil {
		ac.Shortcut = ShortcutForFun(opts.ShortcutKey)
	}
	ac.Data = opts.Data
	ac.UpdateFunc = opts.UpdateFunc
	if fun != nil {
		ac.On(events.Click, func(e events.Event) {
			fun(ac)
		})
	}
	return ac
}

// AddSeparator adds a new separator to the toolbar. It automatically
// sets its orientation depending on the layout of the toolbar. The
// name does not need to be specified, and will default to "separator-"
// plus the [ki.Ki.NumLifetimeChildren] of the toolbar.
func (tb *ToolBar) AddSeparator(name ...string) *Separator {
	sp := NewSeparator(tb, name...)
	sp.Horiz = tb.Lay != LayoutHoriz
	return sp
}

// SetShortcuts sets the shortcuts to window associated with Toolbar
func (tb *ToolBar) SetShortcuts() {
	em := tb.EventMgr()
	if em == nil {
		return
	}
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			em.AddShortcut(ac.Shortcut, ac)
		}
	}
}

func (tb *ToolBar) Destroy() {
	tb.DeleteShortcuts()
	tb.Frame.Destroy()
}

// DeleteShortcuts deletes the shortcuts -- called when destroyed
func (tb *ToolBar) DeleteShortcuts() {
	em := tb.EventMgr()
	if em == nil {
		return
	}
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			em.DeleteShortcut(ac.Shortcut, ac)
		}
	}
}

// UpdateButtons calls UpdateFunc on all buttons in toolbar -- individual
// menus are automatically updated just prior to menu popup
func (tb *ToolBar) UpdateButtons() {
	if tb == nil {
		return
	}
	updt := tb.UpdateStart()
	defer tb.UpdateEnd(updt)
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			ac.UpdateButtons()
		}
	}
}

// FindButtonByName finds an button on the toolbar, or any sub-menu, with
// given name (exact match) -- this is not the Text label but the Name of the
// element (for AddButton items, this is the same as Label or Icon (if Label
// is empty)) -- returns false if not found
func (tb *ToolBar) FindButtonByName(name string) (*Button, bool) {
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ButtonType) {
			ac := AsButton(mi)
			if ac.Name() == name {
				return ac, true
			}
			if ac.Menu != nil {
				if sac, ok := ac.Menu.FindButtonByName(name); ok {
					return sac, ok
				}
			}
		}
	}
	return nil, false
}
