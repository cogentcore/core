// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"sync/atomic"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/system/composer"
)

// getImager is implemented by offscreen.Drawer for [Body.AssertRender].
type getImager interface {
	GetImage() *image.RGBA
}

// AssertRender makes a new window from the body, waits until it is shown
// and all events have been handled, does any necessary re-rendering,
// asserts that its rendered image is the same as that stored at the given
// filename, saving the image to that filename if it does not already exist,
// and then closes the window. It does not return until all of those steps
// are completed. Each (optional) function passed is called after the
// window is shown, and all system events are handled before proessing continues.
// A testdata directory and png file extension are automatically added to
// the the filename, and forward slashes are automatically replaced with
// backslashes on Windows.
func (b *Body) AssertRender(t imagex.TestingT, filename string, fun ...func()) {
	b.runAndShowNewWindow()

	rw := b.Scene.RenderWindow()
	for i := 0; i < len(fun); i++ {
		fun[i]()
		b.waitNoEvents(rw)
	}
	if len(fun) == 0 {
		// we didn't get it above
		b.waitNoEvents(rw)
	}

	// Ensure that everything is updated and rendered. If there are no changes,
	// the performance impact is minimal.
	// A render that is still being composed in another goroutine would
	// make renderWindow skip its render, so we wait for it first.
	for range 10 { // note: 10 is essential for textcore tests
		rw.waitRenderDone()
		rw.renderWindow()
		b.AsyncLock()
		rw.mains.runDeferred()
		b.AsyncUnlock()
	}
	rw.waitRenderDone()

	// The image is copied under the window lock, which is held while it is
	// composed, as paint events can still make the event loop render.
	dw := rw.SystemWindow.Composer().(*composer.ComposerDrawer).Drawer
	if !rw.SystemWindow.Lock() {
		t.Errorf("AssertRender: the window of %s was closed", filename)
		return
	}
	img := imagex.CloneAsRGBA(dw.(getImager).GetImage())
	rw.SystemWindow.Unlock()
	imagex.Assert(t, img, filename)

	// When closing the scene, our access to the render context stops working,
	// so using normal AsyncLock and AsyncUnlock will lead to AsyncLock failing.
	// That leaves the lock on, which prevents the WinClose event from being
	// received. Therefore, we get the rc ahead of time.
	rc := b.Scene.renderContext()
	rc.Lock()
	b.Close()
	rc.Unlock()
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

// waitRenderDone waits until any render of the window that is being
// composed in another goroutine (see [renderWindow.renderAsync]) is done.
func (w *renderWindow) waitRenderDone() {
	for atomic.LoadInt32(&w.winRenderCounter) > 0 {
		time.Sleep(time.Millisecond)
	}
}

// waitNoEvents waits until the event loop has handled an event and there
// are no events left, which includes any rendering of the body necessary,
// as rendering is done in response to the periodic paint events.
func (b *Body) waitNoEvents(rw *renderWindow) {
	ch := make(chan struct{})
	rw.noEventsMu.Lock()
	rw.noEventsChan = ch
	rw.noEventsMu.Unlock()
	<-ch
}
