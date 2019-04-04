// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build 3d

package glos

import (
	"fmt"
	"go/build"
	"image"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/clip"
	"github.com/goki/gi/oswin/cursor"
)

func init() {
	runtime.LockOSThread()
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
}

var theApp = &appImpl{
	windows:      make(map[*glfw.Window]*windowImpl),
	winlist:      make([]*windowImpl, 0),
	screens:      make([]*oswin.Screen, 0),
	name:         "GoGi",
	quitCloseCnt: make(chan struct{}),
}

type appImpl struct {
	texture struct {
		init    bool
		program uint32
		pos     uint32
		mvp     int32
		uvp     int32
		inUV    uint32
		sample  int32
		quad    uint32
	}
	fill struct {
		program uint32
		pos     int32
		mvp     int32
		color   int32
		quad    uint32
	}

	mu            sync.Mutex
	mainQueue     chan funcRun
	mainDone      chan struct{}
	windows       map[*glfw.Window]*windowImpl
	winlist       []*windowImpl
	screens       []*oswin.Screen
	ctxtwin       *windowImpl
	name          string
	about         string
	quitting      bool          // set to true when quitting and closing windows
	quitCloseCnt  chan struct{} // counts windows to make sure all are closed before done
	quitReqFunc   func()
	quitCleanFunc func()
}

var mainCallback func(oswin.App)

// Main is called from main thread when it is time to start running the
// main loop.  When function f returns, the app ends automatically.
func Main(f func(oswin.App)) {
	mainCallback = f
	theApp.getScreens()
	oswin.TheApp = theApp
	go func() {
		mainCallback(theApp)
		theApp.stopMain()
	}()
	theApp.mainLoop()
}

type funcRun struct {
	f    func()
	done chan bool
}

// RunOnMain runs given function on main thread
func (app *appImpl) RunOnMain(f func()) {
	if app.mainQueue == nil {
		f()
	} else {
		glfw.PostEmptyEvent()
		done := make(chan bool)
		app.mainQueue <- funcRun{f: f, done: done}
		<-done
	}
}

// GoRunOnMain runs given function on main thread and returns immediately
func (app *appImpl) GoRunOnMain(f func()) {
	go func() {
		glfw.PostEmptyEvent()
		app.mainQueue <- funcRun{f: f, done: nil}
	}()
}

// MainLoop starts running event loop on main thread (must be called
// from the main thread).
func (app *appImpl) mainLoop() {
	app.mainQueue = make(chan funcRun)
	app.mainDone = make(chan struct{})
	for {
		select {
		case <-app.mainDone:
			glfw.Terminate()
			return
		case f := <-app.mainQueue:
			f.f()
			if f.done != nil {
				f.done <- true
			}
		default:
			if len(app.windows) == 0 {
				time.Sleep(1)
			} else {
				glfw.WaitEvents()
			}
		}
	}
}

// stopMain stops the main loop and thus terminates the app
func (app *appImpl) stopMain() {
	app.mainDone <- struct{}{}
}

////////////////////////////////////////////////////////
//  Window

func (app *appImpl) NewWindow(opts *oswin.NewWindowOptions) (oswin.Window, error) {
	if opts == nil {
		opts = &oswin.NewWindowOptions{}
	}
	opts.Fixup()
	// can also apply further tuning here..

	var glw *glfw.Window
	var err error
	app.RunOnMain(func() {
		glw, err = newGLWindow(opts)
	})
	if err != nil {
		return nil, err
	}
	w := &windowImpl{
		app:         app,
		glw:         glw,
		publish:     make(chan struct{}),
		winClose:    make(chan struct{}),
		publishDone: make(chan oswin.PublishResult),
		drawDone:    make(chan struct{}),
		WindowBase: oswin.WindowBase{
			Titl: opts.GetTitle(),
			Flag: opts.Flags,
		},
	}

	app.mu.Lock()
	app.windows[glw] = w
	app.winlist = append(app.winlist, w)
	app.mu.Unlock()

	if !app.texture.init {
		app.RunOnMain(func() {
			theGPU.UseContext(w)
			app.initGLPrograms()
			theGPU.ClearContext(w)
		})
	}

	glw.SetPosCallback(w.moved)
	glw.SetSizeCallback(w.winResized)
	glw.SetFramebufferSizeCallback(w.fbResized)
	glw.SetCloseCallback(w.closeReq)
	glw.SetRefreshCallback(w.refresh)
	glw.SetFocusCallback(w.focus)
	glw.SetIconifyCallback(w.iconify)

	glw.SetKeyCallback(w.keyEvent)
	glw.SetCharModsCallback(w.charEvent)
	glw.SetMouseButtonCallback(w.mouseButtonEvent)
	glw.SetScrollCallback(w.scrollEvent)
	glw.SetCursorPosCallback(w.cursorPosEvent)
	glw.SetCursorEnterCallback(w.cursorEnterEvent)
	glw.SetDropCallback(w.dropEvent)

	go w.drawLoop() // monitors for publish events

	w.getScreen()
	w.show() // todo: need to raise window too -- not supported in glfw

	return w, nil
}

func (app *appImpl) DeleteWin(glw *glfw.Window) {
	app.mu.Lock()
	defer app.mu.Unlock()
	_, ok := app.windows[glw]
	if !ok {
		return
	}
	for i, w := range app.winlist {
		if w.glw == glw {
			app.winlist = append(app.winlist[:i], app.winlist[i+1:]...)
			break
		}
	}
	delete(app.windows, glw)
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

func (app *appImpl) ScreenByName(name string) *oswin.Screen {
	for _, sc := range app.screens {
		if sc.Name == name {
			return sc
		}
	}
	return nil
}

func (app *appImpl) NWindows() int {
	app.mu.Lock()
	defer app.mu.Unlock()
	return len(app.winlist)
}

func (app *appImpl) Window(win int) oswin.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	sz := len(app.winlist)
	if win < sz {
		return app.winlist[win]
	}
	return nil
}

func (app *appImpl) WindowByName(name string) oswin.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	for _, win := range app.winlist {
		if win.Name() == name {
			return win
		}
	}
	return nil
}

func (app *appImpl) WindowInFocus() oswin.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	for _, win := range app.winlist {
		if win.IsFocus() {
			return win
		}
	}
	return nil
}

func (app *appImpl) ContextWindow() oswin.Window {
	app.mu.Lock()
	cw := app.ctxtwin
	app.mu.Unlock()
	return cw
}

func (app *appImpl) NewImage(size image.Point) (retBuf oswin.Image, retErr error) {
	m := image.NewRGBA(image.Rectangle{Max: size})
	return &imageImpl{
		buf:  m.Pix,
		rgba: *m,
		size: size,
	}, nil
}

func (app *appImpl) initGLPrograms() error {
	if app.texture.init {
		return nil
	}
	if err := gl.Init(); err != nil {
		return err
	}
	gl.Enable(gl.DEBUG_OUTPUT)
	p, err := theGPU.NewProgram(textureVertexSrc, textureFragmentSrc)
	if err != nil {
		return err
	}
	app.texture.program = p
	app.texture.pos = uint32(gl.GetAttribLocation(p, gl.Str("pos\x00")))
	app.texture.mvp = gl.GetUniformLocation(p, gl.Str("mvp\x00"))
	app.texture.uvp = gl.GetUniformLocation(p, gl.Str("uvp\x00"))
	app.texture.inUV = uint32(gl.GetAttribLocation(p, gl.Str("inUV\x00")))
	app.texture.sample = gl.GetUniformLocation(p, gl.Str("sample\x00"))

	gl.GenBuffers(1, &app.texture.quad)
	gl.BindBuffer(gl.ARRAY_BUFFER, app.texture.quad)
	gl.BufferData(gl.ARRAY_BUFFER, len(quadCoords)*4, gl.Ptr(quadCoords), gl.STATIC_DRAW)

	p, err = theGPU.NewProgram(fillVertexSrc, fillFragmentSrc)
	if err != nil {
		return err
	}
	app.fill.program = p
	app.fill.pos = gl.GetAttribLocation(p, gl.Str("pos\x00"))
	app.fill.mvp = gl.GetUniformLocation(p, gl.Str("mvp\x00"))
	app.fill.color = gl.GetUniformLocation(p, gl.Str("color\x00"))
	gl.GenBuffers(1, &app.fill.quad)

	gl.BindBuffer(gl.ARRAY_BUFFER, app.fill.quad)
	gl.BufferData(gl.ARRAY_BUFFER, len(quadCoords)*4, gl.Ptr(quadCoords), gl.STATIC_DRAW)

	app.texture.init = true
	return nil
}

func (app *appImpl) NewTexture(win oswin.Window, size image.Point) (oswin.Texture, error) {
	var t oswin.Texture
	var err error
	app.RunOnMain(func() {
		t, err = app.newTexture(win, size)
	})
	return t, err
}

func (app *appImpl) newTexture(win oswin.Window, size image.Point) (oswin.Texture, error) {
	w := win.(*windowImpl)

	theGPU.UseContext(w)
	defer theGPU.ClearContext(w)

	var tex uint32
	gl.GenTextures(1, &tex)

	t := &textureImpl{
		w:    w,
		id:   tex,
		size: size,
	}

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, t.id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(size.X),
		int32(size.Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		nil)

	w.AddTexture(t)

	return t, nil
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

func (app *appImpl) Platform() oswin.Platforms {
	return oswin.MacOS
}

func (app *appImpl) PrefsDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Print(err)
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, "Library")
}

func (app *appImpl) GoGiPrefsDir() string {
	pdir := filepath.Join(app.PrefsDir(), "GoGi")
	os.MkdirAll(pdir, 0755)
	return pdir
}

func (app *appImpl) AppPrefsDir() string {
	pdir := filepath.Join(app.PrefsDir(), app.Name())
	os.MkdirAll(pdir, 0755)
	return pdir
}

// SrcDir tries to locate dir in GOPATH/src/ or GOROOT/src/pkg/ and returns its
// full path. GOPATH may contain a list of paths.  From Robin Elkind github.com/mewkiz/pkg
func SrcDir(dir string) (absDir string, err error) {
	for _, srcDir := range build.Default.SrcDirs() {
		absDir = filepath.Join(srcDir, dir)
		finfo, err := os.Stat(absDir)
		if err == nil && finfo.IsDir() {
			return absDir, nil
		}
	}
	return "", fmt.Errorf("unable to locate directory (%q) in GOPATH/src/ (%q) or GOROOT/src/pkg/ (%q)", dir, os.Getenv("GOPATH"), os.Getenv("GOROOT"))
}

func (app *appImpl) FontPaths() []string {
	return []string{"/System/Library/Fonts", "/Library/Fonts"}
}

func (app *appImpl) ClipBoard(win oswin.Window) clip.Board {
	app.mu.Lock()
	app.ctxtwin = win.(*windowImpl)
	app.mu.Unlock()
	return &theClip
}

func (app *appImpl) Cursor(win oswin.Window) cursor.Cursor {
	app.mu.Lock()
	app.ctxtwin = win.(*windowImpl)
	app.mu.Unlock()
	return &theCursor
}

func (app *appImpl) OpenURL(url string) {
	cmd := exec.Command("open", url)
	cmd.Run()
}

func (app *appImpl) SetQuitReqFunc(fun func()) {
	app.quitReqFunc = fun
}

func (app *appImpl) SetQuitCleanFunc(fun func()) {
	app.quitCleanFunc = fun
}

func (app *appImpl) QuitReq() {
	if app.quitting {
		return
	}
	if app.quitReqFunc != nil {
		app.quitReqFunc()
	} else {
		app.Quit()
	}
}

func (app *appImpl) IsQuitting() bool {
	return app.quitting
}

func (app *appImpl) QuitClean() {
	app.quitting = true
	if app.quitCleanFunc != nil {
		app.quitCleanFunc()
	}
	app.mu.Lock()
	nwin := len(app.winlist)
	for i := nwin - 1; i >= 0; i-- {
		win := app.winlist[i]
		go win.Close()
	}
	app.mu.Unlock()
	for i := 0; i < nwin; i++ {
		<-app.quitCloseCnt
		// fmt.Printf("win closed: %v\n", i)
	}
}

func (app *appImpl) Quit() {
	app.QuitClean()
	app.stopMain()
}
