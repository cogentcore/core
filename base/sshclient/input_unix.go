// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build unix

package sshclient

import (
	"errors"
	"io"
	"os"
	"unsafe"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// terminalInput returns the given standard input as a terminal,
// if it is one, or nil.
func terminalInput(in io.Reader) *os.File {
	f, ok := in.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) || int(f.Fd()) >= fdSetSize {
		return nil
	}
	return f
}

// fdSetSize is the number of file descriptors that a [unix.FdSet] holds,
// for select.
const fdSetSize = int(unsafe.Sizeof(unix.FdSet{})) * 8

// forwardInput sends the input of the given terminal to the command of the
// given session, which has a terminal on the host (see [Client.newSession]),
// with the terminal in raw mode, such that the terminal on the host
// handles it, as for the keys of interactive programs and Control+C,
// and tells it about changes of the size of the terminal. It must be
// called before the command starts, and the returned function must be
// called when it ends: it stops reading the input, and restores the
// mode of the terminal. The input is only read when it is available, so
// that none of it is read after the command ends, as the input that
// follows is for the next command.
func forwardInput(ses *ssh.Session, tty *os.File) (stop func(), err error) {
	w, err := ses.StdinPipe()
	if err != nil {
		return nil, err
	}
	fd := int(tty.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		cols, rows, _ := term.GetSize(fd)
		buf := make([]byte, 1024)
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
			// poll does not work with terminals on macOS, so select is used
			var fds unix.FdSet
			fds.Set(fd)
			tv := unix.Timeval{Usec: 100_000}
			n, err := unix.Select(fd+1, &fds, nil, nil, &tv)
			if errors.Is(err, unix.EINTR) || (err == nil && n == 0) {
				continue
			}
			if err != nil {
				return
			}
			n, err = unix.Read(fd, buf)
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
				continue
			}
			if err != nil || n == 0 {
				return
			}
			if _, err := w.Write(buf[:n]); err != nil {
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
