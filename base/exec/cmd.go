// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"bytes"
	"os/exec"
	"strings"
)

// Cmd is a type alias for [exec.Cmd].
type Cmd = exec.Cmd

// CmdIO maintains an exec.Cmd pointer and IO state saved for the command
type CmdIO struct {
	StdIOState

	Cmd *exec.Cmd
}

func NewCmdIO(c *Config) *CmdIO {
	cio := &CmdIO{}
	cio.StdIO = c.StdIO
	return cio
}

// RunIO runs the given command using the given
// configuration information and arguments,
// waiting for it to complete before returning.
// IO version uses specified stdio and sets the
// command in it as well.
func (c *Config) RunIO(cio *CmdIO, cmd string, args ...string) error {
	cm, _, err := c.exec(&cio.StdIO, false, cmd, args...)
	cio.Cmd = cm
	return err
}

// StartIO starts the given command using the given
// configuration information and arguments,
// just starting the command but not waiting for it to finish.
// Returns the exec.Cmd command which can be used to kill the
// command later, if necessary.  In general calling code should
// keep track of these commands and manage them appropriately.
func (c *Config) StartIO(cio *CmdIO, cmd string, args ...string) error {
	cm, _, err := c.exec(&cio.StdIO, true, cmd, args...)
	cio.Cmd = cm
	return err
}

// OutputIO runs the command and returns the text from stdout.
func (c *Config) OutputIO(cio *CmdIO, cmd string, args ...string) (string, error) {
	// need to use buf to capture output
	sio := cio.StdIO
	buf := &bytes.Buffer{}
	sio.Out = buf
	_, _, err := c.exec(&sio, false, cmd, args...)
	if cio.Out != nil {
		cio.Out.Write(buf.Bytes())
	}
	return strings.TrimSuffix(buf.String(), "\n"), err
}
