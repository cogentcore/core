// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted in part from: https://github.com/magefile/mage
// Copyright presumably by Nate Finch, primary contributor
// Apache License, Version 2.0, January 2004

package exec

import (
	"bytes"
	"os/exec"
	"strings"
)

// Run runs the given command using the given
// configuration information and arguments,
// waiting for it to complete before returning.
func (c *Config) Run(cmd string, args ...string) error {
	_, _, err := c.exec(&c.StdIO, false, cmd, args...)
	return err
}

// Start starts the given command using the given
// configuration information and arguments,
// just starting the command but not waiting for it to finish.
// Returns the exec.Cmd command which can be used to kill the
// command later, if necessary.  In general calling code should
// keep track of these commands and manage them appropriately.
func (c *Config) Start(cmd string, args ...string) (*exec.Cmd, error) {
	excmd, _, err := c.exec(&c.StdIO, true, cmd, args...)
	return excmd, err
}

// Output runs the command and returns the text from stdout.
func (c *Config) Output(cmd string, args ...string) (string, error) {
	// need to use buf to capture output
	buf := &bytes.Buffer{}
	sio := c.StdIO
	sio.Out = buf
	_, _, err := c.exec(&sio, false, cmd, args...)
	if c.StdIO.Out != nil {
		c.StdIO.Out.Write(buf.Bytes())
	}
	return strings.TrimSuffix(buf.String(), "\n"), err
}

// Run calls [Config.Run] on [Major]
func Run(cmd string, args ...string) error {
	return Major().Run(cmd, args...)
}

// Start calls [Config.Start] on [Major]
func Start(cmd string, args ...string) (*exec.Cmd, error) {
	return Major().Start(cmd, args...)
}

// Output calls [Config.Output] on [Major]
func Output(cmd string, args ...string) (string, error) {
	return Major().Output(cmd, args...)
}
