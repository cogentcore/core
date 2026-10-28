// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vcs

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// repoBase has the location of a repository, and runs commands in it.
type repoBase struct {
	// remote is the location of the remote repository (a URL).
	remote string

	// local is the path of the local repository (working copy).
	local string
}

// Remote returns the location of the remote repository (a URL),
// or "" if there is none.
func (rb *repoBase) Remote() string {
	return rb.remote
}

// LocalPath returns the path of the local repository (working copy).
func (rb *repoBase) LocalPath() string {
	return rb.local
}

// CmdFromDir returns a command to run in the local repository directory.
func (rb *repoBase) CmdFromDir(cmd string, args ...string) *exec.Cmd {
	c := exec.Command(cmd, args...)
	c.Dir = rb.local
	c.Env = append(os.Environ(), "PWD="+rb.local)
	return c
}

// RunFromDir runs the given command in the local repository directory,
// returning its combined output.
func (rb *repoBase) RunFromDir(cmd string, args ...string) ([]byte, error) {
	return rb.CmdFromDir(cmd, args...).CombinedOutput()
}

// runErr runs the given command in the local repository directory,
// returning its output, and an error including the output if it fails.
func (rb *repoBase) runErr(cmd string, args ...string) (string, error) {
	out, err := rb.RunFromDir(cmd, args...)
	sout := strings.TrimSpace(string(out))
	if err != nil {
		return sout, fmt.Errorf("%s %s: %w: %s", cmd, args[0], err, sout)
	}
	return sout, nil
}
