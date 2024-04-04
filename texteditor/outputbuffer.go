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
// output text by adding html tags.  It is essential that it ONLY adds tags,
// and otherwise has the exact same visible bytes as the input
type OutputBufferMarkupFunc func(line []byte) []byte

// OutputBuffer is a [Buffer] that records the output from an io.Reader using
// bufio.Scanner -- optimized to combine fast chunks of output into
// large blocks of updating.  Also supports arbitrary markup function
// that operates on each line of output bytes.
type OutputBuffer struct {

	// the output that we are reading from, as an io.Reader
	Output io.Reader

	// the [Buffer] that we output to
	Buffer *Buffer

	// how much time to wait while batching output (default: 200ms)
	Batch time.Duration

	// optional markup function that adds html tags to given line of output -- essential that it ONLY adds tags, and otherwise has the exact same visible bytes as the input
	MarkupFun OutputBufferMarkupFunc

	// current buffered output raw lines, which are not yet sent to the Buffer
	CurrentOutputLines [][]byte

	// current buffered output markup lines, which are not yet sent to the Buffer
	CurrentOutputMarkupLines [][]byte

	// mutex protecting updating of CurrentOutputLines and Buffer, and timer
	Mu sync.Mutex

	// time when last output was sent to buffer
	LastOut time.Time

	// time.AfterFunc that is started after new input is received and not immediately output -- ensures that it will get output if no further burst happens
	AfterTimer *time.Timer
}

// Init sets the various params and prepares for running.
func (ob *OutputBuffer) Init(out io.Reader, buf *Buffer, batch time.Duration, markup OutputBufferMarkupFunc) {
	ob.Output = out
	ob.Buffer = buf
	ob.MarkupFun = markup
	if batch == 0 {
		ob.Batch = 200 * time.Millisecond
	} else {
		ob.Batch = batch
	}
}

// MonitorOutput monitors the output and updates the [Buffer].
func (ob *OutputBuffer) MonitorOutput() {
	outscan := bufio.NewScanner(ob.Output) // line at a time
	ob.CurrentOutputLines = make([][]byte, 0, 100)
	ob.CurrentOutputMarkupLines = make([][]byte, 0, 100)
	for outscan.Scan() {
		b := outscan.Bytes()
		bc := slices.Clone(b) // outscan bytes are temp
		bec := HTMLEscapeBytes(bc)

		ob.Mu.Lock()
		if ob.AfterTimer != nil {
			ob.AfterTimer.Stop()
			ob.AfterTimer = nil
		}
		ob.CurrentOutputLines = append(ob.CurrentOutputLines, bc)
		mup := bec
		if ob.MarkupFun != nil {
			mup = ob.MarkupFun(bec)
		}
		ob.CurrentOutputMarkupLines = append(ob.CurrentOutputMarkupLines, mup)
		lag := time.Since(ob.LastOut)
		if lag > ob.Batch {
			ob.LastOut = time.Now()
			ob.OutputToBuffer()
		} else {
			ob.AfterTimer = time.AfterFunc(ob.Batch*2, func() {
				ob.Mu.Lock()
				ob.LastOut = time.Now()
				ob.OutputToBuffer()
				ob.AfterTimer = nil
				ob.Mu.Unlock()
			})
		}
		ob.Mu.Unlock()
	}
	ob.OutputToBuffer()
}

// OutputToBuffer sends the current output to Buf.
// MUST be called under mutex protection
func (ob *OutputBuffer) OutputToBuffer() {
	lfb := []byte("\n")
	if len(ob.CurrentOutputLines) == 0 {
		return
	}
	tlns := bytes.Join(ob.CurrentOutputLines, lfb)
	mlns := bytes.Join(ob.CurrentOutputMarkupLines, lfb)
	tlns = append(tlns, lfb...)
	mlns = append(mlns, lfb...)
	ob.Buffer.Undos.Off = true
	ob.Buffer.AppendTextMarkup(tlns, mlns, EditSignal)
	// ob.Buf.AppendText(mlns, EditSignal) // todo: trying to allow markup according to styles
	ob.Buffer.AutoScrollViews()
	ob.CurrentOutputLines = make([][]byte, 0, 100)
	ob.CurrentOutputMarkupLines = make([][]byte, 0, 100)
}
