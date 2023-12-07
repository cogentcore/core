// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package offscreen provides placeholder implementations of goosi interfaces
// to allow for offscreen testing and capturing of apps.
package offscreen

import (
	"fmt"
	"go/build"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"goki.dev/goosi"
	"goki.dev/goosi/clip"
	"goki.dev/goosi/cursor"
	"goki.dev/goosi/driver/base"
	"goki.dev/goosi/events"
)

// TheApp is the single [goosi.App] for the offscreen platform
var TheApp = &App{}

var _ goosi.App = TheApp

// App is the [goosi.App] implementation on the offscreen platform
type App struct {
	base.AppSingle[*drawerImpl, *windowImpl]
}

// handleRecover takes the given value of recover, and, if it is not nil,
// prints a panic message and a stack trace, using a string-based log
// method that guarantees that the stack trace will be printed before
// the program exits. This is needed because, without this, the program
// will exit before it can print the stack trace, which makes debugging
// nearly impossible. The correct usage of handleRecover is:
//
//	func myFunc() {
//		defer func() { handleRecover(recover()) }()
//		...
//	}
func handleRecover(r any) {
	if r == nil {
		return
	}
	log.Println("panic:", r)
	log.Println("")
	log.Println("----- START OF STACK TRACE: -----")
	log.Println(string(debug.Stack()))
	log.Fatalln("----- END OF STACK TRACE -----")
}

// Main is called from main thread when it is time to start running the
// main loop.  When function f returns, the app ends automatically.
func Main(f func(goosi.App)) {
	debug.SetPanicOnFault(true)
	defer func() { handleRecover(recover()) }()
	TheApp.This = TheApp
	TheApp.GetScreens()
	goosi.TheApp = TheApp
	go func() {
		f(TheApp)
		TheApp.stopMain()
	}()
	TheApp.mainLoop()
}

func (app *App) mainLoop() {
	app.mainQueue = make(chan funcRun)
	app.mainDone = make(chan struct{})
	for {
		select {
		case <-app.mainDone:
			return
		case f := <-app.mainQueue:
			f.f()
			if f.done != nil {
				f.done <- true
			}
		}
	}
}

type funcRun struct {
	f    func()
	done chan bool
}

// RunOnMain runs given function on main thread
func (app *App) RunOnMain(f func()) {
	if app.mainQueue == nil {
		f()
	} else {
		done := make(chan bool)
		app.mainQueue <- funcRun{f: f, done: done}
		<-done
	}
}

// GoRunOnMain runs given function on main thread and returns immediately
func (app *App) GoRunOnMain(f func()) {
	go func() {
		app.mainQueue <- funcRun{f: f, done: nil}
	}()
}

// SendEmptyEvent sends an empty, blank event to global event processing
// system, which has the effect of pushing the system along during cases when
// the event loop needs to be "pinged" to get things moving along..
func (app *App) SendEmptyEvent() {
	app.window.SendEmptyEvent()
}

// PollEventsOnMain does the equivalent of the mainLoop but using PollEvents
// and returning when there are no more events.
func (app *App) PollEventsOnMain() {
	// TODO: implement?
}

// PollEvents tells the main event loop to check for any gui events right now.
// Call this periodically from longer-running functions to ensure
// GUI responsiveness.
func (app *App) PollEvents() {
	// TODO: implement?
}

// stopMain stops the main loop and thus terminates the app
func (app *App) stopMain() {
	app.mainDone <- struct{}{}
}

////////////////////////////////////////////////////////
//  Window

// NewWindow creates a new window with the given options.
// It waits for the underlying system window to be created first.
// Also, it hides all other windows and shows the new one.
func (app *App) NewWindow(opts *goosi.NewWindowOptions) (goosi.Window, error) {
	defer func() { handleRecover(recover()) }()
	app.window = &windowImpl{
		app:         app,
		isVisible:   true,
		publish:     make(chan struct{}),
		winClose:    make(chan struct{}),
		publishDone: make(chan struct{}),
		WindowBase: goosi.WindowBase{
			Titl: opts.GetTitle(),
			Flag: opts.Flags,
			FPS:  60,
		},
	}
	app.window.EvMgr.Deque = &app.window.Deque
	app.setSysWindow(opts.Size)

	go app.window.winLoop()

	return app.window, nil
}

// setSysWindow sets the underlying system window information.
func (app *App) setSysWindow(sz image.Point) error {
	debug.SetPanicOnFault(true)
	defer func() { handleRecover(recover()) }()

	if sz.X == 0 {
		sz.X = 800
	}
	if sz.Y == 0 {
		sz.Y = 600
	}

	app.window.PhysDPI = app.screen.PhysicalDPI
	app.window.LogDPI = app.screen.LogicalDPI
	app.window.PxSize = sz
	app.window.WnSize = sz
	app.window.DevPixRatio = app.screen.DevicePixelRatio
	app.window.RenderSize = sz

	app.window.EvMgr.WindowResize()
	app.window.EvMgr.Window(events.WinShow)
	app.window.EvMgr.Window(events.ScreenUpdate)
	app.window.EvMgr.Window(events.WinFocus)
	return nil
}

func (app *App) DeleteWin(w *windowImpl) {
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
	// TODO(kai): figure out a better solution to offscreen prefs dir
	return filepath.Join(".", "tmpPrefsDir")
}

func (app *App) GetScreens() {
	sz := image.Point{1920, 1080}
	app.screen.DevicePixelRatio = 1
	app.screen.PixSize = sz
	app.screen.Geometry.Max = app.screen.PixSize
	dpi := float32(160)
	app.screen.PhysicalDPI = dpi
	app.screen.LogicalDPI = dpi

	physX := 25.4 * float32(sz.X) / dpi
	physY := 25.4 * float32(sz.Y) / dpi
	app.screen.PhysicalSize = image.Pt(int(physX), int(physY))
}

func (app *App) Platform() goosi.Platforms {
	return goosi.Offscreen
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

func (app *App) ShowVirtualKeyboard(typ goosi.VirtualKeyboardTypes) {
	// no-op
}

func (app *App) HideVirtualKeyboard() {
	// no-op
}
