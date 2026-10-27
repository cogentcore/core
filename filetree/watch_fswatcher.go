// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !ios

package filetree

import (
	"context"
	"log/slog"
	"slices"
	"sync"

	"github.com/sgtdi/fswatcher"
)

// watchDirs starts watching the files directly within the given
// directories, until ctx is canceled. ready is closed when watching
// has started. Version control directories are not watched.
func watchDirs(ctx context.Context, dirs []string, ready chan struct{}) (*watchBackend, error) {
	opts := []fswatcher.WatcherOpt{
		fswatcher.WithExcRegex(`[/\\]\.git([/\\]|$)`),
		fswatcher.WithReadyChannel(ready),
	}
	for _, d := range dirs {
		opts = append(opts, fswatcher.WithPath(d, fswatcher.WithDepth(fswatcher.WatchTopLevel)))
	}
	w, err := fswatcher.New(opts...)
	if err != nil {
		return nil, err
	}
	go func() {
		if err := w.Watch(ctx); err != nil {
			slog.Error("filetree: file watcher failed", "err", err)
		}
	}()

	paths := make(chan string, 256)
	go func() {
		defer close(paths)
		for ev := range w.Events() {
			path := ev.Path
			if slices.Contains(ev.Types, fswatcher.EventOverflow) {
				path = ""
			}
			select {
			case paths <- path:
			case <-ctx.Done():
				return
			}
		}
	}()

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
			push(func() { w.AddPath(dir, fswatcher.WithDepth(fswatcher.WatchTopLevel)) })
		},
		drop: func(dir string) {
			push(func() { w.DropPath(dir) })
		},
		busy: func() bool {
			mu.Lock()
			defer mu.Unlock()
			return running || len(ops) > 0
		},
	}, nil
}
