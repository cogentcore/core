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
type App struct { //gti:add
	base.AppSingle[*vdraw.Drawer, *Window]

	// GPU is the system GPU used for the app
	GPU *vgpu.GPU

	// Winptr is the pointer to the underlying system window
	Winptr uintptr
}

// Main is called from main thread when it is time to start running the
// main loop. When function f returns, the app ends automatically.
func Main(f func(goosi.App)) {
	TheApp.InitVk()
	base.Main(f, TheApp, &TheApp.App)
}

// InitVk initializes Vulkan things for the app
func (a *App) InitVk() {
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
	a.GPU = vgpu.NewGPU()
	a.GPU.AddInstanceExt(winext...)
	a.GPU.Config(a.Name())
}

// DestroyVk destroys vulkan things (the drawer and surface of the window) for when the app becomes invisible
func (a *App) DestroyVk() {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	vk.DeviceWaitIdle(a.Drawer.Surf.Device.Device)
	a.Drawer.Destroy()
	a.Drawer.Surf.Destroy()
	a.Drawer = nil
}

// FullDestroyVk destroys all vulkan things for when the app is fully quit
func (a *App) FullDestroyVk() {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.GPU.Destroy()
}

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (a *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	defer func() { base.HandleRecover(recover()) }()
	// the actual system window has to exist before we can create the window
	var winptr uintptr
	for {
		a.Mu.Lock()
		winptr = a.Winptr
		a.Mu.Unlock()

		if winptr != 0 {
			break
		}
	}
	if goosi.InitScreenLogicalDPIFunc != nil {
		goosi.InitScreenLogicalDPIFunc()
	}
	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.Win = &Window{base.NewWindowSingle(a, opts)}
	a.Win.This = a.Win
	a.Win.EvMgr.Deque = &a.Win.Deque
	a.Win.EvMgr.Window(events.WinShow)
	a.Win.EvMgr.Window(events.WinFocus)

	TheApp.Win.EvMgr.WindowResize()
	TheApp.Win.EvMgr.WindowPaint()

	go a.Win.WinLoop()

	return a.Win, nil
}

// SetSystemWindow sets the underlying system window pointer, surface, system, and drawer.
// It should only be called when [App.Mu] is already locked.
func (a *App) SetSystemWindow(winptr uintptr) error {
	defer func() { base.HandleRecover(recover()) }()
	var vsf vk.Surface
	// we have to remake the surface, system, and drawer every time someone reopens the window
	// because the operating system changes the underlying window
	ret := vk.CreateWindowSurface(a.GPU.Instance, winptr, nil, &vsf)
	if err := vk.Error(ret); err != nil {
		return err
	}
	sf := vgpu.NewSurface(a.GPU, vsf)

	sys := a.GPU.NewGraphicsSystem(a.Name(), &sf.Device)
	sys.ConfigRender(&sf.Format, vgpu.UndefType)
	sf.SetRender(&sys.Render)
	// sys.Mem.Vars.NDescs = vgpu.MaxTexturesPerSet
	sys.Config()
	a.Drawer = &vdraw.Drawer{
		Sys:     *sys,
		YIsDown: true,
	}
	// a.Drawer.ConfigSys()
	a.Drawer.ConfigSurface(sf, vgpu.MaxTexturesPerSet)

	a.Winptr = winptr
	// if the window already exists, we are coming back to it, so we need to show it
	// again and send a screen update
	if a.Win != nil {
		a.Win.EvMgr.Window(events.WinShow)
		a.Win.EvMgr.Window(events.ScreenUpdate)
	}
	return nil
}

func (a *App) PrefsDir() string {
	usr, err := user.Current()
	if grr.Log(err) != nil {
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, "Library")
}

func (a *App) Platform() goosi.Platforms {
	return goosi.IOS
}

func (a *App) OpenURL(url string) {
	// TODO(kai): implement OpenURL on iOS
}

func (a *App) ClipBoard(win goosi.Window) clip.Board {
	// TODO(kai): implement clipboard on iOS
	return &clip.BoardBase{}
}
