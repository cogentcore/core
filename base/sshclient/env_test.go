// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/sshclient/sshtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoginEnv tests that the commands get the environment of a login
// shell of the user on the host, as its PATH, even with output from its
// startup files.
func TestLoginEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	s := sshtest.NewServer(t)
	bin := filepath.Join(s.Root, "mybin")
	require.NoError(t, os.Mkdir(bin, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(bin, "mycmd"), []byte("#!/bin/sh\necho from mycmd\n"), 0755))
	profile := `echo hello from the profile
export PATH="$HOME/mybin:$PATH"
export QUOTED="it's a 'test'"
`
	require.NoError(t, os.WriteFile(filepath.Join(s.Root, ".profile"), []byte(profile), 0644))

	connect := func(loginEnv bool) *Client {
		// with the output of the client set, as goal does
		cl := NewClient(NewConfig(&exec.Config{StdIO: exec.StdIO{Out: io.Discard, Err: io.Discard}}))
		cl.User.KeyPath = s.KeyPath
		cl.LoginEnv = loginEnv
		require.NoError(t, cl.Connect(s.Addr))
		t.Cleanup(cl.Close)
		return cl
	}
	sio := &exec.StdIOState{StdIO: exec.StdIO{Out: io.Discard, Err: io.Discard}}
	cl := connect(true)
	out, err := cl.Output(sio, "mycmd")
	require.NoError(t, err)
	assert.Equal(t, "from mycmd", out)
	out, err = cl.Output(sio, "printenv", "QUOTED")
	require.NoError(t, err)
	assert.Equal(t, "it's a 'test'", out)

	cl = connect(false)
	_, err = cl.Output(sio, "mycmd")
	assert.Error(t, err, "without LoginEnv")
}
