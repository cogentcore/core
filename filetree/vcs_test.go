// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGitRepo returns a new git repository in a temporary directory, with
// a committed file a.go, and a function that runs git in it.
func newGitRepo(t *testing.T) (string, func(args ...string) string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
		return string(out)
	}
	git("init", "-q", "-b", "main")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.com")
	git("config", "commit.gpgsign", "false")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n\nvar A = 1\n"), 0o644))
	git("add", "a.go")
	git("commit", "-q", "-m", "first")
	return dir, git
}

// newVCSTree returns a tree in a window, opened at the given directory.
func newVCSTree(t *testing.T, dir string) *Tree {
	t.Helper()
	b := core.NewBody()
	ft := NewTree(b)
	ft.OpenPath(dir)
	runTestWindow(t, b)
	return ft
}

// vcsNode returns the node of the given file in the tree, updating the
// tree first so that it has new files.
func vcsNode(t *testing.T, ft *Tree, dir, name string) *Node {
	t.Helper()
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	ft.Update() // reads the directory again
	n, ok := ft.FindFile(filepath.Join(dir, name))
	require.True(t, ok, name)
	return n
}

// TestVCSRepo tests finding the repository of a tree and of its files.
func TestVCSRepo(t *testing.T) {
	dir, _ := newGitRepo(t)
	ft := newVCSTree(t, dir)
	a := vcsNode(t, ft, dir, "a.go")
	ft.AsyncLock()
	defer ft.AsyncUnlock()
	repo, rnode := ft.FirstVCS()
	require.NotNil(t, repo)
	assert.Equal(t, vcs.Git, repo.Type())
	assert.Equal(t, ft.AsNode(), rnode)
	frepo, fnode := a.Repo()
	assert.Equal(t, repo, frepo)
	assert.Equal(t, rnode, fnode)
}

// TestVCSAddCommitRevert tests adding a new file, committing it, and
// reverting a change to it, through the nodes of the tree.
func TestVCSAddCommitRevert(t *testing.T) {
	dir, git := newGitRepo(t)
	ft := newVCSTree(t, dir)
	bpath := filepath.Join(dir, "b.go")
	require.NoError(t, os.WriteFile(bpath, []byte("package a\n\nvar B = 2\n"), 0o644))
	b := vcsNode(t, ft, dir, "b.go")

	ft.AsyncLock()
	b.AddToVCS()
	assert.Equal(t, vcs.Added, b.Info.VCS)
	ft.AsyncUnlock()
	assert.Contains(t, git("status", "--porcelain"), "A  b.go")

	ft.AsyncLock()
	require.NoError(t, b.commitToVCS("add b"))
	assert.Equal(t, vcs.Stored, b.Info.VCS)
	ft.AsyncUnlock()
	assert.Contains(t, git("log", "--oneline"), "add b")

	require.NoError(t, os.WriteFile(bpath, []byte("changed\n"), 0o644))
	ft.AsyncLock()
	b.Info.VCS = vcs.Modified
	require.NoError(t, b.revertVCS())
	assert.Equal(t, vcs.Stored, b.Info.VCS)
	ft.AsyncUnlock()
	got, err := os.ReadFile(bpath)
	require.NoError(t, err)
	assert.Equal(t, "package a\n\nvar B = 2\n", string(got), "reverted")

	ft.AsyncLock()
	b.Info.VCS = vcs.Untracked
	assert.Error(t, b.commitToVCS("no"), "an untracked file is not committed")
	assert.Error(t, b.revertVCS())
	ft.AsyncUnlock()
}

// TestVCSDelete tests removing a file from the repository, which keeps
// the file.
func TestVCSDelete(t *testing.T) {
	dir, git := newGitRepo(t)
	ft := newVCSTree(t, dir)
	a := vcsNode(t, ft, dir, "a.go")
	ft.AsyncLock()
	a.deleteFromVCS()
	assert.Equal(t, vcs.Deleted, a.Info.VCS)
	ft.AsyncUnlock()
	assert.Contains(t, git("status", "--porcelain"), "D  a.go")
	_, err := os.Stat(filepath.Join(dir, "a.go"))
	assert.NoError(t, err, "the file is kept")
}

// TestVCSLogBlame tests the log and blame of a file, which are shown in
// dialogs.
func TestVCSLogBlame(t *testing.T) {
	dir, git := newGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n\nvar A = 2\n"), 0o644))
	git("commit", "-q", "-am", "second")
	ft := newVCSTree(t, dir)
	a := vcsNode(t, ft, dir, "a.go")

	ft.AsyncLock()
	a.Info.VCS = vcs.Stored
	lg, err := a.LogVCS(false, "")
	ft.AsyncUnlock()
	require.NoError(t, err)
	require.Len(t, lg, 2)
	assert.Equal(t, "second", lg[0].Message)
	assert.Equal(t, "Test", lg[0].Author)

	ft.AsyncLock()
	all, err := a.LogVCS(true, "")
	ft.AsyncUnlock()
	require.NoError(t, err)
	assert.Len(t, all, 2)

	ft.AsyncLock()
	blame, err := a.blameVCS()
	ft.AsyncUnlock()
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(blame)), "\n")
	assert.Len(t, lines, 3)
	assert.Contains(t, lines[2], "var A = 2")

	ft.AsyncLock()
	a.Info.VCS = vcs.Untracked
	_, err = a.LogVCS(false, "")
	assert.Error(t, err)
	assert.Error(t, a.diffVCS("", ""))
	ft.AsyncUnlock()
}
