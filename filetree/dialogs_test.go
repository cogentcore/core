// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// clickDialog waits for the dialog with the given title to be on top of
// the window of the given tree, and clicks its button with the given text.
func clickDialog(t *testing.T, ft *Tree, title, button string) {
	t.Helper()
	b := ft.Scene.Body
	for range 500 {
		b.AsyncLock()
		if sc := core.TopScene(b); sc != nil && sc.Body != nil && sc.Body.Title == title {
			bt := core.ButtonByText(sc, button)
			if bt != nil {
				bt.Send(events.Click)
			}
			b.AsyncUnlock() // (before failing, which runs the cleanup that locks)
			require.NotNil(t, bt, button)
			return
		}
		b.AsyncUnlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no dialog %q", title)
}

// selectFiles selects the nodes of the given relative paths.
func selectFiles(t *testing.T, ft *Tree, dir string, rels ...string) *Node {
	t.Helper()
	var last *Node
	for i, rel := range rels {
		n := opsNode(t, ft, dir, rel)
		mode := events.ExtendOne
		if i == 0 {
			mode = events.SelectOne
		}
		n.SelectEvent(mode)
		last = n
	}
	return last
}

// TestDeleteFilesDialog tests deleting the selected files after confirming
// it in a dialog, and not deleting them when it is canceled.
func TestDeleteFilesDialog(t *testing.T) {
	ft, dir := newOpsTree(t)
	ft.AsyncLock()
	selectFiles(t, ft, dir, "a.go", "c.txt").DeleteFiles()
	ft.AsyncUnlock()
	clickDialog(t, ft, "Delete Files?", "Cancel")
	_, err := os.Stat(filepath.Join(dir, "a.go"))
	assert.NoError(t, err, "not deleted")

	ft.AsyncLock()
	selectFiles(t, ft, dir, "a.go", "c.txt").DeleteFiles()
	ft.AsyncUnlock()
	clickDialog(t, ft, "Delete Files?", "Delete Files")
	for _, f := range []string{"a.go", "c.txt"} {
		_, err := os.Stat(filepath.Join(dir, f))
		assert.True(t, os.IsNotExist(err), f)
	}
	_, err = os.Stat(filepath.Join(dir, "b.go"))
	assert.NoError(t, err, "not selected")
}

// TestPasteOntoFile tests pasting a file onto another one, which asks
// whether to overwrite it.
func TestPasteOntoFile(t *testing.T) {
	ft, dir := newOpsTree(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a // new\n"), 0o644))
	ft.AsyncLock()
	selectFiles(t, ft, dir, "a.go").Copy()
	b := selectFiles(t, ft, dir, "b.go")
	b.Paste()
	ft.AsyncUnlock()
	clickDialog(t, ft, "Overwrite?", "Overwrite target")
	got, err := os.ReadFile(filepath.Join(dir, "b.go"))
	require.NoError(t, err)
	assert.Equal(t, "package a // new\n", string(got))
}

// TestFileCommands tests cutting (which only copies), the files dragged
// to other apps, opening files in their default apps, and the file info.
func TestFileCommands(t *testing.T) {
	ft, dir := newOpsTree(t)
	var opened []string
	prev := openURL
	openURL = func(url string) { opened = append(opened, url) }
	defer func() { openURL = prev }()

	ft.AsyncLock()
	c := selectFiles(t, ft, dir, "a.go", "c.txt")
	assert.ElementsMatch(t, []string{filepath.Join(dir, "a.go"), filepath.Join(dir, "c.txt")}, c.DragFiles())
	c.OpenFilesDefault()
	assert.Len(t, opened, 2)
	assert.True(t, strings.HasPrefix(opened[0], "file://"), opened[0])
	c.Cut()
	md := c.Clipboard().Read([]string{fileinfo.TextPlain})
	assert.Contains(t, md.Text(fileinfo.TextPlain), "c.txt")
	ft.AsyncUnlock()
	clickDialog(t, ft, "", "OK") // (a message dialog has no title)
	_, err := os.Stat(filepath.Join(dir, "c.txt"))
	assert.NoError(t, err, "cut does not delete")
}

// TestVCSSelected tests the version control commands for the selected
// files: adding, removing, reverting, and the log and blame windows.
func TestVCSSelected(t *testing.T) {
	defer core.TrackWindows()()
	dir, git := newGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.go"), []byte("package a\n"), 0o644))
	ft := newVCSTree(t, dir)
	b := vcsNode(t, ft, dir, "b.go")
	a := vcsNode(t, ft, dir, "a.go")

	ft.AsyncLock()
	b.SelectEvent(events.SelectOne)
	b.addToVCSSelected()
	ft.AsyncUnlock()
	assert.Contains(t, git("status", "--porcelain"), "A  b.go")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("changed\n"), 0o644))
	ft.AsyncLock()
	a.Info.VCS = vcs.Modified
	a.SelectEvent(events.SelectOne)
	a.revertVCSSelected()
	ft.AsyncUnlock()
	got, err := os.ReadFile(filepath.Join(dir, "a.go"))
	require.NoError(t, err)
	assert.Equal(t, "package a\n\nvar A = 1\n", string(got), "reverted")

	nw := core.AllRenderWindows.Len()
	ft.AsyncLock()
	a.Info.VCS = vcs.Stored
	a.logVCSSelected()
	a.blameVCSSelected()
	ft.AsyncUnlock()
	assert.Eventually(t, func() bool { return core.AllRenderWindows.Len() >= nw+2 }, 5*time.Second, 10*time.Millisecond, "the log and blame windows")

	ft.AsyncLock()
	a.deleteFromVCSSelected()
	ft.AsyncUnlock()
	assert.Contains(t, git("status", "--porcelain"), "D  a.go")
}
