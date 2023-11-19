// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package desktop

// TODO(kai): implement IsDark and SetTitleBarIsDark on linux

// IsDark returns whether the system color theme is dark (as opposed to light).
func (app *appImpl) IsDark() bool {
	return false
}

func (w *windowImpl) SetTitleBarIsDark(isDark bool) {
	// no-op
}
