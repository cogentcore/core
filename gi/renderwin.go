// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/mat32"
	"golang.org/x/image/draw"
)

// WinWait is a wait group for waiting for all the open window event
// loops to finish. It is incremented by [RenderWin.GoStartEventLoop]
// and decremented when the event loop terminates.
var WinWait sync.WaitGroup

// Wait waits for all windows to close and runs the main app loop.
// This should be put at the end of the main function, and is typically
// called through [Stage.Wait].
func Wait() {
	defer func() { goosi.HandleRecover(recover()) }()
	go func() {
		defer func() { goosi.HandleRecover(recover()) }()
		WinWait.Wait()
		goosi.TheApp.Quit()
	}()
	goosi.TheApp.MainLoop()
}

// CurRenderWin is the current RenderWin window.
// On single window platforms (mobile, web, and offscreen),
// this is the _only_ render window.
var CurRenderWin *RenderWin

var (

	// WinNewCloseTime records last time a new window was opened or another
	// closed -- used to trigger updating of RenderWin menus on each window.
	WinNewCloseTime time.Time

	// RenderWinGlobalMu is a mutex for any global state associated with windows
	RenderWinGlobalMu sync.Mutex

	// RenderWinOpenTimer is used for profiling the open time of windows
	// if doing profiling, it will report the time elapsed in milliseconds
	// to point of establishing initial focus in the window.
	RenderWinOpenTimer time.Time
)

// RenderWin provides an outer "actual" window where everything is rendered,
// and is the point of entry for all events coming in from user actions.
//
// RenderWin contents are all managed by the StageMgr that
// handles Main Stage elements such as Window, Dialog, and Sheet, which in
// turn manage their own stack of Popup Stage elements such as Menu, Tooltip, etc.
// The contents of each Stage is provided by a Scene, containing Widgets,
// and the Stage Pixels image is drawn to the RenderWin in the RenderWindow method.
//
// Rendering is handled by the vdraw.Drawer from the vgpu package, which is provided
// by the goosi framework.  It is akin to a window manager overlaying Go image bitmaps
// on top of each other in the proper order, based on the StageMgr stacking order.
//   - Sprites are managed by the Main Stage, as layered textures of the same size,
//     to enable unlimited number packed into a few descriptors for standard sizes.
type RenderWin struct {
	Flags WinFlags

	Name string

	// displayed name of window, for window manager etc.
	// Window object name is the internal handle and is used for tracking property info etc
	Title string

	// OS-specific window interface -- handles all the os-specific functions, including delivering events etc
	GoosiWin goosi.Window `json:"-" xml:"-"`

	// MainStageMgr controlling the Main Stage elements in this window.
	// The Render Context in this manager is the original source for all Stages.
	MainStageMgr StageMgr

	// RenderScenes are the Scene elements that draw directly to the window,
	// arranged in order, and continuously updated during Render.
	RenderScenes RenderScenes

	// below are internal vars used during the event loop

	// NoEventsChan is a channel on which a signal is sent when there are
	// no events left in the window [events.Deque]. It is used internally
	// for event handling in tests, and should typically not be used by
	// end-users.
	NoEventsChan chan struct{}

	// todo: need some other way of freeing GPU resources -- this is not clean:
	// // the phongs for the window
	// Phongs []*vphong.Phong ` json:"-" xml:"-" desc:"the phongs for the window"`
	//
	// // the render frames for the window
	// Frames []*vgpu.RenderFrame ` json:"-" xml:"-" desc:"the render frames for the window"`
}

// WinFlags represent RenderWin state
type WinFlags int64 //enums:bitflag -trim-prefix Win

const (
	// WinHasGeomPrefs indicates if this window has WinGeomPrefs setting that
	// sized it -- affects whether other default geom should be applied.
	WinHasGeomPrefs WinFlags = iota

	// WinClosing is atomic flag indicating window is closing
	WinClosing

	// WinResizing is atomic flag indicating window is resizing
	WinResizing

	// WinGotFocus indicates that have we received RenderWin focus
	WinGotFocus

	// WinSentShow have we sent the show event yet?  Only ever sent ONCE
	WinSentShow

	// WinGoLoop true if we are running from GoStartEventLoop -- requires a WinWait.Done at end
	WinGoLoop

	// WinStopEventLoop is set when event loop stop is requested
	WinStopEventLoop

	// WinSelectionMode indicates that the window is in Cogent Core inspect editor edit mode
	WinSelectionMode
)

// HasFlag returns true if given flag is set
func (w *RenderWin) HasFlag(flag enums.BitFlag) bool {
	return w.Flags.HasFlag(flag)
}

// Is returns true if given flag is set
func (w *RenderWin) Is(flag enums.BitFlag) bool {
	return w.Flags.HasFlag(flag)
}

// SetFlag sets given flag(s) on or off
func (w *RenderWin) SetFlag(on bool, flag ...enums.BitFlag) {
	w.Flags.SetFlag(on, flag...)
}

/////////////////////////////////////////////////////////////////////////////
//                   New RenderWins and Init

// NewRenderWin creates a new window with given internal name handle,
// display name, and options.  This is called by Stage.NewRenderWin
// which handles setting the opts and other infrastructure.
func NewRenderWin(name, title string, opts *goosi.NewWindowOptions) *RenderWin {
	w := &RenderWin{}
	w.Name = name
	w.Title = title
	var err error
	w.GoosiWin, err = goosi.TheApp.NewWindow(opts)
	if err != nil {
		fmt.Printf("Cogent Core NewRenderWin error: %v \n", err)
		return nil
	}
	w.GoosiWin.SetName(title)
	w.GoosiWin.SetTitleBarIsDark(matcolor.SchemeIsDark)
	w.GoosiWin.SetCloseReqFunc(func(win goosi.Window) {
		w.SetFlag(true, WinClosing)
		// ensure that everyone is closed first
		for _, kv := range w.MainStageMgr.Stack.Order {
			if !kv.Value.Scene.Close() {
				w.SetFlag(false, WinClosing)
				return
			}
		}
		win.Close()
	})

	drw := w.GoosiWin.Drawer()
	drw.SetMaxTextures(goosi.MaxTexturesPerSet * 3)     // use 3 sets
	w.RenderScenes.MaxIdx = goosi.MaxTexturesPerSet * 2 // reserve last for sprites

	// win.GoosiWin.SetDestroyGPUResourcesFunc(func() {
	// 	for _, ph := range win.Phongs {
	// 		ph.Destroy()
	// 	}
	// 	for _, fr := range win.Frames {
	// 		fr.Destroy()
	// 	}
	// })
	return w
}

// MainScene returns the current MainStageMgr Top Scene,
// which is the current Window or FullWindow Dialog occupying the RenderWin.
func (w *RenderWin) MainScene() *Scene {
	top := w.MainStageMgr.Top()
	if top == nil {
		return nil
	}
	return top.Scene
}

// ActivateExistingMainWindow looks for existing window with given Data.
// If found brings that to the front, returns true for bool, else false.
func ActivateExistingMainWindow(data any) bool {
	if data == nil {
		return false
	}
	ew, has := MainRenderWins.FindData(data)
	if !has {
		return false
	}
	if DebugSettings.WinEventTrace {
		fmt.Printf("Win: %v getting recycled based on data match\n", ew.Name)
	}
	ew.Raise()
	return true
}

// ActivateExistingDialogWindow looks for existing dialog window with given Data.
// If found brings that to the front, returns true for bool, else false.
func ActivateExistingDialogWindow(data any) bool {
	if data == nil {
		return false
	}
	ew, has := DialogRenderWins.FindData(data)
	if !has {
		return false
	}
	if DebugSettings.WinEventTrace {
		fmt.Printf("Win: %v getting recycled based on data match\n", ew.Name)
	}
	ew.Raise()
	return true
}

// SetName sets name of this window and also the RenderWin, and applies any window
// geometry settings associated with the new name if it is different from before
func (w *RenderWin) SetName(name string) {
	curnm := w.Name
	isdif := curnm != name
	w.Name = name
	if w.GoosiWin != nil {
		w.GoosiWin.SetName(name)
	}
	if isdif && w.GoosiWin != nil {
		wgp := WinGeomMgr.Pref(w.Title, w.GoosiWin.Screen())
		if wgp != nil {
			WinGeomMgr.SettingStart()
			if w.GoosiWin.Size() != wgp.Size() || w.GoosiWin.Position() != wgp.Pos() {
				if DebugSettings.WinGeomTrace {
					log.Printf("WinGeomPrefs: SetName setting geom for window: %v pos: %v size: %v\n", w.Name, wgp.Pos(), wgp.Size())
				}
				w.GoosiWin.SetGeom(wgp.Pos(), wgp.Size())
				goosi.TheApp.SendEmptyEvent()
			}
			WinGeomMgr.SettingEnd()
		}
	}
}

// SetTitle sets title of this window and its underlying GoosiWin.
func (w *RenderWin) SetTitle(title string) {
	w.Title = title
	if w.GoosiWin != nil {
		w.GoosiWin.SetTitle(title)
	}
	WinNewCloseStamp()
}

// SetStageTitle sets the title of the underlying GoosiWin to the given stage title
// combined with the RenderWin title.
func (w *RenderWin) SetStageTitle(title string) {
	if title != w.Title {
		title = title + " • " + w.Title
	}
	w.GoosiWin.SetTitle(title)
}

// LogicalDPI returns the current logical dots-per-inch resolution of the
// window, which should be used for most conversion of standard units --
// physical DPI can be found in the Screen
func (w *RenderWin) LogicalDPI() float32 {
	if w.GoosiWin == nil {
		sc := goosi.TheApp.Screen(0)
		if sc == nil {
			return 160 // null default
		}
		return sc.LogicalDPI
	}
	return w.GoosiWin.LogicalDPI()
}

// StepZoom calls [SetZoom] with the current zoom plus 10 times the given number of steps.
func (w *RenderWin) StepZoom(steps float32) {
	w.SetZoom(AppearanceSettings.Zoom + 10*steps)
}

// SetZoom sets [AppearanceSettingsData.Zoom] to the given value and then triggers
// necessary updating and makes a snackbar.
func (w *RenderWin) SetZoom(zoom float32) {
	AppearanceSettings.Zoom = mat32.Clamp(zoom, 10, 500)
	AppearanceSettings.Apply()
	UpdateAll()
	grr.Log(SaveSettings(AppearanceSettings))

	if ms := w.MainScene(); ms != nil {
		b := NewBody().AddSnackbarText(fmt.Sprintf("%.f%%", AppearanceSettings.Zoom))
		NewStretch(b)
		b.AddSnackbarIcon(icons.Remove, func(e events.Event) {
			w.StepZoom(-1)
		})
		b.AddSnackbarIcon(icons.Add, func(e events.Event) {
			w.StepZoom(1)
		})
		b.AddSnackbarButton("Reset", func(e events.Event) {
			w.SetZoom(100)
		})
		b.DeleteChildByName("stretch", true)
		b.NewSnackbar(ms).Run()
	}
}

// SetWinSize requests that the window be resized to the given size
// in OS window manager specific coordinates, which may be different
// from the underlying pixel-level resolution of the window.
// This will trigger a resize event and be processed
// that way when it occurs.
func (w *RenderWin) SetWinSize(sz image.Point) {
	w.GoosiWin.SetWinSize(sz)
}

// SetSize requests that the window be resized to the given size
// in underlying pixel coordinates, which means that the requested
// size is divided by the screen's DevicePixelRatio
func (w *RenderWin) SetSize(sz image.Point) {
	w.GoosiWin.SetSize(sz)
}

// StackAll returns a formatted stack trace of all goroutines.
// It calls runtime.Stack with a large enough buffer to capture the entire trace.
func StackAll() []byte {
	buf := make([]byte, 1024*10)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}

// Resized updates internal buffers after a window has been resized.
func (w *RenderWin) Resized() {
	rc := w.RenderCtx()
	if !w.IsVisible() {
		rc.SetFlag(false, RenderVisible)
		return
	}

	drw := w.GoosiWin.Drawer()

	rg := w.GoosiWin.RenderGeom()

	curRg := rc.Geom
	if curRg == rg {
		if DebugSettings.WinEventTrace {
			fmt.Printf("Win: %v skipped same-size Resized: %v\n", w.Name, curRg)
		}
		// still need to apply style even if size is same
		for _, kv := range w.MainStageMgr.Stack.Order {
			sc := kv.Value.Scene
			sc.ApplyStyleScene()
		}
		return
	}
	if drw.MaxTextures() != goosi.MaxTexturesPerSet*3 { // this is essential after hibernate
		drw.SetMaxTextures(goosi.MaxTexturesPerSet * 3) // use 3 sets
	}
	// w.FocusInactivate()
	// w.InactivateAllSprites()
	if !w.IsVisible() {
		rc.SetFlag(false, RenderVisible)
		if DebugSettings.WinEventTrace {
			fmt.Printf("Win: %v Resized already closed\n", w.Name)
		}
		return
	}
	if DebugSettings.WinEventTrace {
		fmt.Printf("Win: %v Resized from: %v to: %v\n", w.Name, curRg, rg)
	}
	rc.Geom = rg
	rc.SetFlag(true, RenderVisible)
	rc.LogicalDPI = w.LogicalDPI()
	// fmt.Printf("resize dpi: %v\n", w.LogicalDPI())
	w.MainStageMgr.Resize(rg)
	if DebugSettings.WinGeomTrace {
		log.Printf("WinGeomPrefs: recording from Resize\n")
	}
	WinGeomMgr.RecordPref(w)
}

// Raise requests that the window be at the top of the stack of windows,
// and receive focus.  If it is iconified, it will be de-iconified.  This
// is the only supported mechanism for de-iconifying. This also sets
// CurRenderWin to the window.
func (w *RenderWin) Raise() {
	w.GoosiWin.Raise()
	CurRenderWin = w
}

// Minimize requests that the window be iconified, making it no longer
// visible or active -- rendering should not occur for minimized windows.
func (w *RenderWin) Minimize() {
	w.GoosiWin.Minimize()
}

// CloseReq requests that the window be closed -- could be rejected
func (w *RenderWin) CloseReq() {
	w.GoosiWin.CloseReq()
}

// Closed frees any resources after the window has been closed.
func (w *RenderWin) Closed() {
	w.RenderCtx().Lock()
	defer w.RenderCtx().Unlock()

	AllRenderWins.Delete(w)
	MainRenderWins.Delete(w)
	DialogRenderWins.Delete(w)
	WinNewCloseStamp()
	if DebugSettings.WinEventTrace {
		fmt.Printf("Win: %v Closed\n", w.Name)
	}
	if len(AllRenderWins) > 0 {
		pfw := AllRenderWins[len(AllRenderWins)-1]
		if DebugSettings.WinEventTrace {
			fmt.Printf("Win: %v getting restored focus after: %v closed\n", pfw.Name, w.Name)
		}
		pfw.Raise()
	}
	// these are managed by the window itself
	// w.Sprites.Reset()

	w.RenderScenes.Reset()
	// todo: delete the contents of the window here??
}

// IsClosed reports if the window has been closed
func (w *RenderWin) IsClosed() bool {
	return w.GoosiWin.IsClosed() || w.MainStageMgr.Stack.Len() == 0
}

// SetCloseReqFunc sets the function that is called whenever there is a
// request to close the window (via a OS or a call to CloseReq() method).  That
// function can then adjudicate whether and when to actually call Close.
func (w *RenderWin) SetCloseReqFunc(fun func(win *RenderWin)) {
	w.GoosiWin.SetCloseReqFunc(func(owin goosi.Window) {
		fun(w)
	})
}

// SetCloseCleanFunc sets the function that is called whenever window is
// actually about to close (irrevocably) -- can do any necessary
// last-minute cleanup here.
func (w *RenderWin) SetCloseCleanFunc(fun func(win *RenderWin)) {
	w.GoosiWin.SetCloseCleanFunc(func(owin goosi.Window) {
		fun(w)
	})
}

// IsVisible is the main visibility check -- don't do any window updates if not visible!
func (w *RenderWin) IsVisible() bool {
	if w == nil || w.GoosiWin == nil || w.IsClosed() || w.Is(WinClosing) || !w.GoosiWin.IsVisible() {
		return false
	}
	return true
}

// WinNewCloseStamp updates the global WinNewCloseTime timestamp for updating windows menus
func WinNewCloseStamp() {
	RenderWinGlobalMu.Lock()
	WinNewCloseTime = time.Now()
	RenderWinGlobalMu.Unlock()
}

/////////////////////////////////////////////////////////////////////////////
//                   Event Loop

// StartEventLoop is the main startup method to call after the initial window
// configuration is setup -- does any necessary final initialization and then
// starts the event loop in this same goroutine, and does not return until the
// window is closed -- see GoStartEventLoop for a version that starts in a
// separate goroutine and returns immediately.
func (w *RenderWin) StartEventLoop() {
	w.EventLoop()
}

// GoStartEventLoop starts the event processing loop for this window in a new
// goroutine, and returns immediately.  Adds to WinWait waitgroup so a main
// thread can wait on that for all windows to close.
func (w *RenderWin) GoStartEventLoop() {
	WinWait.Add(1)
	w.SetFlag(true, WinGoLoop)
	go w.EventLoop()
}

// StopEventLoop tells the event loop to stop running when the next event arrives.
func (w *RenderWin) StopEventLoop() {
	w.SetFlag(true, WinStopEventLoop)
}

// SendCustomEvent sends a custom event with given data to this window -- widgets can connect
// to receive CustomEventTypes events to receive them.  Sometimes it is useful
// to send a custom event just to trigger a pass through the event loop, even
// if nobody is listening (e.g., if a popup is posted without a surrounding
// event, as in Complete.ShowCompletions
func (w *RenderWin) SendCustomEvent(data any) {
	w.GoosiWin.EventMgr().Custom(data)
}

// todo: fix or remove
// SendWinFocusEvent sends the RenderWinFocusEvent to widgets
func (w *RenderWin) SendWinFocusEvent(act events.WinActions) {
	// se := window.NewEvent(act)
	// se.Init()
	// w.MainStageMgr.HandleEvent(se)
}

/////////////////////////////////////////////////////////////////////////////
//                   Main Method: EventLoop

// EventLoop runs the event processing loop for the RenderWin -- grabs goosi
// events for the window and dispatches them to receiving nodes, and manages
// other state etc (popups, etc).
func (w *RenderWin) EventLoop() {
	defer func() { goosi.HandleRecover(recover()) }()

	d := &w.GoosiWin.EventMgr().Deque

	for {
		if w.HasFlag(WinStopEventLoop) {
			w.SetFlag(false, WinStopEventLoop)
			break
		}
		e := d.NextEvent()
		if w.HasFlag(WinStopEventLoop) {
			w.SetFlag(false, WinStopEventLoop)
			break
		}
		w.HandleEvent(e)
		if w.NoEventsChan != nil && len(d.Back) == 0 && len(d.Front) == 0 {
			w.NoEventsChan <- struct{}{}
		}
	}
	if DebugSettings.WinEventTrace {
		fmt.Printf("Win: %v out of event loop\n", w.Name)
	}
	if w.HasFlag(WinGoLoop) {
		WinWait.Done()
	}
	// our last act must be self destruction!
	w.MainStageMgr.DeleteAll()
}

// HandleEvent processes given events.Event.
// All event processing operates under a RenderCtx.Lock
// so that no rendering update can occur during event-driven updates.
// Because rendering itself is event driven, this extra level of safety
// is redundant in this case, but other non-event-driven updates require
// the lock protection.
func (w *RenderWin) HandleEvent(e events.Event) {
	rc := w.RenderCtx()
	rc.Lock()
	// we manually handle Unlock's in this function instead of deferring
	// it to avoid a cryptic "sync: can't unlock an already unlocked RWMutex"
	// error when panicking in the rendering goroutine. This is critical for
	// debugging on Android. TODO: maybe figure out a more sustainable approach to this.

	et := e.Type()
	if DebugSettings.EventTrace && et != events.WindowPaint && et != events.MouseMove {
		fmt.Println("Window got event", e)
	}
	if et >= events.Window && et <= events.WindowPaint {
		w.HandleWindowEvents(e)
		rc.Unlock()
		return
	}
	// fmt.Printf("got event type: %v: %v\n", et.BitIndexString(), evi)
	w.MainStageMgr.MainHandleEvent(e)
	rc.Unlock()
}

func (w *RenderWin) HandleWindowEvents(e events.Event) {
	et := e.Type()
	switch et {
	case events.WindowPaint:
		e.SetHandled()
		rc := w.RenderCtx()
		rc.Unlock() // one case where we need to break lock
		w.RenderWindow()
		rc.Lock()
		w.SendShowEvents()

	case events.WindowResize:
		e.SetHandled()
		w.Resized()

	case events.Window:
		ev := e.(*events.WindowEvent)
		switch ev.Action {
		case events.WinClose:
			// fmt.Printf("got close event for window %v \n", w.Name)
			e.SetHandled()
			w.StopEventLoop()
			rc := w.RenderCtx()
			rc.Unlock() // one case where we need to break lock
			w.Closed()
			rc.Lock()
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
				fmt.Printf("Win: %v got show event\n", w.Name)
			}
		case events.WinMove:
			e.SetHandled()
			// fmt.Printf("win move: %v\n", w.GoosiWin.Position())
			if DebugSettings.WinGeomTrace {
				log.Printf("WinGeomPrefs: recording from Move\n")
			}
			WinGeomMgr.RecordPref(w)
		case events.WinFocus:
			// if we are not already the last in AllRenderWins, we go there,
			// as this allows focus to be restored to us in the future
			if len(AllRenderWins) > 0 && AllRenderWins[len(AllRenderWins)-1] != w {
				AllRenderWins.Delete(w)
				AllRenderWins.Add(w)
			}
			if !w.HasFlag(WinGotFocus) {
				w.SetFlag(true, WinGotFocus)
				w.SendWinFocusEvent(events.WinFocus)
				if DebugSettings.WinEventTrace {
					fmt.Printf("Win: %v got focus\n", w.Name)
				}
			} else {
				if DebugSettings.WinEventTrace {
					fmt.Printf("Win: %v got extra focus\n", w.Name)
				}
			}
			CurRenderWin = w
		case events.WinFocusLost:
			if DebugSettings.WinEventTrace {
				fmt.Printf("Win: %v lost focus\n", w.Name)
			}
			w.SetFlag(false, WinGotFocus)
			w.SendWinFocusEvent(events.WinFocusLost)
		case events.ScreenUpdate:
			w.Resized()
			// TODO: figure out how to restore this stuff without breaking window size on mobile

			// WinGeomMgr.AbortSave() // anything just prior to this is sus
			// if !goosi.TheApp.NoScreens() {
			// 	Prefs.UpdateAll()
			// 	WinGeomMgr.RestoreAll()
			// }
		}
	}
}

/////////////////////////////////////////////////////////////////////////////
//                   Rendering

const (
	// Sprites are stored as arrays of same-sized textures,
	// allocated by size in Set 2, starting at 32
	SpriteStart = goosi.MaxTexturesPerSet * 2

	// Full set of sprite textures in set = 2
	MaxSpriteTextures = goosi.MaxTexturesPerSet

	// Allocate 128 layers within each sprite size
	MaxSpritesPerTexture = 128
)

// RenderParams are the key RenderWin params that determine if
// a scene needs to be restyled since last render, if these params change.
type RenderParams struct {
	// LogicalDPI is the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units.
	LogicalDPI float32

	// Geometry of the rendering window, in actual "dot" pixels used for rendering.
	Geom mat32.Geom2DInt
}

// NeedsRestyle returns true if the current render context
// params differ from those used in last render.
func (rp *RenderParams) NeedsRestyle(rc *RenderContext) bool {
	return rp.LogicalDPI != rc.LogicalDPI || rp.Geom != rc.Geom
}

// SaveRender grabs current render context params
func (rp *RenderParams) SaveRender(rc *RenderContext) {
	rp.LogicalDPI = rc.LogicalDPI
	rp.Geom = rc.Geom
}

// RenderContextFlags represent RenderContext state
type RenderContextFlags int64 //enums:bitflag -trim-prefix Render

const (
	// the window is visible and should be rendered to
	RenderVisible RenderContextFlags = iota

	// forces a rebuild of all scene elements
	RenderRebuild
)

// RenderContext provides rendering context from outer RenderWin
// window to Stage and Scene elements to inform styling, layout
// and rendering.  It also has the master Mutex for any updates
// to the window contents: use Read lock for anything updating.
type RenderContext struct {
	// Flags hold binary context state
	Flags RenderContextFlags

	// LogicalDPI is the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units.
	LogicalDPI float32

	// Geometry of the rendering window, in actual "dot" pixels used for rendering.
	Geom mat32.Geom2DInt

	// Mu is mutex for locking out rendering and any destructive updates.
	// It is locked at the RenderWin level during rendering and
	// event processing to provide exclusive blocking of external updates.
	// Use UpdateStartAsync from any outside routine to grab the lock before
	// doing modifications.
	Mu sync.Mutex
}

// NewRenderContext returns a new RenderContext initialized according to
// the main Screen size and LogicalDPI as initial defaults.
// The actual window size is set during Resized method, which is typically
// called after the window is created by the OS.
func NewRenderContext() *RenderContext {
	rc := &RenderContext{}
	scr := goosi.TheApp.Screen(0)
	if scr != nil {
		rc.Geom.SetRect(scr.Geometry)
		rc.LogicalDPI = scr.LogicalDPI
	} else {
		rc.Geom = mat32.Geom2DInt{Size: image.Pt(1080, 720)}
		rc.LogicalDPI = 160
	}
	rc.SetFlag(true, RenderVisible)
	return rc
}

// Lock is called by RenderWin during RenderWindow and HandleEvent
// when updating all widgets and rendering the screen.
// Any outside access to window contents / scene must acquire this
// lock first.  In general, use UpdateStartAsync to do this.
func (rc *RenderContext) Lock() {
	rc.Mu.Lock()
}

// Unlock must be called for each Lock, when done.
func (rc *RenderContext) Unlock() {
	rc.Mu.Unlock()
}

// HasFlag returns true if given flag is set
func (rc *RenderContext) HasFlag(flag enums.BitFlag) bool {
	return rc.Flags.HasFlag(flag)
}

// SetFlag sets given flag(s) on or off
func (rc *RenderContext) SetFlag(on bool, flag ...enums.BitFlag) {
	rc.Flags.SetFlag(on, flag...)
}

func (rc *RenderContext) String() string {
	str := fmt.Sprintf("Geom: %s  Visible: %v", rc.Geom, rc.HasFlag(RenderVisible))
	return str
}

//////////////////////////////////////////////////////////////////////
//  RenderScenes

// RenderScenes are a list of Scene objects, compiled in rendering order,
// whose Pixels images are composed directly to the RenderWin window.
type RenderScenes struct {

	// starting index for this set of Scenes
	StartIdx int

	// max index (exclusive) for this set of Scenes
	MaxIdx int

	// set to true to flip Y axis in drawing these images
	FlipY bool

	// ordered list of scenes -- index is Drawer image index.
	Scenes []*Scene

	// SceneIdx holds the index for each scene -- used to detect changes in index
	SceneIdx map[*Scene]int
}

// SetIdxRange sets the index range based on starting index and n
func (rs *RenderScenes) SetIdxRange(st, n int) {
	rs.StartIdx = st
	rs.MaxIdx = st + n
}

// Reset resets the list
func (rs *RenderScenes) Reset() {
	rs.Scenes = nil
	if rs.SceneIdx == nil {
		rs.SceneIdx = make(map[*Scene]int)
	}
}

// Add adds a new node, returning index
func (rs *RenderScenes) Add(sc *Scene, scIdx map[*Scene]int) int {
	if sc.Pixels == nil {
		return -1
	}
	idx := len(rs.Scenes)
	if idx >= rs.MaxIdx {
		slog.Error("gi.RenderScenes: too many Scenes to render all of them!", "max", rs.MaxIdx)
		return -1
	}
	if prvIdx, has := rs.SceneIdx[sc]; has {
		if prvIdx != idx {
			sc.SetFlag(true, ScImageUpdated) // need to copy b/c cur has diff image
		}
	} else {
		sc.SetFlag(true, ScImageUpdated) // need to copy b/c new
	}
	scIdx[sc] = idx
	rs.Scenes = append(rs.Scenes, sc)
	return idx
}

// SetImages calls drw.SetGoImage on all updated Scene images
func (rs *RenderScenes) SetImages(drw goosi.Drawer) {
	if len(rs.Scenes) == 0 {
		if DebugSettings.WinRenderTrace {
			fmt.Println("RenderScene.SetImages: no scenes")
		}
	}
	for i, sc := range rs.Scenes {
		if sc.Is(ScUpdating) || !sc.Is(ScImageUpdated) {
			if DebugSettings.WinRenderTrace {
				if sc.Is(ScUpdating) {
					fmt.Println("RenderScenes.SetImages: sc IsUpdating", sc.Name())
				}
				if !sc.Is(ScImageUpdated) {
					fmt.Println("RenderScenes.SetImages: sc Image NotUpdated", sc.Name())
				}
			}
			continue
		}
		if DebugSettings.WinRenderTrace {
			fmt.Println("RenderScenes.SetImages:", sc.Name(), sc.Pixels.Bounds())
		}
		drw.SetGoImage(i, 0, sc.Pixels, goosi.NoFlipY)
		sc.SetFlag(false, ScImageUpdated)
	}
}

// DrawAll does drw.Copy drawing call for all Scenes,
// using proper TextureSet for each of goosi.MaxTexturesPerSet Scenes.
func (rs *RenderScenes) DrawAll(drw goosi.Drawer) {
	nPerSet := goosi.MaxTexturesPerSet

	for i, sc := range rs.Scenes {
		set := i / nPerSet
		if i%nPerSet == 0 && set > 0 {
			drw.UseTextureSet(set)
		}
		bb := sc.Pixels.Bounds()
		op := draw.Over
		if i == 0 {
			op = draw.Src
		}
		drw.Copy(i, 0, sc.SceneGeom.Pos, bb, op, rs.FlipY)
	}
}

//////////////////////////////////////////////////////////////////////
//  RenderWin methods

func (w *RenderWin) RenderCtx() *RenderContext {
	return w.MainStageMgr.RenderCtx
}

// RenderWindow performs all rendering based on current StageMgr config.
// It sets the Write lock on RenderCtx Mutex, so nothing else can update
// during this time.  All other updates are done with a Read lock so they
// won't interfere with each other.
func (w *RenderWin) RenderWindow() {
	rc := w.RenderCtx()
	rc.Lock()
	defer func() {
		rc.SetFlag(false, RenderRebuild)
		rc.Unlock()
	}()
	rebuild := rc.HasFlag(RenderRebuild)

	stageMods, sceneMods := w.MainStageMgr.UpdateAll() // handles all Scene / Widget updates!
	top := w.MainStageMgr.Top()
	if top == nil {
		return
	}
	if !top.Sprites.Modified && !rebuild && !stageMods && !sceneMods { // nothing to do!
		// fmt.Println("no mods") // note: get a ton of these..
		return
	}

	if DebugSettings.WinRenderTrace {
		fmt.Println("RenderWindow: doing render:", w.Name)
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
func (w *RenderWin) DrawScenes() {
	if !w.IsVisible() || w.GoosiWin.Is(goosi.Minimized) {
		if DebugSettings.WinRenderTrace {
			fmt.Printf("RenderWindow: skipping update on inactive / minimized window: %v\n", w.Name)
		}
		return
	}
	// if !w.HasFlag(WinSentShow) {
	// 	return
	// }
	if !w.GoosiWin.Lock() {
		if DebugSettings.WinRenderTrace {
			fmt.Printf("RenderWindow: window was closed: %v\n", w.Name)
		}
		return
	}
	defer w.GoosiWin.Unlock()

	// pr := prof.Start("win.DrawScenes")

	drw := w.GoosiWin.Drawer()
	rs := &w.RenderScenes

	rs.SetImages(drw) // ensure all updated images copied

	top := w.MainStageMgr.Top()
	if top.Sprites.Modified {
		top.Sprites.ConfigSprites(drw)
	}

	drw.SyncImages()
	w.FillInsets()
	drw.StartDraw(0)
	drw.UseTextureSet(0)
	rs.DrawAll(drw)

	drw.UseTextureSet(2)
	top.Sprites.DrawSprites(drw)

	drw.EndDraw()

	// pr.End()
}

// FillInsets fills the window insets, if any, with [colors.Scheme.Background].
func (w *RenderWin) FillInsets() {
	// render geom and window geom
	rg := w.GoosiWin.RenderGeom()
	wg := mat32.Geom2DInt{Size: w.GoosiWin.Size()}

	// if our window geom is the same as our render geom, we have no
	// window insets to fill
	if wg == rg {
		return
	}

	drw := w.GoosiWin.Drawer()
	drw.StartFill()

	fill := func(x0, y0, x1, y1 int) {
		r := image.Rect(x0, y0, x1, y1)
		if r.Dx() == 0 || r.Dy() == 0 {
			return
		}
		drw.Fill(colors.Scheme.Background, *mat32.Identity3(), r, draw.Src)
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
func (w *RenderWin) GatherScenes() bool {
	rs := &w.RenderScenes
	rs.Reset()
	scIdx := make(map[*Scene]int)

	sm := &w.MainStageMgr
	n := sm.Stack.Len()
	if n == 0 {
		slog.Error("GatherScenes stack empty")
		return false // shouldn't happen!
	}

	// first, find the top-level window:
	winIdx := 0
	for i := n - 1; i >= 0; i-- {
		st := sm.Stack.ValueByIndex(i)
		if st.Type == WindowStage {
			if DebugSettings.WinRenderTrace {
				fmt.Println("GatherScenes: main Window:", st.String())
			}
			rs.Add(st.Scene, scIdx)
			winIdx = i
			break
		}
	}

	// then add everyone above that
	for i := winIdx + 1; i < n; i++ {
		st := sm.Stack.ValueByIndex(i)
		rs.Add(st.Scene, scIdx)
		if DebugSettings.WinRenderTrace {
			fmt.Println("GatherScenes: overlay Stage:", st.String())
		}
	}

	top := sm.Top()
	top.Sprites.Modified = true // ensure configured

	// then add the popups for the top main stage
	for _, kv := range top.PopupMgr.Stack.Order {
		st := kv.Value
		rs.Add(st.Scene, scIdx)
		if DebugSettings.WinRenderTrace {
			fmt.Println("GatherScenes: popup:", st.String())
		}
	}
	rs.SceneIdx = scIdx
	return true
}

func (w *RenderWin) SendShowEvents() {
	w.MainStageMgr.SendShowEvents()
}
