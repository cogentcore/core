// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import (
	"fmt"
	"go/build"
	"image"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/clip"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/gpu"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki/bitflag"
)

func init() {
	runtime.LockOSThread()
}

// var glosDebug = false
var glosDebug = true

// 3.3 is pretty similar to 4.1 and more widely supported (e.g., crostini)
// 4.1 is max supported on macos
var glosGlMajor = 3
var glosGlMinor = 3

var theApp = &appImpl{
	windows:      make(map[*glfw.Window]*windowImpl),
	oswindows:    make(map[uintptr]*windowImpl),
	winlist:      make([]*windowImpl, 0),
	screens:      make([]*oswin.Screen, 0),
	name:         "GoGi",
	quitCloseCnt: make(chan struct{}),
}

type appImpl struct {
	mu            sync.Mutex
	mainQueue     chan funcRun
	mainDone      chan struct{}
	shareWin      *glfw.Window // a non-visible, always-present window that all windows share gl context with
	windows       map[*glfw.Window]*windowImpl
	oswindows     map[uintptr]*windowImpl
	winlist       []*windowImpl
	screens       []*oswin.Screen
	ctxtwin       *windowImpl // context window, dynamically set, for e.g., pointer and other methods
	name          string
	about         string
	quitting      bool          // set to true when quitting and closing windows
	quitCloseCnt  chan struct{} // counts windows to make sure all are closed before done
	quitReqFunc   func()
	quitCleanFunc func()

	// gl drawing programs
	progInit bool
	drawProg gpu.Program
	fillProg gpu.Program
}

var mainCallback func(oswin.App)

// Main is called from main thread when it is time to start running the
// main loop.  When function f returns, the app ends automatically.
func Main(f func(oswin.App)) {
	mainCallback = f
	theApp.initGl()
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
		glfw.PostEmptyEvent()
	}
}

// GoRunOnMain runs given function on main thread and returns immediately
func (app *appImpl) GoRunOnMain(f func()) {
	go func() {
		glfw.PostEmptyEvent()
		app.mainQueue <- funcRun{f: f, done: nil}
		glfw.PostEmptyEvent()
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
				glfw.WaitEventsTimeout(0.1) // maybe prevents hanging..
				// glfw.WaitEvents()
			}
		}
	}
}

// stopMain stops the main loop and thus terminates the app
func (app *appImpl) stopMain() {
	app.mainDone <- struct{}{}
}

// initGl initializes glfw, opengl, etc
func (app *appImpl) initGl() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("oswin.glos failed to initialize glfw:", err)
	}
	glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, glosGlMajor)
	glfw.WindowHint(glfw.ContextVersionMinor, glosGlMinor)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	var err error
	app.shareWin, err = glfw.CreateWindow(16, 16, "Share Window", nil, nil)
	if err != nil {
		log.Fatalln(fmt.Sprintf("oswin.glos failed to create hidden share window -- this usually means that the OpenGL version is not sufficient (should be at least: %d.%d):", glosGlMajor, glosGlMinor), err)
	}
	app.shareWin.MakeContextCurrent()
	theGPU.Init(glosDebug)
	err = app.initDrawProgs()
	if err != nil {
		log.Printf("oswin.glos initDrawProgs err:\n%s\n", err)
	}
	glfw.DetachCurrentContext()
	app.getScreens()
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
		publishDone: make(chan struct{}),
		WindowBase: oswin.WindowBase{
			Titl: opts.GetTitle(),
			Flag: opts.Flags,
		},
	}

	bitflag.SetAtomic(&w.Flag, int(oswin.Focus)) // starts out focused

	app.mu.Lock()
	app.windows[glw] = w
	app.oswindows[w.OSHandle()] = w
	app.winlist = append(app.winlist, w)
	app.mu.Unlock()

	app.RunOnMain(func() {
		w.Activate()
		gl.Init() // call to init in each context
		w.winTex = &textureImpl{size: opts.Size}
		w.winTex.Activate(0)
	})

	glw.SetPosCallback(w.moved)
	glw.SetSizeCallback(w.winResized)
	glw.SetFramebufferSizeCallback(w.fbResized)
	glw.SetCloseCallback(w.closeReq)
	// glw.SetRefreshCallback(w.refresh)
	glw.SetFocusCallback(w.focus)
	glw.SetIconifyCallback(w.iconify)

	glw.SetKeyCallback(w.keyEvent)
	glw.SetCharModsCallback(w.charEvent)
	glw.SetMouseButtonCallback(w.mouseButtonEvent)
	glw.SetScrollCallback(w.scrollEvent)
	glw.SetCursorPosCallback(w.cursorPosEvent)
	glw.SetCursorEnterCallback(w.cursorEnterEvent)
	glw.SetDropCallback(w.dropEvent)

	w.show()
	app.RunOnMain(func() {
		w.updtGeom()
	})

	go w.winLoop() // start window's own dedicated publish update loop

	w.sendWindowEvent(window.Paint)
	w.sendWindowEvent(window.Paint)

	return w, nil
}

func (app *appImpl) DeleteWin(w *windowImpl) {
	app.mu.Lock()
	defer app.mu.Unlock()
	_, ok := app.windows[w.glw]
	if !ok {
		return
	}
	for i, wl := range app.winlist {
		if wl == w {
			app.winlist = append(app.winlist[:i], app.winlist[i+1:]...)
			break
		}
	}
	delete(app.oswindows, w.OSHandle())
	delete(app.windows, w.glw)
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
	app.getScreens()
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

func (app *appImpl) NewTexture(win oswin.Window, size image.Point) oswin.Texture {
	var tx *textureImpl
	app.RunOnMain(func() {
		win.Activate()
		tx = &textureImpl{size: size}
		tx.Activate(0)
	})
	return tx
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
