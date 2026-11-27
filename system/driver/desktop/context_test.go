// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestClipboardNoWindow tests the clipboard of a widget without a window,
// as for one in a snackbar that has closed: it keeps the last context
// window, without a panic, and does not leave the mutex locked.
func TestClipboardNoWindow(t *testing.T) {
	a := &App{}
	w := &Window{}
	a.CtxWindow = w
	assert.NotPanics(t, func() {
		assert.Equal(t, TheClipboard, a.Clipboard(nil))
		a.Cursor(nil)
	})
	assert.Same(t, w, a.CtxWindow)
	assert.True(t, a.Mu.TryLock(), "not left locked")
	a.Mu.Unlock()
	w2 := &Window{}
	a.Clipboard(w2)
	assert.Same(t, w2, a.CtxWindow)
}
