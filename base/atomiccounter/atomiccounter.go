// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package atomiccounter implements a basic atomic int64 counter.
package atomiccounter

import (
	"sync/atomic"
)

// Counter implements a basic atomic int64 counter.
type Counter int64

// Add to counter.
func (a *Counter) Add(inc int64) int64 {
	return atomic.AddInt64((*int64)(a), inc)
}

// Subtract from counter.
func (a *Counter) Sub(dec int64) int64 {
	return atomic.AddInt64((*int64)(a), -dec)
}

// Increment by 1.
func (a *Counter) Inc() int64 {
	return atomic.AddInt64((*int64)(a), 1)
}

// Decrement by 1.
func (a *Counter) Dec() int64 {
	return atomic.AddInt64((*int64)(a), -1)
}

// Return the current value.
func (a *Counter) Value() int64 {
	return atomic.LoadInt64((*int64)(a))
}

// Set the counter to a new value.
func (a *Counter) Set(val int64) {
	atomic.StoreInt64((*int64)(a), val)
}

// Swap new value in and return old value.
func (a *Counter) Swap(val int64) int64 {
	return atomic.SwapInt64((*int64)(a), val)
}
