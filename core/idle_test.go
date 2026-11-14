// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestIdleWake checks that the paint events of a window with nothing to
// update are paused (see idle.go), and that it is woken to render a
// change made from another goroutine, after which it is idle again.
func TestIdleWake(t *testing.T) {
	b := NewBody()
	tx := NewText(b).SetText("Before")
	b.runAndShowNewWindow()
	defer func() {
		rc := b.Scene.renderContext()
		rc.Lock()
		b.Close()
		rc.Unlock()
	}()
	rw := b.Scene.RenderWindow()
	sw, ok := rw.SystemWindow.(interface{ PaintIsIdle() bool })
	if !ok {
		t.Skip("the window does not report whether it is idle")
	}
	assert.Eventually(t, sw.PaintIsIdle, 2*time.Second, time.Millisecond, "idle")

	b.AsyncLock()
	tx.SetText("After").UpdateRender()
	assert.True(t, b.Scene.hasFlag(sceneNeedsRender))
	b.AsyncUnlock()
	assert.Eventually(t, func() bool {
		b.AsyncLock()
		defer b.AsyncUnlock()
		return !b.Scene.hasFlag(sceneNeedsRender)
	}, 2*time.Second, time.Millisecond, "woken to render the change")
	assert.Eventually(t, sw.PaintIsIdle, 2*time.Second, time.Millisecond, "idle again")
}
