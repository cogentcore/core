// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/window"
	"goki.dev/vgpu/v2/vgpu"

	vk "github.com/goki/vulkan"
)

func init() {
	runtime.LockOSThread()
}

var VkOsDebug = false

var theApp = &appImpl{
	windows:      make(map[*glfw.Window]*windowImpl),
	oswindows:    make(map[uintptr]*windowImpl),
	winlist:      make([]*windowImpl, 0),
	screens:      make([]*goosi.Screen, 0),
	name:         "GoGi",
	quitCloseCnt: make(chan struct{}),
}

type appImpl struct {
	mu            sync.Mutex
	mainQueue     chan funcRun
	mainDone      chan struct{}
	gpu           *vgpu.GPU
	shareWin      *glfw.Window // a non-visible, always-present window that all windows share gl context with
	windows       map[*glfw.Window]*windowImpl
	oswindows     map[uintptr]*windowImpl
	winlist       []*windowImpl
	screens       []*goosi.Screen
	screensAll    []*goosi.Screen // unique list of all screens ever seen -- get info from here if fails
	noScreens     bool            // if all screens have been disconnected, don't do anything..
	ctxtwin       *windowImpl     // context window, dynamically set, for e.g., pointer and other methods
	name          string
	about         string
	openFiles     []string
	quitting      bool          // set to true when quitting and closing windows
	quitCloseCnt  chan struct{} // counts windows to make sure all are closed before done
	quitReqFunc   func()
	quitCleanFunc func()
}

var mainCallback func(goosi.App)

// Main is called from main thread when it is time to start running the
// main loop.  When function f returns, the app ends automatically.
func Main(f func(goosi.App)) {
	mainCallback = f
	theApp.initVk()
	goosi.TheApp = theApp
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

// SendEmptyEvent sends an empty, blank event to global event processing
// system, which has the effect of pushing the system along during cases when
// the event loop needs to be "pinged" to get things moving along..
func (app *appImpl) SendEmptyEvent() {
	glfw.PostEmptyEvent()
}

// PollEventsOnMain does the equivalent of the mainLoop but using PollEvents
// and returning when there are no more events.
func (app *appImpl) PollEventsOnMain() {
outer:
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
			glfw.PollEvents()
			break outer
		}
	}
}

// PollEvents tells the main event loop to check for any gui events right now.
// Call this periodically from longer-running functions to ensure
// GUI responsiveness.
func (app *appImpl) PollEvents() {
	app.RunOnMain(func() { app.PollEventsOnMain() })
}

// MainLoop starts running event loop on main thread (must be called
// from the main thread).
func (app *appImpl) mainLoop() {
	app.mainQueue = make(chan funcRun)
	app.mainDone = make(chan struct{})
	// SetThreadPri(1)
	// time.Sleep(100 * time.Millisecond)
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
			glfw.WaitEventsTimeout(0.2) // timeout is essential to prevent hanging (on mac at least)
		}
	}
}

// stopMain stops the main loop and thus terminates the app
func (app *appImpl) stopMain() {
	app.mainDone <- struct{}{}
}

// initVk initializes glfw, vulkan (vgpu), etc
func (app *appImpl) initVk() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("goosi.vkos failed to initialize glfw:", err)
	}
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	vk.Init()
	glfw.SetMonitorCallback(monitorChange)
	// glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.Visible, glfw.False)
	var err error
	app.shareWin, err = glfw.CreateWindow(16, 16, "Share Window", nil, nil)
	if err != nil {
		log.Fatalln("goosi.vkos failed to create hidden share window", err)
	}

	winext := app.shareWin.GetRequiredInstanceExtensions()
	app.gpu = vgpu.NewGPU()
	app.gpu.AddInstanceExt(winext...)
	vgpu.Debug = VkOsDebug
	app.gpu.Config(app.name)

	app.GetScreens()
}

////////////////////////////////////////////////////////
//  Window

func (app *appImpl) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	if len(app.winlist) == 0 && goosi.InitScreenLogicalDPIFunc != nil {
		if monitorDebug {
			log.Printf("app first new window calling InitScreenLogicalDPIFunc\n")
		}
		goosi.InitScreenLogicalDPIFunc()
	}

	sc := app.screens[0]

	if opts == nil {
		opts = &goosi.NewWindowOptions{}
	}
	opts.Fixup()
	// can also apply further tuning here..

	var glw *glfw.Window
	var err error
	app.RunOnMain(func() {
		glw, err = newVkWindow(opts, sc)
	})
	if err != nil {
		return nil, err
	}

	w := &windowImpl{
		app:         app,
		glw:         glw,
		scrnName:    sc.Name,
		publish:     make(chan struct{}),
		winClose:    make(chan struct{}),
		publishDone: make(chan struct{}),
		WindowBase: goosi.WindowBase{
			Titl: opts.GetTitle(),
			Flag: opts.Flags,
			FPS:  60,
		},
	}
	w.EventMgr.Win = w

	app.RunOnMain(func() {
		surfPtr, err := glw.CreateWindowSurface(app.gpu.Instance, nil)
		if err != nil {
			log.Println(err)
		}
		w.Surface = vgpu.NewSurface(app.gpu, vk.SurfaceFromPointer(surfPtr))
		w.Draw.YIsDown = true
		w.Draw.ConfigSurface(w.Surface, vgpu.MaxTexturesPerSet) // note: can expand
	})

	// bitflag.SetAtomic(&w.Flag, int(goosi.Focus)) // starts out focused

	app.mu.Lock()
	app.windows[glw] = w
	app.oswindows[w.OSHandle()] = w
	app.winlist = append(app.winlist, w)
	app.mu.Unlock()

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
	// w.sendWindowEvent(window.Paint)

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

func (app *appImpl) Screen(scrN int) *goosi.Screen {
	sz := len(app.screens)
	if scrN < sz {
		return app.screens[scrN]
	}
	return nil
}

func (app *appImpl) ScreenByName(name string) *goosi.Screen {
	for _, sc := range app.screens {
		if sc.Name == name {
			return sc
		}
	}
	return nil
}

func (app *appImpl) NoScreens() bool {
	return app.noScreens
}

func (app *appImpl) NWindows() int {
	app.mu.Lock()
	defer app.mu.Unlock()
	return len(app.winlist)
}

func (app *appImpl) Window(win int) goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	sz := len(app.winlist)
	if win < sz {
		return app.winlist[win]
	}
	return nil
}

func (app *appImpl) WindowByName(name string) goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	for _, win := range app.winlist {
		if win.Name() == name {
			return win
		}
	}
	return nil
}

func (app *appImpl) WindowInFocus() goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	for _, win := range app.winlist {
		if win.IsFocus() {
			return win
		}
	}
	return nil
}

func (app *appImpl) ContextWindow() goosi.Window {
	app.mu.Lock()
	cw := app.ctxtwin
	app.mu.Unlock()
	return cw
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

func (app *appImpl) OpenFiles() []string {
	return app.openFiles
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

func (app *appImpl) ClipBoard(win goosi.Window) clip.Board {
	app.mu.Lock()
	app.ctxtwin = win.(*windowImpl)
	app.mu.Unlock()
	return &theClip
}

func (app *appImpl) Cursor(win goosi.Window) cursor.Cursor {
	app.mu.Lock()
	app.ctxtwin = win.(*windowImpl)
	app.mu.Unlock()
	return &theCursor
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
	if app.quitting {
		return
	}
	app.QuitClean()
	app.stopMain()
}

func (app *appImpl) ShowVirtualKeyboard(typ goosi.VirtualKeyboardTypes) {
	// no-op
}

func (app *appImpl) HideVirtualKeyboard() {
	// no-op
}
