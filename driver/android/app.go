// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

// Package android implements goosi interfaces on Android mobile devices
package android

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"sync"

	vk "github.com/goki/vulkan"
	"goki.dev/girl/styles"
	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/events"
	"goki.dev/mobile/event/size"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"
)

var theApp = &appImpl{
	screen:       &goosi.Screen{},
	name:         "GoGi",
	quitCloseCnt: make(chan struct{}),
}

var _ goosi.App = theApp

type appImpl struct {
	mu            sync.Mutex
	mainQueue     chan funcRun
	mainDone      chan struct{}
	winptr        uintptr
	System        *vgpu.System
	Surface       *vgpu.Surface
	Draw          vdraw.Drawer
	window        *windowImpl
	gpu           *vgpu.GPU
	sizeEvent     size.Event // the last size event
	screen        *goosi.Screen
	noScreens     bool // if all screens have been disconnected, don't do anything..
	name          string
	about         string
	openFiles     []string
	quitting      bool          // set to true when quitting and closing windows
	quitCloseCnt  chan struct{} // counts windows to make sure all are closed before done
	quitReqFunc   func()
	quitCleanFunc func()
	darkMode      bool
	insets        styles.SideFloats
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
	theApp.eventLoop()
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
		done := make(chan bool)
		app.mainQueue <- funcRun{f: f, done: done}
		<-done
	}
}

// GoRunOnMain runs given function on main thread and returns immediately
func (app *appImpl) GoRunOnMain(f func()) {
	go func() {
		app.mainQueue <- funcRun{f: f, done: nil}
	}()
}

// SendEmptyEvent sends an empty, blank event to global event processing
// system, which has the effect of pushing the system along during cases when
// the event loop needs to be "pinged" to get things moving along..
func (app *appImpl) SendEmptyEvent() {
	app.window.SendEmptyEvent()
}

// PollEventsOnMain does the equivalent of the mainLoop but using PollEvents
// and returning when there are no more events.
func (app *appImpl) PollEventsOnMain() {
	// TODO: implement?
}

// PollEvents tells the main event loop to check for any gui events right now.
// Call this periodically from longer-running functions to ensure
// GUI responsiveness.
func (app *appImpl) PollEvents() {
	// TODO: implement?
}

// stopMain stops the main loop and thus terminates the app
func (app *appImpl) stopMain() {
	app.mainDone <- struct{}{}
}

// initVk initializes vulkan things
func (app *appImpl) initVk() {
	log.SetPrefix("GoMobile: ")
	err := vk.SetDefaultGetInstanceProcAddr()
	if err != nil {
		log.Fatalln("goosi/driver/android.app.initVk: failed to set Vulkan DefaultGetInstanceProcAddr")
	}
	err = vk.Init()
	if err != nil {
		log.Fatalln("goosi/driver/android.app.initVk: failed to initialize vulkan")
	}

	winext := vk.GetRequiredInstanceExtensions()
	app.gpu = vgpu.NewGPU()
	app.gpu.AddInstanceExt(winext...)
	app.gpu.Config(app.name)
}

// destroyVk destroys vulkan things (the drawer and surface of the window) for when the app becomes invisible
func (app *appImpl) destroyVk() {
	app.mu.Lock()
	defer app.mu.Unlock()
	vk.DeviceWaitIdle(app.Surface.Device.Device)
	app.Draw.Destroy()
	app.Surface.Destroy()
	app.Surface = nil
}

// fullDestroyVk destroys all vulkan things for when the app is fully quit
func (app *appImpl) fullDestroyVk() {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.window = nil
	app.gpu.Destroy()
	// vgpu.Terminate()
}

////////////////////////////////////////////////////////
//  Window

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (app *appImpl) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	// the actual system window has to exist before we can create the window
	var winptr uintptr
	for {
		app.mu.Lock()
		winptr = app.winptr
		app.mu.Unlock()

		if winptr != 0 {
			break
		}
	}
	// TODO: do we need to do this?
	// app.mu.Lock()
	// defer app.mu.Unlock()
	// app.getScreen()
	// goosi.InitScreenLogicalDPIFunc()
	// app.window.LogDPI = app.screens[0].LogicalDPI
	// app.window.sendWindowEvent(window.ScreenUpdate)
	return app.window, nil
}

// setSysWindow sets the underlying system window pointer, surface, system, and drawer
func (app *appImpl) setSysWindow(opts *goosi.NewWindowOptions, winPtr uintptr) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	var sf vk.Surface
	// we have to remake the surface, system, and drawer every time someone reopens the window
	// because the operating system changes the underlying window
	ret := vk.CreateWindowSurface(app.gpu.Instance, winPtr, nil, &sf)
	if err := vk.Error(ret); err != nil {
		return err
	}
	app.Surface = vgpu.NewSurface(app.gpu, sf)

	app.System = app.gpu.NewGraphicsSystem(app.name, &app.Surface.Device)
	app.System.ConfigRender(&app.Surface.Format, vgpu.UndefType)
	app.Surface.SetRender(&app.System.Render)
	// app.window.System.Mem.Vars.NDescs = vgpu.MaxTexturesPerSet
	app.System.Config()

	app.Draw = vdraw.Drawer{
		Sys:     *app.System,
		YIsDown: true,
	}
	// app.window.Draw.ConfigSys()
	app.Draw.ConfigSurface(app.Surface, vgpu.MaxTexturesPerSet)

	app.winptr = winPtr
	app.window.EvMgr.Window(events.Focus)
	return nil
}

func (app *appImpl) DeleteWin(w *windowImpl) {
	// TODO: implement?
}

func (app *appImpl) NScreens() int {
	if app.screen != nil {
		return 1
	}
	return 0
}

func (app *appImpl) Screen(scrN int) *goosi.Screen {
	if scrN == 0 {
		return app.screen
	}
	return nil
}

func (app *appImpl) ScreenByName(name string) *goosi.Screen {
	if app.screen.Name == name {
		return app.screen
	}
	return nil
}

func (app *appImpl) NoScreens() bool {
	return app.screen == nil
}

func (app *appImpl) NWindows() int {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window != nil {
		return 1
	}
	return 0
}

func (app *appImpl) Window(win int) goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if win == 0 {
		return app.window
	}
	return nil
}

func (app *appImpl) WindowByName(name string) goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window.Name() == name {
		return app.window
	}
	return nil
}

func (app *appImpl) WindowInFocus() goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window.IsFocus() {
		return app.window
	}
	return nil
}

func (app *appImpl) ContextWindow() goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.window
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

func (app *appImpl) PrefsDir() string {
	return "/data/data"
}

func (app *appImpl) GetScreens() {
	// note: this is not applicable in mobile because screen info is not avail until Size event
}

func (app *appImpl) Platform() goosi.Platforms {
	return goosi.Android
}

func (app *appImpl) OpenURL(url string) {
	// TODO: implement
}

// SrcDir tries to locate dir in GOPATH/src/ or GOROOT/src/pkg/ and returns its
// full path. GOPATH may contain a list of paths.  From Robin Elkind github.com/mewkiz/pkg
func SrcDir(dir string) (absDir string, err error) {
	// TODO: does this make sense?
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
	// TODO: implement
	// app.mu.Lock()
	// app.ctxtwin = win.(*windowImpl)
	// app.mu.Unlock()
	return nil
	// return &theClip
}

func (app *appImpl) Cursor(win goosi.Window) cursor.Cursor {
	return &cursor.CursorBase{} // no-op
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
	// TODO: implement?
	// app.quitting = true
	// if app.quitCleanFunc != nil {
	// 	app.quitCleanFunc()
	// }
	// app.mu.Lock()
	// nwin := len(app.winlist)
	// for i := nwin - 1; i >= 0; i-- {
	// 	win := app.winlist[i]
	// 	go win.Close()
	// }
	// app.mu.Unlock()
	// for i := 0; i < nwin; i++ {
	// 	<-app.quitCloseCnt
	// 	// fmt.Printf("win closed: %v\n", i)
	// }
}

func (app *appImpl) Quit() {
	if app.quitting {
		return
	}
	app.QuitClean()
	app.stopMain()
}
