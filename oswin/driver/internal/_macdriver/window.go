// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build !3d

package macdriver

import (
	"image"
	"image/color"
	"image/draw"
	"sync"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/drawer"
	"github.com/goki/gi/oswin/driver/internal/event"
	"github.com/goki/gi/oswin/window"
	"golang.org/x/image/math/f64"
	"golang.org/x/mobile/gl"
)

type windowImpl struct {
	oswin.WindowBase

	app *appImpl

	// id is an OS-specific data structure for the window.
	//	- Cocoa:   AppGLView*
	//	- X11:     Window
	//	- Windows: win32.HWND
	id uintptr

	// ctx is a C data structure for the GL context.
	//	- Cocoa:   uintptr holding a NSOpenGLContext*.
	//	- X11:     uintptr holding an EGLSurface.
	//	- Windows: ctxWin32
	ctx interface{}

	event.Deque
	publish     chan struct{}
	publishDone chan oswin.PublishResult
	drawDone    chan struct{}
	winClose    chan struct{}

	// glctxMu is a mutex that enforces the atomicity of methods like
	// Texture.Upload or Window.Draw that are conceptually one operation
	// but are implemented by multiple OpenGL calls. OpenGL is a stateful
	// API, so interleaving OpenGL calls from separate higher-level
	// operations causes inconsistencies.
	glctxMu sync.Mutex
	glctx   gl.Context
	worker  gl.Worker
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

// for sending any kind of event
func sendEvent(id uintptr, ev oswin.Event) {
	theApp.mu.Lock()
	w := theApp.windows[id]
	theApp.mu.Unlock()
	if w == nil {
		return
	}
	ev.Init()
	w.Send(ev)
}

// for sending window.Event's
func sendWindowEvent(w *windowImpl, act window.Actions) {
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

func useOp(glctx gl.Context, op draw.Op) {
	if op == draw.Over {
		glctx.Enable(gl.BLEND)
		glctx.BlendFunc(gl.ONE, gl.ONE_MINUS_SRC_ALPHA)
	} else {
		glctx.Disable(gl.BLEND)
	}
}

func (w *windowImpl) bindBackBuffer() {
	w.mu.Lock()
	size := w.Sz
	w.mu.Unlock()

	w.backBufferBound = true
	w.glctx.BindFramebuffer(gl.FRAMEBUFFER, gl.Framebuffer{Value: 0})
	w.glctx.Viewport(0, 0, size.X, size.Y)
}

func (w *windowImpl) fill(mvp f64.Aff3, src color.Color, op draw.Op) {
	w.glctxMu.Lock()
	defer w.glctxMu.Unlock()

	if !w.backBufferBound {
		w.bindBackBuffer()
	}

	doFill(w.app, w.glctx, mvp, src, op)
}

func doFill(app *appImpl, glctx gl.Context, mvp f64.Aff3, src color.Color, op draw.Op) {
	useOp(glctx, op)
	if !glctx.IsProgram(app.fill.program) {
		p, err := compileProgram(glctx, fillVertexSrc, fillFragmentSrc)
		if err != nil {
			// TODO: initialize this somewhere else we can better handle the error.
			panic(err.Error())
		}
		app.fill.program = p
		app.fill.pos = glctx.GetAttribLocation(p, "pos")
		app.fill.mvp = glctx.GetUniformLocation(p, "mvp")
		app.fill.color = glctx.GetUniformLocation(p, "color")
		app.fill.quad = glctx.CreateBuffer()

		glctx.BindBuffer(gl.ARRAY_BUFFER, app.fill.quad)
		glctx.BufferData(gl.ARRAY_BUFFER, quadCoords, gl.STATIC_DRAW)
	}
	glctx.UseProgram(app.fill.program)

	writeAff3(glctx, app.fill.mvp, mvp)

	r, g, b, a := src.RGBA()
	glctx.Uniform4f(
		app.fill.color,
		float32(r)/65535,
		float32(g)/65535,
		float32(b)/65535,
		float32(a)/65535,
	)

	glctx.BindBuffer(gl.ARRAY_BUFFER, app.fill.quad)
	glctx.EnableVertexAttribArray(app.fill.pos)
	glctx.VertexAttribPointer(app.fill.pos, 2, gl.FLOAT, false, 0, 0)

	glctx.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	glctx.DisableVertexAttribArray(app.fill.pos)
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

	w.glctxMu.Lock()
	defer w.glctxMu.Unlock()

	if !w.backBufferBound {
		w.bindBackBuffer()
	}

	useOp(w.glctx, op)
	w.glctx.UseProgram(w.app.texture.program)

	// Start with src-space left, top, right and bottom.
	srcL := float64(sr.Min.X)
	srcT := float64(sr.Min.Y)
	srcR := float64(sr.Max.X)
	srcB := float64(sr.Max.Y)
	// Transform to dst-space via the src2dst matrix, then to a MVP matrix.
	writeAff3(w.glctx, w.app.texture.mvp, w.mvp(
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
	writeAff3(w.glctx, w.app.texture.uvp, f64.Aff3{
		qx - px, 0, px,
		0, sy - py, py,
	})

	w.glctx.ActiveTexture(gl.TEXTURE0)
	w.glctx.BindTexture(gl.TEXTURE_2D, t.id)
	w.glctx.Uniform1i(w.app.texture.sample, 0)

	w.glctx.BindBuffer(gl.ARRAY_BUFFER, w.app.texture.quad)
	w.glctx.EnableVertexAttribArray(w.app.texture.pos)
	w.glctx.VertexAttribPointer(w.app.texture.pos, 2, gl.FLOAT, false, 0, 0)

	w.glctx.BindBuffer(gl.ARRAY_BUFFER, w.app.texture.quad)
	w.glctx.EnableVertexAttribArray(w.app.texture.inUV)
	w.glctx.VertexAttribPointer(w.app.texture.inUV, 2, gl.FLOAT, false, 0, 0)

	w.glctx.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	w.glctx.DisableVertexAttribArray(w.app.texture.pos)
	w.glctx.DisableVertexAttribArray(w.app.texture.inUV)
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

func (w *windowImpl) Publish() oswin.PublishResult {
	// gl.Flush is a lightweight (on modern GL drivers) blocking call
	// that ensures all GL functions pending in the gl package have
	// been passed onto the GL driver before the app package attempts
	// to swap the buffer.
	//
	// This enforces that the final receive (for this paint cycle) on
	// gl.WorkAvailable happens before the send on publish.
	w.glctxMu.Lock()
	w.glctx.Flush()
	w.glctxMu.Unlock()

	w.publish <- struct{}{}
	res := <-w.publishDone

	select {
	case w.drawDone <- struct{}{}:
	default:
	}

	return res
}

func (w *windowImpl) Screen() *oswin.Screen {
	w.mu.Lock()
	sc := w.Scrn
	w.mu.Unlock()
	return sc
}

func (w *windowImpl) Size() image.Point {
	w.mu.Lock()
	sz := w.Sz
	w.mu.Unlock()
	return sz
}

func (w *windowImpl) Position() image.Point {
	w.mu.Lock()
	ps := w.Pos
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
	updateTitle(w, title)
}

func (w *windowImpl) SetSize(sz image.Point) {
	resizeWindow(w, sz)
}

func (w *windowImpl) SetPos(pos image.Point) {
	posWindow(w, pos)
}

func (w *windowImpl) SetGeom(pos image.Point, sz image.Point) {
	setGeomWindow(w, pos, sz)
}

func (w *windowImpl) MainMenu() oswin.MainMenu {
	if w.mainMenu == nil {
		mm := &mainMenuImpl{win: w}
		w.mainMenu = mm
	}
	return w.mainMenu.(*mainMenuImpl)
}

func (w *windowImpl) Raise() {
	raiseWindow(w)
}

func (w *windowImpl) Minimize() {
	minimizeWindow(w)
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
	sendWindowEvent(w, window.Close)
	if w.textures != nil {
		for t := range w.textures {
			t.Release() // deletes from map
		}
	}
	w.textures = nil
	closeWindow(w.id)
}
