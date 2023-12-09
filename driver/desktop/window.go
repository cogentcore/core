// Copyright 2019 The GoKi Authors. All rights reserved.
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

	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/goosi"
	"goki.dev/goosi/driver/base"
	"goki.dev/goosi/events"
	"goki.dev/vgpu/v2/vdraw"

	vk "github.com/goki/vulkan"
)

// Window is the implementation of [goosi.Window] for the desktop platform.
type Window struct { //gti:add
	base.WindowMulti[*App, *vdraw.Drawer]

	// Glw is the glfw window associated with this window
	Glw *glfw.Window

	// ScreenName is the name of the last known screen this window was on
	ScreenWindow string
}

// Activate() sets this window as the current render target for gpu rendering
// functions, and the current context for gpu state (equivalent to
// MakeCurrentContext on OpenGL).
// If it returns false, then window is not visible / valid and
// nothing further should happen.
// Must call this on app main thread using goosi.TheApp.RunOnMain
//
//	goosi.TheApp.RunOnMain(func() {
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
// Must call this on app main thread using goosi.TheApp.RunOnMain
func (w *Window) DeActivate() {
	glfw.DetachCurrentContext()
}

// NewGlfwWindow makes a new glfw window.
// It must be run on main.
func NewGlfwWindow(opts *goosi.NewWindowOptions, sc *goosi.Screen) (*glfw.Window, error) {
	_, _, tool, fullscreen := goosi.WindowFlagsToBool(opts.Flags)
	// glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Visible, glfw.False) // needed to position
	glfw.WindowHint(glfw.Focused, glfw.True)
	// glfw.WindowHint(glfw.ScaleToMonitor, glfw.True)
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	// glfw.WindowHint(glfw.Samples, 0) // don't do multisampling for main window -- only in sub-render
	if fullscreen {
		glfw.WindowHint(glfw.Maximized, glfw.True)
	}
	if tool {
		glfw.WindowHint(glfw.Decorated, glfw.False)
	} else {
		glfw.WindowHint(glfw.Decorated, glfw.True)
	}
	// glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)
	// todo: glfw.Floating for always-on-top -- could set for modal
	sz := sc.WinSizeFmPix(opts.Size) // note: this is in physical device units
	win, err := glfw.CreateWindow(sz.X, sz.Y, opts.GetTitle(), nil, nil)
	if err != nil {
		return win, err
	}

	win.SetPos(opts.Pos.X, opts.Pos.Y)
	return win, err
}

// Screen gets the screen of the window, computing various window parameters.
func (w *Window) Screen() *goosi.Screen {
	if w == nil || w.Glw == nil {
		return TheApp.Screens[0]
	}
	w.Mu.Lock()
	defer w.Mu.Unlock()

	var sc *goosi.Screen
	mon := w.Glw.GetMonitor() // this returns nil for windowed windows -- i.e., most windows
	// that is super useless it seems. only works for fullscreen
	if mon != nil {
		if MonitorDebug {
			log.Printf("MonitorDebug: vkos window: %v getScreen() -- got screen: %v\n", w.Nm, mon.GetName())
		}
		sc = TheApp.ScreenByName(mon.GetName())
		if sc == nil {
			log.Printf("MonitorDebug: vkos getScreen: could not find screen of name: %v\n", mon.GetName())
			sc = TheApp.Screens[0]
		}
		goto setScreen
	}
	sc = w.GetScreenOverlap()
	// if monitorDebug {
	// 	log.Printf("MonitorDebug: vkos window: %v getScreenOvlp() -- got screen: %v\n", w.Nm, sc.Name)
	// }
setScreen:
	w.ScreenWindow = sc.Name
	w.PhysDPI = sc.PhysicalDPI
	w.DevicePixelRatio = sc.DevicePixelRatio
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
func (w *Window) GetScreenOverlap() *goosi.Screen {
	var wgeom image.Rectangle
	wgeom.Min.X, wgeom.Min.Y = w.Glw.GetPos()
	var sz image.Point
	sz.X, sz.Y = w.Glw.GetSize()
	wgeom.Max = wgeom.Min.Add(sz)

	var csc *goosi.Screen
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

func (w *Window) SetWinSize(sz image.Point) {
	if w.IsClosed() {
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

func (w *Window) SetPos(pos image.Point) {
	if w.IsClosed() {
		return
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.App.RunOnMain(func() {
		if w.Glw == nil { // by time we got to main, could be diff
			return
		}
		w.Glw.SetPos(pos.X, pos.Y)
	})
}

func (w *Window) SetGeom(pos image.Point, sz image.Point) {
	if w.IsClosed() {
		return
	}
	sc := w.Screen()
	sz = sc.WinSizeFmPix(sz)
	// note: anything run on main only doesn't need lock -- implicit lock
	w.App.RunOnMain(func() {
		if w.Glw == nil { // by time we got to main, could be diff
			return
		}
		w.Glw.SetSize(sz.X, sz.Y)
		w.Glw.SetPos(pos.X, pos.Y)
	})
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
		if w.Is(goosi.Minimized) {
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
	w.Window.Close()

	w.Mu.Lock()
	defer w.Mu.Unlock()

	w.App.RunOnMain(func() {
		vk.DeviceWaitIdle(w.Draw.Surf.Device.Device)
		if w.DestroyGPUFunc != nil {
			w.DestroyGPUFunc()
		}
		w.Draw.Destroy()
		w.Draw.Surf.Destroy()
		w.Glw.Destroy()
		w.Glw = nil // marks as closed for all other calls
		w.Draw = nil
	})
}

func (w *Window) SetMousePos(x, y float64) {
	if !w.IsVisible() {
		return
	}
	w.Mu.Lock()
	defer w.Mu.Unlock()
	if TheApp.Platform() == goosi.MacOS {
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
	w.Pos = image.Point{x, y}
	w.Mu.Unlock()
	// w.app.GetScreens() // this can crash here on win disconnect..
	w.Screen() // gets parameters
	w.EvMgr.Window(events.WinMove)
}

func (w *Window) WinResized(gw *glfw.Window, width, height int) {
	// w.app.GetScreens()  // this can crash here on win disconnect..
	w.UpdateGeom()
}

func (w *Window) UpdateGeom() {
	w.Mu.Lock()
	cursc := w.ScreenWindow
	w.Mu.Unlock()
	sc := w.Screen() // gets parameters
	w.Mu.Lock()
	var wsz image.Point
	wsz.X, wsz.Y = w.Glw.GetSize()
	// fmt.Printf("win size: %v\n", wsz)
	w.WnSize = wsz
	var fbsz image.Point
	fbsz.X, fbsz.Y = w.Glw.GetFramebufferSize()
	w.PixSize = fbsz
	w.PhysDPI = sc.PhysicalDPI
	w.LogDPI = sc.LogicalDPI
	w.Mu.Unlock()
	// if w.Activate() {
	// 	w.winTex.SetSize(w.PxSize)
	// }
	if cursc != w.ScreenWindow {
		if MonitorDebug {
			log.Printf("vkos window: %v updtGeom() -- got new screen: %v (was: %v)\n", w.Nm, w.ScreenWindow, cursc)
		}
	}
	w.EvMgr.WindowResize()
}

func (w *Window) FbResized(gw *glfw.Window, width, height int) {
	fbsz := image.Point{width, height}
	if w.PixSize != fbsz {
		w.UpdateGeom()
	}
}

func (w *Window) OnCloseReq(gw *glfw.Window) {
	go w.CloseReq()
}

func (w *Window) Focused(gw *glfw.Window, focused bool) {
	if focused {
		// fmt.Printf("foc win: %v, foc: %v\n", w.Nm, bitflag.HasAtomic(&w.Flag, int(goosi.Focus)))
		// TODO(kai): main menu
		// if w.mainMenu != nil {
		// 	w.mainMenu.SetMenu()
		// }
		// bitflag.ClearAtomic(&w.Flag, int(goosi.Minimized))
		// bitflag.SetAtomic(&w.Flag, int(goosi.Focus))
		w.EvMgr.Window(events.WinFocus)
	} else {
		// fmt.Printf("unfoc win: %v, foc: %v\n", w.Nm, bitflag.HasAtomic(&w.Flag, int(goosi.Focus)))
		// bitflag.ClearAtomic(&w.Flag, int(goosi.Focus))
		w.EvMgr.Last.MousePos = image.Point{-1, -1} // key for preventing random click to same location
		w.EvMgr.Window(events.WinFocusLost)
	}
}

func (w *Window) Iconify(gw *glfw.Window, iconified bool) {
	if iconified {
		// bitflag.SetAtomic(&w.Flag, int(goosi.Minimized))
		// bitflag.ClearAtomic(&w.Flag, int(goosi.Focus))
		w.EvMgr.Window(events.WinMinimize)
	} else {
		// bitflag.ClearAtomic(&w.Flag, int(goosi.Minimized))
		w.Screen() // gets parameters
		w.EvMgr.Window(events.WinMinimize)
	}
}
