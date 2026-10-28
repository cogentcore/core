// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"io"
	"sync"
)

// inputQueue is a queue of input for the program, which is written to it
// in a separate goroutine, so that sending input never blocks, even when
// the program is not reading its input.
type inputQueue struct {
	mu      sync.Mutex
	cond    *sync.Cond
	queue   [][]byte
	stopped bool
}

// newInputQueue returns a new input queue that writes to the given writer
// until it is stopped.
func newInputQueue(w io.Writer) *inputQueue {
	iq := &inputQueue{}
	iq.cond = sync.NewCond(&iq.mu)
	go iq.write(w)
	return iq
}

// stop stops writing the input.
func (iq *inputQueue) stop() {
	iq.mu.Lock()
	iq.stopped = true
	iq.queue = nil
	iq.cond.Broadcast()
	iq.mu.Unlock()
}

// send adds the given input to the queue.
func (iq *inputQueue) send(b []byte) {
	iq.mu.Lock()
	defer iq.mu.Unlock()
	if iq.stopped {
		return
	}
	iq.queue = append(iq.queue, b)
	iq.cond.Signal()
}

// write writes the input in the queue to w, until stopped.
func (iq *inputQueue) write(w io.Writer) {
	for {
		iq.mu.Lock()
		for !iq.stopped && len(iq.queue) == 0 {
			iq.cond.Wait()
		}
		if iq.stopped {
			iq.mu.Unlock()
			return
		}
		b := iq.queue[0]
		iq.queue = iq.queue[1:]
		iq.mu.Unlock()
		if _, err := w.Write(b); err != nil {
			iq.stop()
			return
		}
	}
}
