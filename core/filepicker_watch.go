// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !ios

package core

import (
	"context"
	"log/slog"
	"time"

	"github.com/sgtdi/fswatcher"
)

// watchFilePickerDir watches the files directly within the given
// directory until ctx is canceled, calling changed from another
// goroutine after changes, which are collected for 200ms.
func watchFilePickerDir(ctx context.Context, dir string, changed func()) error {
	w, err := fswatcher.New(fswatcher.WithPath(dir, fswatcher.WithDepth(fswatcher.WatchTopLevel)))
	if err != nil {
		return err
	}
	go func() {
		if err := w.Watch(ctx); err != nil {
			slog.Error("FilePicker: file watcher failed", "path", dir, "err", err)
		}
	}()
	go func() {
		timer := time.NewTimer(time.Hour)
		timer.Stop()
		pending := false
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-w.Events():
				if !ok {
					return
				}
				if !pending {
					pending = true
					timer.Reset(200 * time.Millisecond)
				}
			case <-timer.C:
				pending = false
				changed()
			}
		}
	}()
	return nil
}
