// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !darwin

package desktop

// onMainThread returns true: only macOS requires the windowing system
// to be initialized on the main thread, which is checked there.
func onMainThread() bool {
	return true
}
