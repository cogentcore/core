// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStdIO(t *testing.T) {
	var st StdIO
	st.SetFromOS()
	assert.Equal(t, os.Stdout, st.Out)
	assert.Equal(t, os.Stderr, st.Err)
	assert.Equal(t, os.Stdin, st.In)
	// assert.Equal(t, false, st.OutIsPipe())

	obuf := &bytes.Buffer{}
	ibuf := &bytes.Buffer{}
	var ss StdIOState
	ss.SetFromOS()
	ss.StackStart()
	// assert.Equal(t, false, ss.OutIsPipe())

	ss.PushOut(obuf)
	assert.NotEqual(t, os.Stdout, ss.Out)
	assert.Equal(t, obuf, ss.Out)
	ss.PushErr(obuf)
	assert.NotEqual(t, os.Stderr, ss.Err)
	assert.Equal(t, obuf, ss.Err)
	ss.PushIn(ibuf)
	assert.NotEqual(t, os.Stdin, ss.In)
	assert.Equal(t, ibuf, ss.In)
	assert.Equal(t, false, ss.OutIsPipe())

	ss.PopToStart()
	assert.Equal(t, os.Stdout, ss.Out)
	assert.Equal(t, os.Stderr, ss.Err)
	assert.Equal(t, os.Stdin, ss.In)
}

// closeCounter is a reader and writer that counts the calls of its Close.
type closeCounter struct {
	bytes.Buffer
	closed int
}

func (c *closeCounter) Close() error {
	c.closed++
	return nil
}

// TestPopToStartCloses checks that PopToStart closes what was pushed, once,
// including an Err that is the same as the current Out (>&), and Ins, and
// not the IO in place at StackStart.
func TestPopToStartCloses(t *testing.T) {
	orig := &closeCounter{}
	var ss StdIOState
	ss.SetAll(orig, orig, orig)
	ss.StackStart()

	out, in := &closeCounter{}, &closeCounter{}
	ss.PushOut(out)
	ss.PushErr(out)
	ss.PushIn(in)
	ss.PopToStart()
	assert.Equal(t, 1, out.closed)
	assert.Equal(t, 1, in.closed)
	assert.Equal(t, 0, orig.closed)
	assert.Equal(t, orig, ss.Out)
	assert.Equal(t, orig, ss.Err)
	assert.Equal(t, orig, ss.In)

	// an Err that is the Out in place at StackStart is not closed
	ss.StackStart()
	ss.PushErr(orig)
	ss.PopToStart()
	assert.Equal(t, 0, orig.closed)
}

// TestPushOutPipe checks that what is written to the Out of PushOutPipe is
// read from its PipeIn, until the Out is closed by PopToStart.
func TestPushOutPipe(t *testing.T) {
	var ss StdIOState
	ss.SetFromOS()
	ss.StackStart()
	ss.PushOutPipe()
	assert.True(t, ss.OutIsPipe())
	r := ss.PipeIn.Pop() // the reader is owned by the reading side
	done := make(chan string)
	go func() {
		b, err := io.ReadAll(r)
		assert.NoError(t, err)
		r.Close()
		done <- string(b)
	}()
	io.WriteString(ss.Out, "test")
	ss.PopToStart()
	assert.Equal(t, "test", <-done)
	assert.Equal(t, os.Stdout, ss.Out)
}

// TestSetToOS checks that SetToOS sets the os standard files from the
// files of StdIO, keeps those that are not files, and restores them.
func TestSetToOS(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "out")
	assert.NoError(t, err)
	defer f.Close()
	osOut, osIn := os.Stdout, os.Stdin
	st := &StdIO{Out: f, Err: os.Stderr, In: strings.NewReader("in")}
	prev := st.SetToOS()
	assert.Equal(t, f, os.Stdout)
	assert.Equal(t, osIn, os.Stdin)
	prev.SetToOS()
	assert.Equal(t, osOut, os.Stdout)
}
