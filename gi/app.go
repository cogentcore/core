// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "goki.dev/goosi"

// App wrappers for goosi so that the end-user doesn't need to import it

// SetAppName sets the application name, which defaults to GoGi if not otherwise set.
// Name appears in the first app menu, and specifies the default application-specific
// preferences directory, etc
func SetAppName(name string) {
	goosi.TheApp.SetName(name)
}

// AppName returns the application name; see SetAppName to set
func AppName() string {
	return goosi.TheApp.Name()
}

// SetAppAbout sets the 'about' info for the app, which appears as a menu option
// in the default app menu
func SetAppAbout(about string) {
	goosi.TheApp.SetAbout(about)
}

// SetQuitReqFunc sets the function that is called whenever there is a
// request to quit the app (via a OS or a call to QuitReq() method).  That
// function can then adjudicate whether and when to actually call Quit.
func SetQuitReqFunc(fun func()) {
	goosi.TheApp.SetQuitReqFunc(fun)
}

// SetQuitCleanFunc sets the function that is called whenever app is
// actually about to quit (irrevocably) -- can do any necessary
// last-minute cleanup here.
func SetQuitCleanFunc(fun func()) {
	goosi.TheApp.SetQuitCleanFunc(fun)
}

// Quit closes all windows and exits the program.
func Quit() {
	if !goosi.TheApp.IsQuitting() {
		goosi.TheApp.Quit()
	}
}

// QuitReq requests to Quit -- calls QuitReqFunc if present
func QuitReq() {
	goosi.TheApp.QuitReq()
}

// PollEvents tells the main event loop to check for any gui events right now.
// Call this periodically from longer-running functions to ensure
// GUI responsiveness.
func PollEvents() {
	goosi.TheApp.PollEvents()
}

// OpenURL opens the given URL in the user's default browser.  On Linux
// this requires that xdg-utils package has been installed -- uses
// xdg-open command.
func OpenURL(url string) {
	goosi.TheApp.OpenURL(url)
}

// AppPrefsDir returns the application-specific preferences directory:
// [PrefsDir] + [AppName]. It ensures that the directory exists first.
func AppPrefsDir() string {
	return goosi.TheApp.AppPrefsDir()
}

// GoGiPrefsDir returns the GoGi preferences directory: [PrefsDir] + "GoGi".
// It ensures that the directory exists first.
func GoGiPrefsDir() string {
	return goosi.TheApp.GoGiPrefsDir()
}

// PrefsDir returns the OS-specific preferences directory: Mac: ~/Library,
// Linux: ~/.config, Windows: ~/AppData/Roaming
func PrefsDir() string {
	return goosi.TheApp.PrefsDir()
}
