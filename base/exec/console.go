// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows

package exec

import (
	"io"
	"os"
)

// interruptible returns whether a command that is run with a context
// is interrupted when the context is done (see [interrupt]).
func interruptible() bool {
	return true
}

// interrupt interrupts the given process of a command whose context is
// done, with an interrupt signal, as a shell does for Control+C. It is
// killed if that fails, and otherwise if it has not ended after
// [InterruptWait].
func interrupt(p *os.Process) error {
	if err := p.Signal(os.Interrupt); err != nil {
		return p.Kill()
	}
	return nil
}

// saveConsole returns a function that restores the modes of the console
// of the given standard input and output, which is only needed on Windows.
func saveConsole(in io.Reader, out io.Writer) func() {
	return func() {}
}
