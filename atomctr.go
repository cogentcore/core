// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"sync/atomic"
)

// implements basic atomic int64 counter, used for update counter on Ki Node

type AtomCtr int64

// increment counter
func (a *AtomCtr) Add(inc int64) int64 {
	return atomic.AddInt64((*int64)(a), inc)
}

// decrement counter
func (a *AtomCtr) Sub(dec int64) int64 {
	return atomic.AddInt64((*int64)(a), -dec)
}

// inc = ++
func (a *AtomCtr) Inc() int64 {
	return atomic.AddInt64((*int64)(a), 1)
}

// dec = --
func (a *AtomCtr) Dec() int64 {
	return atomic.AddInt64((*int64)(a), -1)
}

// current value
func (a *AtomCtr) Value() int64 {
	return atomic.LoadInt64((*int64)(a))
}

// set current value
func (a *AtomCtr) Set(val int64) {
	atomic.StoreInt64((*int64)(a), val)
}

// swap new value and return old
func (a *AtomCtr) Swap(val int64) int64 {
	return atomic.SwapInt64((*int64)(a), val)
}
