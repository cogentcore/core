// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
package oswin provides OS-specific windows and events.
It is largely copied directly from https://github.com/skelterjohn/go.wde:

Copyright 2012 the go.wde authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Modifications and extensions were needed for supporting the full GoGi system
*/
package oswin

import (
	// "fmt"
	"image"
	"image/draw"
	// "log"
)

////////////////////////////////////////////////////////////////////////////////////////
// OS-specific window

// general interface into the operating-specific window structure
type OSWindow interface {
	SetTitle(title string)
	SetSize(width, height int)
	Size() (width, height int)
	LockSize(lock bool)
	Show()
	Screen() (im WinImage)
	FlushImage(bounds ...image.Rectangle)
	EventChan() (events <-chan interface{})
	Close() (err error)
	SetCursor(cursor Cursor)
}

// window image
type WinImage interface {
	draw.Image
	// CopyRGBA() copies the source image to this image, translating
	// the source image to the provided bounds.
	CopyRGBA(src *image.RGBA, bounds image.Rectangle)
}

/*
Some wde backends (cocoa) require that this function be called in the
main thread. To make your code as cross-platform as possible, it is
recommended that your main function look like the the code below.

	func main() {
		go theRestOfYourProgram()
		gi.RunBackendEventLoop()
	}

gi.Run() will return when gi.Stop() is called.

For this to work, you must import one of the gi backends. For
instance,

	import _ "github.com/rcoreilly/goki/gi/xgb"

or

	import _ "github.com/rcoreilly/goki/gi/win"

or

	import _ "github.com/rcoreilly/goki/gi/cocoa"


will register a backend with GoGi, allowing you to call
gi.RunBackendEventLoop(), gi.StopBackendEventLoop() and gi.NewOSWindow() without referring to the
backend explicitly.

If you pupt the registration import in a separate file filtered for
the correct platform, your project will work on all three major
platforms without configuration.

That is, if you import gi/xgb in a file named "gi_linux.go",
gi/win in a file named "gi_windows.go" and gi/cocoa in a
file named "gi_darwin.go", the go tool will import the correct one.

*/
func RunBackendEventLoop() {
	BackendRun()
}

var BackendRun = func() {
	panic("no gi backend imported")
}

/*
Call this when you want gi.Run() to return. Usually to allow your
program to exit gracefully.
*/
func StopBackendEventLoop() {
	BackendStop()
}

var BackendStop = func() {
	panic("no gi backend imported")
}

/*
Create a new OS window with the specified width and height.
*/
func NewOSWindow(width, height int) (OSWindow, error) {
	return BackendNewWindow(width, height)
}

var BackendNewWindow = func(width, height int) (OSWindow, error) {
	panic("no gi backend imported")
}
