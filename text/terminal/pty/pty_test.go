// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pty

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPty(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "js" {
		t.Skip("uses sh")
	}
	cmd := exec.Command("sh", "-c", `stty size; [ -t 0 ] && echo tty; read x; echo "got $x"`)
	p, err := Start(cmd, 33, 7)
	require.NoError(t, err)
	_, err = p.Write([]byte("hello\r"))
	require.NoError(t, err)
	var out bytes.Buffer
	done := make(chan struct{})
	go func() { // reads until the last output, as macOS does not give EOF
		b := make([]byte, 1024)
		for !strings.Contains(out.String(), "got hello") {
			n, err := p.Read(b)
			out.Write(b[:n])
			if err != nil {
				break
			}
		}
		close(done)
	}()
	assert.NoError(t, p.Wait())
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	}
	p.Close()
	s := strings.ReplaceAll(out.String(), "\r", "")
	assert.Contains(t, s, "7 33\n")
	assert.Contains(t, s, "tty\n")
	assert.Contains(t, s, "got hello")
}
