// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gzipx

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeText returns a function that writes the given text.
func writeText(text string) func(w io.Writer) error {
	return func(w io.Writer) error {
		_, err := io.WriteString(w, text)
		return err
	}
}

// readText returns a function that reads all of the text into s.
func readText(s *string) func(r io.Reader) error {
	return func(r io.Reader) error {
		b, err := io.ReadAll(r)
		*s = string(b)
		return err
	}
}

// TestSaveOpen tests saving and opening files, which are compressed only
// with a .gz extension.
func TestSaveOpen(t *testing.T) {
	dir := t.TempDir()
	text := strings.Repeat("hello gzip ", 100)
	gz := filepath.Join(dir, "a.txt.gz")
	plain := filepath.Join(dir, "a.txt")
	require.NoError(t, Save(gz, writeText(text)))
	require.NoError(t, Save(plain, writeText(text)))

	zb, err := os.ReadFile(gz)
	require.NoError(t, err)
	assert.Less(t, len(zb), len(text), "compressed")
	assert.Equal(t, []byte{0x1f, 0x8b}, zb[:2], "the gzip magic number")
	pb, err := os.ReadFile(plain)
	require.NoError(t, err)
	assert.Equal(t, text, string(pb), "not compressed")

	var got string
	require.NoError(t, Open(gz, readText(&got)))
	assert.Equal(t, text, got)
	require.NoError(t, Open(plain, readText(&got)))
	assert.Equal(t, text, got)

	fsys := fstest.MapFS{"a.txt.gz": {Data: zb}, "a.txt": {Data: pb}}
	require.NoError(t, OpenFS(fsys, "a.txt.gz", readText(&got)))
	assert.Equal(t, text, got)
	require.NoError(t, OpenFS(fsys, "a.txt", readText(&got)))
	assert.Equal(t, text, got)

	var buf bytes.Buffer
	require.NoError(t, Write(&buf, writeText("x")))
	require.NoError(t, Read(&buf, readText(&got)))
	assert.Equal(t, "x", got)
}

// TestErrors tests that missing files and invalid gzip data are errors,
// not panics.
func TestErrors(t *testing.T) {
	dir := t.TempDir()
	var got string
	assert.Error(t, Open(filepath.Join(dir, "missing.gz"), readText(&got)))
	assert.Error(t, OpenFS(fstest.MapFS{}, "missing.gz", readText(&got)))
	assert.Error(t, Save(filepath.Join(dir, "no-dir", "a.gz"), writeText("x")))
	assert.Error(t, Read(strings.NewReader("not gzip"), readText(&got)))
	bad := filepath.Join(dir, "bad.gz")
	require.NoError(t, os.WriteFile(bad, []byte("not gzip"), 0666))
	assert.Error(t, Open(bad, readText(&got)))
}
