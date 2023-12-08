// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"log"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/driver/base"
	"goki.dev/vgpu/v2/vdraw"
	"goki.dev/vgpu/v2/vgpu"

	vk "github.com/goki/vulkan"
)

func init() {
	runtime.LockOSThread()
}

var TheApp = &App{AppMulti: base.NewAppMulti[*Window]()}

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
		TheApp.StopMain()
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
		WindowMulti: base.NewWindowMulti[*App, *vdraw.Drawer](app, opts),
		glw:         glw,
		scrnName:    sc.Name,
	}
	w.Draw = &vdraw.Drawer{}
	w.EvMgr.Deque = &w.Deque

	app.RunOnMain(func() {
		surfPtr, err := glw.CreateWindowSurface(app.GPU.Instance, nil)
		if err != nil {
			log.Println(err)
		}
		sf := vgpu.NewSurface(app.GPU, vk.SurfaceFromPointer(surfPtr))
		w.Draw.YIsDown = true
		w.Draw.ConfigSurface(sf, vgpu.MaxTexturesPerSet) // note: can expand
	})

	// bitflag.SetAtomic(&w.Flag, int(goosi.Focus)) // starts out focused

	app.Mu.Lock()
	app.Windows = append(app.Windows, w)
	app.Mu.Unlock()

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

	go w.WinLoop() // start window's own dedicated publish update loop

	return w, nil
}

func (app *App) ClipBoard(win goosi.Window) clip.Board {
	app.Mu.Lock()
	app.CtxWindow = win.(*Window)
	app.Mu.Unlock()
	return &theClip
}

func (app *App) Cursor(win goosi.Window) cursor.Cursor {
	app.Mu.Lock()
	app.CtxWindow = win.(*Window)
	app.Mu.Unlock()
	return &TheCursor
}
