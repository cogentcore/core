// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package sshclient

import (
	"io"
	"os"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

// terminalInput returns the given standard input as a terminal (console),
// if it is one, or nil.
func terminalInput(in io.Reader) *os.File {
	f, ok := in.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return nil
	}
	return f
}

var procReadConsoleInputW = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReadConsoleInputW")

// inputRecord is the Windows INPUT_RECORD, with the KEY_EVENT_RECORD
// of a key event.
type inputRecord struct {
	eventType       uint16
	_               uint16
	keyDown         int32
	repeatCount     uint16
	virtualKeyCode  uint16
	virtualScanCode uint16
	char            uint16
	controlKeyState uint32
}

// readConsoleInput reads the available input records of the given console,
// without blocking if there are none.
func readConsoleInput(h windows.Handle, recs []inputRecord) (int, error) {
	var avail uint32
	if err := windows.GetNumberOfConsoleInputEvents(h, &avail); err != nil || avail == 0 {
		return 0, err
	}
	var n uint32
	r, _, err := procReadConsoleInputW.Call(uintptr(h), uintptr(unsafe.Pointer(&recs[0])), uintptr(min(int(avail), len(recs))), uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		return 0, err
	}
	return int(n), nil
}

// forwardInput sends the input of the given terminal (console) to the
// command of the given session, which has a terminal on the host (see
// [Client.newSession]), with the console in raw mode with virtual terminal
// input, such that the terminal on the host handles it, as for the keys of
// interactive programs and Control+C, and tells it about changes of the
// size of the console. It must be called before the command starts, and
// the returned function must be called when it ends: it stops reading the
// input, and restores the mode of the console. The input is only read when
// it is available, so that none of it is read after the command ends, as
// the input that follows is for the next command.
func forwardInput(ses *ssh.Session, tty *os.File) (stop func(), err error) {
	w, err := ses.StdinPipe()
	if err != nil {
		return nil, err
	}
	fd := int(tty.Fd())
	h := windows.Handle(fd)
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		cols, rows, _ := term.GetSize(fd)
		recs := make([]inputRecord, 128)
		var pending uint16 // first half of a surrogate pair
		var buf []byte
		for {
			select {
			case <-done:
				return
			default:
			}
			if c, r, err := term.GetSize(fd); err == nil && (c != cols || r != rows) {
				cols, rows = c, r
				ses.WindowChange(rows, cols)
			}
			ev, err := windows.WaitForSingleObject(h, 100)
			if err != nil {
				return
			}
			if ev != windows.WAIT_OBJECT_0 {
				continue
			}
			// the records are read directly, rather than the characters,
			// as reading characters blocks if there are only other events
			// (such as focus and key up), which also signal the console;
			// virtual terminal input gives the keys as characters
			n, err := readConsoleInput(h, recs)
			if err != nil {
				return
			}
			buf = buf[:0]
			for _, rec := range recs[:n] {
				if rec.eventType != windows.KEY_EVENT || rec.keyDown == 0 || rec.char == 0 {
					continue
				}
				var r rune
				switch {
				case utf16.IsSurrogate(rune(rec.char)) && rec.char < 0xdc00:
					pending = rec.char
					continue
				case utf16.IsSurrogate(rune(rec.char)):
					r = utf16.DecodeRune(rune(pending), rune(rec.char))
					pending = 0
				default:
					r = rune(rec.char)
				}
				for range max(rec.repeatCount, 1) {
					buf = utf8.AppendRune(buf, r)
				}
			}
			if len(buf) == 0 {
				continue
			}
			if _, err := w.Write(buf); err != nil {
				return
			}
		}
	}()
	return func() {
		close(done)
		<-ended
		term.Restore(fd, state)
	}, nil
}
