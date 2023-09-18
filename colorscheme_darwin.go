// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Code heavily based on https://gist.github.com/jerblack/1d05bbcebb50ad55c312e4d7cf1bc909
// MIT License
// Copyright 2022 Jeremy Black
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

//go:build darwin

package goosi

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

const plistPath = `/Library/Preferences/.GlobalPreferences.plist`

var plist = filepath.Join(os.Getenv("HOME"), plistPath)
var wasDark bool

func main() {
	// get initial state
	wasDark = checkDarkMode()

	// Start watcher and give it a function to call when the state changes
	startWatcher(react)
}

// react to the change
func react(isDark bool) {
	if isDark {
		fmt.Println("Dark Mode ON")
	} else {
		fmt.Println("Dark Mode OFF")
	}
}

func checkDarkMode() bool {
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false
		}
	}
	return true
}

func startWatcher(fn func(bool)) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					isDark := checkDarkMode()
					if isDark && !wasDark {
						fn(isDark)
						wasDark = isDark
					}
					if !isDark && wasDark {
						fn(isDark)
						wasDark = isDark
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(plist)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
