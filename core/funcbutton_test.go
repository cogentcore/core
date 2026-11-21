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
	AssertRender(t, b, "func-button/args")
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
	skipOnWeb(t, "there is one window")
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

// callFuncSyntax is a named bool type, as textpos.MatchSyntax.
type callFuncSyntax bool

// callFuncNames is a named slice type.
type callFuncNames []string

// TestCallFuncNamedTypes tests that a function with arguments of named
// types is called with the values of the underlying types that the value
// widgets of the arguments set (as a bool from a switch), and with nil
// values, which was a panic in the Find of Cogent Code.
func TestCallFuncNamedTypes(t *testing.T) {
	var gotSyntax callFuncSyntax
	var gotNames callFuncNames
	gotN := -1
	myFunc := func(s callFuncSyntax, names callFuncNames, n int) {
		gotSyntax, gotNames, gotN = s, names, n
	}
	fb := NewFuncButton(NewBody()).SetFunc(myFunc)
	fb.Args[0].Value = true          // a bool, not a callFuncSyntax
	fb.Args[1].Value = []string{"a"} // a []string, not a callFuncNames
	fb.Args[2].Value = nil           // the zero value
	assert.NotPanics(t, fb.callFuncShowReturns)
	assert.Equal(t, callFuncSyntax(true), gotSyntax)
	assert.Equal(t, callFuncNames{"a"}, gotNames)
	assert.Equal(t, 0, gotN)
}
