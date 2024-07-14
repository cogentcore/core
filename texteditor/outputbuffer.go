// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"bufio"
	"bytes"
	"io"
	"slices"
	"sync"
	"time"
)

// OutputBufferMarkupFunc is a function that returns a marked-up version of a given line of
// output text by adding html tags. It is essential that it ONLY adds tags,
// and otherwise has the exact same visible bytes as the input
type OutputBufferMarkupFunc func(line []byte) []byte

// OutputBuffer is a [Buffer] that records the output from an [io.Reader] using
// [bufio.Scanner]. It is optimized to combine fast chunks of output into
// large blocks of updating. It also supports an arbitrary markup function
// that operates on each line of output bytes.
type OutputBuffer struct { //types:add -setters

	// the output that we are reading from, as an io.Reader
	Output io.Reader

	// the [Buffer] that we output to
	Buffer *Buffer

	// how much time to wait while batching output (default: 200ms)
	Batch time.Duration

	// optional markup function that adds html tags to given line of output -- essential that it ONLY adds tags, and otherwise has the exact same visible bytes as the input
	MarkupFunc OutputBufferMarkupFunc

	// current buffered output raw lines, which are not yet sent to the Buffer
	currentOutputLines [][]byte

	// current buffered output markup lines, which are not yet sent to the Buffer
	currentOutputMarkupLines [][]byte

	// mutex protecting updating of CurrentOutputLines and Buffer, and timer
	mu sync.Mutex

	// time when last output was sent to buffer
	lastOutput time.Time

	// time.AfterFunc that is started after new input is received and not immediately output -- ensures that it will get output if no further burst happens
	afterTimer *time.Timer
}

// MonitorOutput monitors the output and updates the [Buffer].
func (ob *OutputBuffer) MonitorOutput() {
	if ob.Batch == 0 {
		ob.Batch = 200 * time.Millisecond
	}
	outscan := bufio.NewScanner(ob.Output) // line at a time
	ob.currentOutputLines = make([][]byte, 0, 100)
	ob.currentOutputMarkupLines = make([][]byte, 0, 100)
	for outscan.Scan() {
		b := outscan.Bytes()
		bc := slices.Clone(b) // outscan bytes are temp
		bec := htmlEscapeBytes(bc)

		ob.mu.Lock()
		if ob.afterTimer != nil {
			ob.afterTimer.Stop()
			ob.afterTimer = nil
		}
		ob.currentOutputLines = append(ob.currentOutputLines, bc)
		mup := bec
		if ob.MarkupFunc != nil {
			mup = ob.MarkupFunc(bec)
		}
		ob.currentOutputMarkupLines = append(ob.currentOutputMarkupLines, mup)
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
	lfb := []byte("\n")
	if len(ob.currentOutputLines) == 0 {
		return
	}
	tlns := bytes.Join(ob.currentOutputLines, lfb)
	mlns := bytes.Join(ob.currentOutputMarkupLines, lfb)
	tlns = append(tlns, lfb...)
	mlns = append(mlns, lfb...)
	ob.Buffer.Undos.Off = true
	ob.Buffer.AppendTextMarkup(tlns, mlns, EditSignal)
	// ob.Buf.AppendText(mlns, EditSignal) // todo: trying to allow markup according to styles
	ob.Buffer.AutoScrollEditors()
	ob.currentOutputLines = make([][]byte, 0, 100)
	ob.currentOutputMarkupLines = make([][]byte, 0, 100)
}
