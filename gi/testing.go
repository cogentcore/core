// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grows/images"
)

// AssertRender makes a new window from the body, waits until it is shown
// and all events have been handled, does any necessary re-rendering,
// asserts that its rendered image is the same as that stored at the given
// filename, saving the image to that filename if it does not already exist,
// and then closes the window. It does not return until all of those steps
// are completed. If a function is passed for the final argument, it is called
// after the scene is shown. See [Body.AssertScreenRender] for a version that
// asserts the rendered image of the entire screen, not just this body.
func (b *Body) AssertRender(t images.TestingT, filename string, fun ...func()) {
	b.RunAndShowNewWindow()
	if len(fun) > 0 {
		fun[0]()
	}
	b.WaitNoEvents()

	b.Scene.AssertPixels(t, filename)
	b.Close()
}

// AssertScreenRender is the same as [Body.AssertRender] except that it asserts the
// rendered image of the entire screen, not just this body. It should be used for
// multi-scene tests like those of snackbars and dialogs.
func (b *Body) AssertScreenRender(t images.TestingT, filename string, fun ...func()) {
	b.RunAndShowNewWindow()
	if len(fun) > 0 {
		fun[0]()
	}
	b.WaitNoEvents()

	goosi.AssertCapture(t, filename)
	b.Close()
}

// RunAndShowNewWindow runs a new window and waits for it to be shown.
// It it used internally in test infrastructure, and it should typically
// not be used by end users.
func (b *Body) RunAndShowNewWindow() {
	showed := make(chan struct{})
	b.OnShow(func(e events.Event) {
		showed <- struct{}{}
	})
	b.NewWindow().Run()
	<-showed
}

// WaitNoEvents waits for all events to be handled and does any rendering
// of the body necessary. It it used internally in test infrastructure, and
// it should typically not be used by end users.
func (b *Body) WaitNoEvents() {
	rw := b.Scene.RenderWin()
	rw.NoEventsChan = make(chan struct{})
	<-rw.NoEventsChan
	rw.NoEventsChan = nil

	b.DoNeedsRender()
}

// AssertPixels asserts that [Scene.Pixels] is equivalent
// to the image stored at the given filename in the testdata directory,
// with ".png" added to the filename if there is no extension
// (eg: "button" becomes "testdata/button.png").
// If it is not, it fails the test with an error, but continues its
// execution. If there is no image at the given filename in the testdata
// directory, it creates the image.
func (sc *Scene) AssertPixels(t images.TestingT, filename string) {
	images.Assert(t, sc.Pixels, filename)
}
