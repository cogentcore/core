// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !(android || ios || js)

package desktop

import (
	"image"
	"os"
	"runtime"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/gpu"
	_ "cogentcore.org/core/paint/renderers" // for the svgs of cursors, as in an app
	"cogentcore.org/core/system"
	_ "cogentcore.org/core/text/shaped/shapers" // (as are imported by core)
	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// guiRunning is whether the windowing system runs for the tests, which
// needs a display (see [displayAvailable]). Set CORE_NO_GUI_TESTS to
// skip the tests that need it, as when running over ssh.
var guiRunning bool

// TestMain runs the main loop of the app on the main thread, as the
// windowing system requires, and the tests in another goroutine.
func TestMain(m *testing.M) {
	if os.Getenv("CORE_NO_GUI_TESTS") != "" || !displayAvailable() {
		os.Exit(m.Run())
	}
	Init() // locks the main thread
	guiRunning = true
	code := 0
	go func() {
		code = m.Run()
		TheApp.StopMain()
	}()
	TheApp.MainLoop()
	os.Exit(code)
}

// requireGUI skips the test if the windowing system is not running,
// or it has no screens (monitors), as with an X display without RandR
// outputs.
func requireGUI(t *testing.T) {
	t.Helper()
	if !guiRunning {
		t.Skip("no display for the windowing system (or CORE_NO_GUI_TESTS is set)")
	}
	if TheApp.NScreens() == 0 {
		t.Skip("the windowing system has no screens")
	}
}

// TestNewWindowNoScreens tests that opening a window without screens is
// an error, not a panic.
func TestNewWindowNoScreens(t *testing.T) {
	if !guiRunning {
		t.Skip("no display for the windowing system")
	}
	a := &App{}
	a.Screens = nil
	a.guiDone.Store(true) // the GUI is initialized (by TheApp)
	_, err := a.NewWindow(nil)
	assert.Error(t, err)
}

func TestGlfwMods(t *testing.T) {
	assert.Equal(t, key.Modifiers(0), GlfwMods(0))
	m := GlfwMods(glfw.ModShift | glfw.ModSuper | glfw.ModCapsLock)
	assert.True(t, m.HasFlag(key.Shift))
	assert.True(t, m.HasFlag(key.Meta))
	assert.True(t, m.HasFlag(key.CapsLock))
	assert.False(t, m.HasFlag(key.Control))
	m = GlfwMods(glfw.ModControl | glfw.ModAlt | glfw.ModNumLock)
	assert.True(t, m.HasFlag(key.Control))
	assert.True(t, m.HasFlag(key.Alt))
	assert.True(t, m.HasFlag(key.NumLock))
}

func TestGlfwKeyCode(t *testing.T) {
	codes := map[glfw.Key]key.Codes{
		glfw.KeyA:         key.CodeA,
		glfw.KeyZ:         key.CodeZ,
		glfw.Key0:         key.Code0,
		glfw.Key9:         key.Code9,
		glfw.KeyEnter:     key.CodeReturnEnter,
		glfw.KeyEscape:    key.CodeEscape,
		glfw.KeyBackspace: key.CodeBackspace,
		glfw.KeyTab:       key.CodeTab,
		glfw.KeySpace:     key.CodeSpacebar,
		glfw.KeyLeft:      key.CodeLeftArrow,
		glfw.KeyRight:     key.CodeRightArrow,
		glfw.KeyUp:        key.CodeUpArrow,
		glfw.KeyDown:      key.CodeDownArrow,
		glfw.KeyHome:      key.CodeHome,
		glfw.KeyEnd:       key.CodeEnd,
		glfw.KeyPageUp:    key.CodePageUp,
		glfw.KeyPageDown:  key.CodePageDown,
		glfw.KeyDelete:    key.CodeDelete,
		glfw.KeyF1:        key.CodeF1,
		glfw.KeyF12:       key.CodeF12,
		glfw.KeyLeftShift: key.CodeLeftShift,
		glfw.KeyKP5:       key.CodeKeypad5,
	}
	for gk, want := range codes {
		assert.Equal(t, want, GlfwKeyCode(gk), "glfw key %d", gk)
	}
	assert.Equal(t, key.CodeUnknown, GlfwKeyCode(glfw.KeyUnknown))
}

func TestScreenInfo(t *testing.T) {
	a := &App{}
	sc := &system.Screen{Name: "test screen", DevicePixelRatio: 2}
	assert.True(t, a.SaveScreenInfo(sc))
	assert.False(t, a.SaveScreenInfo(sc), "saved once by name")
	got, ok := a.FindScreenInfo("test screen")
	require.True(t, ok)
	assert.Equal(t, float32(2), got.DevicePixelRatio)
	assert.NotSame(t, sc, got, "a copy")
	_, ok = a.FindScreenInfo("other")
	assert.False(t, ok)
}

func TestAppScreens(t *testing.T) {
	requireGUI(t)
	require.Greater(t, TheApp.NScreens(), 0)
	sc := TheApp.Screen(0)
	assert.NotEmpty(t, sc.Name)
	assert.Same(t, sc, TheApp.ScreenByName(sc.Name))
	assert.Greater(t, sc.PixelSize.X, 0)
	assert.Greater(t, sc.PixelSize.Y, 0)
	assert.Greater(t, sc.DevicePixelRatio, float32(0))
	assert.Greater(t, sc.LogicalDPI, float32(0))
	// (the main thread is known on these platforms; the screens being found
	// here depends on it, as other goroutines wait for the initialization)
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		assert.False(t, onMainThread(), "the tests run off the main thread")
	}
	assert.NotPanics(t, func() { TheApp.IsDark() })
}

func TestNewWindow(t *testing.T) {
	requireGUI(t)
	gpu.SkipIfNoGPU(t)
	sw, err := TheApp.NewWindow(&system.NewWindowOptions{Title: "desktop test", Size: image.Pt(320, 240)})
	require.NoError(t, err)
	w := sw.(*Window)
	assert.Equal(t, 1, TheApp.NWindows())
	assert.Equal(t, "desktop test", w.Title())
	w.SetTitle("renamed")
	assert.Equal(t, "renamed", w.Title())
	assert.False(t, w.IsClosed())
	assert.NotNil(t, w.Screen())
	sz := w.Size()
	assert.Greater(t, sz.X, 0)
	assert.Greater(t, sz.Y, 0)
	assert.Equal(t, sz, w.Screen().WindowSizeToPixels(w.WinSize()), "pixel size = window size * device pixel ratio")

	painted := waitEvent(w, func(e events.Event) bool { return e.Type() == events.WindowPaint })
	assert.True(t, painted, "the window loop sends paint events")

	nsz := w.Screen().WindowSizeToPixels(image.Pt(400, 300))
	w.SetSize(nsz)
	resized := waitEvent(w, func(e events.Event) bool { return e.Type() == events.WindowResize })
	assert.True(t, resized, "a resize event")
	// (on X11, a resize event can come before the size of the window is set)
	assert.Eventually(t, func() bool { return w.WinSize() == image.Pt(400, 300) }, 2*time.Second, 10*time.Millisecond, "the new size: %v", w.WinSize())
	assert.Equal(t, nsz, w.Size())

	assert.NoError(t, TheApp.Cursor(w).Set(cursors.Text)) // sets the context window
	assert.Equal(t, cursors.Text, TheCursor.Current())

	closed := false
	w.SetCloseCleanFunc(func(win system.Window) { closed = true })
	w.Close()
	assert.True(t, closed)
	assert.True(t, w.IsClosed())
	assert.False(t, w.Lock(), "a closed window cannot be locked")
	assert.Equal(t, 0, TheApp.NWindows())
	w.SetTitle("after close") // no-op
}

// waitEvent waits up to two seconds for an event of the window that
// matches, returning whether there was one.
func waitEvent(w *Window, match func(e events.Event) bool) bool {
	done := make(chan bool, 1)
	go func() {
		for {
			e, ok := w.Events().Deque.PollEvent()
			if !ok {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			if match(e) {
				done <- true
				return
			}
		}
	}()
	select {
	case <-done:
		return true
	case <-time.After(2 * time.Second):
		return false
	}
}

// TestClipboard writes to and reads from the real system clipboard,
// restoring its text after. It is skipped if the clipboard may have data
// that is not text, which could not be restored, except on the CI (where
// the CI environment variable is set), whose clipboard is not a user's.
// (On Linux, IsEmpty is always false, so that is always the case.)
func TestClipboard(t *testing.T) {
	requireGUI(t)
	cb := TheClipboard
	prev := cb.Read([]string{mimedata.TextPlain})
	if prev == nil && !cb.IsEmpty() && os.Getenv("CI") == "" {
		t.Skip("the clipboard may have data that is not text, which could not be restored")
	}
	defer func() {
		if prev != nil {
			cb.Write(prev)
		} else {
			cb.Clear()
		}
	}()

	require.NoError(t, cb.Write(mimedata.NewText("cogent core clipboard test")))
	assert.False(t, cb.IsEmpty())
	got := cb.Read([]string{mimedata.TextPlain})
	assert.Equal(t, "cogent core clipboard test", got.Text(mimedata.TextPlain))

	md := mimedata.NewTextPlus("path/to/node", mimedata.DataJson, []byte(`{"a":1}`))
	require.NoError(t, cb.Write(md))
	got = cb.Read([]string{mimedata.DataJson})
	require.Len(t, got, 2, "multiple items go through multipart text")
	assert.Equal(t, `{"a":1}`, string(got.TypeData(mimedata.DataJson)))
}
