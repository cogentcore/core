// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fsx

//core:js-test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"cogentcore.org/core/base/goscript"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeFiles makes the given files (with their names as contents) in dir,
// and the given subdirectories.
func makeFiles(t *testing.T, dir string, files []string, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, d), 0755))
	}
	for _, f := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, f), []byte(f), 0644))
	}
}

// TestFilesInDir tests listing the files and directories of a directory,
// with extension filters, and their latest modification time.
func TestFilesInDir(t *testing.T) {
	dir := t.TempDir()
	makeFiles(t, dir, []string{"b.go", "a.GO", "c.txt", "d"}, "sub", "zdir")

	var names []string
	for _, f := range Files(dir, ".go") {
		names = append(names, f.Name())
	}
	assert.Equal(t, []string{"a.GO", "b.go"}, names, "extensions match any case")
	assert.Len(t, Files(dir), 6, "all files and directories")
	assert.Nil(t, Files(filepath.Join(dir, "missing")))
	assert.Nil(t, Files(filepath.Join(dir, "sub"), ".go"), "empty directory")

	assert.Equal(t, []string{"a.GO", "b.go", "c.txt"}, Filenames(dir, ".go", ".txt"))
	assert.Equal(t, []string{"a.GO", "b.go", "c.txt", "d", "sub", "zdir"}, Filenames(dir))
	assert.Nil(t, Filenames(filepath.Join(dir, "missing")))
	assert.Nil(t, Filenames(filepath.Join(dir, "sub"), ".go"))
	assert.Equal(t, []string{"sub", "zdir"}, Dirs(dir))
	assert.Nil(t, Dirs(filepath.Join(dir, "missing")))

	old := time.Now().Add(-time.Hour)
	newer := time.Now().Add(-time.Minute)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "a.GO"), old, old))
	require.NoError(t, os.Chtimes(filepath.Join(dir, "b.go"), newer, newer))
	assert.WithinDuration(t, newer, LatestMod(dir, ".go"), time.Second)
	assert.True(t, LatestMod(filepath.Join(dir, "missing")).IsZero())

	assert.True(t, HasFile(dir, "d"))
	assert.False(t, HasFile(dir, "e"))
	assert.False(t, HasFile(filepath.Join(dir, "missing"), "d"))
	assert.Equal(t, []string{filepath.Join(dir, "c.txt")}, FindFilesOnPaths([]string{filepath.Join(dir, "sub"), dir}, "c.txt", "sub"), "only files, not directories")
}

// TestExists tests the functions that check for files and directories.
func TestExists(t *testing.T) {
	dir := t.TempDir()
	makeFiles(t, dir, []string{"f"}, "d")
	f, d, m := filepath.Join(dir, "f"), filepath.Join(dir, "d"), filepath.Join(dir, "m")

	ok, err := FileExists(f)
	assert.True(t, ok)
	assert.NoError(t, err)
	ok, _ = FileExists(d)
	assert.False(t, ok, "a directory is not a file")
	ok, err = FileExists(m)
	assert.False(t, ok)
	assert.NoError(t, err)
	assert.True(t, Exists(d))
	assert.False(t, Exists(m))
	ok, err = DirExists(d)
	assert.True(t, ok)
	assert.NoError(t, err)
	ok, _ = DirExists(f)
	assert.False(t, ok)
	_, err = DirExists(m)
	assert.Error(t, err)
}

// noStatFS is an [fs.FS] without Stat, which is checked by opening files.
type noStatFS struct{ fsys fs.FS }

func (n noStatFS) Open(name string) (fs.File, error) { return n.fsys.Open(name) }

// TestFS tests the file system functions, with and without Stat.
func TestFS(t *testing.T) {
	mfs := fstest.MapFS{"a/b.txt": {Data: []byte("b")}}
	for _, fsys := range []fs.FS{mfs, noStatFS{mfs}} {
		ok, err := FileExistsFS(fsys, "a/b.txt")
		assert.True(t, ok)
		assert.NoError(t, err)
		ok, err = FileExistsFS(fsys, "a/c.txt")
		assert.False(t, ok)
		assert.NoError(t, err)
		ok, _ = FileExistsFS(fsys, "a")
		assert.False(t, ok, "a directory is not a file, %T", fsys)
	}
	sub := Sub(mfs, "a")
	ok, _ := FileExistsFS(sub, "b.txt")
	assert.True(t, ok)

	dir := t.TempDir()
	makeFiles(t, dir, []string{"x.txt"})
	dfs, name, err := DirFS(filepath.Join(dir, "x.txt"))
	require.NoError(t, err)
	assert.Equal(t, "x.txt", name)
	ok, _ = FileExistsFS(dfs, name)
	assert.True(t, ok)
}

// TestPaths tests the path helper functions.
func TestPaths(t *testing.T) {
	goscript.Skip(t, "uses the files of the operating system")
	assert.Equal(t, filepath.Join("b", "c.go"), DirAndFile(filepath.Join("a", "b", "c.go")))
	assert.Equal(t, filepath.Join("b", "c.go"), RelativeFilePath(filepath.Join("/x", "a", "b", "c.go"), filepath.Join("/x", "a")))
	assert.Equal(t, filepath.Join("b", "c.go"), RelativeFilePath(filepath.Join("/x", "a", "b", "c.go"), filepath.Join("/y")), "outside the root")
	base, ext := ExtSplit("dir/file.tar.gz")
	assert.Equal(t, "dir/file.tar", base)
	assert.Equal(t, ".gz", ext)
	base, ext = ExtSplit("noext")
	assert.Equal(t, "noext", base)
	assert.Equal(t, "", ext)
	assert.NotEmpty(t, GoSrcDirs())
}

// TestCopyFile tests copying files with permissions, and the errors.
func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	makeFiles(t, dir, []string{"src.txt"})
	dst := filepath.Join(dir, "dst.txt")
	require.NoError(t, CopyFile(dst, filepath.Join(dir, "src.txt"), 0600))
	b, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "src.txt", string(b))
	fi, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fi.Mode().Perm())
	assert.Error(t, CopyFile(dst, filepath.Join(dir, "missing"), 0600))
	assert.Error(t, CopyFile(filepath.Join(dir, "no-dir", "x"), filepath.Join(dir, "src.txt"), 0600))
	assert.Error(t, CopyPath(dst, filepath.Join(dir, "missing")))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 2, "no temporary files are left")
}
