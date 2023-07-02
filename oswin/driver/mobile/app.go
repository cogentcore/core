// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android || ios

// Package mobile implements oswin interfaces on mobile devices
package mobile

import (
	"fmt"
	"go/build"
	"image"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/clip"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/units"
	mapp "github.com/goki/mobile/app"
	"github.com/goki/vgpu/vdraw"
	"github.com/goki/vgpu/vgpu"
	vk "github.com/goki/vulkan"
)

// TODO: actually implement things for mobile app

var theApp = &appImpl{
	screens:      make([]*oswin.Screen, 0),
	name:         "GoGi",
	quitCloseCnt: make(chan struct{}),
}

var _ oswin.App = theApp

type appImpl struct {
	mu            sync.Mutex
	mainQueue     chan funcRun
	mainDone      chan struct{}
	window        *windowImpl
	winPtr        uintptr
	gpu           *vgpu.GPU
	screens       []*oswin.Screen
	screensAll    []*oswin.Screen // unique list of all screens ever seen -- get info from here if fails
	noScreens     bool            // if all screens have been disconnected, don't do anything..
	name          string
	about         string
	openFiles     []string
	quitting      bool          // set to true when quitting and closing windows
	quitCloseCnt  chan struct{} // counts windows to make sure all are closed before done
	quitReqFunc   func()
	quitCleanFunc func()
	mobapp        mapp.App
}

var mainCallback func(oswin.App)

// Main is called from main thread when it is time to start running the
// main loop.  When function f returns, the app ends automatically.
func Main(f func(oswin.App)) {
	log.Println("in Main")
	gi.DialogsSepWindow = false
	mainCallback = f
	theApp.initVk()
	oswin.TheApp = theApp
	go theApp.eventLoop()
	go func() {
		mainCallback(theApp)
		log.Println("main callback done")
		theApp.stopMain()
	}()
	theApp.mainLoop()
	log.Println("main loop done")
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

}

// PollEvents tells the main event loop to check for any gui events right now.
// Call this periodically from longer-running functions to ensure
// GUI responsiveness.
func (app *appImpl) PollEvents() {
}

// MainLoop starts running event loop on main thread (must be called
// from the main thread).
func (app *appImpl) mainLoop() {
	app.mainQueue = make(chan funcRun)
	app.mainDone = make(chan struct{})
	// SetThreadPri(1)
	// time.Sleep(100 * time.Millisecond)
	for {
		log.Println("loop")
		select {
		case <-app.mainDone:
			app.destroyVk()
			return
		case f := <-app.mainQueue:
			f.f()
			if f.done != nil {
				f.done <- true
			}
			// default:
			// 	glfw.WaitEventsTimeout(0.2) // timeout is essential to prevent hanging (on mac at least)
		}
	}
}

// stopMain stops the main loop and thus terminates the app
func (app *appImpl) stopMain() {
	log.Println("in stop main")
	app.RunOnMain(app.destroyVk)
	// app.mainDone <- struct{}{}
}

// initVk initializes vulkan things
func (app *appImpl) initVk() {
	log.SetPrefix("GoMobileVulkan: ")
	vgpu.Debug = true
	err := vk.SetDefaultGetInstanceProcAddr()
	if err != nil {
		log.Fatalln("oswin/driver/mobile: failed to set Vulkan DefaultGetInstanceProcAddr")
	}
	err = vk.Init()
	if err != nil {
		log.Fatalln("oswin/driver/mobile: failed to initialize vulkan")
	}

	winext := vk.GetRequiredInstanceExtensions()
	log.Printf("required exts: %#v\n", winext)
	app.gpu = vgpu.NewGPU()
	app.gpu.AddInstanceExt(winext...)
	app.gpu.Config(app.name)
}

// destroyVk gets removes vulkan things (ie: when the app is closed)
func (app *appImpl) destroyVk() {
	log.Println("destroying vk")
	app.mu.Lock()
	log.Println("past mutex")
	defer app.mu.Unlock()
	vk.DeviceWaitIdle(app.window.Surface.Device.Device)
	app.window.Draw.Destroy()
	// app.window.Draw = nil
	// app.window.Draw = vdraw.Drawer{}
	// app.window.System.Destroy()
	// app.window.System = nil
	app.window.Surface.Destroy()
	// app.window = nil
	// app.gpu.Destroy()
	// vgpu.Terminate()
}

////////////////////////////////////////////////////////
//  Window

// NewWindow returns the already existing window to satisfy the oswin.App interface. The newWindow is what actually creates a new window.
func (app *appImpl) NewWindow(opts *oswin.NewWindowOptions) (oswin.Window, error) {
	var winptr uintptr
	for {
		app.mu.Lock()
		if app.window != nil {
			winptr = app.window.window
		}
		app.mu.Unlock()

		if winptr != 0 {
			break
		}
	}
	return app.window, nil
}

func (app *appImpl) newWindow(opts *oswin.NewWindowOptions, winPtr uintptr) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	var sf vk.Surface
	if app.window == nil {
		app.window = &windowImpl{}
		app.window.app = app
	}
	log.Println("in NewWindow", app.gpu.Instance, winPtr, &sf)
	ret := vk.CreateWindowSurface(app.gpu.Instance, winPtr, nil, &sf)
	if err := vk.Error(ret); err != nil {
		log.Println("oswin/driver/mobile new window: vulkan error:", err)
		return err
	}
	app.window.Surface = vgpu.NewSurface(app.gpu, sf)

	log.Printf("format: %s\n", app.window.Surface.Format.String())

	app.window.System = app.gpu.NewGraphicsSystem(app.name, &app.window.Surface.Device)
	app.window.System.ConfigRender(&app.window.Surface.Format, vgpu.UndefType)
	app.window.Surface.SetRender(&app.window.System.Render)
	// app.window.System.Mem.Vars.NDescs = vgpu.MaxTexturesPerSet
	app.window.System.Config()

	app.window.Draw = vdraw.Drawer{
		Sys:     *app.window.System,
		YIsDown: true,
	}
	// app.window.Draw.ConfigSys()
	app.window.Draw.ConfigSurface(app.window.Surface, vgpu.MaxTexturesPerSet)

	app.window.window = winPtr
	log.Println("set window pointer to", app.window.window)

	return nil
}

func (app *appImpl) setScreen(sc *oswin.Screen) {
	if len(app.screens) == 0 {
		app.screens = make([]*oswin.Screen, 1)
	}
	app.screens[0] = sc
}

func (app *appImpl) getScreen() {
	w := app.window
	physX, physY := units.NewPt(float32(w.size.WidthPt)), units.NewPt(float32(w.size.HeightPt))
	physX.Convert(units.Mm, &units.Context{})
	physY.Convert(units.Mm, &units.Context{})
	w.PhysDPI = 36 * w.size.PixelsPerPt
	fmt.Println("pixels per pt", w.size.PixelsPerPt)
	sc := &oswin.Screen{
		ScreenNumber: 0,
		Geometry:     w.size.Bounds(),
		PixSize:      w.size.Size(),
		PhysicalSize: image.Point{X: int(physX.Val), Y: int(physY.Val)},
		PhysicalDPI:  36 * w.size.PixelsPerPt,
		LogicalDPI:   2.0,

		Orientation: oswin.ScreenOrientation(w.size.Orientation),
	}
	app.setScreen(sc)
}

// func (app *appImpl) getInitialScreen() {
// 	sz := app.window.Surface.Format.Size
// 	physX, physY := units.NewPt(float32(sz.X)), units.NewPt(float32(sz.Y))
// 	physX.Convert(units.Mm, &units.Context{})
// 	physY.Convert(units.Mm, &units.Context{})
// 	app.window.PhysDPI = 36 * 6.0
// 	sc := &oswin.Screen{
// 		ScreenNumber: 0,
// 		// Geometry:     w.size.Bounds(),
// 		// PixSize:      w.size.Size(),
// 		PixSize:      sz,
// 		PhysicalSize: image.Point{X: int(physX.Val), Y: int(physY.Val)},
// 		PhysicalDPI:  36 * 6.0,
// 		LogicalDPI:   2.0,
// 		// Orientation: oswin.ScreenOrientation(w.size.Orientation),
// 	}
// 	app.setScreen(sc)
// 	oswin.InitScreenLogicalDPIFunc()
// 	app.window.LogDPI = sc.LogicalDPI
// }

func (app *appImpl) DeleteWin(w *windowImpl) {
	return
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

func (app *appImpl) NoScreens() bool {
	return app.noScreens
}

func (app *appImpl) NWindows() int {
	return 1
}

func (app *appImpl) Window(win int) oswin.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if win == 0 {
		return app.window
	}
	return nil
}

func (app *appImpl) WindowByName(name string) oswin.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window.Name() == name {
		return app.window
	}
	return nil
}

func (app *appImpl) WindowInFocus() oswin.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window.IsFocus() {
		return app.window
	}
	return nil
}

func (app *appImpl) ContextWindow() oswin.Window {
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

func (app *appImpl) FontPaths() []string {
	return []string{"/system/fonts"}
}
func (app *appImpl) GetScreens() {
	// note: this is not applicable in mobile because screen info is not avail until Size event
}

func (app *appImpl) Platform() oswin.Platforms {
	return oswin.Android
}

func (app *appImpl) OpenURL(url string) {

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
	// app.mu.Lock()
	// app.ctxtwin = win.(*windowImpl)
	// app.mu.Unlock()
	return nil
	// return &theClip
}

func (app *appImpl) Cursor(win oswin.Window) cursor.Cursor {
	// app.mu.Lock()
	// app.ctxtwin = win.(*windowImpl)
	// app.mu.Unlock()
	return nil
	// return &theCursor
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
	log.Println("IN QUIT")
	if app.quitting {
		return
	}
	app.QuitClean()
	app.stopMain()
}

func (app *appImpl) ShowVirtualKeyboard(typ oswin.VirtualKeyboardTypes) {
	app.mobapp.ShowVirtualKeyboard(mapp.KeyboardType(typ))
}

func (app *appImpl) HideVirtualKeyboard() {
	app.mobapp.HideVirtualKeyboard()
}
