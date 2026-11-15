// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system_test

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/offscreen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	offscreen.Init() // the app is the offscreen app, with an 800x600 screen
	os.Exit(m.Run())
}

// closeWindows closes all of the windows of the app.
func closeWindows() {
	for system.TheApp.NWindows() > 0 {
		system.TheApp.Window(0).Close()
	}
}

func TestFixupNewWindow(t *testing.T) {
	defer closeWindows()
	o := &system.NewWindowOptions{}
	o.Fixup()
	assert.Equal(t, image.Pt(640, 480), o.Size, "80% of the screen")
	assert.Equal(t, image.Pt(80, 60), o.Pos, "centered in the screen")

	o = &system.NewWindowOptions{Size: image.Pt(2000, 100), Pos: image.Pt(5, 5)}
	o.Fixup()
	assert.Equal(t, image.Pt(800, 100), o.Size, "constrained to the screen")
	assert.Equal(t, image.Pt(0, 5), o.Pos)

	o = &system.NewWindowOptions{Flags: 1 << system.Fullscreen}
	o.Fixup()
	assert.Equal(t, image.Pt(800, 600), o.Size)
	assert.Equal(t, image.Point{}, o.Pos)
}

func TestFixupPlacement(t *testing.T) {
	defer closeWindows()
	first := &system.NewWindowOptions{Size: image.Pt(400, 300), Pos: image.Pt(10, 10)}
	first.Fixup()
	w, err := system.TheApp.NewWindow(first)
	require.NoError(t, err)
	w.SetGeometry(false, image.Pt(10, 10), image.Pt(400, 300), nil)

	dialog := &system.NewWindowOptions{Size: image.Pt(200, 100)}
	dialog.Flags.SetFlag(true, system.Dialog)
	dialog.Fixup()
	assert.Equal(t, image.Pt(10+200-100, 10+150-50), dialog.Pos, "centered on the last window")

	big := &system.NewWindowOptions{Size: image.Pt(500, 400)}
	big.Fixup()
	assert.Equal(t, image.Pt(300, 10+72), big.Pos, "cascaded to the right and down, constrained to the screen")
}

func TestScreenConstrainWindowGeometry(t *testing.T) {
	sc := system.TheApp.Screen(0)
	pos, sz := sc.ConstrainWindowGeometry(image.Pt(700, -5), image.Pt(300, 200))
	assert.Equal(t, image.Pt(500, 0), pos)
	assert.Equal(t, image.Pt(300, 200), sz)
}

func TestCrashLog(t *testing.T) {
	txt := system.CrashLogText("boom", "the stack")
	assert.Contains(t, txt, "Platform: Offscreen")
	assert.Contains(t, txt, "panic: boom")
	assert.True(t, strings.HasSuffix(txt, "the stack"))

	system.HandleRecoverBase(nil) // nothing
	system.HandleRecoverBase("boom")
	files, err := filepath.Glob(filepath.Join(system.TheApp.AppDataDir(), "crash-logs", "crash_*"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "a crash log is saved")
	b, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Contains(t, string(b), "panic: boom")

	system.HandleRecoverPanic(nil) // nothing
	assert.PanicsWithValue(t, "boom", func() { system.HandleRecoverPanic("boom") }, "panics on desktop platforms")
}
