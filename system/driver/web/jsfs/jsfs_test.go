// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package jsfs

import (
	"io"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOS(t *testing.T) {
	fs, err := Config(Global())
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, fs.ConfigUnix())

	dir := "/home/me/Documents/jsfstest"
	assert.NoError(t, os.RemoveAll(dir))
	assert.NoError(t, os.MkdirAll(dir+"/sub", 0755))
	assert.NoError(t, os.WriteFile(dir+"/a.txt", []byte("hello"), 0644))

	b, err := os.ReadFile(dir + "/a.txt")
	assert.NoError(t, err)
	assert.Equal(t, "hello", string(b))

	info, err := os.Stat(dir + "/a.txt")
	assert.NoError(t, err)
	assert.Equal(t, int64(5), info.Size())
	assert.False(t, info.IsDir())

	f, err := os.Create(dir + "/b.txt")
	assert.NoError(t, err)
	_, err = f.WriteString("abcdef")
	assert.NoError(t, err)
	off, err := f.Seek(2, io.SeekStart)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), off)
	buf := make([]byte, 3)
	n, err := f.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "cde", string(buf[:n]))
	assert.NoError(t, f.Close())

	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Equal(t, []string{"a.txt", "b.txt", "sub"}, names)

	assert.NoError(t, os.Rename(dir+"/b.txt", dir+"/sub/c.txt"))
	assert.NoError(t, os.Remove(dir+"/a.txt"))
	_, err = os.Stat(dir + "/a.txt")
	assert.True(t, os.IsNotExist(err))

	sub, err := os.ReadDir(dir + "/sub")
	assert.NoError(t, err)
	assert.True(t, slices.ContainsFunc(sub, func(e os.DirEntry) bool { return e.Name() == "c.txt" }))

	assert.NoError(t, os.RemoveAll(dir))
	_, err = os.Stat(dir)
	assert.True(t, os.IsNotExist(err))
}
