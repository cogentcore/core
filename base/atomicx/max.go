// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package atomicx

import "sync/atomic"

// MaxInt32 performs an atomic Max operation: a = max(a, b)
func MaxInt32(a *int32, b int32) {
	old := atomic.LoadInt32(a)
	for old < b && !atomic.CompareAndSwapInt32(a, old, b) {
		old = atomic.LoadInt32(a)
	}
}
