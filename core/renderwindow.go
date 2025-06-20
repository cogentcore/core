// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/text/shaped"
	"golang.org/x/image/draw"
)

// windowWait is a wait group for waiting for all the open window event
// loops to finish. It is incremented by [renderWindow.GoStartEventLoop]
// and decremented when the event loop terminates.
var windowWait sync.WaitGroup

// Wait waits for all windows to close and runs the main app loop.
// This should be put at the end of the main function if
// [Body.RunMainWindow] is not used.
//
// For offscreen testing, Wait is typically never called, as it is
// not necessary (the app will already terminate once all tests are done,
// and nothing needs to run on the main thread).
func Wait() {
	waitCalled = true
	defer func() { system.HandleRecover(recover()) }()
	go func() {
		defer func() { system.HandleRecover(recover()) }()
		windowWait.Wait()
		system.TheApp.Quit()
	}()
	system.TheApp.MainLoop()
}

var (
	// currentRenderWindow is the current [renderWindow].
	// On single window platforms (mobile, web, and offscreen),
	// this is the only render window.
	currentRenderWindow *renderWindow

	// renderWindowGlobalMu is a mutex for any global state associated with windows
	renderWindowGlobalMu sync.Mutex
)

func setCurrentRenderWindow(w *renderWindow) {
	renderWindowGlobalMu.Lock()
	currentRenderWindow = w
	renderWindowGlobalMu.Unlock()
}

// renderWindow provides an outer "actual" window where everything is rendered,
// and is the point of entry for all events coming in from user actions.
//
// renderWindow contents are all managed by the [stages] stack that
// handles main [Stage] elements such as [WindowStage] and [DialogStage], which in
// turn manage their own stack of popup stage elements such as menus and tooltips.
// The contents of each Stage is provided by a Scene, containing Widgets,
// and the Stage Pixels image is drawn to the renderWindow in the renderWindow method.
//
// Rendering is handled by the [system.Drawer]. It is akin to a window manager overlaying Go image bitmaps
// on top of each other in the proper order, based on the [stages] stacking order.
// Sprites are managed by the main stage, as layered textures of the same size,
// to enable unlimited number packed into a few descriptors for standard sizes.
type renderWindow struct {

	// name is the name of the window.
	name string

	// title is the displayed name of window, for window manager etc.
	// Window object name is the internal handle and is used for tracking property info etc
	title string

	// SystemWindow is the OS-specific window interface, which handles
	// all the os-specific functions, including delivering events etc
	SystemWindow system.Window `json:"-" xml:"-"`

	// mains is the stack of main stages in this render window.
	// The [RenderContext] in this manager is the original source for all Stages.
	mains stages

	// noEventsChan is a channel on which a signal is sent when there are
	// no events left in the window [events.Deque]. It is used internally
	// for event handling in tests.
	noEventsChan chan struct{}

	// flags are atomic renderWindow flags.
	flags renderWindowFlags

	// lastResize is the time stamp of last resize event -- used for efficient updating.
	lastResize time.Time

	lastSpriteDraw time.Time

	// winRenderCounter is maintained under atomic locking to coordinate
	// the launching of renderAsync functions and when those functions
	// actually complete. Each time one is launched, the counter is incremented
	// and each time one completes, it is decremented. This ensures
	// everything is synchronized. Basically a [sync.WaitGroup] but we
	// need to just bail if not done, not wait.
	winRenderCounter int32
}

// newRenderWindow creates a new window with given internal name handle,
// display name, and options.  This is called by Stage.newRenderWindow
// which handles setting the opts and other infrastructure.
func newRenderWindow(name, title string, opts *system.NewWindowOptions) *renderWindow {
	w := &renderWindow{}
	w.name = name
	w.title = title
	var err error
	w.SystemWindow, err = system.TheApp.NewWindow(opts)
	if err != nil {
		fmt.Printf("Cogent Core NewRenderWindow error: %v \n", err)
		return nil
	}
	w.SystemWindow.SetName(title)
	w.SystemWindow.SetTitleBarIsDark(matcolor.SchemeIsDark)
	w.SystemWindow.SetCloseReqFunc(func(win system.Window) {
		rc := w.renderContext()
		rc.Lock()
		w.flags.SetFlag(true, winClosing)
		// ensure that everyone is closed first
		for _, kv := range w.mains.stack.Order {
			if kv.Value == nil || kv.Value.Scene == nil || kv.Value.Scene.This == nil {
				continue
			}
			if !kv.Value.Scene.Close() {
				w.flags.SetFlag(false, winClosing)
				rc.Unlock()
				return
			}
		}
		rc.Unlock()
		win.Close()
	})
	return w
}

// MainScene returns the current [renderWindow.mains] top Scene,
// which is the current window or full window dialog occupying the RenderWindow.
func (w *renderWindow) MainScene() *Scene {
	top := w.mains.top()
	if top == nil {
		return nil
	}
	return top.Scene
}

// RecycleMainWindow looks for an existing non-dialog window with the given Data.
// If it finds it, it shows it and returns true. Otherwise, it returns false.
// See [RecycleDialog] for a dialog version.
func RecycleMainWindow(data any) bool {
	if data == nil {
		return false
	}
	ew, got := mainRenderWindows.findData(data)
	if !got {
		return false
	}
	if DebugSettings.WindowEventTrace {
		fmt.Printf("Win: %v getting recycled based on data match\n", ew.name)
	}
	ew.Raise()
	return true
}

// setName sets name of this window and also the RenderWindow, and applies any window
// geometry settings associated with the new name if it is different from before
func (w *renderWindow) setName(name string) {
	curnm := w.name
	isdif := curnm != name
	w.name = name
	if w.SystemWindow != nil {
		w.SystemWindow.SetName(name)
	}
	if isdif && w.SystemWindow != nil && !w.SystemWindow.Is(system.Fullscreen) {
		wgp, sc := theWindowGeometrySaver.get(w.title, "")
		if wgp != nil {
			theWindowGeometrySaver.settingStart()
			if w.SystemWindow.Size() != wgp.Size || w.SystemWindow.Position(sc) != wgp.Pos {
				if DebugSettings.WindowGeometryTrace {
					log.Printf("WindowGeometry: SetName setting geom for window: %v pos: %v size: %v\n", w.name, wgp.Pos, wgp.Size)
				}
				w.SystemWindow.SetGeometry(false, wgp.Pos, wgp.Size, sc)
				system.TheApp.SendEmptyEvent()
			}
			theWindowGeometrySaver.settingEnd()
		}
	}
}

// setTitle sets title of this window and its underlying SystemWin.
func (w *renderWindow) setTitle(title string) {
	w.title = title
	if w.SystemWindow != nil {
		w.SystemWindow.SetTitle(title)
	}
}

// SetStageTitle sets the title of the underlying [system.Window] to the given stage title
// combined with the [renderWindow] title.
func (w *renderWindow) SetStageTitle(title string) {
	if title == "" {
		title = w.title
	} else if title != w.title {
		title = title + " • " + w.title
	}
	w.SystemWindow.SetTitle(title)
}

// logicalDPI returns the current logical dots-per-inch resolution of the
// window, which should be used for most conversion of standard units --
// physical DPI can be found in the Screen
func (w *renderWindow) logicalDPI() float32 {
	if w.SystemWindow == nil {
		sc := system.TheApp.Screen(0)
		if sc == nil {
			return 160 // null default
		}
		return sc.LogicalDPI
	}
	return w.SystemWindow.LogicalDPI()
}

// stepZoom calls [SetZoom] with the current zoom plus 10 times the given number of steps.
func (w *renderWindow) stepZoom(steps float32) {
	sc := w.SystemWindow.Screen()
	curZoom := AppearanceSettings.Zoom
	screenName := ""
	sset, ok := AppearanceSettings.Screens[sc.Name]
	if ok {
		screenName = sc.Name
		curZoom = sset.Zoom
	}
	w.setZoom(curZoom+10*steps, screenName)
}

// setZoom sets [AppearanceSettingsData.Zoom] to the given value and then triggers
// necessary updating and makes a snackbar. If screenName is non-empty, then the
// zoom is set on the screen-specific settings, instead of the global.
func (w *renderWindow) setZoom(zoom float32, screenName string) {
	zoom = math32.Clamp(zoom, 10, 500)
	if screenName != "" {
		sset := AppearanceSettings.Screens[screenName]
		sset.Zoom = zoom
		AppearanceSettings.Screens[screenName] = sset
	} else {
		AppearanceSettings.Zoom = zoom
	}
	AppearanceSettings.Apply()
	UpdateAll()
	errors.Log(SaveSettings(AppearanceSettings))

	if ms := w.MainScene(); ms != nil {
		b := NewBody().AddSnackbarText(fmt.Sprintf("%.f%%", zoom))
		NewStretch(b)
		b.AddSnackbarIcon(icons.Remove, func(e events.Event) {
			w.stepZoom(-1)
		})
		b.AddSnackbarIcon(icons.Add, func(e events.Event) {
			w.stepZoom(1)
		})
		b.AddSnackbarButton("Reset", func(e events.Event) {
			w.setZoom(100, screenName)
		})
		b.DeleteChildByName("stretch")
		b.RunSnackbar(ms)
	}
}

// resized updates Scene sizes after a window has been resized.
// It is called on any geometry update, including move and
// DPI changes, so it detects what actually needs to be updated.
func (w *renderWindow) resized() {
	rc := w.renderContext()
	if !w.isVisible() {
		rc.visible = false
		return
	}

	w.SystemWindow.Lock()
	rg := w.SystemWindow.RenderGeom()
	w.SystemWindow.Unlock()

	curRg := rc.geom
	curDPI := w.logicalDPI()
	if curRg == rg {
		newDPI := false
		if rc.logicalDPI != curDPI {
			rc.logicalDPI = curDPI
			newDPI = true
		}
		if DebugSettings.WindowEventTrace {
			fmt.Printf("Win: %v same-size resized: %v newDPI: %v\n", w.name, curRg, newDPI)
		}
		if w.mains.resize(rg) || newDPI {
			for _, kv := range w.mains.stack.Order {
				st := kv.Value
				sc := st.Scene
				sc.applyStyleScene()
			}
		}
		return
	}
	rc.logicalDPI = curDPI
	if !w.isVisible() {
		rc.visible = false
		if DebugSettings.WindowEventTrace {
			fmt.Printf("Win: %v Resized already closed\n", w.name)
		}
		return
	}
	if DebugSettings.WindowEventTrace {
		fmt.Printf("Win: %v Resized from: %v to: %v\n", w.name, curRg, rg)
	}
	rc.geom = rg
	rc.visible = true
	w.flags.SetFlag(true, winResize)
	w.mains.resize(rg)
	if DebugSettings.WindowGeometryTrace {
		log.Printf("WindowGeometry: recording from Resize\n")
	}
	theWindowGeometrySaver.record(w)
}

// Raise requests that the window be at the top of the stack of windows,
// and receive focus. If it is minimized, it will be un-minimized. This
// is the only supported mechanism for un-minimizing. This also sets
// [currentRenderWindow] to the window.
func (w *renderWindow) Raise() {
	w.SystemWindow.Raise()
	setCurrentRenderWindow(w)
}

// minimize requests that the window be minimized, making it no longer
// visible or active; rendering should not occur for minimized windows.
func (w *renderWindow) minimize() {
	w.SystemWindow.Minimize()
}

// closeReq requests that the window be closed, which could be rejected.
// It firsts unlocks and then locks the [renderContext] to prevent deadlocks.
// If this is called asynchronously outside of the main event loop,
// [renderWindow.SystemWin.closeReq] should be called directly instead.
func (w *renderWindow) closeReq() {
	rc := w.renderContext()
	rc.Unlock()
	w.SystemWindow.CloseReq()
	rc.Lock()
}

// closed frees any resources after the window has been closed.
func (w *renderWindow) closed() {
	AllRenderWindows.delete(w)
	mainRenderWindows.delete(w)
	dialogRenderWindows.delete(w)
	if DebugSettings.WindowEventTrace {
		fmt.Printf("Win: %v Closed\n", w.name)
	}
	if len(AllRenderWindows) > 0 {
		pfw := AllRenderWindows[len(AllRenderWindows)-1]
		if DebugSettings.WindowEventTrace {
			fmt.Printf("Win: %v getting restored focus after: %v closed\n", pfw.name, w.name)
		}
		pfw.Raise()
	}
}

// isClosed reports if the window has been closed
func (w *renderWindow) isClosed() bool {
	return w.SystemWindow.IsClosed() || w.mains.stack.Len() == 0
}

// isVisible is the main visibility check; don't do any window updates if not visible!
func (w *renderWindow) isVisible() bool {
	if w == nil || w.SystemWindow == nil || w.isClosed() || w.flags.HasFlag(winClosing) || !w.SystemWindow.IsVisible() {
		return false
	}
	return true
}

// goStartEventLoop starts the event processing loop for this window in a new
// goroutine, and returns immediately.  Adds to WindowWait wait group so a main
// thread can wait on that for all windows to close.
func (w *renderWindow) goStartEventLoop() {
	windowWait.Add(1)
	go w.eventLoop()
}

// todo: fix or remove
// sendWinFocusEvent sends the RenderWinFocusEvent to widgets
func (w *renderWindow) sendWinFocusEvent(act events.WinActions) {
	// se := window.NewEvent(act)
	// se.Init()
	// w.Mains.HandleEvent(se)
}

// eventLoop runs the event processing loop for the RenderWindow -- grabs system
// events for the window and dispatches them to receiving nodes, and manages
// other state etc (popups, etc).
func (w *renderWindow) eventLoop() {
	defer func() { system.HandleRecover(recover()) }()

	d := &w.SystemWindow.Events().Deque

	for {
		if w.flags.HasFlag(winStopEventLoop) {
			w.flags.SetFlag(false, winStopEventLoop)
			break
		}
		e := d.NextEvent()
		if w.flags.HasFlag(winStopEventLoop) {
			w.flags.SetFlag(false, winStopEventLoop)
			break
		}
		w.handleEvent(e)
		if w.noEventsChan != nil && len(d.Back) == 0 && len(d.Front) == 0 {
			w.noEventsChan <- struct{}{}
		}
	}
	if DebugSettings.WindowEventTrace {
		fmt.Printf("Win: %v out of event loop\n", w.name)
	}
	windowWait.Done()
	// our last act must be self destruction!
	w.mains.deleteAll()
}

// handleEvent processes given events.Event.
// All event processing operates under a RenderContext.Lock
// so that no rendering update can occur during event-driven updates.
// Because rendering itself is event driven, this extra level of safety
// is redundant in this case, but other non-event-driven updates require
// the lock protection.
func (w *renderWindow) handleEvent(e events.Event) {
	rc := w.renderContext()
	rc.Lock()
	// we manually handle Unlock's in this function instead of deferring
	// it to avoid a cryptic "sync: can't unlock an already unlocked Mutex"
	// error when panicking in the rendering goroutine. This is critical for
	// debugging on Android. TODO: maybe figure out a more sustainable approach to this.

	et := e.Type()
	if DebugSettings.EventTrace && et != events.WindowPaint && et != events.MouseMove {
		log.Println("Window got event", e)
	}
	if et >= events.Window && et <= events.WindowPaint {
		w.handleWindowEvents(e)
		rc.Unlock()
		return
	}
	if DebugSettings.EventTrace && (!w.isVisible() || w.SystemWindow.Is(system.Minimized)) {
		log.Println("got event while invisible:", e)
		log.Println("w.isClosed:", w.isClosed(), "winClosing flag:", w.flags.HasFlag(winClosing), "syswin !isvis:", !w.SystemWindow.IsVisible(), "minimized:", w.SystemWindow.Is(system.Minimized))
	}
	// fmt.Printf("got event type: %v: %v\n", et.BitIndexString(), evi)
	w.mains.mainHandleEvent(e)
	rc.Unlock()
}

func (w *renderWindow) handleWindowEvents(e events.Event) {
	et := e.Type()
	switch et {
	case events.WindowPaint:
		e.SetHandled()
		rc := w.renderContext()
		rc.Unlock() // one case where we need to break lock
		w.renderWindow()
		rc.Lock()
		w.mains.runDeferred() // note: must be outside of locks in renderWindow

	case events.WindowResize:
		e.SetHandled()
		w.resized()

	case events.Window:
		ev := e.(*events.WindowEvent)
		switch ev.Action {
		case events.WinClose:
			if w.SystemWindow.Lock() {
				// fmt.Printf("got close event for window %v \n", w.name)
				e.SetHandled()
				w.flags.SetFlag(true, winStopEventLoop)
				w.closed()
				w.SystemWindow.Unlock()
			}
		case events.WinMinimize:
			e.SetHandled()
			// on mobile platforms, we need to set the size to 0 so that it detects a size difference
			// and lets the size event go through when we come back later
			// if Platform().IsMobile() {
			// 	w.Scene.Geom.Size = image.Point{}
			// }
		case events.WinShow:
			e.SetHandled()
			// note that this is sent delayed by driver
			if DebugSettings.WindowEventTrace {
				fmt.Printf("Win: %v got show event\n", w.name)
			}
		case events.WinMove:
			e.SetHandled()
			// fmt.Printf("win move: %v\n", w.SystemWin.Position())
			if DebugSettings.WindowGeometryTrace {
				log.Printf("WindowGeometry: recording from Move\n")
			}
			w.SystemWindow.ConstrainFrame(true) // top only
			theWindowGeometrySaver.record(w)
		case events.WinFocus:
			// if we are not already the last in AllRenderWins, we go there,
			// as this allows focus to be restored to us in the future
			if len(AllRenderWindows) > 0 && AllRenderWindows[len(AllRenderWindows)-1] != w {
				AllRenderWindows.delete(w)
				AllRenderWindows.add(w)
			}
			if !w.flags.HasFlag(winGotFocus) {
				w.flags.SetFlag(true, winGotFocus)
				w.sendWinFocusEvent(events.WinFocus)
				if DebugSettings.WindowEventTrace {
					fmt.Printf("Win: %v got focus\n", w.name)
				}
			} else {
				if DebugSettings.WindowEventTrace {
					fmt.Printf("Win: %v got extra focus\n", w.name)
				}
			}
			setCurrentRenderWindow(w)
		case events.WinFocusLost:
			if DebugSettings.WindowEventTrace {
				fmt.Printf("Win: %v lost focus\n", w.name)
			}
			w.flags.SetFlag(false, winGotFocus)
			w.sendWinFocusEvent(events.WinFocusLost)
		case events.ScreenUpdate:
			if DebugSettings.WindowEventTrace {
				log.Println("Win: ScreenUpdate", w.name, screenConfig())
			}
			if !TheApp.Platform().IsMobile() { // native desktop
				if TheApp.NScreens() > 0 {
					AppearanceSettings.Apply()
					UpdateAll()
					theWindowGeometrySaver.restoreAll()
				}
			} else {
				w.resized()
			}
		}
	}
}

////////  Rendering

// renderParams are the key [renderWindow] params that determine if
// a scene needs to be restyled since last render, if these params change.
type renderParams struct {
	// logicalDPI is the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units.
	logicalDPI float32

	// Geometry of the rendering window, in actual "dot" pixels used for rendering.
	geom math32.Geom2DInt
}

// needsRestyle returns true if the current render context
// params differ from those used in last render.
func (rp *renderParams) needsRestyle(rc *renderContext) bool {
	return rp.logicalDPI != rc.logicalDPI || rp.geom != rc.geom
}

// saveRender grabs current render context params
func (rp *renderParams) saveRender(rc *renderContext) {
	rp.logicalDPI = rc.logicalDPI
	rp.geom = rc.geom
}

// renderContext provides rendering context from outer RenderWindow
// window to Stage and Scene elements to inform styling, layout
// and rendering. It also has the main Mutex for any updates
// to the window contents: use Lock for anything updating.
type renderContext struct {

	// logicalDPI is the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units.
	logicalDPI float32

	// Geometry of the rendering window, in actual "dot" pixels used for rendering.
	geom math32.Geom2DInt

	// visible is whether the window is visible and should be rendered to.
	visible bool

	// rebuild is whether to force a rebuild of all Scene elements.
	rebuild bool

	// TextShaper is the text shaping system for the render context,
	// for doing text layout.
	textShaper shaped.Shaper

	// render mutex for locking out rendering and any destructive updates.
	// It is locked at the [renderWindow] level during rendering and
	// event processing to provide exclusive blocking of external updates.
	// Use [WidgetBase.AsyncLock] from any outside routine to grab the lock before
	// doing modifications.
	sync.Mutex
}

// newRenderContext returns a new [renderContext] initialized according to
// the main Screen size and LogicalDPI as initial defaults.
// The actual window size is set during Resized method, which is typically
// called after the window is created by the OS.
func newRenderContext() *renderContext {
	rc := &renderContext{}
	scr := system.TheApp.Screen(0)
	if scr != nil {
		rc.geom.SetRect(image.Rectangle{Max: scr.PixelSize})
		rc.logicalDPI = scr.LogicalDPI
	} else {
		rc.geom = math32.Geom2DInt{Size: image.Pt(1080, 720)}
		rc.logicalDPI = 160
	}
	rc.visible = true
	rc.textShaper = shaped.NewShaper()
	return rc
}

func (rc *renderContext) String() string {
	str := fmt.Sprintf("Geom: %s  Visible: %v", rc.geom, rc.visible)
	return str
}

func (w *renderWindow) renderContext() *renderContext {
	return w.mains.renderContext
}

////////  renderWindow

// renderWindow performs all rendering based on current Stages config.
// It locks and unlocks the renderContext itself, which is necessary so that
// there is a moment for other goroutines to acquire the lock and get necessary
// updates through (such as in offscreen testing).
func (w *renderWindow) renderWindow() {
	if atomic.LoadInt32(&w.winRenderCounter) > 0 { // still working
		w.flags.SetFlag(true, winRenderSkipped)
		if DebugSettings.WindowRenderTrace {
			log.Printf("RenderWindow: still rendering, skipped: %v\n", w.name)
		}
		return
	}

	offscreen := TheApp.Platform() == system.Offscreen

	sinceResize := time.Since(w.lastResize)
	if !offscreen && sinceResize < 100*time.Millisecond {
		// get many rapid updates during resizing, so just rerender last one if so.
		// this works best in practice after a lot of experimentation.
		w.flags.SetFlag(true, winRenderSkipped)
		w.SystemWindow.Composer().Redraw()
		return
	}

	rc := w.renderContext()
	rc.Lock()
	defer func() {
		rc.rebuild = false
		rc.Unlock()
	}()
	rebuild := rc.rebuild

	stageMods, sceneMods := w.mains.updateAll() // handles all Scene / Widget updates!
	top := w.mains.top()
	if top == nil || w.mains.stack.Len() == 0 {
		return
	}
	spriteMods := top.Sprites.IsModified()

	spriteUpdateTime := SystemSettings.CursorBlinkTime
	if spriteUpdateTime == 0 {
		spriteUpdateTime = 500 * time.Millisecond
	}
	if w.flags.HasFlag(winGotFocus) && time.Since(w.lastSpriteDraw) > spriteUpdateTime {
		spriteMods = true
	}

	if !spriteMods && !rebuild && !stageMods && !sceneMods { // nothing to do!
		if w.flags.HasFlag(winRenderSkipped) {
			w.flags.SetFlag(false, winRenderSkipped)
		} else {
			return
		}
	}
	if !w.isVisible() || w.SystemWindow.Is(system.Minimized) {
		if DebugSettings.WindowRenderTrace {
			log.Printf("RenderWindow: skipping update on inactive / minimized window: %v\n", w.name)
		}
		return
	}

	if DebugSettings.WindowRenderTrace {
		log.Println("RenderWindow: doing render:", w.name)
		log.Println("rebuild:", rebuild, "stageMods:", stageMods, "sceneMods:", sceneMods)
	}

	if !w.SystemWindow.Lock() {
		if DebugSettings.WindowRenderTrace {
			log.Printf("RenderWindow: window was closed: %v\n", w.name)
		}
		return
	}

	// now we go in the proper bottom-up order to generate the [render.Scene]
	cp := w.SystemWindow.Composer()
	cp.Start()
	sm := &w.mains
	n := sm.stack.Len()

	w.fillInsets(cp) // only does something on non-js

	// first, find the top-level window:
	winIndex := 0
	var winScene *Scene
	for i := n - 1; i >= 0; i-- {
		st := sm.stack.ValueByIndex(i)
		if st.Type == WindowStage {
			if DebugSettings.WindowRenderTrace {
				log.Println("GatherScenes: main Window:", st.String())
			}
			winScene = st.Scene
			winIndex = i
			cp.Add(winScene.RenderSource(draw.Src), winScene)
			for _, dr := range winScene.directRenders {
				cp.Add(dr.RenderSource(draw.Over), dr)
			}
			break
		}
	}

	// then add everyone above that
	for i := winIndex + 1; i < n; i++ {
		st := sm.stack.ValueByIndex(i)
		if st.Scrim && i == n-1 {
			cp.Add(ScrimSource(winScene.Geom.TotalBBox), &st.Scrim)
		}
		cp.Add(st.Scene.RenderSource(draw.Over), st.Scene)
		if DebugSettings.WindowRenderTrace {
			log.Println("GatherScenes: overlay Stage:", st.String())
		}
	}

	// then add the popups for the top main stage
	for _, kv := range top.popups.stack.Order {
		st := kv.Value
		cp.Add(st.Scene.RenderSource(draw.Over), st.Scene)
		if DebugSettings.WindowRenderTrace {
			log.Println("GatherScenes: popup:", st.String())
		}
	}
	cp.Add(SpritesSource(top, winScene), &top.Sprites)
	w.lastSpriteDraw = time.Now()

	w.SystemWindow.Unlock()
	if offscreen || w.flags.HasFlag(winResize) || sinceResize < 500*time.Millisecond {
		atomic.AddInt32(&w.winRenderCounter, 1)
		w.renderAsync(cp)
		if w.flags.HasFlag(winResize) {
			w.lastResize = time.Now()
		}
		w.flags.SetFlag(false, winResize)
	} else {
		// note: it is critical to set *before* going into loop
		// because otherwise we can lose an entire pass before the goroutine starts!
		// function will turn flag off when it finishes.
		atomic.AddInt32(&w.winRenderCounter, 1)
		go w.renderAsync(cp)
	}
}

// renderAsync is the implementation of the main render pass,
// which is usually called in a goroutine.
// It calls the Compose function on the given composer.
func (w *renderWindow) renderAsync(cp composer.Composer) {
	if !w.SystemWindow.Lock() {
		atomic.AddInt32(&w.winRenderCounter, -1)
		// fmt.Println("renderAsync SystemWindow lock fail")
		return
	}
	// pr := profile.Start("Compose")
	// fmt.Println("start compose")
	cp.Compose()
	// pr.End()
	w.SystemWindow.Unlock()
	atomic.AddInt32(&w.winRenderCounter, -1)
}

// RenderSource returns the [render.Render] state from the [Scene.Painter].
func (sc *Scene) RenderSource(op draw.Op) composer.Source {
	sc.setFlag(false, sceneImageUpdated)
	return SceneSource(sc, op)
}

// renderWindowFlags are atomic bit flags for [renderWindow] state.
// They must be atomic to prevent race conditions.
type renderWindowFlags int64 //enums:bitflag -trim-prefix win

const (
	// winResize indicates that the window was just resized.
	winResize renderWindowFlags = iota

	// winStopEventLoop indicates that the event loop should be stopped.
	winStopEventLoop

	// winClosing is whether the window is closing.
	winClosing

	// winGotFocus indicates that have we received focus.
	winGotFocus

	// winRenderSkipped indicates that a render update was skipped, so
	// another update will be run to ensure full updating.
	winRenderSkipped
)
