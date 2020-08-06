// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// originally based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"sync"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/drawer"
	"github.com/goki/gi/oswin/driver/internal/event"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki/bitflag"
	"github.com/goki/mat32"
)

type windowImpl struct {
	oswin.WindowBase
	event.Deque
	app            *appImpl
	glw            *glfw.Window
	scrnName       string // last known screen name
	runQueue       chan funcRun
	publish        chan struct{}
	publishDone    chan struct{}
	winClose       chan struct{}
	winTex         *textureImpl
	mu             sync.Mutex
	mainMenu       oswin.MainMenu
	closeReqFunc   func(win oswin.Window)
	closeCleanFunc func(win oswin.Window)
	drawQuads      gpu.BufferMgr
	fillQuads      gpu.BufferMgr
	mouseDisabled  bool
	resettingPos   bool
}

// Handle returns the driver-specific handle for this window.
// Currently, for all platforms, this is *glfw.Window, but that
// cannot always be assumed.  Only provided for unforeseen emergency use --
// please file an Issue for anything that should be added to Window
// interface.
func (w *windowImpl) Handle() interface{} {
	return w.glw
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
	return w.glw != nil && w.winTex != nil && !w.IsMinimized()
}

// Activate() sets this window as the current render target for gpu rendering
// functions, and the current context for gpu state (equivalent to
// MakeCurrentContext on OpenGL).
// If it returns false, then window is not visible / valid and
// nothing further should happen.
// Must call this on app main thread using oswin.TheApp.RunOnMain
//
// oswin.TheApp.RunOnMain(func() {
//    if !win.Activate() {
//        return
//    }
//    // do GPU calls here
// })
//
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
// Must call this on app main thread using oswin.TheApp.RunOnMain
func (w *windowImpl) DeActivate() {
	glfw.DetachCurrentContext()
}

// must be run on main
func newGLWindow(opts *oswin.NewWindowOptions, sc *oswin.Screen) (*glfw.Window, error) {
	_, _, tool, fullscreen := oswin.WindowFlagsToBool(opts.Flags)
	glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Visible, glfw.False) // needed to position
	glfw.WindowHint(glfw.Focused, glfw.True)
	// glfw.WindowHint(glfw.ScaleToMonitor, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, glosGlMajor)
	glfw.WindowHint(glfw.ContextVersionMinor, glosGlMinor)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Samples, 0) // don't do multisampling for main window -- only in sub-render
	if glosDebug {
		glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	}

	// todo: glfw.Samples -- multisampling
	if fullscreen {
		glfw.WindowHint(glfw.Maximized, glfw.True)
	}
	if tool {
		glfw.WindowHint(glfw.Decorated, glfw.False)
	} else {
		glfw.WindowHint(glfw.Decorated, glfw.True)
	}
	// todo: glfw.Floating for always-on-top -- could set for modal
	sz := opts.Size // note: this is already in standard window size units!
	win, err := glfw.CreateWindow(sz.X, sz.Y, opts.GetTitle(), nil, theApp.shareWin)
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

// NextEvent implements the oswin.EventDeque interface.
func (w *windowImpl) NextEvent() oswin.Event {
	e := w.Deque.NextEvent()
	return e
}

// winLoop is the window's own locked processing loop.
func (w *windowImpl) winLoop() {
outer:
	for {
		select {
		case <-w.winClose:
			break outer
		case f := <-w.runQueue:
			if w.glw == nil {
				break outer
			}
			f.f()
			if f.done != nil {
				f.done <- true
			}
		case <-w.publish:
			if w.glw == nil {
				break outer
			}
			if !theApp.noScreens {
				theApp.RunOnMain(func() {
					if !w.Activate() {
						return
					}
					w.glw.SwapBuffers() // note: implicitly does a flush
					// note: generally don't need this:
					// gpu.Draw.Clear(true, true)
				})
				w.publishDone <- struct{}{}
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

// Publish does the equivalent of SwapBuffers on OpenGL: pushes the
// current rendered back-buffer to the front (and ensures that any
// ongoing rendering has completed) (see also PublishTex)
func (w *windowImpl) Publish() {
	if !w.IsVisible() {
		return
	}
	glfw.PostEmptyEvent()
	w.publish <- struct{}{}
	<-w.publishDone
	glfw.PostEmptyEvent()
}

// PublishTex draws the current WinTex texture to the window and then
// calls Publish() -- this is the typical update call.
func (w *windowImpl) PublishTex() {
	if !w.IsVisible() {
		return
	}
	theApp.RunOnMain(func() {
		if !w.Activate() || w.winTex == nil {
			return
		}
		w.Copy(image.ZP, w.winTex, w.winTex.Bounds(), oswin.Src, nil)
	})
	w.Publish()
}

// SendEmptyEvent sends an empty, blank event to this window, which just has
// the effect of pushing the system along during cases when the window
// event loop needs to be "pinged" to get things moving along..
func (w *windowImpl) SendEmptyEvent() {
	if w.IsClosed() {
		return
	}
	oswin.SendCustomEvent(w, nil)
	glfw.PostEmptyEvent() // for good measure
}

// WinTex() returns the current Texture of the same size as the window that
// is typically used to update the window contents.
// Use the various Drawer and SetSubImage methods to update this Texture, and
// then call PublishTex() to update the window.
// This Texture is automatically resized when the window is resized, and
// when that occurs, existing contents are lost -- a full update of the
// Texture at the current size is required at that point.
func (w *windowImpl) WinTex() oswin.Texture {
	return w.winTex
}

// SetWinTexSubImage calls SetSubImage on WinTex with given parameters.
// convenience routine that activates the window context and runs on the
// window's thread.
func (w *windowImpl) SetWinTexSubImage(dp image.Point, src image.Image, sr image.Rectangle) error {
	if !w.IsVisible() {
		return nil
	}
	var err error
	theApp.RunOnMain(func() {
		if !w.Activate() || w.winTex == nil {
			return
		}
		err = w.winTex.SetSubImage(dp, src, sr)
	})
	return err
}

////////////////////////////////////////////////
//   Drawer wrappers

func (w *windowImpl) Draw(src2dst mat32.Mat3, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	if !w.IsVisible() {
		return
	}
	theApp.RunOnMain(func() {
		if !w.Activate() {
			return
		}
		gpu.TheGPU.RenderToWindow()
		gpu.Draw.Viewport(image.Rectangle{Max: w.PxSize})
		if w.drawQuads == nil {
			w.drawQuads = theApp.drawQuadsBuff()
		}
		sz := w.Size()
		theApp.draw(sz, src2dst, src, sr, op, opts, w.drawQuads, true) // true = dest has botZero
	})
}

func (w *windowImpl) DrawUniform(src2dst mat32.Mat3, src color.Color, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	if !w.IsVisible() {
		return
	}
	theApp.RunOnMain(func() {
		if !w.Activate() {
			return
		}
		gpu.TheGPU.RenderToWindow()
		gpu.Draw.Viewport(image.Rectangle{Max: w.PxSize})
		if w.fillQuads == nil {
			w.fillQuads = theApp.fillQuadsBuff()
		}
		sz := w.Size()
		theApp.drawUniform(sz, src2dst, src, sr, op, opts, w.fillQuads, true) // true = dest has botZero
	})
}

func (w *windowImpl) Copy(dp image.Point, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	if !w.IsVisible() {
		return
	}
	drawer.Copy(w, dp, src, sr, op, opts)
}

func (w *windowImpl) Scale(dr image.Rectangle, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	if !w.IsVisible() {
		return
	}
	drawer.Scale(w, dr, src, sr, op, opts)
}

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	if !w.IsVisible() {
		return
	}
	theApp.RunOnMain(func() {
		if !w.Activate() {
			return
		}
		gpu.TheGPU.RenderToWindow()
		gpu.Draw.Viewport(image.Rectangle{Max: w.PxSize})
		if w.fillQuads == nil {
			w.fillQuads = theApp.fillQuadsBuff()
		}
		sz := w.Size()
		theApp.fillRect(sz, dr, src, op, w.fillQuads, true) // true = dest has botZero
	})
}

////////////////////////////////////////////////////////////
//  Geom etc

func (w *windowImpl) Screen() *oswin.Screen {
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

func (w *windowImpl) SetSize(sz image.Point) {
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

func (w *windowImpl) SetPixSize(sz image.Point) {
	if w.IsClosed() {
		return
	}
	sc := w.getScreen()
	sz.X = int(float32(sz.X) / sc.DevicePixelRatio)
	sz.Y = int(float32(sz.Y) / sc.DevicePixelRatio)
	w.SetSize(sz)
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
		if bitflag.HasAtomic(&w.Flag, int(oswin.Minimized)) {
			w.glw.Restore()
		} else {
			w.glw.Focus()
		}
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

func (w *windowImpl) SetCloseReqFunc(fun func(win oswin.Window)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closeReqFunc = fun
}

func (w *windowImpl) SetCloseCleanFunc(fun func(win oswin.Window)) {
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
		if w.winTex != nil {
			w.winTex.Delete()
			w.winTex = nil
		}
		if w.drawQuads != nil {
			w.drawQuads.Delete()
			w.drawQuads = nil
		}
		if w.fillQuads != nil {
			w.fillQuads.Delete()
			w.fillQuads = nil
		}
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
	if theApp.Platform() == oswin.MacOS {
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

func (w *windowImpl) getScreen() *oswin.Screen {
	if w == nil || w.glw == nil {
		return theApp.screens[0]
	}
	w.mu.Lock()
	var sc *oswin.Screen
	mon := w.glw.GetMonitor() // this returns nil for windowed windows -- i.e., most windows
	// that is super useless it seems.
	if mon != nil {
		if monitorDebug {
			log.Printf("glos window: %v getScreen() -- got screen: %v\n", w.Nm, mon.GetName())
		}
		sc = theApp.ScreenByName(mon.GetName())
		if sc == nil {
			log.Printf("glos getScreen: could not find screen of name: %v\n", mon.GetName())
			sc = theApp.screens[0]
		}
	} else {
		sc = theApp.ScreenByName(w.scrnName)
		got := false
		if sc == nil || w.DevPixRatio != sc.DevicePixelRatio {
			for _, scc := range theApp.screens {
				if w.DevPixRatio == scc.DevicePixelRatio {
					sc = scc
					got = true
					if monitorDebug {
						log.Printf("glos window: %v getScreen(): matched pix ratio %v for screen: %v\n", w.Nm, w.DevPixRatio, sc.Name)
					}
					w.LogDPI = sc.LogicalDPI
					break
				}
			}
			if !got {
				sc = theApp.screens[0]
				w.LogDPI = sc.LogicalDPI
				if monitorDebug {
					log.Printf("glos window: %v getScreen(): reverting to first screen %v\n", w.Nm, sc.Name)
				}
			}
		}
	}
	w.scrnName = sc.Name
	w.PhysDPI = sc.PhysicalDPI
	w.DevPixRatio = sc.DevicePixelRatio
	if w.LogDPI == 0 {
		w.LogDPI = sc.LogicalDPI
	}
	w.mu.Unlock()
	return sc
}

func (w *windowImpl) moved(gw *glfw.Window, x, y int) {
	w.mu.Lock()
	w.Pos = image.Point{x, y}
	w.mu.Unlock()
	w.getScreen()
	w.sendWindowEvent(window.Move)
}

func (w *windowImpl) winResized(gw *glfw.Window, width, height int) {
	w.updtGeom()
}

func (w *windowImpl) updtGeom() {
	w.mu.Lock()
	cscx, _ := w.glw.GetContentScale()
	// curDevPixRatio := w.DevPixRatio
	w.DevPixRatio = cscx
	// if curDevPixRatio != w.DevPixRatio {
	// 	fmt.Printf("got cont scale: %v\n", cscx)
	// }
	cursc := w.scrnName
	w.mu.Unlock()
	sc := w.getScreen()
	w.mu.Lock()
	var wsz image.Point
	wsz.X, wsz.Y = w.glw.GetSize()
	// fmt.Printf("win size: %v\n", wsz)
	w.WnSize = wsz
	// todo: this doesn't work on mac -- ignores the size -- uses glw directly probably
	// if curDevPixRatio != w.DevPixRatio && curDevPixRatio > 0 {
	// 	rr := w.DevPixRatio / curDevPixRatio
	// 	w.WnSize.X = int(float32(w.WnSize.X) * rr)
	// 	w.WnSize.Y = int(float32(w.WnSize.Y) * rr)
	// 	fmt.Printf("resized based on pix ratio: %v\n", w.WnSize)
	// }
	var fbsz image.Point
	fbsz.X, fbsz.Y = w.glw.GetFramebufferSize()
	w.PxSize = fbsz
	w.PhysDPI = sc.PhysicalDPI
	w.LogDPI = sc.LogicalDPI
	w.mu.Unlock()
	if w.Activate() {
		w.winTex.SetSize(w.PxSize)
	}
	if cursc != w.scrnName {
		if monitorDebug {
			log.Printf("glos window: %v updtGeom() -- got new screen: %v (was: %v)\n", w.Nm, w.scrnName, cursc)
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
	go w.Publish()
}

func (w *windowImpl) focus(gw *glfw.Window, focused bool) {
	if focused {
		// fmt.Printf("foc win: %v, foc: %v\n", w.Nm, bitflag.HasAtomic(&w.Flag, int(oswin.Focus)))
		if w.mainMenu != nil {
			w.mainMenu.SetMenu()
		}
		bitflag.ClearAtomic(&w.Flag, int(oswin.Minimized))
		bitflag.SetAtomic(&w.Flag, int(oswin.Focus))
		w.sendWindowEvent(window.Focus)
	} else {
		// fmt.Printf("unfoc win: %v, foc: %v\n", w.Nm, bitflag.HasAtomic(&w.Flag, int(oswin.Focus)))
		bitflag.ClearAtomic(&w.Flag, int(oswin.Focus))
		lastMousePos = image.Point{-1, -1} // key for preventing random click to same location
		w.sendWindowEvent(window.DeFocus)
	}
}

func (w *windowImpl) iconify(gw *glfw.Window, iconified bool) {
	if iconified {
		bitflag.SetAtomic(&w.Flag, int(oswin.Minimized))
		bitflag.ClearAtomic(&w.Flag, int(oswin.Focus))
		w.sendWindowEvent(window.Minimize)
	} else {
		bitflag.ClearAtomic(&w.Flag, int(oswin.Minimized))
		w.getScreen()
		w.sendWindowEvent(window.Minimize)
	}
}
