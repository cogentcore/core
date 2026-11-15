// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"sync/atomic"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
)

// AssertRender makes a new window from the given body, waits until it is shown
// and all events have been handled, does any necessary re-rendering,
// asserts that its rendered image is the same as that stored at the given
// filename, saving the image to that filename if it does not already exist,
// and then closes the window. It does not return until all of those steps
// are completed. Each (optional) function passed is called after the
// window is shown, and all system events are handled before proessing continues.
// A testdata directory and png file extension are automatically added to
// the the filename, and forward slashes are automatically replaced with
// backslashes on Windows. On the web, the images are in testdata/js, as
// the browser renders text and paths differently (see [testImageName]).
func AssertRender(t imagex.TestingT, b *Body, filename string, fun ...func()) {
	img := RenderImage(b, fun...)
	if img == nil {
		t.Errorf("AssertRender: the window of %s was closed", filename)
		return
	}
	imagex.Assert(t, img, testImageName(filename))
}

// RenderImage makes a new window from the given body, waits until it is shown
// and all events have been handled, does any necessary re-rendering,
// and returns a copy of its rendered image, closing the window after.
// It is the rendering part of [AssertRender], with the same handling
// of the optional functions, which are called after the window is shown,
// and it is only for offscreen rendering (testing). It returns nil if the
// window was closed before its image could be copied.
func RenderImage(b *Body, fun ...func()) *image.RGBA {
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

	img := windowImage(rw)
	if img == nil {
		return nil
	}

	// When closing the scene, our access to the render context stops working,
	// so using normal AsyncLock and AsyncUnlock will lead to AsyncLock failing.
	// That leaves the lock on, which prevents the WinClose event from being
	// received. Therefore, we get the rc ahead of time.
	rc := b.Scene.renderContext()
	rc.Lock()
	b.Close()
	rc.Unlock()
	return img
}

// runAndShowNewWindow runs a new window and waits for it to be shown,
// in an empty window on the web (see [clearTestWindow]).
func (b *Body) runAndShowNewWindow() {
	clearTestWindow()
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
// as rendering is done in response to the periodic paint events, which
// the window is woken for, as they are paused while it is idle.
func (b *Body) waitNoEvents(rw *renderWindow) {
	ch := make(chan struct{})
	rw.noEventsMu.Lock()
	rw.noEventsChan = ch
	rw.noEventsMu.Unlock()
	rw.wake()
	<-ch
}

// Locked returns a function that calls the given function with the
// AsyncLock of the given body held, for use as one of the functions passed
// to [AssertRender] or [RenderImage], which are called from outside of the
// event loop:
//
//	AssertRender(t, b, "name", Locked(b, func() { w.Send(events.Click) }))
func Locked(b *Body, fun func()) func() {
	return func() {
		b.AsyncLock()
		defer b.AsyncUnlock()
		fun()
	}
}

// SendKey sends a [events.KeyChord] event with the given rune, code, and
// modifier keys (such as [key.Shift], which are flags, not bit masks) to
// the given widget, as if it had been typed while the widget had the focus.
func SendKey(w Widget, rn rune, code key.Codes, mods ...key.Modifiers) {
	var m key.Modifiers
	for _, mod := range mods {
		m.SetFlag(true, mod)
	}
	sendKeyMods(w, rn, code, m)
}

// sendKeyMods sends a [events.KeyChord] event with the given rune, code,
// and modifier bits to the given widget.
func sendKeyMods(w Widget, rn rune, code key.Codes, mods key.Modifiers) {
	w.AsWidget().HandleEvent(events.NewKey(events.KeyChord, rn, code, mods))
}

// TypeText sends a [events.KeyChord] event for each rune of the given
// text to the given widget, as if it had been typed.
func TypeText(w Widget, text string) {
	for _, r := range text {
		SendKey(w, r, 0)
	}
}

// SendKeyFunction sends to the given widget the [events.KeyChord] event
// for the first key chord of the given key function in the active
// [keymap.Map], so that tests do not depend on the platform keymap.
// It panics if the key function has no chord.
func SendKeyFunction(w Widget, kf keymap.Functions) {
	rn, code, mods, err := kf.Chord().Decode()
	if err != nil {
		panic(fmt.Errorf("core.SendKeyFunction: %v has no usable key chord: %w", kf, err))
	}
	sendKeyMods(w, rn, code, mods)
}
