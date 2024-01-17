// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/hack-pad/hackpad
// Licensed under the Apache 2.0 License

//go:build js

package jsfs

import (
	"bytes"
	"io"
	"os"
	"sync"
	"syscall/js"
	"time"

	"github.com/hack-pad/hackpadfs"
)

var (
	Stdout hackpadfs.File = &BufferedLogger{Nm: "dev/stdout", PrintFn: func(args ...any) {
		js.Global().Get("console").Call("log", args...)
	}}
	Stderr hackpadfs.File = &BufferedLogger{Nm: "dev/stderr", PrintFn: func(args ...any) {
		js.Global().Get("console").Call("error", args...)
	}}
)

type BufferedLogger struct {
	Nm        string
	PrintFn   func(args ...any)
	Mu        sync.Mutex
	Buf       bytes.Buffer
	TimerOnce sync.Once
}

func (b *BufferedLogger) Flush() {
	if b.Buf.Len() == 0 {
		return
	}

	const maxBufLen = 4096

	b.Mu.Lock()
	i := bytes.LastIndexByte(b.Buf.Bytes(), '\n')
	var buf []byte
	if i == -1 || b.Buf.Len() > maxBufLen {
		buf = b.Buf.Bytes()
		b.Buf.Reset()
	} else {
		buf = make([]byte, i)
		n, _ := b.Buf.Read(buf) // at time of writing, only io.EOF can be returned -- which we don't need
		buf = buf[:n]
	}
	b.Mu.Unlock()
	if len(buf) != 0 {
		b.PrintFn(string(buf))
	}
}

func (b *BufferedLogger) Print(s string) int {
	n, _ := b.Write([]byte(s))
	return n
}

func (b *BufferedLogger) Write(p []byte) (n int, err error) {
	b.TimerOnce.Do(func() {
		const waitTime = time.Second / 2
		go func() {
			ticker := time.NewTicker(waitTime)
			for range ticker.C {
				b.Flush()
			}
		}()
	})

	b.Mu.Lock()
	_, _ = b.Buf.Write(p) // at time of writing, bytes.Buffer.Write cannot return an error
	b.Mu.Unlock()
	return len(p), nil
}

func (b *BufferedLogger) Name() string {
	return b.Nm
}

func (b *BufferedLogger) Close() error {
	// TODO: prevent writes and return os.ErrClosed
	return nil
}

func (b *BufferedLogger) Read(p []byte) (n int, err error) { return 0, io.EOF }

func (b *BufferedLogger) Stat() (os.FileInfo, error) { return NullStat{b.Nm}, nil }
