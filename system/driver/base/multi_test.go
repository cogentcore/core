// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base_test

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/offscreen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	offscreen.Init()
	os.Exit(m.Run())
}

// newWindow makes a new offscreen window with the given name and size.
func newWindow(t *testing.T, name string, sz image.Point) system.Window {
	w, err := system.TheApp.NewWindow(&system.NewWindowOptions{Title: name + " title", Size: sz})
	require.NoError(t, err)
	w.SetName(name)
	return w
}

func TestAppMultiWindows(t *testing.T) {
	a := offscreen.TheApp
	a1 := newWindow(t, "a", image.Pt(300, 200))
	b := newWindow(t, "b", image.Pt(200, 100))
	assert.Equal(t, 2, a.NWindows())
	assert.Equal(t, a1, a.Window(0))
	assert.Equal(t, b, a.Window(1))
	assert.Nil(t, a.Window(2))
	assert.Nil(t, a.Window(-1))
	assert.Equal(t, b, a.WindowByName("b"))
	assert.Nil(t, a.WindowByName("c"))
	assert.Nil(t, a.ContextWindow())

	assert.Equal(t, 1, a.NScreens())
	assert.Same(t, a.Screen(0), a.Screen(3), "the first screen for an invalid index")
	assert.Nil(t, a.ScreenByName("none"))

	assert.Equal(t, "b title", b.Title())
	b.SetTitle("new")
	assert.Equal(t, "new", b.Title())
	assert.Equal(t, image.Pt(200, 100), b.Size())
	assert.True(t, b.IsVisible())
	assert.False(t, b.IsClosed())
	assert.True(t, b.IsCursorEnabled())
	b.SetCursorEnabled(false, false)
	assert.False(t, b.IsCursorEnabled())
	assert.False(t, b.StartDrag(nil, nil, nil))
	assert.True(t, b.DragAccepted())
	assert.Equal(t, image.Pt(200, 100), b.RenderGeom().Size)
	b.SetLogicalDPI(192)
	assert.Equal(t, float32(192), b.LogicalDPI())

	b.SetGeometry(false, image.Pt(5, 6), image.Pt(120, 80), nil)
	assert.Equal(t, image.Pt(5, 6), b.Position(nil))
	assert.Equal(t, image.Pt(120, 80), b.WinSize())

	var closed []string
	b.SetCloseCleanFunc(func(w system.Window) { closed = append(closed, w.Name()) })
	b.Close()
	assert.Equal(t, []string{"b"}, closed)
	assert.Equal(t, 1, a.NWindows())
	assert.True(t, b.IsClosed() || a.WindowByName("b") == nil)
	a1.Close()
	assert.Equal(t, 0, a.NWindows())
}

func TestAppMultiCloseReq(t *testing.T) {
	w := newWindow(t, "req", image.Pt(100, 100))
	asked := 0
	w.SetCloseReqFunc(func(win system.Window) { asked++ })
	w.CloseReq()
	assert.Equal(t, 1, asked, "the request function decides")
	assert.Equal(t, 1, system.TheApp.NWindows())
	w.SetCloseReqFunc(nil)
	w.CloseReq()
	assert.Equal(t, 0, system.TheApp.NWindows(), "closed without a request function")
}

func TestAppQuit(t *testing.T) {
	a := offscreen.TheApp
	newWindow(t, "q1", image.Pt(100, 100))
	newWindow(t, "q2", image.Pt(100, 100))
	cleaned := 0
	a.AddQuitCleanFunc(func() { cleaned++ })
	defer func() { a.QuitCleanFuncs = nil }()
	assert.True(t, a.AppMulti.QuitClean(), "all of the windows are closed")
	assert.Equal(t, 1, cleaned)
	assert.Equal(t, 0, a.NWindows())
}

func TestAppQuitReq(t *testing.T) {
	a := offscreen.TheApp
	reqs := 0
	a.SetQuitReqFunc(func() { reqs++ })
	defer a.SetQuitReqFunc(nil)
	a.QuitReq()
	assert.Equal(t, 1, reqs)
	assert.False(t, a.IsQuitting())
}

func TestAppInfo(t *testing.T) {
	a := offscreen.TheApp
	a.SetName("testapp")
	assert.Equal(t, "testapp", a.Name())
	assert.Equal(t, system.Offscreen, a.SystemPlatform())
	assert.Equal(t, "", a.SystemInfo())
	assert.NotPanics(t, func() { a.SystemLocale() })
	assert.Empty(t, a.OpenFiles())
	assert.False(t, a.IsDark())
	assert.Equal(t, filepath.Join(a.DataDir(), "testapp"), a.AppDataDir())
	assert.DirExists(t, a.AppDataDir())
	assert.Equal(t, filepath.Join(a.DataDir(), "Cogent Core"), a.CogentCoreDataDir())
	assert.NotNil(t, a.Cursor(nil))
	a.BeginBackgroundTask("task")()
	a.OpenURL("https://example.com") // no-op
}

func TestRunOnMain(t *testing.T) {
	a := offscreen.TheApp
	ran := false
	a.RunOnMain(func() { ran = true })
	assert.True(t, ran, "run at once without a main loop")

}

// TestAppCloseRemovesFirst tests that a closed window is removed from the
// app before its WinClose event is sent: the event stops the event loop
// of the window in core, and the app quits once the loop of its last
// window stops, which fails if the window is still in the app then
// (leaving a process with no windows that never quits).
func TestAppCloseRemovesFirst(t *testing.T) {
	w := newWindow(t, "last", image.Pt(100, 100))
	nwin := make(chan int, 1)
	go func() {
		for {
			e := w.Events().Deque.NextEvent()
			if we, ok := e.(*events.WindowEvent); ok && we.Action == events.WinClose {
				nwin <- system.TheApp.NWindows()
				return
			}
		}
	}()
	w.Close()
	assert.Equal(t, 0, <-nwin, "the windows of the app when it gets WinClose")
}
