// Copyright 2018 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"sync"
	"sync/atomic"
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
	head atomic.Pointer[queueEvent]
	tail atomic.Pointer[queueEvent]
	len  atomic.Uint64
}

// Init initializes the queue.
func (q *Deque) Init() {
	head := &queueEvent{}
	q.head.Store(head)
	q.tail.Store(head)
}

type queueEvent struct {
	next atomic.Pointer[queueEvent]
	v    Event
}

var queueEventPool = sync.Pool{
	New: func() any { return &queueEvent{} },
}

// NextEvent returns the next event in the deque.
// It blocks until such an event has been sent.
func (q *Deque) NextEvent() Event {
	var first, last, firstnext *queueEvent
	for {
		first = q.head.Load()
		last = q.tail.Load()
		firstnext = first.next.Load()
		if first == q.head.Load() {
			if first == last {
				if firstnext == nil {
					return nil
				}

				q.tail.CompareAndSwap(last, firstnext)
			} else {
				v := firstnext.v
				if q.head.CompareAndSwap(first, firstnext) {
					q.len.Add(^uint64(0))
					queueEventPool.Put(first)
					return v
				}
			}
		}
	}
}

// Send adds an event to the end of the deque,
// replacing the last of the same type unless marked
// as Unique.
// They are returned by NextEvent in FIFO order.
func (q *Deque) Send(ev Event) {
	i := queueEventPool.Get().(*queueEvent)
	i.next.Store(nil)
	i.v = ev

	var last, lastnext *queueEvent
	for {
		last = q.tail.Load()
		lastnext = last.next.Load()
		if q.tail.Load() == last {
			if lastnext == nil {
				if last.next.CompareAndSwap(lastnext, i) {
					q.tail.CompareAndSwap(last, i)
					q.len.Add(1)
					return
				}
			} else {
				q.tail.CompareAndSwap(last, lastnext)
			}
		}
	}
}

// Len returns the length of the queue.
func (q *Deque) Len() uint64 {
	return q.len.Load()
}
