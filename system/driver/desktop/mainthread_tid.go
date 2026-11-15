// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux || windows

package desktop

import "sync/atomic"

// mainThreadID is the id of the thread that [Init] locks the main goroutine
// to, on which the main loop runs, or 0 before Init.
var mainThreadID atomic.Int64

// setMainThread records the current thread as the main thread; it is
// called by [Init] after locking the main goroutine to it.
func setMainThread() {
	mainThreadID.Store(int64(currentThreadID()))
}

// onMainThread returns whether the calling goroutine runs on the main
// thread (see [setMainThread]), or true before [Init] (when it is not known).
// Other goroutines must not be taken to be on the main thread, as the
// initialization of the windowing system returns at once for calls on the
// main thread while it is in progress (see [App.initGUI]).
func onMainThread() bool {
	id := mainThreadID.Load()
	return id == 0 || int64(currentThreadID()) == id
}
