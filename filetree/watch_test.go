// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// watchTest is a shown [Tree] of a temporary directory, recording
// the changes passed to [Tree.WatchFunc].
type watchTest struct {
	t    *testing.T
	b    *core.Body
	ft   *Tree
	root string

	mu      sync.Mutex
	changes []FileChange
}

// newWatchTest makes a temporary directory with some files,
// shows a tree for it with the sub directory open, and waits
// until the watcher is running.
func newWatchTest(t *testing.T) *watchTest {
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	wt := &watchTest{t: t, root: root}
	wt.write("a.go", "package a\n")
	wt.write("sub/b.go", "package sub\n")
	wt.write("closed/c.txt", "c\n")
	wt.write(".git/HEAD", "ref: refs/heads/main\n")

	wt.b = core.NewBody()
	wt.ft = NewTree(wt.b)
	wt.ft.WatchFunc = func(changes []FileChange) {
		wt.mu.Lock()
		wt.changes = append(wt.changes, changes...)
		wt.mu.Unlock()
	}
	wt.ft.Dirs.setOpen("sub", true)
	wt.ft.OpenPath(root)
	t.Cleanup(wt.ft.stopWatcher)

	showed := make(chan struct{})
	wt.b.OnFinal(events.Show, func(e events.Event) {
		showed <- struct{}{}
	})
	wt.b.RunWindow()
	<-showed
	require.True(t, wt.ft.waitWatchReady(5*time.Second), "watcher did not start")
	// FSEvents can report the files written above just after starting.
	time.Sleep(500 * time.Millisecond)
	wt.clearChanges()
	return wt
}

// path returns the absolute path for the given relative path.
func (wt *watchTest) path(rel string) string {
	return filepath.Join(wt.root, filepath.FromSlash(rel))
}

// write writes the given text to the relative path, making directories.
func (wt *watchTest) write(rel, text string) {
	p := wt.path(rel)
	require.NoError(wt.t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(wt.t, os.WriteFile(p, []byte(text), 0o644))
}

func (wt *watchTest) clearChanges() {
	wt.mu.Lock()
	wt.changes = nil
	wt.mu.Unlock()
}

// hasChange returns whether a change was reported for the relative path.
func (wt *watchTest) hasChange(rel string, removed bool) bool {
	wt.mu.Lock()
	defer wt.mu.Unlock()
	return slices.Contains(wt.changes, FileChange{Path: wt.path(rel), Removed: removed})
}

// node returns the tree node for the relative path, or nil.
// It must be called under the lock.
func (wt *watchTest) node(rel string) *Node {
	p := wt.path(rel)
	dn := wt.ft.dirNode(filepath.Dir(p))
	if dn == nil {
		return nil
	}
	if c := dn.ChildByName(filepath.Base(p), 0); c != nil {
		return AsNode(c)
	}
	return nil
}

// hasNode returns whether the tree has a node for the relative path.
func (wt *watchTest) hasNode(rel string) bool {
	wt.b.AsyncLock()
	defer wt.b.AsyncUnlock()
	return wt.node(rel) != nil
}

// waitFor waits until cond returns true, failing the test after a timeout.
func (wt *watchTest) waitFor(what string, cond func() bool) {
	wt.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			wt.mu.Lock()
			wt.t.Fatalf("timed out waiting for %s; changes: %v", what, wt.changes)
			wt.mu.Unlock()
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWatchCreateRemove(t *testing.T) {
	wt := newWatchTest(t)
	assert.True(t, wt.hasNode("a.go"))
	assert.True(t, wt.hasNode("sub/b.go"))
	assert.False(t, wt.hasNode("closed/c.txt"), "closed directory is not shown")

	wt.write("new.txt", "new\n")
	wt.waitFor("new.txt node", func() bool { return wt.hasNode("new.txt") })
	assert.True(t, wt.hasChange("new.txt", false))

	wt.write("sub/new2.go", "package sub\n")
	wt.waitFor("sub/new2.go node", func() bool { return wt.hasNode("sub/new2.go") })

	require.NoError(t, os.Remove(wt.path("new.txt")))
	wt.waitFor("new.txt removed", func() bool { return !wt.hasNode("new.txt") })
	assert.True(t, wt.hasChange("new.txt", true))

	require.NoError(t, os.Mkdir(wt.path("newdir"), 0o755))
	wt.write("newdir/x.txt", "x\n")
	wt.waitFor("newdir node", func() bool { return wt.hasNode("newdir") })
	wt.b.AsyncLock()
	assert.True(t, wt.node("newdir").IsDir())
	wt.b.AsyncUnlock()

	require.NoError(t, os.RemoveAll(wt.path("newdir")))
	wt.waitFor("newdir removed", func() bool { return !wt.hasNode("newdir") })
}

func TestWatchRename(t *testing.T) {
	wt := newWatchTest(t)
	require.NoError(t, os.Rename(wt.path("sub/b.go"), wt.path("sub/c.go")))
	wt.waitFor("sub/c.go node", func() bool { return wt.hasNode("sub/c.go") })
	assert.False(t, wt.hasNode("sub/b.go"))
	wt.waitFor("sub/b.go removed change", func() bool { return wt.hasChange("sub/b.go", true) })
	assert.True(t, wt.hasChange("sub/c.go", false))
}

func TestWatchModify(t *testing.T) {
	wt := newWatchTest(t)
	text := "package a\n\nfunc A() {}\n"
	wt.write("a.go", text)
	wt.waitFor("a.go change", func() bool { return wt.hasChange("a.go", false) })
	wt.waitFor("a.go size", func() bool {
		wt.b.AsyncLock()
		defer wt.b.AsyncUnlock()
		return int(wt.node("a.go").Info.Size) == len(text)
	})

	// Editors often save by writing a temporary file and renaming it.
	wt.clearChanges()
	tmp := wt.path(".a.go.tmp123")
	require.NoError(t, os.WriteFile(tmp, []byte("package a\n"), 0o644))
	require.NoError(t, os.Rename(tmp, wt.path("a.go")))
	wt.waitFor("atomic save change", func() bool { return wt.hasChange("a.go", false) })
	assert.True(t, wt.hasNode("a.go"))
	assert.False(t, wt.hasNode(".a.go.tmp123"))

}

func TestWatchOpenClose(t *testing.T) {
	wt := newWatchTest(t)
	// Closed and hidden directories are not watched.
	wt.write("closed/c.txt", "changed\n")
	wt.write(".hidden/h.txt", "h\n")
	wt.write("marker.txt", "m\n")
	wt.waitFor("marker.txt change", func() bool { return wt.hasChange("marker.txt", false) })
	assert.False(t, wt.hasChange("closed/c.txt", false))
	assert.False(t, wt.hasChange(".hidden/h.txt", false))

	wt.b.AsyncLock()
	wt.node("closed").Open()
	wt.b.AsyncUnlock()
	require.True(t, wt.ft.waitWatchReady(5*time.Second))
	assert.True(t, wt.hasNode("closed/c.txt"))
	wt.write("closed/d.txt", "d\n")
	wt.waitFor("closed/d.txt node", func() bool { return wt.hasNode("closed/d.txt") })
	// the node can be added by a VCS update before the (batched) change
	wt.waitFor("closed/d.txt change", func() bool { return wt.hasChange("closed/d.txt", false) })

	wt.b.AsyncLock()
	wt.node("closed").Close()
	wt.b.AsyncUnlock()
	require.True(t, wt.ft.waitWatchReady(5*time.Second))
	wt.clearChanges()
	wt.write("closed/e.txt", "e\n")
	wt.write("marker2.txt", "m\n")
	wt.waitFor("marker2.txt change", func() bool { return wt.hasChange("marker2.txt", false) })
	time.Sleep(300 * time.Millisecond)
	assert.False(t, wt.hasChange("closed/e.txt", false))
}

func TestWatchIgnoresGit(t *testing.T) {
	wt := newWatchTest(t)
	wt.write(".git/HEAD", "ref: refs/heads/other\n")
	wt.write("marker.txt", "m\n")
	wt.waitFor("marker.txt change", func() bool { return wt.hasChange("marker.txt", false) })
	wt.mu.Lock()
	defer wt.mu.Unlock()
	for _, c := range wt.changes {
		assert.NotContains(t, c.Path, string(filepath.Separator)+".git", "change in .git reported")
	}
}

func TestWatchMany(t *testing.T) {
	wt := newWatchTest(t)
	const n = 200
	for i := range n {
		wt.write(filepath.Join("sub", "f"+string(rune('a'+i/26))+string(rune('a'+i%26))+".txt"), "x\n")
	}
	wt.waitFor("all files shown", func() bool {
		wt.b.AsyncLock()
		defer wt.b.AsyncUnlock()
		return wt.node("sub").NumChildren() == n+1 // plus b.go
	})
}

func TestWatchStop(t *testing.T) {
	wt := newWatchTest(t)
	wt.b.AsyncLock()
	wt.ft.stopWatcher()
	wt.b.AsyncUnlock()
	wt.write("late.txt", "late\n")
	time.Sleep(time.Second)
	assert.False(t, wt.hasNode("late.txt"))
	assert.False(t, wt.hasChange("late.txt", false))
}

func TestWatchNoWatch(t *testing.T) {
	b := core.NewBody()
	ft := NewTree(b).SetNoWatch(true)
	ft.OpenPath(t.TempDir())
	assert.Nil(t, ft.watch.Load())

	ft = NewTree(b)
	ft.OpenPathFS(os.DirFS(t.TempDir()), ".")
	assert.Nil(t, ft.watch.Load(), "no watcher for an fs.FS")
}
