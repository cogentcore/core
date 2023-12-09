// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

// Package ios implements goosi interfaces on iOS mobile devices
package ios

import (
	"log"
	"os/user"
	"path/filepath"

	vk "github.com/goki/vulkan"
	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/driver/base"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"
)

// TheApp is the single [goosi.App] for the iOS platform
var TheApp = &App{AppSingle: base.NewAppSingle[*vdraw.Drawer, *Window]()}

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
	app.Win = &Window{base.NewWindowSingle(app, opts)}
	app.Win.This = app.Win
	app.Win.EvMgr.Deque = &app.Win.Deque
	app.Win.EvMgr.Window(events.WinShow)
	app.Win.EvMgr.Window(events.WinFocus)

	TheApp.Win.EvMgr.WindowResize()
	TheApp.Win.EvMgr.WindowPaint()

	go app.Win.WinLoop()

	return app.Win, nil
}

// setSysWindow sets the underlying system window pointer, surface, system, and drawer.
// It should only be called when app.mu is already locked.
func (app *App) setSysWindow(winptr uintptr) error {
	defer func() { base.HandleRecover(recover()) }()
	var vsf vk.Surface
	// we have to remake the surface, system, and drawer every time someone reopens the window
	// because the operating system changes the underlying window
	ret := vk.CreateWindowSurface(app.GPU.Instance, winptr, nil, &vsf)
	if err := vk.Error(ret); err != nil {
		return err
	}
	sf := vgpu.NewSurface(app.GPU, vsf)

	sys := app.GPU.NewGraphicsSystem(app.Name(), &sf.Device)
	sys.ConfigRender(&sf.Format, vgpu.UndefType)
	sf.SetRender(&sys.Render)
	// sys.Mem.Vars.NDescs = vgpu.MaxTexturesPerSet
	sys.Config()
	app.Drawer = &vdraw.Drawer{
		Sys:     *sys,
		YIsDown: true,
	}
	// a.Drawer.ConfigSys()
	app.Drawer.ConfigSurface(sf, vgpu.MaxTexturesPerSet)

	app.Winptr = winptr
	// if the window already exists, we are coming back to it, so we need to show it
	// again and send a screen update
	if app.Win != nil {
		app.Win.EvMgr.Window(events.WinShow)
		app.Win.EvMgr.Window(events.ScreenUpdate)
	}
	return nil
}

func (app *App) PrefsDir() string {
	usr, err := user.Current()
	if grr.Log(err) != nil {
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, "Library")
}

func (app *App) Platform() goosi.Platforms {
	return goosi.IOS
}

func (app *App) OpenURL(url string) {
	// TODO(kai): implement OpenURL on iOS
}

func (app *App) ClipBoard(win goosi.Window) clip.Board {
	// TODO(kai): implement clipboard on iOS
	return &clip.BoardBase{}
}
