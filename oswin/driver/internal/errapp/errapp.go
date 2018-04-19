// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package errapp provides a stub App implementation.
package errapp

import (
	"image"

	"github.com/rcoreilly/goki/gi/oswin"
)

// Stub returns an App whose methods all return the given error.
func Stub(err error) oswin.App {
	return stub{err}
}

type stub struct {
	err error
}

func (s stub) NewImage(size image.Point) (oswin.Image, error)               { return nil, s.err }
func (s stub) NewTexture(size image.Point) (oswin.Texture, error)           { return nil, s.err }
func (s stub) NewWindow(opts *oswin.NewWindowOptions) (oswin.Window, error) { return nil, s.err }
func (s stub) NScreens() int                                                { return 0 }
func (s stub) Screen(scrN int) *oswin.Screen                                { return nil }
func (s stub) NWindows() int                                                { return 0 }
func (s stub) Window(win int) oswin.Window                                  { return nil }
func (s stub) WindowByName(name string) oswin.Window                        { return nil }

// check for interface implementation
var _ oswin.App = &stub{}
