// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"cogentcore.org/core/base/exec"
)

// Run runs given command, using config input / outputs.
// Must have already made a successful Connect.
func (cl *Client) Run(sio *exec.StdIOState, cmd string, args ...string) error {
	_, err := cl.Exec(sio, false, false, cmd, args...)
	return err
}

// Start starts the given command with arguments.
func (cl *Client) Start(sio *exec.StdIOState, cmd string, args ...string) error {
	_, err := cl.Exec(sio, true, false, cmd, args...)
	return err
}

// Output runs the command and returns the text from stdout.
func (cl *Client) Output(sio *exec.StdIOState, cmd string, args ...string) (string, error) {
	return cl.Exec(sio, false, true, cmd, args...)
}
