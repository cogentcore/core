// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGitTest returns a new git repository (without a remote)
// with a committed file a.txt, and a function to run git in it.
func newGitTest(t *testing.T) (string, func(args ...string) string) {
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
	git("init", "-q", "-b", "main")
	git("add", ".")
	git("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "init")
	return dir, git
}

func TestGitRepo(t *testing.T) {
	dir, git := newGitTest(t)
	for _, remote := range []string{"origin", ""} {
		repo, err := NewRepo(remote, dir)
		require.NoError(t, err, remote)
		assert.Equal(t, Git, repo.Type())
		assert.Equal(t, dir, repo.LocalPath())
		assert.Equal(t, "", repo.Remote(), "no remote")
		st, _ := repo.Status(filepath.Join(dir, "a.txt"))
		assert.Equal(t, Stored, st)
		cur, err := repo.Current()
		assert.NoError(t, err)
		assert.Equal(t, "main", cur)
	}

	git("remote", "add", "origin", "https://example.com/r.git")
	repo, err := NewRepo("origin", dir)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/r.git", repo.Remote())
	repo, err = NewRepo("https://example.com/other.git", dir)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/other.git", repo.Remote())

	git("branch", "dev")
	br, err := repo.Branches()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"main", "dev"}, br)

	// checking out a revision detaches HEAD
	rev := strings.TrimSpace(git("rev-parse", "HEAD"))
	require.NoError(t, repo.UpdateVersion(rev))
	cur, err := repo.Current()
	require.NoError(t, err)
	assert.Equal(t, rev, cur)
	assert.Error(t, repo.UpdateVersion("no-such-revision"))

	require.NoError(t, repo.Move(filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")))
	assert.Contains(t, git("status", "--porcelain"), "R  a.txt -> b.txt")
}

// TestGitUpdate tests updating a clone from its origin with git pull.
func TestGitUpdate(t *testing.T) {
	dir, git := newGitTest(t)
	clone := filepath.Join(t.TempDir(), "clone")
	git("clone", "-q", dir, clone)
	repo, err := NewRepo("", clone)
	require.NoError(t, err)
	assert.Equal(t, dir, repo.Remote())

	require.NoError(t, os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new\n"), 0o644))
	git("add", ".")
	git("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "new")
	require.NoError(t, repo.Update())
	assert.FileExists(t, filepath.Join(clone, "new.txt"))
}

func TestNewRepoNone(t *testing.T) {
	_, err := NewRepo("", t.TempDir())
	assert.Error(t, err)
	// a .git that is not a valid repository
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	_, err = NewRepo("", dir)
	assert.Error(t, err)
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
