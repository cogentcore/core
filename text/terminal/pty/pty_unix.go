// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows && !js

package pty

import (
	"context"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

// unixPty is the master side of a Unix pty.
type unixPty struct {
	*os.File
}

// start starts the command in a new session with the pty as its
// controlling terminal, as needed for job control in shells.
func start(cmd *exec.Cmd, cols, rows int) (pseudoTerminal, error) {
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return nil, err
	}
	return &unixPty{f}, nil
}

// open opens a new pty, returning its master side and terminal device.
func open(cols, rows int) (pseudoTerminal, *os.File, error) {
	f, tty, err := pty.Open()
	if err != nil {
		return nil, nil, err
	}
	p := &unixPty{f}
	if err := p.Resize(cols, rows); err != nil {
		f.Close()
		tty.Close()
		return nil, nil, err
	}
	return p, tty, nil
}

func signals(pt pseudoTerminal) bool {
	p, ok := pt.(*unixPty)
	if !ok {
		return false
	}
	t, err := unix.IoctlGetTermios(int(p.Fd()), ioctlGetTermios)
	return err == nil && t.Lflag&unix.ISIG != 0
}

func (p *unixPty) Resize(cols, rows int) error {
	return pty.Setsize(p.File, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func wait(ctx context.Context, cmd *exec.Cmd) error {
	return cmd.Wait()
}
