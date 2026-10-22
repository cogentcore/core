// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCallFunc(t *testing.T) {
	called := false
	myFunc := func() {
		called = true
	}
	CallFunc(nil, myFunc)
	assert.True(t, called)
}

func TestCallFuncArgs(t *testing.T) {
	b := NewBody()
	myFunc := func(a int, b string) {}
	CallFunc(b, myFunc)
	b.AssertRender(t, "func-button/args")
}

// callFuncItem is a struct type, so that a slice of it ([callFuncItems],
// which is named to name the argument) is edited with a [ListButton],
// which opens a dialog.
type callFuncItem struct {
	A, B int
}

type callFuncItems []callFuncItem

// A function with a single argument that has a value button, called with
// a context in another window, is called in the event loop of that window,
// instead of locking it from the event loop of the button, which could
// deadlock. This test is also for the race detector.
func TestCallFuncArgOtherWindow(t *testing.T) {
	a := NewBody()
	fb := NewFuncButton(a)
	a.runAndShowNewWindow()
	b := NewBody()
	ctx := NewText(b).SetText("Context")
	b.runAndShowNewWindow()

	called := make(chan struct{})
	fb.SetFunc(func(items callFuncItems) {
		ctx.SetText("Called").Update() // needs the lock of the window of b
		close(called)
	}).SetContext(ctx)

	// The button opens the dialog of the argument in the window of a.
	a.AsyncLock()
	fb.CallFunc()
	a.AsyncUnlock()
	rwa := a.Scene.RenderWindow()
	a.waitNoEvents(rwa)

	// Closing the dialog accepts it.
	a.AsyncLock()
	d := rwa.mains.top().Scene
	assert.NotEqual(t, a.Scene, d, "the dialog is not open")
	d.Close()
	a.AsyncUnlock()
	select {
	case <-called:
	case <-time.After(10 * time.Second):
		t.Fatal("the function was not called")
	}

	b.AsyncLock()
	assert.Equal(t, "Called", ctx.Text)
	b.AsyncUnlock()
	for _, bd := range []*Body{a, b} {
		rc := bd.Scene.renderContext()
		rc.Lock()
		bd.Close()
		rc.Unlock()
	}
}
