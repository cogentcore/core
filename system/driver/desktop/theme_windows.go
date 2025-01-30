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
	"fmt"
	"log/slog"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	themeRegKey  = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize` // in HKCU
	themeRegName = `AppsUseLightTheme`                                            // <- For apps. Use SystemUsesLightTheme for taskbar and tray
)

// IsDark returns whether the system color theme is dark (as opposed to light).
func (app *App) IsDark() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, themeRegKey, registry.QUERY_VALUE)
	if err != nil {
		slog.Error("error opening theme registry key: " + err.Error())
		return false
	}
	defer k.Close()
	val, _, err := k.GetIntegerValue(themeRegName)
	if err != nil {
		slog.Error("error getting theme registry value: " + err.Error())
		return false
	}
	// dark mode is 0
	return val == 0
}

// TODO(kai): fix IsDarkMonitor on windows

// IsDarkMonitor monitors the state of the dark mode and calls the given function
// with the new value whenever it changes. It returns a channel that will
// receive any errors that occur during the monitoring, as it happens in a
// separate goroutine. It also returns any error that occurred during the
// initial set up of the monitoring. If the error is non-nil, the error channel
// will be nil. It also takes a done channel, and it will stop monitoring when
// that done channel is closed.
func (app *App) IsDarkMonitor(fn func(isDark bool), done chan struct{}) (chan error, error) {
	var regNotifyChangeKeyValue *syscall.Proc

	if advapi32, err := syscall.LoadDLL("Advapi32.dll"); err == nil {
		if p, err := advapi32.FindProc("RegNotifyChangeKeyValue"); err == nil {
			regNotifyChangeKeyValue = p
		} else {
			return nil, fmt.Errorf("error finding function RegNotifyChangeKeyValue in Advapi32.dll: %w", err)
		}
	}

	ec := make(chan error)
	if regNotifyChangeKeyValue != nil {
		go func() {
			k, err := registry.OpenKey(registry.CURRENT_USER, themeRegKey, syscall.KEY_NOTIFY|registry.QUERY_VALUE)
			if err != nil {
				ec <- fmt.Errorf("error opening theme registry key: %w", err)
				return
			}
			// need haveSetWasDark to capture first change correctly
			var wasDark, haveSetWasDark bool
			for {
				select {
				case <-done:
					// if done is closed, we return
					return
				default:
					regNotifyChangeKeyValue.Call(uintptr(k), 0, 0x00000001|0x00000004, 0, 0)
					val, _, err := k.GetIntegerValue(themeRegName)
					if err != nil {
						ec <- fmt.Errorf("error getting theme registry value: %w", err)
						return
					}
					// dark mode is 0
					isDark := val == 0

					if isDark != wasDark || !haveSetWasDark {
						fn(isDark)
						wasDark = isDark
						haveSetWasDark = true
					}
				}
			}
		}()
	}
	return ec, nil
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
