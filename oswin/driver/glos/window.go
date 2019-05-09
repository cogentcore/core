// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
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

	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/drawer"
	"github.com/goki/gi/oswin/driver/internal/event"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki/bitflag"
)

type windowImpl struct {
	oswin.WindowBase
	event.Deque
	app            *appImpl
	glw            *glfw.Window
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
}

// Handle returns the driver-specific handle for this window.
// Currently, for all platforms, this is *glfw.Window, but that
// cannot always be assumed.  Only provided for unforseen emergency use --
// please file an Issue for anything that should be added to Window
// interface.
func (w *windowImpl) Handle() interface{} {
	return w.glw
}

// Activate() sets this window as the current render target for gpu rendering
// functions, and the current context for gpu state (equivalent to
// MakeCurrentContext on OpenGL).
// Must call this on app main thread using oswin.TheApp.RunOnMain
//
// oswin.TheApp.RunOnMain(func() {
//    win.Activate()
//    // do GPU calls here
// })
//
func (w *windowImpl) Activate() {
	w.glw.MakeContextCurrent()
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
func newGLWindow(opts *oswin.NewWindowOptions) (*glfw.Window, error) {
	_, _, tool, fullscreen := oswin.WindowFlagsToBool(opts.Flags)
	glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Visible, glfw.False) // needed to position
	glfw.WindowHint(glfw.Focused, glfw.True)
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
	sc := theApp.screens[0]
	sz := opts.Size
	if sc.DevicePixelRatio != 1.0 {
		sz.X = int(float32(sz.X) / sc.DevicePixelRatio)
		sz.Y = int(float32(sz.Y) / sc.DevicePixelRatio)
	}
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
			f.f()
			if f.done != nil {
				f.done <- true
			}
		case <-w.publish:
			theApp.RunOnMain(func() {
				w.Activate()
				w.glw.SwapBuffers() // note: implicitly does a flush
				// note: generally don't need this:
				// gpu.Draw.Clear(true, true)
			})
			w.publishDone <- struct{}{}
		}
	}
}

// RunOnWin runs given function on the window's unique locked thread.
func (w *windowImpl) RunOnWin(f func()) {
	done := make(chan bool)
	w.runQueue <- funcRun{f: f, done: done}
	<-done
}

// GoRunOnWin runs given function on window's unique locked thread and returns immediately
func (w *windowImpl) GoRunOnWin(f func()) {
	go func() {
		w.runQueue <- funcRun{f: f, done: nil}
	}()
}

// Publish does the equivalent of SwapBuffers on OpenGL: pushes the
// current rendered back-buffer to the front (and ensures that any
// ongoing rendering has completed) (see also PublishTex)
func (w *windowImpl) Publish() {
	glfw.PostEmptyEvent()
	w.publish <- struct{}{}
	<-w.publishDone
	glfw.PostEmptyEvent()
}

// PublishTex draws the current WinTex texture to the window and then
// calls Publish() -- this is the typical update call.
func (w *windowImpl) PublishTex() {
	theApp.RunOnMain(func() {
		w.Activate()
		w.Copy(image.ZP, w.winTex, w.winTex.Bounds(), oswin.Src, nil)
	})
	w.Publish()
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
	var err error
	theApp.RunOnMain(func() {
		w.Activate()
		wt := w.winTex
		err = wt.SetSubImage(dp, src, sr)
	})
	return err
}

////////////////////////////////////////////////
//   Drawer wrappers

func (w *windowImpl) Draw(src2dst mat32.Mat3, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	theApp.RunOnMain(func() {
		w.Activate()
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
	theApp.RunOnMain(func() {
		w.Activate()
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
	drawer.Copy(w, dp, src, sr, op, opts)
}

func (w *windowImpl) Scale(dr image.Rectangle, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Scale(w, dr, src, sr, op, opts)
}

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	theApp.RunOnMain(func() {
		w.Activate()
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
	if w.Scrn == nil {
		w.getScreen()
	}
	if w.Scrn == nil {
		return theApp.screens[0]
	}
	w.mu.Lock()
	sc := w.Scrn
	w.mu.Unlock()
	return sc
}

func (w *windowImpl) Size() image.Point {
	return w.PxSize
}

func (w *windowImpl) WinSize() image.Point {
	return w.WnSize
}

func (w *windowImpl) Position() image.Point {
	w.mu.Lock()
	var ps image.Point
	ps.X, ps.Y = w.glw.GetPos()
	w.Pos = ps
	w.mu.Unlock()
	return ps
}

func (w *windowImpl) PhysicalDPI() float32 {
	w.mu.Lock()
	dpi := w.PhysDPI
	w.mu.Unlock()
	return dpi
}

func (w *windowImpl) LogicalDPI() float32 {
	w.mu.Lock()
	dpi := w.LogDPI
	w.mu.Unlock()
	return dpi
}

func (w *windowImpl) SetLogicalDPI(dpi float32) {
	w.mu.Lock()
	w.LogDPI = dpi
	w.mu.Unlock()
}

func (w *windowImpl) SetTitle(title string) {
	w.Titl = title
	w.app.RunOnMain(func() {
		w.glw.SetTitle(title)
	})
}

func (w *windowImpl) SetSize(sz image.Point) {
	w.app.RunOnMain(func() {
		w.glw.SetSize(sz.X, sz.Y)
	})
}

func (w *windowImpl) SetPos(pos image.Point) {
	w.app.RunOnMain(func() {
		w.glw.SetPos(pos.X, pos.Y)
	})
}

func (w *windowImpl) SetGeom(pos image.Point, sz image.Point) {
	w.app.RunOnMain(func() {
		w.glw.SetSize(sz.X, sz.Y)
		w.glw.SetPos(pos.X, pos.Y)
	})
}

func (w *windowImpl) show() {
	w.app.RunOnMain(func() {
		w.glw.Show()
	})
}

func (w *windowImpl) Raise() {
	w.app.RunOnMain(func() {
		if bitflag.HasAtomic(&w.Flag, int(oswin.Minimized)) {
			w.glw.Restore()
		} else {
			w.glw.Focus()
		}
	})
}

func (w *windowImpl) Minimize() {
	w.app.RunOnMain(func() {
		w.glw.Iconify()
	})
}

func (w *windowImpl) SetCloseReqFunc(fun func(win oswin.Window)) {
	w.closeReqFunc = fun
}

func (w *windowImpl) SetCloseCleanFunc(fun func(win oswin.Window)) {
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
	})
}

func (w *windowImpl) SetMousePos(x, y float64) {
	w.glw.SetCursorPos(x, y)
}

/////////////////////////////////////////////////////////
//  Window Callbacks

func (w *windowImpl) getScreen() {
	w.mu.Lock()
	mon := w.glw.GetMonitor() // this returns nil for windowed windows -- i.e., most windows
	// that is super useless it seems.
	if mon != nil {
		sc := theApp.ScreenByName(mon.GetName())
		if sc != nil {
			w.Scrn = sc
			w.PhysDPI = sc.PhysicalDPI
		} else {
			log.Printf("glos getScreen: could not find screen of name: %v\n", mon.GetName())
			w.Scrn = theApp.screens[0]
			w.PhysDPI = w.Scrn.PhysicalDPI
		}
	} else {
		if len(theApp.screens) > 0 {
			w.Scrn = theApp.screens[0]
			w.PhysDPI = w.Scrn.PhysicalDPI
		}
	}
	if w.LogDPI == 0 {
		w.LogDPI = w.Scrn.LogicalDPI
	}
	w.mu.Unlock()
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
	w.getScreen()
	w.mu.Lock()
	var wsz image.Point
	wsz.X, wsz.Y = w.glw.GetSize()
	w.WnSize = wsz
	var fbsz image.Point
	fbsz.X, fbsz.Y = w.glw.GetFramebufferSize()
	w.PxSize = fbsz
	if w.PxSize != w.WnSize {
		w.DevPixRatio = float32(w.PxSize.X) / float32(w.WnSize.X)
	} else {
		w.DevPixRatio = 1
	}
	if w.DevPixRatio != w.Scrn.DevicePixelRatio {
		rr := w.DevPixRatio / w.Scrn.DevicePixelRatio
		w.Scrn.PhysicalDPI *= rr
		w.Scrn.LogicalDPI *= rr
		w.Scrn.DevicePixelRatio = w.DevPixRatio
	}
	w.PhysDPI = w.Scrn.PhysicalDPI
	w.LogDPI = w.Scrn.LogicalDPI
	w.Activate()
	w.winTex.SetSize(w.PxSize)
	w.mu.Unlock()
	w.sendWindowEvent(window.Resize)
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
