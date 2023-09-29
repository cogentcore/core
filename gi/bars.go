// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/goosi"
	"goki.dev/goosi/key"
	"goki.dev/goosi/mouse"
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
	OSMainMenus map[string]*Action `json:"-" xml:"-" desc:"map of main menu items for callback from OS main menu (MacOS specific)"`
}

// event functions for this type
var MenuBarEventFuncs WidgetEvents

func (mb *MenuBar) OnInit() {
	mb.AddEvents(&MenuBarEventFuncs)
	mb.AddStyler(func(w *WidgetBase, s *styles.Style) {
		s.MaxWidth.SetPx(-1)
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
		s.Color = colors.Scheme.OnSurface
	})
}

func (mb *MenuBar) CopyFieldsFrom(frm any) {
	fr := frm.(*MenuBar)
	mb.Layout.CopyFieldsFrom(&fr.Layout)
	mb.MainMenu = fr.MainMenu
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
	wi := mb.This().(Widget)
	if !mb.ShowMenuBar() {
		return
	}
	if mb.PushBounds(sc) {
		wi.FilterEvents()
		mb.MenuBarStdRender(sc)
		mb.RenderScrolls(sc)
		mb.RenderChildren(sc)
		mb.PopBounds(sc)
	}
}

// UpdateActions calls UpdateFunc on all actions in menu -- individual menus
// are automatically updated just prior to menu popup
func (mb *MenuBar) UpdateActions() {
	if mb == nil {
		return
	}
	for _, mi := range mb.Kids {
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
			ac.UpdateActions()
		}
	}
}

// SetShortcuts sets the shortcuts to window associated with Toolbar
// Called in AddEvents()
func (mb *MenuBar) SetShortcuts() {
	win := mb.ParentRenderWin()
	if win == nil {
		return
	}
	for _, mi := range mb.Kids {
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
			win.AddShortcut(ac.Shortcut, ac)
		}
	}
}

func (mb *MenuBar) Destroy() {
	mb.DeleteShortcuts()
	mb.Layout.Destroy()
}

// DeleteShortcuts deletes the shortcuts -- called when destroyed
func (mb *MenuBar) DeleteShortcuts() {
	win := mb.ParentRenderWin()
	if win == nil {
		return
	}
	for _, mi := range mb.Kids {
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
			win.DeleteShortcut(ac.Shortcut, ac)
		}
	}
}

// FindActionByName finds an action on the menu, or any sub-menu, with given
// name (exact match) -- this is not the Text label but the Name of the
// element (for AddAction items, this is the same as Label or Icon (if Label
// is empty)) -- returns false if not found
func (m *MenuBar) FindActionByName(name string) (*Action, bool) {
	if m == nil {
		return nil, false
	}
	for _, mi := range m.Kids {
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
			if ac.Name() == name {
				return ac, true
			}
			if ac.Menu != nil {
				if sac, ok := ac.Menu.FindActionByName(name); ok {
					return sac, ok
				}
			}
		}
	}
	return nil, false
}

// ConfigMenus configures Action items as children of MenuBar with the given
// names, which function as the main menu panels for the menu bar (File, Edit,
// etc).  Access the resulting menus as .ChildByName("name").(*Action).
func (mb *MenuBar) ConfigMenus(menus []string) {
	if mb == nil {
		return
	}
	sz := len(menus)
	tnl := make(ki.Config, sz+1)
	typ := ActionType // note: could pass in action type to make it more flexible, but..
	for i, m := range menus {
		tnl[i].Type = typ
		tnl[i].Name = m
	}
	tnl[sz].Type = StretchType
	tnl[sz].Name = "menstr"
	_, updt := mb.ConfigChildren(tnl)
	for i, m := range menus {
		mi := mb.Kids[i]
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
			ac.SetText(m)
			ac.SetAsMenu()
		}
	}
	mb.UpdateEnd(updt)
}

/*

todo:

// MainMenuFunc is the callback function for OS-generated menu actions.
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

	mb.UpdateActions()
	osmm.SetFunc(MainMenuFunc)

	mm := osmm.StartUpdate() // locks
	osmm.Reset(mm)
	mb.OSMainMenus = make(map[string]*Action, 100)
	for _, mi := range mb.Kids {
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
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
func (mb *MenuBar) SetMainMenuSub(osmm goosi.MainMenu, subm goosi.Menu, am *Action) {
	for i, mi := range am.Menu {
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
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
// active state of corresponding Actions (action self-update functions are
// called via UpdateActions) -- can be called by method of same name on
// RenderWin.
func (mb *MenuBar) MainMenuUpdateActives(win *RenderWin) {
	osmm := win.GoosiWin.MainMenu()
	if osmm == nil { // no OS main menu
		return
	}

	mb.UpdateActions()
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

type ToolBarEmbedder interface {
	AsToolBar() *ToolBar
}

func AsToolBar(k ki.Ki) *ToolBar {
	if k == nil || k.This() == nil {
		return nil
	}
	if ac, ok := k.(ToolBarEmbedder); ok {
		return ac.AsToolBar()
	}
	return nil
}

func (tb *ToolBar) AsToolBar() *ToolBar {
	return tb
}

// ToolBar is a Layout (typically LayoutHoriz) that renders a gradient
// background and is useful for holding Actions that do things
type ToolBar struct { //goki:embedder
	Layout
}

// event functions for this type
var ToolBarEventFuncs WidgetEvents

func (tb *ToolBar) OnInit() {
	tb.AddEvents(&ToolBarEventFuncs)
	tb.AddStyler(func(w *WidgetBase, s *styles.Style) {
		s.MaxWidth.SetPx(-1)
		s.Border.Radius = styles.BorderRadiusFull
	})
}

func (tb *ToolBar) CopyFieldsFrom(frm any) {
	fr := frm.(*ToolBar)
	tb.Layout.CopyFieldsFrom(&fr.Layout)
}

// AddAction adds an action to the toolbar using given options, and connects
// the action signal to given receiver object and function, along with given
// data which is stored on the action and then passed in the action signal.
// Optional updateFunc is a function called prior to showing the menu to
// update the actions (enabled or not typically).
func (tb *ToolBar) AddAction(opts ActOpts, sigTo ki.Ki, fun func()) *Action {
	nm := opts.Name
	if nm == "" {
		nm = opts.Label
	}
	if nm == "" {
		nm = string(opts.Icon)
	}
	ac := NewAction(tb, nm)
	ac.Text = opts.Label
	ac.Icon = icons.Icon(opts.Icon)
	ac.Tooltip = opts.Tooltip
	ac.Shortcut = key.Chord(opts.Shortcut).OSShortcut()
	if opts.ShortcutKey != KeyFunNil {
		ac.Shortcut = ShortcutForFun(opts.ShortcutKey)
	}
	ac.Data = opts.Data
	ac.UpdateFunc = opts.UpdateFunc
	// if sigTo != nil && fun != nil {
	// 	ac.ActionSig.Connect(sigTo, fun)
	// }
	return ac
}

// AddSeparator adds a new separator to the toolbar -- automatically sets orientation
// depending on layout.  All nodes need a name identifier.
func (tb *ToolBar) AddSeparator(sepnm string) *Separator {
	sp := NewSeparator(tb, sepnm)
	if tb.Lay == LayoutHoriz {
		sp.Horiz = false
	} else {
		sp.Horiz = true
	}
	return sp
}

// ToolBarStdRender does the standard rendering of the bar
func (tb *ToolBar) ToolBarStdRender(sc *Scene) {
	rs, pc, st := tb.RenderLock(sc)
	pos := tb.LayState.Alloc.Pos
	sz := tb.LayState.Alloc.Size
	bg := st.BackgroundColor
	if st.BackgroundColor.IsNil() {
		bg = tb.ParentBackgroundColor()
	}
	pc.FillBox(rs, pos, sz, &bg)
	tb.RenderUnlock(rs)
}

func (tb *ToolBar) Render(sc *Scene) {
	if len(tb.Kids) == 0 { // todo: check for mac menu and don't render -- also need checks higher up
		return
	}
	wi := tb.This().(Widget)
	if tb.PushBounds(sc) {
		wi.FilterEvents()
		tb.ToolBarStdRender(sc)
		tb.RenderScrolls(sc)
		tb.RenderChildren(sc)
		tb.PopBounds(sc)
	}
}

func (tb *ToolBar) MouseFocusEvent(we *WidgetEvents) {
	we.AddFunc(goosi.MouseFocusEvent, HiPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.Event)
		if me.Action == mouse.Enter {
			tbb := AsToolBar(recv)
			if tbb != nil {
				tbb.UpdateActions()
			}
			// do NOT mark as processed -- HiPri and not mutex
		}
	})
}

func (tb *ToolBar) AddEvents(we *WidgetEvents) {
	if we.HasFuncs() {
		return
	}
	tb.Layout.AddEvents(we)
	tb.MouseFocusEvent(we)
}

func (tb *ToolBar) FilterEvents() {
	tb.SetShortcuts()
}

// SetShortcuts sets the shortcuts to window associated with Toolbar
// Called in AddEvents()
func (tb *ToolBar) SetShortcuts() {
	win := tb.ParentRenderWin()
	if win == nil {
		return
	}
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
			win.AddShortcut(ac.Shortcut, ac)
		}
	}
}

func (tb *ToolBar) Destroy() {
	tb.DeleteShortcuts()
	tb.Layout.Destroy()
}

// DeleteShortcuts deletes the shortcuts -- called when destroyed
func (tb *ToolBar) DeleteShortcuts() {
	win := tb.ParentRenderWin()
	if win == nil {
		return
	}
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
			win.DeleteShortcut(ac.Shortcut, ac)
		}
	}
}

// UpdateActions calls UpdateFunc on all actions in toolbar -- individual
// menus are automatically updated just prior to menu popup
func (tb *ToolBar) UpdateActions() {
	if tb == nil {
		return
	}
	updt := tb.UpdateStart()
	defer tb.UpdateEnd(updt)
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
			ac.UpdateActions()
		}
	}
}

// FindActionByName finds an action on the toolbar, or any sub-menu, with
// given name (exact match) -- this is not the Text label but the Name of the
// element (for AddAction items, this is the same as Label or Icon (if Label
// is empty)) -- returns false if not found
func (tb *ToolBar) FindActionByName(name string) (*Action, bool) {
	for _, mi := range tb.Kids {
		if mi.KiType().HasEmbed(ActionType) {
			ac := AsAction(mi)
			if ac.Name() == name {
				return ac, true
			}
			if ac.Menu != nil {
				if sac, ok := ac.Menu.FindActionByName(name); ok {
					return sac, ok
				}
			}
		}
	}
	return nil, false
}
