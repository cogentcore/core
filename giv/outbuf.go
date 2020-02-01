// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bufio"
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/goki/ki/sliceclone"
)

// OutBufMarkupFunc is a function that returns a marked-up version of a given line of
// output text by adding html tags.  It is essential that it ONLY adds tags,
// and otherwise has the exact same visible bytes as the input
type OutBufMarkupFunc func(line []byte) []byte

// OutBuf is a TextBuf that records the output from an io.Reader using
// bufio.Scanner -- optimized to combine fast chunks of output into
// large blocks of updating.  Also supports arbitrary markup function
// that operates on each line of output bytes.
type OutBuf struct {
	Out        io.Reader        `desc:"the output that we are reading from, as an io.Reader"`
	Buf        *TextBuf         `desc:"the TextBuf that we output to"`
	BatchMSec  int              `desc:"default 200: how many milliseconds to wait while batching output"`
	MarkupFun  OutBufMarkupFunc `desc:"optional markup function that adds html tags to given line of output -- essential that it ONLY adds tags, and otherwise has the exact same visible bytes as the input"`
	CurOutLns  [][]byte         `desc:"current buffered output raw lines -- not yet sent to Buf"`
	CurOutMus  [][]byte         `desc:"current buffered output markup lines -- not yet sent to Buf"`
	Mu         sync.Mutex       `desc:"mutex protecting updating of CurOutLns and Buf, and timer"`
	LastOut    time.Time        `desc:"time when last output was sent to buffer"`
	AfterTimer *time.Timer      `desc:"time.AfterFunc that is started after new input is received and not immediately output -- ensures that it will get output if no further burst happens"`
}

// Init sets the various params and prepares for running
func (ob *OutBuf) Init(out io.Reader, buf *TextBuf, batchMSec int, markup OutBufMarkupFunc) {
	ob.Out = out
	ob.Buf = buf
	ob.MarkupFun = markup
	if batchMSec == 0 {
		ob.BatchMSec = 200
	} else {
		ob.BatchMSec = batchMSec
	}
}

// MonOut monitors the output and updates the TextBuf
func (ob *OutBuf) MonOut() {
	outscan := bufio.NewScanner(ob.Out) // line at a time
	ob.CurOutLns = make([][]byte, 0, 100)
	ob.CurOutMus = make([][]byte, 0, 100)
	for outscan.Scan() {
		b := outscan.Bytes()
		bc := sliceclone.Byte(b) // outscan bytes are temp
		bec := HTMLEscapeBytes(bc)

		ob.Mu.Lock()
		if ob.AfterTimer != nil {
			ob.AfterTimer.Stop()
			ob.AfterTimer = nil
		}
		ob.CurOutLns = append(ob.CurOutLns, bc)
		mup := bec
		if ob.MarkupFun != nil {
			mup = ob.MarkupFun(bec)
		}
		ob.CurOutMus = append(ob.CurOutMus, mup)
		now := time.Now()
		lag := int(now.Sub(ob.LastOut) / time.Millisecond)
		if lag > ob.BatchMSec {
			ob.LastOut = now
			ob.OutToBuf()
		} else {
			ob.AfterTimer = time.AfterFunc(time.Duration(ob.BatchMSec*2)*time.Millisecond, func() {
				ob.Mu.Lock()
				ob.LastOut = time.Now()
				ob.OutToBuf()
				ob.AfterTimer = nil
				ob.Mu.Unlock()
			})
		}
		ob.Mu.Unlock()
	}
	ob.OutToBuf()
}

// OutToBuf sends the current output to TextBuf
// MUST be called under mutex protection
func (ob *OutBuf) OutToBuf() {
	lfb := []byte("\n")
	if len(ob.CurOutLns) == 0 {
		return
	}
	tlns := bytes.Join(ob.CurOutLns, lfb)
	mlns := bytes.Join(ob.CurOutMus, lfb)
	tlns = append(tlns, lfb...)
	mlns = append(mlns, lfb...)
	ob.Buf.Undos.Off = true
	ob.Buf.AppendTextMarkup(tlns, mlns, EditSignal)
	ob.Buf.AutoScrollViews()
	ob.CurOutLns = make([][]byte, 0, 100)
	ob.CurOutMus = make([][]byte, 0, 100)
}
