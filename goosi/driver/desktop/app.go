// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

//go:generate core generate

import (
	"log"
	"runtime"

	"cogentcore.org/core/goosi"
	"cogentcore.org/core/goosi/driver/base"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/vgpu"
	"cogentcore.org/core/vgpu/vdraw"
	"github.com/go-gl/glfw/v3.3/glfw"

	vk "github.com/goki/vulkan"
)

func Init() {
	// some operating systems require us to be on the main thread
	runtime.LockOSThread()

	TheApp.InitVk()
	base.Init(TheApp, &TheApp.App)
}

// TheApp is the single [goosi.App] for the desktop platform
var TheApp = &App{AppMulti: base.NewAppMulti[*Window]()}

// App is the [goosi.App] implementation for the desktop platform
type App struct { //gti:add
	base.AppMulti[*Window]

	// GPU is the system GPU used for the app
	GPU *vgpu.GPU

	// ShareWin is a non-visible, always-present window that all windows share gl context with
	ShareWin *glfw.Window
}

// SendEmptyEvent sends an empty, blank event to global event processing
// system, which has the effect of pushing the system along during cases when
// the event loop needs to be "pinged" to get things moving along..
func (app *App) SendEmptyEvent() {
	glfw.PostEmptyEvent()
}

// MainLoop starts running event loop on main thread (must be called
// from the main thread).
func (app *App) MainLoop() {
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

// InitVk initializes glfw, vulkan, vgpu, and the screens.
func (app *App) InitVk() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("goosi/driver/desktop failed to initialize glfw:", err)
	}
	vk.SetGetInstanceProcAddr(glfw.GetVulkanGetInstanceProcAddress())
	vk.Init()
	glfw.SetMonitorCallback(app.MonitorChange)
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

func (app *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	if len(app.Windows) == 0 && goosi.InitScreenLogicalDPIFunc != nil {
		if MonitorDebug {
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
		glw, err = NewGlfwWindow(opts, sc)
	})
	if err != nil {
		return nil, err
	}

	w := &Window{
		WindowMulti:  base.NewWindowMulti[*App, *vdraw.Drawer](app, opts),
		Glw:          glw,
		ScreenWindow: sc.Name,
	}
	w.This = w
	w.Draw = &vdraw.Drawer{}

	app.RunOnMain(func() {
		surfPtr := grr.Log1(glw.CreateWindowSurface(app.GPU.Instance, nil))
		sf := vgpu.NewSurface(app.GPU, vk.SurfaceFromPointer(surfPtr))
		w.Draw.YIsDown = true
		w.Draw.ConfigSurface(sf, vgpu.MaxTexturesPerSet) // note: can expand
	})

	// w.Flgs.SetFlag(true, goosi.Focused) // starts out focused

	app.Mu.Lock()
	app.Windows = append(app.Windows, w)
	app.Mu.Unlock()

	glw.SetPosCallback(w.Moved)
	glw.SetSizeCallback(w.WinResized)
	glw.SetFramebufferSizeCallback(w.FbResized)
	glw.SetCloseCallback(w.OnCloseReq)
	// glw.SetRefreshCallback(w.refresh)
	glw.SetFocusCallback(w.Focused)
	glw.SetIconifyCallback(w.Iconify)

	glw.SetKeyCallback(w.KeyEvent)
	glw.SetCharModsCallback(w.CharEvent)
	glw.SetMouseButtonCallback(w.MouseButtonEvent)
	glw.SetScrollCallback(w.ScrollEvent)
	glw.SetCursorPosCallback(w.CursorPosEvent)
	glw.SetCursorEnterCallback(w.CursorEnterEvent)
	glw.SetDropCallback(w.DropEvent)

	w.Show()
	app.RunOnMain(func() {
		w.UpdateGeom()
	})

	go w.WinLoop() // start window's own dedicated publish update loop

	return w, nil
}

func (app *App) Clipboard(win goosi.Window) goosi.Clipboard {
	app.Mu.Lock()
	app.CtxWindow = win.(*Window)
	app.Mu.Unlock()
	return TheClip
}

func (app *App) Cursor(win goosi.Window) goosi.Cursor {
	app.Mu.Lock()
	app.CtxWindow = win.(*Window)
	app.Mu.Unlock()
	return TheCursor
}
