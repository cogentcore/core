// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"bytes"
	"context"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/text/terminal/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunInterrupt tests that a command is ended when its context is
// done, after [exec.InterruptWait] if the interrupt does not end it.
func TestRunInterrupt(t *testing.T) {
	cl, _ := testClient(t)
	sio := &exec.StdIOState{StdIO: exec.StdIO{Out: io.Discard, Err: io.Discard}}
	defer func(w time.Duration) { exec.InterruptWait = w }(exec.InterruptWait)
	exec.InterruptWait = 300 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	start := time.Now()
	err := cl.RunContext(ctx, sio, "sleep", "10")
	took := time.Since(start)
	assert.Error(t, err)
	assert.Less(t, took, 2*time.Second)

	// the client still works
	out, err := cl.OutputContext(context.Background(), sio, "echo", "ok")
	require.NoError(t, err)
	assert.Equal(t, "ok", strings.TrimSpace(out))
}

// TestRunTerminalInput tests that a command run with its input from a
// terminal gets the input of the terminal while it runs, and only then,
// as for interactive programs, with Control+C interrupting it on the host.
func TestRunTerminalInput(t *testing.T) {
	cl, s := testClient(t)
	s.Pty.Store(true)
	result := make(chan string, 1)
	readLine := func(in *os.File) {
		buf := make([]byte, 100)
		n, _ := in.Read(buf)
		result <- strings.ReplaceAll(string(buf[:n]), "\r", "")
	}
	p, err := pty.Run(func(in, out *os.File) error {
		sio := &exec.StdIOState{StdIO: exec.StdIO{In: in, Out: out, Err: out}}
		if err := cl.Run(sio, "head", "-1"); err != nil {
			return err
		}
		readLine(in) // the input that follows is for this process
		err := cl.Run(sio, "sleep", "10")
		readLine(in)
		return err
	}, 80, 24)
	require.NoError(t, err)
	defer p.Close()
	var out bytes.Buffer
	copied := make(chan struct{})
	go func() {
		io.Copy(&out, p) // ends when the pseudo-terminal is closed after the function
		close(copied)
	}()
	waitResult := func(want string) {
		t.Helper()
		select {
		case s := <-result:
			assert.Equal(t, want, s)
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for %q after the command", want)
		}
	}

	time.Sleep(300 * time.Millisecond) // head is running
	p.Write([]byte("to the host\r"))
	time.Sleep(300 * time.Millisecond) // head is done
	p.Write([]byte("after\r"))
	waitResult("after\n")

	time.Sleep(300 * time.Millisecond) // sleep is running
	start := time.Now()
	p.Write([]byte{0x03})
	time.Sleep(300 * time.Millisecond)
	p.Write([]byte("after interrupt\r"))
	waitResult("after interrupt\n")
	assert.Less(t, time.Since(start), 2*time.Second)
	assert.Error(t, p.Wait()) // sleep was interrupted
	<-copied
	assert.Contains(t, out.String(), "to the host")
}

// TestShell tests an interactive shell on the host, which starts in the
// current directory of the client and ends with exit.
func TestShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the paths of the test host (sh) are not Windows paths")
	}
	cl, s := testClient(t)
	s.Pty.Store(true)
	require.NoError(t, os.Mkdir(s.Root+"/sub", 0755))
	shell := func(input ...string) string {
		t.Helper()
		p, err := pty.Run(func(in, out *os.File) error {
			return cl.Shell(&exec.StdIOState{StdIO: exec.StdIO{In: in, Out: out, Err: out}})
		}, 80, 24)
		require.NoError(t, err)
		defer p.Close()
		var out bytes.Buffer
		copied := make(chan struct{})
		go func() {
			io.Copy(&out, p)
			close(copied)
		}()
		for _, in := range input {
			time.Sleep(300 * time.Millisecond)
			p.Write([]byte(in))
		}
		done := make(chan error, 1)
		go func() { done <- p.Wait() }()
		select {
		case err := <-done:
			assert.NoError(t, err) // the exit status is not an error
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for the shell to end")
		}
		<-copied
		return out.String()
	}
	out := shell("pwd\r", "exit 3\r")
	assert.Contains(t, out, s.Root+"\r\n")

	_, err := cl.Output(&exec.StdIOState{StdIO: exec.StdIO{Out: io.Discard, Err: io.Discard}}, "cd", "sub")
	require.NoError(t, err)
	out = shell("pwd\r", "\x04") // Control+D
	assert.Contains(t, out, s.Root+"/sub\r\n")
}
