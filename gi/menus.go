// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"

	"github.com/goki/gi/gist"
	"github.com/goki/gi/icons"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
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

func (m *Menu) CopyFrom(men *Menu) {
	ks := (*ki.Slice)(m)
	ks.CopyFrom((ki.Slice)(*men))
}

// MakeMenuFunc is a callback for making a menu on demand, receives the object
// calling this function (typically an Action or Button) and the menu
type MakeMenuFunc func(obj ki.Ki, m *Menu)

// ActOpts provides named and partial parameters for AddAction method
type ActOpts struct {
	Name        string
	Label       string
	Icon        icons.Icon
	Tooltip     string
	Shortcut    key.Chord
	ShortcutKey KeyFuns
	Data        any
	UpdateFunc  func(act *Action)
}

// SetAction sets properties of given action
func (m *Menu) SetAction(ac *Action, opts ActOpts, sigTo ki.Ki, fun ki.RecvFunc) {
	nm := opts.Name
	if nm == "" {
		nm = opts.Label
	}
	if nm == "" {
		nm = string(opts.Icon)
	}
	ac.InitName(ac, nm)
	ac.Text = opts.Label
	ac.Tooltip = opts.Tooltip
	ac.Icon = icons.Icon(opts.Icon)
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
	sp := &Separator{}
	if name == "" {
		name = "sep"
	}
	sp.InitName(sp, name)
	sp.Horiz = true
	*m = append(*m, sp.This().(Node2D))
	return sp
}

// AddLabel adds a label to the menu
func (m *Menu) AddLabel(lbl string) *Label {
	if m == nil {
		*m = make(Menu, 0, 10)
	}
	lb := &Label{}
	lb.InitName(lb, lbl)
	lb.SetText(lbl)
	lb.SetProp("background-color", &Prefs.Colors.Control)
	*m = append(*m, lb.This().(Node2D))
	return lb
}

// SetShortcuts sets the shortcuts to given window -- call when the menu has
// been attached to a window
func (m *Menu) SetShortcuts(win *Window) {
	if win == nil {
		return
	}
	for _, mi := range *m {
		if ki.TypeEmbeds(mi, TypeAction) {
			ac := mi.Embed(TypeAction).(*Action)
			win.AddShortcut(ac.Shortcut, ac)
		}
	}
}

// DeleteShortcuts deletes the shortcuts in given window
func (m *Menu) DeleteShortcuts(win *Window) {
	if win == nil {
		return
	}
	for _, mi := range *m {
		if ki.TypeEmbeds(mi, TypeAction) {
			ac := mi.Embed(TypeAction).(*Action)
			win.DeleteShortcut(ac.Shortcut, ac)
		}
	}
}

// UpdateActions calls update function on all the actions in the menu, and any
// of their sub-actions
func (m *Menu) UpdateActions() {
	for _, mi := range *m {
		if ki.TypeEmbeds(mi, TypeAction) {
			ac := mi.Embed(TypeAction).(*Action)
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
		if ki.TypeEmbeds(mi, TypeAction) {
			ac := mi.Embed(TypeAction).(*Action)
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
	m.AddAction(ActOpts{Label: "Copy", ShortcutKey: KeyFunCopy},
		win, func(recv, send ki.Ki, sig int64, data any) {
			ww := recv.Embed(TypeWindow).(*Window)
			ww.EventMgr.SendKeyFunEvent(KeyFunCopy, false) // false = ignore popups -- don't send to menu
		})
	m.AddAction(ActOpts{Label: "Cut", ShortcutKey: KeyFunCut},
		win, func(recv, send ki.Ki, sig int64, data any) {
			ww := recv.Embed(TypeWindow).(*Window)
			ww.EventMgr.SendKeyFunEvent(KeyFunCut, false) // false = ignore popups -- don't send to menu
		})
	m.AddAction(ActOpts{Label: "Paste", ShortcutKey: KeyFunPaste,
		UpdateFunc: func(ac *Action) {
			ac.SetInactiveState(oswin.TheApp.ClipBoard(win.OSWin).IsEmpty())
		}}, win, func(recv, send ki.Ki, sig int64, data any) {
		ww := recv.Embed(TypeWindow).(*Window)
		ww.EventMgr.SendKeyFunEvent(KeyFunPaste, false) // false = ignore popups -- don't send to menu
	})
}

// AddCopyCutPasteDupe adds a Copy, Cut, Paste, and Duplicate actions that
// just emit the corresponding keyboard shortcut.  Paste is automatically
// enabled by clipboard having something in it.
func (m *Menu) AddCopyCutPasteDupe(win *Window) {
	m.AddCopyCutPaste(win)
	dpsc := ActiveKeyMap.ChordForFun(KeyFunDuplicate)
	m.AddAction(ActOpts{Label: "Duplicate", Shortcut: dpsc},
		win, func(recv, send ki.Ki, sig int64, data any) {
			ww := recv.Embed(TypeWindow).(*Window)
			ww.EventMgr.SendKeyFunEvent(KeyFunDuplicate, false) // false = ignore popups -- don't send to menu
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
		win, func(recv, send ki.Ki, sig int64, data any) {
			ww := recv.Embed(TypeWindow).(*Window)
			PromptDialog(ww.Viewport, DlgOpts{Title: aboutitle, Prompt: oswin.TheApp.About()}, AddOk, NoCancel, nil, nil)
		})
	m.AddAction(ActOpts{Label: "GoGi Preferences...", Shortcut: "Command+P"},
		win, func(recv, send ki.Ki, sig int64, data any) {
			TheViewIFace.PrefsView(&Prefs)
		})
	m.AddSeparator("sepq")
	m.AddAction(ActOpts{Label: "Quit", Shortcut: "Command+Q"},
		win, func(recv, send ki.Ki, sig int64, data any) {
			oswin.TheApp.QuitReq()
		})
}

// AddWindowsMenu adds menu items for current main and dialog windows.
// must be called under WindowGlobalMu mutex lock!
func (m *Menu) AddWindowsMenu(win *Window) {
	m.AddAction(ActOpts{Label: "Minimize"},
		win, func(recv, send ki.Ki, sig int64, data any) {
			ww := recv.Embed(TypeWindow).(*Window)
			ww.OSWin.Minimize()
		})
	m.AddAction(ActOpts{Label: "Focus Next", ShortcutKey: KeyFunWinFocusNext},
		win, func(recv, send ki.Ki, sig int64, data any) {
			AllWindows.FocusNext()
		})
	m.AddSeparator("sepa")
	for _, w := range MainWindows {
		if w != nil {
			m.AddAction(ActOpts{Label: w.Title},
				w, func(recv, send ki.Ki, sig int64, data any) {
					ww := recv.Embed(TypeWindow).(*Window)
					ww.OSWin.Raise()
				})
		}
	}
	if len(DialogWindows) > 0 {
		m.AddSeparator("sepw")
		for _, w := range DialogWindows {
			if w != nil {
				m.AddAction(ActOpts{Label: w.Title},
					w, func(recv, send ki.Ki, sig int64, data any) {
						ww := recv.Embed(TypeWindow).(*Window)
						ww.OSWin.Raise()
					})
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// PopupMenu function

//	var MenuFrameProps = ki.Props{
//		"border-width":        units.Px(0),
//		"border-color":        "none",
//		"margin":              units.Px(4),
//		"padding":             units.Px(2),
//		"box-shadow.h-offset": units.Px(2),
//		"box-shadow.v-offset": units.Px(2),
//		"box-shadow.blur":     units.Px(2),
//		"box-shadow.color":    &Prefs.Colors.Shadow,
//	}
//
// MenuFrameConfigStyles configures the default styles
// for the given pop-up menu frame with the given parent.
// It should be called on menu frames when they are created.
func MenuFrameConfigStyles(par *WidgetBase, frame *Frame) {
	frame.AddStyleFunc(StyleFuncParts(par), func() {
		frame.Style.Border.Style.Set(gist.BorderNone)
		frame.Style.Padding.Set()
		frame.Style.Margin.Set()
		// doesn't seem to work; TODO: fix box shadow here
		// frame.Style.BoxShadow.HOffset.SetPx(2)
		// frame.Style.BoxShadow.VOffset.SetPx(2)
		// frame.Style.BoxShadow.Blur.SetPx(2)
		// frame.Style.BoxShadow.Color = Colors.Background.Highlight(30)
	})
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

	pvp := &Viewport2D{}
	pvp.InitName(pvp, name+"Menu")
	pvp.Win = win
	updt := pvp.UpdateStart()
	pvp.SetProp("color", &Prefs.Colors.Font)
	pvp.Fill = true
	pvp.SetFlag(int(VpFlagPopup))
	pvp.SetFlag(int(VpFlagMenu))

	pvp.Geom.Pos = image.Point{x, y}
	// note: not setting VpFlagPopupDestroyAll -- we keep the menu list intact
	frame := AddNewFrame(pvp, "Frame", LayoutVert)
	MenuFrameConfigStyles(&parVp.WidgetBase, frame)
	// frame.Properties().CopyFrom(MenuFrameProps, ki.DeepCopy)
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
	frame.Style2DTree()                                    // sufficient to get sizes
	frame.LayState.Alloc.Size = mainVp.LayState.Alloc.Size // give it the whole vp initially
	frame.Size2DTree(0)                                    // collect sizes
	pvp.Win = nil
	scextra := frame.Style.ScrollBarWidth.Dots
	frame.LayState.Size.Pref.X += scextra // make room for scrollbar..
	vpsz := frame.LayState.Size.Pref.Min(mainVp.LayState.Alloc.Size.MulScalar(.9)).ToPoint()
	maxht := int(32 * frame.Style.Font.Face.Metrics.Height)
	vpsz.Y = ints.MinInt(maxht, vpsz.Y)
	x = ints.MaxInt(0, x)
	y = ints.MaxInt(0, y)
	x = ints.MinInt(x, mainVp.Geom.Size.X-vpsz.X) // fit
	y = ints.MinInt(y, mainVp.Geom.Size.Y-vpsz.Y) // fit
	pvp.Resize(vpsz)
	pvp.Geom.Pos = image.Point{x, y}
	pvp.UpdateEndNoSig(updt)
	win.SetNextPopup(pvp.This(), focus)
	return pvp
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
	vp := nb.ViewportSafe()
	if vp == nil {
		vp = recv.AsViewport2D()
	}
	return PopupMenu(menu, pos.X, pos.Y, vp, recv.Name())
}

// SubStringsChooserPopup creates a menu of the sub-strings in the given
// slice of string slices, and calls the given function on receiver when
// the user selects.  This is the ActionSig signal, coming from the Action
// for the given menu item.
// The sub-menu name is the first element of each sub-slice.
// The name of the Action is the string value, and the data is an
// []int{s,i} slice of submenu and item indexes.
// A string of subMenu: item equal to curSel will be marked as selected.
// Location is from the ContextMenuPos of recv node.
func SubStringsChooserPopup(strs [][]string, curSel string, recv Node2D, fun ki.RecvFunc) *Viewport2D {
	var menu Menu
	for si, ss := range strs {
		sz := len(ss)
		if sz < 2 {
			continue
		}
		s1 := ss[0]
		sm := menu.AddAction(ActOpts{Label: s1}, nil, nil)
		sm.SetAsMenu()
		for i := 1; i < sz; i++ {
			it := ss[i]
			cnm := s1 + ": " + it
			ac := sm.Menu.AddAction(ActOpts{Label: it, Data: []int{si, i}}, recv, fun)
			ac.SetSelectedState(cnm == curSel)
		}
	}
	nb := recv.AsNode2D()
	pos := recv.ContextMenuPos()
	vp := nb.ViewportSafe()
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

// StringsDelete deletes item from strings list
func StringsDelete(strs *[]string, str string) {
	for i, s := range *strs {
		if s == str {
			*strs = append((*strs)[:i], (*strs)[i+1:]...)
			return
		}
	}
}

// StringsAppendIfUnique append str to strs if not already in slice
func StringsAppendIfUnique(strs *[]string, str string, max int) {
	if strs == nil {
		*strs = make([]string, 0, max)
	}
	for _, s := range *strs {
		if s == str {
			return
		}
	}
	*strs = append(*strs, str)
}

// StringsAddExtras is a generic function for appending a slice to a slice used to add items to menus
func StringsAddExtras(items *[]string, extras []string) {
	*items = append(*items, extras...)
}

// StringsRemoveExtras is a generic function for removing items of a slice from another slice
func StringsRemoveExtras(items *[]string, extras []string) {
	for _, extra := range extras {
		i := 0
		for _, item := range *items {
			if item != extra {
				(*items)[i] = item
				i++
			}
		}
		*items = (*items)[:i]
	}
}

// //////////////////////////////////////////////////////////////////////////////////////

// MenuButton is a button that pops up a menu.
// It has an indicator by default.
type MenuButton struct {
	ButtonBase
	Type MenuButtonTypes `desc:"type is the type of the menu button"`
}

var TypeMenuButton = kit.Types.AddType(&MenuButton{}, MenuButtonProps)

// MenuButtonTypes is an enum containing the
// different possible types of menu buttons
type MenuButtonTypes int

const (
	// MenuButtonFilled represents a filled
	// MenuButton with a background color
	// and no border
	MenuButtonFilled MenuButtonTypes = iota
	// MenuButtonOutlined represents an outlined
	// MenuButton with a border on all sides
	// and no background color
	MenuButtonOutlined
	// MenuButtonText represents a MenuButton
	// with no border or background color.
	MenuButtonText

	MenuButtonTypesN
)

var TypeMenuButtonTypes = kit.Enums.AddEnumAltLower(MenuButtonTypesN, kit.NotBitFlag, gist.StylePropProps, "MenuButton")

//go:generate stringer -type=MenuButtonTypes

// AddNewMenuButton adds a new button to given parent node, with given name.
func AddNewMenuButton(parent ki.Ki, name string) *MenuButton {
	return parent.AddNewChild(TypeMenuButton, name).(*MenuButton)
}

func (mb *MenuButton) CopyFieldsFrom(frm any) {
	fr := frm.(*MenuButton)
	mb.ButtonBase.CopyFieldsFrom(&fr.ButtonBase)
}

// // DefaultStyle implements the [DefaultStyler] interface
// func (mb *MenuButton) DefaultStyle() {
// 	cs := CurrentColorScheme()
// 	s := &mb.Style

// 	s.Border.Style.Set(gist.BorderNone)
// 	s.Border.Width.Set()
// 	s.Margin.Set(units.Px(4))
// 	s.Padding.Set(units.Px(4))
// 	s.Text.Align = gist.AlignCenter
// 	s.BackgroundColor.SetColor(cs.Background.Highlight(10))
// 	s.Color.SetColor(cs.Font)
// }

var MenuButtonProps = ki.Props{
	"EnumType:Flag": TypeButtonFlags,
	// "border-width":     units.Px(1),
	// "border-radius":    units.Px(4),
	// "border-color":     &Prefs.Colors.Border,
	// "border-style":     gist.BorderSolid,
	// "padding":          units.Px(4),
	// "margin":           units.Px(4),
	// "box-shadow.color": &Prefs.Colors.Shadow,
	// "text-align":       gist.AlignCenter,
	// "background-color": &Prefs.Colors.Control,
	// "color":            &Prefs.Colors.Font,
	// "#icon": ki.Props{
	// 	"width":   units.Em(1),
	// 	"height":  units.Em(1),
	// 	"margin":  units.Px(0),
	// 	"padding": units.Px(0),
	// 	"fill":    &Prefs.Colors.Icon,
	// 	"stroke":  &Prefs.Colors.Font,
	// },
	// "#label": ki.Props{
	// 	"margin":  units.Px(0),
	// 	"padding": units.Px(0),
	// },
	// "#indicator": ki.Props{
	// 	"width":          units.Ex(1.5),
	// 	"height":         units.Ex(1.5),
	// 	"margin":         units.Px(0),
	// 	"padding":        units.Px(0),
	// 	"vertical-align": gist.AlignBottom,
	// 	"fill":           &Prefs.Colors.Icon,
	// 	"stroke":         &Prefs.Colors.Font,
	// },
	// "#ind-stretch": ki.Props{
	// 	"width": units.Em(1),
	// },
	// ButtonSelectors[ButtonActive]: ki.Props{
	// 	"background-color": "linear-gradient(lighter-0, highlight-10)",
	// },
	// ButtonSelectors[ButtonInactive]: ki.Props{
	// 	"border-color": "highlight-50",
	// 	"color":        "highlight-50",
	// },
	// ButtonSelectors[ButtonHover]: ki.Props{
	// 	"background-color": "linear-gradient(highlight-10, highlight-10)",
	// },
	// ButtonSelectors[ButtonFocus]: ki.Props{
	// 	"border-width":     units.Px(2),
	// 	"background-color": "linear-gradient(samelight-50, highlight-10)",
	// },
	// ButtonSelectors[ButtonDown]: ki.Props{
	// 	"color":            "highlight-90",
	// 	"background-color": "linear-gradient(highlight-30, highlight-10)",
	// },
	// ButtonSelectors[ButtonSelected]: ki.Props{
	// 	"background-color": "linear-gradient(pref(Select), highlight-10)",
	// },
}

func (mb *MenuButton) ConfigParts() {
	config := kit.TypeAndNameList{}
	if mb.Icon == "" {
		mb.Icon = icons.Menu
	}
	if mb.Indicator == "" {
		mb.Indicator = icons.None
	}
	icIdx, lbIdx := mb.ConfigPartsIconLabel(&config, mb.Icon, mb.Text)
	indIdx := mb.ConfigPartsAddIndicator(&config, false) // default on
	mods, updt := mb.Parts.ConfigChildren(config)
	mb.ConfigPartsSetIconLabel(mb.Icon, mb.Text, icIdx, lbIdx)
	mb.ConfigPartsIndicator(indIdx)
	if mods {
		mb.UpdateEnd(updt)
	}
}

func (mb *MenuButton) Init2D() {
	mb.Init2DWidget()
	mb.ConfigStyles()
}

func (mb *MenuButton) ConfigStyles() {
	mb.AddStyleFunc(StyleFuncDefault, func() {
		mb.Style.Margin.Set(units.Px(4 * Prefs.DensityMul()))
		mb.Style.Text.Align = gist.AlignCenter
		mb.Style.Color = Colors.Text
		mb.Style.Padding.Set(units.Px(4 * Prefs.DensityMul()))
		mb.Style.Border.Radius.Set(units.Px(10))
		switch mb.Type {
		case MenuButtonFilled:
			mb.Style.Border.Style.Set(gist.BorderNone)
			mb.Style.BackgroundColor.SetColor(Colors.Background.Highlight(10))
		case MenuButtonOutlined:
			mb.Style.Border.Style.Set(gist.BorderSolid)
			mb.Style.Border.Width.Set(units.Px(1))
			mb.Style.Border.Color.Set(Colors.Text)
			mb.Style.BackgroundColor.SetColor(Colors.Background)
		case MenuButtonText:
			mb.Style.Border.Style.Set(gist.BorderNone)
			mb.Style.BackgroundColor.SetColor(Colors.Background)
		}
		switch mb.State {
		case ButtonActive:
			// use background as already specified above
		case ButtonInactive:
			mb.Style.BackgroundColor.SetColor(mb.Style.BackgroundColor.Color.Highlight(20))
			mb.Style.Color = Colors.Text.Highlight(20)
		case ButtonFocus, ButtonSelected:
			mb.Style.BackgroundColor.SetColor(mb.Style.BackgroundColor.Color.Highlight(10))
		case ButtonHover:
			mb.Style.BackgroundColor.SetColor(mb.Style.BackgroundColor.Color.Highlight(15))
		case ButtonDown:
			mb.Style.BackgroundColor.SetColor(mb.Style.BackgroundColor.Color.Highlight(20))
		}
	})
	mb.Parts.AddChildStyleFunc("icon", 0, StyleFuncParts(mb), func(icon *WidgetBase) {
		icon.Style.Width.SetEm(1)
		icon.Style.Height.SetEm(1)
		icon.Style.Margin.Set()
		icon.Style.Padding.Set()
	})
	mb.Parts.AddChildStyleFunc("label", 1, StyleFuncParts(mb), func(label *WidgetBase) {
		label.Style.Margin.Set()
		label.Style.Padding.Set()
		label.Style.AlignV = gist.AlignMiddle
	})
	mb.Parts.AddChildStyleFunc("ind-stretch", 2, StyleFuncParts(mb), func(ins *WidgetBase) {
		ins.Style.Width.SetEm(1)
	})
	mb.Parts.AddChildStyleFunc("indicator", 3, StyleFuncParts(mb), func(ind *WidgetBase) {
		ind.Style.Width.SetEx(1.5)
		ind.Style.Height.SetEx(1.5)
		ind.Style.Margin.Set()
		ind.Style.Padding.Set()
		ind.Style.AlignV = gist.AlignMiddle
		ind.Style.AlignH = gist.AlignCenter
	})
}

////////////////////////////////////////////////////////////////////////////////////////
// Separator

// Separator defines a string to indicate a menu separator item
var MenuTextSeparator = "-------------"

// Separator draws a vertical or horizontal line
type Separator struct {
	WidgetBase
	Horiz bool `xml:"horiz" desc:"is this a horizontal separator -- otherwise vertical"`
}

var TypeSeparator = kit.Types.AddType(&Separator{}, SeparatorProps)

// AddNewSeparator adds a new separator to given parent node, with given name and Horiz (else Vert).
func AddNewSeparator(parent ki.Ki, name string, horiz bool) *Separator {
	sp := parent.AddNewChild(TypeSeparator, name).(*Separator)
	sp.Horiz = horiz
	return sp
}

func (sp *Separator) CopyFieldsFrom(frm any) {
	fr := frm.(*Separator)
	sp.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sp.Horiz = fr.Horiz
}

// // DefaultStyle implements the [DefaultStyler] interface
// func (sp *Separator) DefaultStyle() {
// 	cs := CurrentColorScheme()
// 	s := &sp.Style

// 	s.Padding.Set()
// 	s.Margin.Set()
// 	s.AlignV = gist.AlignCenter
// 	s.AlignH = gist.AlignCenter
// 	s.Border.Color.Set(cs.Background.Highlight(30))
// 	s.Border.Width.Set(units.Px(2))
// 	s.BackgroundColor.SetColor(cs.Background.Highlight(10))
// }

var SeparatorProps = ki.Props{
	"EnumType:Flag": TypeNodeFlags,
	// "padding":          units.Px(0),
	// "margin":           units.Px(0),
	// "vertical-align":   gist.AlignCenter,
	// "horizontal-align": gist.AlignCenter,
	// "border-color":     &Prefs.Colors.Border,
	// "border-width":     units.Px(2),
	// "background-color": &Prefs.Colors.Control,
	// todo: dotted
}

func (sp *Separator) Style2D() {
	// sp.StyMu.Lock()
	// if sp.Horiz {
	// 	sp.SetProp("max-width", -1)
	// 	sp.SetProp("min-height", units.Ex(0.5))
	// } else {
	// 	sp.SetProp("max-height", -1)
	// 	sp.SetProp("min-width", units.Ch(0.5))
	// }
	// sp.StyMu.Unlock()
	sp.WidgetBase.Style2D()
}

func (sp *Separator) RenderSeparator() {
	rs, pc, st := sp.RenderLock()
	defer sp.RenderUnlock(rs)

	pos := sp.LayState.Alloc.Pos.Add(st.Margin.Dots().Pos())
	sz := sp.LayState.Alloc.Size.Sub(st.Margin.Dots().Size())

	if !st.BackgroundColor.IsNil() {
		pc.FillBox(rs, pos, sz, &st.BackgroundColor)
	}
	// border-top is standard property for separators in CSS (see https://www.w3schools.com/howto/howto_css_dividers.asp)
	pc.StrokeStyle.Width = st.Border.Width.Top
	pc.StrokeStyle.SetColor(&st.Border.Color.Top)
	if sp.Horiz {
		pc.DrawLine(rs, pos.X, pos.Y+0.5*sz.Y, pos.X+sz.X, pos.Y+0.5*sz.Y)
	} else {
		pc.DrawLine(rs, pos.X+0.5*sz.X, pos.Y, pos.X+0.5*sz.X, pos.Y+sz.Y)
	}
	pc.FillStrokeClear(rs)
}

func (sp *Separator) Render2D() {
	if sp.PushBounds() {
		sp.RenderSeparator()
		sp.Render2DChildren()
		sp.PopBounds()
	}
}

func (sp *Separator) Init2D() {
	sp.Init2DWidget()
	sp.ConfigStyles()
}

func (sp *Separator) ConfigStyles() {
	// TODO: fix disappearing separator in menu
	sp.AddStyleFunc(StyleFuncDefault, func() {
		sp.Style.Margin.Set()
		sp.Style.Padding.Set()
		sp.Style.AlignV = gist.AlignCenter
		sp.Style.AlignH = gist.AlignCenter
		sp.Style.Border.Style.Set(gist.BorderSolid)
		sp.Style.Border.Width.Set(units.Px(0))
		sp.Style.Border.Color.Set(Colors.Text.Highlight(20))
		sp.Style.BackgroundColor.SetColor(Colors.Text.Highlight(20))
		if sp.Horiz {
			sp.Style.MaxWidth.SetPx(-1)
			sp.Style.MinHeight.SetPx(1)
		} else {
			sp.Style.MaxHeight.SetPx(-1)
			sp.Style.MinWidth.SetPx(1)
		}
	})
}
