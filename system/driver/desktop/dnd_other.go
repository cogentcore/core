// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !darwin

package desktop

// initDrag sets up drag-and-drop for the window.
// Other than file drops (see [Window.DropEvent]), it is not
// yet supported on this platform.
func (w *Window) initDrag() {}

// closeDrag removes the window from drag-and-drop.
func (w *Window) closeDrag() {}
