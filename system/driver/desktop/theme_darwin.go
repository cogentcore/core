// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package desktop

/*
#import <Cocoa/Cocoa.h>
bool isDarkTheme();
void watchTheme();
*/
import "C"

// watchTheme sets the initial system color theme and starts watching it for
// changes, through key-value observing of the effective appearance of the
// application. It must be called on the main thread after glfw is initialized.
func (a *App) watchTheme() {
	a.dark.Store(bool(C.isDarkTheme()))
	C.watchTheme()
}

// themeChanged is called on the main thread when the effective appearance
// of the application changes.
//
//export themeChanged
func themeChanged(dark C.bool) {
	TheApp.setDark(bool(dark))
}
