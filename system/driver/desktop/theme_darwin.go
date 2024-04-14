// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Code based on https://gist.github.com/jerblack/869a303d1a604171bf8f00bbbefa59c2
// MIT License
// Copyright 2022 Jeremy Black
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

//go:build darwin

package desktop

import (
	"os"
	"os/exec"
	"path/filepath"

	"log/slog"

	"github.com/fsnotify/fsnotify"
)

const plistPath = `/Library/Preferences/.GlobalPreferences.plist`

var plist = filepath.Join(os.Getenv("HOME"), plistPath)

// IsDark returns whether the system color theme is dark (as opposed to light).
func (a *App) IsDark() bool {
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false
		} else {
			slog.Error("unexpected error when running command to get system color theme: " + err.Error())
			return false
		}
	}
	return true
}

// IsDarkMonitor monitors whether the system is in dark mode.
func (a *App) IsDarkMonitor() {
	// TODO: do we need to close gracefully here if the app is done?
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("error creating file watcher for system color theme: " + err.Error())
		return
	}

	err = watcher.Add(plist)
	if err != nil {
		slog.Error("error adding system color theme file watcher: " + err.Error())
	}

	defer watcher.Close()
	wasDark := a.IsDark() // we need to store this so that we only update when it changes
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				isDark := a.IsDark()
				if isDark != wasDark {
					a.Dark = isDark
					wasDark = isDark
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			slog.Error("system color theme watcher error: " + err.Error())
		}
	}
}
