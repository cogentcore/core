// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !darwin

package desktop

// initIME sets up input methods (IME) for the window,
// which are not supported on this platform yet.
func (w *Window) initIME() {}
