// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"log/slog"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"goki.dev/enums"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"golang.org/x/image/draw"
)

// WinWait is a wait group for waiting for all the open window event
// loops to finish -- this can be used for cases where the initial main run
// uses a GoStartEventLoop for example.  It is incremented by GoStartEventLoop
// and decremented when the event loop terminates.
var WinWait sync.WaitGroup

// Wait waits for all windows to close -- put this at the end of
// a main function that opens multiple windows.
func Wait() {
	WinWait.Wait()
}

// CurRenderWin is the current RenderWin window
// On mobile, this is the _only_ window.
var CurRenderWin *RenderWin

var (
	// LocalMainMenu controls whether the main menu is displayed locally at top of
	// each window, in addition to the global menu at the top of the screen.  Mac
	// native apps do not do this, but OTOH it makes things more consistent with
	// other platforms, and with larger screens, it can be convenient to have
	// access to all the menu items right there.  Controlled by Prefs.Params
	// variable.
	LocalMainMenu = false

	// WinNewCloseTime records last time a new window was opened or another
	// closed -- used to trigger updating of RenderWin menus on each window.
	WinNewCloseTime time.Time

	// RenderWinGlobalMu is a mutex for any global state associated with windows
	RenderWinGlobalMu sync.Mutex

	// RenderWinOpenTimer is used for profiling the open time of windows
	// if doing profiling, it will report the time elapsed in msec
	// to point of establishing initial focus in the window.
	RenderWinOpenTimer time.Time
)

// RenderWin provides an outer "actual" window where everything is rendered,
// and is the point of entry for all events coming in from user actions.
//
// RenderWin contents are all managed by the StageMgr (MainStageMgr) that
// handles MainStage elements such as Window, Dialog, and Sheet, which in
// turn manage their own stack of PopupStage elements such as Menu, Tooltip, etc.
// The contents of each Stage is provided by a Scene, containing Widgets,
// and the Stage Pixels image is drawn to the RenderWin in the RenderWindow method.
//
// Rendering is handled by the vdraw.Drawer from the vgpu package, which is provided
// by the goosi framework.  It is akin to a window manager overlaying Go image bitmaps
// on top of each other in the proper order, based on the StageMgr stacking order.
//   - Sprites are managed as layered textures of the same size, to enable
//     unlimited number packed into a few descriptors for standard sizes.
type RenderWin struct {
	Flags WinFlags

	Name string

	// displayed name of window, for window manager etc -- window object name is the internal handle and is used for tracking property info etc
	Title string

	// OS-specific window interface -- handles all the os-specific functions, including delivering events etc
	GoosiWin goosi.Window `json:"-" xml:"-"`

	// MainStageMgr controlling the MainStage elements in this window.
	// The Render Context in this manager is the original source for all Stages
	StageMgr MainStageMgr

	// RenderScenes are the Scene elements that draw directly to the window,
	// arranged in order.  See winrender.go for all rendering code.
	RenderScenes RenderScenes

	// main menu -- is first element of Scene always -- leave empty to not render.  On MacOS, this drives screen main menu
	MainMenu *MenuBar `json:"-" xml:"-"`

	// below are internal vars used during the event loop

	lastWinMenuUpdate time.Time

	// todo: these are bad:

	// the currently selected widget through the inspect editor selection mode
	SelectedWidget *WidgetBase

	// the channel on which the selected widget through the inspect editor selection mode is transmitted to the inspect editor after the user is done selecting
	SelectedWidgetChan chan *WidgetBase

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

	// WinSelectionMode indicates that the window is in GoGi inspect editor edit mode
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
//        App wrappers for oswin (end-user doesn't need to import)

// SetAppName sets the application name -- defaults to GoGi if not otherwise set
// Name appears in the first app menu, and specifies the default application-specific
// preferences directory, etc
func SetAppName(name string) {
	goosi.TheApp.SetName(name)
}

// AppName returns the application name -- see SetAppName to set
func AppName() string {
	return goosi.TheApp.Name()
}

// SetAppAbout sets the 'about' info for the app -- appears as a menu option
// in the default app menu
func SetAppAbout(about string) {
	goosi.TheApp.SetAbout(about)
}

// SetQuitReqFunc sets the function that is called whenever there is a
// request to quit the app (via a OS or a call to QuitReq() method).  That
// function can then adjudicate whether and when to actually call Quit.
func SetQuitReqFunc(fun func()) {
	goosi.TheApp.SetQuitReqFunc(fun)
}

// SetQuitCleanFunc sets the function that is called whenever app is
// actually about to quit (irrevocably) -- can do any necessary
// last-minute cleanup here.
func SetQuitCleanFunc(fun func()) {
	goosi.TheApp.SetQuitCleanFunc(fun)
}

// Quit closes all windows and exits the program.
func Quit() {
	if !goosi.TheApp.IsQuitting() {
		goosi.TheApp.Quit()
	}
}

// QuitReq requests to Quit -- calls QuitReqFunc if present
func QuitReq() {
	goosi.TheApp.QuitReq()
}

// PollEvents tells the main event loop to check for any gui events right now.
// Call this periodically from longer-running functions to ensure
// GUI responsiveness.
func PollEvents() {
	goosi.TheApp.PollEvents()
}

// OpenURL opens the given URL in the user's default browser.  On Linux
// this requires that xdg-utils package has been installed -- uses
// xdg-open command.
func OpenURL(url string) {
	goosi.TheApp.OpenURL(url)
}

/////////////////////////////////////////////////////////////////////////////
//                   New RenderWins and Init

// NewRenderWin creates a new window with given internal name handle,
// display name, and options.
func NewRenderWin(name, title string, opts *goosi.NewWindowOptions) *RenderWin {
	win := &RenderWin{}
	win.Name = name
	win.Title = title
	var err error
	win.GoosiWin, err = goosi.TheApp.NewWindow(opts)
	if err != nil {
		fmt.Printf("GoGi NewRenderWin error: %v \n", err)
		return nil
	}
	win.GoosiWin.SetName(title)
	win.GoosiWin.SetParent(win)
	drw := win.GoosiWin.Drawer()
	drw.SetMaxTextures(goosi.MaxTexturesPerSet * 3)       // use 3 sets
	win.RenderScenes.MaxIdx = goosi.MaxTexturesPerSet * 2 // reserve last for sprites
	win.StageMgr.Init(&win.StageMgr, win)

	// win.GoosiWin.SetDestroyGPUResourcesFunc(func() {
	// 	for _, ph := range win.Phongs {
	// 		ph.Destroy()
	// 	}
	// 	for _, fr := range win.Frames {
	// 		fr.Destroy()
	// 	}
	// })
	return win
}

// MainScene returns the current MainStage Scene
func (w *RenderWin) MainScene() *Scene {
	top := w.StageMgr.Top()
	if top == nil {
		return nil
	}
	return top.AsBase().Scene
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
	if WinEventTrace {
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
	if WinEventTrace {
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
	if isdif {
		for i, fw := range FocusRenderWins { // rename focus windows so we get focus later..
			if fw == curnm {
				FocusRenderWins[i] = name
			}
		}
	}
	if isdif && w.GoosiWin != nil {
		wgp := WinGeomMgr.Pref(w.Title, w.GoosiWin.Screen())
		if wgp != nil {
			WinGeomMgr.SettingStart()
			if w.GoosiWin.Size() != wgp.Size() || w.GoosiWin.Position() != wgp.Pos() {
				if WinGeomTrace {
					log.Printf("WinGeomPrefs: SetName setting geom for window: %v pos: %v size: %v\n", w.Name, wgp.Pos(), wgp.Size())
				}
				w.GoosiWin.SetGeom(wgp.Pos(), wgp.Size())
				goosi.TheApp.SendEmptyEvent()
			}
			WinGeomMgr.SettingEnd()
		}
	}
}

// SetTitle sets title of this window and also the RenderWin
func (w *RenderWin) SetTitle(name string) {
	w.Title = name
	if w.GoosiWin != nil {
		w.GoosiWin.SetTitle(name)
	}
	WinNewCloseStamp()
}

// LogicalDPI returns the current logical dots-per-inch resolution of the
// window, which should be used for most conversion of standard units --
// physical DPI can be found in the Screen
func (w *RenderWin) LogicalDPI() float32 {
	if w.GoosiWin == nil {
		return 96.0 // null default
	}
	return w.GoosiWin.LogicalDPI()
}

// ZoomDPI -- positive steps increase logical DPI, negative steps decrease it,
// in increments of 6 dots to keep fonts rendering clearly.
func (w *RenderWin) ZoomDPI(steps int) {
	rctx := w.StageMgr.RenderCtx
	rctx.Mu.RLock()

	// w.InactivateAllSprites()
	sc := w.GoosiWin.Screen()
	if sc == nil {
		sc = goosi.TheApp.Screen(0)
	}
	pdpi := sc.PhysicalDPI
	// ldpi = pdpi * zoom * ldpi
	cldpinet := sc.LogicalDPI
	cldpi := cldpinet / goosi.ZoomFactor
	nldpinet := cldpinet + float32(6*steps)
	if nldpinet < 6 {
		nldpinet = 6
	}
	// oldzoom := goosi.ZoomFactor
	goosi.ZoomFactor = nldpinet / cldpi
	Prefs.ApplyDPI()
	fmt.Printf("Effective LogicalDPI now: %v  PhysicalDPI: %v  Eff LogicalDPIScale: %v  ZoomFactor: %v\n", nldpinet, pdpi, nldpinet/pdpi, goosi.ZoomFactor)

	// actually resize window in proportion:
	// zr := goosi.ZoomFactor / oldzoom
	// curSz := rctx.Size
	// nsz := mat32.NewVec2FmPoint(curSz).MulScalar(zr).ToPointCeil()
	// rctx.SetFlag(true, RenderRebuild) // trigger full rebuild
	rctx.Mu.RUnlock()
	// w.GoosiWin.SetSize(nsz)
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
func (w *RenderWin) Resized(sz image.Point) {
	rctx := w.StageMgr.RenderCtx
	if !w.IsVisible() {
		rctx.SetFlag(false, RenderVisible)
		return
	}
	rctx.Mu.RLock()
	defer rctx.Mu.RUnlock()

	curSz := rctx.Size
	if curSz == sz {
		if WinEventTrace {
			fmt.Printf("Win: %v skipped same-size Resized: %v\n", w.Name, curSz)
		}
		return
	}
	drw := w.GoosiWin.Drawer()
	if drw.MaxTextures() != goosi.MaxTexturesPerSet*3 { // this is essential after hibernate
		drw.SetMaxTextures(goosi.MaxTexturesPerSet * 3) // use 3 sets
	}
	// w.FocusInactivate()
	// w.InactivateAllSprites()
	if !w.IsVisible() {
		rctx.SetFlag(false, RenderVisible)
		if WinEventTrace {
			fmt.Printf("Win: %v Resized already closed\n", w.Name)
		}
		return
	}
	if WinEventTrace {
		fmt.Printf("Win: %v Resized from: %v to: %v\n", w.Name, curSz, sz)
	}
	if curSz == (image.Point{}) { // first open
		StringsInsertFirstUnique(&FocusRenderWins, w.Name, 10)
	}
	rctx.Size = sz
	rctx.SetFlag(true, RenderVisible)
	rctx.LogicalDPI = w.LogicalDPI()
	// fmt.Printf("resize dpi: %v\n", w.LogicalDPI())
	w.StageMgr.Resize(sz)
	if WinGeomTrace {
		log.Printf("WinGeomPrefs: recording from Resize\n")
	}
	WinGeomMgr.RecordPref(w)
}

// Raise requests that the window be at the top of the stack of windows,
// and receive focus.  If it is iconified, it will be de-iconified.  This
// is the only supported mechanism for de-iconifying.
func (w *RenderWin) Raise() {
	w.GoosiWin.Raise()
}

// Minimize requests that the window be iconified, making it no longer
// visible or active -- rendering should not occur for minimized windows.
func (w *RenderWin) Minimize() {
	w.GoosiWin.Minimize()
}

// Close closes the window -- this is not a request -- it means:
// definitely close it -- flags window as such -- check Is(WinClosing)
func (w *RenderWin) Close() {
	if w.Is(WinClosing) {
		return
	}
	// this causes hangs etc: not good
	// w.StageMgr.RenderCtx.Mu.Lock() // allow other stuff to finish
	w.SetFlag(true, WinClosing)
	// w.StageMgr.RenderCtx.Mu.Unlock()
	w.GoosiWin.Close()
}

// CloseReq requests that the window be closed -- could be rejected
func (w *RenderWin) CloseReq() {
	w.GoosiWin.CloseReq()
}

// Closed frees any resources after the window has been closed.
func (w *RenderWin) Closed() {
	w.RenderCtx().WriteLock()
	defer w.RenderCtx().WriteUnlock()

	AllRenderWins.Delete(w)
	MainRenderWins.Delete(w)
	DialogRenderWins.Delete(w)
	RenderWinGlobalMu.Lock()
	StringsDelete(&FocusRenderWins, w.Name)
	RenderWinGlobalMu.Unlock()
	WinNewCloseStamp()
	if WinEventTrace {
		fmt.Printf("Win: %v Closed\n", w.Name)
	}
	if w.IsClosed() {
		if WinEventTrace {
			fmt.Printf("Win: %v Already Closed\n", w.Name)
		}
		return
	}
	// w.FocusInactivate()
	RenderWinGlobalMu.Lock()
	if len(FocusRenderWins) > 0 {
		pf := FocusRenderWins[0]
		RenderWinGlobalMu.Unlock()
		pfw, has := AllRenderWins.FindName(pf)
		if has {
			if WinEventTrace {
				fmt.Printf("Win: %v getting restored focus after: %v closed\n", pfw.Name, w.Name)
			}
			pfw.GoosiWin.Raise()
		} else {
			if WinEventTrace {
				fmt.Printf("Win: %v not found to restored focus: %v closed\n", pf, w.Name)
			}
		}
	} else {
		RenderWinGlobalMu.Unlock()
	}
	// these are managed by the window itself
	// w.Sprites.Reset()

	w.RenderScenes.Reset()
	// todo: delete the contents of the window here??
}

// IsClosed reports if the window has been closed
func (w *RenderWin) IsClosed() bool {
	return w.GoosiWin.IsClosed() || w.StageMgr.Stack.Len() == 0
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

// NeedWinMenuUpdate returns true if our lastWinMenuUpdate is != WinNewCloseTime
func (w *RenderWin) NeedWinMenuUpdate() bool {
	RenderWinGlobalMu.Lock()
	updt := false
	if w.lastWinMenuUpdate != WinNewCloseTime {
		w.lastWinMenuUpdate = WinNewCloseTime
		updt = true
	}
	RenderWinGlobalMu.Unlock()
	return updt
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

// SendShowEvent sends the WinShowEvent to anyone listening -- only sent once..
func (w *RenderWin) SendShowEvent() {
	if w.HasFlag(WinSentShow) {
		return
	}
	w.SetFlag(true, WinSentShow)
	// se := window.NewEvent(window.Show)
	// se.Init()
	// w.StageMgr.HandleEvent(se)
}

// SendWinFocusEvent sends the RenderWinFocusEvent to widgets
func (w *RenderWin) SendWinFocusEvent(act events.WinActions) {
	// se := window.NewEvent(act)
	// se.Init()
	// w.StageMgr.HandleEvent(se)
}

/////////////////////////////////////////////////////////////////////////////
//                   Main Method: EventLoop

// PollEvents first tells the main event loop to check for any gui events now
// and then it runs the event processing loop for the RenderWin as long
// as there are events to be processed, and then returns.
func (w *RenderWin) PollEvents() {
	goosi.TheApp.PollEvents()
	for {
		evi, has := w.GoosiWin.PollEvent()
		if !has {
			break
		}
		w.HandleEvent(evi)
	}
}

// EventLoop runs the event processing loop for the RenderWin -- grabs oswin
// events for the window and dispatches them to receiving nodes, and manages
// other state etc (popups, etc).
func (w *RenderWin) EventLoop() {
	// this recover allows for debugging on Android, and we need to do it separately
	// here because this is the main thing in a separate goroutine that goosi doesn't
	// control. TODO: maybe figure out a more sustainable approach to this.
	defer func() {
		if r := recover(); r != nil {
			log.Println("panic:", r)
			log.Println("")
			log.Println("----- START OF STACK TRACE: -----")
			log.Println(string(debug.Stack()))
			log.Println("----- END OF STACK TRACE -----")
			// if we are on mobile, panicking leads to crashing too quickly to get the log messages
			if goosi.TheApp.Platform().IsMobile() {
				log.Fatalln("Terminating app due to panic stated above")
			} else {
				panic(r)
			}
		}
	}()
	for {
		if w.HasFlag(WinStopEventLoop) {
			w.SetFlag(false, WinStopEventLoop)
			break
		}
		evi := w.GoosiWin.NextEvent()
		if w.HasFlag(WinStopEventLoop) {
			w.SetFlag(false, WinStopEventLoop)
			break
		}
		w.HandleEvent(evi)
	}
	if WinEventTrace {
		fmt.Printf("Win: %v out of event loop\n", w.Name)
	}
	if w.HasFlag(WinGoLoop) {
		WinWait.Done()
	}
	// our last act must be self destruction!
}

// HandleEvent processes given events.Event.
// All event processing operates under a RenderCtx.ReadLock
// so that no rendering update can occur during event-driven updates.
// Because rendering itself is event driven, this extra level of safety
// is redundant in this case, but other non-event-driven updates require
// the lock protection.
func (w *RenderWin) HandleEvent(evi events.Event) {
	w.RenderCtx().ReadLock()
	// we manually handle ReadUnlock's in this function instead of deferring
	// it to avoid a cryptic "sync: can't unlock an already unlocked RWMutex"
	// error when panicking in the rendering goroutine. This is critical for
	// debugging on Android. TODO: maybe figure out a more sustainable approach to this.

	et := evi.Type()
	if EventTrace && et != events.WindowPaint && et != events.MouseMove {
		log.Printf("Got event: %s\n", evi)
	}
	if et >= events.Window && et <= events.WindowPaint {
		w.HandleWindowEvents(evi)
		w.RenderCtx().ReadUnlock()
		return
	}
	// fmt.Printf("got event type: %v: %v\n", et.BitIndexString(), evi)
	w.StageMgr.HandleEvent(evi)
	w.RenderCtx().ReadUnlock()
}

func (w *RenderWin) HandleWindowEvents(evi events.Event) {
	et := evi.Type()
	switch et {
	case events.WindowPaint:
		evi.SetHandled()
		w.RenderCtx().ReadUnlock() // one case where we need to break lock
		w.RenderWindow()
		w.RenderCtx().ReadLock()

	case events.WindowResize:
		evi.SetHandled()
		w.Resized(w.GoosiWin.Size())

	case events.Window:
		ev := evi.(*events.WindowEvent)
		switch ev.Action {
		case events.WinClose:
			// fmt.Printf("got close event for window %v \n", w.Name)
			evi.SetHandled()
			w.SetFlag(true, WinStopEventLoop)
			w.RenderCtx().ReadUnlock() // one case where we need to break lock
			w.Closed()
			w.RenderCtx().ReadLock()
		case events.WinMinimize:
			evi.SetHandled()
			// on mobile platforms, we need to set the size to 0 so that it detects a size difference
			// and lets the size event go through when we come back later
			// if goosi.TheApp.Platform().IsMobile() {
			// 	w.Scene.Geom.Size = image.Point{}
			// }
		case events.WinShow:
			evi.SetHandled()
			// note that this is sent delayed by driver
			if WinEventTrace {
				fmt.Printf("Win: %v got show event\n", w.Name)
			}
			// if w.NeedWinMenuUpdate() {
			// 	w.MainMenuUpdateRenderWins()
			// }
			w.SendShowEvent() // happens AFTER full render
		case events.WinMove:
			evi.SetHandled()
			// fmt.Printf("win move: %v\n", w.GoosiWin.Position())
			if WinGeomTrace {
				log.Printf("WinGeomPrefs: recording from Move\n")
			}
			WinGeomMgr.RecordPref(w)
		case events.WinFocus:
			StringsInsertFirstUnique(&FocusRenderWins, w.Name, 10)
			if !w.HasFlag(WinGotFocus) {
				w.SetFlag(true, WinGotFocus)
				w.SendWinFocusEvent(events.WinFocus)
				if WinEventTrace {
					fmt.Printf("Win: %v got focus\n", w.Name)
				}
				// if w.NeedWinMenuUpdate() {
				// 	w.MainMenuUpdateRenderWins()
				// }
			} else {
				if WinEventTrace {
					fmt.Printf("Win: %v got extra focus\n", w.Name)
				}
			}
		case events.WinFocusLost:
			if WinEventTrace {
				fmt.Printf("Win: %v lost focus\n", w.Name)
			}
			w.SetFlag(false, WinGotFocus)
			w.SendWinFocusEvent(events.WinFocusLost)
		case events.ScreenUpdate:
			w.Resized(w.GoosiWin.Size())
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

	// Size of the rendering window, in actual "dot" pixels used for rendering.
	Size image.Point
}

// NeedsRestyle returns true if the current render context
// params differ from those used in last render.
func (rp *RenderParams) NeedsRestyle(rc *RenderContext) bool {
	if rp.LogicalDPI != rc.LogicalDPI || rp.Size != rc.Size {
		return true
	}
	return false
}

// SaveRender grabs current render context params
func (rp *RenderParams) SaveRender(rc *RenderContext) {
	rp.LogicalDPI = rc.LogicalDPI
	rp.Size = rc.Size
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

	// Size of the rendering window, in actual "dot" pixels used for rendering.
	Size image.Point

	// Mu is mutex for locking out rendering and any destructive updates.
	// it is Write locked during rendering to provide exclusive blocking
	// of any other updates, which are Read locked so that you don't
	// get caught in deadlocks among all the different Read locks.
	Mu sync.RWMutex
}

// WriteLock is only called by RenderWin during RenderWindow function
// when updating all widgets and rendering the screen.  All others should
// call ReadLock.  Always call corresponding Unlock in defer!
func (rc *RenderContext) WriteLock() {
	rc.Mu.Lock()
}

// WriteUnlock must be called for each WriteLock, when done.
func (rc *RenderContext) WriteUnlock() {
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

// ReadLock should be called whenever modifying anything in the entire
// RenderWin context.  Because it is a Read lock, it does _not_ block
// any other updates happening at the same time -- it only prevents
// the full Render from happening until these updates finish.
// Other direct resources must have their own separate Write locks to protect them.
// It is automatically called at the start of HandleEvents, so all
// standard Event-driven updates are automatically covered.
// Other update entry points, such as animations, need to call this.
// Always call corresponding Unlock in defer!
func (rc *RenderContext) ReadLock() {
	rc.Mu.RLock()
}

// ReadUnlock must be called for each ReadLock, when done.
func (rc *RenderContext) ReadUnlock() {
	rc.Mu.RUnlock()
}

func (rc *RenderContext) String() string {
	str := fmt.Sprintf("Size: %s  Visible: %v", rc.Size, rc.HasFlag(RenderVisible))
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
}

// SetIdxRange sets the index range based on starting index and n
func (rs *RenderScenes) SetIdxRange(st, n int) {
	rs.StartIdx = st
	rs.MaxIdx = st + n
}

// Reset resets the list
func (rs *RenderScenes) Reset() {
	rs.Scenes = nil
}

// Add adds a new node, returning index
func (rs *RenderScenes) Add(sc *Scene) int {
	if sc.Pixels == nil {
		return -1
	}
	idx := len(rs.Scenes)
	if idx >= rs.MaxIdx {
		fmt.Printf("gi.RenderScenes: ERROR too many Scenes to render all of them!  Max: %d\n", rs.MaxIdx)
		return -1
	}
	rs.Scenes = append(rs.Scenes, sc)
	return idx
}

// SetImages calls drw.SetGoImage on all updated Scene images
func (rs *RenderScenes) SetImages(drw goosi.Drawer) {
	if len(rs.Scenes) == 0 {
		if WinRenderTrace {
			fmt.Println("RenderScene.SetImages: no scenes")
		}
	}
	for i, sc := range rs.Scenes {
		if sc.Is(ScUpdating) || !sc.Is(ScImageUpdated) {
			if WinRenderTrace {
				if sc.Is(ScUpdating) {
					fmt.Println("RenderScenes.SetImages: sc IsUpdating", sc.Name())
				}
				if !sc.Is(ScImageUpdated) {
					fmt.Println("RenderScenes.SetImages: sc Image NotUpdated", sc.Name())
				}
			}
			continue
		}
		if WinRenderTrace {
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
		drw.Copy(i, 0, sc.Geom.Pos, bb, op, rs.FlipY)
	}
}

//////////////////////////////////////////////////////////////////////
//  RenderWin methods

func (w *RenderWin) RenderCtx() *RenderContext {
	return w.StageMgr.RenderCtx
}

// RenderWindow performs all rendering based on current StageMgr config.
// It sets the Write lock on RenderCtx Mutex, so nothing else can update
// during this time.  All other updates are done with a Read lock so they
// won't interfere with each other.
func (w *RenderWin) RenderWindow() {
	rc := w.RenderCtx()
	rc.WriteLock()
	rebuild := rc.HasFlag(RenderRebuild)

	defer func() {
		rc.WriteUnlock()
		rc.SetFlag(false, RenderRebuild)
	}()

	stageMods, sceneMods := w.StageMgr.UpdateAll() // handles all Scene / Widget updates!
	top := w.StageMgr.Top().AsMain()
	if !top.Sprites.Modified && !rebuild && !stageMods && !sceneMods { // nothing to do!
		// fmt.Println("no mods") // note: get a ton of these..
		return
	}

	if WinRenderTrace {
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
	// fmt.Println("done render")
}

// DrawScenes does the drawing of RenderScenes to the window.
func (w *RenderWin) DrawScenes() {
	if !w.IsVisible() || w.GoosiWin.IsMinimized() {
		if WinRenderTrace {
			fmt.Printf("RenderWindow: skipping update on inactive / minimized window: %v\n", w.Name)
		}
		return
	}
	// if !w.HasFlag(WinSentShow) {
	// 	return
	// }
	if !w.GoosiWin.Lock() {
		if WinRenderTrace {
			fmt.Printf("RenderWindow: window was closed: %v\n", w.Name)
		}
		return
	}
	defer w.GoosiWin.Unlock()

	// pr := prof.Start("win.DrawScenes")

	drw := w.GoosiWin.Drawer()
	rs := &w.RenderScenes

	rs.SetImages(drw) // ensure all updated images copied

	top := w.StageMgr.Top().AsMain()
	if top.Sprites.Modified {
		top.Sprites.ConfigSprites(drw)
	}

	drw.SyncImages()
	drw.StartDraw(0)
	drw.UseTextureSet(0)
	drw.Scale(0, 0, drw.DestBounds(), image.Rectangle{}, draw.Src, goosi.NoFlipY)
	rs.DrawAll(drw)

	drw.UseTextureSet(2)
	top.Sprites.DrawSprites(drw)

	drw.EndDraw()

	// pr.End()
}

// GatherScenes finds all the Scene elements that drive rendering
// into the RenderScenes list.  Returns false on failure / nothing to render.
func (w *RenderWin) GatherScenes() bool {
	rs := &w.RenderScenes
	rs.Reset()

	sm := &w.StageMgr
	n := sm.Stack.Len()
	if n == 0 {
		slog.Error("GatherScenes stack empty")
		return false // shouldn't happen!
	}

	// first, find the top-level window:
	winIdx := 0
	for i := n - 1; i >= 0; i-- {
		st := sm.Stack.ValByIdx(i).AsMain()
		if st.Type == WindowStage {
			if WinRenderTrace {
				fmt.Println("GatherScenes: main Window:", st.String())
			}
			rs.Add(st.Scene)
			winIdx = i
			break
		}
	}

	// then add everyone above that
	for i := winIdx + 1; i < n; i++ {
		st := sm.Stack.ValByIdx(i).AsMain()
		rs.Add(st.Scene)
		if WinRenderTrace {
			fmt.Println("GatherScenes: overlay Stage:", st.String())
		}
	}

	top := sm.Top().AsMain()
	top.Sprites.Modified = true // ensure configured

	// then add the popups for the top main stage
	for _, kv := range top.PopupMgr.Stack.Order {
		st := kv.Val.AsBase()
		rs.Add(st.Scene)
		if WinRenderTrace {
			fmt.Println("GatherScenes: popup:", st.String())
		}
	}
	return true
}

/*
/////////////////////////////////////////////////////////////////////////////
//                   MainMenu Updating

// MainMenuUpdated needs to be called whenever the main menu for this window
// is updated in terms of items added or removed.
func (w *RenderWin) MainMenuUpdated() {
	if w == nil || w.MainMenu == nil || !w.IsVisible() {
		return
	}
	w.StageMgr.RenderCtx.Mu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.StageMgr.RenderCtx.Mu.Unlock()
		return
	}
	w.MainMenu.UpdateMainMenu(w) // main update menu call, in bars.go for MenuBar
	w.StageMgr.RenderCtx.Mu.Unlock()
}

// MainMenuUpdateActives needs to be called whenever items on the main menu
// for this window have their IsActive status updated.
func (w *RenderWin) MainMenuUpdateActives() {
	if w == nil || w.MainMenu == nil || !w.IsVisible() {
		return
	}
	w.StageMgr.RenderCtx.Mu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.StageMgr.RenderCtx.Mu.Unlock()
		return
	}
	w.MainMenu.MainMenuUpdateActives(w) // also in bars.go for MenuBar
	w.StageMgr.RenderCtx.Mu.Unlock()
}

// MainMenuUpdateRenderWins updates a RenderWin menu with a list of active menus.
func (w *RenderWin) MainMenuUpdateRenderWins() {
	if w == nil || w.MainMenu == nil || !w.IsVisible() {
		return
	}
	w.StageMgr.RenderCtx.Mu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.StageMgr.RenderCtx.Mu.Unlock()
		return
	}
	RenderWinGlobalMu.Lock()
	wmeni := w.MainMenu.ChildByName("RenderWin", 3)
	if wmeni == nil {
		RenderWinGlobalMu.Unlock()
		w.StageMgr.RenderCtx.Mu.Unlock()
		return
	}
	wmen := wmeni.(*Action)
	men := make(Menu, 0, len(AllRenderWins))
	men.AddRenderWinsMenu(w)
	wmen.Menu = men
	RenderWinGlobalMu.Unlock()
	w.StageMgr.RenderCtx.Mu.Unlock()
	w.MainMenuUpdated()
}
*/

// RenderWinSelectionSpriteName is the sprite name used for the semi-transparent
// blue box rendered above elements selected in selection mode
var RenderWinSelectionSpriteName = "gi.RenderWin.SelectionBox"

// SelectionSprite deletes any existing selection box sprite
// and returns a new one for the given widget base. This should
// only be used in inspect editor Selection Mode.
func (w *RenderWin) SelectionSprite(wb *WidgetBase) *Sprite {
	/*
		w.DeleteSprite(RenderWinSelectionSpriteName)
		sp := NewSprite(RenderWinSelectionSpriteName, wb.WinBBox.Size(), image.Point{})
		draw.Draw(sp.Pixels, sp.Pixels.Bounds(), &image.Uniform{colors.SetAF32(colors.Scheme.Primary, 0.5)}, image.Point{}, draw.Src)
		sp.Geom.Pos = wb.WinBBox.Min
		w.AddSprite(sp)
		w.ActivateSprite(RenderWinSelectionSpriteName)
		return sp
	*/
	return nil
}
