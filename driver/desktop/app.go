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

	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/driver/base"
	"goki.dev/vgpu/v2/vgpu"

	vk "github.com/goki/vulkan"
)

func init() {
	runtime.LockOSThread()
}

var TheApp = &App{
	windows:      make(map[*glfw.Window]*Window),
	oswindows:    make(map[uintptr]*Window),
	winlist:      make([]*Window, 0),
	screens:      make([]*goosi.Screen, 0),
	name:         "GoGi",
	quitCloseCnt: make(chan struct{}),
}

type App struct {
	base.AppMulti[*Window]

	// GPU is the system GPU used for the app
	GPU *vgpu.GPU

	// ShareWin is a non-visible, always-present window that all windows share gl context with
	ShareWin *glfw.Window
}

// Main is called from main thread when it is time to start running the
// main loop.  When function f returns, the app ends automatically.
func Main(f func(goosi.App)) {
	TheApp.initVk()
	goosi.TheApp = TheApp
	go func() {
		f(TheApp)
		TheApp.stopMain()
	}()
	TheApp.mainLoop()
}

// SendEmptyEvent sends an empty, blank event to global event processing
// system, which has the effect of pushing the system along during cases when
// the event loop needs to be "pinged" to get things moving along..
func (app *App) SendEmptyEvent() {
	glfw.PostEmptyEvent()
}

// mainLoop starts running event loop on main thread (must be called
// from the main thread).
func (app *App) mainLoop() {
	app.MainQueue = make(chan base.FuncRun)
	app.MainDone = make(chan struct{})
	for {
		select {
		case <-app.MainDone:
			glfw.Terminate()
			return
		case f := <-app.MainQueue:
			f.F()
			if f.Done != nil {
				f.Done <- struct{}{}
			}
		default:
			glfw.WaitEvents()
		}
	}
}

// initVk initializes glfw, vulkan (vgpu), etc
func (app *App) initVk() {
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
	app.ShareWin, err = glfw.CreateWindow(16, 16, "Share Window", nil, nil)
	if err != nil {
		log.Fatalln("goosi.vkos failed to create hidden share window", err)
	}

	winext := app.ShareWin.GetRequiredInstanceExtensions()
	app.GPU = vgpu.NewGPU()
	app.GPU.AddInstanceExt(winext...)
	app.GPU.Config(app.Name())

	app.GetScreens()
}

////////////////////////////////////////////////////////
//  Window

func (app *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	if len(app.Windows) == 0 && goosi.InitScreenLogicalDPIFunc != nil {
		if monitorDebug {
			log.Println("app first new window calling InitScreenLogicalDPIFunc")
		}
		goosi.InitScreenLogicalDPIFunc()
	}

	sc := app.Screens[0]

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

	w := &Window{
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
	w.EvMgr.Deque = &w.Deque

	app.RunOnMain(func() {
		surfPtr, err := glw.CreateWindowSurface(app.GPU.Instance, nil)
		if err != nil {
			log.Println(err)
		}
		w.Surface = vgpu.NewSurface(app.GPU, vk.SurfaceFromPointer(surfPtr))
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

	return w, nil
}

func (app *App) DeleteWin(w *Window) {
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

func (app *App) NScreens() int {
	return len(app.screens)
}

func (app *App) Screen(scrN int) *goosi.Screen {
	sz := len(app.screens)
	if scrN < sz {
		return app.screens[scrN]
	}
	return nil
}

func (app *App) ScreenByName(name string) *goosi.Screen {
	for _, sc := range app.screens {
		if sc.Name == name {
			return sc
		}
	}
	return nil
}

func (app *App) NoScreens() bool {
	return app.noScreens
}

func (app *App) NWindows() int {
	app.mu.Lock()
	defer app.mu.Unlock()
	return len(app.winlist)
}

func (app *App) Window(win int) goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	sz := len(app.winlist)
	if win < sz {
		return app.winlist[win]
	}
	return nil
}

func (app *App) WindowByName(name string) goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	for _, win := range app.winlist {
		if win.Name() == name {
			return win
		}
	}
	return nil
}

func (app *App) WindowInFocus() goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	for _, win := range app.winlist {
		if win.IsFocus() {
			return win
		}
	}
	return nil
}

func (app *App) ContextWindow() goosi.Window {
	app.mu.Lock()
	cw := app.ctxtwin
	app.mu.Unlock()
	return cw
}

func (app *App) Name() string {
	return app.name
}

func (app *App) SetName(name string) {
	app.name = name
}

func (app *App) About() string {
	return app.about
}

func (app *App) SetAbout(about string) {
	app.about = about
}

func (app *App) OpenFiles() []string {
	return app.openFiles
}

func (app *App) GoGiPrefsDir() string {
	pdir := filepath.Join(app.PrefsDir(), "GoGi")
	os.MkdirAll(pdir, 0755)
	return pdir
}

func (app *App) AppPrefsDir() string {
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

func (app *App) ClipBoard(win goosi.Window) clip.Board {
	app.mu.Lock()
	app.ctxtwin = win.(*Window)
	app.mu.Unlock()
	return &theClip
}

func (app *App) Cursor(win goosi.Window) cursor.Cursor {
	app.mu.Lock()
	app.ctxtwin = win.(*Window)
	app.mu.Unlock()
	return &theCursor
}

func (app *App) SetQuitReqFunc(fun func()) {
	app.quitReqFunc = fun
}

func (app *App) SetQuitCleanFunc(fun func()) {
	app.quitCleanFunc = fun
}

func (app *App) QuitReq() {
	if app.quitting {
		return
	}
	if app.quitReqFunc != nil {
		app.quitReqFunc()
	} else {
		app.Quit()
	}
}

func (app *App) IsQuitting() bool {
	return app.quitting
}

func (app *App) QuitClean() {
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

func (app *App) Quit() {
	if app.quitting {
		return
	}
	app.QuitClean()
	app.stopMain()
}

func (app *App) ShowVirtualKeyboard(typ goosi.VirtualKeyboardTypes) {
	// no-op
}

func (app *App) HideVirtualKeyboard() {
	// no-op
}
