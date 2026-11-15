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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newGitRepo returns a git test repository (see newGitTest) with a local
// identity for commits, as a [Repo].
func newGitRepo(t *testing.T) (string, func(args ...string) string, *GitRepo) {
	dir, git := newGitTest(t)
	git("config", "user.name", "Tester")
	git("config", "user.email", "tester@example.com")
	repo, err := NewRepo("", dir)
	require.NoError(t, err)
	return dir, git, repo.(*GitRepo)
}

// TestGitFiles tests the statuses of files, from git and as listed.
func TestGitFiles(t *testing.T) {
	dir, _, repo := newGitRepo(t)
	write := func(name, s string) string {
		fn := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(fn, []byte(s), 0o644))
		return fn
	}
	a := write("a.txt", "changed\n")
	u := write("u.txt", "new\n")
	n := write("n.txt", "added\n")
	require.NoError(t, repo.Add(n))

	st, _ := repo.Status(a)
	assert.Equal(t, Modified, st)
	st, _ = repo.Status(u)
	assert.Equal(t, Untracked, st)
	st, _ = repo.Status(n)
	assert.Equal(t, Added, st)
	assert.Equal(t, Stored, repo.StatusFast(a), "not known yet")

	done := make(chan Files, 1)
	f, err := repo.Files(func(f Files) { done <- f })
	require.NoError(t, err)
	assert.Nil(t, f, "the files are being listed")
	var got Files
	select {
	case got = <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the files were not listed")
	}
	assert.Equal(t, Modified, got["a.txt"])
	assert.Equal(t, Untracked, got["u.txt"])
	assert.Equal(t, Added, got["n.txt"])
	assert.Equal(t, Modified, repo.StatusFast(a))
	assert.Equal(t, Untracked, got.Status(repo, filepath.Join(dir, "missing.txt")))
	f, err = repo.Files(nil)
	require.NoError(t, err)
	assert.Equal(t, got, f, "now known")

	for c, want := range map[byte]FileStatus{'M': Modified, 'A': Added, 'D': Deleted, 'U': Conflicted, '?': Untracked, 'X': Untracked} {
		assert.Equal(t, want, repo.charToStat(c), string(c))
	}
}

// TestGitHistory tests committing, the log, contents of revisions,
// commit descriptions, changes, and blame.
func TestGitHistory(t *testing.T) {
	dir, git, repo := newGitRepo(t)
	a := filepath.Join(dir, "a.txt")
	require.NoError(t, os.WriteFile(a, []byte("a2\n"), 0o644))
	require.NoError(t, repo.CommitFile(a, "second"))

	lg, err := repo.Log("", "")
	require.NoError(t, err)
	require.Len(t, lg, 2)
	assert.Equal(t, "second", lg[0].Message)
	assert.Equal(t, "init", lg[1].Message)
	assert.Equal(t, "test", lg[1].Author, "from the commit of newGitTest")
	assert.Equal(t, "tester@example.com", lg[0].Email)
	lg, err = repo.Log(a, "1 year ago")
	require.NoError(t, err)
	assert.Len(t, lg, 2, "since a year ago")

	cur, err := repo.FileContents(a, "")
	require.NoError(t, err)
	assert.Equal(t, "a2\n", string(cur))
	prev, err := repo.FileContents(a, "-1")
	require.NoError(t, err)
	assert.Equal(t, "a\n", string(prev))
	first := strings.TrimSpace(git("rev-list", "--max-parents=0", "HEAD"))
	prev, err = repo.FileContents(a, first)
	require.NoError(t, err)
	assert.Equal(t, "a\n", string(prev))
	_, err = repo.FileContents(filepath.Join(dir, "missing.txt"), "")
	assert.Error(t, err)
	_, err = repo.FileContents(a, "no-such-rev")
	assert.Error(t, err)

	desc, err := repo.CommitDesc("", false)
	require.NoError(t, err)
	assert.Contains(t, string(desc), "M\ta.txt")
	desc, err = repo.CommitDesc("-1", true)
	require.NoError(t, err)
	assert.Contains(t, string(desc), "+a", "the diff of the first commit")
	_, err = repo.CommitDesc("no-such-rev", false)
	assert.Error(t, err)

	ch, err := repo.FilesChanged("-1", "", false)
	require.NoError(t, err)
	assert.Contains(t, string(ch), "M\ta.txt")
	ch, err = repo.FilesChanged("-1", "HEAD", true)
	require.NoError(t, err)
	assert.Contains(t, string(ch), "+a2")
	ch, err = repo.FilesChanged("", "-1", false)
	require.NoError(t, err)
	assert.Contains(t, string(ch), "a.txt")
	_, err = repo.FilesChanged("no-such-rev", "", false)
	assert.Error(t, err)

	bl, err := repo.Blame(a)
	require.NoError(t, err)
	assert.Contains(t, string(bl), "a2")
	_, err = repo.Blame(filepath.Join(dir, "missing.txt"))
	assert.Error(t, err)
}

// TestGitDelete tests deleting files, and removing them from the
// repository only.
func TestGitDelete(t *testing.T) {
	dir, git, repo := newGitRepo(t)
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	require.NoError(t, os.WriteFile(b, []byte("b\n"), 0o644))
	require.NoError(t, repo.Add(b))
	require.NoError(t, repo.CommitFile(b, "b"))

	require.NoError(t, repo.DeleteRemote(b))
	assert.FileExists(t, b, "kept on disk")
	assert.Contains(t, git("status", "--porcelain"), "D  b.txt")
	require.NoError(t, repo.Delete(a))
	assert.NoFileExists(t, a)
	assert.Error(t, repo.Delete(filepath.Join(dir, "missing.txt")))

	require.NoError(t, os.WriteFile(b, []byte("changed\n"), 0o644))
	git("add", "b.txt")
	git("commit", "-q", "-m", "b back")
	require.NoError(t, os.WriteFile(b, []byte("changed again\n"), 0o644))
	require.NoError(t, repo.RevertFile(b))
	got, err := os.ReadFile(b)
	require.NoError(t, err)
	assert.Equal(t, "changed\n", string(got), "reverted")
}

// TestDetectRepoRoot tests finding the root of the repository of a
// directory inside of it.
func TestDetectRepoRoot(t *testing.T) {
	dir, _, _ := newGitRepo(t)
	sub := filepath.Join(dir, "x", "y")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	root, typ := DetectRepoRoot(sub)
	assert.Equal(t, dir, root)
	assert.Equal(t, Git, typ)
	root, typ = DetectRepoRoot(t.TempDir())
	if typ == NoVCS {
		assert.Equal(t, "", root)
	}
}

// TestGitLogSince tests the log since a relative date (which quotes
// around the date in the git argument broke), with a commit from long ago.
func TestGitLogSince(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	commit := func(name, date string) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644))
		for _, args := range [][]string{{"add", name}, {"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", name}} {
			cmd := exec.Command("git", args...)
			cmd.Dir = dir
			if date != "" {
				cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
			}
			out, err := cmd.CombinedOutput()
			require.NoError(t, err, string(out))
		}
	}
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	commit("old.txt", "2020-01-01T00:00:00")
	commit("new.txt", "")
	repo, err := NewRepo("", dir)
	require.NoError(t, err)
	lg, err := repo.Log("", "1 year ago")
	require.NoError(t, err)
	require.Len(t, lg, 1, "only the new commit")
	assert.Equal(t, "new.txt", lg[0].Message)
	lg, err = repo.Log("", "")
	require.NoError(t, err)
	assert.Len(t, lg, 2)
}
