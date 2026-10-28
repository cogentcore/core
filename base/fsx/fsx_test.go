// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fsx

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoSrcDir(t *testing.T) {
	if len(GoSrcDirs()) == 0 {
		t.Skip("there are no Go source directories on the web")
	}
	dir1 := "fmt"
	expected1 := filepath.Join(build.Default.GOROOT, "src", dir1)
	absDir1, err := GoSrcDir(dir1)
	assert.NoError(t, err)
	assert.Equal(t, expected1, absDir1)

	dir2 := "nonexistent"
	_, err = GoSrcDir(dir2)
	assert.Error(t, err)
	assert.EqualError(t, err, fmt.Sprintf("fsx.GoSrcDir: unable to locate directory (%q) in GOPATH/src/ (%q) or GOROOT/src/pkg/ (%q)", dir2, os.Getenv("GOPATH"), os.Getenv("GOROOT")))
}

func TestSplitRootPathFS(t *testing.T) {
	type tst struct{ p, root, rest string }
	cases := []tst{{"/a/b/c", "/", "a/b/c"}, {"a/b/c", "a", "b/c"}, {"a", "a", ""}, {"a/", "a", ""}}

	for _, ts := range cases {
		root, rest := SplitRootPathFS(ts.p)
		assert.Equal(t, ts.root, root)
		assert.Equal(t, ts.rest, rest)
	}
}

func TestCopyPath(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	assert.NoError(t, os.MkdirAll(filepath.Join(src, "sub"), 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(src, "a.txt"), []byte("new a"), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("b"), 0644))

	// an existing directory is merged into, overwriting files
	assert.NoError(t, os.WriteFile(filepath.Join(dst, "a.txt"), []byte("old a"), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(dst, "keep.txt"), []byte("keep"), 0644))
	assert.NoError(t, CopyPath(dst, src))
	for f, want := range map[string]string{"a.txt": "new a", "sub/b.txt": "b", "keep.txt": "keep"} {
		b, err := os.ReadFile(filepath.Join(dst, f))
		assert.NoError(t, err)
		assert.Equal(t, want, string(b))
	}

	// a single file
	assert.NoError(t, CopyPath(filepath.Join(dst, "c.txt"), filepath.Join(src, "a.txt")))
	b, err := os.ReadFile(filepath.Join(dst, "c.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "new a", string(b))

	// not into itself, which would recurse forever
	assert.Error(t, CopyPath(src, src))
	assert.Error(t, CopyPath(filepath.Join(src, "sub", "x"), src))
	assert.NoDirExists(t, filepath.Join(src, "sub", "x"))
}

func TestFileURL(t *testing.T) {
	assert.Equal(t, "file:///home/me/a%20b%23c.txt", FileURL("/home/me/a b#c.txt"))
	assert.Equal(t, "file:///C:/Users/me/a.txt", FileURL(`C:/Users/me/a.txt`))
	if filepath.Separator == '\\' {
		assert.Equal(t, "file:///C:/Users/me/a.txt", FileURL(`C:\Users\me\a.txt`))
	}
}
