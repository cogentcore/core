// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pty runs a command in a pseudo-terminal (pty), so that it
// acts as it does in a terminal: for example, a shell is interactive.
// It can also run a function of this process with a pseudo-terminal
// (see [Run]), such as an interpreter of the app.
// It uses github.com/creack/pty on Unix, and Windows ConPTY through
// github.com/charmbracelet/x/xpty.
package pty

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// ErrUnsupported is returned on platforms without pseudo-terminals,
// and by [Run] on Windows.
var ErrUnsupported = errors.New("pty: pseudo-terminals are not supported on this platform")

// Pty is a command or function running in a pseudo-terminal. Reading from it
// returns the output of the program, and writing to it sends input to it.
type Pty struct {
	// Cmd is the command, if it was started with [Start].
	Cmd *exec.Cmd

	// Tty is the terminal device of the pseudo-terminal, if it was
	// started with [Run]: the function uses it as its standard input
	// and output, and gives it to the commands it runs.
	Tty *os.File

	// pty is the pseudo-terminal.
	pty pseudoTerminal

	// done is closed when the function of [Run] returns, with its error in err.
	done chan struct{}
	err  error
}

// pseudoTerminal is the interface of the platform pseudo-terminal.
type pseudoTerminal interface {
	Read(b []byte) (int, error)
	Write(b []byte) (int, error)
	Close() error
	Resize(cols, rows int) error
}

// Start starts the given command in a new pseudo-terminal of the given size.
func Start(cmd *exec.Cmd, cols, rows int) (*Pty, error) {
	pt, err := start(cmd, cols, rows)
	if err != nil {
		return nil, err
	}
	return &Pty{Cmd: cmd, pty: pt}, nil
}

// Run runs the given function in a goroutine with a new pseudo-terminal
// of the given size, whose terminal device it gets: it uses that as its
// standard input and output, as a command started with [Start] does,
// and it can give it to the commands it runs. This is for programs that
// run in this process, such as an interpreter of the app. The terminal
// device is closed when the function returns, and the program ends
// (see [Pty.Wait]). Closing the pseudo-terminal makes the reads and
// writes of the terminal device fail, which should make the function
// return. Unlike a command, it gets no signals: see [Pty.Signals] and
// [Pty.Resize]. It is not supported on Windows.
func Run(fun func(tty *os.File) error, cols, rows int) (*Pty, error) {
	pt, tty, err := open(cols, rows)
	if err != nil {
		return nil, err
	}
	p := &Pty{Tty: tty, pty: pt, done: make(chan struct{})}
	go func() {
		p.err = fun(tty)
		tty.Close()
		close(p.done)
		// reading the output does not end when the terminal device is
		// closed on macOS, or while a read of it is still pending
		time.AfterFunc(200*time.Millisecond, func() { pt.Close() })
	}()
	return p, nil
}

// Read reads output from the command.
func (p *Pty) Read(b []byte) (int, error) {
	return p.pty.Read(b)
}

// Write writes input to the command.
func (p *Pty) Write(b []byte) (int, error) {
	return p.pty.Write(b)
}

// Resize sets the size of the terminal, which the command is told about.
func (p *Pty) Resize(cols, rows int) error {
	return p.pty.Resize(cols, rows)
}

// Close closes the pseudo-terminal, which ends the command if it is still running
// (a shell gets a hangup signal).
func (p *Pty) Close() error {
	return p.pty.Close()
}

// Signals returns whether the terminal makes signals of the
// interrupt, quit and suspend characters (ISIG), as it does unless the
// program reads raw input. A function started with [Run] gets no
// signals, so the caller has to tell it about these characters.
func (p *Pty) Signals() bool {
	return signals(p.pty)
}

// Wait waits for the command or function to finish and returns its error.
func (p *Pty) Wait() error {
	if p.Cmd == nil {
		<-p.done
		return p.err
	}
	return wait(context.Background(), p.Cmd)
}

// DefaultShell returns the default shell command for the user:
// $SHELL, or /bin/sh, and on Windows, powershell.exe if available,
// or else %ComSpec% (cmd.exe).
func DefaultShell() []string {
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("powershell.exe"); err == nil {
			return []string{p, "-NoLogo"}
		}
		if cs := os.Getenv("ComSpec"); cs != "" {
			return []string{cs}
		}
		return []string{"cmd.exe"}
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		return []string{sh}
	}
	return []string{"/bin/sh"}
}
