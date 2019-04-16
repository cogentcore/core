// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build 3d

package glos

import (
	"image"
	"image/draw"
	"runtime"
	"sync"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/drawer"
	"github.com/goki/gi/oswin/driver/internal/event"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki/bitflag"
	"golang.org/x/image/math/f64"
)

type windowImpl struct {
	oswin.WindowBase
	app *appImpl
	glw *glfw.Window
	event.Deque
	runQueue    chan funcRun
	publish     chan struct{}
	publishDone chan oswin.PublishResult
	drawDone    chan struct{}
	winClose    chan struct{}

	// glctxMu is mutex for all OpenGL calls, locked in GPU.Context
	glctxMu sync.Mutex

	// textures are the textures created for this window -- they are released
	// when the window is closed
	textures map[*textureImpl]struct{}

	// mu is general state mutex. If you need to hold both glctxMu and mu,
	// the lock ordering is to lock glctxMu first (and unlock it last).
	mu sync.Mutex

	// mainMenu is the main menu associated with window, if applicable.
	mainMenu oswin.MainMenu

	closeReqFunc   func(win oswin.Window)
	closeCleanFunc func(win oswin.Window)
}

func (w *windowImpl) Handle() interface{} {
	return w.glw
}

// must be run on main
func newGLWindow(opts *oswin.NewWindowOptions) (*glfw.Window, error) {
	_, _, tool, fullscreen := oswin.WindowFlagsToBool(opts.Flags)
	glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Visible, glfw.False) // needed to position
	glfw.WindowHint(glfw.Focused, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4) // 4.1 is max supported on macos
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
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
	win, err := glfw.CreateWindow(opts.Size.X, opts.Size.Y, opts.GetTitle(), nil, theApp.shareWin)
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

// winLoop is the window's own locked processing loop
// all gl processing should be done on this loop by calling RunOnWin
// or GoRunOnWin.
func (w *windowImpl) winLoop() {
	runtime.LockOSThread()
	theGPU.UseContext(w)
	gl.Init() // call to init in each context
	theGPU.ClearContext(w)
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
			theGPU.UseContext(w)
			w.glw.SwapBuffers() // note: implicitly does a flush
			// note: generally don't need this:
			// theGPU.Clear(true, true)
			theGPU.ClearContext(w)
			w.publishDone <- oswin.PublishResult{}
		}
	}
}

// RunOnWin runs given function on window's unique locked thread
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

func (w *windowImpl) Publish() oswin.PublishResult {
	w.publish <- struct{}{}
	res := <-w.publishDone

	select {
	case w.drawDone <- struct{}{}:
	default:
	}

	return res
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	w.RunOnWin(func() {
		w.draw(src2dst, src, sr, op, opts)
	})
}

func (w *windowImpl) Copy(dp image.Point, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Copy(w, dp, src, sr, op, opts)
}

func (w *windowImpl) Scale(dr image.Rectangle, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Scale(w, dr, src, sr, op, opts)
}

func (w *windowImpl) mvp(tlx, tly, trx, try, blx, bly float64) f64.Aff3 {
	w.mu.Lock()
	size := w.Sz
	w.mu.Unlock()

	return calcMVP(size.X, size.Y, tlx, tly, trx, try, blx, bly)
}

// calcMVP returns the Model View Projection matrix that maps the quadCoords
// unit square, (0, 0) to (1, 1), to a quad QV, such that QV in vertex shader
// space corresponds to the quad QP in pixel space, where QP is defined by
// three of its four corners - the arguments to this function. The three
// corners are nominally the top-left, top-right and bottom-left, but there is
// no constraint that e.g. tlx < trx.
//
// In pixel space, the window ranges from (0, 0) to (widthPx, heightPx). The
// Y-axis points downwards.
//
// In vertex shader space, the window ranges from (-1, +1) to (+1, -1), which
// is a 2-unit by 2-unit square. The Y-axis points upwards.
func calcMVP(widthPx, heightPx int, tlx, tly, trx, try, blx, bly float64) f64.Aff3 {
	// Convert from pixel coords to vertex shader coords.
	invHalfWidth := +2 / float64(widthPx)
	invHalfHeight := -2 / float64(heightPx)
	tlx = tlx*invHalfWidth - 1
	tly = tly*invHalfHeight + 1
	trx = trx*invHalfWidth - 1
	try = try*invHalfHeight + 1
	blx = blx*invHalfWidth - 1
	bly = bly*invHalfHeight + 1

	// The resultant affine matrix:
	//	- maps (0, 0) to (tlx, tly).
	//	- maps (1, 0) to (trx, try).
	//	- maps (0, 1) to (blx, bly).
	return f64.Aff3{
		trx - tlx, blx - tlx, tlx,
		try - tly, bly - tly, tly,
	}
}

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
	w.mu.Lock()
	var sz image.Point
	sz.X, sz.Y = w.glw.GetSize()
	w.Sz = sz
	w.mu.Unlock()
	return sz
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

func (w *windowImpl) MainMenu() oswin.MainMenu {
	return nil
	// if w.mainMenu == nil {
	// 	mm := &mainMenuImpl{win: w}
	// 	w.mainMenu = mm
	// }
	// return w.mainMenu.(*mainMenuImpl)
}

func (w *windowImpl) show() {
	w.app.RunOnMain(func() {
		w.glw.Show()
	})
}

func (w *windowImpl) Raise() {
	w.app.RunOnMain(func() {
		w.glw.Restore()
	})
}

func (w *windowImpl) Minimize() {
	w.app.RunOnMain(func() {
		w.glw.Hide()
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

func (w *windowImpl) AddTexture(t *textureImpl) {
	if w.textures == nil {
		w.textures = make(map[*textureImpl]struct{})
	}
	w.textures[t] = struct{}{}
}

// DeleteTexture just deletes it from our list -- does not Release -- is called during t.Release
func (w *windowImpl) DeleteTexture(t *textureImpl) {
	if w.textures == nil {
		return
	}
	delete(w.textures, t)
}

func (w *windowImpl) Close() {
	// this is actually the final common pathway for closing here
	w.winClose <- struct{}{} // break out of draw loop
	w.CloseClean()
	// fmt.Printf("sending close event to window: %v\n", w.Nm)
	w.sendWindowEvent(window.Close)
	if w.textures != nil {
		for t, _ := range w.textures {
			t.Release() // deletes from map
		}
	}
	w.textures = nil
	w.app.RunOnMain(func() {
		w.glw.Destroy()
	})
	// 	closeWindow(w.id)
}

/////////////////////////////////////////////////////////
//  Window Callbacks

func (w *windowImpl) getScreen() {
	w.mu.Lock()
	mon := w.glw.GetMonitor() // this returns nil for windowed windows -- i.e., most windows
	// that is super useless it seems.
	if mon != nil {
		sc := theApp.ScreenByName(mon.GetName())
		w.Scrn = sc
		w.PhysDPI = sc.PhysicalDPI
	} else {
		w.Scrn = theApp.screens[0]
		w.PhysDPI = w.Scrn.PhysicalDPI
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
	w.mu.Lock()
	w.Sz = image.Point{width, height}
	w.mu.Unlock()
	w.getScreen()
	w.sendWindowEvent(window.Resize)
}

func (w *windowImpl) fbResized(gw *glfw.Window, width, height int) {
}

func (w *windowImpl) closeReq(gw *glfw.Window) {
	go w.CloseReq()
}

func (w *windowImpl) refresh(gw *glfw.Window) {
	go w.Publish()
}

func (w *windowImpl) focus(gw *glfw.Window, focused bool) {
	if focused {
		bitflag.ClearAtomic(&w.Flag, int(oswin.Minimized))
		bitflag.SetAtomic(&w.Flag, int(oswin.Focus))
		w.sendWindowEvent(window.Focus)
	} else {
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
