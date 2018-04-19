// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package errscreen provides a stub Screen implementation.
package errscreen

import (
	"image"

	"github.com/rcoreilly/goki/gi/oswin"
)

// Stub returns a Screen whose methods all return the given error.
func Stub(err error) oswin.Screen {
	return stub{err}
}

type stub struct {
	err error
}

func (s stub) NewImage(size image.Point) (oswin.Image, error)               { return nil, s.err }
func (s stub) NewTexture(size image.Point) (oswin.Texture, error)           { return nil, s.err }
func (s stub) NewWindow(opts *oswin.NewWindowOptions) (oswin.Window, error) { return nil, s.err }
func (s stub) NScreens() int                                                { return 0 }
func (s stub) ScreenData(scrN int) *oswin.ScreenData                        { return nil }
func (s stub) NWindows() int                                                { return 0 }
func (s stub) Window(win int) oswin.Window                                  { return nil }

// check for interface implementation
var _ oswin.Screen = &stub{}
