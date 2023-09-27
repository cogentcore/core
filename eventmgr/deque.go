// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package eventmgr

import (
	"sync"

	"goki.dev/goosi"
	"goki.dev/goosi/mouse"
)

// Deque is an infinitely buffered double-ended queue of events.
// The zero value is usable, but a Deque value must not be copied.
type Deque struct {
	Back  []goosi.Event // FIFO.
	Front []goosi.Event // LIFO.

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
func (q *Deque) NextEvent() goosi.Event {
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

// PollEvent returns the next event in the deque if available,
// and returns true.
// If none are available, it returns false immediately.
func (q *Deque) PollEvent() (goosi.Event, bool) {
	q.LockAndInit()
	defer q.Mu.Unlock()

	if n := len(q.Front); n > 0 {
		e := q.Front[n-1]
		q.Front[n-1] = nil
		q.Front = q.Front[:n-1]
		return e, true
	}

	if n := len(q.Back); n > 0 {
		e := q.Back[0]
		q.Back[0] = nil
		q.Back = q.Back[1:]
		return e, true
	}
	return nil, false
}

// Send adds an event to the end of the deque,
// replacing the last of the same type unless marked
// as Unique.
// They are returned by NextEvent in FIFO order.
func (q *Deque) Send(ev goosi.Event) {
	q.LockAndInit()
	defer q.Mu.Unlock()

	n := len(q.Back)
	if !ev.IsUnique() && n > 0 {
		lev := q.Back[n-1]
		if ev.IsSame(lev) {
			q.Back[n-1] = ev // replace
			switch ev.Type() {
			case goosi.MouseMoveEvent, goosi.MouseDragEvent:
				me := ev.(*mouse.Event)
				le := lev.(*mouse.Event)
				me.Prev = le.Prev
				me.PrvTime = le.PrvTime
			case goosi.MouseScrollEvent:
				me := ev.(*mouse.ScrollEvent)
				le := lev.(*mouse.ScrollEvent)
				me.Delta = me.Delta.Add(le.Delta)
			}
			q.Cond.Signal()
			return
		}
	}
	q.Back = append(q.Back, ev)
	q.Cond.Signal()
}

// SendFirst adds an event to the start of the deque.
// They are returned by NextEvent in LIFO order,
// and have priority over events sent via Send.
func (q *Deque) SendFirst(ev goosi.Event) {
	q.LockAndInit()
	defer q.Mu.Unlock()

	q.Front = append(q.Front, ev)
	q.Cond.Signal()
}
