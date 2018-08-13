// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gldriver

import (
	"fmt"
	"image"
	"os"
	"sync"

	"github.com/goki/gi/oswin"
	"golang.org/x/mobile/gl"
)

var theApp = &appImpl{
	windows: make(map[uintptr]*windowImpl),
	winlist: make([]*windowImpl, 0),
	screens: make([]*oswin.Screen, 0),
	name:    "GoGi",
}

type appImpl struct {
	texture struct {
		program gl.Program
		pos     gl.Attrib
		mvp     gl.Uniform
		uvp     gl.Uniform
		inUV    gl.Attrib
		sample  gl.Uniform
		quad    gl.Buffer
	}
	fill struct {
		program gl.Program
		pos     gl.Attrib
		mvp     gl.Uniform
		color   gl.Uniform
		quad    gl.Buffer
	}

	mu            sync.Mutex
	windows       map[uintptr]*windowImpl
	winlist       []*windowImpl
	screens       []*oswin.Screen
	name          string
	about         string
	quitReqFunc   func()
	quitCleanFunc func()
}

func (app *appImpl) NewImage(size image.Point) (retBuf oswin.Image, retErr error) {
	m := image.NewRGBA(image.Rectangle{Max: size})
	return &imageImpl{
		buf:  m.Pix,
		rgba: *m,
		size: size,
	}, nil
}

func (app *appImpl) NewTexture(win oswin.Window, size image.Point) (oswin.Texture, error) {
	// TODO: can we compile these programs eagerly instead of lazily?

	w := win.(*windowImpl)

	w.glctxMu.Lock()
	defer w.glctxMu.Unlock()
	glctx := w.glctx
	if glctx == nil {
		return nil, fmt.Errorf("gldriver: no GL context available")
	}

	if !glctx.IsProgram(app.texture.program) {
		p, err := compileProgram(glctx, textureVertexSrc, textureFragmentSrc)
		if err != nil {
			return nil, err
		}
		app.texture.program = p
		app.texture.pos = glctx.GetAttribLocation(p, "pos")
		app.texture.mvp = glctx.GetUniformLocation(p, "mvp")
		app.texture.uvp = glctx.GetUniformLocation(p, "uvp")
		app.texture.inUV = glctx.GetAttribLocation(p, "inUV")
		app.texture.sample = glctx.GetUniformLocation(p, "sample")
		app.texture.quad = glctx.CreateBuffer()

		glctx.BindBuffer(gl.ARRAY_BUFFER, app.texture.quad)
		glctx.BufferData(gl.ARRAY_BUFFER, quadCoords, gl.STATIC_DRAW)
	}

	t := &textureImpl{
		w:    w,
		id:   glctx.CreateTexture(),
		size: size,
	}

	glctx.BindTexture(gl.TEXTURE_2D, t.id)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, size.X, size.Y, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	return t, nil
}

func (app *appImpl) NewWindow(opts *oswin.NewWindowOptions) (oswin.Window, error) {
	if opts == nil {
		opts = &oswin.NewWindowOptions{}
	}
	opts.Fixup()
	// can also apply further tuning here..

	id, err := newWindow(opts)
	if err != nil {
		return nil, err
	}
	w := &windowImpl{
		app:         app,
		id:          id,
		publish:     make(chan struct{}),
		publishDone: make(chan oswin.PublishResult),
		drawDone:    make(chan struct{}),
	}
	initWindow(w)

	if opts.Title != "" {
		w.SetTitle(opts.Title)
	}

	app.mu.Lock()
	app.windows[id] = w
	app.winlist = append(app.winlist, w)
	app.mu.Unlock()

	// todo: could try to find alternative screen number here..
	sc := app.Screen(0)
	w.PhysDPI = sc.PhysicalDPI
	w.LogDPI = sc.LogicalDPI
	w.Scrn = sc
	w.Flag = opts.Flags

	showWindow(w)

	return w, nil
}

func (app *appImpl) DeleteWin(id uintptr) {
	app.mu.Lock()
	defer app.mu.Unlock()
	_, ok := app.windows[id]
	if !ok {
		return
	}
	for i, w := range app.winlist {
		if w.id == id {
			app.winlist = append(app.winlist[:i], app.winlist[i+1:]...)
			break
		}
	}
	delete(app.windows, id)
}

func (app *appImpl) NScreens() int {
	return len(app.screens)
}

func (app *appImpl) Screen(scrN int) *oswin.Screen {
	sz := len(app.screens)
	if scrN < sz {
		return app.screens[scrN]
	}
	return nil
}

func (app *appImpl) NWindows() int {
	return len(app.winlist)
}

func (app *appImpl) Window(win int) oswin.Window {
	sz := len(app.winlist)
	if win < sz {
		return app.winlist[win]
	}
	return nil
}

func (app *appImpl) WindowByName(name string) oswin.Window {
	for _, win := range app.winlist {
		if win.Name() == name {
			return win
		}
	}
	return nil
}

func (app *appImpl) Name() string {
	return app.name
}

func (app *appImpl) SetName(name string) {
	app.name = name
}

func (app *appImpl) About() string {
	return app.about
}

func (app *appImpl) SetAbout(about string) {
	app.about = about
}

func (app *appImpl) SetQuitReqFunc(fun func()) {
	app.quitReqFunc = fun
}

func (app *appImpl) SetQuitCleanFunc(fun func()) {
	app.quitCleanFunc = fun
}

func (app *appImpl) QuitReq() {
	if app.quitReqFunc != nil {
		app.quitReqFunc()
	}
}

func (app *appImpl) QuitClean() {
	if app.quitCleanFunc != nil {
		app.quitCleanFunc()
	}
}

func (app *appImpl) Quit() {
	// todo: could try to invoke NSApp terminate method instead
	app.QuitClean()
	for _, win := range app.winlist {
		win.Close()
	}
	os.Exit(0)
}
