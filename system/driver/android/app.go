// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package android

import (
	"image"
	"unsafe"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/gpudraw"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/system/driver/base"
	"github.com/cogentcore/webgpu/wgpu"
)

func Init() {
	system.OnSystemWindowCreated = make(chan struct{})
	TheApp.InitGPU()
	base.Init(TheApp, &TheApp.App)
}

// TheApp is the single [system.App] for the Android platform
var TheApp = &App{AppSingle: base.NewAppSingle[*composer.ComposerDrawer, *Window]()}

// App is the [system.App] implementation for the Android platform
type App struct {
	base.AppSingle[*composer.ComposerDrawer, *Window]

	// GPU is the system GPU used for the app
	GPU *gpu.GPU

	// Draw is the GPU drawer associated with the [composer.ComposerDrawer].
	Draw *gpudraw.Drawer

	// Winptr is the pointer to the underlying system window
	Winptr uintptr
}

// InitGPU initializes WebGPU for the app.
func (a *App) InitGPU() {
	a.GPU = gpu.NewGPU(nil)
}

// DestroyGPU releases GPU things (the drawer of the window) for when the app becomes invisible
func (a *App) DestroyGPU() {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.Draw.Release()
	a.Draw = nil
	a.Compose = nil
}

// FullDestroyGPU destroys all GPU things for when the app is fully quit.
func (a *App) FullDestroyGPU() {
	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.GPU.Release()
}

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (a *App) NewWindow(opts *system.NewWindowOptions) (system.Window, error) {
	defer func() { system.HandleRecover(recover()) }()

	a.Mu.Lock()
	defer a.Mu.Unlock()
	a.Win = &Window{base.NewWindowSingle(a, opts)}
	a.Win.This = a.Win
	a.Event.Window(events.WinShow)
	a.Event.Window(events.WinFocus)

	go a.Win.WinLoop()

	return a.Win, nil
}

// SetSystemWindow sets the underlying system window pointer, surface, system, and drawer.
// It should only be called when [App.Mu] is already locked.
func (a *App) SetSystemWindow(winptr uintptr) error {
	defer func() { system.HandleRecover(recover()) }()
	// we have to remake the surface and drawer every time someone reopens the window
	// because the operating system changes the underlying window
	wsd := &wgpu.SurfaceDescriptor{
		AndroidNativeWindow: &wgpu.SurfaceDescriptorFromAndroidNativeWindow{
			Window: unsafe.Pointer(winptr),
		},
	}
	wsf := gpu.Instance().CreateSurface(wsd)
	sf := gpu.NewSurface(a.GPU, wsf, image.Pt(512, 512), 1, gpu.UndefinedType) // placeholder size
	a.Draw = gpudraw.NewDrawer(a.GPU, sf)
	a.Compose = &composer.ComposerDrawer{Drawer: a.Draw}

	a.Winptr = winptr

	// if the window already exists, we are coming back to it, so we need to show it
	// again and send a screen update
	if a.Win != nil {
		a.Event.Window(events.WinShow)
		a.Event.Window(events.ScreenUpdate)
	}
	return nil
}

func (a *App) DataDir() string {
	return "/data/data"
}

func (a *App) Platform() system.Platforms {
	return system.Android
}

func (a *App) OpenURL(url string) {
	// TODO(kai): implement OpenURL on Android
}

func (a *App) Clipboard(win system.Window) system.Clipboard {
	return &Clipboard{}
}
