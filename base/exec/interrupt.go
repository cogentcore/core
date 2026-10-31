// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows

package exec

import "os"

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
