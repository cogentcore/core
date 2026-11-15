// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"image"
	"os/user"
	"path/filepath"
	"testing"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/system"
	"github.com/stretchr/testify/assert"
)

// displayAvailable returns whether there is a display, which there always
// is on Windows (except in a service: set CORE_NO_GUI_TESTS there).
func displayAvailable() bool { return true }

func TestPlatformWindows(t *testing.T) {
	assert.Equal(t, system.Windows, TheApp.Platform())
	usr, err := user.Current()
	if assert.NoError(t, err) {
		assert.Equal(t, filepath.Join(usr.HomeDir, "AppData", "Roaming"), TheApp.DataDir())
	}
}

// TestTitleBarDark tests setting the dark mode of the title bar of a
// window (with DwmSetWindowAttribute), which must not fail on Windows 10
// or 11.
func TestTitleBarDark(t *testing.T) {
	requireGUI(t)
	gpu.SkipIfNoGPU(t)
	sw, err := TheApp.NewWindow(&system.NewWindowOptions{Title: "title bar", Size: image.Pt(200, 100)})
	if err != nil {
		t.Fatal(err)
	}
	w := sw.(*Window)
	defer w.Close()
	assert.NotPanics(t, func() {
		w.SetTitleBarIsDark(true)
		w.SetTitleBarIsDark(false)
	})
}
