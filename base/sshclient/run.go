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
	_, err := cl.exec(cmd, args...)
	return err
}

// Start starts the given command with arguments.
func (cl *Client) Start(cmd string, args ...string) error {
	// todo: implement this!
	return nil
}

// Output runs the command and returns the text from stdout.
func (cl *Client) Output(cmd string, args ...string) (string, error) {
	buf := &bytes.Buffer{}
	cl.StdIO.PushOut(buf)
	err := cl.Run(cmd, args...)
	cl.StdIO.PopOut()
	if cl.StdIO.Out != nil {
		cl.StdIO.Out.Write(buf.Bytes())
	}
	return strings.TrimSuffix(buf.String(), "\n"), err
}
