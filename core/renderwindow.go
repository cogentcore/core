// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"log"
	"log/slog"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
	"golang.org/x/image/draw"
)

// windowWait is a wait group for waiting for all the open window event
// loops to finish. It is incremented by [renderWindow.GoStartEventLoop]
// and decremented when the event loop terminates.
var windowWait sync.WaitGroup

// Wait waits for all windows to close and runs the main app loop.
// This should be put at the end of the main function if
// [Body.RunMainWindow] is not used.
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

// currentRenderWindow is the current [renderWindow].
// On single window platforms (mobile, web, and offscreen),
// this is the only render window.
var currentRenderWindow *renderWindow

// renderWindowGlobalMu is a mutex for any global state associated with windows
var renderWindowGlobalMu sync.Mutex

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

	// renderScenes are the Scene elements that draw directly to the window,
	// arranged in order, and continuously updated during Render.
	renderScenes renderScenes

	// noEventsChan is a channel on which a signal is sent when there are
	// no events left in the window [events.Deque]. It is used internally
	// for event handling in tests.
	noEventsChan chan struct{}

	// closing is whether the window is closing.
	closing bool

	// gotFocus indicates that have we received focus.
	gotFocus bool

	// stopEventLoop indicates that the event loop should be stopped.
	stopEventLoop bool
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
		rc.lock()
		defer rc.unlock()
		w.closing = true
		// ensure that everyone is closed first
		for _, kv := range w.mains.stack.Order {
			if kv.Value == nil || kv.Value.Scene == nil || kv.Value.Scene.This == nil {
				continue
			}
			if !kv.Value.Scene.Close() {
				w.closing = false
				return
			}
		}
		win.Close()
	})

	drw := w.SystemWindow.Drawer()
	drw.SetMaxTextures(system.MaxTexturesPerSet * 3)       // use 3 sets
	w.renderScenes.maxIndex = system.MaxTexturesPerSet * 2 // reserve last for sprites

	// win.SystemWin.SetDestroyGPUResourcesFunc(func() {
	// 	for _, ph := range win.Phongs {
	// 		ph.Destroy()
	// 	}
	// 	for _, fr := range win.Frames {
	// 		fr.Destroy()
	// 	}
	// })
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
	if DebugSettings.WinEventTrace {
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
	if isdif && w.SystemWindow != nil {
		wgp := theWindowGeometrySaver.pref(w.title, w.SystemWindow.Screen())
		if wgp != nil {
			theWindowGeometrySaver.settingStart()
			if w.SystemWindow.Size() != wgp.size() || w.SystemWindow.Position() != wgp.pos() {
				if DebugSettings.WinGeomTrace {
					log.Printf("WindowGeometry: SetName setting geom for window: %v pos: %v size: %v\n", w.name, wgp.pos(), wgp.size())
				}
				w.SystemWindow.SetGeom(wgp.pos(), wgp.size())
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
	if title != w.title {
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
	w.setZoom(AppearanceSettings.Zoom + 10*steps)
}

// setZoom sets [AppearanceSettingsData.Zoom] to the given value and then triggers
// necessary updating and makes a snackbar.
func (w *renderWindow) setZoom(zoom float32) {
	AppearanceSettings.Zoom = math32.Clamp(zoom, 10, 500)
	AppearanceSettings.Apply()
	UpdateAll()
	errors.Log(SaveSettings(AppearanceSettings))

	if ms := w.MainScene(); ms != nil {
		b := NewBody().AddSnackbarText(fmt.Sprintf("%.f%%", AppearanceSettings.Zoom))
		NewStretch(b)
		b.AddSnackbarIcon(icons.Remove, func(e events.Event) {
			w.stepZoom(-1)
		})
		b.AddSnackbarIcon(icons.Add, func(e events.Event) {
			w.stepZoom(1)
		})
		b.AddSnackbarButton("Reset", func(e events.Event) {
			w.setZoom(100)
		})
		b.DeleteChildByName("stretch")
		b.RunSnackbar(ms)
	}
}

// resized updates internal buffers after a window has been resized.
func (w *renderWindow) resized() {
	rc := w.renderContext()
	if !w.isVisible() {
		rc.visible = false
		return
	}

	drw := w.SystemWindow.Drawer()

	rg := w.SystemWindow.RenderGeom()

	curRg := rc.geom
	if curRg == rg {
		if DebugSettings.WinEventTrace {
			fmt.Printf("Win: %v skipped same-size Resized: %v\n", w.name, curRg)
		}
		// still need to apply style even if size is same
		for _, kv := range w.mains.stack.Order {
			sc := kv.Value.Scene
			sc.applyStyleScene()
		}
		return
	}
	if drw.MaxTextures() != system.MaxTexturesPerSet*3 { // this is essential after hibernate
		drw.SetMaxTextures(system.MaxTexturesPerSet * 3) // use 3 sets
	}
	// w.FocusInactivate()
	// w.InactivateAllSprites()
	if !w.isVisible() {
		rc.visible = false
		if DebugSettings.WinEventTrace {
			fmt.Printf("Win: %v Resized already closed\n", w.name)
		}
		return
	}
	if DebugSettings.WinEventTrace {
		fmt.Printf("Win: %v Resized from: %v to: %v\n", w.name, curRg, rg)
	}
	rc.geom = rg
	rc.visible = true
	rc.logicalDPI = w.logicalDPI()
	// fmt.Printf("resize dpi: %v\n", w.LogicalDPI())
	w.mains.resize(rg)
	if DebugSettings.WinGeomTrace {
		log.Printf("WindowGeometry: recording from Resize\n")
	}
	theWindowGeometrySaver.recordPref(w)
}

// Raise requests that the window be at the top of the stack of windows,
// and receive focus. If it is minimized, it will be un-minimized. This
// is the only supported mechanism for un-minimizing. This also sets
// [currentRenderWindow] to the window.
func (w *renderWindow) Raise() {
	w.SystemWindow.Raise()
	currentRenderWindow = w
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
	rc.unlock()
	w.SystemWindow.CloseReq()
	rc.lock()
}

// closed frees any resources after the window has been closed.
func (w *renderWindow) closed() {
	AllRenderWindows.delete(w)
	mainRenderWindows.delete(w)
	dialogRenderWindows.delete(w)
	if DebugSettings.WinEventTrace {
		fmt.Printf("Win: %v Closed\n", w.name)
	}
	if len(AllRenderWindows) > 0 {
		pfw := AllRenderWindows[len(AllRenderWindows)-1]
		if DebugSettings.WinEventTrace {
			fmt.Printf("Win: %v getting restored focus after: %v closed\n", pfw.name, w.name)
		}
		pfw.Raise()
	}
	// these are managed by the window itself
	// w.Sprites.Reset()

	w.renderScenes.reset()
	// todo: delete the contents of the window here??
}

// isClosed reports if the window has been closed
func (w *renderWindow) isClosed() bool {
	return w.SystemWindow.IsClosed() || w.mains.stack.Len() == 0
}

// isVisible is the main visibility check; don't do any window updates if not visible!
func (w *renderWindow) isVisible() bool {
	if w == nil || w.SystemWindow == nil || w.isClosed() || w.closing || !w.SystemWindow.IsVisible() {
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
		if w.stopEventLoop {
			w.stopEventLoop = false
			break
		}
		e := d.NextEvent()
		if w.stopEventLoop {
			w.stopEventLoop = false
			break
		}
		w.handleEvent(e)
		if w.noEventsChan != nil && len(d.Back) == 0 && len(d.Front) == 0 {
			w.noEventsChan <- struct{}{}
		}
	}
	if DebugSettings.WinEventTrace {
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
	rc.lock()
	// we manually handle Unlock's in this function instead of deferring
	// it to avoid a cryptic "sync: can't unlock an already unlocked Mutex"
	// error when panicking in the rendering goroutine. This is critical for
	// debugging on Android. TODO: maybe figure out a more sustainable approach to this.

	et := e.Type()
	if DebugSettings.EventTrace && et != events.WindowPaint && et != events.MouseMove {
		fmt.Println("Window got event", e)
	}
	if et >= events.Window && et <= events.WindowPaint {
		w.handleWindowEvents(e)
		rc.unlock()
		return
	}
	// fmt.Printf("got event type: %v: %v\n", et.BitIndexString(), evi)
	w.mains.mainHandleEvent(e)
	rc.unlock()
}

func (w *renderWindow) handleWindowEvents(e events.Event) {
	et := e.Type()
	switch et {
	case events.WindowPaint:
		e.SetHandled()
		rc := w.renderContext()
		rc.unlock() // one case where we need to break lock
		w.renderWindow()
		rc.lock()
		w.mains.sendShowEvents()

	case events.WindowResize:
		e.SetHandled()
		w.resized()

	case events.Window:
		ev := e.(*events.WindowEvent)
		switch ev.Action {
		case events.WinClose:
			// fmt.Printf("got close event for window %v \n", w.Name)
			e.SetHandled()
			w.stopEventLoop = true
			w.closed()
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
			if DebugSettings.WinEventTrace {
				fmt.Printf("Win: %v got show event\n", w.name)
			}
		case events.WinMove:
			e.SetHandled()
			// fmt.Printf("win move: %v\n", w.SystemWin.Position())
			if DebugSettings.WinGeomTrace {
				log.Printf("WindowGeometry: recording from Move\n")
			}
			theWindowGeometrySaver.recordPref(w)
		case events.WinFocus:
			// if we are not already the last in AllRenderWins, we go there,
			// as this allows focus to be restored to us in the future
			if len(AllRenderWindows) > 0 && AllRenderWindows[len(AllRenderWindows)-1] != w {
				AllRenderWindows.delete(w)
				AllRenderWindows.add(w)
			}
			if !w.gotFocus {
				w.gotFocus = true
				w.sendWinFocusEvent(events.WinFocus)
				if DebugSettings.WinEventTrace {
					fmt.Printf("Win: %v got focus\n", w.name)
				}
			} else {
				if DebugSettings.WinEventTrace {
					fmt.Printf("Win: %v got extra focus\n", w.name)
				}
			}
			currentRenderWindow = w
		case events.WinFocusLost:
			if DebugSettings.WinEventTrace {
				fmt.Printf("Win: %v lost focus\n", w.name)
			}
			w.gotFocus = false
			w.sendWinFocusEvent(events.WinFocusLost)
		case events.ScreenUpdate:
			w.resized()
			// TODO: figure out how to restore this stuff without breaking window size on mobile

			// TheWindowGeometryaver.AbortSave() // anything just prior to this is sus
			// if !system.TheApp.NoScreens() {
			// 	Settings.UpdateAll()
			// 	WindowGeometrySave.RestoreAll()
			// }
		}
	}
}

/////////////////////////////////////////////////////////////////////////////
//                   Rendering

const (
	// Sprites are stored as arrays of same-sized textures,
	// allocated by size in Set 2, starting at 32
	spriteStart = system.MaxTexturesPerSet * 2
)

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

	// mu is mutex for locking out rendering and any destructive updates.
	// It is locked at the RenderWindow level during rendering and
	// event processing to provide exclusive blocking of external updates.
	// Use AsyncLock from any outside routine to grab the lock before
	// doing modifications.
	mu sync.Mutex

	// visible is whether the window is visible and should be rendered to.
	visible bool

	// rebuild is whether to force a rebuild of all Scene elements.
	rebuild bool
}

// newRenderContext returns a new [renderContext] initialized according to
// the main Screen size and LogicalDPI as initial defaults.
// The actual window size is set during Resized method, which is typically
// called after the window is created by the OS.
func newRenderContext() *renderContext {
	rc := &renderContext{}
	scr := system.TheApp.Screen(0)
	if scr != nil {
		rc.geom.SetRect(scr.Geometry)
		rc.logicalDPI = scr.LogicalDPI
	} else {
		rc.geom = math32.Geom2DInt{Size: image.Pt(1080, 720)}
		rc.logicalDPI = 160
	}
	rc.visible = true
	return rc
}

// lock is called by RenderWindow during RenderWindow and HandleEvent
// when updating all widgets and rendering the screen.
// Any outside access to window contents / scene must acquire this
// lock first.  In general, use AsyncLock to do this.
func (rc *renderContext) lock() {
	rc.mu.Lock()
}

// unlock must be called for each Lock, when done.
func (rc *renderContext) unlock() {
	rc.mu.Unlock()
}

func (rc *renderContext) String() string {
	str := fmt.Sprintf("Geom: %s  Visible: %v", rc.geom, rc.visible)
	return str
}

// renderScenes are a list of Scene and direct rendering widgets,
// compiled in rendering order, whose Pixels images are composed
// directly to the RenderWindow window.
type renderScenes struct {

	// max index (exclusive) for this set of Scenes
	maxIndex int

	// set to true to flip Y axis in drawing these images
	flipY bool

	// ordered list of scenes and direct rendering widgets. Index is Drawer image index.
	scenes []Widget

	// sceneIndex holds the index for each scene / direct render widget.
	// Used to detect changes in index.
	sceneIndex map[Widget]int
}

// reset resets the list
func (rs *renderScenes) reset() {
	rs.scenes = nil
	if rs.sceneIndex == nil {
		rs.sceneIndex = make(map[Widget]int)
	}
}

// add adds a new node, returning index
func (rs *renderScenes) add(w Widget, scIndex map[Widget]int) int {
	sc := w.AsWidget().Scene
	if sc.Pixels == nil {
		return -1
	}
	idx := len(rs.scenes)
	if idx >= rs.maxIndex {
		slog.Error("RenderScenes: too many Scenes to render all of them!", "max", rs.maxIndex)
		return -1
	}
	if prvIndex, has := rs.sceneIndex[w]; has {
		if prvIndex != idx {
			sc.imageUpdated = true // need to copy b/c cur has diff image
		}
	} else {
		sc.imageUpdated = true // need to copy b/c new
	}
	scIndex[w] = idx
	rs.scenes = append(rs.scenes, w)
	return idx
}

// setImages calls drw.SetGoImage on all updated Scene images
func (rs *renderScenes) setImages(drw system.Drawer) {
	if len(rs.scenes) == 0 {
		if DebugSettings.WinRenderTrace {
			fmt.Println("RenderScene.SetImages: no scenes")
		}
	}
	var skipScene *Scene
	for i, w := range rs.scenes {
		sc := w.AsWidget().Scene
		_, isSc := w.(*Scene)
		if isSc && (sc.updating || !sc.imageUpdated) {
			if DebugSettings.WinRenderTrace {
				if sc.updating {
					fmt.Println("RenderScenes.SetImages: sc IsUpdating", sc.Name)
				}
				if !sc.imageUpdated {
					fmt.Println("RenderScenes.SetImages: sc Image NotUpdated", sc.Name)
				}
			}
			skipScene = sc
			continue
		}
		if DebugSettings.WinRenderTrace {
			fmt.Println("RenderScenes.SetImages:", sc.Name)
		}
		if isSc || sc != skipScene {
			w.DirectRenderImage(drw, i)
		}
	}
}

// drawAll does drw.Copy drawing call for all Scenes,
// using proper TextureSet for each of system.MaxTexturesPerSet Scenes.
func (rs *renderScenes) drawAll(drw system.Drawer) {
	nPerSet := system.MaxTexturesPerSet
	if len(rs.scenes) == 0 {
		return
	}
	for i, w := range rs.scenes {
		set := i / nPerSet
		if i%nPerSet == 0 && set > 0 {
			drw.UseTextureSet(set)
		}
		w.DirectRenderDraw(drw, i, rs.flipY)
	}
}

func (sc *Scene) DirectRenderImage(drw system.Drawer, idx int) {
	drw.SetGoImage(idx, 0, sc.Pixels, system.NoFlipY)
	sc.imageUpdated = false
}

func (sc *Scene) DirectRenderDraw(drw system.Drawer, idx int, flipY bool) {
	op := draw.Over
	if idx == 0 {
		op = draw.Src
	}
	bb := sc.Pixels.Bounds()
	drw.Copy(idx, 0, sc.sceneGeom.Pos, bb, op, flipY)
}

func (w *renderWindow) renderContext() *renderContext {
	return w.mains.renderContext
}

// renderWindow performs all rendering based on current Stages config.
// It sets the Write lock on RenderContext Mutex, so nothing else can update
// during this time.  All other updates are done with a Read lock so they
// won't interfere with each other.
func (w *renderWindow) renderWindow() {
	rc := w.renderContext()
	rc.lock()
	defer func() {
		rc.rebuild = false
		rc.unlock()
	}()
	rebuild := rc.rebuild

	stageMods, sceneMods := w.mains.updateAll() // handles all Scene / Widget updates!
	top := w.mains.top()
	if top == nil {
		return
	}
	if !top.Sprites.Modified && !rebuild && !stageMods && !sceneMods { // nothing to do!
		// fmt.Println("no mods") // note: get a ton of these..
		return
	}

	if DebugSettings.WinRenderTrace {
		fmt.Println("RenderWindow: doing render:", w.name)
		fmt.Println("rebuild:", rebuild, "stageMods:", stageMods, "sceneMods:", sceneMods)
	}

	if stageMods || rebuild {
		if !w.gatherScenes() {
			slog.Error("RenderWindow: no scenes")
			return
		}
	}
	w.drawScenes()
}

// drawScenes does the drawing of RenderScenes to the window.
func (w *renderWindow) drawScenes() {
	if !w.isVisible() || w.SystemWindow.Is(system.Minimized) {
		if DebugSettings.WinRenderTrace {
			fmt.Printf("RenderWindow: skipping update on inactive / minimized window: %v\n", w.name)
		}
		return
	}
	// if !w.HasFlag(WinSentShow) {
	// 	return
	// }
	if !w.SystemWindow.Lock() {
		if DebugSettings.WinRenderTrace {
			fmt.Printf("RenderWindow: window was closed: %v\n", w.name)
		}
		return
	}
	defer w.SystemWindow.Unlock()

	// pr := profile.Start("win.DrawScenes")

	drw := w.SystemWindow.Drawer()
	rs := &w.renderScenes

	rs.setImages(drw) // ensure all updated images copied

	top := w.mains.top()
	if top.Sprites.Modified {
		top.Sprites.configSprites(drw)
	}

	drw.SyncImages()
	w.fillInsets()
	if !drw.StartDraw(0) {
		return
	}
	drw.UseTextureSet(0)
	rs.drawAll(drw)

	drw.UseTextureSet(2)
	top.Sprites.drawSprites(drw)

	drw.EndDraw()

	// pr.End()
}

// fillInsets fills the window insets, if any, with [colors.Scheme.Background].
func (w *renderWindow) fillInsets() {
	// render geom and window geom
	rg := w.SystemWindow.RenderGeom()
	wg := math32.Geom2DInt{Size: w.SystemWindow.Size()}

	// if our window geom is the same as our render geom, we have no
	// window insets to fill
	if wg == rg {
		return
	}

	drw := w.SystemWindow.Drawer()
	if !drw.StartFill() {
		return
	}

	fill := func(x0, y0, x1, y1 int) {
		r := image.Rect(x0, y0, x1, y1)
		if r.Dx() == 0 || r.Dy() == 0 {
			return
		}
		drw.Fill(colors.ToUniform(colors.Scheme.Background), math32.Identity3(), r, draw.Src)
	}
	rb := rg.Bounds()
	wb := wg.Bounds()
	fill(0, 0, wb.Max.X, rb.Min.Y)        // top
	fill(0, rb.Max.Y, wb.Max.X, wb.Max.Y) // bottom
	fill(rb.Max.X, 0, wb.Max.X, wb.Max.Y) // right
	fill(0, 0, rb.Min.X, wb.Max.Y)        // left

	drw.EndFill()
}

// gatherScenes finds all the Scene elements that drive rendering
// into the RenderScenes list.  Returns false on failure / nothing to render.
func (w *renderWindow) gatherScenes() bool {
	rs := &w.renderScenes
	rs.reset()
	scIndex := make(map[Widget]int)

	sm := &w.mains
	n := sm.stack.Len()
	if n == 0 {
		slog.Error("GatherScenes stack empty")
		return false // shouldn't happen!
	}

	// first, find the top-level window:
	winIndex := 0
	var winScene *Scene
	for i := n - 1; i >= 0; i-- {
		st := sm.stack.ValueByIndex(i)
		if st.Type == WindowStage {
			if DebugSettings.WinRenderTrace {
				fmt.Println("GatherScenes: main Window:", st.String())
			}
			winScene = st.Scene
			rs.add(st.Scene, scIndex)
			for _, w := range st.Scene.directRenders {
				rs.add(w, scIndex)
			}
			winIndex = i
			break
		}
	}

	// then add everyone above that
	for i := winIndex + 1; i < n; i++ {
		st := sm.stack.ValueByIndex(i)
		if st.Scrim && i == n-1 {
			rs.add(newScrim(winScene), scIndex)
		}
		rs.add(st.Scene, scIndex)
		if DebugSettings.WinRenderTrace {
			fmt.Println("GatherScenes: overlay Stage:", st.String())
		}
	}

	top := sm.top()
	top.Sprites.Modified = true // ensure configured

	// then add the popups for the top main stage
	for _, kv := range top.popups.stack.Order {
		st := kv.Value
		rs.add(st.Scene, scIndex)
		if DebugSettings.WinRenderTrace {
			fmt.Println("GatherScenes: popup:", st.String())
		}
	}
	rs.sceneIndex = scIndex
	return true
}

// A scrim is just a dummy Widget used for rendering a scrim.
// Only used for its type. Everything else managed by [renderWindow].
type scrim struct { //core:no-new
	WidgetBase
}

// newScrim creates a new [scrim] for use in rendering.
// It does not actually add the Scrim to the Scene,
// just sets its pointers.
func newScrim(sc *Scene) *scrim {
	sr := tree.New[scrim]() // critical to not add to scene!
	tree.SetParent(sr, sc)
	return sr
}

func (sr *scrim) DirectRenderImage(drw system.Drawer, idx int) {
	// no-op
}

func (sr *scrim) DirectRenderDraw(drw system.Drawer, idx int, flipY bool) {
	sc := sr.Parent.(*Scene)
	drw.Fill(colors.ApplyOpacity(colors.ToUniform(colors.Scheme.Scrim), 0.5), math32.Identity3(), sc.Geom.TotalBBox, draw.Over)
}
