// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

/*

// Menu is a slice list of Buttons (or other Widgets)
// that are used for generating a Menu.
type Menu ki.Slice

/*
func (m Menu) MarshalJSON() ([]byte, error) {
	ks := (ki.Slice)(m)
	_ = ks
	// return ks.MarshalJSON()
	return nil, nil
}

func (m *Menu) UnmarshalJSON(b []byte) error {
	ks := (*ki.Slice)(m)
	_ = ks
	// return ks.UnmarshalJSON(b)
	return nil
}

func (m *Menu) CopyFrom(men *Menu) {
	ks := (*ki.Slice)(m)
	ks.CopyFrom((ki.Slice)(*men))
}

// MakeMenuFunc is a callback for making a menu on demand, receives the object
// calling this function (typically a Button) and the menu
type MakeMenuFunc func(obj Widget, m *Menu)

// SetButton sets properties of given button
func (m *Menu) SetButton(bt *Button, opts ActOpts, fun func(bt *Button)) {
	nm := opts.Name
	if nm == "" {
		nm = opts.Label
	}
	if nm == "" {
		nm = string(opts.Icon)
	}
	bt.InitName(bt, nm)
	bt.Type = ButtonMenu
	bt.Text = opts.Label
	bt.Tooltip = opts.Tooltip
	bt.Icon = icons.Icon(opts.Icon)
	bt.Shortcut = key.Chord(opts.Shortcut).OSShortcut()
	if opts.ShortcutKey != keyfun.Nil {
		bt.Shortcut = ShortcutForFun(opts.ShortcutKey)
		// todo: need a flag for menu-based?
	}
	bt.Data = opts.Data
	bt.UpdateFunc = opts.UpdateFunc
	if fun != nil {
		bt.OnClick(func(e events.Event) {
			fun(bt)
		})
	}
}

// AddButton adds an button to the menu using given options, and connects the
// button signal to given receiver object and function, along with given data
// which is stored on the button and then passed in the button signal.
// Optional updateFunc is a function called prior to showing the menu to
// update the buttons (enabled or not typically).
func (m *Menu) AddButton(opts ActOpts, fun func(bt *Button)) *Button {
	if m == nil {
		*m = make(Menu, 0, 10)
	}
	bt := &Button{}
	m.SetButton(bt, opts, fun)
	*m = append(*m, bt.This().(Widget))
	return bt
}

// InsertButtonBefore adds an button to the menu before existing item of given
// name, using given options, and connects the button signal to given receiver
// object and function, along with given data which is stored on the button
// and then passed in the button signal.  Optional updateFunc is a function
// called prior to showing the menu to update the buttons (enabled or not
// typically).  If name not found, adds to end of list..
func (m *Menu) InsertButtonBefore(before string, opts ActOpts, fun func(bt *Button)) *Button {
	sl := (*[]ki.Ki)(m)
	if idx, got := ki.SliceIndexByName(sl, before, 0); got {
		bt := &Button{}
		m.SetButton(bt, opts, fun)
		ki.SliceInsert(sl, bt.This(), idx)
		return bt
	} else {
		return m.AddButton(opts, fun)
	}
}

// InsertButtonAfter adds an button to the menu after existing item of given
// name, using given options, and connects the button signal to given receiver
// object and function, along with given data which is stored on the button
// and then passed in the button signal.  Optional updateFunc is a function
// called prior to showing the menu to update the buttons (enabled or not
// typically).  If name not found, adds to end of list..
func (m *Menu) InsertButtonAfter(after string, opts ActOpts, fun func(bt *Button)) *Button {
	sl := (*[]ki.Ki)(m)
	if idx, got := ki.SliceIndexByName(sl, after, 0); got {
		bt := &Button{}
		m.SetButton(bt, opts, fun)
		ki.SliceInsert(sl, bt.This(), idx+1)
		return bt
	} else {
		return m.AddButton(opts, fun)
	}
}

// AddSeparator adds a separator at the next point in the menu (name is just
// internal label of element, defaults to "separator" if unspecified)
func (m *Menu) AddSeparator(name ...string) *Separator {
	if m == nil {
		*m = make(Menu, 0, 10)
	}
	sp := &Separator{}
	sp.InitName(sp, name...)
	sp.Horiz = true
	*m = append(*m, sp.This().(Widget))
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
	*m = append(*m, lb.This().(Widget))
	return lb
}

// SetShortcuts sets the shortcuts to given window -- call when the menu has
// been attached to a window
func (m *Menu) SetShortcuts(em *EventMgr) {
	if em == nil {
		return
	}
	for _, mi := range *m {
		if bt := AsButton(mi); bt != nil {
			em.AddShortcut(bt.Shortcut, bt)
		}
	}
}

// DeleteShortcuts deletes the shortcuts in given window
func (m *Menu) DeleteShortcuts(em *EventMgr) {
	if em == nil {
		return
	}
	for _, mi := range *m {
		if bt := AsButton(mi); bt != nil {
			em.DeleteShortcut(bt.Shortcut, bt)
		}
	}
}

// UpdateButtons calls update function on all the buttons in the menu, and any
// of their sub-buttons
func (m *Menu) UpdateButtons() {
	for _, mi := range *m {
		if bt := AsButton(mi); bt != nil {
			bt.UpdateButtons()
		}
	}
}

// FindButtonByName finds an button on the menu, or any sub-menu, with given
// name (exact match) -- this is not the Text label but the Name of the
// element (for AddButton items, this is the same as Label or Icon (if Label
// is empty)) -- returns false if not found
func (m *Menu) FindButtonByName(name string) (*Button, bool) {
	for _, mi := range *m {
		if bt := AsButton(mi); bt != nil {
			if bt.Name() == name {
				return bt, true
			}
			if bt.Menu != nil {
				if sbt, ok := bt.Menu.FindButtonByName(name); ok {
					return sbt, ok
				}
			}
		}
	}
	return nil, false
}

///////////////////////////////////////////////////////////////
// Standard menu elements

/*

// AddCopyCutPaste adds a Copy, Cut, Paste buttons that just emit the
// corresponding keyboard shortcut.  Paste is automatically enabled by
// clipboard having something in it.
func (m *Menu) AddCopyCutPaste(win *RenderWin) {
	m.AddButton(ActOpts{Label: "Copy", ShortcutKey: keyfun.Copy},
		nil, func(recv, send ki.Ki, sig int64, data any) {
			win.EventMgr.Sendkeyfun.Event(keyfun.Copy, false) // false = ignore popups -- don't send to menu
		})
	m.AddButton(ActOpts{Label: "Cut", ShortcutKey: keyfun.Cut},
		nil, func(recv, send ki.Ki, sig int64, data any) {
			win.EventMgr.Sendkeyfun.Event(keyfun.Cut, false) // false = ignore popups -- don't send to menu
		})
	m.AddButton(ActOpts{Label: "Paste", ShortcutKey: keyfun.Paste,
		UpdateFunc: func(bt *Button) {
			bt.SetEnabledState(!goosi.TheApp.ClipBoard(win.RenderWin).IsEmpty())
		}}, nil, func(recv, send ki.Ki, sig int64, data any) {
		win.EventMgr.Sendkeyfun.Event(keyfun.Paste, false) // false = ignore popups -- don't send to menu
	})
}

// AddCopyCutPasteDupe adds a Copy, Cut, Paste, and Duplicate buttons that
// just emit the corresponding keyboard shortcut.  Paste is automatically
// enabled by clipboard having something in it.
func (m *Menu) AddCopyCutPasteDupe(win *RenderWin) {
	m.AddCopyCutPaste(win)
	dpsc := ActiveKeyMap.ChordForFun(keyfun.Duplicate)
	m.AddButton(ActOpts{Label: "Duplicate", Shortcut: dpsc},
		nil, func(recv, send ki.Ki, sig int64, data any) {
			win.EventMgr.Sendkeyfun.Event(keyfun.Duplicate, false) // false = ignore popups -- don't send to menu
		})
}


// CustomAppMenuFunc is a function called by AddAppMenu after the
// AddStdAppMenu is called -- apps can set this function to add / modify / etc
// the menu
var CustomAppMenuFunc = (func(m *Menu, win *RenderWin))(nil)

// AddAppMenu adds an "app" menu to the menu -- calls AddStdAppMenu and then
// CustomAppMenuFunc if non-nil
func (m *Menu) AddAppMenu(win *RenderWin) {
	m.AddStdAppMenu(win)
	if CustomAppMenuFunc != nil {
		CustomAppMenuFunc(m, win)
	}
}

/*
// AddStdAppMenu adds a standard set of menu items for application-level control.
func (m *Menu) AddStdAppMenu(win *RenderWin) {
	aboutitle := "About " + goosi.TheApp.Name()
	m.AddButton(ActOpts{Label: aboutitle},
		nil, func(recv, send ki.Ki, sig int64, data any) {
			PromptDialog(win.Scene, DlgOpts{Title: aboutitle, Prompt: goosi.TheApp.About()}, AddOk, NoCancel, nil, nil)
		})
	m.AddButton(ActOpts{Label: "GoGi Preferences...", Shortcut: "Command+P"},
		nil, func(recv, send ki.Ki, sig int64, data any) {
			TheViewIFace.PrefsView(&Prefs)
		})
	m.AddSeparator("sepq")
	m.AddButton(ActOpts{Label: "Quit", Shortcut: "Command+Q"},
		nil, func(recv, send ki.Ki, sig int64, data any) {
			goosi.TheApp.QuitReq()
		})
}

// AddRenderWinsMenu adds menu items for current main and dialog windows.
// must be called under RenderWinGlobalMu mutex lock!
func (m *Menu) AddRenderWinsMenu(win *RenderWin) {
	/*  todo
	m.AddButton(ActOpts{Label: "Minimize"},
		nil, func(recv, send ki.Ki, sig int64, data any) {
			win.GoosiWin.Minimize()
		})
	m.AddButton(ActOpts{Label: "Focus Next", ShortcutKey: keyfun.WinFocusNext},
		nil, func(recv, send ki.Ki, sig int64, data any) {
			AllRenderWins.FocusNext()
		})
	m.AddSeparator("sepa")
	for _, w := range MainRenderWins {
		if w != nil {
			m.AddButton(ActOpts{Label: w.Title},
				nil, func(recv, send ki.Ki, sig int64, data any) {
					w.GoosiWin.Raise()
				})
		}
	}
	if len(DialogRenderWins) > 0 {
		m.AddSeparator("sepw")
		for _, w := range DialogRenderWins {
			if w != nil {
				m.AddButton(ActOpts{Label: w.Title},
					nil, func(recv, send ki.Ki, sig int64, data any) {
						w.GoosiWin.Raise()
					})
			}
		}
	}
}

///////////////////////////////////////////////////////////////////
// PopupMenu function

///////////////////////////////////////////////////////////////
// 	Context Menu

// CtxtMenuFunc is a function for creating a context menu for given node
type CtxtMenuFunc func(g Widget, m *Menu)

///////////////////////////////////////////////////////////////
// 	Choosers

// StringsChooserPopup creates a menu of the strings in the given string
// slice, and calls the given function on receiver when the user selects.
// This is an event coming from the Button for the given menu
// item; the name of the Button is the string value, and the data will be
// the index in the slice.  A string equal to curSel will be marked as
// selected. ctx Widget provides position etc for the menu.
func StringsChooserPopup(strs []string, curSel string, ctx Widget, fun func(bt *Button)) {
	var menu Menu
	for i, it := range strs {
		bt := menu.AddButton(ActOpts{Label: it, Data: i}, fun)
		bt.SetSelected(it == curSel)
	}
	NewMenu(menu, ctx, ctx.ContextMenuPos(nil)).Run()
}

// SubStringsChooserPopup creates a menu of the sub-strings in the given
// slice of string slices, and calls the given function on receiver when
// the user selects.  This is an event coming from the Button for the given
// menu item. The sub-menu name is the first element of each sub-slice.
// The name of the Button is the string value, and the data is an
// []int{s,i} slice of submenu and item indexes.
// A string of subMenu: item equal to curSel will be marked as selected.
// Location is from the ContextMenuPos of recv node.
func SubStringsChooserPopup(strs [][]string, curSel string, ctx Widget, fun func(bt *Button)) *Scene {
	var menu Menu
	for si, ss := range strs {
		sz := len(ss)
		if sz < 2 {
			continue
		}
		s1 := ss[0]
		sm := menu.AddButton(ActOpts{Label: s1}, nil)
		sm.Type = ButtonMenu
		for i := 1; i < sz; i++ {
			it := ss[i]
			cnm := s1 + ": " + it
			bt := sm.Menu.AddButton(ActOpts{Label: it, Data: []int{si, i}}, fun)
			bt.SetSelected(cnm == curSel)
		}
	}
	// wb := ctx.AsWidget()
	// pos := recv.ContextMenuPos()
	// sc := wb.Sc
	// return PopupMenu(menu, pos.X, pos.Y, sc, recv.Name())
	return nil
}
*/
