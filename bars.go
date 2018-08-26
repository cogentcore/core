// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"sync"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
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

var MenuBarProps = ki.Props{
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(0, units.Px),
	"spacing":          units.NewValue(4, units.Px),
	"color":            &Prefs.Colors.Font,
	"background-color": "linear-gradient(pref(Control), highlight-10)",
}

// MenuBarStdRender does the standard rendering of the bar
func (mb *MenuBar) MenuBarStdRender() {
	st := &mb.Sty
	rs := &mb.Viewport.Render
	pc := &rs.Paint

	pos := mb.LayData.AllocPos
	sz := mb.LayData.AllocSize
	pc.FillBox(rs, pos, sz, &st.Font.BgColor)
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
		mb.LayoutEvents()
		mb.RenderScrolls()
		mb.Render2DChildren()
		mb.PopBounds()
	} else {
		mb.DisconnectAllEvents(AllPris) // uses both Low and Hi
	}
}

// UpdateActions calls UpdateFunc on all actions in menu -- individual menus
// are automatically updated just prior to menu popup
func (g *MenuBar) UpdateActions() {
	for _, mi := range g.Kids {
		if mi.TypeEmbeds(KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
			ac.UpdateActions()
		}
	}
}

// FindActionByName finds an action on the menu, or any sub-menu, with given
// name (exact match) -- this is not the Text label but the Name of the
// element (for AddAction items, this is the same as Label or Icon (if Label
// is empty)) -- returns false if not found
func (m *MenuBar) FindActionByName(name string) (*Action, bool) {
	for _, mi := range m.Kids {
		if mi.TypeEmbeds(KiT_Action) {
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
// etc).  Access the resulting menus as .KnownChildByName("name").(*Action).
func (mb *MenuBar) ConfigMenus(menus []string) {
	sz := len(menus)
	tnl := make(kit.TypeAndNameList, sz+1)
	typ := KiT_Action // note: could pass in action type to make it more flexible, but..
	for i, m := range menus {
		tnl[i].Type = typ
		tnl[i].Name = m
	}
	tnl[sz].Type = KiT_Stretch
	tnl[sz].Name = "menstr"
	_, updt := mb.ConfigChildren(tnl, false)
	for i, m := range menus {
		mi := mb.Kids[i]
		if mi.TypeEmbeds(KiT_Action) {
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

// mainMenuMu protects updating of the central main menu, which is not
// concurrent safe and different windows run on different threads
var mainMenuMu sync.Mutex

// SetMainMenu sets this menubar as the main menu of given window -- called by
// Window.MainMenuUpdated.
func (mb *MenuBar) SetMainMenu(win *Window) {
	osmm := win.OSWin.MainMenu()
	if osmm == nil { // no OS main menu
		return
	}

	mainMenuMu.Lock()
	defer mainMenuMu.Unlock()

	mb.UpdateActions()
	osmm.SetFunc(MainMenuFunc)
	mm := osmm.Menu()
	osmm.Reset(mm)
	mb.OSMainMenus = make(map[string]*Action, 100)
	for _, mi := range mb.Kids {
		if mi.TypeEmbeds(KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
			subm := osmm.AddSubMenu(mm, ac.Text)
			mb.SetMainMenuSub(osmm, subm, ac)
		}
	}
}

// SetMainMenuSub iterates over sub-menus, adding items to overall main menu.
func (mb *MenuBar) SetMainMenuSub(osmm oswin.MainMenu, subm oswin.Menu, am *Action) {
	for i, mi := range am.Menu {
		if mi.TypeEmbeds(KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
			if len(ac.Menu) > 0 {
				ssubm := osmm.AddSubMenu(subm, ac.Text)
				mb.SetMainMenuSub(osmm, ssubm, ac)
			} else {
				mid := osmm.AddItem(subm, ac.Text, ac.Shortcut, i, ac.IsActive())
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

	mainMenuMu.Lock()
	defer mainMenuMu.Unlock()

	mb.UpdateActions()
	if mb.OSMainMenus == nil {
		return
	}
	for _, ma := range mb.OSMainMenus {
		mid, ok := ma.Prop("__OSMainMenuItemID")
		if !ok {
			continue
		}
		osmm.SetItemActive(mid.(oswin.MenuItem), ma.IsActive())
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

var ToolBarProps = ki.Props{
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(0, units.Px),
	"spacing":          units.NewValue(4, units.Px),
	"color":            &Prefs.Colors.Font,
	"background-color": "linear-gradient(pref(Control), highlight-10)",
}

// AddAction adds an action to the toolbar using given options, and connects
// the action signal to given receiver object and function, along with given
// data which is stored on the action and then passed in the action signal.
// Optional updateFunc is a function called prior to showing the menu to
// update the actions (enabled or not typically).
func (tb *ToolBar) AddAction(opts ActOpts, sigTo ki.Ki, fun ki.RecvFunc) *Action {
	nm := opts.Label
	if nm == "" {
		nm = opts.Icon
	}
	ac := tb.AddNewChild(KiT_Action, nm).(*Action)
	ac.Text = opts.Label
	ac.Icon = IconName(opts.Icon)
	ac.Shortcut = OSShortcut(opts.Shortcut)
	ac.Data = opts.Data
	ac.UpdateFunc = opts.UpdateFunc
	if sigTo != nil && fun != nil {
		ac.ActionSig.Connect(sigTo, fun)
	}
	return ac
}

// ToolBarStdRender does the standard rendering of the bar
func (tb *ToolBar) ToolBarStdRender() {
	st := &tb.Sty
	rs := &tb.Viewport.Render
	pc := &rs.Paint

	pos := tb.LayData.AllocPos
	sz := tb.LayData.AllocSize
	pc.FillBox(rs, pos, sz, &st.Font.BgColor)
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
		tb.ToolBarEvents()
		tb.RenderScrolls()
		tb.Render2DChildren()
		tb.PopBounds()
	} else {
		tb.DisconnectAllEvents(AllPris) // uses both Low and Hi
	}
}

func (tb *ToolBar) ToolBarEvents() {
	tb.LayoutEvents()
	tb.ConnectEvent(oswin.MouseFocusEvent, HiPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.FocusEvent)
		if me.Action == mouse.Enter {
			tbb := recv.Embed(KiT_ToolBar).(*ToolBar)
			tbb.UpdateActions()
			// do NOT mark as processed -- HiPri and not mutex
		}
	})
}

// UpdateActions calls UpdateFunc on all actions in toolbar -- individual
// menus are automatically updated just prior to menu popup
func (tb *ToolBar) UpdateActions() {
	for _, mi := range tb.Kids {
		if mi.TypeEmbeds(KiT_Action) {
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
		if mi.TypeEmbeds(KiT_Action) {
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
