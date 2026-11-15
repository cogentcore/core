// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package yamlx

//core:js-test

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
	fn := filepath.Join(dir, "a.yaml")
	require.NoError(t, Save(s, fn))

	var buf bytes.Buffer
	require.NoError(t, Write(s, &buf))
	b, err := WriteBytes(s)
	require.NoError(t, err)
	assert.Equal(t, buf.Bytes(), b)

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
	fn2 := filepath.Join(dir, "b.yaml")
	require.NoError(t, Save(&testStruct{A: "bbb", B: 1}, fn2))
	r = &testStruct{}
	require.NoError(t, OpenFiles(r, fn, fn2))
	assert.Equal(t, "bbb", r.A)
	assert.Error(t, OpenFiles(r, fn, filepath.Join(dir, "missing.yaml")))

	data, err := os.ReadFile(fn)
	require.NoError(t, err)
	fsys := fstest.MapFS{"a.yaml": {Data: data}, "b.yaml": {Data: b}}
	r = &testStruct{}
	require.NoError(t, OpenFS(r, fsys, "a.yaml"))
	assert.Equal(t, s, r)
	r = &testStruct{}
	require.NoError(t, OpenFilesFS(r, fsys, "a.yaml", "b.yaml"))
	assert.Equal(t, s, r)
	assert.Error(t, OpenFS(r, fsys, "missing.yaml"))
	assert.Error(t, OpenFilesFS(r, fsys, "missing.yaml"))

	assert.Error(t, Open(r, filepath.Join(dir, "missing.yaml")))
	assert.Error(t, Save(s, filepath.Join(dir, "no-dir", "a.yaml")))
	assert.Error(t, ReadBytes(r, []byte(`a: [b`)))
}
