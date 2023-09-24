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
	"sync"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/girl/gist"
	"goki.dev/goosi"
	"goki.dev/goosi/driver/internal/event"
	"goki.dev/goosi/window"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"

	vk "github.com/goki/vulkan"
)

type windowImpl struct {
	goosi.WindowBase
	event.Deque
	app            *appImpl
	glw            *glfw.Window
	Surface        *vgpu.Surface
	Draw           vdraw.Drawer
	EventMgr       eventmgr.Mgr
	scrnName       string // last known screen name
	runQueue       chan funcRun
	publish        chan struct{}
	publishDone    chan struct{}
	winClose       chan struct{}
	mu             sync.Mutex
	mainMenu       goosi.MainMenu
	closeReqFunc   func(win goosi.Window)
	closeCleanFunc func(win goosi.Window)
	mouseDisabled  bool
	resettingPos   bool
}

var _ goosi.Window = &windowImpl{}

// Handle returns the driver-specific handle for this window.
// Currently, for all platforms, this is *glfw.Window, but that
// cannot always be assumed.  Only provided for unforeseen emergency use --
// please file an Issue for anything that should be added to Window
// interface.
func (w *windowImpl) Handle() any {
	return w.glw
}

func (w *windowImpl) Drawer() *vdraw.Drawer {
	return &w.Draw
}

func (w *windowImpl) IsClosed() bool {
	if w == nil {
		return true
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.glw == nil
}

func (w *windowImpl) IsVisible() bool {
	if w == nil || theApp.noScreens {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.glw != nil && !w.IsMinimized()
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
func (w *windowImpl) Activate() bool {
	// note: activate is only run on main thread so we don't need to check for mutex
	if w == nil || w.glw == nil {
		return false
	}
	w.glw.MakeContextCurrent()
	return true
}

// DeActivate() clears the current render target and gpu rendering context.
// Generally more efficient to NOT call this and just be sure to call
// Activate where relevant, so that if the window is already current context
// no switching is required.
// Must call this on app main thread using goosi.TheApp.RunOnMain
func (w *windowImpl) DeActivate() {
	glfw.DetachCurrentContext()
}

// must be run on main
func newVkWindow(opts *goosi.NewWindowOptions, sc *goosi.Screen) (*glfw.Window, error) {
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
	// todo: glfw.Floating for always-on-top -- could set for modal
	sz := sc.WinSizeFmPix(opts.Size) // note: this is in physical device units
	win, err := glfw.CreateWindow(sz.X, sz.Y, opts.GetTitle(), nil, nil)
	if err != nil {
		return win, err
	}

	win.SetPos(opts.Pos.X, opts.Pos.Y)
	return win, err
}

// for sending window.Event's
func (w *windowImpl) sendWindowEvent(act window.Actions) {
	winEv := window.Event{
		Action: act,
	}
	winEv.Init()
	w.Send(&winEv)
}

// NextEvent implements the goosi.EventDeque interface.
func (w *windowImpl) NextEvent() goosi.Event {
	e := w.Deque.NextEvent()
	return e
}

// winLoop is the window's own locked processing loop.
func (w *windowImpl) winLoop() {
	winShow := time.NewTimer(time.Second / 2) // this is a backup to ensure shown eventually..
	var winPaint *time.Ticker
	if w.FPS > 0 {
		winPaint = time.NewTicker(time.Second / time.Duration(w.FPS))
	} else {
		winPaint = &time.Ticker{C: make(chan time.Time)} // nop
	}
	hasShown := false
outer:
	for {
		select {
		case <-w.winClose:
			winShow.Stop() // todo: close channel too??
			winPaint.Stop()
			hasShown = false
			break outer
		case f := <-w.runQueue:
			if w.glw == nil {
				break outer
			}
			f.f()
			if f.done != nil {
				f.done <- true
			}
		case <-winShow.C:
			if w.glw == nil {
				break outer
			}
			w.sendWindowEvent(window.Show)
			hasShown = true
		case <-winPaint.C:
			if w.glw == nil {
				break outer
			}
			if hasShown {
				w.sendWindowEvent(window.Paint)
			}
		}
	}
}

// RunOnWin runs given function on the window's unique locked thread.
func (w *windowImpl) RunOnWin(f func()) {
	if w.IsClosed() {
		return
	}
	done := make(chan bool)
	w.runQueue <- funcRun{f: f, done: done}
	<-done
}

// GoRunOnWin runs given function on window's unique locked thread and returns immediately
func (w *windowImpl) GoRunOnWin(f func()) {
	if w.IsClosed() {
		return
	}
	go func() {
		w.runQueue <- funcRun{f: f, done: nil}
	}()
}

// SendEmptyEvent sends an empty, blank event to this window, which just has
// the effect of pushing the system along during cases when the window
// event loop needs to be "pinged" to get things moving along..
func (w *windowImpl) SendEmptyEvent() {
	if w.IsClosed() {
		return
	}
	goosi.SendCustomEvent(w, nil)
	glfw.PostEmptyEvent() // for good measure
}

////////////////////////////////////////////////////////////
//  Geom etc

func (w *windowImpl) Screen() *goosi.Screen {
	sc := w.getScreen()
	return sc
}

func (w *windowImpl) Size() image.Point {
	// w.mu.Lock() // this prevents race conditions but also locks up
	// defer w.mu.Unlock()
	return w.PxSize
}

func (w *windowImpl) WinSize() image.Point {
	// w.mu.Lock()
	// defer w.mu.Unlock()
	return w.WnSize
}

func (w *windowImpl) Position() image.Point {
	w.mu.Lock()
	defer w.mu.Unlock()
	var ps image.Point
	ps.X, ps.Y = w.glw.GetPos()
	w.Pos = ps
	return ps
}

func (w *windowImpl) Insets() gist.SideFloats {
	return gist.NewSideFloats()
}

func (w *windowImpl) PhysicalDPI() float32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.PhysDPI
}

func (w *windowImpl) LogicalDPI() float32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.LogDPI
}

func (w *windowImpl) SetLogicalDPI(dpi float32) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.LogDPI = dpi
}

func (w *windowImpl) SetTitle(title string) {
	if w.IsClosed() {
		return
	}
	w.Titl = title
	w.app.RunOnMain(func() {
		if w.glw == nil { // by time we got to main, could be diff
			return
		}
		w.glw.SetTitle(title)
	})
}

func (w *windowImpl) SetWinSize(sz image.Point) {
	if w.IsClosed() {
		return
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.app.RunOnMain(func() {
		if w.glw == nil { // by time we got to main, could be diff
			return
		}
		w.glw.SetSize(sz.X, sz.Y)
	})
}

func (w *windowImpl) SetSize(sz image.Point) {
	if w.IsClosed() {
		return
	}
	sc := w.getScreen()
	sz = sc.WinSizeFmPix(sz)
	w.SetWinSize(sz)
}

func (w *windowImpl) SetPos(pos image.Point) {
	if w.IsClosed() {
		return
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.app.RunOnMain(func() {
		if w.glw == nil { // by time we got to main, could be diff
			return
		}
		w.glw.SetPos(pos.X, pos.Y)
	})
}

func (w *windowImpl) SetGeom(pos image.Point, sz image.Point) {
	if w.IsClosed() {
		return
	}
	sc := w.getScreen()
	sz = sc.WinSizeFmPix(sz)
	// note: anything run on main only doesn't need lock -- implicit lock
	w.app.RunOnMain(func() {
		if w.glw == nil { // by time we got to main, could be diff
			return
		}
		w.glw.SetSize(sz.X, sz.Y)
		w.glw.SetPos(pos.X, pos.Y)
	})
}

func (w *windowImpl) show() {
	if w.IsClosed() {
		return
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.app.RunOnMain(func() {
		if w.glw == nil { // by time we got to main, could be diff
			return
		}
		w.glw.Show()
	})
}

func (w *windowImpl) Raise() {
	if w.IsClosed() {
		return
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.app.RunOnMain(func() {
		if w.glw == nil { // by time we got to main, could be diff
			return
		}
		// if bitflag.HasAtomic(&w.Flag, int(goosi.Minimized)) {
		// 	w.glw.Restore()
		// } else {
		// 	w.glw.Focus()
		// }
	})
}

func (w *windowImpl) Minimize() {
	if w.IsClosed() {
		return
	}
	// note: anything run on main only doesn't need lock -- implicit lock
	w.app.RunOnMain(func() {
		if w.glw == nil { // by time we got to main, could be diff
			return
		}
		w.glw.Iconify()
	})
}

func (w *windowImpl) SetCloseReqFunc(fun func(win goosi.Window)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closeReqFunc = fun
}

func (w *windowImpl) SetCloseCleanFunc(fun func(win goosi.Window)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closeCleanFunc = fun
}

func (w *windowImpl) CloseReq() {
	if theApp.quitting {
		w.Close()
	}
	if w.closeReqFunc != nil {
		w.closeReqFunc(w)
	} else {
		w.Close()
	}
}

func (w *windowImpl) CloseClean() {
	if w.closeCleanFunc != nil {
		w.closeCleanFunc(w)
	}
}

func (w *windowImpl) Close() {
	// this is actually the final common pathway for closing here
	w.mu.Lock()
	w.winClose <- struct{}{} // break out of draw loop
	w.CloseClean()
	// fmt.Printf("sending close event to window: %v\n", w.Nm)
	w.sendWindowEvent(window.Close)
	theApp.DeleteWin(w)
	w.app.RunOnMain(func() {
		vk.DeviceWaitIdle(w.Surface.Device.Device)
		w.DestroyGPUfunc()
		w.Draw.Destroy()
		w.Surface.Destroy()
		w.glw.Destroy()
		w.glw = nil // marks as closed for all other calls
	})
	if theApp.quitting {
		theApp.quitCloseCnt <- struct{}{}
	}
	w.mu.Unlock()
}

func (w *windowImpl) SetMousePos(x, y float64) {
	if !w.IsVisible() {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if theApp.Platform() == goosi.MacOS {
		w.glw.SetCursorPos(x/float64(w.DevPixRatio), y/float64(w.DevPixRatio))
	} else {
		w.glw.SetCursorPos(x, y)
	}
}

func (w *windowImpl) SetCursorEnabled(enabled, raw bool) {
	if enabled {
		w.mouseDisabled = false
		w.glw.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
	} else {
		w.mouseDisabled = true
		w.glw.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		if raw && glfw.RawMouseMotionSupported() {
			w.glw.SetInputMode(glfw.RawMouseMotion, glfw.True)
		}
	}
}

/////////////////////////////////////////////////////////
//  Window Callbacks

func (w *windowImpl) getScreen() *goosi.Screen {
	if w == nil || w.glw == nil {
		return theApp.screens[0]
	}
	w.mu.Lock()
	var sc *goosi.Screen
	mon := w.glw.GetMonitor() // this returns nil for windowed windows -- i.e., most windows
	// that is super useless it seems. only works for fullscreen
	if mon != nil {
		if monitorDebug {
			log.Printf("MonitorDebug: vkos window: %v getScreen() -- got screen: %v\n", w.Nm, mon.GetName())
		}
		sc = theApp.ScreenByName(mon.GetName())
		if sc == nil {
			log.Printf("MonitorDebug: vkos getScreen: could not find screen of name: %v\n", mon.GetName())
			sc = theApp.screens[0]
		}
		goto setScreen
	}
	sc = w.getScreenOvlp()
	// if monitorDebug {
	// 	log.Printf("MonitorDebug: vkos window: %v getScreenOvlp() -- got screen: %v\n", w.Nm, sc.Name)
	// }
setScreen:
	w.scrnName = sc.Name
	w.PhysDPI = sc.PhysicalDPI
	w.DevPixRatio = sc.DevicePixelRatio
	if w.LogDPI == 0 {
		w.LogDPI = sc.LogicalDPI
	}
	w.mu.Unlock()
	return sc
}

// getScreenOvlp gets the monitor for given window
// based on overlap of geometry, using limited glfw 3.3 api,
// which does not provide this functionality.
// See: https://github.com/glfw/glfw/issues/1699
// This is adapted from slawrence2302's code posted there.
func (w *windowImpl) getScreenOvlp() *goosi.Screen {
	var wgeom image.Rectangle
	wgeom.Min.X, wgeom.Min.Y = w.glw.GetPos()
	var sz image.Point
	sz.X, sz.Y = w.glw.GetSize()
	wgeom.Max = wgeom.Min.Add(sz)

	var csc *goosi.Screen
	var ovlp int
	for _, sc := range theApp.screens {
		isect := sc.Geometry.Intersect(wgeom).Size()
		ov := isect.X * isect.Y
		if ov > ovlp || ovlp == 0 {
			csc = sc
			ovlp = ov
		}
	}
	return csc
}

func (w *windowImpl) moved(gw *glfw.Window, x, y int) {
	w.mu.Lock()
	w.Pos = image.Point{x, y}
	w.mu.Unlock()
	// w.app.GetScreens() // this can crash here on win disconnect..
	w.getScreen()
	w.sendWindowEvent(window.Move)
}

func (w *windowImpl) winResized(gw *glfw.Window, width, height int) {
	// w.app.GetScreens()  // this can crash here on win disconnect..
	w.updtGeom()
}

func (w *windowImpl) updtGeom() {
	w.mu.Lock()
	cursc := w.scrnName
	w.mu.Unlock()
	sc := w.getScreen() // gets devpixratio etc
	w.mu.Lock()
	var wsz image.Point
	wsz.X, wsz.Y = w.glw.GetSize()
	// fmt.Printf("win size: %v\n", wsz)
	w.WnSize = wsz
	var fbsz image.Point
	fbsz.X, fbsz.Y = w.glw.GetFramebufferSize()
	w.PxSize = fbsz
	w.PhysDPI = sc.PhysicalDPI
	w.LogDPI = sc.LogicalDPI
	w.mu.Unlock()
	// if w.Activate() {
	// 	w.winTex.SetSize(w.PxSize)
	// }
	if cursc != w.scrnName {
		if monitorDebug {
			log.Printf("vkos window: %v updtGeom() -- got new screen: %v (was: %v)\n", w.Nm, w.scrnName, cursc)
		}
	}
	w.sendWindowEvent(window.Resize) // this will not get processed until the end
}

func (w *windowImpl) fbResized(gw *glfw.Window, width, height int) {
	fbsz := image.Point{width, height}
	if w.PxSize != fbsz {
		w.updtGeom()
	}
}

func (w *windowImpl) closeReq(gw *glfw.Window) {
	go w.CloseReq()
}

func (w *windowImpl) refresh(gw *glfw.Window) {
	// go w.Publish()
}

func (w *windowImpl) focus(gw *glfw.Window, focused bool) {
	if focused {
		// fmt.Printf("foc win: %v, foc: %v\n", w.Nm, bitflag.HasAtomic(&w.Flag, int(goosi.Focus)))
		if w.mainMenu != nil {
			w.mainMenu.SetMenu()
		}
		// bitflag.ClearAtomic(&w.Flag, int(goosi.Minimized))
		// bitflag.SetAtomic(&w.Flag, int(goosi.Focus))
		w.sendWindowEvent(window.Focus)
	} else {
		// fmt.Printf("unfoc win: %v, foc: %v\n", w.Nm, bitflag.HasAtomic(&w.Flag, int(goosi.Focus)))
		// bitflag.ClearAtomic(&w.Flag, int(goosi.Focus))
		lastMousePos = image.Point{-1, -1} // key for preventing random click to same location
		w.sendWindowEvent(window.DeFocus)
	}
}

func (w *windowImpl) iconify(gw *glfw.Window, iconified bool) {
	if iconified {
		// bitflag.SetAtomic(&w.Flag, int(goosi.Minimized))
		// bitflag.ClearAtomic(&w.Flag, int(goosi.Focus))
		w.sendWindowEvent(window.Minimize)
	} else {
		// bitflag.ClearAtomic(&w.Flag, int(goosi.Minimized))
		w.getScreen()
		w.sendWindowEvent(window.Minimize)
	}
}
