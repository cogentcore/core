// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Cmd is a type alias for [exec.Cmd].
type Cmd = exec.Cmd

// CmdIO maintains an [exec.Cmd] pointer and IO state saved for the command
type CmdIO struct {
	StdIOState

	// Cmd is the [exec.Cmd]
	Cmd *exec.Cmd
}

func (c *CmdIO) String() string {
	if c.Cmd == nil {
		return "<nil>"
	}
	str := ""
	if c.Cmd.ProcessState != nil {
		str = c.Cmd.ProcessState.String()
	} else if c.Cmd.Process != nil {
		str = fmt.Sprintf("%d  	", c.Cmd.Process.Pid)
	} else {
		str = "no process info"
	}
	str += c.Cmd.String()
	return str
}

// NewCmdIO returns a new [CmdIO] initialized with StdIO settings from given Config
func NewCmdIO(c *Config) *CmdIO {
	cio := &CmdIO{}
	cio.StdIO = c.StdIO
	return cio
}

// RunIO runs the given command using the given
// configuration information and arguments,
// waiting for it to complete before returning.
// This IO version of [Run] uses specified stdio and sets the
// command in it as well, for easier management of
// dynamically updated IO routing.
func (c *Config) RunIO(cio *CmdIO, cmd string, args ...string) error {
	cm, _, err := c.exec(&cio.StdIO, false, cmd, args...)
	cio.Cmd = cm
	return err
}

// StartIO starts the given command using the given
// configuration information and arguments,
// just starting the command but not waiting for it to finish.
// This IO version of [Start] uses specified stdio and sets the
// command in it as well, which should be used to Wait for the
// command to finish (in a separate goroutine).
// For dynamic IO routing uses, call [CmdIO.StackStart] prior to
// setting the IO values using Push commands, and then call
// [PopToStart] after Wait finishes, to close any open IO and reset.
func (c *Config) StartIO(cio *CmdIO, cmd string, args ...string) error {
	cm, _, err := c.exec(&cio.StdIO, true, cmd, args...)
	cio.Cmd = cm
	return err
}

// OutputIO runs the command and returns the text from stdout.
func (c *Config) OutputIO(cio *CmdIO, cmd string, args ...string) (string, error) {
	// need to use buf to capture output
	sio := cio.StdIO // copy
	buf := &bytes.Buffer{}
	sio.Out = buf
	_, _, err := c.exec(&sio, false, cmd, args...)
	if cio.Out != nil {
		cio.Out.Write(buf.Bytes())
	}
	return strings.TrimSuffix(buf.String(), "\n"), err
}
