// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"sync"
	"testing"
	"time"

	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// The menu search dialog lists the stages of other windows while their
// stacks change, and raises a stage of another window in the event loop
// of that window. This test is also for the race detector.
func TestMenuSearchOtherWindow(t *testing.T) {
	a := NewBody("Window A")
	a.runAndShowNewWindow()
	b := NewBody("Window B")
	b.runAndShowNewWindow()
	rwa, rwb := a.Scene.RenderWindow(), b.Scene.RenderWindow()

	// A dialog in the window of b, which is raised from the search.
	b.AsyncLock()
	NewBody("Dialog B").RunDialog(b)
	b.AsyncUnlock()
	b.waitNoEvents(rwb)

	// Meanwhile, the window of b opens and closes other dialogs.
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			b.AsyncLock()
			d := NewBody("Temporary")
			d.RunDialog(b)
			b.AsyncUnlock()
			b.AsyncLock()
			d.Close()
			b.AsyncUnlock()
		}
	}()

	a.AsyncLock()
	a.Scene.MenuSearchDialog("Search", "Search for a stage")
	a.AsyncUnlock()
	a.waitNoEvents(rwa)

	var ch *Chooser
	var raise func()
	a.AsyncLock()
	rwa.mains.top().Scene.WalkDown(func(n tree.Node) bool {
		if c, ok := n.(*Chooser); ok {
			ch = c
			return tree.Break
		}
		return tree.Continue
	})
	for range 20 {
		ch.Items = nil
		ch.CallItemsFuncs()
	}
	for _, it := range ch.Items {
		if it.Text == "Dialog B" {
			raise = it.Func
		}
	}
	a.AsyncUnlock()
	close(stop)
	wg.Wait()
	if !assert.NotNil(t, raise, "the dialog of the other window is not listed") {
		return
	}

	// The window of b is made the top again by another dialog, and then
	// the dialog is raised from the event loop of a.
	b.AsyncLock()
	NewBody("Other").RunDialog(b)
	b.AsyncUnlock()
	b.waitNoEvents(rwb)
	a.AsyncLock()
	raise()
	a.AsyncUnlock()
	deadline := time.Now().Add(10 * time.Second)
	for {
		b.AsyncLock()
		title := rwb.mains.top().Title
		b.AsyncUnlock()
		if title == "Dialog B" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the dialog was not raised: the top stage is %q", title)
		}
		time.Sleep(10 * time.Millisecond)
	}

	for _, bd := range []*Body{a, b} {
		rc := bd.Scene.renderContext()
		rc.Lock()
		bd.Close()
		rc.Unlock()
	}
}
