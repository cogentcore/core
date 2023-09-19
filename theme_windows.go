// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Code heavily based on https://gist.github.com/jerblack/1d05bbcebb50ad55c312e4d7cf1bc909
// MIT License
// Copyright 2022 Jeremy Black
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

//go:build windows

package goosi

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const (
	themeRegKey  = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize` // in HKCU
	themeRegName = `AppsUseLightTheme`                                            // <- For apps. Use SystemUsesLightTheme for taskbar and tray
)

// IsDark returns whether the system color theme is dark (as opposed to light)
// and any error that occurred when getting that information.
func IsDark() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, themeRegKey, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("error opening theme registry key: %w", err)
	}
	defer k.Close()
	val, _, err := k.GetIntegerValue(themeRegName)
	if err != nil {
		return false, fmt.Errorf("error getting theme registry value: %w", err)
	}
	// dark mode is 0
	return val == 0, nil
}

// IsDarkMonitor monitors the state of the dark mode and calls the given function
// with the new value whenever it changes. It returns a channel that will
// receive any errors that occur during the monitoring, as it happens in a
// separate goroutine. It also returns any error that occurred during the
// initial set up of the monitoring. If the error is non-nil, the error channel
// will be nil. It also takes a done channel, and it will stop monitoring when
// that done channel is closed.
func IsDarkMonitor(fn func(isDark bool), done chan struct{}) (chan error, error) {
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
