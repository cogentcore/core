// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

// Package ios implements system interfaces on iOS mobile devices
package ios

import (
	"os/user"
	"path/filepath"
	"runtime"
	"unsafe"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/gpudraw"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/system/driver/base"
	"github.com/cogentcore/webgpu/wgpu"
)

func Init() {
	// Lock the goroutine responsible for initialization to an OS thread.
	// This means the goroutine running main (and calling the run function
	// below) is locked to the OS thread that started the program. This is
	// necessary for the correct delivery of UIKit events to the process.
	//
	// A discussion on this topic:
	// https://groups.google.com/forum/#!msg/golang-nuts/IiWZ2hUuLDA/SNKYYZBelsYJ
	runtime.LockOSThread()

	system.OnSystemWindowCreated = make(chan struct{})
	TheApp.InitGPU()
	base.Init(TheApp, &TheApp.App)
}

// TheApp is the single [system.App] for the iOS platform
var TheApp = &App{AppSingle: base.NewAppSingle[*composer.ComposerDrawer, *Window]()}

// App is the [system.App] implementation for the iOS platform
type App struct {
	base.AppSingle[*composer.ComposerDrawer, *Window]

	// GPU is the system GPU used for the app
	GPU *gpu.GPU

	// Draw is the GPU drawer associated with the [composer.ComposerDrawer].
	Draw *gpudraw.Drawer

	// Winptr is the pointer to the underlying system CAMetalLayer.
	Winptr uintptr
}

// InitGPU initializes GPU things for the app
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

// FullDestroyGPU destroys all GPU things for when the app is fully quit
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

	a.Event.WindowResize()
	a.Event.WindowPaint()

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
		MetalLayer: &wgpu.SurfaceDescriptorFromMetalLayer{
			Layer: unsafe.Pointer(winptr), // TODO: probably not layer
		},
	}
	wsf := gpu.Instance().CreateSurface(wsd)
	sf := gpu.NewSurface(a.GPU, wsf, a.Scrn.PixelSize, 1, gpu.UndefinedType)
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
	usr, err := user.Current()
	if errors.Log(err) != nil {
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, "Library")
}

func (a *App) Platform() system.Platforms {
	return system.IOS
}

func (a *App) OpenURL(url string) {
	// TODO(kai): implement OpenURL on iOS
}

func (a *App) Clipboard(win system.Window) system.Clipboard {
	return &Clipboard{}
}
