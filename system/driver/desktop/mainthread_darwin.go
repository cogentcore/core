// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package desktop

/*
#include <pthread.h>
*/
import "C"

// onMainThread returns whether this is the main thread, on which macOS
// requires the windowing system to be used.
func onMainThread() bool {
	return C.pthread_main_np() != 0
}
