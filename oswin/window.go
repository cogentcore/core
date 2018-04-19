// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oswin

import (
	"image"
	"unicode/utf8"
)

// Window is a double-buffered OS-specific hardware window
type Window interface {
	// Name returns the name of the window -- name is used strictly for
	// internal tracking and finding of windows -- see Title for the displayed
	// title of the window
	Name() string

	// SetName sets the name of the window
	SetName(name string)

	// Title returns the current title of the window, which is displayed in the GUI
	Title() string

	// SetTitle sets the current title of the window, which is displayed in the GUI
	SetTitle(title string)

	// Size returns the current size of the window, in raw underlying dots / pixels
	Size() image.Point

	// LogicalDPI returns the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units --
	// physical DPI can be found in the Screen
	LogicalDPI() float32

	// Geometry returns the current position and size of the window relative
	// to the screen
	Geometry() image.Rectangle

	// Screen returns the screen for this window, with all the relevant
	// information about its properties
	Screen() *Screen

	// todo: lots of other basic props of windows!

	// Release closes the window. The behavior of the Window after Release,
	// whether calling its methods or passing it as an argument, is undefined.
	Release()

	EventDeque

	Uploader

	Drawer

	// Publish flushes any pending Upload and Draw calls to the window, and
	// swaps the back buffer to the front.
	Publish() PublishResult
}

// PublishResult is the result of an Window.Publish call.
type PublishResult struct {
	// BackImagePreserved is whether the contents of the back buffer was
	// preserved. If false, the contents are undefined.
	BackImagePreserved bool
}

// NewWindowOptions are optional arguments to NewWindow.
type NewWindowOptions struct {
	// Width and Height specify the dimensions of the new window, in raw
	// pixels. If Width or Height are zero, a driver-dependent default will be
	// used for each zero value dimension.
	Width, Height int

	// Title specifies the window title.
	Title string

	// TODO: fullscreen, icon, cursorHidden?
}

// GetTitle returns a sanitized form of o.Title. In particular, its length will
// not exceed 4096, and it may be further truncated so that it is valid UTF-8
// and will not contain the NUL byte.
//
// o may be nil, in which case "" is returned.
func (o *NewWindowOptions) GetTitle() string {
	if o == nil {
		return ""
	}
	return sanitizeUTF8(o.Title, 4096)
}

func sanitizeUTF8(s string, n int) string {
	if n < len(s) {
		s = s[:n]
	}
	i := 0
	for i < len(s) {
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == 0 || (r == utf8.RuneError && n == 1) {
			break
		}
		i += n
	}
	return s[:i]
}

// WindowBase provides a base-level implementation of the generic data aspects
// of the window, including maintaining the current window size and dpi
type WindowBase struct {
	Nm     string
	Titl   string
	Sz     image.Point
	LogDPI float32
}

func (w WindowBase) Name() string {
	return w.Nm
}

func (w *WindowBase) SetName(name string) {
	w.Nm = name
}

func (w WindowBase) Title() string {
	return w.Titl
}

func (w *WindowBase) SetTitle(title string) {
	w.Titl = title
}

func (w WindowBase) Size() image.Point {
	return w.Sz
}

func (w WindowBase) LogicalDPI() float32 {
	return w.LogDPI
}
