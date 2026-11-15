// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !linux && !windows

package desktop

// setMainThread does nothing on platforms where the main thread is known
// (macOS) or not known (others) without recording it.
func setMainThread() {}
