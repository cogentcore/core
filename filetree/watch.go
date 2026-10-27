// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"cmp"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// FileChange is a change to a file or directory on disk,
// detected by the file system watcher of a [Tree].
// See [Tree.WatchFunc].
type FileChange struct {

	// Path is the absolute path of the file or directory.
	Path string

	// Removed is true if the path no longer exists
	// (it was deleted or renamed to another path).
	Removed bool

	// IsDir is true if the path is a directory. Files within a changed
	// directory may have changed without being listed separately,
	// for example when the watcher could not keep up with all events.
	IsDir bool
}

// watchBatch is how long the watcher collects changes before applying them.
const watchBatch = 200 * time.Millisecond

// treeWatcher is a running file system watcher for a [Tree].
type treeWatcher struct {

	// root is the root directory of the tree.
	root string

	// ctx is canceled when the watcher is stopped.
	ctx context.Context

	// cancel stops the watcher.
	cancel context.CancelFunc

	// ready is closed when the watcher has started watching.
	ready chan struct{}

	// backend is the platform file system watcher.
	backend *watchBackend

	// watched are the directories being watched.
	// It is only used in the event loop.
	watched map[string]bool
}

// watchBackend is the interface to the platform file system watcher.
type watchBackend struct {

	// events receives the changed paths. An empty path
	// means that events were lost, so anything may have changed.
	events <-chan string

	// add starts watching the files in the given directory,
	// and drop stops it. They do not block.
	add, drop func(dir string)

	// busy returns whether add or drop operations are in progress.
	busy func() bool
}

// startWatcher starts watching the open directories of the tree
// for changes, stopping any watcher of a different root directory.
// Only the files directly within open, non-hidden directories are
// watched, so the watched directories are updated as directories
// are opened and closed (see [Tree.syncWatchDirs]).
func (ft *Tree) startWatcher() {
	root := string(ft.Filepath)
	if tw := ft.watch.Load(); tw != nil {
		if tw.root == root {
			return
		}
		ft.stopWatcher()
	}
	if ft.NoWatch || ft.FS != nil || root == "" {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	tw := &treeWatcher{root: root, ctx: ctx, cancel: cancel, ready: make(chan struct{}), watched: make(map[string]bool)}
	dirs := ft.watchableDirs()
	be, err := watchDirs(ctx, dirs, tw.ready)
	if err != nil || be == nil {
		cancel()
		if err != nil {
			slog.Error("filetree: could not watch directory", "path", root, "err", err)
		}
		return
	}
	tw.backend = be
	for _, d := range dirs {
		tw.watched[d] = true
	}
	ft.watch.Store(tw)
	go ft.watchEvents(tw, be.events)
}

// watchableDirs returns the directories that should be watched:
// the root and the open directories shown in the tree, excluding
// hidden ones (starting with a .) and those within them.
func (ft *Tree) watchableDirs() []string {
	dirs := []string{string(ft.Filepath)}
	var walk func(n *Node)
	walk = func(n *Node) {
		for _, c := range n.Children {
			cn := AsNode(c)
			if cn == nil || !cn.IsDir() || cn.IsIrregular() || cn.Name == externalFilesName || strings.HasPrefix(cn.Name, ".") {
				continue
			}
			if !ft.isDirOpen(cn.Filepath) {
				continue
			}
			dirs = append(dirs, string(cn.Filepath))
			walk(cn)
		}
	}
	walk(&ft.Node)
	return dirs
}

// syncWatchDirs updates the watched directories to match
// the open directories in the tree.
func (ft *Tree) syncWatchDirs() {
	tw := ft.watch.Load()
	if tw == nil {
		return
	}
	want := make(map[string]bool)
	for _, d := range ft.watchableDirs() {
		want[d] = true
		if !tw.watched[d] {
			tw.watched[d] = true
			tw.backend.add(d)
		}
	}
	for d := range tw.watched {
		if !want[d] {
			delete(tw.watched, d)
			tw.backend.drop(d)
		}
	}
}

// stopWatcher stops the file system watcher, if any.
func (ft *Tree) stopWatcher() {
	if tw := ft.watch.Swap(nil); tw != nil {
		tw.cancel()
	}
}

// waitWatchReady waits until the watcher has started watching and
// has finished adding and dropping directories, returning false if
// there is no watcher or it does not get there in time.
func (ft *Tree) waitWatchReady(timeout time.Duration) bool {
	tw := ft.watch.Load()
	if tw == nil {
		return false
	}
	deadline := time.After(timeout)
	select {
	case <-tw.ready:
	case <-deadline:
		return false
	}
	for tw.backend.busy() {
		select {
		case <-deadline:
			return false
		case <-time.After(10 * time.Millisecond):
		}
	}
	return true
}

// watchEvents collects the changed paths sent by the watcher
// into batches and applies them to the tree. An empty path means
// that events were lost, so anything may have changed.
func (ft *Tree) watchEvents(tw *treeWatcher, events <-chan string) {
	pending := make(map[string]bool)
	timer := time.NewTimer(watchBatch)
	timer.Stop()
	for {
		select {
		case <-tw.ctx.Done():
			return
		case path, ok := <-events:
			if !ok {
				return
			}
			if len(pending) == 0 {
				timer.Reset(watchBatch)
			}
			pending[path] = true
		case <-timer.C:
			if !ft.applyChanges(tw, slices.Sorted(maps.Keys(pending))) {
				timer.Reset(watchBatch) // not shown yet; try again
				continue
			}
			clear(pending)
		}
	}
}

// applyChanges updates the tree for the given changed paths and calls
// [Tree.WatchFunc]. The state of each path is read from disk, as event
// types are not reliable (FSEvents accumulates flags for a path).
// It returns false if the tree is not shown, so the changes can be
// applied later.
func (ft *Tree) applyChanges(tw *treeWatcher, paths []string) bool {
	if !ft.AsyncLockIfShown() {
		return tw.ctx.Err() != nil
	}
	defer ft.AsyncUnlock()
	if ft.watch.Load() != tw {
		return true
	}
	changes := make([]FileChange, 0, len(paths))
	update := make(map[*Node]bool)
	for _, path := range paths {
		if path == "" { // lost events
			update[&ft.Node] = true
			changes = append(changes, FileChange{Path: tw.root, IsDir: true})
			continue
		}
		fc := FileChange{Path: path}
		fi, err := os.Stat(path)
		switch {
		case err == nil:
			fc.IsDir = fi.IsDir()
		case errors.Is(err, fs.ErrNotExist):
			fc.Removed = true
		default:
			continue
		}
		changes = append(changes, fc)
		ft.nodeToUpdate(fc, update)
	}
	ft.updateNodes(update)
	ft.syncWatchDirs()
	if ft.WatchFunc != nil && len(changes) > 0 {
		ft.WatchFunc(changes)
	}
	return true
}

// nodeToUpdate adds the node that needs to be updated for the
// given change to update: the parent directory if a node must be
// added or removed, or the node of a modified file.
func (ft *Tree) nodeToUpdate(fc FileChange, update map[*Node]bool) {
	if fc.Path == string(ft.Filepath) {
		return
	}
	dn := ft.dirNode(filepath.Dir(fc.Path))
	if dn == nil || !ft.isDirOpen(dn.Filepath) {
		return
	}
	cn := dn.ChildByName(filepath.Base(fc.Path), 0)
	if cn == nil {
		if !fc.Removed {
			update[dn] = true
		}
		return
	}
	n := AsNode(cn)
	if fc.Removed || n.IsDir() != fc.IsDir {
		update[dn] = true
		return
	}
	if !n.IsDir() {
		update[n] = true
	}
}

// updateNodes updates the given nodes, skipping those
// within a directory that is updated, as that updates them too.
func (ft *Tree) updateNodes(update map[*Node]bool) {
	nodes := slices.SortedFunc(maps.Keys(update), func(a, b *Node) int {
		return cmp.Compare(len(a.Filepath), len(b.Filepath))
	})
	var dirs []string
	for _, n := range nodes {
		if n.This == nil {
			continue
		}
		path := string(n.Filepath)
		if slices.ContainsFunc(dirs, func(dir string) bool {
			return strings.HasPrefix(path, dir+string(filepath.Separator))
		}) {
			continue
		}
		if n.IsDir() {
			dirs = append(dirs, path)
		}
		n.Update()
	}
	if len(dirs) > 0 {
		ft.NeedsLayout()
	}
}

// dirNode returns the node for the given directory path
// if it is in the tree, or nil otherwise.
func (ft *Tree) dirNode(path string) *Node {
	rp, err := filepath.Rel(string(ft.Filepath), path)
	if err != nil || rp == ".." || strings.HasPrefix(rp, ".."+string(filepath.Separator)) {
		return nil
	}
	n := &ft.Node
	if rp == "." {
		return n
	}
	for _, name := range strings.Split(rp, string(filepath.Separator)) {
		c := n.ChildByName(name, 0)
		if c == nil {
			return nil
		}
		n = AsNode(c)
		if !n.IsDir() {
			return nil
		}
	}
	return n
}
