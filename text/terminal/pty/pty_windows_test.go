// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pty

import (
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

// TestRunConsole tests the console of a function run with [Run]: there
// is only one, the commands in it get Control+C, which this process
// ignores, and reading it fails when the pseudo-terminal is closed.
func TestRunConsole(t *testing.T) {
	ran := make(chan time.Duration, 1)
	sizes := make(chan [2]int, 2)
	p, err := Run(func(in, out *os.File) error {
		w, h, _ := term.GetSize(int(out.Fd()))
		sizes <- [2]int{w, h}
		cmd := exec.Command(os.Getenv("ComSpec"), "/c", "ping -n 30 127.0.0.1 >nul")
		cmd.Stdin, cmd.Stdout, cmd.Stderr = in, out, out
		start := time.Now()
		cmd.Run()
		ran <- time.Since(start)
		w, h, _ = term.GetSize(int(out.Fd()))
		sizes <- [2]int{w, h}
		_, err := in.Read(make([]byte, 10))
		return err
	}, 80, 24)
	require.NoError(t, err)
	go io.Copy(io.Discard, p)
	assert.Equal(t, [2]int{80, 24}, <-sizes)

	_, err = Run(func(in, out *os.File) error { return nil }, 80, 24)
	assert.Error(t, err)

	time.Sleep(500 * time.Millisecond) // the command is running
	assert.True(t, p.Signals())
	require.NoError(t, p.Resize(100, 30))
	_, err = p.Write([]byte{0x03})
	require.NoError(t, err)
	select {
	case d := <-ran:
		assert.Less(t, d, 5*time.Second)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for the command to be interrupted")
	}
	assert.Equal(t, [2]int{100, 30}, <-sizes)

	time.Sleep(200 * time.Millisecond) // reading the input
	require.NoError(t, p.Close())
	done := make(chan error, 1)
	go func() { done <- p.Wait() }()
	select {
	case err := <-done:
		assert.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for the function to end after closing")
	}

	p, err = Run(func(in, out *os.File) error { return nil }, 80, 24) // after the other one
	require.NoError(t, err)
	go io.Copy(io.Discard, p)
	assert.NoError(t, p.Wait())
}
