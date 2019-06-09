// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package event provides an infinitely buffered double-ended queue of events.
package event

import (
	"sync"

	"github.com/goki/gi/oswin"
)

// Deque is an infinitely buffered double-ended queue of events. The zero value
// is usable, but a Deque value must not be copied.
type Deque struct {
	mu    sync.Mutex
	cond  sync.Cond     // cond.L is lazily initialized to &Deque.mu.
	back  []oswin.Event // FIFO.
	front []oswin.Event // LIFO.
}

func (q *Deque) lockAndInit() {
	q.mu.Lock()
	if q.cond.L == nil {
		q.cond.L = &q.mu
	}
}

// NextEvent implements the oswin.EventDeque interface.
func (q *Deque) NextEvent() oswin.Event {
	q.lockAndInit()
	defer q.mu.Unlock()

	for {
		if n := len(q.front); n > 0 {
			e := q.front[n-1]
			q.front[n-1] = nil
			q.front = q.front[:n-1]
			return e
		}

		if n := len(q.back); n > 0 {
			e := q.back[0]
			q.back[0] = nil
			q.back = q.back[1:]
			return e
		}

		q.cond.Wait()
	}
}

// PollEvent returns the next event in the deque if available, returns true
// returns false and does not wait if no events currently available
func (q *Deque) PollEvent() (oswin.Event, bool) {
	q.lockAndInit()
	defer q.mu.Unlock()

	if n := len(q.front); n > 0 {
		e := q.front[n-1]
		q.front[n-1] = nil
		q.front = q.front[:n-1]
		return e, true
	}

	if n := len(q.back); n > 0 {
		e := q.back[0]
		q.back[0] = nil
		q.back = q.back[1:]
		return e, true
	}
	return nil, false
}

// Send implements the oswin.EventDeque interface.
func (q *Deque) Send(event oswin.Event) {
	q.lockAndInit()
	defer q.mu.Unlock()

	q.back = append(q.back, event)
	q.cond.Signal()
}

// SendFirst implements the oswin.EventDeque interface.
func (q *Deque) SendFirst(event oswin.Event) {
	q.lockAndInit()
	defer q.mu.Unlock()

	q.front = append(q.front, event)
	q.cond.Signal()
}
