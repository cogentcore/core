// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Code based on https://gist.github.com/jerblack/1d05bbcebb50ad55c312e4d7cf1bc909
// MIT License
// Copyright 2022 Jeremy Black
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

//go:build windows

package desktop

import (
	"log/slog"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	themeRegKey  = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize` // in HKCU
	themeRegName = `AppsUseLightTheme`                                            // <- For apps. Use SystemUsesLightTheme for taskbar and tray
)

// watchTheme sets the initial system color theme and starts watching the
// registry value for it for changes.
func (a *App) watchTheme() {
	k, err := registry.OpenKey(registry.CURRENT_USER, themeRegKey, registry.QUERY_VALUE|registry.NOTIFY)
	if err != nil {
		slog.Error("error opening theme registry key: " + err.Error())
		return
	}
	a.dark.Store(isDarkKey(k))
	go func() {
		defer k.Close()
		for {
			// this blocks until the key changes
			err := windows.RegNotifyChangeKeyValue(windows.Handle(k), false, windows.REG_NOTIFY_CHANGE_LAST_SET, 0, false)
			if err != nil {
				slog.Error("error watching theme registry key: " + err.Error())
				return
			}
			a.setDark(isDarkKey(k))
		}
	}()
}

// isDarkKey returns whether the system color theme is dark,
// based on the given theme registry key.
func isDarkKey(k registry.Key) bool {
	val, _, err := k.GetIntegerValue(themeRegName)
	if err != nil {
		// the value does not exist on versions of Windows without a dark theme
		return false
	}
	// dark mode is 0
	return val == 0
}

func (w *Window) SetTitleBarIsDark(isDark bool) {
	if !w.IsVisible() {
		return
	}

	value := int32(0)
	if isDark {
		value = 1
	}

	dll := syscall.NewLazyDLL("dwmapi.dll")
	fun := dll.NewProc("DwmSetWindowAttribute")

	// set an DWMWA_USE_IMMERSIVE_DARK_MODE (20) attribute
	// of type BOOL (typedef int BOOL, int is 32 bit integer here)
	// to a value (on or off)
	// that has size in bytes 4
	// for a HWND window
	ret, _, err := fun.Call(
		uintptr(unsafe.Pointer(w.Glw.GetWin32Window())), // HWND
		20,                              //DWMWA_USE_IMMERSIVE_DARK_MODE
		uintptr(unsafe.Pointer(&value)), // on or off
		4,                               // sizeof(BOOL) for Win32 API
	)

	// HRESULT S_OK = 0 (ret), everything else is an error
	if ret != 0 {
		slog.Error("failed to set window title bar color",
			"hresult", ret,
			"error", err)
	}
}
