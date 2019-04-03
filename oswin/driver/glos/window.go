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
	"image/color"
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
	publish     chan struct{}
	publishDone chan oswin.PublishResult
	drawDone    chan struct{}
	winClose    chan struct{}

	// glctxMu is mutex for all OpenGL calls, locked in GPU.Context
	glctxMu sync.Mutex

	// backBufferBound is whether the default Framebuffer, with ID 0, also
	// known as the back buffer or the window's Framebuffer, is bound and its
	// viewport is known to equal the window size. It can become false when we
	// bind to a texture's Framebuffer or when the window size changes.
	backBufferBound bool

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

// must be run on main
func newGLWindow(opts *oswin.NewWindowOptions) (*glfw.Window, error) {
	_, _, tool, fullscreen := oswin.WindowFlagsToBool(opts.Flags)
	glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.Visible, glfw.False) // needed to position
	glfw.WindowHint(glfw.Focused, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

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
	win, err := glfw.CreateWindow(opts.Size.X, opts.Size.Y, opts.GetTitle(), nil, nil)
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

func (w *windowImpl) Upload(dp image.Point, src oswin.Image, sr image.Rectangle) {
	originalSRMin := sr.Min
	sr = sr.Intersect(src.Bounds())
	if sr.Empty() {
		return
	}
	dp = dp.Add(sr.Min.Sub(originalSRMin))
	// TODO: keep a texture around for this purpose?
	t, err := w.app.NewTexture(w, sr.Size())
	if err != nil {
		panic(err)
	}
	t.Upload(image.Point{}, src, sr)
	w.Draw(f64.Aff3{
		1, 0, float64(dp.X),
		0, 1, float64(dp.Y),
	}, t, t.Bounds(), draw.Src, nil)
	t.Release()
}

func useOp(op draw.Op) {
	if op == draw.Over {
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
	} else {
		gl.Disable(gl.BLEND)
	}
}

func (w *windowImpl) bindBackBuffer() {
	w.mu.Lock()
	size := w.Sz
	w.mu.Unlock()

	w.backBufferBound = true
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, int32(size.X), int32(size.Y))
}

func (w *windowImpl) fill(mvp f64.Aff3, src color.Color, op draw.Op) {
	theGPU.UseContext(w)
	defer theGPU.ClearContext(w)

	if !w.backBufferBound {
		w.bindBackBuffer()
	}

	doFill(w.app, mvp, src, op)
}

func doFill(app *appImpl, mvp f64.Aff3, src color.Color, op draw.Op) {
	useOp(op)
	gl.UseProgram(app.fill.program)

	writeAff3(app.fill.mvp, mvp)

	r, g, b, a := src.RGBA()
	gl.Uniform4f(
		app.fill.color,
		float32(r)/65535,
		float32(g)/65535,
		float32(b)/65535,
		float32(a)/65535,
	)

	gl.BindBuffer(gl.ARRAY_BUFFER, app.fill.quad)
	gl.EnableVertexAttribArray(uint32(app.fill.pos))
	gl.VertexAttribPointer(uint32(app.fill.pos), 2, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.DisableVertexAttribArray(uint32(app.fill.pos))
}

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	minX := float64(dr.Min.X)
	minY := float64(dr.Min.Y)
	maxX := float64(dr.Max.X)
	maxY := float64(dr.Max.Y)
	w.fill(w.mvp(
		minX, minY,
		maxX, minY,
		minX, maxY,
	), src, op)
}

func (w *windowImpl) DrawUniform(src2dst f64.Aff3, src color.Color, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	minX := float64(sr.Min.X)
	minY := float64(sr.Min.Y)
	maxX := float64(sr.Max.X)
	maxY := float64(sr.Max.Y)
	w.fill(w.mvp(
		src2dst[0]*minX+src2dst[1]*minY+src2dst[2],
		src2dst[3]*minX+src2dst[4]*minY+src2dst[5],
		src2dst[0]*maxX+src2dst[1]*minY+src2dst[2],
		src2dst[3]*maxX+src2dst[4]*minY+src2dst[5],
		src2dst[0]*minX+src2dst[1]*maxY+src2dst[2],
		src2dst[3]*minX+src2dst[4]*maxY+src2dst[5],
	), src, op)
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	t := src.(*textureImpl)
	sr = sr.Intersect(t.Bounds())
	if sr.Empty() {
		return
	}

	theGPU.UseContext(w)
	defer theGPU.ClearContext(w)

	if !w.backBufferBound {
		w.bindBackBuffer()
	}

	useOp(op)
	gl.UseProgram(w.app.texture.program)

	// Start with src-space left, top, right and bottom.
	srcL := float64(sr.Min.X)
	srcT := float64(sr.Min.Y)
	srcR := float64(sr.Max.X)
	srcB := float64(sr.Max.Y)
	// Transform to dst-space via the src2dst matrix, then to a MVP matrix.
	writeAff3(w.app.texture.mvp, w.mvp(
		src2dst[0]*srcL+src2dst[1]*srcT+src2dst[2],
		src2dst[3]*srcL+src2dst[4]*srcT+src2dst[5],
		src2dst[0]*srcR+src2dst[1]*srcT+src2dst[2],
		src2dst[3]*srcR+src2dst[4]*srcT+src2dst[5],
		src2dst[0]*srcL+src2dst[1]*srcB+src2dst[2],
		src2dst[3]*srcL+src2dst[4]*srcB+src2dst[5],
	))

	// OpenGL's fragment shaders' UV coordinates run from (0,0)-(1,1),
	// unlike vertex shaders' XY coordinates running from (-1,+1)-(+1,-1).
	//
	// We are drawing a rectangle PQRS, defined by two of its
	// corners, onto the entire texture. The two quads may actually
	// be equal, but in the general case, PQRS can be smaller.
	//
	//	(0,0) +---------------+ (1,0)
	//	      |  P +-----+ Q  |
	//	      |    |     |    |
	//	      |  S +-----+ R  |
	//	(0,1) +---------------+ (1,1)
	//
	// The PQRS quad is always axis-aligned. First of all, convert
	// from pixel space to texture space.
	tw := float64(t.size.X)
	th := float64(t.size.Y)
	px := float64(sr.Min.X-0) / tw
	py := float64(sr.Min.Y-0) / th
	qx := float64(sr.Max.X-0) / tw
	sy := float64(sr.Max.Y-0) / th
	// Due to axis alignment, qy = py and sx = px.
	//
	// The simultaneous equations are:
	//	  0 +   0 + a02 = px
	//	  0 +   0 + a12 = py
	//	a00 +   0 + a02 = qx
	//	a10 +   0 + a12 = qy = py
	//	  0 + a01 + a02 = sx = px
	//	  0 + a11 + a12 = sy
	writeAff3(w.app.texture.uvp, f64.Aff3{
		qx - px, 0, px,
		0, sy - py, py,
	})

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, t.id)
	gl.Uniform1i(w.app.texture.sample, 0)

	gl.BindBuffer(gl.ARRAY_BUFFER, w.app.texture.quad)
	gl.EnableVertexAttribArray(w.app.texture.pos)
	gl.VertexAttribPointer(w.app.texture.pos, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

	gl.BindBuffer(gl.ARRAY_BUFFER, w.app.texture.quad)
	gl.EnableVertexAttribArray(w.app.texture.inUV)
	gl.VertexAttribPointer(w.app.texture.inUV, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	gl.DisableVertexAttribArray(w.app.texture.pos)
	gl.DisableVertexAttribArray(w.app.texture.inUV)
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

// drawLoop is the primary drawing loop.
func (w *windowImpl) drawLoop() {
	runtime.LockOSThread()

	// Starting in OS X 10.11 (El Capitan), the vertex array is
	// occasionally getting unbound when the context changes threads.
	//
	// Avoid this by binding it again.
	// 	C.glBindVertexArray(C.GLuint(vba))
	// if errno := C.glGetError(); errno != 0 {
	// 	panic(fmt.Sprintf("macdriver: glBindVertexArray failed: %d", errno))
	// }
	//
	// workAvailable := w.worker.WorkAvailable()

outer:
	for {
		select {
		case <-w.winClose:
			break outer
		case <-w.publish:
			theGPU.UseContext(w)
			gl.Flush()
			theGPU.ClearContext(w)
			w.publishDone <- oswin.PublishResult{}
		}
	}
}

func (w *windowImpl) Publish() oswin.PublishResult {
	// gl.Flush is a lightweight (on modern GL drivers) blocking call
	// that ensures all GL functions pending in the gl package have
	// been passed onto the GL driver before the app package attempts
	// to swap the buffer.
	//
	// This enforces that the final receive (for this paint cycle) on
	// gl.WorkAvailable happens before the send on publish.
	// theGPU.UseContext(w)
	// gl.Flush()
	// theGPU.ClearContext(w)

	w.publish <- struct{}{}
	res := <-w.publishDone

	select {
	case w.drawDone <- struct{}{}:
	default:
	}

	return res
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
	w.Publish()
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
