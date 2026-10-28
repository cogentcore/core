// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errorLog records the messages of error logs about the files of a test,
// which include those shown in error snackbars. Trees of earlier tests
// can still be updated, logging errors for their deleted directories.
type errorLog struct {
	slog.Handler
	root string
	mu   sync.Mutex
	msgs []string
}

func (el *errorLog) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError && strings.Contains(r.Message, el.root) {
		el.mu.Lock()
		el.msgs = append(el.msgs, r.Message)
		el.mu.Unlock()
	}
	return el.Handler.Handle(ctx, r)
}

// errors returns the logged error messages, and clears them.
func (el *errorLog) errors() []string {
	el.mu.Lock()
	defer el.mu.Unlock()
	m := el.msgs
	el.msgs = nil
	return m
}

// logErrors records error logs about the files of the test
// (in its root directory) for the duration of the test.
func (wt *watchTest) logErrors() *errorLog {
	t := wt.t
	prev := slog.Default()
	// not wrapping prev.Handler(), which logs through slog once replaced: a deadlock
	el := &errorLog{Handler: slog.NewTextHandler(os.Stderr, nil), root: wt.root}
	slog.SetDefault(slog.New(el))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return el
}

// do runs the given function under the tree lock, as for an action.
func (wt *watchTest) do(f func()) {
	wt.ft.AsyncLock()
	defer wt.ft.AsyncUnlock()
	f()
}

// selectNodes selects the nodes for the given relative paths (only).
func (wt *watchTest) selectNodes(rels ...string) {
	wt.do(func() {
		wt.ft.UnselectAll()
		for _, rel := range rels {
			n := wt.node(rel)
			require.NotNil(wt.t, n, rel)
			n.Select()
		}
	})
}

// drop drops the nodes for the given relative paths on the node for
// target, with the given drop modifier, as for a drag in the tree.
func (wt *watchTest) drop(target string, mod events.DropMods, rels ...string) {
	wt.do(func() {
		var md mimedata.Mimes
		var src *Node
		for _, rel := range rels {
			n := wt.node(rel)
			require.NotNil(wt.t, n, rel)
			if src == nil {
				src = n
			}
			n.MimeData(&md)
		}
		tn := wt.node(target)
		require.NotNil(wt.t, tn, target)
		tn.DragDrop(dropEvent(src, md, mod))
	})
}

// dropExternal drops the given data from another app on the node for target.
func (wt *watchTest) dropExternal(target string, md mimedata.Mimes) {
	wt.do(func() {
		tn := wt.node(target)
		require.NotNil(wt.t, tn, target)
		tn.DragDrop(dropEvent(nil, md, events.DropCopy))
	})
}

func dropEvent(src *Node, md mimedata.Mimes, mod events.DropMods) *events.DragDrop {
	de := &events.DragDrop{DropMod: mod, Data: md}
	if src != nil {
		de.Source = src.This
	}
	de.Typ = events.Drop
	return de
}

func (wt *watchTest) read(rel string) string {
	b, err := os.ReadFile(wt.path(rel))
	require.NoError(wt.t, err, rel)
	return string(b)
}

func TestNewFileFolder(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	wt.selectNodes("sub")
	wt.do(func() { wt.node("sub").newFiles("n.txt", false) })
	assert.Equal(t, "", wt.read("sub/n.txt"))
	wt.waitFor("new file node", func() bool { return wt.hasNode("sub/n.txt") })
	assert.Empty(t, el.errors())

	// an existing file is not truncated
	wt.do(func() { wt.node("sub").newFile("b.go", false) })
	assert.Equal(t, "package sub\n", wt.read("sub/b.go"))
	assert.NotEmpty(t, el.errors())

	wt.do(func() { wt.node("sub").newFolder("nf") })
	assert.DirExists(t, wt.path("sub/nf"))
	wt.waitFor("new folder node", func() bool { return wt.hasNode("sub/nf") })
	assert.Empty(t, el.errors())
}

func TestDuplicateFiles(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	wt.selectNodes("a.go", "sub")
	wt.do(func() { wt.node("a.go").duplicateFiles() })
	assert.Equal(t, "package a\n", wt.read("a_Copy.go"))
	assert.Equal(t, "package sub\n", wt.read("sub_Copy/b.go"))
	wt.waitFor("duplicate nodes", func() bool { return wt.hasNode("a_Copy.go") && wt.hasNode("sub_Copy") })
	assert.Empty(t, el.errors())
}

// TestDeleteFiles tests deleting files, including a directory and a file
// within it, which must not report an error for the file.
func TestDeleteFiles(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	wt.selectNodes("sub", "sub/b.go", "u.txt", "a.go")
	wt.do(func() { wt.node("a.go").DeleteFilesNoPrompts() })
	assert.NoDirExists(t, wt.path("sub"))
	assert.NoFileExists(t, wt.path("u.txt"))
	assert.NoFileExists(t, wt.path("a.go"))
	wt.waitFor("deleted nodes", func() bool {
		return !wt.hasNode("sub") && !wt.hasNode("u.txt") && !wt.hasNode("a.go")
	})
	assert.Empty(t, el.errors())
	assert.Contains(t, wt.git("status", "--porcelain"), "D  a.go", "tracked: git rm")
}

func TestRenameDirVCS(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	require.NoError(t, wt.rename("sub", "pkg"))
	assert.NoDirExists(t, wt.path("sub"))
	assert.Equal(t, "package sub\n", wt.read("pkg/b.go"))
	assert.Contains(t, wt.git("status", "--porcelain"), "R  sub/b.go -> pkg/b.go")
	wt.waitFor("renamed nodes", func() bool { return wt.hasNode("pkg/b.go") && !wt.hasNode("sub") })
	wt.do(func() {
		assert.Equal(t, wt.path("pkg/b.go"), string(wt.node("pkg/b.go").Filepath))
	})
	assert.Empty(t, el.errors())
}

// TestRenameDirUntracked tests renaming a directory with no files in git.
func TestRenameDirUntracked(t *testing.T) {
	wt := newRenameTest(t)
	wt.write("new/x.txt", "x\n")
	wt.do(func() { wt.ft.UpdatePath(wt.root) })
	wt.waitFor("new dir", func() bool { return wt.hasNode("new") })
	require.NoError(t, wt.rename("new", "newer"))
	assert.Equal(t, "x\n", wt.read("newer/x.txt"))
}

func TestRenameExisting(t *testing.T) {
	wt := newRenameTest(t)
	assert.Error(t, wt.rename("a.go", "u.txt"))
	assert.Equal(t, "package a\n", wt.read("a.go"))
	assert.Equal(t, "untracked\n", wt.read("u.txt"))
	assert.Error(t, wt.rename("u.txt", "a.go"))
	assert.Equal(t, "package a\n", wt.read("a.go"))
}

func TestDropMove(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	wt.drop("sub", events.DropMove, "a.go")
	assert.Equal(t, "package a\n", wt.read("sub/a.go"))
	assert.NoFileExists(t, wt.path("a.go"))
	wt.waitFor("moved node", func() bool { return wt.hasNode("sub/a.go") && !wt.hasNode("a.go") })
	assert.Empty(t, el.errors())
}

func TestDropCopy(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	wt.drop("sub", events.DropCopy, "a.go", "u.txt")
	assert.Equal(t, "package a\n", wt.read("sub/a.go"))
	assert.Equal(t, "untracked\n", wt.read("sub/u.txt"))
	assert.FileExists(t, wt.path("a.go"))
	assert.FileExists(t, wt.path("u.txt"))
	assert.Empty(t, el.errors())
}

// TestDropInPlace tests dropping files on the directory they are in:
// a move does nothing (it used to delete them), and a copy duplicates them.
func TestDropInPlace(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	wt.drop("sub", events.DropMove, "sub/b.go")
	assert.Equal(t, "package sub\n", wt.read("sub/b.go"))
	assert.NoFileExists(t, wt.path("sub/b_Copy.go"))

	// on a sibling file in the same directory, with the file
	wt.drop("a.go", events.DropMove, "u.txt", "a.go")
	assert.Equal(t, "package a\n", wt.read("a.go"))
	assert.Equal(t, "untracked\n", wt.read("u.txt"))

	// dropped on itself
	wt.drop("a.go", events.DropMove, "a.go")
	assert.Equal(t, "package a\n", wt.read("a.go"))

	wt.drop("sub", events.DropCopy, "sub/b.go")
	assert.Equal(t, "package sub\n", wt.read("sub/b_Copy.go"))
	assert.Empty(t, el.errors())
}

// TestDropIntoItself tests moving a directory into itself, which fails
// without deleting it.
func TestDropIntoItself(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	wt.drop("sub", events.DropMove, "sub")
	assert.Equal(t, "package sub\n", wt.read("sub/b.go"))
	assert.NoDirExists(t, wt.path("sub/sub"))
	assert.NotEmpty(t, el.errors())
}

// TestDropNested tests moving a directory along with a file within it,
// which is moved with the directory.
func TestDropNested(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	wt.do(func() { wt.ft.newFolder("dst") })
	wt.waitFor("dst", func() bool { return wt.hasNode("dst") })
	wt.drop("dst", events.DropMove, "sub", "sub/b.go")
	assert.Equal(t, "package sub\n", wt.read("dst/sub/b.go"))
	assert.NoFileExists(t, wt.path("dst/b.go"))
	assert.NoDirExists(t, wt.path("sub"))
	assert.Empty(t, el.errors())
}

// TestDropExternal tests dropping files from another app: several on a
// file go into its directory, and directories and file modes are copied.
func TestDropExternal(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	other := t.TempDir()
	x := filepath.Join(other, "x.sh")
	require.NoError(t, os.WriteFile(x, []byte("#!/bin/sh\n"), 0o755))
	y := filepath.Join(other, "y.txt")
	require.NoError(t, os.WriteFile(y, []byte("y\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(other, "d"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(other, "d", "z.txt"), []byte("z\n"), 0o644))

	wt.dropExternal("a.go", mimedata.Mimes{mimedata.NewTextData(x), mimedata.NewTextData("file://" + y)})
	assert.Equal(t, "#!/bin/sh\n", wt.read("x.sh"))
	assert.Equal(t, "y\n", wt.read("y.txt"))
	assert.Equal(t, "package a\n", wt.read("a.go"))
	if runtime.GOOS != "windows" {
		st, err := os.Stat(wt.path("x.sh"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), st.Mode().Perm())
	}

	// a directory on a file goes into its directory
	wt.dropExternal("sub/b.go", mimedata.Mimes{mimedata.NewTextData(filepath.Join(other, "d"))})
	assert.Equal(t, "z\n", wt.read("sub/d/z.txt"))
	assert.Empty(t, el.errors())
}

// TestDropOtherApp tests dropping files dragged from a file tree in
// another app, which has the data of [Node.MimeData] without a source.
func TestDropOtherApp(t *testing.T) {
	wt := newRenameTest(t)
	el := wt.logErrors()
	other := t.TempDir()
	x := filepath.Join(other, "x.txt")
	require.NoError(t, os.WriteFile(x, []byte("x\n"), 0o644))
	md := mimedata.Mimes{mimedata.NewTextData("x.txt"), mimedata.NewTextData(x), mimedata.NewTextData("x\n")}
	wt.dropExternal("sub", md)
	assert.Equal(t, "x\n", wt.read("sub/x.txt"))
	assert.Empty(t, el.errors())
}

func TestMimeDataItems(t *testing.T) {
	wt := newRenameTest(t)
	var md mimedata.Mimes
	wt.do(func() {
		for _, rel := range []string{"sub", "a.go", "u.txt"} {
			wt.node(rel).MimeData(&md)
		}
	})
	assert.Len(t, md, 9, "three items per file, including directories")
	assert.Equal(t, []string{wt.path("sub"), wt.path("a.go"), wt.path("u.txt")}, filePaths(md))
	assert.True(t, isNodeData(md))
	assert.Equal(t, "package a\n", string(md[5].Data))

	// clipboard data as copied from the tree
	assert.Equal(t, filePaths(md), clipboardPaths(md))
	// paths in text, with any other text ignored
	text := wt.path("a.go") + "\nfile://" + wt.path("u.txt") + "\nnot a path\n/no/such/file\n"
	assert.Equal(t, []string{wt.path("a.go"), wt.path("u.txt")}, clipboardPaths(mimedata.NewText(text)))
	assert.Empty(t, clipboardPaths(mimedata.NewText("just some text")))

	assert.Equal(t, []string{wt.path("a.go")}, filePaths(withoutFiles(md, []string{wt.path("sub"), wt.path("u.txt")})))
}

func TestTopPaths(t *testing.T) {
	sep := string(filepath.Separator)
	d := sep + "d"
	assert.Equal(t, []string{d, sep + "e"}, topPaths([]string{d, d + sep + "a", sep + "e", d + sep + "x" + sep + "y"}))
	assert.Equal(t, []string{d, d + "x"}, topPaths([]string{d, d + "x"}), "prefix of the name is not within")
	assert.True(t, strings.HasPrefix(d+sep+"a", d))
}
