// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package errapp provides a stub App implementation.
package errapp

import (
	"image"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/clip"
	"github.com/goki/gi/oswin/cursor"
)

// Stub returns an App whose methods all return the given error.
func Stub(err error) oswin.App {
	return stub{err}
}

type stub struct {
	err error
}

func (s stub) NewImage(size image.Point) (oswin.Image, error) { return nil, s.err }
func (s stub) NewTexture(win oswin.Window, size image.Point) (oswin.Texture, error) {
	return nil, s.err
}
func (s stub) NewWindow(opts *oswin.NewWindowOptions) (oswin.Window, error) { return nil, s.err }
func (s stub) NScreens() int                                                { return 0 }
func (s stub) Screen(scrN int) *oswin.Screen                                { return nil }
func (s stub) ScreenByName(name string) *oswin.Screen                       { return nil }
func (s stub) NWindows() int                                                { return 0 }
func (s stub) Window(win int) oswin.Window                                  { return nil }
func (s stub) WindowByName(name string) oswin.Window                        { return nil }
func (s stub) WindowInFocus() oswin.Window                                  { return nil }
func (s stub) ContextWindow() oswin.Window                                  { return nil }
func (s stub) ClipBoard(win oswin.Window) clip.Board                        { return nil }
func (s stub) Cursor(win oswin.Window) cursor.Cursor                        { return nil }

func (s stub) Platform() oswin.Platforms   { return oswin.PlatformsN }
func (s stub) Name() string                { return "" }
func (s stub) SetName(name string)         {}
func (s stub) PrefsDir() string            { return "" }
func (s stub) GoGiPrefsDir() string        { return "" }
func (s stub) AppPrefsDir() string         { return "" }
func (s stub) FontPaths() []string         { return nil }
func (s stub) About() string               { return "" }
func (s stub) SetAbout(about string)       {}
func (s stub) OpenURL(url string)          {}
func (s stub) SetQuitReqFunc(fun func())   {}
func (s stub) SetQuitCleanFunc(fun func()) {}
func (s stub) IsQuitting() bool            { return false }
func (s stub) QuitReq()                    {}
func (s stub) QuitClean()                  {}
func (s stub) Quit()                       {}

// check for interface implementation
var _ oswin.App = &stub{}
