// Copyright 2019 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"image"
	"log"
	"runtime"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/gpudraw"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/base"
	"github.com/cogentcore/webgpu/wgpuglfw"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func Init() {
	// some operating systems require us to be on the main thread
	runtime.LockOSThread()

	TheApp.InitGPU()
	base.Init(TheApp, &TheApp.App)
}

// TheApp is the single [system.App] for the desktop platform
var TheApp = &App{AppMulti: base.NewAppMulti[*Window]()}

// App is the [system.App] implementation for the desktop platform
type App struct {
	base.AppMulti[*Window]

	// GPU is the system GPU used for the app
	GPU *gpu.GPU

	// ShareWin is a non-visible, always-present window that all windows share gl context with
	ShareWin *glfw.Window
}

// SendEmptyEvent sends an empty, blank event to global event processing
// system, which has the effect of pushing the system along during cases when
// the event loop needs to be "pinged" to get things moving along..
func (a *App) SendEmptyEvent() {
	glfw.PostEmptyEvent()
}

// MainLoop starts running event loop on main thread (must be called
// from the main thread).
func (a *App) MainLoop() {
	a.MainQueue = make(chan base.FuncRun)
	a.MainDone = make(chan struct{})
	for {
		select {
		case <-a.MainDone:
			glfw.Terminate()
			return
		case f := <-a.MainQueue:
			f.F()
			if f.Done != nil {
				f.Done <- struct{}{}
			}
		default:
			glfw.WaitEventsTimeout(0.1) // this is essential on linux
		}
	}
}

// InitGPU initializes glfw, gpu, and the screens.
func (a *App) InitGPU() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("system/driver/desktop failed to initialize glfw:", err)
	}
	glfw.SetMonitorCallback(a.MonitorChange)
	// glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.Visible, glfw.False)
	var err error
	a.ShareWin, err = glfw.CreateWindow(16, 16, "Share Window", nil, nil)
	if err != nil {
		log.Fatalln("desktop.App failed to create hidden share window", err)
	}

	a.GPU = gpu.NewGPU()
	a.GPU.Config(a.Name())

	a.GetScreens()
}

func (a *App) NewWindow(opts *system.NewWindowOptions) (system.Window, error) {
	if len(a.Windows) == 0 && system.InitScreenLogicalDPIFunc != nil {
		if MonitorDebug {
			log.Println("app first new window calling InitScreenLogicalDPIFunc")
		}
		system.InitScreenLogicalDPIFunc()
	}

	sc := a.Screens[0]

	if opts == nil {
		opts = &system.NewWindowOptions{}
	}
	opts.Fixup()
	// can also apply further tuning here..

	var glw *glfw.Window
	var err error
	a.RunOnMain(func() {
		glw, err = NewGlfwWindow(opts, sc)
	})
	if err != nil {
		return nil, err
	}

	w := &Window{
		WindowMulti:  base.NewWindowMulti[*App, *gpudraw.Drawer](a, opts),
		Glw:          glw,
		ScreenWindow: sc.Name,
	}
	w.This = w
	w.Events().Queue.Init()

	a.RunOnMain(func() {
		surf := a.GPU.Instance.CreateSurface(wgpuglfw.GetSurfaceDescriptor(glw))
		var fbsz image.Point
		fbsz.X, fbsz.Y = glw.GetFramebufferSize()
		if fbsz == (image.Point{}) {
			fbsz = opts.Size
		}
		// no multisample and no depth
		sf := gpu.NewSurface(a.GPU, surf, fbsz, 1, gpu.UndefinedType)
		w.Draw = gpudraw.NewDrawer(a.GPU, sf)
	})

	// w.Flgs.SetFlag(true, system.Focused) // starts out focused

	a.Mu.Lock()
	a.Windows = append(a.Windows, w)
	a.Mu.Unlock()

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
	a.RunOnMain(func() {
		w.UpdateGeom()
	})

	go w.WinLoop() // start window's own dedicated publish update loop

	return w, nil
}

func (a *App) Clipboard(win system.Window) system.Clipboard {
	a.Mu.Lock()
	a.CtxWindow = win.(*Window)
	a.Mu.Unlock()
	return TheClipboard
}

func (a *App) Cursor(win system.Window) system.Cursor {
	a.Mu.Lock()
	a.CtxWindow = win.(*Window)
	a.Mu.Unlock()
	return TheCursor
}
