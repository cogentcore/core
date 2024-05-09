// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"bytes"
	"strings"
)

// Run runs given command, using config input / outputs.
// Must have already made a successful Connect.
func (cl *Client) Run(cmd string, args ...string) error {
	_, err := cl.Exec(cmd, args...)
	return err
}

// Start starts the given command with arguments.
func (cl *Client) Start(cmd string, args ...string) error {
	// todo: implement this!
	return nil
}

// Output runs the command and returns the text from stdout.
func (cl *Client) Output(cmd string, args ...string) (string, error) {
	oldStdout := cl.Stdout
	// need to use buf to capture output
	buf := &bytes.Buffer{}
	cl.Stdout = buf
	_, err := cl.Exec(cmd, args...)
	cl.Stdout = oldStdout
	if cl.Stdout != nil {
		cl.Stdout.Write(buf.Bytes())
	}
	return strings.TrimSuffix(buf.String(), "\n"), err
}
