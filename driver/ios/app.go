// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

// Package ios implements goosi interfaces on iOS mobile devices
package ios

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	vk "github.com/goki/vulkan"
	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/driver/base"
	"goki.dev/goosi/events"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"
)

// TheApp is the single [goosi.App] for the iOS platform
var TheApp = &App{
	screen:       &goosi.Screen{},
	name:         "GoGi",
	quitCloseCnt: make(chan struct{}),
}

// App is the [goosi.App] implementation for the iOS platform
type App struct {
	base.AppSingle[*vdraw.Drawer, *Window]

	// GPU is the system GPU used for the app
	GPU *vgpu.GPU

	// Winptr is the pointer to the underlying system window
	Winptr uintptr
}

// Main is called from main thread when it is time to start running the
// main loop. When function f returns, the app ends automatically.
func Main(f func(goosi.App)) {
	TheApp.initVk()
	base.Main(f, TheApp, &TheApp.App)
}

// initVk initializes Vulkan things for the app
func (app *App) initVk() {
	err := vk.SetDefaultGetInstanceProcAddr()
	if err != nil {
		// TODO(kai): maybe implement better error handling here
		log.Fatalln("goosi/driver/ios.App.InitVk: failed to set Vulkan DefaultGetInstanceProcAddr")
	}
	err = vk.Init()
	if err != nil {
		log.Fatalln("goosi/driver/ios.App.InitVk: failed to initialize vulkan")
	}

	winext := vk.GetRequiredInstanceExtensions()
	app.GPU = vgpu.NewGPU()
	app.GPU.AddInstanceExt(winext...)
	app.GPU.Config(app.Name())
}

// destroyVk destroys vulkan things (the drawer and surface of the window) for when the app becomes invisible
func (app *App) destroyVk() {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	vk.DeviceWaitIdle(app.Drawer.Surf.Device.Device)
	app.Drawer.Destroy()
	app.Drawer.Surf.Destroy()
	app.Drawer = nil
}

// fullDestroyVk destroys all vulkan things for when the app is fully quit
func (app *App) fullDestroyVk() {
	app.Mu.Lock()
	defer app.Mu.Unlock()
	app.GPU.Destroy()
}

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (app *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	defer func() { base.HandleRecover(recover()) }()
	// the actual system window has to exist before we can create the window
	var winptr uintptr
	for {
		app.Mu.Lock()
		winptr = app.Winptr
		app.Mu.Unlock()

		if winptr != 0 {
			break
		}
	}
	if goosi.InitScreenLogicalDPIFunc != nil {
		goosi.InitScreenLogicalDPIFunc()
	}
	app.Mu.Lock()
	defer app.Mu.Unlock()
	app.Win = &Window{}
	app.window.EvMgr.Deque = &app.window.Deque
	app.window.EvMgr.Window(events.WinShow)
	app.window.EvMgr.Window(events.WinFocus)

	// on iOS, NewWindow happens after updateConfig, so we copy the
	// info over from the screen here.
	fmt.Println("copying physical dpi; screen:", TheApp.screen, "; window:", TheApp.window)
	TheApp.window.PhysDPI = TheApp.screen.PhysicalDPI
	fmt.Println("copied physical dpi")
	TheApp.window.LogDPI = TheApp.screen.LogicalDPI
	TheApp.window.PxSize = TheApp.screen.PixSize
	TheApp.window.WnSize = TheApp.screen.Geometry.Max
	TheApp.window.DevPixRatio = TheApp.screen.DevicePixelRatio

	fmt.Println("sending window events")
	TheApp.window.EvMgr.WindowResize()
	TheApp.window.EvMgr.WindowPaint()

	go app.window.winLoop()

	return app.window, nil
}

// setSysWindow sets the underlying system window pointer, surface, system, and drawer.
// It should only be called when app.mu is already locked.
func (app *App) setSysWindow(winptr uintptr) error {
	debug.SetPanicOnFault(true)
	defer func() { handleRecover(recover()) }()
	fmt.Println("setting sys window")
	var sf vk.Surface
	// we have to remake the surface, system, and drawer every time someone reopens the window
	// because the operating system changes the underlying window
	ret := vk.CreateWindowSurface(app.gpu.Instance, winptr, nil, &sf)
	if err := vk.Error(ret); err != nil {
		return err
	}
	app.Surface = vgpu.NewSurface(app.gpu, sf)

	fmt.Println("setting system")
	app.System = app.gpu.NewGraphicsSystem(app.name, &app.Surface.Device)
	app.System.ConfigRender(&app.Surface.Format, vgpu.UndefType)
	app.Surface.SetRender(&app.System.Render)
	// app.window.System.Mem.Vars.NDescs = vgpu.MaxTexturesPerSet
	app.System.Config()
	fmt.Println("making drawer")
	app.Draw = vdraw.Drawer{
		Sys:     *app.System,
		YIsDown: true,
	}
	// app.window.Draw.ConfigSys()
	app.Draw.ConfigSurface(app.Surface, vgpu.MaxTexturesPerSet)

	app.winptr = winptr
	// if the window already exists, we are coming back to it, so we need to show it
	// again and send a screen update
	if app.window != nil {
		app.window.EvMgr.Window(events.WinShow)
		app.window.EvMgr.Window(events.ScreenUpdate)
	}
	return nil
}

func (app *App) DeleteWin(w *Window) {
	// TODO: implement?
}

func (app *App) NScreens() int {
	if app.screen != nil {
		return 1
	}
	return 0
}

func (app *App) Screen(scrN int) *goosi.Screen {
	if scrN == 0 {
		return app.screen
	}
	return nil
}

func (app *App) ScreenByName(name string) *goosi.Screen {
	if app.screen.Name == name {
		return app.screen
	}
	return nil
}

func (app *App) NoScreens() bool {
	return app.screen == nil
}

func (app *App) NWindows() int {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window != nil {
		return 1
	}
	return 0
}

func (app *App) Window(win int) goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if win == 0 {
		return app.window
	}
	return nil
}

func (app *App) WindowByName(name string) goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window.Name() == name {
		return app.window
	}
	return nil
}

func (app *App) WindowInFocus() goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.window.IsFocus() {
		return app.window
	}
	return nil
}

func (app *App) ContextWindow() goosi.Window {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.window
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

func (app *App) PrefsDir() string {
	return "/data/data"
}

func (app *App) GetScreens() {
	// note: this is not applicable in mobile because screen info is not avail until Size event
}

func (app *App) Platform() goosi.Platforms {
	return goosi.Android
}

func (app *App) OpenURL(url string) {
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

func (app *App) ClipBoard(win goosi.Window) clip.Board {
	// TODO: implement clipboard
	// app.mu.Lock()
	// app.ctxtwin = win.(*windowImpl)
	// app.mu.Unlock()
	return nil
	// return &theClip
}

func (app *App) Cursor(win goosi.Window) cursor.Cursor {
	return &cursor.CursorBase{} // no-op
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

func (app *App) Quit() {
	if app.quitting {
		return
	}
	app.QuitClean()
	app.stopMain()
}

func (app *App) IsDark() bool {
	return app.isDark
}
