// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted in part from: https://github.com/magefile/mage
// Copyright presumably by Nate Finch, primary contributor
// Apache License, Version 2.0, January 2004

package exec

import (
	"bytes"
	"strings"
)

// Run runs the given command using the given configuration information and arguments.
func (c *Config) Run(cmd string, args ...string) error {
	_, err := c.Exec(cmd, args...)
	return err
}

// Output runs the command and returns the text from stdout.
func (c *Config) Output(cmd string, args ...string) (string, error) {
	oldStdout := c.Stdout
	// need to use buf to capture output
	buf := &bytes.Buffer{}
	c.Stdout = buf
	_, err := c.Exec(cmd, args...)
	c.Stdout = oldStdout
	if c.Stdout != nil {
		c.Stdout.Write(buf.Bytes())
	}
	return strings.TrimSuffix(buf.String(), "\n"), err
}

// Run calls [Config.Run] on [Major]
func Run(cmd string, args ...string) error {
	return Major().Run(cmd, args...)
}

// Output calls [Config.Output] on [Major]
func Output(cmd string, args ...string) (string, error) {
	return Major().Output(cmd, args...)
}
