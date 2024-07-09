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
// This should be put at the end of the main function, and is typically
// called through [Stage.Wait].
func Wait() {
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
// renderWindow contents are all managed by the [Stages] stack that
// handles main [Stage] elements such as [WindowStage] and [DialogStage], which in
// turn manage their own stack of popup stage elements such as menus and tooltips.
// The contents of each Stage is provided by a Scene, containing Widgets,
// and the Stage Pixels image is drawn to the renderWindow in the renderWindow method.
//
// Rendering is handled by the [system.Drawer]. It is akin to a window manager overlaying Go image bitmaps
// on top of each other in the proper order, based on the [Stages] stacking order.
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
	mains Stages

	// renderScenes are the Scene elements that draw directly to the window,
	// arranged in order, and continuously updated during Render.
	renderScenes RenderScenes

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
		rc := w.RenderContext()
		rc.Lock()
		defer rc.Unlock()
		w.closing = true
		// ensure that everyone is closed first
		for _, kv := range w.mains.Stack.Order {
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
	w.renderScenes.MaxIndex = system.MaxTexturesPerSet * 2 // reserve last for sprites

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
	top := w.mains.Top()
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
	ew, got := MainRenderWindows.FindData(data)
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
		wgp := TheWindowGeometrySaver.Pref(w.title, w.SystemWindow.Screen())
		if wgp != nil {
			TheWindowGeometrySaver.SettingStart()
			if w.SystemWindow.Size() != wgp.Size() || w.SystemWindow.Position() != wgp.Pos() {
				if DebugSettings.WinGeomTrace {
					log.Printf("WindowGeometry: SetName setting geom for window: %v pos: %v size: %v\n", w.name, wgp.Pos(), wgp.Size())
				}
				w.SystemWindow.SetGeom(wgp.Pos(), wgp.Size())
				system.TheApp.SendEmptyEvent()
			}
			TheWindowGeometrySaver.SettingEnd()
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
		b.NewSnackbar(ms).Run()
	}
}

// resized updates internal buffers after a window has been resized.
func (w *renderWindow) resized() {
	rc := w.RenderContext()
	if !w.isVisible() {
		rc.Visible = false
		return
	}

	drw := w.SystemWindow.Drawer()

	rg := w.SystemWindow.RenderGeom()

	curRg := rc.Geom
	if curRg == rg {
		if DebugSettings.WinEventTrace {
			fmt.Printf("Win: %v skipped same-size Resized: %v\n", w.name, curRg)
		}
		// still need to apply style even if size is same
		for _, kv := range w.mains.Stack.Order {
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
		rc.Visible = false
		if DebugSettings.WinEventTrace {
			fmt.Printf("Win: %v Resized already closed\n", w.name)
		}
		return
	}
	if DebugSettings.WinEventTrace {
		fmt.Printf("Win: %v Resized from: %v to: %v\n", w.name, curRg, rg)
	}
	rc.Geom = rg
	rc.Visible = true
	rc.LogicalDPI = w.logicalDPI()
	// fmt.Printf("resize dpi: %v\n", w.LogicalDPI())
	w.mains.Resize(rg)
	if DebugSettings.WinGeomTrace {
		log.Printf("WindowGeometry: recording from Resize\n")
	}
	TheWindowGeometrySaver.RecordPref(w)
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
// It firsts unlocks and then locks the [RenderContext] to prevent deadlocks.
// If this is called asynchronously outside of the main event loop,
// [renderWindow.SystemWin.closeReq] should be called directly instead.
func (w *renderWindow) closeReq() {
	rc := w.RenderContext()
	rc.Unlock()
	w.SystemWindow.CloseReq()
	rc.Lock()
}

// closed frees any resources after the window has been closed.
func (w *renderWindow) closed() {
	AllRenderWindows.Delete(w)
	MainRenderWindows.Delete(w)
	DialogRenderWindows.Delete(w)
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

	w.renderScenes.Reset()
	// todo: delete the contents of the window here??
}

// isClosed reports if the window has been closed
func (w *renderWindow) isClosed() bool {
	return w.SystemWindow.IsClosed() || w.mains.Stack.Len() == 0
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
	w.mains.DeleteAll()
}

// handleEvent processes given events.Event.
// All event processing operates under a RenderContext.Lock
// so that no rendering update can occur during event-driven updates.
// Because rendering itself is event driven, this extra level of safety
// is redundant in this case, but other non-event-driven updates require
// the lock protection.
func (w *renderWindow) handleEvent(e events.Event) {
	rc := w.RenderContext()
	rc.Lock()
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
		rc.Unlock()
		return
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
		rc := w.RenderContext()
		rc.Unlock() // one case where we need to break lock
		w.RenderWindow()
		rc.Lock()
		w.mains.SendShowEvents()

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
			TheWindowGeometrySaver.RecordPref(w)
		case events.WinFocus:
			// if we are not already the last in AllRenderWins, we go there,
			// as this allows focus to be restored to us in the future
			if len(AllRenderWindows) > 0 && AllRenderWindows[len(AllRenderWindows)-1] != w {
				AllRenderWindows.Delete(w)
				AllRenderWindows.Add(w)
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
func (rp *renderParams) needsRestyle(rc *RenderContext) bool {
	return rp.logicalDPI != rc.LogicalDPI || rp.geom != rc.Geom
}

// saveRender grabs current render context params
func (rp *renderParams) saveRender(rc *RenderContext) {
	rp.logicalDPI = rc.LogicalDPI
	rp.geom = rc.Geom
}

// RenderContext provides rendering context from outer RenderWindow
// window to Stage and Scene elements to inform styling, layout
// and rendering. It also has the main Mutex for any updates
// to the window contents: use Lock for anything updating.
type RenderContext struct {

	// LogicalDPI is the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units.
	LogicalDPI float32

	// Geometry of the rendering window, in actual "dot" pixels used for rendering.
	Geom math32.Geom2DInt

	// Mu is mutex for locking out rendering and any destructive updates.
	// It is locked at the RenderWindow level during rendering and
	// event processing to provide exclusive blocking of external updates.
	// Use AsyncLock from any outside routine to grab the lock before
	// doing modifications.
	Mu sync.Mutex

	// Visible is whether the window is visible and should be rendered to.
	Visible bool

	// Rebuild is whether to force a rebuild of all Scene elements.
	Rebuild bool
}

// NewRenderContext returns a new RenderContext initialized according to
// the main Screen size and LogicalDPI as initial defaults.
// The actual window size is set during Resized method, which is typically
// called after the window is created by the OS.
func NewRenderContext() *RenderContext {
	rc := &RenderContext{}
	scr := system.TheApp.Screen(0)
	if scr != nil {
		rc.Geom.SetRect(scr.Geometry)
		rc.LogicalDPI = scr.LogicalDPI
	} else {
		rc.Geom = math32.Geom2DInt{Size: image.Pt(1080, 720)}
		rc.LogicalDPI = 160
	}
	rc.Visible = true
	return rc
}

// Lock is called by RenderWindow during RenderWindow and HandleEvent
// when updating all widgets and rendering the screen.
// Any outside access to window contents / scene must acquire this
// lock first.  In general, use AsyncLock to do this.
func (rc *RenderContext) Lock() {
	rc.Mu.Lock()
}

// Unlock must be called for each Lock, when done.
func (rc *RenderContext) Unlock() {
	rc.Mu.Unlock()
}

func (rc *RenderContext) String() string {
	str := fmt.Sprintf("Geom: %s  Visible: %v", rc.Geom, rc.Visible)
	return str
}

//////////////////////////////////////////////////////////////////////
//  RenderScenes

// RenderScenes are a list of Scene and direct rendering widgets,
// compiled in rendering order, whose Pixels images are composed
// directly to the RenderWindow window.
type RenderScenes struct {

	// starting index for this set of Scenes
	StartIndex int

	// max index (exclusive) for this set of Scenes
	MaxIndex int

	// set to true to flip Y axis in drawing these images
	FlipY bool

	// ordered list of scenes and direct rendering widgets. Index is Drawer image index.
	Scenes []Widget

	// SceneIndex holds the index for each scene / direct render widget.
	// Used to detect changes in index.
	SceneIndex map[Widget]int
}

// SetIndexRange sets the index range based on starting index and n
func (rs *RenderScenes) SetIndexRange(st, n int) {
	rs.StartIndex = st
	rs.MaxIndex = st + n
}

// Reset resets the list
func (rs *RenderScenes) Reset() {
	rs.Scenes = nil
	if rs.SceneIndex == nil {
		rs.SceneIndex = make(map[Widget]int)
	}
}

// Add adds a new node, returning index
func (rs *RenderScenes) Add(w Widget, scIndex map[Widget]int) int {
	sc := w.AsWidget().Scene
	if sc.Pixels == nil {
		return -1
	}
	idx := len(rs.Scenes)
	if idx >= rs.MaxIndex {
		slog.Error("RenderScenes: too many Scenes to render all of them!", "max", rs.MaxIndex)
		return -1
	}
	if prvIndex, has := rs.SceneIndex[w]; has {
		if prvIndex != idx {
			sc.imageUpdated = true // need to copy b/c cur has diff image
		}
	} else {
		sc.imageUpdated = true // need to copy b/c new
	}
	scIndex[w] = idx
	rs.Scenes = append(rs.Scenes, w)
	return idx
}

// SetImages calls drw.SetGoImage on all updated Scene images
func (rs *RenderScenes) SetImages(drw system.Drawer) {
	if len(rs.Scenes) == 0 {
		if DebugSettings.WinRenderTrace {
			fmt.Println("RenderScene.SetImages: no scenes")
		}
	}
	var skipScene *Scene
	for i, w := range rs.Scenes {
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

// DrawAll does drw.Copy drawing call for all Scenes,
// using proper TextureSet for each of system.MaxTexturesPerSet Scenes.
func (rs *RenderScenes) DrawAll(drw system.Drawer) {
	nPerSet := system.MaxTexturesPerSet
	if len(rs.Scenes) == 0 {
		return
	}
	for i, w := range rs.Scenes {
		set := i / nPerSet
		if i%nPerSet == 0 && set > 0 {
			drw.UseTextureSet(set)
		}
		w.DirectRenderDraw(drw, i, rs.FlipY)
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
	drw.Copy(idx, 0, sc.SceneGeom.Pos, bb, op, flipY)
}

//////////////////////////////////////////////////////////////////////
//  RenderWindow methods

func (w *renderWindow) RenderContext() *RenderContext {
	return w.mains.RenderContext
}

// RenderWindow performs all rendering based on current Stages config.
// It sets the Write lock on RenderContext Mutex, so nothing else can update
// during this time.  All other updates are done with a Read lock so they
// won't interfere with each other.
func (w *renderWindow) RenderWindow() {
	rc := w.RenderContext()
	rc.Lock()
	defer func() {
		rc.Rebuild = false
		rc.Unlock()
	}()
	rebuild := rc.Rebuild

	stageMods, sceneMods := w.mains.UpdateAll() // handles all Scene / Widget updates!
	top := w.mains.Top()
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
		if !w.GatherScenes() {
			slog.Error("RenderWindow: no scenes")
			return
		}
	}
	w.DrawScenes()
}

// DrawScenes does the drawing of RenderScenes to the window.
func (w *renderWindow) DrawScenes() {
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

	rs.SetImages(drw) // ensure all updated images copied

	top := w.mains.Top()
	if top.Sprites.Modified {
		top.Sprites.ConfigSprites(drw)
	}

	drw.SyncImages()
	w.FillInsets()
	if !drw.StartDraw(0) {
		return
	}
	drw.UseTextureSet(0)
	rs.DrawAll(drw)

	drw.UseTextureSet(2)
	top.Sprites.DrawSprites(drw)

	drw.EndDraw()

	// pr.End()
}

// FillInsets fills the window insets, if any, with [colors.Scheme.Background].
func (w *renderWindow) FillInsets() {
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

// GatherScenes finds all the Scene elements that drive rendering
// into the RenderScenes list.  Returns false on failure / nothing to render.
func (w *renderWindow) GatherScenes() bool {
	rs := &w.renderScenes
	rs.Reset()
	scIndex := make(map[Widget]int)

	sm := &w.mains
	n := sm.Stack.Len()
	if n == 0 {
		slog.Error("GatherScenes stack empty")
		return false // shouldn't happen!
	}

	// first, find the top-level window:
	winIndex := 0
	var winScene *Scene
	for i := n - 1; i >= 0; i-- {
		st := sm.Stack.ValueByIndex(i)
		if st.Type == WindowStage {
			if DebugSettings.WinRenderTrace {
				fmt.Println("GatherScenes: main Window:", st.String())
			}
			winScene = st.Scene
			rs.Add(st.Scene, scIndex)
			for _, w := range st.Scene.directRenders {
				rs.Add(w, scIndex)
			}
			winIndex = i
			break
		}
	}

	// then add everyone above that
	for i := winIndex + 1; i < n; i++ {
		st := sm.Stack.ValueByIndex(i)
		if st.Scrim && i == n-1 {
			rs.Add(NewScrim(winScene), scIndex)
		}
		rs.Add(st.Scene, scIndex)
		if DebugSettings.WinRenderTrace {
			fmt.Println("GatherScenes: overlay Stage:", st.String())
		}
	}

	top := sm.Top()
	top.Sprites.Modified = true // ensure configured

	// then add the popups for the top main stage
	for _, kv := range top.Popups.Stack.Order {
		st := kv.Value
		rs.Add(st.Scene, scIndex)
		if DebugSettings.WinRenderTrace {
			fmt.Println("GatherScenes: popup:", st.String())
		}
	}
	rs.SceneIndex = scIndex
	return true
}

////////////////////////////////////////////////////////////////////////////
//  Scrim

// A Scrim is just a dummy Widget used for rendering a Scrim.
// Only used for its type. Everything else managed by RenderWindow.
type Scrim struct { //core:no-new
	WidgetBase
}

// NewScrim creates a new Scrim for use in rendering.
// It does not actually add the Scrim to the Scene,
// just sets its pointers.
func NewScrim(sc *Scene) *Scrim {
	sr := tree.New[Scrim]() // critical to not add to scene!
	tree.SetParent(sr, sc)
	return sr
}

func (sr *Scrim) DirectRenderImage(drw system.Drawer, idx int) {
	// no-op
}

func (sr *Scrim) DirectRenderDraw(drw system.Drawer, idx int, flipY bool) {
	sc := sr.Parent.(*Scene)
	drw.Fill(colors.ApplyOpacity(colors.ToUniform(colors.Scheme.Scrim), 0.5), math32.Identity3(), sc.Geom.TotalBBox, draw.Over)
}
