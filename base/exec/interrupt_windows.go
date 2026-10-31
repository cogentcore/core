// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package exec

import (
	"os"

	"golang.org/x/sys/windows"
)

// interrupt interrupts the given process of a command whose context is
// done. Windows has no interrupt signal for a process, but a command in
// the console of this process gets the Control+C typed in it itself (as
// this process does, which is what typically makes the context done), so
// it is given [InterruptWait] to end, after which it is killed. Otherwise,
// it is killed now.
func interrupt(p *os.Process) error {
	var mode uint32
	if windows.GetConsoleMode(windows.Stdin, &mode) == nil {
		return nil
	}
	return p.Kill()
}
