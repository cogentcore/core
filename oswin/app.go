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
)

// TheApp is the current oswin App -- only ever one in effect
var TheApp App

// App represents the overall OS GUI hardware, and creates Images, Textures
// and Windows, appropriate for that hardware / OS, and maintains data about
// the physical screen(s)
type App interface {
	// NScreens returns the number of different logical and/or physical
	// screens managed under this overall screen hardware
	NScreens() int

	// Screen returns screen for given screen number, or nil if not a
	// valid screen number
	Screen(scrN int) *Screen

	// NWindows returns the number of windows open for this app
	NWindows() int

	// Window returns given window in list of windows opened under this screen
	// -- list is not in any guaranteed order, but typically in order of
	// creation (see also WindowByName) -- returns nil for invalid index
	Window(win int) Window

	// WindowByName returns given window in list of windows opened under this
	// screen, by name -- nil if not found
	WindowByName(name string) Window

	// NewWindow returns a new Window for this screen. A nil opts is valid and
	// means to use the default option values.
	NewWindow(opts *NewWindowOptions) (Window, error)

	// NewImage returns a new Image for this screen.  Images can be drawn upon
	// directly using image and other packages, and have an accessable []byte
	// slice holding the image data
	NewImage(size image.Point) (Image, error)

	// NewTexture returns a new Texture for the given window.  Textures are opaque
	// and could be non-local, but very fast for rendering to windows --
	// typically create a texture of each window and render to that texture,
	// then Draw that texture to the window when it is time to update (call
	// Publish on window after drawing)
	NewTexture(win Window, size image.Point) (Texture, error)
}
