// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import "cogentcore.org/core/events"

// FuncRun is a simple helper type that contains a function to call and a channel
// to send a signal on when the function is finished running.
type FuncRun struct {
	F    func()
	Done chan struct{}
}

// PendingKeys returns whether the given queue of events has key events
// that have not been handled yet, as for a virtual keyboard whose text is
// set (see [system.TextInput.Before]) only once the keys that it sent
// have been handled, as they change it.
func PendingKeys(q *events.Deque) bool {
	q.LockAndInit()
	defer q.Mu.Unlock()
	for _, evs := range [][]events.Event{q.Front, q.Back} {
		for _, e := range evs {
			if _, ok := e.(*events.Key); ok {
				return true
			}
		}
	}
	return false
}
