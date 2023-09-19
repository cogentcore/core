// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Code heavily based on https://gist.github.com/jerblack/869a303d1a604171bf8f00bbbefa59c2
// MIT License
// Copyright 2022 Jeremy Black
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

//go:build darwin

package goosi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

const plistPath = `/Library/Preferences/.GlobalPreferences.plist`

var plist = filepath.Join(os.Getenv("HOME"), plistPath)

// ThemeIsDark returns whether the system color theme is dark (as opposed to light),
// and any error that occurred when getting that information.
func ThemeIsDark() (bool, error) {
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		} else {
			return false, fmt.Errorf("unexpected error when running command to get system color theme: %w", err)
		}
	}
	return true, nil
}

// MonitorTheme monitors the state of the dark mode and calls the given function
// with the new value whenever it changes. It returns a channel that will
// receive any errors that occur during the monitoring, as it happens in a
// separate goroutine. It also returns any error that occurred during the
// initial set up of the monitoring. If the error is non-nil, the error channel
// will be nil.
func MonitorTheme(fn func(isDark bool)) (chan error, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("error creating file watcher: %w", err)
	}
	defer watcher.Close()

	ec := make(chan error)
	go func() {
		wasDark, err := ThemeIsDark() // we need to store this so that we only update when it changes
		if err != nil {
			ec <- fmt.Errorf("error while getting theme: %w", err)
			return
		}
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					isDark, err := ThemeIsDark()
					if err != nil {
						ec <- fmt.Errorf("error while getting theme: %w", err)
						return
					}
					if isDark != wasDark {
						fn(isDark)
						wasDark = isDark
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				ec <- fmt.Errorf("watcher error: %w", err)
				return
			}
		}
	}()

	err = watcher.Add(plist)
	if err != nil {
		return nil, fmt.Errorf("error adding file watcher: %w", err)
	}
	return ec, nil
}
