// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"io"
	"os"
	"testing"
	"time"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/sshclient/sshtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReconnect tests that commands fail instead of hanging when the
// network is lost, and that the connection is made again when the
// network comes back, keeping the directory of the client.
func TestReconnect(t *testing.T) {
	s := sshtest.NewServer(t)
	cl := NewClient(NewConfig(&exec.Config{}))
	cl.User.KeyPath = s.KeyPath
	cl.KeepAlive = 100 * time.Millisecond
	cl.Timeout = 300 * time.Millisecond
	require.NoError(t, cl.Connect(s.Addr))
	t.Cleanup(cl.Close)
	sio := &exec.StdIOState{StdIO: exec.StdIO{Out: io.Discard, Err: io.Discard}}
	output := func(cmd string, args ...string) (string, error) {
		return cl.Output(sio, cmd, args...)
	}

	require.NoError(t, os.Mkdir(s.Root+"/sub", 0755))
	_, err := output("cd", "sub")
	require.NoError(t, err)
	out, err := output("pwd")
	require.NoError(t, err)
	assert.Equal(t, s.Root+"/sub", out)

	// a command fails when the network is lost, instead of hanging
	s.NetworkDown()
	done := make(chan error, 1)
	go func() {
		_, err := output("echo", "lost")
		done <- err
	}()
	select {
	case err := <-done:
		assert.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the command hangs with the network lost")
	}

	// it fails fast while the network is lost
	start := time.Now()
	assert.Error(t, cl.CheckConnection())
	assert.Error(t, cl.CheckConnection())
	assert.Less(t, time.Since(start), 2*time.Second)

	// and connects again when the network comes back
	s.NetworkUp()
	deadline := time.Now().Add(5 * time.Second)
	for cl.CheckConnection() != nil {
		if time.Now().After(deadline) {
			t.Fatal("not connected again")
		}
		time.Sleep(50 * time.Millisecond)
	}
	out, err = output("pwd")
	require.NoError(t, err)
	assert.Equal(t, s.Root+"/sub", out, "the same directory")

	// files can be copied again
	f, err := cl.OpenFile("file.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	require.NoError(t, err)
	_, err = f.Write([]byte("again"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	b, err := os.ReadFile(s.Root + "/sub/file.txt")
	require.NoError(t, err)
	assert.Equal(t, "again", string(b))

	// the connection stays up with keepalives
	time.Sleep(500 * time.Millisecond)
	assert.NoError(t, cl.CheckConnection())

	// and it is not made again after Close
	cl.Close()
	assert.Error(t, cl.CheckConnection())
}
