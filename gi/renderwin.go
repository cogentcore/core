// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"goki.dev/enums"
	"goki.dev/goosi"
	"goki.dev/goosi/dnd"
	"goki.dev/goosi/key"
	"goki.dev/goosi/mouse"
	"goki.dev/goosi/window"
	"goki.dev/ki/v2"
	"goki.dev/prof/v2"
	"goki.dev/vgpu/v2/vgpu"
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

// RenderContext provides rendering context from outer RenderWin
// window to Stage and Scene elements to inform styling, layout
// and rendering.
type RenderContext struct {
	// LogicalDPI is the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units.
	LogicalDPI float32

	// Size of the rendering window, in actual "dot" pixels used for rendering.
	Size image.Point

	// Visible is true if window is visible and should be rendered to
	Visible bool

	// Mu is mutex for locking out rendering and any destructive updates.
	// it is Write locked during rendering to provide exclusive blocking
	// of any other updates, which are Read locked so that you don't
	// get caught in deadlocks among all the different Read locks.
	Mu sync.RWMutex
}

var (
	// EventSkipLagMSec is the number of milliseconds of lag between the time the
	// event was sent to the time it is being processed, above which a repeated
	// event type (scroll, drag, resize) is skipped
	EventSkipLagMSec = 50

	// FilterLaggyKeyEvents -- set to true to apply laggy filter to KeyEvents
	// (normally excluded)
	FilterLaggyKeyEvents = false

	// DragStartMSec is the number of milliseconds to wait before initiating a
	// regular mouse drag event (as opposed to a basic mouse.Press)
	DragStartMSec = 50

	// DragStartPix is the number of pixels that must be moved before
	// initiating a regular mouse drag event (as opposed to a basic mouse.Press)
	DragStartPix = 4

	// DNDStartMSec is the number of milliseconds to wait before initiating a
	// drag-n-drop event -- gotta drag it like you mean it
	DNDStartMSec = 200

	// DNDStartPix is the number of pixels that must be moved before
	// initiating a drag-n-drop event -- gotta drag it like you mean it
	DNDStartPix = 20

	// HoverStartMSec is the number of milliseconds to wait before initiating a
	// hover event (e.g., for opening a tooltip)
	HoverStartMSec = 1000

	// HoverMaxPix is the maximum number of pixels that mouse can move and still
	// register a Hover event
	HoverMaxPix = 5

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

// These constants define use of Texture images in vdraw.Drawer
// for updating the window efficiently.  They are allocated in
// sets of 16:
// Set 0: 0 = main scene, DirectUploads, Popups
// Set 1: 16 region update images
// Set 2: 16 sprite arrays with 128 allocated per size
const (
	// MaxDirectUploads are direct GPU texture transfer sources
	MaxDirectUploads = 7

	// MaxPopups is the maximum number of scene regions
	// updated directly from popups
	MaxPopups = 8

	// Where region updates start (start of Set 1) -- 16
	RegionUpdateStart = 1 + MaxDirectUploads + MaxPopups

	// Full set of region updates in set = 1
	MaxRegionUpdates = 16

	// Sprites are stored as arrays of same-sized textures,
	// allocated by size in Set 2, starting at 32
	SpriteStart = RegionUpdateStart + MaxRegionUpdates

	// Full set of sprite textures in set = 2
	MaxSpriteTextures = 16

	// Allocate 128 layers within each sprite size
	MaxSpritesPerTexture = 128
)

// RenderWin provides an OS-specific window and all the associated event
// handling.  Widgets connect to event signals to receive relevant GUI events.
//
// RenderWin manages a stack of Scenes, each of which manage a separate bitmap
// image, onto which Widgets render.  For main windows, the Scene Frame
// has a MainMenu for the window (which can be empty, in which case it is not
// displayed).  On MacOS, this main menu updates the overall menubar,
// and also can show the local menu (on by default).
//
// Widgets should always use methods to access / set state, and generally should
// not do much directly with the window.  Almost everything here needs to be
// guarded by various mutexes.  Leaving everything accessible so expert outside
// access is still possible in a pinch, but again don't use it unless you know
// what you're doing (and it might change over time too..)
//
// Rendering logic:
//   - vdraw.Drawer manages all rendering to the window surface, provided via
//     the RenderWin window, using vulkan stored images (16 max)
//   - Order is: Base Scene (image 0), then direct uploads, popups, and sprites.
//   - DirectUps (e.g., gi3d.Scene) directly upload their own texture to a Draw image
//     (note: cannot upload directly to window as this prevents popups and overlays)
//   - Popups (which have their own Scenes)
//   - Sprites are managed as layered textures of the same size, to enable
//     unlimited number packed into a few descriptors for standard sizes.
type RenderWin struct {
	Flags WinFlags

	Name string

	// displayed name of window, for window manager etc -- window object name is the internal handle and is used for tracking property info etc
	Title string `desc:"displayed name of window, for window manager etc -- window object name is the internal handle and is used for tracking property info etc"`

	// OS-specific window interface -- handles all the os-specific functions, including delivering events etc
	GoosiWin goosi.Window `json:"-" xml:"-" desc:"OS-specific window interface -- handles all the os-specific functions, including delivering events etc"`

	// MainStageMgr controlling the MainStage elements in this window.
	// The Render Context in this manager is the original source for all Stages
	StageMgr MainStageMgr

	// main menu -- is first element of Scene.Frame always -- leave empty to not render.  On MacOS, this drives screen main menu
	MainMenu *MenuBar `json:"-" xml:"-" desc:"main menu -- is first element of Scene.Frame always -- leave empty to not render.  On MacOS, this drives screen main menu"`

	// currently active shortcuts for this window (shortcuts are always window-wide -- use widget key event processing for more local key functions)
	Shortcuts Shortcuts `json:"-" xml:"-" desc:"currently active shortcuts for this window (shortcuts are always window-wide -- use widget key event processing for more local key functions)"`

	// below are internal vars used during the event loop

	lastWinMenuUpdate time.Time

	skippedResize *window.Event

	lastEt goosi.EventTypes

	// todo: these are bad:

	// the currently selected widget through the inspect editor selection mode
	SelectedWidget *WidgetBase `desc:"the currently selected widget through the inspect editor selection mode"`

	// the channel on which the selected widget through the inspect editor selection mode is transmitted to the inspect editor after the user is done selecting
	SelectedWidgetChan chan *WidgetBase `desc:"the channel on which the selected widget through the inspect editor selection mode is transmitted to the inspect editor after the user is done selecting"`

	// dir draws are direct upload regions -- direct uploaders upload their images directly to an image here
	DirDraws RenderWinDrawers `desc:"dir draws are direct upload regions -- direct uploaders upload their images directly to an image here"`
	PopDraws RenderWinDrawers // popup regions
	UpdtRegs RenderWinUpdates // misc vp update regions

	// // the phongs for the window
	// Phongs []*vphong.Phong ` json:"-" xml:"-" desc:"the phongs for the window"`
	//
	// // the render frames for the window
	// Frames []*vgpu.RenderFrame ` json:"-" xml:"-" desc:"the render frames for the window"`
}

// WinFlags extend NodeBase NodeFlags to hold RenderWin state
type WinFlags int64 //enums:bitflag

const (
	// WinFlagHasGeomPrefs indicates if this window has WinGeomPrefs setting that
	// sized it -- affects whether other default geom should be applied.
	WinFlagHasGeomPrefs WinFlags = iota

	// WinFlagUpdating is atomic flag around global updating -- routines can check IsWinUpdating and bail
	WinFlagUpdating

	// WinFlagIsClosing is atomic flag indicating window is closing
	WinFlagIsClosing

	// WinFlagIsResizing is atomic flag indicating window is resizing
	WinFlagIsResizing

	// WinFlagGotPaint have we received our first paint event yet?
	// ignore other window events before this point
	WinFlagGotPaint

	// WinFlagGotFocus indicates that have we received RenderWin focus
	WinFlagGotFocus

	// WinFlagSentShow have we sent the show event yet?  Only ever sent ONCE
	WinFlagSentShow

	// WinFlagGoLoop true if we are running from GoStartEventLoop -- requires a WinWait.Done at end
	WinFlagGoLoop

	// WinFlagStopEventLoop is set when event loop stop is requested
	WinFlagStopEventLoop

	// WinFlagDoFullRender is set at event loop startup to trigger a full render once the window
	// is properly shown
	WinFlagDoFullRender

	// WinFlagFocusActive indicates if widget focus is currently in an active state or not
	WinFlagFocusActive

	// WinSelectionMode indicates that the window is in GoGi inspect editor edit mode
	WinFlagSelectionMode
)

// HasFlag returns true if given flag is set
func (w *RenderWin) HasFlag(flag enums.BitFlag) bool {
	return w.Flags.HasFlag(flag)
}

// SetFlag sets given flag(s) on or off
func (w *RenderWin) SetFlag(on bool, flag ...enums.BitFlag) {
	w.Flags.SetFlag(on, flag...)
}

// HasGeomPrefs returns true if geometry prefs were set already
func (w *RenderWin) HasGeomPrefs() bool {
	return w.HasFlag(WinFlagHasGeomPrefs)
}

// IsClosing returns true if window has requested to close -- don't
// attempt to update it any further
func (w *RenderWin) IsClosing() bool {
	return w.HasFlag(WinFlagIsClosing)
}

// IsFocusActive returns true if window has focus active flag set
func (w *RenderWin) IsFocusActive() bool {
	return w.HasFlag(WinFlagFocusActive)
}

// SetFocusActiveState sets focus active flag to given state
func (w *RenderWin) SetFocusActiveState(active bool) {
	w.SetFlag(active, WinFlagFocusActive)
}

// IsInSelectionMode returns true if window has selection mode set
func (w *RenderWin) IsInSelectionMode() bool {
	return w.HasFlag(WinFlagSelectionMode)
}

// SetSelectionModeState sets selection mode to given state
func (w *RenderWin) SetSelectionModeState(selmode bool) {
	w.SetFlag(selmode, WinFlagSelectionMode)
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
	drw.SetMaxTextures(vgpu.MaxTexturesPerSet * 3) // use 3 sets

	win.DirDraws.SetIdxRange(1, MaxDirectUploads)
	// win.DirDraws.FlipY = true // drawing is flipped in general here.
	win.PopDraws.SetIdxRange(win.DirDraws.MaxIdx, MaxPopups)

	win.StageMgr.Init(win)

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

/*
// RecycleMainRenderWin looks for existing window with same Data --
// if found brings that to the front, returns true for bool.
// else (and if data is nil) calls NewDialogWin, and returns false.
func RecycleMainRenderWin(data any, name, title string, width, height int) (*RenderWin, bool) {
	if data == nil {
		return NewMainRenderWin(name, title, width, height), false
	}
	ew, has := MainRenderWins.FindData(data)
	if has {
		if WinEventTrace {
			fmt.Printf("Win: %v getting recycled based on data match\n", ew.Nm)
		}
		ew.RenderWin.Raise()
		return ew, true
	}
	nw := NewMainRenderWin(name, title, width, height)
	nw.Data = data
	return nw, false
}
*/

/*

// RecycleDialogWin looks for existing window with same Data --
// if found brings that to the front, returns true for bool.
// else (and if data is nil) calls [NewDialogWin], and returns false.
func RecycleDialogWin(data any, name, title string, width, height int, modal bool) (*RenderWin, bool) {
	if data == nil {
		return NewDialogWin(name, title, width, height, modal), false
	}
	ew, has := DialogRenderWins.FindData(data)
	if has {
		if WinEventTrace {
			fmt.Printf("Win: %v getting recycled based on data match\n", ew.Nm)
		}
		ew.RenderWin.Raise()
		return ew, true
	}
	nw := NewDialogWin(name, title, width, height, modal)
	nw.Data = data
	return nw, false
}
*/

/*
// SetName sets name of this window and also the RenderWin, and applies any window
// geometry settings associated with the new name if it is different from before
func (w *RenderWin) SetName(name string) {
	curnm := w.Name()
	isdif := curnm != name
	w.NodeBase.SetName(name)
	if w.RenderWin != nil {
		w.RenderWin.SetName(name)
	}
	if isdif {
		for i, fw := range FocusRenderWins { // rename focus windows so we get focus later..
			if fw == curnm {
				FocusRenderWins[i] = name
			}
		}
	}
	if isdif && w.RenderWin != nil {
		wgp := WinGeomMgr.Pref(name, w.RenderWin.Screen())
		if wgp != nil {
			WinGeomMgr.SettingStart()
			if w.RenderWin.Size() != wgp.Size() || w.RenderWin.Position() != wgp.Pos() {
				if WinGeomTrace {
					log.Printf("WinGeomPrefs: SetName setting geom for window: %v pos: %v size: %v\n", w.Name(), wgp.Pos(), wgp.Size())
				}
				w.RenderWin.SetGeom(wgp.Pos(), wgp.Size())
				goosi.TheApp.SendEmptyEvent()
			}
			WinGeomMgr.SettingEnd()
		}
	}
}

// SetTitle sets title of this window and also the RenderWin
func (w *RenderWin) SetTitle(name string) {
	w.Title = name
	if w.RenderWin != nil {
		w.RenderWin.SetTitle(name)
	}
	WinNewCloseStamp()
}
*/

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
	goosi.ZoomFactor = nldpinet / cldpi
	Prefs.ApplyDPI()
	fmt.Printf("Effective LogicalDPI now: %v  PhysicalDPI: %v  Eff LogicalDPIScale: %v  ZoomFactor: %v\n", nldpinet, pdpi, nldpinet/pdpi, goosi.ZoomFactor)
	// w.FullReRender()
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

// IsResizing means the window is actively being resized by user -- don't try
// to update otherwise
func (w *RenderWin) IsResizing() bool {
	return w.HasFlag(WinFlagIsResizing)
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
		rctx.Visible = false
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
	if drw.Impl.MaxTextures != vgpu.MaxTexturesPerSet*3 { // this is essential after hibernate
		drw.SetMaxTextures(vgpu.MaxTexturesPerSet * 3) // use 3 sets
	}
	// w.FocusInactivate()
	// w.InactivateAllSprites()
	if !w.IsVisible() {
		rctx.Visible = false
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
	rctx.Visible = true
	w.StageMgr.Resize(sz)
	// w.ConfigInsets()
	if WinGeomTrace {
		log.Printf("WinGeomPrefs: recording from Resize\n")
	}
	WinGeomMgr.RecordPref(w)
	// w.FullReRender()
	// we need a second full re-render for fullscreen and snap resizes on RenderWins
	// if goosi.TheApp.Platform() == goosi.RenderWins {
	// 	w.FullReRender()
	// }

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
// definitely close it -- flags window as such -- check IsClosing()
func (w *RenderWin) Close() {
	if w.IsClosing() {
		return
	}
	// this causes hangs etc: not good
	// w.StageMgr.RenderCtx.Mu.Lock() // allow other stuff to finish
	w.SetFlag(true, WinFlagIsClosing)
	// w.StageMgr.RenderCtx.Mu.Unlock()
	w.GoosiWin.Close()
}

// CloseReq requests that the window be closed -- could be rejected
func (w *RenderWin) CloseReq() {
	w.GoosiWin.CloseReq()
}

// Closed frees any resources after the window has been closed.
func (w *RenderWin) Closed() {
	// w.StageMgr.RenderCtx.Mu.Lock()
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
		// w.StageMgr.RenderCtx.Mu.Unlock()
		return
	}
	// w.SetDisabled() // marks as closed
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
	// w.StageMgr.RenderCtx.Mu.Unlock()
}

// IsClosed reports if the window has been closed
func (w *RenderWin) IsClosed() bool {
	// if w.IsDisabled() || w.Scene == nil {
	// 	return true
	// }
	return false
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
	if w == nil || w.GoosiWin == nil || w.IsClosed() || w.IsClosing() || !w.GoosiWin.IsVisible() {
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
	w.SetFlag(true, WinFlagDoFullRender)
	w.EventLoop()
}

// GoStartEventLoop starts the event processing loop for this window in a new
// goroutine, and returns immediately.  Adds to WinWait waitgroup so a main
// thread can wait on that for all windows to close.
func (w *RenderWin) GoStartEventLoop() {
	WinWait.Add(1)
	w.SetFlag(true, WinFlagDoFullRender, WinFlagGoLoop)
	go w.EventLoop()
}

// StopEventLoop tells the event loop to stop running when the next event arrives.
func (w *RenderWin) StopEventLoop() {
	w.SetFlag(true, WinFlagStopEventLoop)
}

// SendCustomEvent sends a custom event with given data to this window -- widgets can connect
// to receive CustomEventTypes events to receive them.  Sometimes it is useful
// to send a custom event just to trigger a pass through the event loop, even
// if nobody is listening (e.g., if a popup is posted without a surrounding
// event, as in Complete.ShowCompletions
func (w *RenderWin) SendCustomEvent(data any) {
	goosi.SendCustomEvent(w.GoosiWin, data)
}

/////////////////////////////////////////////////////////////////////////////
//                   Rendering

// SendShowEvent sends the WinShowEvent to anyone listening -- only sent once..
func (w *RenderWin) SendShowEvent() {
	if w.HasFlag(WinFlagSentShow) {
		return
	}
	w.SetFlag(true, WinFlagSentShow)
	se := window.NewShowEvent()
	se.Init()
	w.StageMgr.HandleEvent(se)
}

// SendWinFocusEvent sends the RenderWinFocusEvent to widgets
func (w *RenderWin) SendWinFocusEvent(act window.Actions) {
	se := window.NewFocusEvent(act)
	se.Init()
	w.StageMgr.HandleEvent(se)
}

// // FullReRender performs a full re-render of the window -- each node renders
// // into its scene, aggregating into the main window scene, which will
// // drive an UploadAllScenes call after all the rendering is done, and
// // signal the publishing of the window after that
// func (w *RenderWin) FullReRender() {
// 	if !w.IsVisible() {
// 		return
// 	}
// 	if WinEventTrace {
// 		fmt.Printf("Win: %v FullReRender (w.Scene.SetNeedsFullRender)\n", w.Name)
// 	}
// 	w.Scene.SetNeedsFullRender()
// 	w.InitialFocus()
// }

// InitialFocus establishes the initial focus for the window if no focus
// is set -- uses ActivateStartFocus or FocusNext as backup.
func (w *RenderWin) InitialFocus() {
	// w.EventMgr.InitialFocus()
	if prof.Profiling {
		now := time.Now()
		opent := now.Sub(RenderWinOpenTimer)
		fmt.Printf("Win: %v took: %v to open\n", w.Name, opent)
	}
}

/*

// UploadScRegion uploads image for one scene region on the screen, using
// vpBBox bounding box for the scene, and winBBox bounding box for the
// window -- called after re-rendering specific nodes to update only the
// relevant part of the overall scene image
func (w *RenderWin) UploadScRegion(sc *Scene, vpBBox, winBBox image.Rectangle) {
	// fmt.Printf("win upload vpbox: %v  winbox: %v\n", vpBBox, winBBox)
	winrel := winBBox.Min.Sub(vpBBox.Min)
	if !w.IsVisible() {
		return
	}
	vpPixBB := sc.Pixels.Rect
	inBB := vpBBox.Intersect(vpPixBB)
	if inBB.Empty() {
		return
	}
	vpBBox = inBB
	winBBox = vpBBox.Add(winrel) // fix again
	w.StageMgr.RenderCtx.Mu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.StageMgr.RenderCtx.Mu.Unlock()
		return
	}
	w.SetWinUpdating()
	// pr := prof.Start("win.UploadScRegion")

	idx, over := w.UpdtRegs.Add(winBBox, sc)
	if over {
		w.ResetUpdateRegionsImpl()
		if RenderTrace || WinEventTrace {
			fmt.Printf("Win: %v region Sc %v, winbbox: %v reset updates\n", w.Name, sc.Path(), winBBox)
		}
	} else {
		drw := w.GoosiWin.Drawer()
		sc.Pixels.Rect = vpBBox
		drw.SetGoImage(idx, 0, sc.Pixels, vgpu.NoFlipY)
		sc.Pixels.Rect = vpPixBB
		if RenderTrace || WinEventTrace {
			fmt.Printf("Win: %v uploaded region Sc %v, winbbox: %v to index: %d\n", w.Name, sc.Path(), winBBox, idx)
		}

		if w.DirDraws.Nodes != nil {
			for _, dkv := range w.DirDraws.Nodes.Order {
				dbb := dkv.Val
				if dbb.In(winBBox) {
					oridx := idx - w.UpdtRegs.StartIdx
					w.UpdtRegs.MoveIdxToBeforeDir(oridx)
				}
			}
		}
	}
	// pr.End()
	w.ClearWinUpdating()
	w.StageMgr.RenderCtx.Mu.Unlock()
}

// UploadSc uploads entire scene image for given scene -- e.g., for
// popups etc updating separately
func (w *RenderWin) UploadSc(sc *Scene, offset image.Point) {
	vpr := sc.Pixels.Bounds()
	winBBox := vpr.Add(offset)
	if !w.IsVisible() {
		return
	}
	w.StageMgr.RenderCtx.Mu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.StageMgr.RenderCtx.Mu.Unlock()
		return
	}
	w.SetWinUpdating()
	idx := 0
	drw := w.GoosiWin.Drawer()
	if sc == w.Scene {
		drw.SetGoImage(idx, 0, sc.Pixels, vgpu.NoFlipY)
	} else {
		// pr := prof.Start("win.UploadSc")
		gii, _ := AsWidget(sc.This())
		if gii != nil {
			idx, _ = w.PopDraws.Add(gii, winBBox)
			drw.SetGoImage(idx, 0, sc.Pixels, vgpu.NoFlipY)
		}
	}
	if RenderTrace || WinEventTrace {
		fmt.Printf("Win: %v uploaded entire Sc, winbbox: %v to index: %d\n", w.Name, winBBox, idx)
	}
	w.ClearWinUpdating()
	w.StageMgr.RenderCtx.Mu.Unlock()
}

// UploadAllScenes does a complete upload of all active scenes, in the
// proper order, so as to completely refresh the window texture based on
// everything rendered
func (w *RenderWin) UploadAllScenes() {
	if !w.IsVisible() {
		return
	}
	w.StageMgr.RenderCtx.Mu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.StageMgr.RenderCtx.Mu.Unlock()
		return
	}
	w.SetWinUpdating()
	// pr := prof.Start("win.UploadAllScenes")
	if RenderTrace || WinEventTrace {
		fmt.Printf("Win: %v uploading full Sc, image bound: %v, updt: %v\n", w.Name, w.Scene.Pixels.Bounds(), updt)
	}
	w.ResetUpdateRegionsImpl()

	// fmt.Printf("upload all views pop unlocked: %v\n", w.Name)
	// pr.End()
	w.ClearWinUpdating()
	w.StageMgr.RenderCtx.Mu.Unlock() // need to unlock before publish
}

*/

// IsWinUpdating checks if we are already updating window
func (w *RenderWin) IsWinUpdating() bool {
	return w.HasFlag(WinFlagUpdating)
}

// SetWinUpdating sets the window updating state to true if not already updating
func (w *RenderWin) SetWinUpdating() {
	w.SetFlag(true, WinFlagUpdating)
}

// ClearWinUpdating sets the window updating state to false if not already updating
func (w *RenderWin) ClearWinUpdating() {
	w.SetFlag(false, WinFlagUpdating)
}

/*
// ResetUpdateRegions resets any existing window update regions and
// grabs the current state of the window scene for subsequent rendering.
// This is protected by update mutex and suitable for random calls eg.,
// when a window is scrolling and positions are then invalid.
func (w *RenderWin) ResetUpdateRegions() {
	w.StageMgr.RenderCtx.Mu.Lock() // block all updates while we publish
	w.ResetUpdateRegionsImpl()
	w.StageMgr.RenderCtx.Mu.Unlock()
}

// ResetUpdateRegionsImpl resets any existing window update regions and
// grabs the current state of the window and popup scenes for subsequent rendering.
func (w *RenderWin) ResetUpdateRegionsImpl() {
	w.UpdtRegs.Reset()
	w.PopDraws.Reset()
	drw := w.GoosiWin.Drawer()
	drw.SetGoImage(0, 0, w.Scene.Pixels, vgpu.NoFlipY)
	// then all the current popups
	// fmt.Printf("upload all views pop locked: %v\n", w.Name)
	if w.PopupStack != nil {
		for _, pop := range w.PopupStack {
			gii, _ := AsWidget(pop)
			if gii != nil {
				sc := gii.AsScene()
				r := sc.Geom.Bounds()
				idx, _ := w.PopDraws.Add(gii, sc.WinBBox)
				drw.SetGoImage(idx, 0, sc.Pixels, vgpu.NoFlipY)
				if RenderTrace {
					fmt.Printf("Win: %v uploading popup stack Sc %v, win pos: %v, vp bounds: %v  idx: %d\n", w.Name, sc.Path(), r.Min, sc.Pixels.Bounds(), idx)
				}
			}
		}
	}
	if w.Popup != nil {
		gii, _ := AsWidget(w.Popup)
		if gii != nil {
			sc := gii.AsScene()
			r := sc.Geom.Bounds()
			idx, _ := w.PopDraws.Add(gii, sc.WinBBox)
			drw.SetGoImage(idx, 0, sc.Pixels, vgpu.NoFlipY)
			if RenderTrace || WinEventTrace {
				fmt.Printf("Win: %v uploading top popup Sc %v, win pos: %v, vp bounds: %v  idx: %d\n", w.Name, sc.Path(), r.Min, sc.Pixels.Bounds(), idx)
			}
		}
	}
}

// Publish does the final step of updating of the window based on the current
// texture (and overlay texture if active)
func (w *RenderWin) Publish() {
	if !w.IsVisible() || w.GoosiWin.IsMinimized() {
		if WinEventTrace {
			fmt.Printf("skipping update on inactive / minimized window: %v\n", w.Name)
		}
		return
	}
	w.StageMgr.RenderCtx.Mu.Lock() // block all updates while we publish
	if !w.IsVisible() {            // could have closed while we waited for lock
		if WinEventTrace {
			fmt.Printf("skipping update on inactive / minimized window: %v\n", w.Name)
		}
		w.StageMgr.RenderCtx.Mu.Unlock()
		return
	}

	// note: this is key for finding redundant updates!
	if WinPublishTrace {
		fmt.Printf("\n\n###################################\n%v\n", string(debug.Stack()))
	}

	w.SetWinUpdating()
	if Update2DTrace || WinEventTrace {
		fmt.Printf("Win %v doing publish\n", w.Name)
	}
	// pr := prof.Start("win.Publish")

	// note: vulkan complains about different threads for rendering but should be ok.
	// can't use RunOnWin method because it locks for main thread windows.
	// w.GoosiWin.RunOnWin(func() {})
	// and using RunOnMain makes the thing hella slow -- like opengl -- that was the issue there!
	// goosi.TheApp.RunOnMain(func() {

	// if w.Sprites.Modified || w.Sprites.HasSizeChanged() {
	// 	w.ConfigSprites()
	// 	w.Sprites.Modified = false
	// }

	drw := w.GoosiWin.Drawer()
	vpv := drw.GetImageVal(0).Texture
	if !vpv.HasFlag(Active) {
		if w.Scene.Pixels == nil {
			if Update2DTrace {
				fmt.Printf("Win %s didn't have active image, scene is nil\n", w.Name)
			}
			w.ClearWinUpdating()
			w.StageMgr.RenderCtx.Mu.Unlock()
			return
		} else {
			if Update2DTrace {
				fmt.Printf("Win %s didn't have active image, setting to: %v\n", w.Name, w.Scene.Pixels.Bounds())
			}
			drw.SetGoImage(0, 0, w.Scene.Pixels, vgpu.NoFlipY)
		}
	}
	drw.SyncImages()
	drw.StartDraw(0)
	drw.UseTextureSet(0)
	drw.Scale(0, 0, drw.Surf.Format.Bounds(), image.Rectangle{}, draw.Src, vgpu.NoFlipY)
	if len(w.UpdtRegs.BeforeDir) > 0 {
		drw.UseTextureSet(1)
		w.UpdtRegs.DrawImages(drw, true) // before direct
		drw.UseTextureSet(0)
	}
	w.DirDraws.DrawImages(drw)
	w.PopDraws.DrawImages(drw)

	drw.UseTextureSet(1)
	w.UpdtRegs.DrawImages(drw, false) // after direct

	drw.UseTextureSet(2)
	w.DrawSprites()

	drw.EndDraw()

	if WinDrawTrace { // debugging color overlay
		var clrs [16]color.RGBA
		cmap := colormap.AvailMaps["ROYGBIV"]
		for i := 0; i < 16; i++ {
			clrs[i] = cmap.Map(float64(i) / 16.0)
			clrs[i].A = 16
		}
		wu := &w.UpdtRegs
		if wu.Updates != nil {
			drw.StartFill()
			for i, kv := range wu.Updates.Order {
				winBBox := kv.Key
				drw.FillRect(clrs[i], winBBox, draw.Over)
			}
			drw.EndFill()
		}
	}
	// })

	// 	if RenderTrace {
	// 		fmt.Printf("Win %v did publish\n", w.Name)
	// 	}
	// pr.End()

	w.ClearWinUpdating()
	w.StageMgr.RenderCtx.Mu.Unlock()
}

*/

// AddDirectUploader adds given node to those that have a DirectWinUpload method
// and directly render to the Drawer Texture via their own method.
// This is for gi3d.Scene for example.  Returns the index of the image to upload to.
func (w *RenderWin) AddDirectUploader(node Widget) int {
	w.StageMgr.RenderCtx.Mu.Lock()
	idx, _ := w.DirDraws.Add(node, image.Rectangle{})
	w.StageMgr.RenderCtx.Mu.Unlock()
	return idx
}

// DeleteDirectUploader removes given node to those that have a DirectWinUpload method.
func (w *RenderWin) DeleteDirectUploader(node Widget) {
	w.StageMgr.RenderCtx.Mu.Lock()
	// w.DirDraws.Nodes.DeleteKey(node.AsWidget())
	w.StageMgr.RenderCtx.Mu.Unlock()
}

/*
/////////////////////////////////////////////////////////////////////////////
//                   Sprites

// SpriteByName returns a sprite by name -- false if not created yet
func (w *RenderWin) SpriteByName(nm string) (*Sprite, bool) {
	w.StageMgr.RenderCtx.Mu.Lock()
	defer w.StageMgr.RenderCtx.Mu.Unlock()
	return w.Sprites.SpriteByName(nm)
}

// AddSprite adds an existing sprite to list of sprites, using the sprite.Name
// as the unique name key.
func (w *RenderWin) AddSprite(sp *Sprite) {
	w.StageMgr.RenderCtx.Mu.Lock()
	defer w.StageMgr.RenderCtx.Mu.Unlock()
	w.Sprites.Add(sp)
	if sp.On {
		w.Sprites.Active++
	}
}

// ActivateSprite flags the sprite as active, and increments
// number of Active Sprites, so that it will actually be rendered.
// it is assumed that the image has not changed.
func (w *RenderWin) ActivateSprite(nm string) {
	w.StageMgr.RenderCtx.Mu.Lock()
	defer w.StageMgr.RenderCtx.Mu.Unlock()

	sp, ok := w.Sprites.SpriteByName(nm)
	if !ok {
		return // not worth bothering about errs -- use a consistent string var!
	}
	if !sp.On {
		sp.On = true
		w.Sprites.Active++
	}
}

// InactivateSprite flags the sprite as inactive, and decrements
// number of Active Sprites, so that it will not be rendered.
func (w *RenderWin) InactivateSprite(nm string) {
	w.StageMgr.RenderCtx.Mu.Lock()
	defer w.StageMgr.RenderCtx.Mu.Unlock()

	sp, ok := w.Sprites.SpriteByName(nm)
	if !ok {
		return // not worth bothering about errs -- use a consistent string var!
	}
	if sp.On {
		sp.On = false
		w.Sprites.Active--
	}
}

// InactivateAllSprites inactivates all sprites
func (w *RenderWin) InactivateAllSprites() {
	w.StageMgr.RenderCtx.Mu.Lock()
	defer w.StageMgr.RenderCtx.Mu.Unlock()

	for _, sp := range w.Sprites.Names.Order {
		if sp.Val.On {
			sp.Val.On = false
			w.Sprites.Active--
		}
	}
}

// DeleteSprite deletes given sprite, returns true if actually deleted.
// requires updating other sprites of same size -- use Inactivate if any chance of re-use.
func (w *RenderWin) DeleteSprite(nm string) bool {
	w.StageMgr.RenderCtx.Mu.Lock()
	defer w.StageMgr.RenderCtx.Mu.Unlock()

	sp, ok := w.Sprites.SpriteByName(nm)
	if !ok {
		return false
	}
	w.Sprites.Delete(sp)
	w.Sprites.Active--
	return true
}

// SpriteEvent processes given event for any active sprites
func (w *RenderWin) SelSpriteEvent(evi goosi.Event) {
	// w.StageMgr.RenderCtx.Mu.Lock()
	// defer w.StageMgr.RenderCtx.Mu.Unlock()

	et := evi.Type()

	for _, spkv := range w.Sprites.Names.Order {
		sp := spkv.Val
		if !sp.On {
			continue
		}
		if sp.Events == nil {
			continue
		}
		sig, ok := sp.Events[et]
		if !ok {
			continue
		}
		ep := evi.Pos()
		if et == goosi.MouseDragEvent {
			if sp.Name == w.SpriteDragging {
				sig.Emit(w.This(), int64(et), evi)
			}
		} else if ep.In(sp.Geom.Bounds()) {
			sig.Emit(w.This(), int64(et), evi)
		}
	}
}

// ConfigSprites updates the Drawer configuration of sprites.
// Does a new SzAlloc, and sets corresponding images.
func (w *RenderWin) ConfigSprites() {
	drw := w.GoosiWin.Drawer()
	w.Sprites.AllocSizes()
	sa := &w.Sprites.SzAlloc
	for gpi, ga := range sa.GpAllocs {
		gsz := sa.GpSizes[gpi]
		imgidx := SpriteStart + gpi
		drw.ConfigImage(imgidx, vgpu.NewImageFormat(gsz.X, gsz.Y, len(ga)))
		for ii, spi := range ga {
			if err := w.Sprites.Names.IdxIsValid(spi); err != nil {
				fmt.Println(err)
				continue
			}
			sp := w.Sprites.Names.ValByIdx(spi)
			drw.SetGoImage(imgidx, ii, sp.Pixels, vgpu.NoFlipY)
		}
	}
}

// DrawSprites draws sprites
func (w *RenderWin) DrawSprites() {
	drw := w.GoosiWin.Drawer()
	sa := &w.Sprites.SzAlloc
	for gpi, ga := range sa.GpAllocs {
		imgidx := SpriteStart + gpi
		for ii, spi := range ga {
			if w.Sprites.Names.IdxIsValid(spi) != nil {
				continue
			}
			sp := w.Sprites.Names.ValByIdx(spi)
			if !sp.On {
				continue
			}
			drw.Copy(imgidx, ii, sp.Geom.Pos, image.Rectangle{}, draw.Over, vgpu.NoFlipY)
		}
	}
}
*/

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
		w.ProcessEvent(evi)
	}
}

// EventLoop runs the event processing loop for the RenderWin -- grabs oswin
// events for the window and dispatches them to receiving nodes, and manages
// other state etc (popups, etc).
func (w *RenderWin) EventLoop() {
	for {
		if w.HasFlag(WinFlagStopEventLoop) {
			w.SetFlag(false, WinFlagStopEventLoop)
			break
		}
		evi := w.GoosiWin.NextEvent()
		if w.HasFlag(WinFlagStopEventLoop) {
			w.SetFlag(false, WinFlagStopEventLoop)
			break
		}
		w.ProcessEvent(evi)
	}
	if WinEventTrace {
		fmt.Printf("Win: %v out of event loop\n", w.Name)
	}
	if w.HasFlag(WinFlagGoLoop) {
		WinWait.Done()
	}
	// our last act must be self destruction!
}

// ProcessEvent processes given goosi.Event
func (w *RenderWin) ProcessEvent(evi goosi.Event) {
	et := evi.Type()
	log.Printf("Got event: %v\n", et)
	w.StageMgr.HandleEvent(evi)
}

/*


	w.delPop = false                      // if true, delete this popup after event loop
	if et > goosi.EventTypesN || et < 0 { // we don't handle other types of events here
		fmt.Printf("Win: %v got out-of-range event: %v\n", w.Name, et)
		return
	}

	{ // popup delete check
		w.PopMu.RLock()
		dpop := w.DelPopup
		cpop := w.Popup
		w.PopMu.RUnlock()
		if dpop != nil {
			if dpop == cpop {
				w.ClosePopup(dpop)
			} else {
				if WinEventTrace {
					fmt.Printf("zombie popup: %v  cur: %v\n", dpop.Name(), cpop.Name())
				}
			}
		}
	}
	if FilterLaggyKeyEvents || et != goosi.KeyEvent { // don't filter key events
		if !w.FilterEvent(evi) {
			return
		}
	}
	w.EventMgr.LagLastSkipped = false
	w.lastEt = et

	if w.skippedResize != nil {
		w.Scene.BBoxMu.RLock()
		vpsz := w.Scene.Geom.Size
		w.Scene.BBoxMu.RUnlock()
		if vpsz != w.GoosiWin.Size() {
			w.SetFlag(true, WinFlagIsResizing)
			w.Resized(w.GoosiWin.Size())
			w.skippedResize = nil
		}
	}

	if et != goosi.WindowResizeEvent && et != goosi.WindowPaintEvent {
		w.SetFlag(false, WinFlagIsResizing)
	}

	w.EventMgr.MouseEvents(evi)

	if !w.HiPriorityEvents(evi) {
		return
	}

	////////////////////////////////////////////////////////////////////////////
	// Send Events to Widgets

	hasFocus := w.HasFlag(WinFlagGotFocus)
	if _, ok := evi.(*mouse.ScrollEvent); ok {
		if !hasFocus {
			w.EventMgr.Scrolling = nil // not valid
		}
		hasFocus = true // doesn't need focus!
	}
	if me, ok := evi.(*mouse.Event); ok {
		hasFocus = true // also doesn't need focus (there can be hover events while not focused)
		w.SetCursor(me) // always set cursor on mouse move
	}
	// if someone clicks while in selection mode, stop selection mode and stop the event
	if me, ok := evi.(*mouse.Event); w.IsInSelectionMode() && ok {
		me.SetHandled()
		w.SetSelectionModeState(false)
		w.DeleteSprite(RenderWinSelectionSpriteName)
		w.SelectedWidgetChan <- w.SelectedWidget
	}

	if (hasFocus || !evi.OnWinFocus()) && !evi.IsHandled() {
		evToPopup := !w.CurPopupIsTooltip() // don't send events to tooltips!
		w.EventMgr.SendEventSignal(evi, evToPopup)
		if !w.delPop && et == goosi.MouseMoveEvent && !evi.IsHandled() {
			didFocus := w.EventMgr.GenMouseFocusEvents(evi.(*mouse.Event), evToPopup)
			if didFocus && w.CurPopupIsTooltip() {
				w.delPop = true
			}
		}
	}

	////////////////////////////////////////////////////////////////////////////
	// Low priority windows events

	if !evi.IsHandled() && et == goosi.KeyChordEvent {
		ke := evi.(*key.Event)
		kc := ke.Chord()
		if w.TriggerShortcut(kc) {
			evi.SetHandled()
		}
	}

	if !evi.IsHandled() {
		switch e := evi.(type) {
		case *key.Event:
			keyDelPop := w.KeyChordEventLowPri(e)
			if keyDelPop {
				w.delPop = true
			}
		}
	}

	w.EventMgr.MouseEventReset(evi)
	if evi.Type() == goosi.MouseEvent {
		me := evi.(*mouse.Event)
		if me.Action == mouse.Release {
			w.SpriteDragging = ""
		}
	}

	////////////////////////////////////////////////////////////////////////////
	// Delete popup?

	{
		cpop := w.CurPopup()
		if cpop != nil && !w.delPop {
			if PopupIsTooltip(cpop) {
				if et != goosi.MouseMoveEvent {
					w.delPop = true
				}
			} else if me, ok := evi.(*mouse.Event); ok {
				if me.Action == mouse.Release {
					if w.ShouldDeletePopupMenu(cpop, me) {
						w.delPop = true
					}
				}
			}

			if PopupIsCompleter(cpop) {
				fsz := len(w.EventMgr.FocusStack)
				if fsz > 0 && et == goosi.KeyChordEvent {
					w.EventMgr.SendSig(w.EventMgr.FocusStack[fsz-1], cpop, evi)
				}
			}
		}
	}

	////////////////////////////////////////////////////////////////////////////
	// Actually delete popup and push a new one

	if w.delPop {
		w.ClosePopup(w.CurPopup())
	}

	w.PopMu.RLock()
	npop := w.NextPopup
	w.PopMu.RUnlock()
	if npop != nil {
		w.PushPopup(npop)
	}
}

*/

// SetCursor sets the cursor based on the given mouse event.
// Also handles sending widget selection events.
func (w *RenderWin) SetCursor(me *mouse.Event) {
	/*
		if w.IsClosing() {
			return
		}
		maxLevel := 0
		maxLevelWidget := &WidgetBase{}
		maxLevelCursor := cursor.Arrow

		fun := func(k ki.Ki, level int, data any) bool {
			_, wb := AsWidget(k)
			if wb == nil {
				// could have nodes further down (eg with menu which is ki.Slice), so continue
				return ki.Continue
			}
			if !wb.PosInBBox(me.Pos()) {
				// however, if we are out of bbox, there is no way to get back in
				return ki.Break
			}
			if !wb.IsVisible() || level < maxLevel {
				// could have visible or higher level ones further down
				return ki.Continue
			}

			wb, ok := wb.This().Embed(TypeWidgetBase).(*WidgetBase)
			if !ok {
				// same logic as with Node2D
				return ki.Continue
			}
			maxLevel = level
			maxLevelWidget = wb
			maxLevelCursor = wb.Style.Cursor
			if wb.IsDisabled() {
				maxLevelCursor = cursor.Not
				// once we get to a disabled element,
				// we won't waste time going further
				return ki.Break
			}
			return ki.Continue
		}

		pop := w.CurPopup()
		if pop == nil {
			// if no popup, just do on window
			w.FuncDownMeFirst(0, nil, fun)
		} else {
			_, popni := AsWidget(pop)
			if popni == nil || !popni.PosInBBox(me.Pos()) || PopupIsTooltip(pop) {
				// if not in popup (or it is a tooltip), do on window
				w.FuncDownMeFirst(0, nil, fun)
			} else {
				// if in popup, do on popup
				popni.FuncDownMeFirst(0, nil, fun)
			}

		}

		if w.IsInSelectionMode() && maxLevelWidget != nil {
			me.SetHandled()
			w.SelectionSprite(maxLevelWidget)
			w.SelectedWidget = maxLevelWidget
			goosi.TheApp.Cursor(w.GoosiWin).Set(cursor.Arrow) // always arrow in selection mode
		} else {
			// only set cursor if not in selection mode
			goosi.TheApp.Cursor(w.GoosiWin).Set(maxLevelCursor)
		}
	*/
}

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
		draw.Draw(sp.Pixels, sp.Pixels.Bounds(), &image.Uniform{colors.SetAF32(ColorScheme.Primary, 0.5)}, image.Point{}, draw.Src)
		sp.Geom.Pos = wb.WinBBox.Min
		w.AddSprite(sp)
		w.ActivateSprite(RenderWinSelectionSpriteName)
		return sp
	*/
	return nil
}

// FilterEvent filters repeated laggy events -- key for responsive resize, scroll, etc
// returns false if event should not be processed further, and true if it should.
func (w *RenderWin) FilterEvent(evi goosi.Event) bool {
	return false
	/*
		et := evi.Type()
		if w.HasFlag(WinFlagGotPaint) && et == goosi.WindowPaintEvent && w.lastEt == goosi.WindowResizeEvent {
			if WinEventTrace {
				fmt.Printf("Win: %v skipping paint after resize\n", w.Name)
			}
			// w.Publish() // this is essential on mac for any paint event
			w.SetFlag(true, WinFlagGotPaint)
			return false // X11 always sends a paint after a resize -- we just use resize
		}

		if et != w.lastEt && w.lastEt != goosi.WindowResizeEvent {
			return true // non-repeat
		}

		if et == goosi.WindowResizeEvent {
			now := time.Now()
			lag := now.Sub(evi.Time())
			lagMs := lag / time.Millisecond
			w.SetFlag(true, WinFlagIsResizing)
			we := evi.(*window.Event)
			// fmt.Printf("resize\n")
			if lagMs > EventSkipLagMSec {
				if WinEventTrace {
					fmt.Printf("Win: %v skipped et %v lag %v size: %v\n", w.Name, et, lag, w.GoosiWin.Size())
				}
				w.EventMgr.LagLastSkipped = true
				w.skippedResize = we
				return false
			} else {
				we.SetHandled()
				w.Resized(w.GoosiWin.Size())
				w.EventMgr.LagLastSkipped = false
				w.skippedResize = nil
				return false
			}
		}
		return w.EventMgr.FilterLaggyEvents(evi)
	*/
}

// HiProrityEvents processes High-priority events for RenderWin.
// RenderWin gets first crack at these events, and handles window-specific ones
// returns true if processing should continue and false if was handled
func (w *RenderWin) HiPriorityEvents(evi goosi.Event) bool {
	switch e := evi.(type) {
	case *window.Event:
		switch e.Action {
		// case window.Resize: // note: already handled earlier in lag process
		case window.Close:
			// fmt.Printf("got close event for window %v \n", w.Name)
			w.Closed()
			w.SetFlag(true, WinFlagStopEventLoop)
			return false
		case window.Minimize:
			// on mobile platforms, we need to set the size to 0 so that it detects a size difference
			// and lets the size event go through when we come back later
			// if goosi.TheApp.Platform().IsMobile() {
			// 	w.Scene.Geom.Size = image.Point{}
			// }
		case window.Paint:
			// fmt.Printf("got paint event for window %v \n", w.Name)
			w.SetFlag(true, WinFlagGotPaint)
			if w.HasFlag(WinFlagDoFullRender) {
				w.SetFlag(false, WinFlagDoFullRender)
				// fmt.Printf("Doing full render at size: %v\n", w.Scene.Geom.Size)
				// if w.Scene.Geom.Size != w.GoosiWin.Size() {
				// 	w.Resized(w.GoosiWin.Size())
				// } else {
				// 	w.FullReRender() // note: this is currently needed for focus to actually
				// 	// take effect in a popup, and also ensures dynamically sized elements are
				// 	// properly sized.  It adds a bit of cost but really not that much and
				// 	// probably worth it overall, even if we can fix the initial focus issue
				// 	// w.InitialFocus()
				// }
			}
			// w.Publish()
		case window.Show:
			// note that this is sent delayed by vkos
			if WinEventTrace {
				fmt.Printf("Win: %v got show event\n", w.Name)
			}
			// if w.NeedWinMenuUpdate() {
			// 	w.MainMenuUpdateRenderWins()
			// }
			w.SendShowEvent() // happens AFTER full render
		case window.Move:
			e.SetHandled()
			if w.HasFlag(WinFlagGotPaint) { // moves before paint are not accurate on X11
				// fmt.Printf("win move: %v\n", w.GoosiWin.Position())
				if WinGeomTrace {
					log.Printf("WinGeomPrefs: recording from Move\n")
				}
				WinGeomMgr.RecordPref(w)
			}
		case window.Focus:
			StringsInsertFirstUnique(&FocusRenderWins, w.Name, 10)
			if !w.HasFlag(WinFlagGotFocus) {
				w.SetFlag(true, WinFlagGotFocus)
				w.SendWinFocusEvent(window.Focus)
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
		case window.DeFocus:
			if WinEventTrace {
				fmt.Printf("Win: %v lost focus\n", w.Name)
			}
			w.SetFlag(false, WinFlagGotFocus)
			w.SendWinFocusEvent(window.DeFocus)
		case window.ScreenUpdate:
			w.Resized(w.GoosiWin.Size())
			// TODO: figure out how to restore this stuff without breaking window size on mobile

			// WinGeomMgr.AbortSave() // anything just prior to this is sus
			// if !goosi.TheApp.NoScreens() {
			// 	Prefs.UpdateAll()
			// 	WinGeomMgr.RestoreAll()
			// }
		}
		return false // don't do anything else!
	case *mouse.Event:
		// if w.EventMgr.DNDStage == DNDStarted {
		// 	w.DNDMoveEvent(e)
		// } else {
		// 	w.SelSpriteEvent(evi)
		// 	if !w.EventMgr.dragStarted {
		// 		e.SetHandled() // ignore
		// 	}
		// }
		// case *mouse.Event:
		// if w.EventMgr.DNDStage == DNDStarted && e.Action == mouse.Release {
		// 	w.DNDDropEvent(e)
		// }
		// w.FocusActiveClick(e)
		// w.SelSpriteEvent(evi)
		// if w.NeedWinMenuUpdate() {
		// 	w.MainMenuUpdateRenderWins()
		// }
		// case *mouse.Event:
		// todo:
		// if bitflag.HasAllAtomic(&w.Flag, WinFlagGotPaint), WinFlagGotFocus)) {
		if w.HasFlag(WinFlagDoFullRender) {
			w.SetFlag(false, WinFlagDoFullRender)
			// if we are getting mouse input, and still haven't done this, do it..
			// fmt.Printf("Doing full render at size: %v\n", w.Scene.Geom.Size)
			// if w.Scene.Geom.Size != w.GoosiWin.Size() {
			// 	w.Resized(w.GoosiWin.Size())
			// } else {
			// 	w.FullReRender()
			// }
			w.SendShowEvent() // happens AFTER full render
		}
		// if w.EventMgr.Focus == nil { // not using lock-protected b/c can conflict with popup
		// w.EventMgr.ActivateStartFocus()
		// }
		// }
	case *dnd.Event:
		// if e.Action == dnd.External {
		// 	w.EventMgr.DNDDropMod = e.Mod
		// }
	case *key.Event:
		// keyDelPop := w.KeyChordEventHiPri(e)
		// if keyDelPop {
		// 	w.delPop = true
		// }
	}
	return true
}

/////////////////////////////////////////////////////////////////////////////
//                   Sending Events

// Most of event stuff is in events.go, controlled by EventMgr

// func (w *RenderWin) EventTopNode() ki.Ki {
// 	return w.This()
// }
//
// func (w *RenderWin) FocusTopNode() ki.Ki {
// 	cpop := w.CurPopup()
// 	if cpop != nil {
// 		return cpop
// 	}
// 	return w.Scene.This()
// }

// IsInScope returns true if the given object is in scope for receiving events.
// If popup is true, then only items on popup are in scope, otherwise
// items NOT on popup are in scope (if no popup, everything is in scope).
func (w *RenderWin) IsInScope(k ki.Ki, popup bool) bool {
	/*
		cpop := w.CurPopup()
		if cpop == nil {
			return true
		}
		if k.This() == cpop {
			return popup
		}
		_, wb := AsWidget(k)
		if wb == nil {
			np := k.ParentByType(TypeNode2DBase, ki.Embeds)
			if np != nil {
				wb = np.Embed(TypeNode2DBase).(*Node2DBase)
			} else {
				return false
			}
		}
		mvp := wb.Sc
		if mvp == nil {
			return false
		}
		if mvp.This() == cpop {
			return popup
		}
		return !popup
	*/
	return false
}

// AddShortcut adds given shortcut to given action.
func (w *RenderWin) AddShortcut(chord key.Chord, act *Action) {
	if chord == "" {
		return
	}
	if w.Shortcuts == nil {
		w.Shortcuts = make(Shortcuts, 100)
	}
	sa, exists := w.Shortcuts[chord]
	if exists && sa != act && sa.Text != act.Text {
		if KeyEventTrace {
			log.Printf("gi.RenderWin shortcut: %v already exists on action: %v -- will be overwritten with action: %v\n", chord, sa.Text, act.Text)
		}
	}
	w.Shortcuts[chord] = act
}

// DeleteShortcut deletes given shortcut
func (w *RenderWin) DeleteShortcut(chord key.Chord, act *Action) {
	if chord == "" {
		return
	}
	if w.Shortcuts == nil {
		return
	}
	sa, exists := w.Shortcuts[chord]
	if exists && sa == act {
		delete(w.Shortcuts, chord)
	}
}

// TriggerShortcut attempts to trigger a shortcut, returning true if one was
// triggered, and false otherwise.  Also eliminates any shortcuts with deleted
// actions, and does not trigger for Inactive actions.
func (w *RenderWin) TriggerShortcut(chord key.Chord) bool {
	if KeyEventTrace {
		fmt.Printf("Shortcut chord: %v -- looking for action\n", chord)
	}
	if w.Shortcuts == nil {
		return false
	}
	sa, exists := w.Shortcuts[chord]
	if !exists {
		return false
	}
	if sa.IsDestroyed() {
		delete(w.Shortcuts, chord)
		return false
	}
	if sa.IsDisabled() {
		if KeyEventTrace {
			fmt.Printf("Shortcut chord: %v, action: %v -- is inactive, not fired\n", chord, sa.Text)
		}
		return false
	}

	if KeyEventTrace {
		fmt.Printf("Win: %v Shortcut chord: %v, action: %v triggered\n", w.Name, chord, sa.Text)
	}
	sa.Trigger()
	return true
}

/////////////////////////////////////////////////////////////////////////////
//                   Key Events Handled by RenderWin

// KeyChordEventHiPri handles all the high-priority window-specific key
// events, returning its input on whether any existing popup should be deleted
func (w *RenderWin) KeyChordEventHiPri(e *key.Event) bool {
	delPop := false
	if KeyEventTrace {
		fmt.Printf("RenderWin HiPri KeyInput: %v event: %v\n", w.Name, e.String())
	}
	if e.IsHandled() {
		return false
	}
	cs := e.Chord()
	kf := KeyFun(cs)
	// cpop := w.CurPopup()
	switch kf {
	case KeyFunWinClose:
		w.CloseReq()
		e.SetHandled()
	case KeyFunMenu:
		if w.MainMenu != nil {
			w.MainMenu.GrabFocus()
			e.SetHandled()
		}
	case KeyFunAbort:
		// if PopupIsMenu(cpop) || PopupIsTooltip(cpop) {
		// 	delPop = true
		// 	e.SetHandled()
		// } else if w.EventMgr.DNDStage > DNDNotStarted {
		// 	w.ClearDragNDrop()
		// }
	case KeyFunAccept:
		// if PopupIsMenu(cpop) || PopupIsTooltip(cpop) {
		// 	delPop = true
		// }
	}
	// fmt.Printf("key chord: rune: %v Chord: %v\n", e.Rune, e.Chord())
	return delPop
}

// KeyChordEventLowPri handles all the lower-priority window-specific key
// events, returning its input on whether any existing popup should be deleted
func (w *RenderWin) KeyChordEventLowPri(e *key.Event) bool {
	if e.IsHandled() {
		return false
	}
	// w.EventMgr.ManagerKeyChordEvents(e)
	if e.IsHandled() {
		return false
	}
	cs := e.Chord()
	kf := KeyFun(cs)
	delPop := false
	switch kf {
	case KeyFunWinSnapshot:
		dstr := time.Now().Format("Mon_Jan_2_15:04:05_MST_2006")
		fnm, _ := filepath.Abs("./GrabOf_" + w.Name + "_" + dstr + ".png")
		// SaveImage(fnm, w.Scene.Pixels)
		fmt.Printf("Saved RenderWin Image to: %s\n", fnm)
		e.SetHandled()
	case KeyFunZoomIn:
		w.ZoomDPI(1)
		e.SetHandled()
	case KeyFunZoomOut:
		w.ZoomDPI(-1)
		e.SetHandled()
	case KeyFunRefresh:
		e.SetHandled()
		fmt.Printf("Win: %v display refreshed\n", w.Name)
		goosi.TheApp.GetScreens()
		Prefs.UpdateAll()
		WinGeomMgr.RestoreAll()
		// w.FocusInactivate()
		// w.FullReRender()
		// sz := w.GoosiWin.Size()
		// w.SetSize(sz)
	case KeyFunWinFocusNext:
		e.SetHandled()
		AllRenderWins.FocusNext()
	}
	switch cs { // some other random special codes, during dev..
	case "Control+Alt+R":
		// ProfileToggle()
		e.SetHandled()
	case "Control+Alt+F":
		// w.BenchmarkFullRender()
		e.SetHandled()
	case "Control+Alt+H":
		// w.BenchmarkReRender()
		e.SetHandled()
	}
	// fmt.Printf("key chord: rune: %v Chord: %v\n", e.Rune, e.Chord())
	return delPop
}

/////////////////////////////////////////////////////////////////////////////
//                   Key Focus

// FocusActiveClick updates the FocusActive status based on mouse clicks in
// or out of the focused item
func (w *RenderWin) FocusActiveClick(e *mouse.Event) {
	/*
		cfoc := w.EventMgr.CurFocus()
		if cfoc == nil || e.Button != mouse.Left || e.Action != mouse.Press {
			return
		}
		cpop := w.CurPopup()
		if cpop != nil { // no updating on popups
			return
		}
		wi, wb := AsWidget(cfoc)
		if wb != nil && wb.This() != nil {
			if wb.PosInBBox(e.Pos()) {
				if !w.HasFlag(WinFlagFocusActive) {
					w.SetFlag(true, WinFlagFocusActive)
					wi.FocusChanged(FocusActive)
				}
			} else {
				if w.MainMenu != nil {
					if w.MainMenu.PosInBBox(e.Pos()) { // main menu is not inactivating!
						return
					}
				}
				if w.HasFlag(WinFlagFocusActive) {
					w.SetFlag(false, WinFlagFocusActive)
					wi.FocusChanged(FocusInactive)
				}
			}
		}
	*/
}

/*
// FocusInactivate inactivates the current focus element
func (w *RenderWin) FocusInactivate() {
	cfoc := w.EventMgr.CurFocus()
	if cfoc == nil || !w.HasFlag(WinFlagFocusActive) {
		return
	}
	wi, wb := AsWidget(cfoc)
	if wb != nil && wb.This() != nil {
		w.SetFlag(false, WinFlagFocusActive)
		wi.FocusChanged(FocusInactive)
	}
}

// IsRenderWinInFocus returns true if this window is the one currently in focus
func (w *RenderWin) IsRenderWinInFocus() bool {
	fwin := goosi.TheApp.RenderWinInFocus()
	if w.GoosiWin == fwin {
		return true
	}
	return false
}

// RenderWinInFocus returns the window in focus according to goosi.
// There is a small chance it could be nil.
func RenderWinInFocus() *RenderWin {
	fwin := goosi.TheApp.RenderWinInFocus()
	fw, _ := AllRenderWins.FindRenderWin(fwin)
	return fw
}

/////////////////////////////////////////////////////////////////////////////
//                   DND: Drag-n-Drop

const DNDSpriteName = "gi.RenderWin:DNDSprite"

// StartDragNDrop is called by a node to start a drag-n-drop operation on
// given source node, which is responsible for providing the data and Sprite
// representation of the node.
func (w *RenderWin) StartDragNDrop(src ki.Ki, data mimedata.Mimes, sp *Sprite) {
	w.EventMgr.DNDStart(src, data)
	if _, sw := AsWidget(src); sw != nil {
		sp.SetBottomPos(sw.LayState.Alloc.Pos.ToPo)
	}
	w.DeleteSprite(DNDSpriteName)
	sp.Name = DNDSpriteName
	sp.On = true
	w.AddSprite(sp)
	w.DNDSetCursor(dnd.DefaultModBits(w.EventMgr.LastModBits))
}

// DNDMoveEvent handles drag-n-drop move events.
func (w *RenderWin) DNDMoveEvent(e *mouse.Event) {
	sp, ok := w.SpriteByName(DNDSpriteName)
	if ok {
		sp.SetBottomPos(e.Where)
	}
	de := w.EventMgr.SendDNDMoveEvent(e)
	w.DNDUpdateCursor(de.Mod)
	e.SetHandled()
}

// DNDDropEvent handles drag-n-drop drop event (action = release).
func (w *RenderWin) DNDDropEvent(e *mouse.Event) {
	proc := w.EventMgr.SendDNDDropEvent(e)
	if !proc {
		w.ClearDragNDrop()
	}
}

// FinalizeDragNDrop is called by a node to finalize the drag-n-drop
// operation, after given action has been performed on the target -- allows
// target to cancel, by sending dnd.DropIgnore.
func (w *RenderWin) FinalizeDragNDrop(action dnd.DropMods) {
	if w.EventMgr.DNDStage != DNDDropped {
		w.ClearDragNDrop()
		return
	}
	if w.EventMgr.DNDFinalEvent == nil { // shouldn't happen...
		w.ClearDragNDrop()
		return
	}
	de := w.EventMgr.DNDFinalEvent
	de.ClearHandled()
	de.Mod = action
	if de.Source != nil {
		de.Action = dnd.DropFmSource
		w.EventMgr.SendSig(de.Source, w, de)
	}
	w.ClearDragNDrop()
}

// ClearDragNDrop clears any existing DND values.
func (w *RenderWin) ClearDragNDrop() {
	w.EventMgr.ClearDND()
	w.DeleteSprite(DNDSpriteName)
	w.DNDClearCursor()
}

// DNDModCursor gets the appropriate cursor based on the DND event mod.
func DNDModCursor(dmod dnd.DropMods) cursor.Shapes {
	switch dmod {
	case dnd.DropCopy:
		return cursor.DragCopy
	case dnd.DropMove:
		return cursor.DragMove
	case dnd.DropLink:
		return cursor.DragLink
	}
	return cursor.Not
}

// DNDSetCursor sets the cursor based on the DND event mod -- does a
// "PushIfNot" so safe for multiple calls.
func (w *RenderWin) DNDSetCursor(dmod dnd.DropMods) {
	dndc := DNDModCursor(dmod)
	goosi.TheApp.Cursor(w.GoosiWin).PushIfNot(dndc)
}

// DNDNotCursor sets the cursor to Not = can't accept a drop
func (w *RenderWin) DNDNotCursor() {
	goosi.TheApp.Cursor(w.GoosiWin).PushIfNot(cursor.Not)
}

// DNDUpdateCursor updates the cursor based on the current DND event mod if
// different from current (but no update if Not)
func (w *RenderWin) DNDUpdateCursor(dmod dnd.DropMods) bool {
	dndc := DNDModCursor(dmod)
	curs := goosi.TheApp.Cursor(w.GoosiWin)
	if !curs.IsDrag() || curs.Current() == dndc {
		return false
	}
	curs.Push(dndc)
	return true
}

// DNDClearCursor clears any existing DND cursor that might have been set.
func (w *RenderWin) DNDClearCursor() {
	curs := goosi.TheApp.Cursor(w.GoosiWin)
	for curs.IsDrag() || curs.Current() == cursor.Not {
		curs.Pop()
	}
}

/////////////////////////////////////////////////////////////////////////////
//                   Profiling and Benchmarking, controlled by hot-keys

// ProfileToggle turns profiling on or off
func ProfileToggle() {
	if prof.Profiling {
		EndTargProfile()
		EndCPUMemProfile()
	} else {
		StartTargProfile()
		StartCPUMemProfile()
	}
}

// StartCPUMemProfile starts the standard Go cpu and memory profiling.
func StartCPUMemProfile() {
	fmt.Println("Starting Std CPU / Mem Profiling")
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
}

// EndCPUMemProfile ends the standard Go cpu and memory profiling.
func EndCPUMemProfile() {
	fmt.Println("Ending Std CPU / Mem Profiling")
	pprof.StopCPUProfile()
	f, err := os.Create("mem.prof")
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	f.Close()
}

// StartTargProfile starts targeted profiling using goki prof package.
func StartTargProfile() {
	fmt.Printf("Starting Targeted Profiling\n")
	prof.Reset()
	prof.Profiling = true
}

// EndTargProfile ends targeted profiling and prints report.
func EndTargProfile() {
	prof.Report(time.Millisecond)
	prof.Profiling = false
}

// ReportWinNodes reports the number of nodes in this window
func (w *RenderWin) ReportWinNodes() {
	nn := 0
	w.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d any) bool {
		nn++
		return ki.Continue
	})
	fmt.Printf("Win: %v has: %v nodes\n", w.Name, nn)
}

// BenchmarkFullRender runs benchmark of 50 full re-renders (full restyling, layout,
// and everything), reporting targeted profile results and generating standard
// Go cpu.prof and mem.prof outputs.
func (w *RenderWin) BenchmarkFullRender() {
	fmt.Println("Starting BenchmarkFullRender")
	w.ReportWinNodes()
	StartCPUMemProfile()
	StartTargProfile()
	ts := time.Now()
	n := 50
	for i := 0; i < n; i++ {
		w.Scene.FullRenderTree()
	}
	td := time.Now().Sub(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	EndTargProfile()
	EndCPUMemProfile()
}

// BenchmarkReRender runs benchmark of 50 re-render-only updates of display
// (just the raw rendering, no styling or layout), reporting targeted profile
// results and generating standard Go cpu.prof and mem.prof outputs.
func (w *RenderWin) BenchmarkReRender() {
	fmt.Println("Starting BenchmarkReRender")
	w.ReportWinNodes()
	StartTargProfile()
	ts := time.Now()
	n := 50
	for i := 0; i < n; i++ {
		w.Scene.RenderTree()
	}
	td := time.Now().Sub(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	EndTargProfile()
}
*/
