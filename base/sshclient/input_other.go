// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !unix && !windows

package sshclient

import (
	"io"
	"os"

	"golang.org/x/crypto/ssh"
)

// terminalInput returns nil: the input of a terminal is not sent to
// the commands on the host on this platform.
func terminalInput(in io.Reader) *os.File {
	return nil
}

// forwardInput is not supported on this platform.
func forwardInput(ses *ssh.Session, tty *os.File) (stop func(), err error) {
	return func() {}, nil
}
