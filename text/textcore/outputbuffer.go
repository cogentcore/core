// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"bufio"
	"io"
	"sync"
	"time"

	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
)

// OutputBufferMarkupFunc is a function that returns a marked-up version
// of a given line of output text by adding html tags. It is essential
// that it ONLY adds tags, and otherwise has the exact same visible bytes
// as the input.
type OutputBufferMarkupFunc func(line []rune) rich.Text

// OutputBuffer is a [Buffer] that records the output from an [io.Reader] using
// [bufio.Scanner]. It is optimized to combine fast chunks of output into
// large blocks of updating. It also supports an arbitrary markup function
// that operates on each line of output bytes.
type OutputBuffer struct { //types:add -setters

	// the output that we are reading from, as an io.Reader
	Output io.Reader

	// the [Buffer] that we output to
	Buffer *lines.Lines

	// how much time to wait while batching output (default: 200ms)
	Batch time.Duration

	// optional markup function that adds html tags to given line of output.
	// It is essential that it ONLY adds tags, and otherwise has the exact
	// same visible bytes as the input.
	MarkupFunc OutputBufferMarkupFunc

	// current buffered output raw lines, which are not yet sent to the Buffer
	bufferedLines [][]rune

	// current buffered output markup lines, which are not yet sent to the Buffer
	bufferedMarkup []rich.Text

	// mutex protecting updating of CurrentOutputLines and Buffer, and timer
	mu sync.Mutex

	// time when last output was sent to buffer
	lastOutput time.Time

	// time.AfterFunc that is started after new input is received and not
	// immediately output. Ensures that it will get output if no further burst happens.
	afterTimer *time.Timer
}

// MonitorOutput monitors the output and updates the [Buffer].
func (ob *OutputBuffer) MonitorOutput() {
	if ob.Batch == 0 {
		ob.Batch = 200 * time.Millisecond
	}
	outscan := bufio.NewScanner(ob.Output) // line at a time
	ob.bufferedLines = make([][]rune, 0, 100)
	ob.bufferedMarkup = make([]rich.Text, 0, 100)
	for outscan.Scan() {
		b := outscan.Bytes()
		rln := []rune(string(b))

		ob.mu.Lock()
		if ob.afterTimer != nil {
			ob.afterTimer.Stop()
			ob.afterTimer = nil
		}
		ob.bufferedLines = append(ob.bufferedLines, rln)
		if ob.MarkupFunc != nil {
			mup := ob.MarkupFunc(rln)
			ob.bufferedMarkup = append(ob.bufferedMarkup, mup)
		} else {
			mup := rich.NewText(rich.NewStyle(), rln)
			ob.bufferedMarkup = append(ob.bufferedMarkup, mup)
		}
		lag := time.Since(ob.lastOutput)
		if lag > ob.Batch {
			ob.lastOutput = time.Now()
			ob.outputToBuffer()
		} else {
			ob.afterTimer = time.AfterFunc(ob.Batch*2, func() {
				ob.mu.Lock()
				ob.lastOutput = time.Now()
				ob.outputToBuffer()
				ob.afterTimer = nil
				ob.mu.Unlock()
			})
		}
		ob.mu.Unlock()
	}
	ob.outputToBuffer()
}

// outputToBuffer sends the current output to Buffer.
// MUST be called under mutex protection
func (ob *OutputBuffer) outputToBuffer() {
	if len(ob.bufferedLines) == 0 {
		return
	}
	ob.Buffer.SetUndoOn(false)
	ob.Buffer.AppendTextMarkup(ob.bufferedLines, ob.bufferedMarkup)
	// ob.Buffer.AutoScrollEditors() // todo
	ob.bufferedLines = make([][]rune, 0, 100)
	ob.bufferedMarkup = make([]rich.Text, 0, 100)
}
