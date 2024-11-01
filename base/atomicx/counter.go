// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package atomicx implements misc atomic functions.
package atomicx

import (
	"sync/atomic"
)

// Counter implements a basic atomic int64 counter.
type Counter int64

// Add adds to counter.
func (a *Counter) Add(inc int64) int64 {
	return atomic.AddInt64((*int64)(a), inc)
}

// Sub subtracts from counter.
func (a *Counter) Sub(dec int64) int64 {
	return atomic.AddInt64((*int64)(a), -dec)
}

// Inc increments by 1.
func (a *Counter) Inc() int64 {
	return atomic.AddInt64((*int64)(a), 1)
}

// Dec decrements by 1.
func (a *Counter) Dec() int64 {
	return atomic.AddInt64((*int64)(a), -1)
}

// Value returns the current value.
func (a *Counter) Value() int64 {
	return atomic.LoadInt64((*int64)(a))
}

// Set sets the counter to a new value.
func (a *Counter) Set(val int64) {
	atomic.StoreInt64((*int64)(a), val)
}

// Swap swaps a new value in and returns the old value.
func (a *Counter) Swap(val int64) int64 {
	return atomic.SwapInt64((*int64)(a), val)
}
