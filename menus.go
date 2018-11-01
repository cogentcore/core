// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/kit"
)

// Menu is a slice list of Node2D actions, which can contain sub-actions
// (though it can contain anything -- it is just added to a column layout and
// displayed in a popup) -- don't use stretchy sizes in general for these
// items!
type Menu ki.Slice

func (m Menu) MarshalJSON() ([]byte, error) {
	ks := (ki.Slice)(m)
	return ks.MarshalJSON()
}

func (m *Menu) UnmarshalJSON(b []byte) error {
	ks := (*ki.Slice)(m)
	return ks.UnmarshalJSON(b)
}

// MakeMenuFunc is a callback for making a menu on demand, receives the object
// calling this function (typically an Action or Button) and the menu
type MakeMenuFunc func(obj ki.Ki, m *Menu)

// ActOpts provides named and partial parameters for AddAction method
type ActOpts struct {
	Label       string
	Icon        string
	Tooltip     string
	Shortcut    key.Chord
	ShortcutKey KeyFuns
	Data        interface{}
	UpdateFunc  func(act *Action)
}

// SetAction sets properties of given action
func (m *Menu) SetAction(ac *Action, opts ActOpts, sigTo ki.Ki, fun ki.RecvFunc) {
	nm := opts.Label
	if nm == "" {
		nm = opts.Icon
	}
	ac.InitName(ac, nm)
	ac.Text = opts.Label
	ac.Tooltip = opts.Tooltip
	ac.Icon = IconName(opts.Icon)
	ac.Shortcut = key.Chord(opts.Shortcut).OSShortcut()
	if opts.ShortcutKey != KeyFunNil {
		ac.Shortcut = ShortcutForFun(opts.ShortcutKey)
		// todo: need a flag for menu-based?
	}
	ac.Data = opts.Data
	ac.UpdateFunc = opts.UpdateFunc
	ac.SetAsMenu()
	if sigTo != nil && fun != nil {
		ac.ActionSig.Connect(sigTo, fun)
	}
}

// AddAction adds an action to the menu using given options, and connects the
// action signal to given receiver object and function, along with given data
// which is stored on the action and then passed in the action signal.
// Optional updateFunc is a function called prior to showing the menu to
// update the actions (enabled or not typically).
func (m *Menu) AddAction(opts ActOpts, sigTo ki.Ki, fun ki.RecvFunc) *Action {
	if m == nil {
		*m = make(Menu, 0, 10)
	}
	ac := &Action{}
	m.SetAction(ac, opts, sigTo, fun)
	*m = append(*m, ac.This().(Node2D))
	return ac
}

// InsertActionBefore adds an action to the menu before existing item of given
// name, using given options, and connects the action signal to given receiver
// object and function, along with given data which is stored on the action
// and then passed in the action signal.  Optional updateFunc is a function
// called prior to showing the menu to update the actions (enabled or not
// typically).  If name not found, adds to end of list..
func (m *Menu) InsertActionBefore(before string, opts ActOpts, sigTo ki.Ki, fun ki.RecvFunc) *Action {
	sl := (*[]ki.Ki)(m)
	if idx, got := ki.SliceIndexByName(sl, before, 0); got {
		ac := &Action{}
		m.SetAction(ac, opts, sigTo, fun)
		ki.SliceInsert(sl, ac.This(), idx)
		return ac
	} else {
		return m.AddAction(opts, sigTo, fun)
	}
}

// InsertActionAfter adds an action to the menu after existing item of given
// name, using given options, and connects the action signal to given receiver
// object and function, along with given data which is stored on the action
// and then passed in the action signal.  Optional updateFunc is a function
// called prior to showing the menu to update the actions (enabled or not
// typically).  If name not found, adds to end of list..
func (m *Menu) InsertActionAfter(after string, opts ActOpts, sigTo ki.Ki, fun ki.RecvFunc) *Action {
	sl := (*[]ki.Ki)(m)
	if idx, got := ki.SliceIndexByName(sl, after, 0); got {
		ac := &Action{}
		m.SetAction(ac, opts, sigTo, fun)
		ki.SliceInsert(sl, ac.This(), idx+1)
		return ac
	} else {
		return m.AddAction(opts, sigTo, fun)
	}
}

// AddSeparator adds a separator at the next point in the menu (name is just
// internal label of element, defaults to 'sep' if empty)
func (m *Menu) AddSeparator(name string) *Separator {
	if m == nil {
		*m = make(Menu, 0, 10)
	}
	sp := Separator{}
	if name == "" {
		name = "sep"
	}
	sp.InitName(&sp, name)
	sp.Horiz = true
	*m = append(*m, sp.This().(Node2D))
	return &sp
}

// AddLabel adds a label to the menu
func (m *Menu) AddLabel(lbl string) *Label {
	if m == nil {
		*m = make(Menu, 0, 10)
	}
	lb := Label{}
	lb.InitName(&lb, lbl)
	lb.SetText(lbl)
	lb.SetProp("background-color", &Prefs.Colors.Control)
	*m = append(*m, lb.This().(Node2D))
	return &lb
}

// SetShortcuts sets the shortcuts to given window -- call when the menu has
// been attached to a window
func (m *Menu) SetShortcuts(win *Window) {
	if win == nil {
		return
	}
	for _, mi := range *m {
		if mi.TypeEmbeds(KiT_Action) {
			ac := mi.Embed(KiT_Action).(*Action)
			win.AddShortcut(ac.Shortcut, ac)
		}
	}
}

// UpdateActions calls update function on all the actions in the menu, and any
// of their sub-actions
func (m *Menu) UpdateActions() {
	for _, mi := range *m {
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
func (m *Menu) FindActionByName(name string) (*Action, bool) {
	for _, mi := range *m {
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

////////////////////////////////////////////////////////////////////////////////////////
// Standard menu elements

// AddCopyCutPaste adds a Copy, Cut, Paste actions that just emit the
// corresponding keyboard shortcut.  Paste is automatically enabled by
// clipboard having something in it.
func (m *Menu) AddCopyCutPaste(win *Window) {
	cpsc := ActiveKeyMap.ChordForFun(KeyFunCopy)
	ctsc := ActiveKeyMap.ChordForFun(KeyFunCut)
	ptsc := ActiveKeyMap.ChordForFun(KeyFunPaste)
	m.AddAction(ActOpts{Label: "Copy", Shortcut: cpsc},
		win, func(recv, send ki.Ki, sig int64, data interface{}) {
			ww := recv.Embed(KiT_Window).(*Window)
			ww.SendKeyFunEvent(KeyFunCopy, false) // false = ignore popups -- don't send to menu
		})
	m.AddAction(ActOpts{Label: "Cut", Shortcut: ctsc},
		win, func(recv, send ki.Ki, sig int64, data interface{}) {
			ww := recv.Embed(KiT_Window).(*Window)
			ww.SendKeyFunEvent(KeyFunCut, false) // false = ignore popups -- don't send to menu
		})
	m.AddAction(ActOpts{Label: "Paste", Shortcut: ptsc,
		UpdateFunc: func(ac *Action) {
			ac.SetInactiveState(oswin.TheApp.ClipBoard(win.OSWin).IsEmpty())
		}}, win, func(recv, send ki.Ki, sig int64, data interface{}) {
		ww := recv.Embed(KiT_Window).(*Window)
		ww.SendKeyFunEvent(KeyFunPaste, false) // false = ignore popups -- don't send to menu
	})
}

// AddCopyCutPasteDupe adds a Copy, Cut, Paste, and Duplicate actions that
// just emit the corresponding keyboard shortcut.  Paste is automatically
// enabled by clipboard having something in it.
func (m *Menu) AddCopyCutPasteDupe(win *Window) {
	m.AddCopyCutPaste(win)
	dpsc := ActiveKeyMap.ChordForFun(KeyFunDuplicate)
	m.AddAction(ActOpts{Label: "Duplicate", Shortcut: dpsc},
		win, func(recv, send ki.Ki, sig int64, data interface{}) {
			ww := recv.Embed(KiT_Window).(*Window)
			ww.SendKeyFunEvent(KeyFunDuplicate, false) // false = ignore popups -- don't send to menu
		})
}

// CustomAppMenuFunc is a function called by AddAppMenu after the
// AddStdAppMenu is called -- apps can set this function to add / modify / etc
// the menu
var CustomAppMenuFunc = (func(m *Menu, win *Window))(nil)

// AddAppMenu adds an "app" menu to the menu -- calls AddStdAppMenu and then
// CustomAppMenuFunc if non-nil
func (m *Menu) AddAppMenu(win *Window) {
	m.AddStdAppMenu(win)
	if CustomAppMenuFunc != nil {
		CustomAppMenuFunc(m, win)
	}
}

// AddStdAppMenu adds a standard set of menu items for application-level control.
func (m *Menu) AddStdAppMenu(win *Window) {
	aboutitle := "About " + oswin.TheApp.Name()
	m.AddAction(ActOpts{Label: aboutitle},
		win, func(recv, send ki.Ki, sig int64, data interface{}) {
			ww := recv.Embed(KiT_Window).(*Window)
			PromptDialog(ww.Viewport, DlgOpts{Title: aboutitle, Prompt: oswin.TheApp.About()}, true, false, nil, nil)
		})
	m.AddAction(ActOpts{Label: "GoGi Preferences...", Shortcut: "Command+P"},
		win, func(recv, send ki.Ki, sig int64, data interface{}) {
			TheViewIFace.PrefsView(&Prefs)
		})
	m.AddSeparator("sepq")
	m.AddAction(ActOpts{Label: "Quit", Shortcut: "Command+Q"},
		win, func(recv, send ki.Ki, sig int64, data interface{}) {
			oswin.TheApp.QuitReq()
		})
}

// AddWindowsMenu adds menu items for current main and dialog windows.
// must be called under WindowGlobalMu mutex lock!
func (m *Menu) AddWindowsMenu(win *Window) {
	m.AddAction(ActOpts{Label: "Minimize"},
		win, func(recv, send ki.Ki, sig int64, data interface{}) {
			ww := recv.Embed(KiT_Window).(*Window)
			ww.OSWin.Minimize()
		})
	m.AddAction(ActOpts{Label: "Focus Next", ShortcutKey: KeyFunWinFocusNext},
		win, func(recv, send ki.Ki, sig int64, data interface{}) {
			AllWindows.FocusNext()
		})
	m.AddSeparator("sepa")
	for _, w := range MainWindows {
		if w != nil {
			m.AddAction(ActOpts{Label: w.Title},
				w, func(recv, send ki.Ki, sig int64, data interface{}) {
					ww := recv.Embed(KiT_Window).(*Window)
					ww.OSWin.Raise()
				})
		}
	}
	if len(DialogWindows) > 0 {
		m.AddSeparator("sepw")
		for _, w := range DialogWindows {
			if w != nil {
				m.AddAction(ActOpts{Label: w.Title},
					w, func(recv, send ki.Ki, sig int64, data interface{}) {
						ww := recv.Embed(KiT_Window).(*Window)
						ww.OSWin.Raise()
					})
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// PopupMenu function

var MenuFrameProps = ki.Props{
	"border-width":        units.NewValue(0, units.Px),
	"border-color":        "none",
	"margin":              units.NewValue(4, units.Px),
	"padding":             units.NewValue(2, units.Px),
	"box-shadow.h-offset": units.NewValue(2, units.Px),
	"box-shadow.v-offset": units.NewValue(2, units.Px),
	"box-shadow.blur":     units.NewValue(2, units.Px),
	"box-shadow.color":    &Prefs.Colors.Shadow,
}

// MenuMaxHeight is the maximum height of any menu popup panel in units of font height
// scroll bars are enforced beyond that size.
var MenuMaxHeight = 30

// PopupMenu pops up a viewport with a layout that draws the supplied actions
// positions are relative to given viewport -- name is relevant base name to
// which Menu is appended
func PopupMenu(menu Menu, x, y int, parVp *Viewport2D, name string) *Viewport2D {
	win := parVp.Win
	mainVp := win.Viewport
	if len(menu) == 0 {
		log.Printf("GoGi PopupMenu: empty menu given\n")
		return nil
	}

	menu.UpdateActions()

	pvp := Viewport2D{}
	pvp.InitName(&pvp, name+"Menu")
	pvp.Win = win
	updt := pvp.UpdateStart()
	pvp.SetProp("color", &Prefs.Colors.Font)
	pvp.Fill = true
	pvp.SetFlag(int(VpFlagPopup))
	pvp.SetFlag(int(VpFlagMenu))

	pvp.Geom.Pos = image.Point{x, y}
	// note: not setting VpFlagPopopDestroyAll -- we keep the menu list intact
	frame := pvp.AddNewChild(KiT_Frame, "Frame").(*Frame)
	frame.Lay = LayoutVert
	frame.SetProps(MenuFrameProps, false)
	var focus ki.Ki
	for _, ac := range menu {
		acn, ac := KiToNode2D(ac)
		if acn != nil {
			frame.AddChild(acn)
			if ac.IsSelected() {
				focus = acn
			}
		}
	}
	frame.Init2DTree()
	frame.Style2DTree()                                // sufficient to get sizes
	frame.LayData.AllocSize = mainVp.LayData.AllocSize // give it the whole vp initially
	frame.Size2DTree(0)                                // collect sizes
	pvp.Win = nil
	scextra := frame.Sty.Layout.ScrollBarWidth.Dots
	frame.LayData.Size.Pref.X += scextra // make room for scrollbar..
	vpsz := frame.LayData.Size.Pref.Min(mainVp.LayData.AllocSize.MulVal(.9)).ToPoint()
	maxht := int(32 * frame.Sty.Font.Height)
	vpsz.Y = ints.MinInt(maxht, vpsz.Y)
	x = ints.MinInt(x, mainVp.Geom.Size.X-vpsz.X) // fit
	y = ints.MinInt(y, mainVp.Geom.Size.Y-vpsz.Y) // fit
	pvp.Resize(vpsz)
	pvp.Geom.Pos = image.Point{x, y}
	pvp.UpdateEndNoSig(updt)
	win.SetNextPopup(pvp.This(), focus)
	return &pvp
}

// StringsChooserPopup creates a menu of the strings in the given string
// slice, and calls the given function on receiver when the user selects --
// this is the ActionSig signal, coming from the Action for the given menu
// item -- the name of the Action is the string value, and the data will be
// the index in the slice.  A string equal to curSel will be marked as
// selected.  Location is from the ContextMenuPos of recv node.
func StringsChooserPopup(strs []string, curSel string, recv Node2D, fun ki.RecvFunc) *Viewport2D {
	var menu Menu
	for i, it := range strs {
		ac := menu.AddAction(ActOpts{Label: it, Data: i}, recv, fun)
		ac.SetSelectedState(it == curSel)
	}
	nb := recv.AsNode2D()
	pos := recv.ContextMenuPos()
	vp := nb.Viewport
	if vp == nil {
		vp = recv.AsViewport2D()
	}
	return PopupMenu(menu, pos.X, pos.Y, vp, recv.Name())
}

// StringsInsertFirst inserts the given string at start of a string slice,
// while keeping overall length to given max value
// useful for a "recents" kind of string list
func StringsInsertFirst(strs *[]string, str string, max int) {
	if strs == nil {
		*strs = make([]string, 0, max)
	}
	sz := len(*strs)
	if sz > max {
		*strs = (*strs)[:max]
	}
	if sz >= max {
		copy((*strs)[1:max], (*strs)[0:max-1])
		(*strs)[0] = str
	} else {
		*strs = append(*strs, "")
		if sz > 0 {
			copy((*strs)[1:], (*strs)[0:sz])
		}
		(*strs)[0] = str
	}
}

// StringsInsertFirstUnique inserts the given string at start of a string slice,
// while keeping overall length to given max value.
// if item is already on the list, then it is moved to the top and not re-added (unique items only)
// useful for a "recents" kind of string list
func StringsInsertFirstUnique(strs *[]string, str string, max int) {
	if strs == nil {
		*strs = make([]string, 0, max)
	}
	sz := len(*strs)
	if sz > max {
		*strs = (*strs)[:max]
	}
	for i, s := range *strs {
		if s == str {
			if i == 0 {
				return
			}
			copy((*strs)[1:i+1], (*strs)[0:i])
			(*strs)[0] = str
			return
		}
	}
	if sz >= max {
		copy((*strs)[1:max], (*strs)[0:max-1])
		(*strs)[0] = str
	} else {
		*strs = append(*strs, "")
		if sz > 0 {
			copy((*strs)[1:], (*strs)[0:sz])
		}
		(*strs)[0] = str
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// MenuButton pops up a menu, has an indicator by default

type MenuButton struct {
	ButtonBase
}

var KiT_MenuButton = kit.Types.AddType(&MenuButton{}, MenuButtonProps)

var MenuButtonProps = ki.Props{
	"border-width":     units.NewValue(1, units.Px),
	"border-radius":    units.NewValue(4, units.Px),
	"border-color":     &Prefs.Colors.Border,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(4, units.Px),
	"margin":           units.NewValue(4, units.Px),
	"box-shadow.color": &Prefs.Colors.Shadow,
	"text-align":       AlignCenter,
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &Prefs.Colors.Icon,
		"stroke":  &Prefs.Colors.Font,
	},
	"#label": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	"#indicator": ki.Props{
		"width":          units.NewValue(1.5, units.Ex),
		"height":         units.NewValue(1.5, units.Ex),
		"margin":         units.NewValue(0, units.Px),
		"padding":        units.NewValue(0, units.Px),
		"vertical-align": AlignBottom,
		"fill":           &Prefs.Colors.Icon,
		"stroke":         &Prefs.Colors.Font,
	},
	"#ind-stretch": ki.Props{
		"width": units.NewValue(1, units.Em),
	},
	ButtonSelectors[ButtonActive]: ki.Props{
		"background-color": "linear-gradient(lighter-0, highlight-10)",
	},
	ButtonSelectors[ButtonInactive]: ki.Props{
		"border-color": "highlight-50",
		"color":        "highlight-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "linear-gradient(highlight-10, highlight-10)",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "linear-gradient(samelight-50, highlight-10)",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "highlight-90",
		"background-color": "linear-gradient(highlight-30, highlight-10)",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"background-color": "linear-gradient(pref(Select), highlight-10)",
	},
}

func (mb *MenuButton) ButtonAsBase() *ButtonBase {
	return &(mb.ButtonBase)
}

func (mb *MenuButton) ConfigParts() {
	config := kit.TypeAndNameList{}
	icIdx, lbIdx := mb.ConfigPartsIconLabel(&config, string(mb.Icon), mb.Text)
	indIdx := mb.ConfigPartsAddIndicator(&config, true)  // default on
	mods, updt := mb.Parts.ConfigChildren(config, false) // not unique names
	mb.ConfigPartsSetIconLabel(string(mb.Icon), mb.Text, icIdx, lbIdx)
	mb.ConfigPartsIndicator(indIdx)
	if mods {
		mb.UpdateEnd(updt)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Separator

// Separator draws a vertical or horizontal line
type Separator struct {
	WidgetBase
	Horiz bool `xml:"horiz" desc:"is this a horizontal separator -- otherwise vertical"`
}

var KiT_Separator = kit.Types.AddType(&Separator{}, SeparatorProps)

var SeparatorProps = ki.Props{
	"padding":          units.NewValue(0, units.Px),
	"margin":           units.NewValue(0, units.Px),
	"vertical-align":   AlignCenter,
	"horizontal-align": AlignCenter,
	"border-color":     &Prefs.Colors.Border,
	"border-width":     units.NewValue(2, units.Px),
	"background-color": &Prefs.Colors.Control,
	// todo: dotted
}

func (sp *Separator) Style2D() {
	if sp.Horiz {
		sp.SetProp("max-width", -1)
		sp.SetProp("min-height", units.NewValue(0.5, units.Ex))
	} else {
		sp.SetProp("max-height", -1)
		sp.SetProp("min-width", units.NewValue(0.5, units.Ch))
	}
	sp.WidgetBase.Style2D()
}

func (sp *Separator) Render2D() {
	if sp.PushBounds() {
		rs := &sp.Viewport.Render
		rs.Lock()
		pc := &rs.Paint
		st := &sp.Sty

		pos := sp.LayData.AllocPos.AddVal(st.Layout.Margin.Dots)
		sz := sp.LayData.AllocSize.AddVal(-2.0 * st.Layout.Margin.Dots)

		if !st.Font.BgColor.IsNil() {
			pc.FillBox(rs, pos, sz, &st.Font.BgColor)
		}

		pc.StrokeStyle.Width = st.Border.Width
		pc.StrokeStyle.SetColor(&st.Border.Color)
		if sp.Horiz {
			pc.DrawLine(rs, pos.X, pos.Y+0.5*sz.Y, pos.X+sz.X, pos.Y+0.5*sz.Y)
		} else {
			pc.DrawLine(rs, pos.X+0.5*sz.X, pos.Y, pos.X+0.5*sz.X, pos.Y+sz.Y)
		}
		pc.FillStrokeClear(rs)
		rs.Unlock()
		sp.Render2DChildren()
		sp.PopBounds()
	}
}
