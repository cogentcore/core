// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/system"
)

// AssertRender makes a new window from the body, waits until it is shown
// and all events have been handled, does any necessary re-rendering,
// asserts that its rendered image is the same as that stored at the given
// filename, saving the image to that filename if it does not already exist,
// and then closes the window. It does not return until all of those steps
// are completed. Each (optional) function passed is called after the
// window is shown, and all system events are handled before proessing continues.
// A testdata directory and png file extension are automatically added to
// the the filename, and forward slashes are automatically replaced with
// backslashes on Windows. See [Body.AssertRenderScreen] for a version
// that asserts the rendered image of the entire screen, not just this body.
func (b *Body) AssertRender(t imagex.TestingT, filename string, fun ...func()) {
	b.runAndShowNewWindow()
	for i := 0; i < len(fun); i++ {
		fun[i]()
		b.waitNoEvents()
	}
	if len(fun) == 0 {
		// we didn't get it above
		b.waitNoEvents()
	}

	b.Scene.AssertPixels(t, filename)
	b.Close()
}

// AssertRenderScreen is the same as [Body.AssertRender] except that it asserts the
// rendered image of the entire screen, not just this body. It should be used for
// multi-scene tests like those of snackbars and dialogs.
func (b *Body) AssertRenderScreen(t imagex.TestingT, filename string, fun ...func()) {
	b.runAndShowNewWindow()
	for i := 0; i < len(fun); i++ {
		fun[i]()
		b.waitNoEvents()
	}
	if len(fun) == 0 {
		// we didn't get it above
		b.waitNoEvents()
	}

	system.AssertCapture(t, filename)
	b.Close()
}

// runAndShowNewWindow runs a new window and waits for it to be shown.
func (b *Body) runAndShowNewWindow() {
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) {
		showed <- struct{}{}
	})
	b.RunWindow()
	<-showed
}

// waitNoEvents waits for all events to be handled and does any rendering
// of the body necessary.
func (b *Body) waitNoEvents() {
	rw := b.Scene.RenderWindow()
	rw.noEventsChan = make(chan struct{})
	<-rw.noEventsChan
	rw.noEventsChan = nil

	b.AsyncLock()
	b.DoNeedsRender()
	b.AsyncUnlock()
}

// AssertPixels asserts that [Scene.Pixels] is equivalent
// to the image stored at the given filename in the testdata directory,
// with ".png" added to the filename if there is no extension
// (eg: "button" becomes "testdata/button.png"). Forward slashes are
// automatically replaced with backslashes on Windows.
// If it is not, it fails the test with an error, but continues its
// execution. If there is no image at the given filename in the testdata
// directory, it creates the image.
func (sc *Scene) AssertPixels(t imagex.TestingT, filename string) {
	imagex.Assert(t, sc.Pixels, filename)
}
