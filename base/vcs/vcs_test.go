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
// which run in the repository directory, not the current one.
func TestSvnOps(t *testing.T) {
	if _, err := exec.LookPath("svnadmin"); err != nil {
		t.Skip("svn is not available")
	}
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	run := func(dir string, args ...string) string {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return string(out)
	}
	run(tmp, "svnadmin", "create", "repo")
	url := "file://" + filepath.ToSlash(filepath.Join(tmp, "repo"))
	run(tmp, "svn", "checkout", "-q", url, "wc")
	wc := filepath.Join(tmp, "wc")
	path := func(name string) string { return filepath.Join(wc, name) }
	write := func(name, text string) {
		require.NoError(t, os.WriteFile(path(name), []byte(text), 0o644))
	}
	read := func(name string) string {
		b, err := os.ReadFile(path(name))
		require.NoError(t, err)
		return string(b)
	}

	repo, err := NewRepo("", wc)
	require.NoError(t, err)
	require.Equal(t, Svn, repo.Type())
	assert.Equal(t, wc, repo.LocalPath())
	assert.Equal(t, url, repo.Remote())

	write("a.txt", "a\n")
	require.NoError(t, repo.Add(path("a.txt")))
	st, _ := repo.Status(path("a.txt"))
	assert.Equal(t, Added, st)
	require.NoError(t, repo.CommitFile(path("a.txt"), "add a"))
	st, _ = repo.Status(path("a.txt"))
	assert.Equal(t, Stored, st)
	cur, err := repo.Current()
	require.NoError(t, err)
	assert.Equal(t, "0", cur, "the working copy is at revision 0 until updated")
	require.NoError(t, repo.Update())
	cur, err = repo.Current()
	require.NoError(t, err)
	assert.Equal(t, "1", cur)

	// the cached file status
	files := make(chan Files, 1)
	repo.Files(func(f Files) { files <- f })
	<-files
	assert.Equal(t, Stored, repo.StatusFast(path("a.txt")))

	// local changes: the working copy file for "", committed for HEAD
	write("a.txt", "changed\n")
	st, _ = repo.Status(path("a.txt"))
	assert.Equal(t, Modified, st)
	b, err := repo.FileContents(path("a.txt"), "")
	require.NoError(t, err)
	assert.Equal(t, "changed\n", string(b))
	b, err = repo.FileContents(path("a.txt"), "HEAD")
	require.NoError(t, err)
	assert.Equal(t, "a\n", string(b))
	b, err = repo.FileContents(path("a.txt"), "1")
	require.NoError(t, err)
	assert.Equal(t, "a\n", string(b))
	require.NoError(t, repo.RevertFile(path("a.txt")))
	assert.Equal(t, "a\n", read("a.txt"))

	blame, err := repo.Blame(path("a.txt"))
	require.NoError(t, err)
	assert.Contains(t, string(blame), "a")
	lg, err := repo.Log(path("a.txt"), "")
	require.NoError(t, err)
	assert.Len(t, lg, 1)

	require.NoError(t, repo.Move(path("a.txt"), path("b.txt")))
	assert.FileExists(t, path("b.txt"))
	st, _ = repo.Status(path("b.txt"))
	assert.Equal(t, Added, st)
	assert.Equal(t, Added, repo.StatusFast(path("b.txt")), "cache updated by Move")
	require.NoError(t, repo.CommitFile(wc, "move a to b"))

	require.NoError(t, repo.DeleteRemote(path("b.txt")))
	assert.FileExists(t, path("b.txt"), "kept locally")
	st, _ = repo.Status(path("b.txt"))
	assert.Equal(t, Deleted, st)
	require.NoError(t, repo.RevertFile(path("b.txt")))
	require.NoError(t, repo.Delete(path("b.txt")))
	assert.NoFileExists(t, path("b.txt"))

	require.NoError(t, repo.Update())
	require.NoError(t, repo.UpdateVersion("1"))
	cur, err = repo.Current()
	require.NoError(t, err)
	assert.Equal(t, "1", cur)
	assert.FileExists(t, path("a.txt"), "back at revision 1")

	assert.Error(t, repo.Add(path("missing.txt")), "with svn's message")
}
