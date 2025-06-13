// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// originally based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"image"
	"log"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/gpudraw"
	"cogentcore.org/core/styles/sides"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/system/driver/base"
	"github.com/go-gl/glfw/v3.3/glfw"
)

// Window is the implementation of [system.Window] for the desktop platform.
type Window struct {
	base.WindowMulti[*App, *composer.ComposerDrawer]

	// Glw is the glfw window associated with this window
	Glw *glfw.Window

	// Draw is the [gpudraw.Drawer] used for the Composer.
	Draw *gpudraw.Drawer // TODO: really need this separately?

	// ScreenName is the name of the last known screen this window was on
	ScreenWindow string
}

func (w *Window) IsVisible() bool {
	return w.WindowMulti.IsVisible() && w.Glw != nil
}

func (w *Window) SendPaintEvent() {
	TheApp.PollScreenChanges()
	w.This.Events().WindowPaint()
}

// Activate() sets this window as the current render target for gpu rendering
// functions, and the current context for gpu state (equivalent to
// MakeCurrentContext on OpenGL).
// If it returns false, then window is not visible / valid and
// nothing further should happen.
// Must call this on app main thread using system.TheApp.RunOnMain
//
//	system.TheApp.RunOnMain(func() {
//	   if !win.Activate() {
//	       return
//	   }
//	   // do GPU calls here
//	})
func (w *Window) Activate() bool {
	// note: activate is only run on main thread so we don't need to check for mutex
	if w == nil || w.Glw == nil {
		return false
	}
	w.Glw.MakeContextCurrent()
	return true
}

// DeActivate() clears the current render target and gpu rendering context.
// Generally more efficient to NOT call this and just be sure to call
// Activate where relevant, so that if the window is already current context
// no switching is required.
// Must call this on app main thread using system.TheApp.RunOnMain
func (w *Window) DeActivate() {
	glfw.DetachCurrentContext()
}

// newGlfwWindow makes a new glfw window for this window.
// It must be run on main.
func (w *Window) newGlfwWindow(opts *system.NewWindowOptions, sc *system.Screen) error {
	// glfw.DefaultWindowHints()
	if opts.Flags.HasFlag(system.FixedSize) {
		glfw.WindowHint(glfw.Resizable, glfw.False)
	} else {
		glfw.WindowHint(glfw.Resizable, glfw.True)
	}
	glfw.WindowHint(glfw.Visible, glfw.False) // needed to position
	glfw.WindowHint(glfw.Focused, glfw.True)
	// glfw.WindowHint(glfw.ScaleToMonitor, glfw.True)
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	// glfw.WindowHint(glfw.Samples, 0) // don't do multisampling for main window -- only in sub-render
	if opts.Flags.HasFlag(system.Maximized) {
		glfw.WindowHint(glfw.Maximized, glfw.True)
	}
	if opts.Flags.HasFlag(system.Tool) {
		glfw.WindowHint(glfw.Decorated, glfw.False)
	} else {
		glfw.WindowHint(glfw.Decorated, glfw.True)
	}
	glfw.WindowHint(glfw.CocoaRetinaFramebuffer, glfw.True)
	// glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)
	// todo: glfw.Floating for always-on-top -- could set for modal
	sz := opts.Size
	if TheApp.Platform() == system.MacOS {
		// on macOS, the size we pass to glfw must be in window manager units
		sz = sc.WindowSizeFromPixels(opts.Size)
	}
	fullscreen := opts.Flags.HasFlag(system.Fullscreen)
	var mon *glfw.Monitor
	if fullscreen && sc.ScreenNumber < len(TheApp.Monitors) {
		mon = TheApp.Monitors[sc.ScreenNumber]
		sz = sc.PixelSize // use screen size for fullscreen video mode resolution
	}
	glw, err := glfw.CreateWindow(sz.X, sz.Y, opts.GetTitle(), mon, nil)
	if err != nil {
		return err
	}
	if !fullscreen {
		pos := opts.Pos.Add(sc.Geometry.Min) // screen relative
		glw.SetPos(pos.X, pos.Y)
		w.Pos = pos
	}
	if opts.Icon != nil {
		glw.SetIcon(opts.Icon)
	}
	w.Glw = glw
	return nil
}

// Screen gets the screen of the window, computing various window parameters.
func (w *Window) Screen() *system.Screen {
	if w == nil || w.Glw == nil {
		return TheApp.Screens[0]
	}
	w.Mu.Lock()
	defer w.Mu.Unlock()

	var sc *system.Screen
	mon := w.Glw.GetMonitor() // this returns nil for windowed windows -- i.e., most windows
	// that is super useless it seems. only works for fullscreen
	if mon != nil {
		if ScreenDebug {
			log.Printf("ScreenDebug: desktop.Window.Screen: %v: got screen: %v\n", w.Nm, mon.GetName())
		}
		sc = TheApp.ScreenByName(mon.GetName())
		if sc == nil {
			log.Printf("ScreenDebug: desktop.Window.Screen: could not find screen of name: %v\n", mon.GetName())
			sc = TheApp.Screens[0]
		}
		goto setScreen
	}
	sc = w.GetScreenOverlap()
	if sc == nil {
		return nil
	}
	// if monitorDebug {
	// 	log.Printf("ScreenDebug: desktop.Window.GetScreenOverlap: %v: got screen: %v\n", w.Nm, sc.Name)
	// }
setScreen:
	w.ScreenWindow = sc.Name
	w.PhysDPI = sc.PhysicalDPI
	if w.DevicePixelRatio != sc.DevicePixelRatio {
		w.DevicePixelRatio = sc.DevicePixelRatio
	}
	if w.LogDPI == 0 {
		w.LogDPI = sc.LogicalDPI
	}
	return sc
}

// GetScreenOverlap gets the monitor for given window
// based on overlap of geometry, using limited glfw 3.3 api,
// which does not provide this functionality.
// See: https://github.com/glfw/glfw/issues/1699
// This is adapted from slawrence2302's code posted there.
func (w *Window) GetScreenOverlap() *system.Screen {
	w.App.Mu.Lock()
	defer w.App.Mu.Unlock()

	var wgeom image.Rectangle
	wgeom.Min.X, wgeom.Min.Y = w.Glw.GetPos()
	var sz image.Point
	sz.X, sz.Y = w.Glw.GetSize()
	wgeom.Max = wgeom.Min.Add(sz)

	var csc *system.Screen
	var ovlp int
	for _, sc := range TheApp.Screens {
		isect := sc.Geometry.Intersect(wgeom).Size()
		ov := isect.X * isect.Y
		if ov > ovlp || ovlp == 0 {
			csc = sc
			ovlp = ov
		}
	}
	return csc
}

func (w *Window) Position(screen *system.Screen) image.Point {
	w.Mu.Lock()
	defer w.Mu.Unlock()
	if w.Glw == nil {
		return w.Pos
	}
	var ps image.Point
	ps.X, ps.Y = w.Glw.GetPos()
	w.Pos = ps
	if screen != nil {
		ps = ps.Sub(screen.Geometry.Min)
	}
	return ps
}

func (w *Window) SetTitle(title string) {
	if w.IsClosed() {
		return
	}
	w.Titl = title
	w.App.RunOnMain(func() {
		if w.Glw == nil { // by time we got to main, could be diff
			return
		}
		w.Glw.SetTitle(title)
	})
}

func (w *Window) SetIcon(images []image.Image) {
	if w.IsClosed() {
		return
	}
	w.App.RunOnMain(func() {
		if w.Glw == nil { // by time we got to main, could be diff
			return
		}
		w.Glw.SetIcon(images)
	})
}

func (w *Window) SetWinSize(sz image.Point) {
	if w.IsClosed() || w.Is(system.Fullscreen) {
		return
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.App.RunOnMain(func() {
		if w.Glw == nil { // by time we got to main, could be diff
			return
		}
		w.Glw.SetSize(sz.X, sz.Y)
	})
}

func (w *Window) SetSize(sz image.Point) {
	sc := w.Screen()
	sz = sc.WindowSizeFromPixels(sz)
	w.SetWinSize(sz)
}

func (w *Window) SetPos(pos image.Point, screen *system.Screen) {
	if w.IsClosed() || w.Is(system.Fullscreen) {
		return
	}
	if screen != nil {
		pos = pos.Add(screen.Geometry.Min)
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.App.RunOnMain(func() {
		if w.Glw == nil { // by time we got to main, could be diff
			return
		}
		w.Glw.SetPos(pos.X, pos.Y)
	})
}

func (w *Window) SetGeometry(fullscreen bool, pos, size image.Point, screen *system.Screen) {
	if w.IsClosed() {
		return
	}
	if pos == (image.Point{}) {
		pos = w.Pos
	}
	if size == (image.Point{}) {
		size = w.PixelSize
	}
	if screen != nil {
		pos = pos.Add(screen.Geometry.Min)
	}
	sc := w.Screen()
	if screen != nil {
		sc = screen // critical to use this b/c w.Screen() can be wrong on new screen sequence
	}
	size = sc.WindowSizeFromPixels(size)
	// note: anything run on main only doesn't need lock -- implicit lock
	w.App.RunOnMain(func() {
		if w.Glw == nil { // by time we got to main, could be diff
			return
		}
		switch {
		case w.Is(system.Fullscreen) && !fullscreen:
			w.Flgs.SetFlag(false, system.Fullscreen)
			w.Glw.SetMonitor(nil, pos.X, pos.Y, size.X, size.Y, glfw.DontCare)
		case fullscreen:
			w.Flgs.SetFlag(true, system.Fullscreen)
			if screen != nil {
				sc = screen
			}
			mon := w.App.Monitors[sc.ScreenNumber]
			w.Glw.SetMonitor(mon, 0, 0, sc.PixelSize.X, sc.PixelSize.Y, glfw.DontCare)
		default:
			w.Glw.SetSize(size.X, size.Y)
			w.Glw.SetPos(pos.X, pos.Y)
		}
	})
}

func (w *Window) ConstrainFrame(topOnly bool) sides.Sides[int] {
	if w.IsClosed() || w.Is(system.Fullscreen) || w.Is(system.Maximized) {
		return w.FrameSize
	}
	if TheApp.Platform() == system.Windows && w.Pos.X == -32000 || w.Pos.Y == -32000 {
		return w.FrameSize
	}
	l, t, r, b := w.Glw.GetFrameSize()
	w.FrameSize.Set(t, r, b, l)
	sc := w.Screen()
	if sc == nil {
		return w.FrameSize
	}
	scSize := sc.Geometry.Size()
	// fmt.Println("\tconstrainframe screen:", sc.Name, scSize, w.FrameSize, "geom:", w.Pos, w.WnSize)
	frSize := image.Pt(w.FrameSize.Left+w.FrameSize.Right, w.FrameSize.Top+w.FrameSize.Bottom)
	frOff := image.Pt(w.FrameSize.Left, w.FrameSize.Top)
	sz := w.WnSize.Add(frSize)
	scpos := w.Pos.Sub(sc.Geometry.Min)
	pos := scpos.Sub(frOff)
	cpos, csz := system.ConstrainWindowGeometry(pos, sz, scSize)
	cpos = cpos.Add(frOff)
	csz = csz.Sub(frSize)
	// fmt.Println("\tconstrainframe pos:", scpos, "cpos:", cpos, "size:", w.WnSize, "csz:", csz)
	change := false
	pos = scpos
	if !topOnly && cpos.X > pos.X {
		change = true
		pos.X = cpos.X
	}
	if cpos.Y > pos.Y {
		change = true
		pos.Y = cpos.Y
	}
	sz = w.WnSize
	if !topOnly && csz.X < sz.X {
		change = true
		sz.X = csz.X
	}
	if !topOnly && csz.Y < sz.Y {
		change = true
		sz.Y = csz.Y
	}
	if change {
		// fmt.Println("\tconstrainframe changed:", pos, sz)
		w.SetGeometry(false, pos, sz, sc)
	}
	return w.FrameSize
}

func (w *Window) Show() {
	if w.IsClosed() {
		return
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.App.RunOnMain(func() {
		if w.Glw == nil { // by time we got to main, could be diff
			return
		}
		w.Glw.Show()
	})
}

func (w *Window) Raise() {
	if w.IsClosed() {
		return
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.App.RunOnMain(func() {
		if w.Glw == nil { // by time we got to main, could be diff
			return
		}
		if w.Is(system.Minimized) {
			w.Glw.Restore()
		} else {
			w.Glw.Focus()
		}
	})
}

func (w *Window) Minimize() {
	if w.IsClosed() {
		return
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.App.RunOnMain(func() {
		if w.Glw == nil { // by time we got to main, could be diff
			return
		}
		w.Glw.Iconify()
	})
}

func (w *Window) Close() {
	if w == nil {
		return
	}
	w.Window.Close()

	w.Mu.Lock()
	defer w.Mu.Unlock()

	w.App.RunOnMain(func() {
		w.Draw.System.WaitDone()
		if w.DestroyGPUFunc != nil {
			w.DestroyGPUFunc()
		}
		var surf *gpu.Surface
		if sf, ok := w.Draw.Renderer().(*gpu.Surface); ok {
			surf = sf
		}
		w.Draw.Release()
		if surf != nil { // note: release surface after draw
			surf.Release()
		}
		w.Glw.Destroy()
		w.Glw = nil // marks as closed for all other calls
		w.Draw = nil
		w.Compose = nil
	})
}

func (w *Window) SetMousePos(x, y float64) {
	if !w.IsVisible() {
		return
	}
	w.Mu.Lock()
	defer w.Mu.Unlock()
	if TheApp.Platform() == system.MacOS {
		w.Glw.SetCursorPos(x/float64(w.DevicePixelRatio), y/float64(w.DevicePixelRatio))
	} else {
		w.Glw.SetCursorPos(x, y)
	}
}

func (w *Window) SetCursorEnabled(enabled, raw bool) {
	w.CursorEnabled = enabled
	if enabled {
		w.Glw.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	} else {
		w.Glw.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		if raw && glfw.RawMouseMotionSupported() {
			w.Glw.SetInputMode(glfw.RawMouseMotion, glfw.True)
		}
	}
}

/////////////////////////////////////////////////////////
//  Window Callbacks

func (w *Window) Moved(gw *glfw.Window, x, y int) {
	w.Mu.Lock()
	w.Pos = image.Pt(x, y)
	w.Mu.Unlock()
	// w.app.GetScreens() // this can crash here on win disconnect..
	w.updateGeometry() // critical to update size etc here.
	w.Event.Window(events.WinMove)
}

func (w *Window) WinResized(gw *glfw.Window, width, height int) {
	// w.app.GetScreens()  // this can crash here on win disconnect..
	if ScreenDebug {
		log.Printf("desktop.Window.WinResized: %v: %v (was: %v)\n", w.Nm, image.Pt(width, height), w.PixelSize)
	}
	w.updateMaximized()
	w.updateGeometry()
}

func (w *Window) updateMaximized() {
	w.Flgs.SetFlag(w.Glw.GetAttrib(glfw.Maximized) == glfw.True, system.Maximized)
}

func (w *Window) updateGeometry() {
	w.Mu.Lock()
	cursc := w.ScreenWindow
	w.Mu.Unlock()
	sc := w.Screen() // gets parameters
	w.Mu.Lock()
	w.updateMaximized()
	var wsz image.Point
	wsz.X, wsz.Y = w.Glw.GetSize()
	w.WnSize = wsz
	var fbsz image.Point
	fbsz.X, fbsz.Y = w.Glw.GetFramebufferSize()
	w.PixelSize = fbsz
	w.PhysDPI = sc.PhysicalDPI
	w.LogDPI = sc.LogicalDPI
	w.Mu.Unlock()
	w.Draw.System.Renderer.SetSize(w.PixelSize)
	if cursc != w.ScreenWindow {
		if ScreenDebug {
			log.Printf("desktop.Window.updateGeometry: %v: got new screen: %v (was: %v)\n", w.Nm, w.ScreenWindow, cursc)
		}
	}
	w.Event.WindowResize()
}

func (w *Window) FbResized(gw *glfw.Window, width, height int) {
	if w.Is(system.Fullscreen) {
		sc := w.Screen()
		width = sc.PixelSize.X
		height = sc.PixelSize.Y
	}
	fbsz := image.Point{width, height}
	if w.PixelSize != fbsz {
		if ScreenDebug {
			log.Printf("desktop.Window.FbResized: %v: %v (was: %v)\n", w.Nm, fbsz, w.PixelSize)
		}
		w.updateGeometry()
	}
}

func (w *Window) OnCloseReq(gw *glfw.Window) {
	go w.CloseReq()
}

func (w *Window) Focused(gw *glfw.Window, focused bool) {
	// w.Flgs.SetFlag(focused, system.Focused)
	if focused {
		w.Event.Window(events.WinFocus)
	} else {
		w.Event.Last.MousePos = image.Point{-1, -1} // key for preventing random click to same location
		w.Event.Window(events.WinFocusLost)
	}
}

func (w *Window) Iconify(gw *glfw.Window, iconified bool) {
	w.Flgs.SetFlag(iconified, system.Minimized)
	if iconified {
		w.Event.Window(events.WinMinimize)
	} else {
		w.Screen() // gets parameters
		w.Event.Window(events.WinMinimize)
	}
}
