// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xmlx

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRoundTrip tests saving, writing, opening, and reading values,
// from files, file systems, and bytes, and the errors of missing files.
func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := &testStruct{A: "aaa", B: 3.5}
	fn := filepath.Join(dir, "a.xml")
	require.NoError(t, Save(s, fn))

	var buf bytes.Buffer
	require.NoError(t, Write(s, &buf))
	b, err := WriteBytes(s)
	require.NoError(t, err)
	assert.Equal(t, buf.Bytes(), b)

	var ibuf bytes.Buffer
	require.NoError(t, WriteIndent(s, &ibuf))
	ib, err := WriteBytesIndent(s)
	require.NoError(t, err)
	assert.Equal(t, ibuf.Bytes(), ib)
	assert.Greater(t, len(ib), len(b), "indented")
	ifn := filepath.Join(dir, "i.xml")
	require.NoError(t, SaveIndent(s, ifn))
	r0 := &testStruct{}
	require.NoError(t, Open(r0, ifn))
	assert.Equal(t, s, r0)

	r := &testStruct{}
	require.NoError(t, Read(r, bytes.NewReader(b)))
	assert.Equal(t, s, r)
	r = &testStruct{}
	require.NoError(t, ReadBytes(r, b))
	assert.Equal(t, s, r)
	r = &testStruct{}
	require.NoError(t, Open(r, fn))
	assert.Equal(t, s, r)

	// later files override earlier ones
	fn2 := filepath.Join(dir, "b.xml")
	require.NoError(t, Save(&testStruct{A: "bbb", B: 1}, fn2))
	r = &testStruct{}
	require.NoError(t, OpenFiles(r, fn, fn2))
	assert.Equal(t, "bbb", r.A)
	assert.Error(t, OpenFiles(r, fn, filepath.Join(dir, "missing.xml")))

	data, err := os.ReadFile(fn)
	require.NoError(t, err)
	fsys := fstest.MapFS{"a.xml": {Data: data}, "b.xml": {Data: b}}
	r = &testStruct{}
	require.NoError(t, OpenFS(r, fsys, "a.xml"))
	assert.Equal(t, s, r)
	r = &testStruct{}
	require.NoError(t, OpenFilesFS(r, fsys, "a.xml", "b.xml"))
	assert.Equal(t, s, r)
	assert.Error(t, OpenFS(r, fsys, "missing.xml"))
	assert.Error(t, OpenFilesFS(r, fsys, "missing.xml"))

	assert.Error(t, Open(r, filepath.Join(dir, "missing.xml")))
	assert.Error(t, Save(s, filepath.Join(dir, "no-dir", "a.xml")))
	assert.Error(t, ReadBytes(r, []byte(`<a><b></a>`)))
}
