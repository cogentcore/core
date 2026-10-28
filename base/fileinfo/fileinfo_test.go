// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDir returns a new directory with files a.go, .hidden,
// and sub/b.txt, and sub/deep/c.txt.
func testDir(t *testing.T) string {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	write(t, filepath.Join(dir, "a.go"), "package a\n")
	write(t, filepath.Join(dir, ".hidden"), "hidden\n")
	write(t, filepath.Join(dir, "sub", "b.txt"), "b\n")
	write(t, filepath.Join(dir, "sub", "deep", "c.txt"), "c\n")
	return dir
}

func write(t *testing.T, path, text string) {
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(text), 0o644))
}

func read(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

func info(t *testing.T, path string) *FileInfo {
	fi, err := NewFileInfo(path)
	require.NoError(t, err)
	return fi
}

func TestNewFileInfo(t *testing.T) {
	dir := testDir(t)
	fi := info(t, filepath.Join(dir, "a.go"))
	assert.Equal(t, "a.go", fi.Name)
	assert.Equal(t, filepath.Join(dir, "a.go"), fi.Path)
	assert.Equal(t, Go, fi.Known)
	assert.False(t, fi.IsDir())
	assert.EqualValues(t, len("package a\n"), fi.Size)

	fi = info(t, filepath.Join(dir, "sub"))
	assert.True(t, fi.IsDir())
	assert.Equal(t, Folder, fi.Cat)

	assert.True(t, info(t, filepath.Join(dir, ".hidden")).IsHidden())

	// a missing file is an error, but the type is set from the name
	fi, err := NewFileInfo(filepath.Join(dir, "missing.go"))
	assert.Error(t, err)
	assert.Equal(t, Go, fi.Known)
}

func TestDuplicate(t *testing.T) {
	dir := testDir(t)
	fi := info(t, filepath.Join(dir, "a.go"))
	dst, err := fi.Duplicate()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "a_Copy.go"), dst)
	assert.Equal(t, "package a\n", read(t, dst))
	dst, err = fi.Duplicate()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "a_Copy1.go"), dst)
	dst, err = fi.Duplicate()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "a_Copy2.go"), dst)

	// a name that is all extension
	dst, err = info(t, filepath.Join(dir, ".hidden")).Duplicate()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, ".hidden_Copy"), dst)

	// a directory, with all of its contents
	dst, err = info(t, filepath.Join(dir, "sub")).Duplicate()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "sub_Copy"), dst)
	assert.Equal(t, "c\n", read(t, filepath.Join(dst, "deep", "c.txt")))
}

func TestDuplicateMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no executable mode bits on Windows")
	}
	dir := testDir(t)
	exe := filepath.Join(dir, "run.sh")
	write(t, exe, "#!/bin/sh\n")
	require.NoError(t, os.Chmod(exe, 0o755))
	dst, err := info(t, exe).Duplicate()
	require.NoError(t, err)
	st, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), st.Mode().Perm())
}

func TestDelete(t *testing.T) {
	dir := testDir(t)
	require.NoError(t, info(t, filepath.Join(dir, "a.go")).Delete())
	assert.NoFileExists(t, filepath.Join(dir, "a.go"))
	require.NoError(t, info(t, filepath.Join(dir, "sub")).Delete())
	assert.NoDirExists(t, filepath.Join(dir, "sub"))
}

func TestRenamePath(t *testing.T) {
	dir := testDir(t)
	fi := info(t, filepath.Join(dir, "a.go"))
	np, err := fi.RenamePath("c.go")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "c.go"), np)

	np, err = fi.RenamePath(filepath.Join("sub", "c.go"))
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "sub", "c.go"), np)

	abs := filepath.Join(dir, "sub", "deep", "c.go")
	np, err = fi.RenamePath(abs)
	assert.NoError(t, err)
	assert.Equal(t, abs, np)

	np, err = fi.RenamePath("a.go")
	assert.NoError(t, err)
	assert.Equal(t, "", np, "same name")

	_, err = fi.RenamePath("")
	assert.Error(t, err)

	_, err = fi.RenamePath(".hidden")
	assert.ErrorContains(t, err, "already exists")
}

func TestRename(t *testing.T) {
	dir := testDir(t)
	fi := info(t, filepath.Join(dir, "a.go"))
	np, err := fi.Rename("c.go")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "c.go"), np)
	assert.Equal(t, np, fi.Path)
	assert.Equal(t, "c.go", fi.Name)
	assert.NoFileExists(t, filepath.Join(dir, "a.go"))
	assert.Equal(t, "package a\n", read(t, np))

	// into a subdirectory
	np, err = fi.Rename(filepath.Join("sub", "c.go"))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "sub", "c.go"), np)
	assert.FileExists(t, np)

	// the same name does nothing (it used to move the file to the current directory)
	wd, err := os.Getwd()
	require.NoError(t, err)
	np, err = fi.Rename("c.go")
	assert.NoError(t, err)
	assert.Equal(t, "", np)
	assert.FileExists(t, filepath.Join(dir, "sub", "c.go"))
	assert.NoFileExists(t, filepath.Join(wd, "c.go"))

	// an existing file is not replaced
	_, err = fi.Rename("b.txt")
	assert.Error(t, err)
	assert.Equal(t, "b\n", read(t, filepath.Join(dir, "sub", "b.txt")))
	assert.Equal(t, "package a\n", read(t, filepath.Join(dir, "sub", "c.go")))

	// a directory
	di := info(t, filepath.Join(dir, "sub"))
	np, err = di.Rename("other")
	require.NoError(t, err)
	assert.Equal(t, "c\n", read(t, filepath.Join(np, "deep", "c.txt")))
}

// TestRenameCase tests renaming to a different case only, which is
// the same file on case-insensitive file systems (macOS, Windows).
func TestRenameCase(t *testing.T) {
	dir := testDir(t)
	fi := info(t, filepath.Join(dir, "a.go"))
	np, err := fi.Rename("A.go")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "A.go"), np)
	ents, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := []string{}
	for _, e := range ents {
		names = append(names, e.Name())
	}
	assert.Contains(t, names, "A.go")
	assert.NotContains(t, names, "a.go")
}

func TestFilenames(t *testing.T) {
	dir := testDir(t)
	var names []string
	require.NoError(t, info(t, dir).Filenames(&names))
	slices.Sort(names)
	want := []string{
		filepath.Join(dir, ".hidden"),
		filepath.Join(dir, "a.go"),
		filepath.Join(dir, "sub", "b.txt"),
		filepath.Join(dir, "sub", "deep", "c.txt"),
	}
	slices.Sort(want)
	assert.Equal(t, want, names)

	assert.Error(t, info(t, filepath.Join(dir, "a.go")).Filenames(&names), "not a directory")
}
