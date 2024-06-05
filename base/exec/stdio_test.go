// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStdIO(t *testing.T) {
	t.Skip("todo: this does not work on CI; mostly reliable on mac")
	var st StdIO
	st.SetFromOS()
	assert.Equal(t, os.Stdout, st.Out)
	assert.Equal(t, os.Stderr, st.Err)
	assert.Equal(t, os.Stdin, st.In)
	assert.Equal(t, false, st.OutIsPipe())

	obuf := &bytes.Buffer{}
	ibuf := &bytes.Buffer{}
	var ss StdIOState
	ss.SetFromOS()
	ss.StackStart()
	assert.Equal(t, false, ss.OutIsPipe())

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

	ss.StackStart()
	ss.PushOutPipe()
	assert.Equal(t, true, ss.OutIsPipe())
	assert.Equal(t, 1, len(ss.PipeIn))
	pi := ss.PipeIn.Peek()
	go func() {
		b, err := io.ReadAll(pi)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, "test", string(b))
	}()
	io.WriteString(ss.Out, "test")

	// this is just cleanup after test:
	ss.PopToStart()
	assert.Equal(t, false, ss.OutIsPipe())
	assert.Equal(t, 0, len(ss.PipeIn))
}
