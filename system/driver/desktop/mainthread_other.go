// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !darwin && !linux && !windows

package desktop

// onMainThread returns true, as the main thread is not known on these
// platforms (see the versions for macOS, Linux, and Windows).
func onMainThread() bool {
	return true
}
