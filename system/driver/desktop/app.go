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
	"sync"
	"sync/atomic"

	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/gpudraw"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/system/driver/base"
	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/oliverbestmann/webgpu/wgpuglfw"
)

// Init sets up the desktop app. The windowing system is initialized
// when it is first needed (see [App.initGUI]).
func Init() {
	// some operating systems require us to be on the main thread
	runtime.LockOSThread()

	base.Init(TheApp, &TheApp.App)
}

// TheApp is the single [system.App] for the desktop platform
var TheApp = &App{AppMulti: base.NewAppMulti[*Window]()}

// App is the [system.App] implementation for the desktop platform
type App struct {
	base.AppMulti[*Window]

	// GPU is the system GPU used for the app
	GPU *gpu.GPU

	// Monitors are pointers to the glfw monitors corresponding to Screens.
	Monitors []*glfw.Monitor

	// dark is whether the system color theme is dark; see [App.IsDark].
	dark atomic.Bool

	// guiOnce initializes the windowing system once (see [App.initGUI]).
	guiOnce sync.Once

	// guiDone is whether the windowing system has been initialized.
	guiDone atomic.Bool

	// guiInitializing is whether the windowing system is being initialized,
	// during which the calls of [App.initGUI] on the main thread, as for the
	// screens when they are updated, return at once.
	guiInitializing atomic.Bool
}

// initGUI initializes the windowing system (glfw), the screens, and the
// theme watcher the first time that they are needed, as for the first
// window or the information about the screens, instead of when the
// program starts, so that a program that does not use the GUI, such as
// a server, does not initialize it, which gives it a dock icon on macOS.
// Like glfw, it runs on the main thread: in [App.MainLoop], which calls
// it first, so that it is done once the main loop runs, and before that,
// on the main goroutine, which [Init] locks to the main thread.
func (a *App) initGUI() {
	if a.guiDone.Load() || (a.guiInitializing.Load() && onMainThread()) {
		return
	}
	a.guiOnce.Do(func() {
		if !onMainThread() {
			panic("system/driver/desktop: the GUI must first be used on the main goroutine, as for the first window or screen information, before the main loop runs")
		}
		a.guiInitializing.Store(true)
		defer a.guiInitializing.Store(false)
		a.InitGPU()
		a.watchTheme()
		a.guiDone.Store(true)
	})
}

// NScreens returns the number of screens, initializing the windowing system
// if needed (see [App.initGUI]).
func (a *App) NScreens() int {
	a.initGUI()
	return a.AppMulti.NScreens()
}

// Screen returns the given screen, initializing the windowing system
// if needed (see [App.initGUI]).
func (a *App) Screen(n int) *system.Screen {
	a.initGUI()
	return a.AppMulti.Screen(n)
}

// ScreenByName returns the screen with the given name, initializing the
// windowing system if needed (see [App.initGUI]).
func (a *App) ScreenByName(name string) *system.Screen {
	a.initGUI()
	return a.AppMulti.ScreenByName(name)
}

// SendEmptyEvent sends an empty, blank event to global event processing
// system, which has the effect of pushing the system along during cases when
// the event loop needs to be "pinged" to get things moving along..
func (a *App) SendEmptyEvent() {
	if !a.guiDone.Load() {
		return // no event loop yet
	}
	glfw.PostEmptyEvent()
}

// MainLoop starts running event loop on main thread (must be called
// from the main thread).
func (a *App) MainLoop() {
	a.initGUI()
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
			glfw.WaitEventsTimeout(0.1) // timeout is essential on linux
			// (here, not with the paint events of the windows, which are
			// paused while they are idle; and glfw is used on the main thread)
			a.PollScreenChanges()
		}
	}
}

// InitGPU initializes glfw, gpu, and the screens.
func (a *App) InitGPU() {
	restore := imeInitApp()
	if err := glfw.Init(); err != nil {
		log.Fatalln("system/driver/desktop failed to initialize glfw:", err)
	}
	restore()
	glfw.SetMonitorCallback(a.MonitorChange)
	a.GetScreens()
}

func (a *App) NewWindow(opts *system.NewWindowOptions) (system.Window, error) {
	a.initGUI()
	if len(a.Windows) == 0 && system.UpdateLogicalDPIScaleFunc != nil {
		if ScreenDebug {
			log.Println("app first new window calling UpdateLogicalDPIScaleFunc")
		}
		system.UpdateLogicalDPIScaleFunc()
	}

	sc := a.Screens[0]

	if opts == nil {
		opts = &system.NewWindowOptions{}
	}
	opts.Fixup()
	// can also apply further tuning here..
	if opts.Screen > 0 && opts.Screen < len(a.Screens) {
		sc = a.Screens[opts.Screen]
	}

	w := &Window{
		WindowMulti:  base.NewWindowMulti[*App, *composer.ComposerDrawer](a, opts),
		ScreenWindow: sc.Name,
	}
	w.This = w

	var err error
	a.RunOnMain(func() {
		err = w.newGlfwWindow(opts, sc)
	})
	if err != nil {
		return nil, err
	}

	a.RunOnMain(func() {
		surf := gpu.Instance().CreateSurface(wgpuglfw.GetSurfaceDescriptor(w.Glw))
		var fbsz image.Point
		fbsz.X, fbsz.Y = w.Glw.GetFramebufferSize()
		if fbsz == (image.Point{}) {
			fbsz = opts.Size
		}
		if a.GPU == nil {
			a.GPU = gpu.NewGPU(surf)
		}
		// no multisample and no depth
		sf := gpu.NewSurface(a.GPU, surf, fbsz, 1, gpu.UndefinedType)
		w.Draw = gpudraw.NewDrawer(a.GPU, sf)
		w.Compose = &composer.ComposerDrawer{Drawer: w.Draw}
	})

	// w.Flgs.SetFlag(true, system.Focused) // starts out focused

	a.Mu.Lock()
	a.Windows = append(a.Windows, w)
	a.Mu.Unlock()

	w.Glw.SetPosCallback(w.Moved)
	w.Glw.SetSizeCallback(w.WinResized)
	w.Glw.SetFramebufferSizeCallback(w.FbResized)
	w.Glw.SetCloseCallback(w.OnCloseReq)
	// w.Glw.SetRefreshCallback(w.refresh)
	w.Glw.SetFocusCallback(w.Focused)
	w.Glw.SetIconifyCallback(w.Iconify)

	w.Glw.SetKeyCallback(w.KeyEvent)
	w.Glw.SetCharModsCallback(w.CharEvent)
	w.Glw.SetMouseButtonCallback(w.MouseButtonEvent)
	w.Glw.SetScrollCallback(w.ScrollEvent)
	w.Glw.SetCursorPosCallback(w.CursorPosEvent)
	w.Glw.SetCursorEnterCallback(w.CursorEnterEvent)
	w.Glw.SetDropCallback(w.DropEvent)
	w.Glw.SetInputMode(glfw.LockKeyMods, glfw.True)

	w.Show()
	a.RunOnMain(func() {
		w.updateGeometry()
		w.ConstrainFrame(false) // constrain full frame on open
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
