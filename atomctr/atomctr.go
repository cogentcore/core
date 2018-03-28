// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package atomctr implements basic atomic int64 counter, used e.g., for
// update counter on Ki Node
package atomctr

// github.com/rcoreilly/goki/ki/atomctr

import (
	"sync/atomic"
)

// Ctr implements basic atomic int64 counter, used e.g., for update counter on Ki Node
type Ctr int64

// increment counter
func (a *Ctr) Add(inc int64) int64 {
	return atomic.AddInt64((*int64)(a), inc)
}

// decrement counter
func (a *Ctr) Sub(dec int64) int64 {
	return atomic.AddInt64((*int64)(a), -dec)
}

// inc = ++
func (a *Ctr) Inc() int64 {
	return atomic.AddInt64((*int64)(a), 1)
}

// dec = --
func (a *Ctr) Dec() int64 {
	return atomic.AddInt64((*int64)(a), -1)
}

// current value
func (a *Ctr) Value() int64 {
	return atomic.LoadInt64((*int64)(a))
}

// set current value
func (a *Ctr) Set(val int64) {
	atomic.StoreInt64((*int64)(a), val)
}

// swap new value in and return old value
func (a *Ctr) Swap(val int64) int64 {
	return atomic.SwapInt64((*int64)(a), val)
}
