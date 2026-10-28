// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pty runs a command in a pseudo-terminal (pty), so that it
// acts as it does in a terminal: for example, a shell is interactive.
// It uses github.com/creack/pty on Unix, and Windows ConPTY through
// github.com/charmbracelet/x/xpty.
package pty

import (
	"context"
	"os"
	"os/exec"
	"runtime"
)

// Pty is a command running in a pseudo-terminal. Reading from it
// returns the output of the command, and writing to it sends input to it.
type Pty struct {
	// Cmd is the command.
	Cmd *exec.Cmd

	// pty is the pseudo-terminal.
	pty pseudoTerminal
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

// Wait waits for the command to finish and returns its error.
func (p *Pty) Wait() error {
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
