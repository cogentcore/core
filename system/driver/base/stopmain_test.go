// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"testing"
	"time"
)

// TestStopMainBeforeLoop tests that stopping the main loop before it
// starts stops it once it starts, instead of blocking forever on a nil
// channel.
func TestStopMainBeforeLoop(t *testing.T) {
	a := &App{}
	stopped := make(chan struct{})
	go func() {
		a.StopMain() // before the loop makes its channels
		close(stopped)
	}()
	time.Sleep(10 * time.Millisecond)
	done := make(chan struct{})
	go func() {
		a.MainLoop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the main loop did not stop")
	}
	<-stopped
}
