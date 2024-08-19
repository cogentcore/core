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

// TODO: event compression

// TraceEventCompression can be set to true to see when events
// are being compressed to eliminate laggy behavior.
var TraceEventCompression = false

// Queue is a lock-free FIFO freelist-based event queue.
// It must be initialized using [Queue.Init] before use.
// It is based on https://github.com/fyne-io/fyne/blob/master/internal/async/queue_canvasobject.go
type Queue struct {
	head atomic.Pointer[queueEvent]
	tail atomic.Pointer[queueEvent]
	len  atomic.Uint64
}

// Init initializes the queue.
func (q *Queue) Init() {
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

// NextEvent removes and returns the next event in the queue.
// It returns nil if the queue is empty.
func (q *Queue) NextEvent() Event {
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

// Send adds an event to the end of the queue.
func (q *Queue) Send(ev Event) {
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
func (q *Queue) Len() uint64 {
	return q.len.Load()
}
