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
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRenameTest returns a tree on a new git repository (without a remote)
// with committed files a.go and sub/b.go, and untracked u.txt.
func newRenameTest(t *testing.T) *watchTest {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	wt := &watchTest{t: t, root: root}
	wt.write("a.go", "package a\n")
	wt.write("sub/b.go", "package sub\n")
	wt.git("init", "-q")
	wt.git("add", ".")
	wt.git("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "init")
	wt.write("u.txt", "untracked\n")

	wt.b = core.NewBody()
	wt.ft = NewTree(wt.b)
	wt.ft.NoWatch = true
	wt.ft.Dirs.setOpen("sub", true)
	wt.ft.OpenPath(root)
	showed := make(chan struct{})
	wt.b.OnFinal(events.Show, func(e events.Event) {
		showed <- struct{}{}
	})
	wt.b.RunWindow()
	<-showed
	wt.waitFor("vcs status", func() bool {
		n := wt.node("a.go")
		return n != nil && n.Info.VCS == vcs.Stored
	})
	return wt
}

// git runs git with the given arguments in the root, returning its output.
func (wt *watchTest) git(args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = wt.root
	out, err := cmd.CombinedOutput()
	require.NoError(wt.t, err, string(out))
	return string(out)
}

// rename renames the node at rel to the given new name with [Node.RenameFile].
func (wt *watchTest) rename(rel, name string) error {
	n := wt.node(rel)
	require.NotNil(wt.t, n, rel)
	wt.ft.AsyncLock()
	defer wt.ft.AsyncUnlock()
	return n.RenameFile(name)
}

func TestRenameVCS(t *testing.T) {
	wt := newRenameTest(t)
	require.NoError(t, wt.rename("a.go", "c.go"))
	assert.NoFileExists(t, wt.path("a.go"))
	assert.FileExists(t, wt.path("c.go"))
	// git knows about the rename (staged), rather than a deleted
	// a.go and an untracked c.go
	status := wt.git("status", "--porcelain")
	assert.Contains(t, status, "R  a.go -> c.go", status)
	wt.waitFor("renamed node", func() bool { return wt.hasNode("c.go") && !wt.hasNode("a.go") })
	assert.Equal(t, wt.path("c.go"), string(wt.node("c.go").Filepath))
}

func TestRenameUntracked(t *testing.T) {
	wt := newRenameTest(t)
	require.NoError(t, wt.rename("u.txt", "v.txt"))
	assert.NoFileExists(t, wt.path("u.txt"))
	assert.FileExists(t, wt.path("v.txt"))
	wt.waitFor("renamed node", func() bool { return wt.hasNode("v.txt") && !wt.hasNode("u.txt") })
}

// TestRenameSameName tests that "renaming" a file to its own name,
// as when accepting the rename prompt without changes, does nothing.
func TestRenameSameName(t *testing.T) {
	wt := newRenameTest(t)
	wd, err := os.Getwd()
	require.NoError(t, err)
	assert.NoError(t, wt.rename("u.txt", "u.txt"))
	assert.FileExists(t, wt.path("u.txt"))
	assert.NoFileExists(t, filepath.Join(wd, "u.txt"))
	os.Remove(filepath.Join(wd, "u.txt")) // in case it was moved there
}

// TestRenameToDir tests renaming with a relative path,
// which moves the file into the sub directory.
func TestRenameToDir(t *testing.T) {
	wt := newRenameTest(t)
	require.NoError(t, wt.rename("a.go", filepath.Join("sub", "a.go")))
	assert.NoFileExists(t, wt.path("a.go"))
	assert.FileExists(t, wt.path("sub/a.go"))
	status := wt.git("status", "--porcelain")
	assert.True(t, strings.Contains(status, "R  a.go -> sub/a.go"), status)
}
