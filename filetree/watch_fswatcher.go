// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !ios

package filetree

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"sync"

	"github.com/sgtdi/fswatcher"
)

// fsWatcher is a running [fswatcher.Watcher], which sends the changed
// paths to the events of its [watchBackend] until it is canceled.
type fsWatcher struct {
	w      fswatcher.Watcher
	cancel context.CancelFunc
}

// newFSWatcher starts an [fswatcher.Watcher] of the files directly within
// the given directories, which sends the changed paths to the given
// channel until ctx is canceled. ready is closed when watching has started.
func newFSWatcher(ctx context.Context, dirs []string, paths chan<- string, ready chan struct{}) (*fsWatcher, error) {
	opts := []fswatcher.WatcherOpt{
		fswatcher.WithReadyChannel(ready),
	}
	for _, d := range dirs {
		opts = append(opts, fswatcher.WithPath(d, fswatcher.WithDepth(fswatcher.WatchTopLevel)))
	}
	w, err := fswatcher.New(opts...)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		if err := w.Watch(ctx); err != nil {
			slog.Error("filetree: file watcher failed", "err", err)
		}
	}()
	go func() {
		for {
			select {
			case ev, ok := <-w.Events():
				if !ok {
					return
				}
				path := ev.Path
				if slices.Contains(ev.Types, fswatcher.EventOverflow) {
					path = ""
				}
				select {
				case paths <- path:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return &fsWatcher{w: w, cancel: cancel}, nil
}

// watchDirs starts watching the files directly within the given
// directories, until ctx is canceled. ready is closed when watching
// has started. The changes in version control directories are
// filtered by [Tree.applyChanges], as it watches git directories.
func watchDirs(ctx context.Context, dirs []string, ready chan struct{}) (*watchBackend, error) {
	paths := make(chan string, 256)
	fw, err := newFSWatcher(ctx, dirs, paths, ready)
	if err != nil {
		return nil, err
	}
	// the directories being watched, only used by the operations
	current := make(map[string]bool)
	for _, d := range dirs {
		current[d] = true
	}

	// restart replaces the watcher with a new one of the current
	// directories, as fswatcher keeps watching a directory that it could
	// not drop (which happens after the directory is renamed or removed,
	// on Linux), and then drops all of the events of a new directory with
	// its path (as a directory within it, which is not watched). The events
	// can be lost while it restarts, so it sends that they were lost.
	restart := func() {
		fw.cancel()
		nready := make(chan struct{})
		nfw, err := newFSWatcher(ctx, slices.Sorted(maps.Keys(current)), paths, nready)
		if err != nil {
			slog.Error("filetree: could not restart the file watcher", "err", err)
			return
		}
		fw = nfw
		select {
		case <-nready:
		case <-ctx.Done():
			return
		}
		select {
		case paths <- "":
		case <-ctx.Done():
		}
	}

	// Directories can only be added and dropped while the watcher is
	// running, so the operations are queued and run in order once it is.
	var mu sync.Mutex
	var ops []func()
	running := false
	signal := make(chan struct{}, 1)
	push := func(op func()) {
		mu.Lock()
		ops = append(ops, op)
		mu.Unlock()
		select {
		case signal <- struct{}{}:
		default:
		}
	}
	go func() {
		select {
		case <-ready:
		case <-ctx.Done():
			return
		}
		for {
			select {
			case <-signal:
			case <-ctx.Done():
				return
			}
			mu.Lock()
			do := ops
			ops = nil
			running = len(do) > 0
			mu.Unlock()
			for _, op := range do {
				op()
			}
			mu.Lock()
			running = false
			mu.Unlock()
		}
	}()
	return &watchBackend{
		events: paths,
		// errors are expected for directories that were just deleted
		add: func(dir string) {
			push(func() {
				current[dir] = true
				fw.w.AddPath(dir, fswatcher.WithDepth(fswatcher.WatchTopLevel))
			})
		},
		drop: func(dir string) {
			push(func() {
				delete(current, dir)
				if fw.w.DropPath(dir) != nil {
					restart()
				}
			})
		},
		busy: func() bool {
			mu.Lock()
			defer mu.Unlock()
			return running || len(ops) > 0
		},
	}, nil
}
