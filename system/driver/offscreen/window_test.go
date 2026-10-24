// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"image"
	"sync"
	"testing"

	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/system/driver/base"
)

// The geometry of a window can be set and read by several goroutines at
// once, including while the window is locked, as it is during rendering.
// This test is for the race detector.
func TestWindowGeometryConcurrent(t *testing.T) {
	TheApp.GetScreens()
	w := &Window{base.NewWindowMulti[*App, *composer.ComposerDrawer](TheApp, &system.NewWindowOptions{Size: image.Pt(300, 200)})}
	w.This = w
	w.Compose = &composer.ComposerDrawer{Drawer: &Drawer{Window: w}}
	w.PixelSize = image.Pt(300, 200)

	// The window is locked as during rendering, which reads the geometry.
	if !w.Lock() {
		t.Fatal("window is closed")
	}
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Go(func() {
			for j := range 100 {
				w.SetSize(image.Pt(100+i, 100+j))
				w.SetPos(image.Pt(i, j), nil)
				w.SetGeometry(false, image.Pt(j, i), image.Pt(200+j, 200+i), nil)
				_ = w.WinSize()
				_ = w.Position(nil)
				_ = w.RenderGeom()
			}
		})
	}
	wg.Go(func() {
		for range 100 {
			w.Compose.Drawer.Start()
		}
	})
	wg.Wait()
	w.Unlock()

	if sz := w.Size(); sz != image.Pt(300, 200) {
		t.Errorf("pixel size %v, want %v", sz, image.Pt(300, 200))
	}
}
