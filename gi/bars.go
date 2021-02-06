// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// MenuBar

// MenuBar is a Layout (typically LayoutHoriz) that renders a gradient
// background and has convenience methods for adding menus.
type MenuBar struct {
	Layout
	MainMenu    bool               `desc:"is this the main menu bar for a window?  controls whether displayed on macOS"`
	OSMainMenus map[string]*Action `json:"-" xml:"-" desc:"map of main menu items for callback from OS main menu (MacOS specific)"`
}

var KiT_MenuBar = kit.Types.AddType(&MenuBar{}, MenuBarProps)

// AddNewMenuBar adds a new menubar to given parent node, with given name.
func AddNewMenuBar(parent ki.Ki, name string) *MenuBar {
	return parent.AddNewChild(KiT_MenuBar, name).(*MenuBar)
}

func (mb *MenuBar) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*MenuBar)
	mb.Layout.CopyFieldsFrom(&fr.Layout)
	mb.MainMenu = fr.MainMenu
}

var MenuBarProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
	"padding":          units.NewPx(2),
	"margin":           units.NewPx(0),
	"spacing":          units.NewPx(4),
	"color":            &Prefs.Colors.Font,
	"background-color": "linear-gradient(pref(Control), highlight-10)",
}

// MenuBarStdRender does the standard rendering of the bar
func (mb *MenuBar) MenuBarStdRender() {
	rs, pc, st := mb.RenderLock()
	pos := mb.LayState.Alloc.Pos
	sz := mb.LayState.Alloc.Size
	pc.FillBox(rs, pos, sz, &st.Font.BgColor)
	mb.RenderUnlock(rs)
}

func (mb *MenuBar) ShowMenuBar() bool {
	if len(mb.Kids) == 0 {
		return false
	}
	if mb.MainMenu {
		if oswin.TheApp.Platform() == oswin.MacOS && !LocalMainMenu {
			return false
		}
	}
	return true
}

func (mb *MenuBar) Size2D(iter int) {
	if !mb.ShowMenuBar() {
		return
	}
	mb.Layout.Size2D(iter)
}

func (mb *MenuBar) Layout2D(parBBox image.Rectangle, iter int) bool {
	if !mb.ShowMenuBar() {
		return false
	}
	return mb.Layout.Layout2D(parBBox, iter)
}

func (mb *MenuBar) Render2D() {
	if !mb.ShowMenuBar() {
		return
	}
	if mb.PushBounds() {
		mb.MenuBarStdRender()
		mb.This().(Node2D).ConnectEvents2D()
		mb.RenderScrolls()
		mb.Render2DChildren()
		mb.PopBounds()
	} else {
		mb.DisconnectAllEvents(AllPris) // uses both Low and Hi
	}
}

// UpdateActions calls UpdateFunc on all actions in menu -- individual menus
// are automatically updated just prior to menu popup
func (mb *MenuBar) UpdateActions() {
	if mb == nil {
		return
	}
	for _, mi := range mb.Kids {
		if ki.TypeEmbeds(mi, KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
			ac.UpdateActions()
		}
	}
}

// SetShortcuts sets the shortcuts to window associated with Toolbar
// Called in ConnectEvents2D()
func (mb *MenuBar) SetShortcuts() {
	win := mb.ParentWindow()
	if win == nil {
		return
	}
	for _, k := range mb.Kids {
		if ki.TypeEmbeds(k, KiT_Action) {
			ac := k.Embed(KiT_Action).(*Action)
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
	win := mb.ParentWindow()
	if win == nil {
		return
	}
	for _, k := range mb.Kids {
		if ki.TypeEmbeds(k, KiT_Action) {
			ac := k.Embed(KiT_Action).(*Action)
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
		if ki.TypeEmbeds(mi, KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
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
	tnl := make(kit.TypeAndNameList, sz+1)
	typ := KiT_Action // note: could pass in action type to make it more flexible, but..
	for i, m := range menus {
		tnl[i].Type = typ
		tnl[i].Name = m
	}
	tnl[sz].Type = KiT_Stretch
	tnl[sz].Name = "menstr"
	_, updt := mb.ConfigChildren(tnl)
	for i, m := range menus {
		mi := mb.Kids[i]
		if ki.TypeEmbeds(mi, KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
			ac.SetText(m)
			ac.SetAsMenu()
		}
	}
	mb.UpdateEnd(updt)
}

// MainMenuFunc is the callback function for OS-generated menu actions.
func MainMenuFunc(owin oswin.Window, title string, tag int) {
	win, ok := owin.Parent().(*Window)
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
// on this MenuBar -- called by Window.MainMenuUpdated.
func (mb *MenuBar) UpdateMainMenu(win *Window) {
	osmm := win.OSWin.MainMenu()
	if osmm == nil { // no OS main menu
		return
	}

	mb.UpdateActions()
	osmm.SetFunc(MainMenuFunc)
	mm := osmm.StartUpdate() // locks
	osmm.Reset(mm)
	mb.OSMainMenus = make(map[string]*Action, 100)
	for _, mi := range mb.Kids {
		if ki.TypeEmbeds(mi, KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
			subm := osmm.AddSubMenu(mm, ac.Text)
			mb.SetMainMenuSub(osmm, subm, ac)
		}
	}
	osmm.EndUpdate(mm) // unlocks
}

// SetMainMenu sets this menu as the current OS-specific, separate main menu
// for given window -- only should be called in window.Focus event.
// Does nothing if menu is empty.
func (mb *MenuBar) SetMainMenu(win *Window) {
	osmm := win.OSWin.MainMenu()
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
func (mb *MenuBar) SetMainMenuSub(osmm oswin.MainMenu, subm oswin.Menu, am *Action) {
	for i, mi := range am.Menu {
		if ki.TypeEmbeds(mi, KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
			if len(ac.Menu) > 0 {
				ssubm := osmm.AddSubMenu(subm, ac.Text)
				mb.SetMainMenuSub(osmm, ssubm, ac)
			} else {
				mid := osmm.AddItem(subm, ac.Text, string(ac.Shortcut), i, ac.IsActive())
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
// Window.
func (mb *MenuBar) MainMenuUpdateActives(win *Window) {
	osmm := win.OSWin.MainMenu()
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
		osmm.SetItemActive(mid.(oswin.MenuItem), ma.IsActive()) // assuming this is threadsafe
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// ToolBar

// ToolBar is a Layout (typically LayoutHoriz) that renders a gradient
// background and is useful for holding Actions that do things
type ToolBar struct {
	Layout
}

var KiT_ToolBar = kit.Types.AddType(&ToolBar{}, ToolBarProps)

// AddNewToolBar adds a new toolbar to given parent node, with given name.
func AddNewToolBar(parent ki.Ki, name string) *ToolBar {
	return parent.AddNewChild(KiT_ToolBar, name).(*ToolBar)
}

func (tb *ToolBar) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*ToolBar)
	tb.Layout.CopyFieldsFrom(&fr.Layout)
}

var ToolBarProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
	"padding":          units.NewPx(2),
	"margin":           units.NewPx(0),
	"spacing":          units.NewPx(4),
	"color":            &Prefs.Colors.Font,
	"background-color": "linear-gradient(pref(Control), highlight-10)",
}

// AddAction adds an action to the toolbar using given options, and connects
// the action signal to given receiver object and function, along with given
// data which is stored on the action and then passed in the action signal.
// Optional updateFunc is a function called prior to showing the menu to
// update the actions (enabled or not typically).
func (tb *ToolBar) AddAction(opts ActOpts, sigTo ki.Ki, fun ki.RecvFunc) *Action {
	nm := opts.Name
	if nm == "" {
		nm = opts.Label
	}
	if nm == "" {
		nm = opts.Icon
	}
	ac := AddNewAction(tb, nm)
	ac.Text = opts.Label
	ac.Icon = IconName(opts.Icon)
	ac.Tooltip = opts.Tooltip
	ac.Shortcut = key.Chord(opts.Shortcut).OSShortcut()
	if opts.ShortcutKey != KeyFunNil {
		ac.Shortcut = ShortcutForFun(opts.ShortcutKey)
	}
	ac.Data = opts.Data
	ac.UpdateFunc = opts.UpdateFunc
	if sigTo != nil && fun != nil {
		ac.ActionSig.Connect(sigTo, fun)
	}
	return ac
}

// AddSeparator adds a new separator to the toolbar -- automatically sets orientation
// depending on layout.  All nodes need a name identifier.
func (tb *ToolBar) AddSeparator(sepnm string) *Separator {
	sp := AddNewSeparator(tb, sepnm, false)
	if tb.Lay == LayoutHoriz {
		sp.Horiz = false
	} else {
		sp.Horiz = true
	}
	return sp
}

// ToolBarStdRender does the standard rendering of the bar
func (tb *ToolBar) ToolBarStdRender() {
	rs, pc, st := tb.RenderLock()
	pos := tb.LayState.Alloc.Pos
	sz := tb.LayState.Alloc.Size
	pc.FillBox(rs, pos, sz, &st.Font.BgColor)
	tb.RenderUnlock(rs)
}

func (tb *ToolBar) Render2D() {
	if len(tb.Kids) == 0 { // todo: check for mac menu and don't render -- also need checks higher up
		return
	}
	if tb.FullReRenderIfNeeded() {
		return
	}
	if tb.PushBounds() {
		tb.ToolBarStdRender()
		tb.This().(Node2D).ConnectEvents2D()
		tb.RenderScrolls()
		tb.Render2DChildren()
		tb.PopBounds()
	} else {
		tb.DisconnectAllEvents(AllPris) // uses both Low and Hi
	}
}

func (tb *ToolBar) MouseFocusEvent() {
	tb.ConnectEvent(oswin.MouseFocusEvent, HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.FocusEvent)
		if me.Action == mouse.Enter {
			tbb := recv.Embed(KiT_ToolBar).(*ToolBar)
			tbb.UpdateActions()
			// do NOT mark as processed -- HiPri and not mutex
		}
	})
}

func (tb *ToolBar) ConnectEvents2D() {
	tb.Layout.ConnectEvents2D()
	tb.MouseFocusEvent()
	tb.SetShortcuts()
}

// SetShortcuts sets the shortcuts to window associated with Toolbar
// Called in ConnectEvents2D()
func (tb *ToolBar) SetShortcuts() {
	win := tb.ParentWindow()
	if win == nil {
		return
	}
	for _, k := range tb.Kids {
		if ki.TypeEmbeds(k, KiT_Action) {
			ac := k.Embed(KiT_Action).(*Action)
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
	win := tb.ParentWindow()
	if win == nil {
		return
	}
	for _, k := range tb.Kids {
		if ki.TypeEmbeds(k, KiT_Action) {
			ac := k.Embed(KiT_Action).(*Action)
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
	if tb.ParentWindow() != nil {
		wupdt := tb.TopUpdateStart()
		defer tb.TopUpdateEnd(wupdt)
	}
	for _, mi := range tb.Kids {
		if ki.TypeEmbeds(mi, KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
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
		if ki.TypeEmbeds(mi, KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
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
