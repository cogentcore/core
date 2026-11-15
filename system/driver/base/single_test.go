// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
	"github.com/stretchr/testify/assert"
)

// singleApp is a single-window app, as on mobile and web platforms.
type singleApp struct {
	AppSingle[*composer.ComposerDrawer, *singleWindow]
	platform system.Platforms
}

func (a *singleApp) Platform() system.Platforms { return a.platform }
func (a *singleApp) DataDir() string            { return "" }

func (a *singleApp) NewWindow(opts *system.NewWindowOptions) (system.Window, error) {
	w := &singleWindow{NewWindowSingle(a, opts)}
	w.This = w
	a.Win = w
	go w.WinLoop()
	return w, nil
}

// singleWindow is the window of a [singleApp].
type singleWindow struct {
	WindowSingle[*singleApp]
}

func newSingleApp(platform system.Platforms) *singleApp {
	a := &singleApp{AppSingle: NewAppSingle[*composer.ComposerDrawer, *singleWindow](), platform: platform}
	a.This = a
	a.Compose = &composer.ComposerDrawer{}
	a.Scrn.PixelSize = image.Pt(400, 300)
	a.Scrn.PhysicalDPI = 100
	a.Scrn.LogicalDPI = 160
	return a
}

func TestAppSingleWindow(t *testing.T) {
	a := newSingleApp(system.Web)
	assert.Equal(t, 0, a.NWindows(), "no window yet")
	assert.Nil(t, a.Window(0))
	assert.Nil(t, a.ContextWindow())
	assert.Nil(t, a.WindowByName("main"))
	assert.Nil(t, a.WindowInFocus())

	assert.Equal(t, 1, a.NScreens())
	assert.Same(t, a.Scrn, a.Screen(5), "the one screen")
	assert.Same(t, a.Scrn, a.ScreenByName("main"))
	assert.Nil(t, a.ScreenByName("other"))

	sw, err := a.NewWindow(&system.NewWindowOptions{Title: "single"})
	assert.NoError(t, err)
	w := sw.(*singleWindow)
	w.SetName("main")
	assert.Equal(t, 1, a.NWindows())
	assert.Equal(t, sw, a.Window(0))
	assert.Nil(t, a.Window(1))
	assert.Equal(t, sw, a.WindowByName("main"))
	assert.Equal(t, sw, a.ContextWindow())
	assert.Nil(t, a.WindowInFocus())
	w.Flgs.SetFlag(true, system.Focused)
	assert.Equal(t, sw, a.WindowInFocus())

	assert.Same(t, a.Events(), w.Events())
	assert.Equal(t, composer.Composer(a.Compose), w.Composer())
	assert.Equal(t, image.Pt(400, 300), w.Size())
	assert.Equal(t, image.Pt(400, 300), w.WinSize())
	assert.Equal(t, image.Point{}, w.Position(nil))
	assert.Equal(t, float32(100), w.PhysicalDPI())
	w.SetLogicalDPI(192)
	assert.Equal(t, float32(192), w.LogicalDPI())
	assert.Equal(t, sides.Sides[int]{}, w.ConstrainFrame(false))

	w.SetSize(image.Pt(10, 10))
	assert.Equal(t, image.Pt(400, 300), w.Size(), "web is mobile: the size is that of the screen")
	w.SetWinSize(image.Pt(500, 350))
	assert.Equal(t, image.Pt(500, 350), w.Size())
	w.SetGeometry(true, image.Point{}, image.Point{}, nil)
	assert.True(t, w.Is(system.Fullscreen))
	w.SetPos(image.Pt(5, 5), nil) // no-op

	a.Insets = sides.Sides[int]{Top: 10, Bottom: 20, Left: 5}
	g := w.RenderGeom()
	assert.Equal(t, image.Pt(5, 10), g.Pos)
	assert.Equal(t, image.Pt(500-5, 350-10-20), g.Size)

	cleaned, closed := false, false
	a.AddQuitCleanFunc(func() { cleaned = true })
	w.SetCloseCleanFunc(func(win system.Window) { closed = true })
	// note: the window of a single-window app is not closed by closing it
	// (its composer stays), so QuitClean returns false here
	a.QuitClean()
	assert.True(t, cleaned)
	assert.True(t, closed)
	assert.True(t, a.IsQuitting())
}

func TestAppSingleDesktopSize(t *testing.T) {
	a := newSingleApp(system.Offscreen)
	sw, _ := a.NewWindow(&system.NewWindowOptions{})
	sw.SetSize(image.Pt(320, 200))
	assert.Equal(t, image.Pt(320, 200), sw.Size(), "settable when not mobile")
	sw.Close()
	sw.SetSize(image.Pt(1, 1))
}

func TestPaintState(t *testing.T) {
	ps := NewPaintState()
	assert.False(t, ps.IsIdle())
	ps.Wake() // not idle: nothing to wake
	select {
	case <-ps.Woken():
		t.Fatal("woken without being idle")
	default:
	}
	ps.SetIdle()
	assert.True(t, ps.IsIdle())
	ps.Wake()
	ps.Wake() // only one wake is queued
	assert.False(t, ps.IsIdle())
	<-ps.Woken()
	select {
	case <-ps.Woken():
		t.Fatal("woken twice")
	default:
	}
}

func TestWindowPaintLoop(t *testing.T) {
	a := newSingleApp(system.Web)
	sw, _ := a.NewWindow(&system.NewWindowOptions{})
	w := sw.(*singleWindow)
	// nextPaint waits for the next paint event, failing after a second.
	nextPaint := func(msg string) {
		got := make(chan bool)
		go func() {
			for {
				if w.Events().Deque.NextEvent().Type() == events.WindowPaint {
					got <- true
					return
				}
			}
		}()
		select {
		case <-got:
		case <-time.After(time.Second):
			t.Fatal(msg)
		}
	}
	nextPaint("painted while active")
	w.PaintIdle()
	assert.True(t, w.PaintIsIdle())
	w.PaintWake()
	assert.False(t, w.PaintIsIdle())
	nextPaint("painted when woken")
	w.SendPaintEvent()
	nextPaint("sent")
	w.Close()
}
