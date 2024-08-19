// Copyright 2018 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
	"sync"
)

// TraceEventCompression can be set to true to see when events
// are being compressed to eliminate laggy behavior.
var TraceEventCompression = false

// Deque is an infinitely buffered double-ended queue of events.
// If an event is not marked as Unique, and the last
// event in the queue is of the same type, then the new one
// replaces the last one.  This automatically implements
// event compression to manage the common situation where
// event processing is slower than event generation,
// such as with Mouse movement and Paint events.
// The zero value is usable, but a Deque value must not be copied.
type Deque struct {
	Back  []Event // FIFO.
	Front []Event // LIFO.

	Mu   sync.Mutex
	Cond sync.Cond // Cond.L is lazily initialized to &Deque.Mu.
}

func (q *Deque) LockAndInit() {
	q.Mu.Lock()
	if q.Cond.L == nil {
		q.Cond.L = &q.Mu
	}
}

// NextEvent returns the next event in the deque.
// It blocks until such an event has been sent.
func (q *Deque) NextEvent() Event {
	q.LockAndInit()
	defer q.Mu.Unlock()

	for {
		if n := len(q.Front); n > 0 {
			e := q.Front[n-1]
			q.Front[n-1] = nil
			q.Front = q.Front[:n-1]
			return e
		}

		if n := len(q.Back); n > 0 {
			e := q.Back[0]
			q.Back[0] = nil
			q.Back = q.Back[1:]
			return e
		}

		q.Cond.Wait()
	}
}

// Send adds an event to the end of the deque,
// replacing the last of the same type unless marked
// as Unique.
// They are returned by NextEvent in FIFO order.
func (q *Deque) Send(ev Event) {
	q.LockAndInit()
	defer q.Mu.Unlock()

	n := len(q.Back)
	if !ev.IsUnique() && n > 0 {
		lev := q.Back[n-1]
		if ev.IsSame(lev) {
			q.Back[n-1] = ev // replace
			switch ev.Type() {
			case MouseMove, MouseDrag:
				me := ev.(*Mouse)
				le := lev.(*Mouse)
				me.Prev = le.Prev
				me.PrvTime = le.PrvTime
			case Scroll:
				me := ev.(*MouseScroll)
				le := lev.(*MouseScroll)
				me.Delta = me.Delta.Add(le.Delta)
			}
			q.Cond.Signal()
			if TraceEventCompression {
				fmt.Println("compressed back:", ev)
			}
			return
		}
	}
	q.Back = append(q.Back, ev)
	q.Cond.Signal()
}
