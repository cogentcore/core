// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package system provides a Go operating system interface framework
// to support events, window management, and other OS-specific
// functionality needed for full GUI support.
package system

import "cogentcore.org/core/styles"

//go:generate core generate

var (
	// TheApp is the current [App]; only one is ever in effect.
	TheApp App

	// AppVersion is the version of the current app.
	// It is set by a linker flag in the core command line tool.
	AppVersion = "dev"

	// CoreVersion is the version of Cogent Core that the current app is using.
	// It is set by a linker flag in the core command line tool.
	CoreVersion = "dev"

	// ReservedWebShortcuts is a list of shortcuts that are reserved on the web
	// platform, meaning that they are handled by the browser instead of Cogent Core.
	// By default, this list contains important web shortcuts like Command+r and Command+(1-9).
	// All instances of "Command" are automatically converted to "Control" on non-macOS system
	// platforms, meaning that shortcuts should typically be expressed using "Command" for
	// cross-platform support. Modifiers should be specified in the order of [key.Modifiers]:
	// Shift, Control, Alt, Command. Shortcuts can be removed from this list by an app whose use of
	// them is more important than the default web action for that shortcut.
	ReservedWebShortcuts = []string{"Command+r", "Shift+Command+r", "Command+w", "Command+t", "Shift+Command+t", "Command+q", "Command+n", "Command+m", "Command+l", "Command+h", "Command+1"}
)

// App represents the overall OS GUI hardware, and creates Images, Textures
// and Windows, appropriate for that hardware / OS, and maintains data about
// the physical screen(s)
type App interface {

	// Platform returns the platform type, which can be used
	// for conditionalizing behavior
	Platform() Platforms

	// SystemPlatform returns the platform type of the underlying
	// system, which can be used for conditionalizing behavior. On platforms
	// other than [Web], this is the same as [App.Platform]. On [Web], it
	// returns the platform of the underlying operating system.
	SystemPlatform() Platforms

	// Name is the overall name of the application -- used for specifying an
	// application-specific preferences directory, etc
	Name() string

	// SetName sets the application name -- defaults to Cogent Core if not otherwise set
	SetName(name string)

	// GetScreens gets the current list of screens
	GetScreens()

	// NScreens returns the number of different logical and/or physical
	// screens managed under this overall screen hardware
	NScreens() int

	// Screen returns screen for given screen number, or nil if not a
	// valid screen number.
	Screen(n int) *Screen

	// ScreenByName returns screen for given screen name, or nil if not a
	// valid screen name.
	ScreenByName(name string) *Screen

	// NWindows returns the number of windows open for this app.
	NWindows() int

	// Window returns given window in list of windows opened under this screen
	// -- list is not in any guaranteed order, but typically in order of
	// creation (see also WindowByName) -- returns nil for invalid index.
	Window(win int) Window

	// WindowByName returns given window in list of windows opened under this
	// screen, by name -- nil if not found.
	WindowByName(name string) Window

	// WindowInFocus returns the window currently in focus (receiving keyboard
	// input) -- could be nil if none are.
	WindowInFocus() Window

	// ContextWindow returns the window passed as context for clipboard, cursor, etc calls.
	ContextWindow() Window

	// NewWindow returns a new Window for this screen. A nil opts is valid and
	// means to use the default option values.
	NewWindow(opts *NewWindowOptions) (Window, error)

	// RemoveWindow removes the given Window from the app's list of windows.
	// It does not actually close it; see [Window.Close] for that.
	RemoveWindow(win Window)

	// Clipboard returns the [Clipboard] handler for the system,
	// in context of given window, which is optional (can be nil)
	// but can provide useful context on some systems.
	Clipboard(win Window) Clipboard

	// Cursor returns the [Cursor] handler for the system, in the context
	// of the given window.
	Cursor(win Window) Cursor

	// DataDir returns the OS-specific data directory: Mac: ~/Library,
	// Linux: ~/.config, Windows: ~/AppData/Roaming
	DataDir() string

	// AppDataDir returns the application-specific data directory: [App.DataDir] + [App.Name].
	// It ensures that the directory exists first.
	AppDataDir() string

	// CogentCoreDataDir returns the Cogent Core data directory: [App.DataDir] + "CogentCoreDataDir".
	// It ensures that the directory exists first.
	CogentCoreDataDir() string

	// OpenURL opens the given URL in the user's default browser.  On Linux
	// this requires that xdg-utils package has been installed -- uses
	// xdg-open command.
	OpenURL(url string)

	// OpenFiles returns file names that have been set to be open at startup.
	OpenFiles() []string

	// SetQuitReqFunc sets the function that is called whenever there is a
	// request to quit the app (via a OS or a call to QuitReq() method).  That
	// function can then adjudicate whether and when to actually call Quit.
	SetQuitReqFunc(fun func())

	// AddQuitCleanFunc adds the given function to a list that is called whenever
	// app is actually about to quit (irrevocably). Can do any necessary
	// last-minute cleanup here.
	AddQuitCleanFunc(fun func())

	// QuitReq is a quit request, triggered either by OS or user call (e.g.,
	// via Quit menu action) -- calls function previously registered by
	// SetQuitReqFunc, which is then solely responsible for actually calling
	// Quit.
	QuitReq()

	// IsQuitting returns true when the app is actually quitting -- it is set
	// to true just before the QuitClean function is called, and before all
	// the windows are closed.
	IsQuitting() bool

	// QuitClean calls the function setup in SetQuitCleanFunc and does other
	// app cleanup -- called on way to quitting. If it returns false, then
	// the app should not quit.
	QuitClean() bool

	// Quit closes all windows and exits the program.
	Quit()

	// MainLoop runs the main loop of the app.
	MainLoop()

	// RunOnMain runs given function on main thread (where [App.MainLoop] is running).
	// Some functions (GUI-specific etc) must run on this initial main thread for the
	// overall app. If [App.MainLoop] has not been called yet, RunOnMain assumes that
	// it is being called from the main thread and thus just calls the given function
	// directly.
	RunOnMain(f func())

	// SendEmptyEvent sends an empty, blank event to global event processing
	// system, which has the effect of pushing the system along during cases when
	// the event loop needs to be "pinged" to get things moving along.
	// See also similar method on Window.
	SendEmptyEvent()

	// ShowVirtualKeyboard shows a virtual keyboard of the given type.
	// ShowVirtualKeyboard only has an effect on mobile platforms (iOS, Android, and Web).
	// It should not be called with [styles.NoKeyboard].
	ShowVirtualKeyboard(typ styles.VirtualKeyboards)

	// HideVirtualKeyboard hides the virtual keyboard.
	// HideVirtualKeyboard only has an effect on mobile platforms (iOS, Android, and Web).
	HideVirtualKeyboard()

	// IsDark returns whether the system color theme is dark (as oppposed to light)
	IsDark() bool
}

// OnSystemWindowCreated is a channel used to communicate that the underlying
// system window has been created on iOS and Android. If it is nil, it indicates
// that the current platform does not have an underlying system window that is
// created asynchronously, or that system window has already been created and
// thus this is no longer applicable. If it is non-nil, no actions with the window
// should be taken until a signal is sent.
var OnSystemWindowCreated chan struct{}

// Platforms are all the supported platforms for system
type Platforms int32 //enums:enum

const (
	// MacOS is a Mac OS machine (aka Darwin)
	MacOS Platforms = iota

	// Linux is a Linux OS machine
	Linux

	// Windows is a Microsoft Windows machine
	Windows

	// IOS is an Apple iOS or iPadOS mobile phone or iPad
	IOS

	// Android is an Android mobile phone or tablet
	Android

	// Web is a web browser running the app through WASM
	Web

	// Offscreen is an offscreen driver typically used for testing,
	// specified using the "offscreen" build tag
	Offscreen
)

// IsMobile returns whether the platform is a mobile platform
// (iOS, Android, Web, or Offscreen). Web and Offscreen are
// considered mobile platforms because they only support one window.
func (p Platforms) IsMobile() bool {
	return p == IOS || p == Android || p == Web || p == Offscreen
}
