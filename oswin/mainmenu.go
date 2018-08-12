// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oswin

// Menu is a pointer to an OS-specific menu structure.
type Menu uintptr

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

	// MainMenu returns the menu pointer for the main menu associated with window.
	Menu() Menu

	// Reset resets all items on given menu -- do this before updating.
	Reset(men Menu)

	// AddSubMenu adds a sub-menu of given title / label to given menu.
	AddSubMenu(men Menu, title string) Menu

	// AddItem adds a menu item of given title / label, shortcut, and tag to
	// given menu.  Callback function set by SetFunc will be called when menu
	// item is selected.
	AddItem(men Menu, title string, shortcut string, tag int)

	// AddSeparator adds a separator menu item.
	AddSeparator(men Menu)
}
