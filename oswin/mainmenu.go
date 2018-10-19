// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oswin

// Menu is a pointer to an OS-specific menu structure.
type Menu uintptr

// MenuItem is a pointer to an OS-specific menu item structure.
type MenuItem uintptr

// MainMenu supports the OS-specific main menu associated with a window.
type MainMenu interface {
	// Window returns the window that this menu is attached to.
	Window() Window

	// SetWindow sets the window associated with this main menu.
	SetWindow(win Window)

	// SetFunc sets the callback function that is called whenever a menu item is selected.
	SetFunc(fun func(win Window, title string, tag int))

	// Triggered is called when a menu item is triggered via OS -- calls
	// callback function set in SetFunc.
	Triggered(win Window, title string, tag int)

	// Menu returns the menu pointer for the main menu associated with window.
	// only for READ ONLY purposes -- use StartUpdate when starting to update it.
	Menu() Menu

	// SetMenu sets the menu as the main menu for the window -- generally call this
	// in response to a window.FocusEvent, after building the menu
	SetMenu()

	// StartUpdate locks the menu pointer for the main menu associated with window.
	// only for READ ONLY purposes -- use StartUpdate when starting to update it.
	StartUpdate() Menu

	// EndUpdate unlocks the current update -- must be matched with StartUpdate
	EndUpdate(men Menu)

	// Reset resets all items on given menu -- do this before updating.
	Reset(men Menu)

	// AddSubMenu adds a sub-menu of given title / label to given menu.
	AddSubMenu(men Menu, title string) Menu

	// AddItem adds a menu item of given title / label, shortcut, and tag to
	// given menu.  Callback function set by SetFunc will be called when menu
	// item is selected.
	AddItem(men Menu, title string, shortcut string, tag int, active bool) MenuItem

	// AddSeparator adds a separator menu item.
	AddSeparator(men Menu)

	// ItemByTitle finds menu item by title on given menu -- does not iterate
	// over sub-menus, so that needs to be done manually.
	ItemByTitle(men Menu, title string) MenuItem

	// ItemByTag finds menu item by tag on given menu -- does not iterate
	// over sub-menus, so that needs to be done manually.
	ItemByTag(men Menu, tag int) MenuItem

	// SetItemActive sets the active status of given item.
	SetItemActive(mitem MenuItem, active bool)
}
