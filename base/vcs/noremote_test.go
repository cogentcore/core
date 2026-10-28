// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoRemote tests a git repository without a remote,
// which the underlying vcs package does not support itself.
func TestNoRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return string(out)
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644))
	git("init", "-q")
	git("add", ".")
	git("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "init")

	for _, remote := range []string{"origin", ""} {
		repo, err := NewRepo(remote, dir)
		require.NoError(t, err, remote)
		assert.Equal(t, Git, repo.Type())
		assert.Equal(t, dir, repo.LocalPath())
		st, _ := repo.Status(filepath.Join(dir, "a.txt"))
		assert.Equal(t, Stored, st)
		cur, err := repo.Current()
		assert.NoError(t, err)
		assert.NotEmpty(t, cur)
	}

	repo, err := NewRepo("origin", dir)
	require.NoError(t, err)
	require.NoError(t, repo.Move(filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")))
	assert.Contains(t, git("status", "--porcelain"), "R  a.txt -> b.txt")
}

func TestFilesMoved(t *testing.T) {
	sep := string(filepath.Separator)
	fl := Files{"a.go": Stored, "d" + sep + "x.go": Modified, "d" + sep + "u.txt": Untracked, "dx.go": Stored}
	fl.moved("d", "e")
	assert.Equal(t, Files{"a.go": Stored, "e" + sep + "x.go": Added, "e" + sep + "u.txt": Untracked, "dx.go": Stored}, fl)
	fl.moved("a.go", "b.go")
	assert.Equal(t, Added, fl["b.go"])
	_, has := fl["a.go"]
	assert.False(t, has)
	var nilf Files
	nilf.moved("a", "b") // no panic
}

// TestSvnOps tests svn operations in a working copy (if svn is installed),
// which run in the repository directory.
func TestSvnOps(t *testing.T) {
	if _, err := exec.LookPath("svnadmin"); err != nil {
		t.Skip("svn is not available")
	}
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	run := func(dir string, args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	run(tmp, "svnadmin", "create", "repo")
	run(tmp, "svn", "checkout", "-q", "file://"+filepath.ToSlash(filepath.Join(tmp, "repo")), "wc")
	wc := filepath.Join(tmp, "wc")
	require.NoError(t, os.WriteFile(filepath.Join(wc, "a.txt"), []byte("a\n"), 0o644))

	repo, err := NewRepo("", wc)
	require.NoError(t, err)
	require.Equal(t, Svn, repo.Type())
	require.NoError(t, repo.Add(filepath.Join(wc, "a.txt")))
	require.NoError(t, repo.CommitFile(filepath.Join(wc, "a.txt"), "add a"))
	require.NoError(t, repo.Move(filepath.Join(wc, "a.txt"), filepath.Join(wc, "b.txt")))
	assert.FileExists(t, filepath.Join(wc, "b.txt"))
	st, _ := repo.Status(filepath.Join(wc, "b.txt"))
	assert.Equal(t, Added, st)
}
