// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package exec

import (
	"io"
	"os"

	"golang.org/x/sys/windows"
)

// interruptible returns whether a command that is run with a context
// is interrupted when the context is done (see [interrupt]). On Windows,
// a command in the console of this process gets the Control+C typed in
// it itself, as this process does (which is what typically makes the
// context done), and handles it as it chooses, as in cmd.exe: for
// example, a program that reads raw input, such as an editor, ignores it.
// Such a command is not interrupted: only one without a console is.
func interruptible() bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Stdin, &mode) != nil
}

// interrupt kills the given process of a command whose context is done,
// as Windows has no interrupt signal for a process.
func interrupt(p *os.Process) error {
	return p.Kill()
}

// saveConsole saves the modes of the console of the given standard input
// and output, if they are a console, returning a function that restores
// them. A command run in the console changes them as it needs, and does
// not restore them if it ends abnormally, such as when it is killed.
func saveConsole(in io.Reader, out io.Writer) func() {
	type saved struct {
		h    windows.Handle
		mode uint32
	}
	var modes []saved
	for _, f := range []any{in, out} {
		if f, ok := f.(*os.File); ok {
			h := windows.Handle(f.Fd())
			var mode uint32
			if windows.GetConsoleMode(h, &mode) == nil {
				modes = append(modes, saved{h, mode})
			}
		}
	}
	return func() {
		for _, s := range modes {
			windows.SetConsoleMode(s.h, s.mode)
		}
	}
}
