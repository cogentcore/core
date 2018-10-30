// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sort"
	"sync"
	"time"

	"github.com/chewxy/math32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
)

// EventSkipLagMSec is the number of milliseconds of lag between the time the
// event was sent to the time it is being processed, above which a repeated
// event type (scroll, drag, resize) is skipped
var EventSkipLagMSec = 50

// DragStartMSec is the number of milliseconds to wait before initiating a
// regular mouse drag event (as opposed to a basic mouse.Press)
var DragStartMSec = 50

// DragStartPix is the number of pixels that must be moved before
// initiating a regular mouse drag event (as opposed to a basic mouse.Press)
var DragStartPix = 4

// DNDStartMSec is the number of milliseconds to wait before initiating a
// drag-n-drop event -- gotta drag it like you mean it
var DNDStartMSec = 200

// DNDStartPix is the number of pixels that must be moved before
// initiating a drag-n-drop event -- gotta drag it like you mean it
var DNDStartPix = 20

// HoverStartMSec is the number of milliseconds to wait before initiating a
// hover event
var HoverStartMSec = 1500

// HoverMaxPix is the maximum number of pixels that mouse can move and still
// register a Hover event
var HoverMaxPix = 5

// LocalMainMenu controls whether the main menu is displayed locally at top of
// each window, in addition to the global menu at the top of the screen.  Mac
// native apps do not do this, but OTOH it makes things more consistent with
// other platforms, and with larger screens, it can be convenient to have
// access to all the menu items right there.  Controlled by Prefs.Params
// variable.
var LocalMainMenu = false

// WinEventTrace reports a trace of window events to stdout
// can be set in PrefsDebug from prefs gui
// excludes mouse move events
var WinEventTrace = false

// WinPublishTrace reports the stack trace leading up to win publish events
// which are expensive -- wrap multiple updates in UpdateStart / End
// to prevent
// can be set in PrefsDebug from prefs gui
var WinPublishTrace = false

// KeyEventTrace reports a trace of keyboard events to stdout
// can be set in PrefsDebug from prefs gui
var KeyEventTrace = false

// WinNewCloseTime records last time a new window was opened or another
// closed -- used to trigger updating of Window menus on each window.
var WinNewCloseTime time.Time

// WindowGlobalMu is a mutex for any global state associated with windows
var WindowGlobalMu sync.Mutex

// notes: oswin/Image is the thing that a Vp should have uploader uploads the
// buffer/image to the window -- can also render directly onto window using
// textures using the drawer interface, but..

// Window provides an OS-specific window and all the associated event
// handling.  Widgets connect to event signals to receive relevant GUI events.
// There is a master Viewport that contains the full bitmap image of the
// window, onto which most widgets render.  For main windows (not dialogs or
// other popups), there is a master vertical layout under the Viewport
// (MasterVLay), whose first element is the MainMenu for the window (which can
// be empty, in which case it is not displayed).  On MacOS, this main menu is
// typically not directly visible, and instead updates the overall menubar.
type Window struct {
	NodeBase
	Title             string                                  `desc:"displayed name of window, for window manager etc -- window object name is the internal handle and is used for tracking property info etc"`
	OSWin             oswin.Window                            `json:"-" xml:"-" view:"-" desc:"OS-specific window interface -- handles all the os-specific functions, including delivering events etc"`
	Viewport          *Viewport2D                             `json:"-" xml:"-" desc:"convenience pointer to window's master viewport child that handles the rendering"`
	MasterVLay        *Layout                                 `json:"-" xml:"-" desc:"main vertical layout under Viewport -- first element is MainMenu (always -- leave empty to not render)"`
	MainMenu          *MenuBar                                `json:"-" xml:"-" desc:"main menu -- is first element of MasterVLay always -- leave empty to not render.  On MacOS, this drives screen main menu"`
	WinTex            oswin.Texture                           `json:"-" xml:"-" view:"-" desc:"texture for the entire window -- all rendering is done onto this texture, which is then published into the window"`
	EventSigs         [oswin.EventTypeN][EventPrisN]ki.Signal `json:"-" xml:"-" view:"-" desc:"signals for communicating each type of event, organized by priority"`
	EventMu           sync.Mutex                              `json:"-" xml:"-" view:"-" desc:"mutex that protects event sending"`
	OverTex           oswin.Texture                           `json:"-" xml:"-" view:"-" desc:"overlay texture that is updated by OverlayVp viewport"`
	OverlayVp         *Viewport2D                             `json:"-" xml:"-" desc:"a separate collection of items to be rendered as overlays -- this viewport is cleared to transparent and all the elements in it are re-rendered if any of them needs to be updated -- generally each item should be manually positioned"`
	Sprites           map[string]*Viewport2D                  `json:"-" xml:"-" desc:"sprites are named viewports that are rendered into the overlay.  If they are marked inactive then they are not rendered, otherwise automatically rendered."`
	SpritesBg         map[string]oswin.Image                  `json:"-" xml:"-" view:"-" desc:"background image for sprite rendering -- one for each sprite -- source window image is first copied into here, then sprite is rendered Over it to support transparency, and then image is uploaded to OverTex."`
	ActiveSprites     int                                     `json:"-" xml:"-" desc:"number of currentlyactive sprites -- must use ActivateSprite to keep track of whether there are active sprites."`
	UpMu              sync.Mutex                              `json:"-" xml:"-" view:"-" desc:"mutex that protects all updating / uploading of Textures"`
	LastModBits       int32                                   `json:"-" xml:"-" desc:"Last modifier key bits from most recent Mouse, Keyboard events"`
	LastSelMode       mouse.SelectModes                       `json:"-" xml:"-" desc:"Last Select Mode from most recent Mouse, Keyboard events"`
	Focus             ki.Ki                                   `json:"-" xml:"-" desc:"node receiving keyboard events"`
	FocusActive       bool                                    `json:"-" xml:"-" desc:"is the focused node active, or have other things been clicked in the meantime?"`
	StartFocus        ki.Ki                                   `json:"-" xml:"-" desc:"node to focus on at start when no other focus has been set yet"`
	FocusMu           sync.Mutex                              `json:"-" xml:"-" view:"-" desc:"mutex that protects focus updating"`
	Shortcuts         Shortcuts                               `json:"-" xml:"-" desc:"currently active shortcuts for this window (shortcuts are always window-wide -- use widget key event processing for more local key functions)"`
	DNDData           mimedata.Mimes                          `json:"-" xml:"-" desc:"drag-n-drop data -- if non-nil, then DND is taking place"`
	DNDSource         ki.Ki                                   `json:"-" xml:"-" desc:"drag-n-drop source node"`
	DNDImage          ki.Ki                                   `json:"-" xml:"-" desc:"drag-n-drop node with image of source, that is actually dragged -- typically a Bitmap but can be anything (that renders in Overlay for 2D)"`
	DNDFinalEvent     *dnd.Event                              `json:"-" xml:"-" view:"-" desc:"final event for DND which is sent if a finalize is received"`
	Dragging          ki.Ki                                   `json:"-" xml:"-" desc:"node receiving mouse dragging events -- not for DND but things like sliders -- anchor to same"`
	Scrolling         ki.Ki                                   `json:"-" xml:"-" desc:"node receiving mouse scrolling events -- anchor to same"`
	Popup             ki.Ki                                   `jsom:"-" xml:"-" desc:"Current popup viewport that gets all events"`
	PopupStack        []ki.Ki                                 `jsom:"-" xml:"-" desc:"stack of popups"`
	FocusStack        []ki.Ki                                 `jsom:"-" xml:"-" desc:"stack of focus"`
	NextPopup         ki.Ki                                   `json:"-" xml:"-" desc:"this popup will be pushed at the end of the current event cycle"`
	DelPopup          ki.Ki                                   `json:"-" xml:"-" desc:"this popup will be popped at the end of the current event cycle"`
	PopupFocus        ki.Ki                                   `json:"-" xml:"-" desc:"node to focus on when next popup is activated"`
	PopMu             sync.RWMutex                            `json:"-" xml:"-" view:"-" desc:"read-write mutex that protects popup updating"`
	TimerMu           sync.Mutex                              `json:"-" xml:"-" view:"-" desc:"mutex that protects timer variable updates (e.g., hover AferFunc's)"`
	DoFullRender      bool                                    `json:"-" xml:"-" desc:"triggers a full re-render of the window within the event loop -- cleared once done"`
	lastWinMenuUpdate time.Time
}

var KiT_Window = kit.Types.AddType(&Window{}, nil)

// EventPris for different queues of event signals, processed in priority order
type EventPris int32

const (
	// HiPri = high priority -- event receivers processed first -- can be used
	// to override default behavior
	HiPri EventPris = iota

	// RegPri = default regular priority -- most should be here
	RegPri

	// LowPri = low priority -- processed last -- typically for containers /
	// dialogs etc
	LowPri

	// LowRawPri = unfiltered (raw) low priority -- ignores whether the event
	// was already processed.
	LowRawPri

	EventPrisN

	// AllPris = -1 = all priorities (for delete cases only)
	AllPris EventPris = -1
)

//go:generate stringer -type=EventPris

// these extend NodeBase NodeFlags to hold Window state
const (
	// WinFlagHasGeomPrefs indicates if this window has WinGeomPrefs setting that sized it -- affects whether other defauld geom should be applied
	WinFlagHasGeomPrefs NodeFlags = NodeFlagsN + iota

	// WinFlagUpdating is atomic flag around global updating -- routines can check IsWinUpdating and bail
	WinFlagUpdating

	// WinFlagIsClosing is atomic flag indicating window is closing
	WinFlagIsClosing

	// WinFlagIsResizing is atomic flag indicating window is resizing
	WinFlagIsResizing

	// WinFlagOverlayVpCleared true if OverlayVp has no kids and has already been cleared -- no need to keep clearing.
	WinFlagOverlayVpCleared

	// WinFlagOverTexActive is the overlay texture active and should be uploaded to window?
	WinFlagOverTexActive

	// WinFlagGotPaint have we received our first paint event yet?  ignore other window events before this point
	WinFlagGotPaint

	// WinFlagGotFocus indicates that have we received OSWin focus
	WinFlagGotFocus

	// WinFlagGoLoop true if we are running from GoStartEventLoop -- requires a WinWait.Done at end
	WinFlagGoLoop

	// WinFlagStopEventLoop is set when event loop stop is requested
	WinFlagStopEventLoop
)

// HasGeomPrefs returns true if geometry prefs were set already
func (w *Window) HasGeomPrefs() bool {
	return w.HasFlag(int(WinFlagHasGeomPrefs))
}

// IsClosing returns true if window has requested to close -- don't
// attempt to update it any further
func (w *Window) IsClosing() bool {
	return w.HasFlag(int(WinFlagIsClosing))
}

/////////////////////////////////////////////////////////////////////////////
//                   New Windows and Init

// NewWindow creates a new window with given internal name handle, display
// name, and options.
func NewWindow(name, title string, opts *oswin.NewWindowOptions) *Window {
	Init() // overall gogi system initialization
	win := &Window{}
	win.InitName(win, name)
	win.Title = title
	win.SetOnlySelfUpdate() // has its own PublishImage update logic
	var err error
	win.OSWin, err = oswin.TheApp.NewWindow(opts)
	if err != nil {
		fmt.Printf("GoGi NewWindow error: %v \n", err)
		return nil
	}
	win.WinTex, err = oswin.TheApp.NewTexture(win.OSWin, opts.Size) // note size will be in dots
	if err != nil {
		fmt.Printf("GoGi NewTexture error: %v \n", err)
		return nil
	}
	win.OSWin.SetName(title)
	win.OSWin.SetParent(win.This())
	win.NodeSig.Connect(win.This(), SignalWindowPublish)
	win.OverlayVp = &Viewport2D{}
	win.OverlayVp.InitName(win.OverlayVp, "overlay-vp")
	return win
}

// NewWindow2D creates a new standard 2D window with given internal handle
// name, display name, and sizing, with default positioning, and initializes a
// 2D viewport within it -- stdPixels means use standardized "pixel" units for
// the display size (96 per inch), not the actual underlying raw display dot
// pixels.
func NewWindow2D(name, title string, width, height int, stdPixels bool) *Window {
	Init() // overall gogi system initialization, at latest possible moment
	opts := &oswin.NewWindowOptions{
		Title: title, Size: image.Point{width, height}, StdPixels: stdPixels,
	}
	wgp := WinGeomPrefs.Pref(name, nil)
	if wgp != nil {
		opts.Size = wgp.Size
		opts.Pos = wgp.Pos
		opts.StdPixels = false
		// fmt.Printf("got prefs for %v: size: %v pos: %v\n", name, opts.Size, opts.Pos)
		if _, found := AllWindows.FindName(name); found {
			opts.Pos.X += 20
			opts.Pos.Y += 20
		}
	}
	win := NewWindow(name, title, opts)
	if win == nil {
		return nil
	}
	if wgp != nil {
		win.SetFlag(int(WinFlagHasGeomPrefs))
	}
	AllWindows.Add(win)
	MainWindows.Add(win)
	vp := NewViewport2D(width, height)
	vp.SetName("WinVp")
	vp.SetProp("color", &Prefs.Colors.Font) // everything inherits this..

	win.AddChild(vp)
	win.Viewport = vp
	vp.Win = win
	win.ConfigVLay()
	WinNewCloseStamp()
	return win
}

// NewDialogWin creates a new dialog window with given internal handle name,
// display name, and sizing (assumed to be in raw dots), without setting its
// main viewport -- user should do win.AddChild(vp); win.Viewport = vp to set
// their own viewport.
func NewDialogWin(name, title string, width, height int, modal bool) *Window {
	opts := &oswin.NewWindowOptions{
		Title: title, Size: image.Point{width, height}, StdPixels: false,
	}
	opts.SetDialog()
	if modal {
		opts.SetModal()
	}
	wgp := WinGeomPrefs.Pref(name, nil)
	if wgp != nil {
		opts.Size = wgp.Size
		opts.Pos = wgp.Pos
		opts.StdPixels = false
	}
	win := NewWindow(name, title, opts)
	if win == nil {
		return nil
	}
	if wgp != nil {
		win.SetFlag(int(WinFlagHasGeomPrefs))
	}
	AllWindows.Add(win)
	DialogWindows.Add(win)
	WinNewCloseStamp()
	return win
}

// ConfigVLay creates and configures the vertical layout as first child of
// Viewport, and installs MainMenu as first element of layout.
func (w *Window) ConfigVLay() {
	vp := w.Viewport
	updt := vp.UpdateStart()
	defer vp.UpdateEnd(updt)
	if !vp.HasChildren() {
		vp.AddNewChild(KiT_Layout, "main-vlay")
	}
	w.MasterVLay = vp.KnownChild(0).(*Layout)
	if !w.MasterVLay.HasChildren() {
		w.MasterVLay.AddNewChild(KiT_MenuBar, "main-menu")
	}
	w.MasterVLay.Lay = LayoutVert
	w.MainMenu = w.MasterVLay.KnownChild(0).(*MenuBar)
	w.MainMenu.MainMenu = true
	w.MainMenu.SetStretchMaxWidth()
}

// SetMainWidget sets given widget as the main widget for the window -- adds
// into MasterVLay after main menu -- if a main widget has already been set then
// it is deleted and this one replaces it.  Use this method to ensure future
// compatibility.
func (w *Window) SetMainWidget(mw ki.Ki) {
	if len(w.MasterVLay.Kids) == 1 {
		w.MasterVLay.AddChild(mw)
		return
	}
	cmw := w.MasterVLay.KnownChild(1)
	if cmw != mw {
		w.MasterVLay.DeleteChildAtIndex(1, true)
		w.MasterVLay.InsertChild(mw, 1)
	}
}

// SetMainWidgetType sets the main widget of this window to given type
// (typically a Layout or Frame), and returns it.  Adds into MasterVLay after
// main menu -- if a main widget has already been set then it is deleted and
// this one replaces it.  Use this method to ensure future compatibility.
func (w *Window) SetMainWidgetType(typ reflect.Type, name string) ki.Ki {
	if len(w.MasterVLay.Kids) == 1 {
		return w.MasterVLay.AddNewChild(typ, name)
	}
	cmw := w.MasterVLay.KnownChild(1)
	if cmw.Type() != typ {
		w.MasterVLay.DeleteChildAtIndex(1, true)
		return w.MasterVLay.InsertNewChild(typ, 1, name)
	}
	return cmw
}

// SetMainFrame sets the main widget of this window as a Frame, with a default
// column-wise vertical layout and max stretch sizing, and returns that frame.
func (w *Window) SetMainFrame() *Frame {
	fr := w.SetMainWidgetType(KiT_Frame, "main-frame").(*Frame)
	fr.Lay = LayoutVert
	fr.SetStretchMaxWidth()
	fr.SetStretchMaxHeight()
	return fr
}

// SetMainLayout sets the main widget of this window as a Layout, with a default
// column-wise vertical layout and max stretch sizing, and returns it.
func (w *Window) SetMainLayout() *Layout {
	fr := w.SetMainWidgetType(KiT_Layout, "main-lay").(*Layout)
	fr.Lay = LayoutVert
	fr.SetStretchMaxWidth()
	fr.SetStretchMaxHeight()
	return fr
}

// SetName sets name of this window and also the OSWin, and applies any window
// geometry settings associated with the new name if it is different from before
func (w *Window) SetName(name string) bool {
	rval := w.NodeBase.SetName(name)
	if w.OSWin != nil {
		w.OSWin.SetName(name)
	}
	if rval && w.OSWin != nil {
		wgp := WinGeomPrefs.Pref(name, nil)
		if wgp != nil {
			if w.OSWin.Size() != wgp.Size || w.OSWin.Position() != wgp.Pos {
				// fmt.Printf("setting geom to: %v %v\n", wgp.Pos, wgp.Size)
				w.OSWin.SetGeom(wgp.Pos, wgp.Size)
			}
		}
	}
	return rval
}

// SetTitle sets title of this window and also the OSWin
func (w *Window) SetTitle(name string) {
	w.Title = name
	if w.OSWin != nil {
		w.OSWin.SetTitle(name)
	}
	WinNewCloseStamp()
}

// MainWidget returns the main widget for this window -- 2nd element in
// MasterVLay -- returns false if not yet set.
func (w *Window) MainWidget() (ki.Ki, bool) {
	return w.MasterVLay.Child(1)
}

// LogicalDPI returns the current logical dots-per-inch resolution of the
// window, which should be used for most conversion of standard units --
// physical DPI can be found in the Screen
func (w *Window) LogicalDPI() float32 {
	if w.OSWin == nil {
		return 96.0 // null default
	}
	return w.OSWin.LogicalDPI()
}

// ZoomDPI -- positive steps increase logical DPI, negative steps decrease it,
// in increments of 6 dots to keep fonts rendering clearly.
func (w *Window) ZoomDPI(steps int) {
	w.InactivateAllSprites()
	sc := oswin.TheApp.Screen(0)
	pdpi := sc.PhysicalDPI
	// ldpi = pdpi * zoom * ldpi
	cldpinet := sc.LogicalDPI
	cldpi := cldpinet / ZoomFactor
	nldpinet := cldpinet + float32(6*steps)
	if nldpinet < 6 {
		nldpinet = 6
	}
	ZoomFactor = nldpinet / cldpi
	Prefs.ApplyDPI()
	fmt.Printf("Effective LogicalDPI now: %v  PhysicalDPI: %v  Eff LogicalDPIScale: %v  ZoomFactor: %v\n", nldpinet, pdpi, nldpinet/pdpi, ZoomFactor)
	w.FullReRender()
}

// WinViewport2D returns the viewport directly under this window that serves
// as the master viewport for the entire window.
func (w *Window) WinViewport2D() *Viewport2D {
	vpi, ok := w.Children().ElemByType(KiT_Viewport2D, true, 0)
	if !ok { // shouldn't happen
		return nil
	}
	vp, _ := vpi.Embed(KiT_Viewport2D).(*Viewport2D)
	return vp
}

// SetSize requests that the window be resized to the given size -- it will
// trigger a resize event and be processed that way when it occurs.
func (w *Window) SetSize(sz image.Point) {
	w.OSWin.SetSize(sz)
}

// IsResizing means the window is actively being resized by user -- don't try
// to update otherwise
func (w *Window) IsResizing() bool {
	return w.HasFlag(int(WinFlagIsResizing))
}

// Resized updates internal buffers after a window has been resized.
func (w *Window) Resized(sz image.Point) {
	if w.IsClosed() || w.IsClosing() {
		return
	}
	curSz := w.Viewport.Geom.Size
	if curSz == sz {
		if WinEventTrace {
			fmt.Printf("Win: %v skipped same-size Resized: %v\n", w.Nm, curSz)
		}
		return
	}
	w.FocusInactivate()
	w.InactivateAllSprites()
	w.UpMu.Lock()
	if w.IsClosed() {
		if WinEventTrace {
			fmt.Printf("Win: %v Resized already closed\n", w.Nm)
		}
		w.UpMu.Unlock()
		return
	}
	if WinEventTrace {
		fmt.Printf("Win: %v Resized from: %v to: %v\n", w.Nm, curSz, sz)
	}
	if w.WinTex != nil {
		w.WinTex.Release()
	}
	if w.OverTex != nil {
		w.OverTex.Release()
	}
	w.WinTex, _ = oswin.TheApp.NewTexture(w.OSWin, sz)
	w.OverTex = nil // dynamically allocated when needed
	w.ClearFlag(int(WinFlagOverTexActive), int(WinFlagOverlayVpCleared))
	w.Viewport.Resize(sz)
	WinGeomPrefs.RecordPref(w)
	w.UpMu.Unlock()
	w.FullReRender()
}

// Close closes the window -- this is not a request -- it means:
// definitely close it -- flags window as such -- check IsClosing()
func (w *Window) Close() {
	if w.IsClosing() {
		return
	}
	w.SetFlag(int(WinFlagIsClosing))
	w.OSWin.Close()
}

// CloseReq requests that the window be closed -- could be rejected
func (w *Window) CloseReq() {
	w.OSWin.CloseReq()
}

// Closed frees any resources after the window has been closed.
func (w *Window) Closed() {
	w.UpMu.Lock()
	AllWindows.Delete(w)
	MainWindows.Delete(w)
	DialogWindows.Delete(w)
	WinNewCloseStamp()
	if WinEventTrace {
		fmt.Printf("Win: %v Closed\n", w.Nm)
	}
	if w.IsClosed() {
		if WinEventTrace {
			fmt.Printf("Win: %v Already Closed\n", w.Nm)
		}
		w.UpMu.Unlock()
		return
	}
	w.SetInactive() // marks as closed
	w.FocusInactivate()
	// these are managed by the window itself
	w.WinTex = nil
	w.OverTex = nil
	if w.OverlayVp != nil {
		w.OverlayVp.Destroy()
		w.OverlayVp = nil
	}
	for _, sp := range w.Sprites {
		sp.Destroy()
	}
	w.Sprites = nil
	for _, sp := range w.SpritesBg {
		sp.Release()
	}
	w.UpMu.Unlock()
}

// IsClosed reports if the window has been closed
func (w *Window) IsClosed() bool {
	if w.WinTex == nil || w.IsInactive() || w.Viewport == nil {
		return true
	}
	return false
}

// WinNewCloseStamp updates the global WinNewCloseTime timestamp for updating windows menus
func WinNewCloseStamp() {
	WindowGlobalMu.Lock()
	WinNewCloseTime = time.Now()
	WindowGlobalMu.Unlock()
}

// NeedWinMenuUpdate returns true if our lastWinMenuUpdate is != WinNewCloseTime
func (w *Window) NeedWinMenuUpdate() bool {
	WindowGlobalMu.Lock()
	updt := false
	if w.lastWinMenuUpdate != WinNewCloseTime {
		w.lastWinMenuUpdate = WinNewCloseTime
		updt = true
	}
	WindowGlobalMu.Unlock()
	return updt
}

// Init performs overall initialization of the gogi system: loading prefs, etc
// -- automatically called when new window opened, but can be called before
// then if pref info needed.
func Init() {
	if Prefs.LogicalDPIScale == 0 {
		Prefs.Defaults()
		PrefsDet.Defaults()
		PrefsDbg.Connect()
		Prefs.Open()
		Prefs.Apply()
		WinGeomPrefs.Open()
	}
}

/////////////////////////////////////////////////////////////////////////////
//                   Event Loop

// WinWait is a wait group for waiting for all the open window event
// loops to finish -- this can be used for cases where the initial main run
// uses a GoStartEventLoop for example.  It is incremented by GoStartEventLoop
// and decremented when the event loop terminates.
var WinWait sync.WaitGroup

// StartEventLoop is the main startup method to call after the initial window
// configuration is setup -- does any necessary final initialization and then
// starts the event loop in this same goroutine, and does not return until the
// window is closed -- see GoStartEventLoop for a version that starts in a
// separate goroutine and returns immediately.
func (w *Window) StartEventLoop() {
	w.DoFullRender = true
	w.EventLoop()
}

// GoStartEventLoop starts the event processing loop for this window in a new
// goroutine, and returns immediately.  Adds to WinWait waitgroup so a main
// thread can wait on that for all windows to close.
func (w *Window) GoStartEventLoop() {
	WinWait.Add(1)
	w.DoFullRender = true
	w.SetFlag(int(WinFlagGoLoop))
	go w.EventLoop()
}

// StopEventLoop tells the event loop to stop running when the next event arrives.
func (w *Window) StopEventLoop() {
	w.SetFlag(int(WinFlagStopEventLoop))
}

// ConnectEvent adds a Signal connection for given event type and
// prioritiy to given receiver
func (w *Window) ConnectEvent(recv ki.Ki, et oswin.EventType, pri EventPris, fun ki.RecvFunc) {
	if et >= oswin.EventTypeN {
		log.Printf("Window ConnectEvent type: %v is not a known event type\n", et)
		return
	}
	w.EventSigs[et][pri].Connect(recv, fun)
}

// DisconnectEvent removes Signal connection for given event type to given
// receiver -- pri is priority -- pass AllPris for all priorities
func (w *Window) DisconnectEvent(recv ki.Ki, et oswin.EventType, pri EventPris) {
	if et >= oswin.EventTypeN {
		log.Printf("Window DisconnectEvent type: %v is not a known event type\n", et)
		return
	}
	if pri == AllPris {
		for p := HiPri; p < EventPrisN; p++ {
			w.EventSigs[et][p].Disconnect(recv)
		}
	} else {
		w.EventSigs[et][pri].Disconnect(recv)
	}
}

// DisconnectAllEvents disconnect node from all event signals -- pri is
// priority -- pass AllPris for all priorities
func (w *Window) DisconnectAllEvents(recv ki.Ki, pri EventPris) {
	if pri == AllPris {
		for et := oswin.EventType(0); et < oswin.EventTypeN; et++ {
			for p := HiPri; p < EventPrisN; p++ {
				w.EventSigs[et][p].Disconnect(recv)
			}
		}
	} else {
		for et := oswin.EventType(0); et < oswin.EventTypeN; et++ {
			w.EventSigs[et][pri].Disconnect(recv)
		}
	}
}

// SendCustomEvent sends a custom event with given data to this window -- widgets can connect
// to receive CustomEventType events to receive them.  Sometimes it is useful
// to send a custom event just to trigger a pass through the event loop, even
// if nobody is listening (e.g., if a popup is posted without a surrounding
// event, as in Complete.ShowCompletions
func (w *Window) SendCustomEvent(data interface{}) {
	oswin.SendCustomEvent(w.OSWin, data)
}

/////////////////////////////////////////////////////////////////////////////
//                   Rendering

// FullReRender performs a full re-render of the window -- each node renders
// into its viewport, aggregating into the main window viewport, which will
// drive an UploadAllViewports call after all the rendering is done, and
// signal the publishing of the window after that
func (w *Window) FullReRender() {
	if w.IsClosed() || w.IsClosing() {
		return
	}
	w.Viewport.FullRender2DTree()
	w.FocusMu.Lock()
	if w.Focus == nil {
		if w.StartFocus != nil {
			sf := w.StartFocus
			w.StartFocus = nil
			w.FocusMu.Unlock()
			w.FocusOnOrNext(sf)
		} else {
			foc := w.Focus
			w.FocusMu.Unlock()
			w.FocusNext(foc)
		}
	} else {
		w.FocusMu.Unlock()
	}
}

// UploadVpRegion uploads image for one viewport region on the screen, using
// vpBBox bounding box for the viewport, and winBBox bounding box for the
// window -- called after re-rendering specific nodes to update only the
// relevant part of the overall viewport image
func (w *Window) UploadVpRegion(vp *Viewport2D, vpBBox, winBBox image.Rectangle) {
	if w.IsClosed() || w.IsClosing() {
		return
	}
	w.UpMu.Lock()
	if w.IsClosed() || w.IsClosing() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.SetWinUpdating()
	pr := prof.Start("win.UploadVpRegion")
	if Render2DTrace {
		fmt.Printf("Window: %v uploading region Vp %v, vpbbox: %v, wintex bounds: %v\n", w.PathUnique(), vp.PathUnique(), vpBBox, w.WinTex.Bounds())
	}
	w.WinTex.Upload(winBBox.Min, vp.OSImage, vpBBox)
	pr.End()
	w.ClearWinUpdating()
	w.UpMu.Unlock()
}

// UploadVp uploads entire viewport image for given viewport -- e.g., for
// popups etc updating separately
func (w *Window) UploadVp(vp *Viewport2D, offset image.Point) {
	if w.IsClosed() || w.IsClosing() {
		return
	}
	w.UpMu.Lock()
	if w.IsClosed() || w.IsClosing() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.SetWinUpdating()
	updt := w.UpdateStart()
	pr := prof.Start("win.UploadVp")
	if Render2DTrace {
		fmt.Printf("Window: %v uploading Vp %v, image bound: %v, wintex bounds: %v\n", w.PathUnique(), vp.PathUnique(), vp.OSImage.Bounds(), w.WinTex.Bounds())
	}
	w.WinTex.Upload(offset, vp.OSImage, vp.OSImage.Bounds())
	pr.End()
	w.ClearWinUpdating()
	w.UpMu.Unlock()
	w.UpdateEnd(updt) // drives publish
}

// UploadAllViewports does a complete upload of all active viewports, in the
// proper order, so as to completely refresh the window texture based on
// everything rendered
func (w *Window) UploadAllViewports() {
	if w.IsClosed() || w.IsClosing() {
		return
	}
	w.UpMu.Lock()
	if w.IsClosed() || w.IsClosing() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.SetWinUpdating()
	pr := prof.Start("win.UploadAllViewports")
	updt := w.UpdateStart()
	if Render2DTrace {
		fmt.Printf("Window: %v uploading full Vp, image bound: %v, wintex bounds: %v\n", w.PathUnique(), w.Viewport.OSImage.Bounds(), w.WinTex.Bounds())
	}
	w.WinTex.Upload(image.ZP, w.Viewport.OSImage, w.Viewport.OSImage.Bounds())
	// then all the current popups
	w.PopMu.RLock()
	// fmt.Printf("upload all views pop locked: %v\n", w.Nm)
	if w.PopupStack != nil {
		for _, pop := range w.PopupStack {
			gii, _ := KiToNode2D(pop)
			if gii != nil {
				vp := gii.AsViewport2D()
				r := vp.Geom.Bounds()
				if Render2DTrace {
					fmt.Printf("Window: %v uploading popup stack Vp %v, image bound: %v, wintex bounds: %v\n", w.PathUnique(), vp.PathUnique(), r.Min, vp.OSImage.Bounds())
				}
				w.WinTex.Upload(r.Min, vp.OSImage, vp.OSImage.Bounds())
			}
		}
	}
	if w.Popup != nil {
		gii, _ := KiToNode2D(w.Popup)
		if gii != nil {
			vp := gii.AsViewport2D()
			r := vp.Geom.Bounds()
			if Render2DTrace {
				fmt.Printf("Window: %v uploading top popup Vp %v, image bound: %v, wintex bounds: %v\n", w.PathUnique(), vp.PathUnique(), r.Min, vp.OSImage.Bounds())
			}
			w.WinTex.Upload(r.Min, vp.OSImage, vp.OSImage.Bounds())
		}
	}
	w.PopMu.RUnlock()
	// fmt.Printf("upload all views pop unlocked: %v\n", w.Nm)
	pr.End()
	w.ClearWinUpdating()
	w.UpMu.Unlock()   // need to unlock before publish
	w.UpdateEnd(updt) // drives the publish
}

// IsWinUpdating checks if we are already updating window
func (w *Window) IsWinUpdating() bool {
	return w.HasFlag(int(WinFlagUpdating))
}

// SetWinUpdating sets the window updating state to true if not already updating
func (w *Window) SetWinUpdating() {
	w.SetFlag(int(WinFlagUpdating))
}

// ClearWinUpdating sets the window updating state to false if not already updating
func (w *Window) ClearWinUpdating() {
	w.ClearFlag(int(WinFlagUpdating))
}

// Publish does the final step of updating of the window based on the current
// texture (and overlay texture if active)
func (w *Window) Publish() {
	if w.IsClosed() || w.OSWin.IsMinimized() {
		// fmt.Printf("skipping update on inactive / minimized window: %v\n", w.Nm)
		return
	}
	// actually, cannot prevent publishing -- will lockup!
	// if !w.TryPublishing() {
	// 	return
	// }
	w.UpMu.Lock()     // block all updates while we publish
	if w.IsClosed() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}

	// note: this is key for finding redundant updates!
	if WinPublishTrace {
		fmt.Printf("\n\n###################################\n%v\n", string(debug.Stack()))
	}

	w.SetWinUpdating()
	// fmt.Printf("Win %v doing publish\n", w.Nm)
	pr := prof.Start("win.Publish.Copy")
	w.OSWin.Copy(image.ZP, w.WinTex, w.WinTex.Bounds(), oswin.Src, nil)
	if w.OverTex != nil && w.HasFlag(int(WinFlagOverTexActive)) {
		w.OSWin.Copy(image.ZP, w.OverTex, w.OverTex.Bounds(), oswin.Over, nil)
	}
	pr.End()
	pr2 := prof.Start("win.Publish.Publish")
	w.OSWin.Publish()
	pr2.End()
	w.ClearWinUpdating()
	w.UpMu.Unlock()
}

// SignalWindowPublish is the signal receiver function that publishes the
// window updates when the window update signal (UpdateEnd) occurs
func SignalWindowPublish(winki, node ki.Ki, sig int64, data interface{}) {
	win := winki.Embed(KiT_Window).(*Window)
	if Render2DTrace {
		fmt.Printf("Window: %v publishing image due to signal: %v from node: %v\n", win.PathUnique(), ki.NodeSignals(sig), node.PathUnique())
	}
	if win.IsClosed() || win.IsClosing() || win.IsResizing() || win.IsWinUpdating() {
		return
	}
	win.Publish()
}

/////////////////////////////////////////////////////////////////////////////
//                   Overlays and Sprites

// RenderOverlays renders overlays and sprites -- clears overlay viewport to
// transparent, renders all overlays, uploads result to OverTex
func (w *Window) RenderOverlays() {
	w.UpMu.Lock()
	if w.OverlayVp == nil || !w.OverlayVp.HasChildren() && w.ActiveSprites == 0 {
		w.ClearFlag(int(WinFlagOverTexActive))
		w.UpMu.Unlock()
		return
	}
	if w.IsClosed() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	updt := w.UpdateStart()
	wsz := w.WinTex.Bounds().Size()
	if w.OverTex == nil || w.OverTex.Bounds() != w.WinTex.Bounds() {
		if w.OverTex != nil {
			w.OverTex.Release()
		}
		w.OverTex, _ = oswin.TheApp.NewTexture(w.OSWin, wsz)
	}
	w.OverlayVp.Win = w
	w.OverlayVp.RenderOverlays(wsz) // handles any resizing etc
	if len(w.OverlayVp.Kids) == 0 {
		if !w.HasFlag(int(WinFlagOverlayVpCleared)) {
			vp := w.OverlayVp
			draw.Draw(vp.Pixels, vp.Pixels.Bounds(), &image.Uniform{color.Transparent}, image.ZP, draw.Src)
			w.SetFlag(int(WinFlagOverlayVpCleared))
		} else {
			w.ClearFlag(int(WinFlagOverlayVpCleared))
		}
	}
	w.OverTex.Upload(image.ZP, w.OverlayVp.OSImage, w.OverlayVp.OSImage.Bounds())

	if w.ActiveSprites > 0 {
		for _, sp := range w.Sprites {
			if sp.IsInactive() {
				continue
			}
			w.RenderSprite(sp)
		}
	}
	w.SetFlag(int(WinFlagOverTexActive))
	w.UpMu.Unlock()
	w.UpdateEnd(updt) // drives the publish
}

// AddSprite adds a new sprite viewport with given name (which must remain
// invariant and unique among all sprites in use, and is used for all access
// -- prefix with package and type name to ensure uniqueness.  Starts out in
// inactive state -- must call ActivateSprite.
func (w *Window) AddSprite(nm string, sz image.Point, pos image.Point) *Viewport2D {
	w.UpMu.Lock()
	defer w.UpMu.Unlock()

	if w.Sprites == nil {
		w.Sprites = make(map[string]*Viewport2D)
		w.SpritesBg = make(map[string]oswin.Image)
	}
	if _, has := w.Sprites[nm]; has {
		log.Printf("gi.Window AddSprite -- name is already in use: %v\n", nm)
		return nil
	}
	sp := &Viewport2D{}
	sp.InitName(sp, nm)
	sp.Win = w
	sp.Resize(sz)
	sp.Geom.Pos = pos
	sp.SetAsOverlay()
	sp.SetInactive() // sprites start inactive
	w.Sprites[nm] = sp
	return sp
}

// ActivateSprite clears the Inactive flag on the sprite, and increments
// ActiveSprites, so that it will actualy be rendered
func (w *Window) ActivateSprite(nm string) {
	w.UpMu.Lock()
	defer w.UpMu.Unlock()

	sp, ok := w.Sprites[nm]
	if !ok {
		return // not worth bothering about errs -- use a consistent string var!
	}
	if sp.IsInactive() {
		sp.SetActiveState(true)
		w.ActiveSprites++
	}
}

// InactivateSprite sets the Inactive flag on the sprite, and decrements
// ActiveSprites, so that it will not be rendered
func (w *Window) InactivateSprite(nm string) {
	w.UpMu.Lock()
	defer w.UpMu.Unlock()

	sp, ok := w.Sprites[nm]
	if !ok {
		return // not worth bothering about errs -- use a consistent string var!
	}
	if sp.IsActive() {
		sp.SetInactive()
		w.ActiveSprites--
	}
}

// InactivateAllSprites inactivates all sprites
func (w *Window) InactivateAllSprites() {
	w.UpMu.Lock()
	defer w.UpMu.Unlock()

	for _, sp := range w.Sprites {
		if sp.IsActive() {
			sp.SetInactive()
			w.ActiveSprites--
		}
	}
}

// RenderSprite renders the sprite onto OverTex -- must be called within UpMu mutex lock
func (w *Window) RenderSprite(sp *Viewport2D) {
	sp.Render2D()
	bg, ok := w.SpritesBg[sp.Nm]
	if !ok {
		bg, _ = oswin.TheApp.NewImage(sp.Geom.Size)
		w.SpritesBg[sp.Nm] = bg
	} else if bg.Size() != sp.Geom.Size {
		bg.Release()
		bg, _ = oswin.TheApp.NewImage(sp.Geom.Size)
		w.SpritesBg[sp.Nm] = bg
	}
	bgi := bg.RGBA()
	// grab source from viewport
	draw.Draw(bgi, bgi.Bounds(), w.Viewport.Pixels, sp.Geom.Pos, draw.Src)
	// draw sprite over
	draw.Draw(bgi, bgi.Bounds(), sp.Pixels, image.ZP, draw.Over)
	// note: already under RenderOverlays mutex protection
	w.OverTex.Upload(sp.Geom.Pos, bg, bg.Bounds())
}

/////////////////////////////////////////////////////////////////////////////
//                   MainMenu Updating

// MainMenuUpdated needs to be called whenever the main menu for this window
// is updated in terms of items added or removed.
func (w *Window) MainMenuUpdated() {
	if w == nil || w.MainMenu == nil || w.IsClosed() || w.IsClosing() {
		return
	}
	w.UpMu.Lock()
	if w.IsClosed() || w.IsClosing() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.MainMenu.UpdateMainMenu(w) // main update menu call, in bars.go for MenuBar
	w.UpMu.Unlock()
}

// MainMenuSet sets the main menu for the window, after window.Focus event
func (w *Window) MainMenuSet() {
	if w == nil || w.MainMenu == nil || w.IsClosed() || w.IsClosing() {
		return
	}
	w.UpMu.Lock()
	if w.IsClosed() || w.IsClosing() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.MainMenu.SetMainMenu(w) // set main menu call, in bars.go for MenuBar
	w.UpMu.Unlock()
}

// MainMenuUpdateActives needs to be called whenever items on the main menu
// for this window have their IsActive status updated.
func (w *Window) MainMenuUpdateActives() {
	if w == nil || w.MainMenu == nil || w.IsClosed() || w.IsClosing() {
		return
	}
	w.UpMu.Lock()
	if w.IsClosed() || w.IsClosing() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.MainMenu.MainMenuUpdateActives(w) // also in bars.go for MenuBar
	w.UpMu.Unlock()
}

// MainMenuUpdateWindows updates a Window menu with a list of active menus.
func (w *Window) MainMenuUpdateWindows() {
	if w == nil || w.MainMenu == nil || w.IsClosed() || w.IsClosing() {
		return
	}
	w.UpMu.Lock()
	if w.IsClosed() || w.IsClosing() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	WindowGlobalMu.Lock()
	wmeni, ok := w.MainMenu.ChildByName("Window", 3)
	if !ok {
		WindowGlobalMu.Unlock()
		w.UpMu.Unlock()
		return
	}
	wmen := wmeni.(*Action)
	men := make(Menu, 0, len(AllWindows))
	men.AddWindowsMenu(w)
	wmen.Menu = men
	WindowGlobalMu.Unlock()
	w.UpMu.Unlock()
	w.MainMenuUpdated()
}

/////////////////////////////////////////////////////////////////////////////
//                   Main Method: EventLoop

// EventLoop runs the event processing loop for the Window -- grabs oswin
// events for the window and dispatches them to receiving nodes, and manages
// other state etc (popups, etc).
func (w *Window) EventLoop() {
	var skippedResize *window.Event

	lastEt := oswin.EventTypeN
	var skipDelta image.Point
	lastSkipped := false
	// lastKeyChord := key.Chord("")

	var startDrag *mouse.DragEvent
	dragStarted := false

	var startDND *mouse.DragEvent
	dndStarted := false

	var startHover, curHover *mouse.MoveEvent
	hoverStarted := false
	var hoverTimer *time.Timer

	var startDNDHover, curDNDHover *mouse.DragEvent
	dndHoverStarted := false
	var dndHoverTimer *time.Timer

mainloop:
	for {
		evi := w.OSWin.NextEvent()
		if w.HasFlag(int(WinFlagStopEventLoop)) {
			w.ClearFlag(int(WinFlagStopEventLoop))
			fmt.Println("stop event loop")
			break
		}
		et := evi.Type()
		delPop := false                      // if true, delete this popup after event loop
		if et > oswin.EventTypeN || et < 0 { // we don't handle other types of events here
			fmt.Printf("Win: %v got out-of-range event: %v\n", w.Nm, et)
			continue
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
					fmt.Printf("zombie popup: %v  cur: %v\n", dpop.Name(), cpop.Name())
				}
			}
		}

		////////////////////////////////////////////////////////////////////////////
		// Filter repeated laggy events -- key for responsive resize, scroll, etc

		now := time.Now()
		lag := now.Sub(evi.Time())
		lagMs := int(lag / time.Millisecond)
		if WinEventTrace {
			if et != oswin.MouseMoveEvent {
				fmt.Printf("Win: %v Event: %v  Lag: %v\n", w.Nm, evi.String(), lag)
			}
		}

		if et != oswin.KeyEvent {
			if w.HasFlag(int(WinFlagGotPaint)) && et == oswin.WindowPaintEvent && lastEt == oswin.WindowResizeEvent {
				if WinEventTrace {
					fmt.Printf("Win: %v skipping paint after resize\n", w.Nm)
				}
				w.Publish() // this is essential on mac for any paint event
				w.SetFlag(int(WinFlagGotPaint))
				continue // X11 always sends a paint after a resize -- we just use resize
			}
			if et == lastEt || lastEt == oswin.WindowResizeEvent {
				switch et {
				case oswin.MouseScrollEvent:
					me := evi.(*mouse.ScrollEvent)
					if lagMs > EventSkipLagMSec {
						// fmt.Printf("skipped et %v lag %v\n", et, lag)
						if !lastSkipped {
							skipDelta = me.Delta
						} else {
							skipDelta = skipDelta.Add(me.Delta)
						}
						lastSkipped = true
						continue
					} else {
						if lastSkipped {
							me.Delta = skipDelta.Add(me.Delta)
						}
						lastSkipped = false
					}
				case oswin.MouseDragEvent:
					me := evi.(*mouse.DragEvent)
					if lagMs > EventSkipLagMSec {
						// fmt.Printf("skipped et %v lag %v\n", et, lag)
						if !lastSkipped {
							skipDelta = me.From
						}
						lastSkipped = true
						continue
					} else {
						if lastSkipped {
							me.From = skipDelta
						}
						lastSkipped = false
					}
				case oswin.WindowResizeEvent:
					w.SetFlag(int(WinFlagIsResizing))
					we := evi.(*window.Event)
					// fmt.Printf("resize\n")
					if lagMs > EventSkipLagMSec {
						if WinEventTrace {
							fmt.Printf("Win: %v skipped et %v lag %v size: %v\n", w.Nm, et, lag, w.OSWin.Size())
						}
						lastSkipped = true
						skippedResize = we
						continue
					} else {
						we.SetProcessed()
						w.Resized(w.OSWin.Size())
						// w.DoFullRender = true
						lastSkipped = false
						skippedResize = nil
						continue
					}
					//				case oswin.KeyChordEvent:
					//					ke := evi.(*key.ChordEvent)
					//					ks := ke.Chord()
					//					if ks == lastKeyChord && lagMs > EventSkipLagMSec {
					//						fmt.Printf("skipped %v key: %v lag %v\n", et, ks, lag)
					//						lastSkipped = true
					//						continue
					//					} else {
					//						lastKeyChord = ks
					//						lastSkipped = false
					//					}
				}
			}
			lastSkipped = false
			lastEt = et
		}

		if skippedResize != nil || w.Viewport.Geom.Size != w.OSWin.Size() {
			w.SetFlag(int(WinFlagIsResizing))
			w.Resized(w.OSWin.Size())
			skippedResize = nil
		}

		if et != oswin.WindowResizeEvent && et != oswin.WindowPaintEvent {
			w.ClearFlag(int(WinFlagIsResizing))
		}

		////////////////////////////////////////////////////////////////////////////
		// Detect start of drag and DND -- both require delays in starting due
		// to minor wiggles when pressing the mouse button

		if et == oswin.MouseDragEvent {
			if !dragStarted {
				if startDrag == nil {
					startDrag = evi.(*mouse.DragEvent)
				} else {
					if w.DoInstaDrag(startDrag, !w.CurPopupIsTooltip()) {
						dragStarted = true
						startDrag = nil
					} else {
						delayMs := int(now.Sub(startDrag.Time()) / time.Millisecond)
						if delayMs >= DragStartMSec {
							dst := int(math32.Hypot(float32(startDrag.Where.X-evi.Pos().X), float32(startDrag.Where.Y-evi.Pos().Y)))
							if dst >= DragStartPix {
								dragStarted = true
								startDrag = nil
							}
						}
					}
				}
			}
			if w.Dragging == nil && !dndStarted {
				if startDND == nil {
					startDND = evi.(*mouse.DragEvent)
				} else {
					delayMs := int(now.Sub(startDND.Time()) / time.Millisecond)
					if delayMs >= DNDStartMSec {
						dst := int(math32.Hypot(float32(startDND.Where.X-evi.Pos().X), float32(startDND.Where.Y-evi.Pos().Y)))
						if dst >= DNDStartPix {
							dndStarted = true
							w.DNDStartEvent(startDND)
							startDND = nil
						}
					}
				}
			} else { // dndStarted
				w.TimerMu.Lock()
				if !dndHoverStarted {
					dndHoverStarted = true
					startDNDHover = evi.(*mouse.DragEvent)
					curDNDHover = startDNDHover
					dndHoverTimer = time.AfterFunc(time.Duration(HoverStartMSec)*time.Millisecond, func() {
						w.TimerMu.Lock()
						hoe := curDNDHover
						dndHoverStarted = false
						startDNDHover = nil
						curDNDHover = nil
						dndHoverTimer = nil
						w.TimerMu.Unlock()
						w.SendDNDHoverEvent(hoe)
					})
				} else {
					dst := int(math32.Hypot(float32(startDNDHover.Where.X-evi.Pos().X), float32(startDNDHover.Where.Y-evi.Pos().Y)))
					if dst > HoverMaxPix {
						dndHoverTimer.Stop()
						dndHoverStarted = false
						startDNDHover = nil
						dndHoverTimer = nil
					} else {
						curDNDHover = evi.(*mouse.DragEvent)
					}
				}
				w.TimerMu.Unlock()
			}
		} else {
			if et != oswin.KeyEvent { // allow modifier keypress
				dragStarted = false
				startDrag = nil
				dndStarted = false
				startDND = nil

				w.TimerMu.Lock()
				dndHoverStarted = false
				startDNDHover = nil
				curDNDHover = nil
				if dndHoverTimer != nil {
					dndHoverTimer.Stop()
					dndHoverTimer = nil
				}
				w.TimerMu.Unlock()
			}
		}

		////////////////////////////////////////////////////////////////////////////
		// Detect hover event -- requires delay timing

		if et == oswin.MouseMoveEvent {
			w.TimerMu.Lock()
			if !hoverStarted {
				hoverStarted = true
				startHover = evi.(*mouse.MoveEvent)
				curHover = startHover
				hoverTimer = time.AfterFunc(time.Duration(HoverStartMSec)*time.Millisecond, func() {
					w.TimerMu.Lock()
					hoe := curHover
					hoverStarted = false
					startHover = nil
					curHover = nil
					hoverTimer = nil
					w.TimerMu.Unlock()
					w.SendHoverEvent(hoe)
				})
			} else {
				dst := int(math32.Hypot(float32(startHover.Where.X-evi.Pos().X), float32(startHover.Where.Y-evi.Pos().Y)))
				if dst > HoverMaxPix {
					hoverTimer.Stop()
					hoverStarted = false
					startHover = nil
					hoverTimer = nil
					w.PopMu.RLock()
					if w.CurPopupIsTooltip() {
						delPop = true
					}
					w.PopMu.RUnlock()
				} else {
					curHover = evi.(*mouse.MoveEvent)
				}
			}
			w.TimerMu.Unlock()
		} else {
			w.TimerMu.Lock()
			hoverStarted = false
			startHover = nil
			curHover = nil
			if hoverTimer != nil {
				hoverTimer.Stop()
				hoverTimer = nil
			}
			w.TimerMu.Unlock()
		}

		////////////////////////////////////////////////////////////////////////////
		//  High-priority events for Window
		//  Window gets first crack at these events, and handles window-specific ones

		switch e := evi.(type) {
		case *window.Event:
			switch e.Action {
			// case window.Resize: // note: already handled earlier in lag process
			case window.Close:
				// fmt.Printf("got close event for window %v \n", w.Nm)
				w.Closed()
				break mainloop
			case window.Paint:
				// fmt.Printf("got paint event for window %v \n", w.Nm)
				w.SetFlag(int(WinFlagGotPaint))
				if w.DoFullRender {
					w.DoFullRender = false
					// fmt.Printf("Doing full render at size: %v\n", w.Viewport.Geom.Size)
					if w.Viewport.Geom.Size != w.OSWin.Size() {
						w.Resized(w.OSWin.Size())
					} else {
						w.FullReRender()
					}
				}
				w.Publish()
			case window.Move:
				e.SetProcessed()
				if w.HasFlag(int(WinFlagGotPaint)) { // moves before paint are not accurate on X11
					// fmt.Printf("win move: %v\n", w.OSWin.Position())
					WinGeomPrefs.RecordPref(w)
				}
			case window.Focus:
				if !w.HasFlag(int(WinFlagGotFocus)) {
					w.SetFlag(int(WinFlagGotFocus))
					// fmt.Printf("win foc: %v\n", w.Nm)
				} else {
					// fmt.Printf("win extra foc: %v\n", w.Nm)
					if w.NeedWinMenuUpdate() {
						w.MainMenuUpdateWindows()
					}
					w.MainMenuSet()
				}
			case window.DeFocus:
				// fmt.Printf("win de-foc: %v\n", w.Nm)
				w.ClearFlag(int(WinFlagGotFocus))
			}
			continue // don't do anything else!
		case *mouse.DragEvent:
			w.LastModBits = e.Modifiers
			w.FocusMu.Lock()
			if w.Focus == nil && w.StartFocus != nil { // why is this here in drag event???
				sf := w.StartFocus
				w.StartFocus = nil
				w.FocusMu.Unlock()
				w.FocusOnOrNext(sf)
			} else {
				w.FocusMu.Unlock()
			}
			w.LastSelMode = e.SelectMode()
			if w.DNDData != nil {
				w.DNDMoveEvent(e)
			} else {
				if !dragStarted {
					e.SetProcessed() // ignore
				}
			}
		case *mouse.Event:
			w.LastModBits = e.Modifiers
			w.LastSelMode = e.SelectMode()
			if w.DNDData != nil && e.Action == mouse.Release {
				w.DNDDropEvent(e)
			}
			w.FocusActiveClick(e)
		case *mouse.MoveEvent:
			w.LastModBits = e.Modifiers
			w.LastSelMode = e.SelectMode()
			if bitflag.HasAllAtomic(&w.Flag, int(WinFlagGotPaint), int(WinFlagGotFocus)) {
				if w.DoFullRender {
					// if we are getting mouse input, and still haven't done this, do it..
					w.DoFullRender = false
					// fmt.Printf("Doing full render at size: %v\n", w.Viewport.Geom.Size)
					if w.Viewport.Geom.Size != w.OSWin.Size() {
						w.Resized(w.OSWin.Size())
					} else {
						w.FullReRender()
					}
				}
				if w.NeedWinMenuUpdate() {
					// fmt.Printf("win menu updt: %v\n", w.Nm)
					w.MainMenuUpdateWindows()
					w.MainMenuSet()
				}
				w.FocusMu.Lock()
				if w.Focus == nil && w.StartFocus != nil {
					sf := w.StartFocus
					w.StartFocus = nil
					w.FocusMu.Unlock()
					w.FocusOnOrNext(sf)
				} else {
					w.FocusMu.Unlock()
				}
			}
		case *key.ChordEvent:
			keyDelPop := w.KeyChordEventHiPri(e)
			if keyDelPop {
				delPop = true
			}
		}

		////////////////////////////////////////////////////////////////////////////
		// Send Events to Widgets

		if !evi.IsProcessed() && w.HasFlag(int(WinFlagGotFocus)) {
			evToPopup := !w.CurPopupIsTooltip() // don't send events to tooltips!
			w.SendEventSignal(evi, evToPopup)
			if !delPop && et == oswin.MouseMoveEvent {
				didFocus := w.GenMouseFocusEvents(evi.(*mouse.MoveEvent), evToPopup)
				if didFocus && w.CurPopupIsTooltip() {
					delPop = true
				}
			}
		}

		////////////////////////////////////////////////////////////////////////////
		// Low priority windows events

		if !evi.IsProcessed() {
			switch e := evi.(type) {
			case *key.ChordEvent:
				keyDelPop := w.KeyChordEventLowPri(e)
				if keyDelPop {
					delPop = true
				}
			}
		}

		// reset "catch" events (Dragging, Scrolling)
		if w.Dragging != nil && et != oswin.MouseDragEvent {
			w.Dragging.ClearFlag(int(NodeDragging))
			w.Dragging = nil
		}
		if w.Scrolling != nil && et != oswin.MouseScrollEvent {
			w.Scrolling = nil
		}

		////////////////////////////////////////////////////////////////////////////
		// Delete popup?

		{
			cpop := w.CurPopup()
			if cpop != nil && !delPop {
				if PopupIsTooltip(cpop) {
					if et != oswin.MouseMoveEvent {
						delPop = true
					}
				} else if me, ok := evi.(*mouse.Event); ok {
					if me.Action == mouse.Release {
						if w.ShouldDeletePopupMenu(cpop, me) {
							delPop = true
						}
					}
				}

				if PopupIsCompleter(cpop) {
					fsz := len(w.FocusStack)
					if fsz > 0 && et == oswin.KeyChordEvent {
						for pri := HiPri; pri < EventPrisN; pri++ {
							w.EventSigs[et][pri].SendSig(w.FocusStack[fsz-1], cpop, int64(et), evi)
						}
					}
				}
			}
		}

		////////////////////////////////////////////////////////////////////////////
		// Shortcuts come last as lowest priority relative to more specific cases

		if !evi.IsProcessed() && et == oswin.KeyChordEvent {
			ke := evi.(*key.ChordEvent)
			kc := ke.Chord()
			w.TriggerShortcut(kc)
		}

		////////////////////////////////////////////////////////////////////////////
		// Actually delete popup and push a new one

		if delPop {
			w.ClosePopup(w.CurPopup())
		}

		w.PopMu.RLock()
		npop := w.NextPopup
		w.PopMu.RUnlock()
		if npop != nil {
			w.PushPopup(npop)
		}
	}
	if WinEventTrace {
		fmt.Printf("Win: %v out of event loop\n", w.Nm)
	}
	if w.HasFlag(int(WinFlagGoLoop)) {
		WinWait.Done()
	}
	// our last act must be self destruction!
	w.Destroy()
}

/////////////////////////////////////////////////////////////////////////////
//                   Sending Events

// IsInScope returns true if the given object is in scope for receiving events.
// If popup is true, then only items on popup are in scope, otherwise
// items NOT on popup are in scope (if no popup, everything is in scope).
func (w *Window) IsInScope(ni *Node2DBase, popup bool) bool {
	cpop := w.CurPopup()
	if cpop == nil {
		return true
	}
	if ni.This() == cpop {
		return popup
	}
	if ni.Viewport == nil {
		return false
	}
	if ni.Viewport.This() == cpop {
		return popup
	}
	return !popup
}

// WinEventRecv is used to hold info about widgets receiving event signals to
// given function, used for sorting and delayed sending.
type WinEventRecv struct {
	Recv ki.Ki
	Func ki.RecvFunc
	Data int
}

// Set sets the recv and fun
func (we *WinEventRecv) Set(r ki.Ki, f ki.RecvFunc, data int) {
	we.Recv = r
	we.Func = f
	we.Data = data
}

// Call calls the function on the recv with the args
func (we *WinEventRecv) Call(send ki.Ki, sig int64, data interface{}) {
	we.Func(we.Recv, send, sig, data)
}

type WinEventRecvList []WinEventRecv

func (wl *WinEventRecvList) Add(recv ki.Ki, fun ki.RecvFunc, data int) {
	rr := WinEventRecv{recv, fun, data}
	*wl = append(*wl, rr)
}

func (wl *WinEventRecvList) AddDepth(recv ki.Ki, fun ki.RecvFunc, w *Window) {
	wl.Add(recv, fun, recv.ParentLevel(w.This()))
}

// SendEventSignalFunc is the inner loop of the SendEventSignal -- needed to deal with
// map iterator locking logic in a cleaner way.  Returns true to continue, false to break
func (w *Window) SendEventSignalFunc(evi oswin.Event, popup bool, rvs *WinEventRecvList, recv ki.Ki, fun ki.RecvFunc) bool {
	nii, ni := KiToNode2D(recv)
	if ni != nil {
		if !w.IsInScope(ni, popup) {
			return true
		}
		if evi.OnFocus() {
			if !nii.HasFocus2D() {
				return true
			}
			w.FocusMu.Lock()
			if !w.FocusActive { // reactivate on keyboard input
				w.FocusActive = true
				// fmt.Printf("set foc active: %v\n", ni.PathUnique())
				nii.FocusChanged2D(FocusActive)
			}
			w.FocusMu.Unlock()
		} else if evi.HasPos() {
			pos := evi.Pos()
			switch evi.(type) {
			case *mouse.DragEvent:
				if w.Dragging != nil {
					if w.Dragging == ni.This() {
						rvs.Add(recv, fun, 10000)
						return false
					} else {
						return true
					}
				} else {
					if pos.In(ni.WinBBox) {
						rvs.AddDepth(recv, fun, w)
						return false
					}
					return true
				}
			case *mouse.ScrollEvent:
				if w.Scrolling != nil {
					if w.Scrolling == ni.This() {
						rvs.Add(recv, fun, 10000)
					} else {
						return true
					}
				} else {
					if pos.In(ni.WinBBox) {
						rvs.AddDepth(recv, fun, w)
						return false
					}
					return true
				}
			default:
				if w.Dragging == ni.This() { // dragger always gets it
					rvs.Add(recv, fun, 10000) // top priority -- can't steal!
					return false
				}
				if !pos.In(ni.WinBBox) {
					return true
				}
			}
		}
		rvs.AddDepth(recv, fun, w)
		return true
	} else {
		// todo: get a 3D
		return true
	}
}

// SendEventSignal sends given event signal to all receivers that want it --
// note that because there is a different EventSig for each event type, we are
// ONLY looking at nodes that have registered to receive that type of event --
// the further filtering is just to ensure that they are in the right position
// to receive the event (focus, popup filtering, etc).  If popup is true, then
// only items on popup are in scope, otherwise items NOT on popup are in scope
// (if no popup, everything is in scope).
func (w *Window) SendEventSignal(evi oswin.Event, popup bool) {
	et := evi.Type()
	if et > oswin.EventTypeN || et < 0 {
		return // can't handle other types of events here due to EventSigs[et] size
	}

	w.EventMu.Lock()
	defer w.EventMu.Unlock()

	// fmt.Printf("got event type: %v\n", et)
	for pri := HiPri; pri < EventPrisN; pri++ {
		if pri != LowRawPri && evi.IsProcessed() { // someone took care of it
			continue
		}

		// we take control of signal process to sort elements by depth, and
		// dispatch to inner-most one first
		rvs := make(WinEventRecvList, 0, 10)

		esig := &w.EventSigs[et][pri]

		esig.Mu.RLock()
		for recv, fun := range esig.Cons {
			if recv.IsDestroyed() {
				// fmt.Printf("ki.Signal deleting destroyed receiver: %v type %T\n", recv.Name(), recv)
				delete(esig.Cons, recv)
				continue
			}
			if recv.IsDeleted() {
				continue
			}
			esig.Mu.RUnlock()
			cont := w.SendEventSignalFunc(evi, popup, &rvs, recv, fun)
			esig.Mu.RLock()
			if !cont {
				break
			}
		}
		esig.Mu.RUnlock()

		if len(rvs) == 0 {
			continue
		}

		// deepest first
		sort.Slice(rvs, func(i, j int) bool {
			return rvs[i].Data > rvs[j].Data
		})

		for _, rr := range rvs {
			switch evi.(type) {
			case *mouse.DragEvent:
				if w.Dragging == nil {
					rr.Recv.SetFlag(int(NodeDragging)) // PROVISIONAL!
				}
			}
			rr.Call(w.This(), int64(et), evi)
			if pri != LowRawPri && evi.IsProcessed() { // someone took care of it
				switch evi.(type) { // only grab events if processed
				case *mouse.DragEvent:
					if w.Dragging == nil {
						w.Dragging = rr.Recv
						rr.Recv.SetFlag(int(NodeDragging))
					}
				case *mouse.ScrollEvent:
					if w.Scrolling == nil {
						w.Scrolling = rr.Recv
					}
				}
				break
			} else {
				switch evi.(type) {
				case *mouse.DragEvent:
					if w.Dragging == nil {
						rr.Recv.ClearFlag(int(NodeDragging)) // clear provisional
					}
				}
			}
		}
	}
}

// GenMouseFocusEvents processes mouse.MoveEvent to generate mouse.FocusEvent
// events -- returns true if any such events were sent.  If popup is true,
// then only items on popup are in scope, otherwise items NOT on popup are in
// scope (if no popup, everything is in scope).
func (w *Window) GenMouseFocusEvents(mev *mouse.MoveEvent, popup bool) bool {
	fe := mouse.FocusEvent{Event: mev.Event}
	pos := mev.Pos()
	ftyp := oswin.MouseFocusEvent
	updated := false
	updt := false
	for pri := HiPri; pri < EventPrisN; pri++ {
		w.EventSigs[ftyp][pri].EmitFiltered(w.This(), int64(ftyp), &fe, func(k ki.Ki) bool {
			if k.IsDeleted() { // destroyed is filtered upstream
				return false
			}
			_, ni := KiToNode2D(k)
			if ni != nil {
				if !w.IsInScope(ni, popup) {
					return false
				}
				in := pos.In(ni.WinBBox)
				if in {
					if !ni.HasFlag(int(MouseHasEntered)) {
						fe.Action = mouse.Enter
						ni.SetFlag(int(MouseHasEntered))
						if !updated {
							updt = w.UpdateStart()
							updated = true
						}
						return true // send event
					} else {
						return false // already in
					}
				} else { // mouse not in object
					if ni.HasFlag(int(MouseHasEntered)) {
						fe.Action = mouse.Exit
						ni.ClearFlag(int(MouseHasEntered))
						if !updated {
							updt = w.UpdateStart()
							updated = true
						}
						return true // send event
					} else {
						return false // already out
					}
				}
			} else {
				// todo: 3D
				return false
			}
		})
	}
	if updated {
		w.UpdateEnd(updt)
	}
	return updated
}

// DoInstaDrag tests whether the given mouse DragEvent is on a widget marked
// with InstaDrag
func (w *Window) DoInstaDrag(me *mouse.DragEvent, popup bool) bool {
	et := me.Type()
	for pri := HiPri; pri < EventPrisN; pri++ {
		esig := w.EventSigs[et][pri]
		for recv, _ := range esig.Cons {
			if recv.IsDestroyed() {
				delete(esig.Cons, recv)
				continue
			}
			if recv.IsDeleted() {
				continue
			}
			_, ni := KiToNode2D(recv)
			if ni != nil {
				if !w.IsInScope(ni, popup) {
					continue
				}
				pos := me.Pos()
				if pos.In(ni.WinBBox) {
					if ni.IsInstaDrag() {
						w.Dragging = ni.This()
						ni.SetFlag(int(NodeDragging))
						return true
					}
				}
			}
		}
	}
	return false
}

// SendHoverEvent sends mouse hover event, based on last mouse move event
func (w *Window) SendHoverEvent(e *mouse.MoveEvent) {
	he := mouse.HoverEvent{Event: e.Event}
	he.Processed = false
	he.Action = mouse.Hover
	w.SendEventSignal(&he, true) // popup = true by default
}

// SendKeyChordEvent sends a KeyChord event with given values.  If popup is
// true, then only items on popup are in scope, otherwise items NOT on popup
// are in scope (if no popup, everything is in scope).
func (w *Window) SendKeyChordEvent(popup bool, r rune, mods ...key.Modifiers) {
	ke := key.ChordEvent{}
	ke.SetTime()
	ke.SetModifiers(mods...)
	ke.Rune = r
	ke.Action = key.Press
	w.SendEventSignal(&ke, popup)
}

// SendKeyFunEvent sends a KeyChord event with params from the given KeyFun.
// If popup is true, then only items on popup are in scope, otherwise items
// NOT on popup are in scope (if no popup, everything is in scope).
func (w *Window) SendKeyFunEvent(kf KeyFuns, popup bool) {
	chord := ActiveKeyMap.ChordForFun(kf)
	if chord == "" {
		return
	}
	r, mods, err := chord.Decode()
	if err != nil {
		return
	}
	ke := key.ChordEvent{}
	ke.SetTime()
	ke.Modifiers = mods
	ke.Rune = r
	ke.Action = key.Press
	w.SendEventSignal(&ke, popup)
}

// AddShortcut adds given shortcut -- will issue warning about conflicting
// shortcuts and use the most recent.
func (w *Window) AddShortcut(chord key.Chord, act *Action) {
	if chord == "" {
		return
	}
	if w.Shortcuts == nil {
		w.Shortcuts = make(Shortcuts, 100)
	}
	sa, exists := w.Shortcuts[chord]
	if exists && sa != act {
		log.Printf("gi.Window shortcut: %v already exists on action: %v -- will be overwritten with action: %v\n", chord, sa.Text, act.Text)
	}
	w.Shortcuts[chord] = act
}

// TriggerShortcut attempts to trigger a shortcut, returning true if one was
// triggered, and false otherwise.  Also elminates any shortcuts with deleted
// actions, and does not trigger for Inactive actions.
func (w *Window) TriggerShortcut(chord key.Chord) bool {
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
	if sa.IsInactive() {
		if KeyEventTrace {
			fmt.Printf("Shortcut chord: %v, action: %v -- is inactive, not fired\n", chord, sa.Text)
		}
		return false
	}

	if KeyEventTrace {
		fmt.Printf("Shortcut chord: %v, action: %v triggered\n", chord, sa.Text)
	}
	sa.Trigger()
	return true
}

/////////////////////////////////////////////////////////////////////////////
//                   Popups

// PopupIsMenu returns true if the given popup item is a menu
func PopupIsMenu(pop ki.Ki) bool {
	if pop == nil {
		return false
	}
	nii, ni := KiToNode2D(pop)
	if ni == nil {
		return false
	}
	vp := nii.AsViewport2D()
	if vp == nil {
		return false
	}
	if vp.IsMenu() {
		return true
	}
	return false
}

// PopupIsTooltip returns true if the given popup item is a menu
func PopupIsTooltip(pop ki.Ki) bool {
	if pop == nil {
		return false
	}
	nii, ni := KiToNode2D(pop)
	if ni == nil {
		return false
	}
	vp := nii.AsViewport2D()
	if vp == nil {
		return false
	}
	if vp.IsTooltip() {
		return true
	}
	return false
}

// PopupIsCompleter returns true if the given popup item is a menu and a completer
func PopupIsCompleter(pop ki.Ki) bool {
	if !PopupIsMenu(pop) {
		return false
	}
	nii, ni := KiToNode2D(pop)
	if ni == nil {
		return false
	}
	vp := nii.AsViewport2D()
	if vp == nil {
		return false
	}
	if vp.IsCompleter() {
		return true
	}
	return false
}

// CurPopup returns the current popup, protected with read mutex
func (w *Window) CurPopup() ki.Ki {
	w.PopMu.RLock()
	cpop := w.Popup
	w.PopMu.RUnlock()
	return cpop
}

// CurPopupIsTooltip returns true if current popup is a tooltip
func (w *Window) CurPopupIsTooltip() bool {
	return PopupIsTooltip(w.CurPopup())
}

// SetNextPopup sets the next popup, and what to focus on in that popup if non-nil
func (w *Window) SetNextPopup(pop, focus ki.Ki) {
	w.PopMu.Lock()
	w.NextPopup = pop
	w.PopupFocus = focus
	w.PopMu.Unlock()
}

// SetDelPopup sets the popup to delete next time through event loop
func (w *Window) SetDelPopup(pop ki.Ki) {
	w.PopMu.Lock()
	w.DelPopup = pop
	w.PopMu.Unlock()
}

// ShouldDeletePopupMenu returns true if the given popup item should be deleted
func (w *Window) ShouldDeletePopupMenu(pop ki.Ki, me *mouse.Event) bool {
	if !PopupIsMenu(pop) {
		return false
	}
	if w.NextPopup != nil && PopupIsMenu(w.NextPopup) { // poping up another menu
		return false
	}
	if me.Button != mouse.Left && w.Dragging == nil { // probably menu activation in first place
		return false
	}
	return true
}

// PushPopup pushes current popup onto stack and set new popup.
func (w *Window) PushPopup(pop ki.Ki) {
	w.PopMu.Lock()
	w.NextPopup = nil
	if w.PopupStack == nil {
		w.PopupStack = make([]ki.Ki, 0, 50)
	}
	pop.SetParent(w.This()) // popup has parent as window -- draws directly in to assoc vp
	w.PopupStack = append(w.PopupStack, w.Popup)
	w.Popup = pop
	_, ni := KiToNode2D(pop)
	if ni != nil {
		ni.FullRender2DTree()
	}
	pfoc := w.PopupFocus
	w.PopupFocus = nil
	w.PopMu.Unlock()
	if pfoc != nil {
		w.PushFocus(pfoc)
	} else {
		w.PushFocus(pop)
	}
}

// DisconnectPopup disconnects given popup -- typically the current one.
func (w *Window) DisconnectPopup(pop ki.Ki) {
	w.DisconnectAllEvents(pop, AllPris)
	pop.SetParent(nil) // don't redraw the popup anymore
}

// ClosePopup close given popup -- must be the current one -- returns false if not.
func (w *Window) ClosePopup(pop ki.Ki) bool {
	if pop != w.CurPopup() {
		return false
	}
	w.PopMu.Lock()
	if w.Popup == w.DelPopup {
		w.DelPopup = nil
	}
	w.DisconnectPopup(pop)
	popped := w.PopPopup(pop)
	w.PopMu.Unlock()
	if popped {
		w.PopFocus()
	}
	w.UploadAllViewports()
	return true
}

// PopPopup pops current popup off the popup stack and set to current popup.
// returns true if was actually popped.  MUST be called within PopMu.Lock scope!
func (w *Window) PopPopup(pop ki.Ki) bool {
	nii, ok := pop.(Node2D)
	if ok {
		pvp := nii.AsViewport2D()
		if pvp != nil {
			pvp.DeletePopup()
		}
	}
	sz := len(w.PopupStack)
	if w.Popup == pop {
		if w.PopupStack == nil || sz == 0 {
			w.Popup = nil
		} else {
			w.Popup = w.PopupStack[sz-1]
			w.PopupStack = w.PopupStack[:sz-1]
		}
		return true
	} else {
		for i := sz - 1; i >= 0; i-- {
			pp := w.PopupStack[i]
			if pp == pop {
				w.PopupStack = w.PopupStack[:i+copy(w.PopupStack[i:], w.PopupStack[i+1:])]
				break
			}
		}
		// do nothing
	}
	return false
}

/////////////////////////////////////////////////////////////////////////////
//                   Key Events Handled by Window

// KeyChordEventHiPri handles all the high-priority window-specific key
// events, returning its input on whether any existing popup should be deleted
func (w *Window) KeyChordEventHiPri(e *key.ChordEvent) bool {
	delPop := false
	cs := e.Chord()
	if KeyEventTrace {
		fmt.Printf("Window HiPri KeyInput: %v event: %v\n", w.PathUnique(), e.String())
	}
	kf := KeyFun(cs)
	w.LastModBits = e.Modifiers
	w.LastSelMode = mouse.SelectModeBits(e.Modifiers)
	if e.IsProcessed() {
		return false
	}
	cpop := w.CurPopup()
	switch kf {
	case KeyFunAbort:
		if PopupIsMenu(cpop) || PopupIsTooltip(cpop) {
			delPop = true
			e.SetProcessed()
		}
	case KeyFunAccept:
		if PopupIsMenu(cpop) || PopupIsTooltip(cpop) {
			delPop = true
		}
	}
	// fmt.Printf("key chord: rune: %v Chord: %v\n", e.Rune, e.Chord())
	return delPop
}

// KeyChordEventLowPri handles all the lower-priority window-specific key
// events, returning its input on whether any existing popup should be deleted
func (w *Window) KeyChordEventLowPri(e *key.ChordEvent) bool {
	delPop := false
	cs := e.Chord()
	if KeyEventTrace {
		fmt.Printf("Window LowPri KeyInput: %v\n", w.PathUnique())
	}
	kf := KeyFun(cs)
	w.LastModBits = e.Modifiers
	w.LastSelMode = mouse.SelectModeBits(e.Modifiers)
	if e.IsProcessed() {
		return false
	}
	switch kf {
	case KeyFunFocusNext:
		w.FocusNext(w.Focus)
		e.SetProcessed()
	case KeyFunFocusPrev:
		w.FocusPrev(w.Focus)
		e.SetProcessed()
	case KeyFunGoGiEditor:
		TheViewIFace.GoGiEditor(w.Viewport.This())
		e.SetProcessed()
	case KeyFunZoomIn:
		w.ZoomDPI(1)
		e.SetProcessed()
	case KeyFunZoomOut:
		w.ZoomDPI(-1)
		e.SetProcessed()
	case KeyFunPrefs:
		TheViewIFace.PrefsView(&Prefs)
		e.SetProcessed()
	case KeyFunRefresh:
		fmt.Printf("Window: %v display refreshed\n", w.Nm)
		w.FocusInactivate()
		w.FullReRender()
		// w.UploadAllViewports()
		e.SetProcessed()
	}
	switch cs { // some other random special codes, during dev..
	case "Control+Alt+R":
		ProfileToggle()
		e.SetProcessed()
	case "Control+Alt+F":
		w.BenchmarkFullRender()
		e.SetProcessed()
	case "Control+Alt+G":
		w.BenchmarkReRender()
		e.SetProcessed()
	}
	// fmt.Printf("key chord: rune: %v Chord: %v\n", e.Rune, e.Chord())
	return delPop
}

/////////////////////////////////////////////////////////////////////////////
//                   Key Focus

// SetStartFocus sets the given item to be first focus when window opens.
func (w *Window) SetStartFocus(k ki.Ki) {
	w.FocusMu.Lock()
	w.StartFocus = k
	w.FocusMu.Unlock()
}

// SetFocus sets focus to given item -- returns true if focus changed.
func (w *Window) SetFocus(k ki.Ki) bool {
	w.FocusMu.Lock()
	if w.Focus == k {
		if k != nil {
			_, ni := KiToNode2D(k)
			if ni != nil && ni.This() != nil {
				ni.SetFlag(int(HasFocus)) // ensure focus flag always set
			}
		}
		w.FocusMu.Unlock()
		return false
	}

	updt := w.UpdateStart()
	defer w.UpdateEnd(updt)

	if w.Focus != nil {
		nii, ni := KiToNode2D(w.Focus)
		if ni != nil && ni.This() != nil {
			ni.ClearFlag(int(HasFocus))
			// fmt.Printf("clear foc: %v\n", ni.PathUnique())
			nii.FocusChanged2D(FocusLost)
		}
	}
	w.Focus = k
	if k == nil {
		return true
	}
	nii, ni := KiToNode2D(k)
	if ni == nil || ni.This() == nil { // only 2d for now
		w.Focus = nil
		return false
	}
	ni.SetFlag(int(HasFocus))
	w.FocusActive = true
	// fmt.Printf("set foc: %v\n", ni.PathUnique())
	w.ClearNonFocus(w.Focus) // shouldn't need this but actually sometimes do
	w.FocusMu.Unlock()
	nii.FocusChanged2D(FocusGot)
	return true
}

// FocusNext sets the focus on the next item that can accept focus after the
// given item (can be nil) -- returns true if a focus item found.
func (w *Window) FocusNext(foc ki.Ki) bool {
	gotFocus := false
	focusNext := false // get the next guy
	if foc == nil {
		focusNext = true
	}

	focRoot := w.Viewport.This()
	cpop := w.CurPopup()
	if cpop != nil {
		focRoot = cpop
	}

	for i := 0; i < 2; i++ {
		focRoot.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
			if gotFocus {
				return false
			}
			_, ni := KiToNode2D(k)
			if ni == nil || ni.This() == nil {
				return true
			}
			if foc == k { // current focus can be a non-can-focus item
				focusNext = true
				return true
			}
			if !ni.CanFocus() {
				return true
			}
			if focusNext {
				w.SetFocus(k)
				gotFocus = true
				return false // done
			}
			return true
		})
		if gotFocus {
			return true
		}
		focusNext = true // this time around, just get the first one
	}
	return gotFocus
}

// FocusOnOrNext sets the focus on the given item, or the next one that can
// accept focus -- returns true if a new focus item found.
func (w *Window) FocusOnOrNext(foc ki.Ki) bool {
	w.FocusMu.Lock()
	if w.Focus == foc {
		w.FocusMu.Unlock()
		return true
	}
	w.FocusMu.Unlock()
	_, ni := KiToNode2D(foc)
	if ni == nil || ni.This() == nil {
		return false
	}
	if ni.CanFocus() {
		w.SetFocus(foc)
		return true
	}
	return w.FocusNext(foc)
}

// FocusPrev sets the focus on the previous item before the given item (can be nil)
func (w *Window) FocusPrev(foc ki.Ki) bool {
	if foc == nil { // must have a current item here
		w.FocusLast()
		return false
	}

	gotFocus := false
	var prevItem ki.Ki

	focRoot := w.Viewport.This()
	cpop := w.CurPopup()
	if cpop != nil {
		focRoot = cpop
	}

	focRoot.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
		if gotFocus {
			return false
		}
		// todo: see about 3D guys
		_, ni := KiToNode2D(k)
		if ni == nil || ni.This() == nil {
			return true
		}
		if foc == k {
			gotFocus = true
			return false
		}
		if !ni.CanFocus() {
			return true
		}
		prevItem = k
		return true
	})
	if gotFocus && prevItem != nil {
		w.SetFocus(prevItem)
		return true
	} else {
		return w.FocusLast()
	}
}

// FocusLast sets the focus on the last item in the tree -- returns true if a
// focusable item was found
func (w *Window) FocusLast() bool {
	var lastItem ki.Ki

	focRoot := w.Viewport.This()
	cpop := w.CurPopup()
	if cpop != nil {
		focRoot = cpop
	}

	focRoot.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
		// todo: see about 3D guys
		_, ni := KiToNode2D(k)
		if ni == nil || ni.This() == nil {
			return true
		}
		if !ni.CanFocus() {
			return true
		}
		lastItem = k
		return true
	})
	w.SetFocus(lastItem)
	if lastItem == nil {
		return false
	}
	return true
}

// ClearNonFocus clears the focus of any non-w.Focus item.
func (w *Window) ClearNonFocus(foc ki.Ki) {
	focRoot := w.Viewport.This()
	cpop := w.CurPopup()
	if cpop != nil {
		focRoot = cpop
	}

	updated := false
	updt := false

	focRoot.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
		if k == focRoot { // skip top-level
			return true
		}
		// todo: see about 3D guys
		nii, ni := KiToNode2D(k)
		if ni == nil || ni.This() == nil {
			return true
		}
		if foc == k {
			return true
		}
		if ni.HasFocus() {
			// fmt.Printf("ClearNonFocus: %v\n", ni.PathUnique())
			if !updated {
				updated = true
				updt = w.UpdateStart()
			}
			ni.ClearFlag(int(HasFocus))
			nii.FocusChanged2D(FocusLost)
		}
		return true
	})
	if updated {
		w.UpdateEnd(updt)
	}
}

// PushFocus pushes current focus onto stack and sets new focus.
func (w *Window) PushFocus(p ki.Ki) {
	w.FocusMu.Lock()
	if w.FocusStack == nil {
		w.FocusStack = make([]ki.Ki, 0, 50)
	}
	w.FocusStack = append(w.FocusStack, w.Focus)
	w.Focus = nil // don't un-focus on prior item when pushing
	w.FocusMu.Unlock()
	w.FocusOnOrNext(p)
}

// PopFocus pops off the focus stack and sets prev to current focus.
func (w *Window) PopFocus() {
	w.FocusMu.Lock()
	if w.FocusStack == nil || len(w.FocusStack) == 0 {
		w.Focus = nil
		return
	}
	sz := len(w.FocusStack)
	w.Focus = nil
	nxtf := w.FocusStack[sz-1]
	_, ni := KiToNode2D(nxtf)
	if ni != nil && ni.This() != nil {
		w.FocusMu.Unlock()
		w.SetFocus(nxtf)
		w.FocusMu.Lock()
	}
	w.FocusStack = w.FocusStack[:sz-1]
	w.FocusMu.Unlock()
}

// FocusActiveClick updates the FocusActive status based on mouse clicks in
// or out of the focused item
func (w *Window) FocusActiveClick(e *mouse.Event) {
	w.FocusMu.Lock()
	defer w.FocusMu.Unlock()
	if w.Focus == nil || e.Button != mouse.Left || e.Action != mouse.Press {
		return
	}
	cpop := w.CurPopup()
	if cpop != nil { // no updating on popus
		return
	}
	nii, ni := KiToNode2D(w.Focus)
	if ni != nil && ni.This() != nil {
		if e.Pos().In(ni.WinBBox) {
			if !w.FocusActive {
				w.FocusActive = true
				nii.FocusChanged2D(FocusActive)
			}
		} else {
			if w.MainMenu != nil {
				if e.Pos().In(w.MainMenu.WinBBox) { // main menu is not inactivating!
					return
				}
			}
			if w.FocusActive {
				w.FocusActive = false
				nii.FocusChanged2D(FocusInactive)
			}
		}
	}
}

// FocusInactivate inactivates the current focus element
func (w *Window) FocusInactivate() {
	w.FocusMu.Lock()
	defer w.FocusMu.Unlock()
	if w.Focus == nil || !w.FocusActive {
		return
	}
	nii, ni := KiToNode2D(w.Focus)
	if ni != nil && ni.This() != nil {
		w.FocusActive = false
		nii.FocusChanged2D(FocusInactive)
	}
}

// WindowInFocus returns true if this window is the one currently in focus
func (w *Window) IsWindowInFocus() bool {
	fwin := oswin.TheApp.WindowInFocus()
	if w.OSWin == fwin {
		return true
	}
	return false
}

/////////////////////////////////////////////////////////////////////////////
//                   DND: Drag-n-Drop

// DNDStartEvent handles drag-n-drop start events.
func (w *Window) DNDStartEvent(e *mouse.DragEvent) {
	de := dnd.Event{EventBase: e.EventBase, Where: e.Where, Modifiers: e.Modifiers}
	de.Processed = false
	de.Action = dnd.Start
	de.DefaultMod()               // based on current key modifiers
	w.SendEventSignal(&de, false) // popup = false: ignore any popups
	// now up to receiver to call StartDragNDrop if they want to..
}

// StartDragNDrop is called by a node to start a drag-n-drop operation on
// given source node, which is responsible for providing the data and image
// representation of the node.
func (w *Window) StartDragNDrop(src ki.Ki, data mimedata.Mimes, img Node2D) {
	// todo: 3d version later..
	w.DNDSource = src
	w.DNDData = data
	wimg := img.AsWidget()
	if _, sni := KiToNode2D(src); sni != nil { // 2d case
		if sw := sni.AsWidget(); sw != nil {
			wimg.LayData.AllocPos.SetPoint(sw.LayData.AllocPos.ToPoint())
		}
	}
	wimg.This().SetName(src.UniqueName())
	w.OverlayVp.AddChild(wimg.This())
	w.DNDImage = wimg.This()
	w.DNDSetCursor(dnd.DefaultModBits(w.LastModBits))
	// fmt.Printf("starting dnd: %v\n", src.Name())
}

// DNDMoveEvent handles drag-n-drop move events.
func (w *Window) DNDMoveEvent(e *mouse.DragEvent) {
	if nii, _ := KiToNode2D(w.DNDImage); nii != nil { // 2d case
		if wg := nii.AsWidget(); wg != nil {
			wg.LayData.AllocPos.SetPoint(e.Where)
		}
	} // else 3d..
	// todo: when e.Where goes negative, transition to OS DND
	// todo: send move / enter / exit events to anyone listening
	de := dnd.MoveEvent{Event: dnd.Event{EventBase: e.Event.EventBase, Where: e.Event.Where, Modifiers: e.Event.Modifiers}, From: e.From, LastTime: e.LastTime}
	de.Processed = false
	de.DefaultMod() // based on current key modifiers
	de.Action = dnd.Move
	w.SendEventSignal(&de, false) // popup = false: ignore any popups
	w.GenDNDFocusEvents(&de, false)
	w.DNDUpdateCursor(de.Mod)
	w.RenderOverlays()
	e.SetProcessed()
}

// GenDNDFocusEvents processes mouse.MoveEvent to generate dnd.FocusEvent
// events -- returns true if any such events were sent.  If popup is true,
// then only items on popup are in scope, otherwise items NOT on popup are in
// scope (if no popup, everything is in scope).  Extra work is done to ensure
// that Exit from prior widget is always sent before Enter to next one.
func (w *Window) GenDNDFocusEvents(mev *dnd.MoveEvent, popup bool) bool {
	fe := dnd.FocusEvent{Event: mev.Event}
	pos := mev.Pos()
	ftyp := oswin.DNDFocusEvent

	// first pass is just to get all the ins and outs
	var ins, outs WinEventRecvList

	for pri := HiPri; pri < EventPrisN; pri++ {
		esig := w.EventSigs[ftyp][pri]
		for recv, fun := range esig.Cons {
			if recv.IsDeleted() { // destroyed is filtered upstream
				continue
			}
			_, ni := KiToNode2D(recv)
			if ni != nil {
				if !w.IsInScope(ni, popup) {
					continue
				}
				in := pos.In(ni.WinBBox)
				if in {
					if !ni.HasFlag(int(DNDHasEntered)) {
						ni.SetFlag(int(DNDHasEntered))
						ins.Add(recv, fun, 0)
					}
				} else { // mouse not in object
					if ni.HasFlag(int(DNDHasEntered)) {
						ni.ClearFlag(int(DNDHasEntered))
						outs.Add(recv, fun, 0)
					}
				}
			} else {
				// todo: 3D
			}
		}
	}
	if len(outs)+len(ins) > 0 {
		updt := w.UpdateStart()
		// now send all the exits before the enters..
		fe.Action = dnd.Exit
		for i := range outs {
			outs[i].Call(w.This(), int64(ftyp), &fe)
		}
		fe.Action = dnd.Enter
		for i := range ins {
			ins[i].Call(w.This(), int64(ftyp), &fe)
		}
		w.UpdateEnd(updt)
		return true
	}
	return false
}

// SendDNDHoverEvent sends DND hover event, based on last mouse move event
func (w *Window) SendDNDHoverEvent(e *mouse.DragEvent) {
	he := dnd.FocusEvent{Event: dnd.Event{EventBase: e.EventBase, Where: e.Where, Modifiers: e.Modifiers}}
	he.Processed = false
	he.Action = dnd.Hover
	w.SendEventSignal(&he, false) // popup = false by default
}

// DNDDropEvent handles drag-n-drop drop event (action = release).
func (w *Window) DNDDropEvent(e *mouse.Event) {
	de := dnd.Event{EventBase: e.EventBase, Where: e.Where, Modifiers: e.Modifiers}
	de.Processed = false
	de.DefaultMod()
	de.Action = dnd.DropOnTarget
	de.Data = w.DNDData
	de.Source = w.DNDSource
	w.DNDSource.ClearFlag(int(NodeDragging))
	w.Dragging = nil
	w.SendEventSignal(&de, false) // popup = false: ignore any popups
	w.DNDFinalEvent = &de
	w.ClearDragNDrop()
	e.SetProcessed()
}

// FinalizeDragNDrop is called by a node to finalize the drag-n-drop
// operation, after given action has been performed on the target -- allows
// target to cancel, by sending dnd.DropIgnore.
func (w *Window) FinalizeDragNDrop(action dnd.DropMods) {
	if w.DNDFinalEvent == nil { // shouldn't happen...
		return
	}
	de := w.DNDFinalEvent
	de.Processed = false
	de.Mod = action
	if de.Source != nil {
		et := de.Type()
		de.Action = dnd.DropFmSource
		for pri := HiPri; pri < EventPrisN; pri++ {
			w.EventSigs[et][pri].SendSig(de.Source, w, int64(et), (oswin.Event)(de))
		}
	}
	w.DNDFinalEvent = nil
}

// ClearDragNDrop clears any existing DND values.
func (w *Window) ClearDragNDrop() {
	w.DNDSource = nil
	w.DNDData = nil
	w.OverlayVp.DeleteChild(w.DNDImage, true)
	w.DNDImage = nil
	w.DNDClearCursor()
	w.Dragging = nil
	w.RenderOverlays()
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
func (w *Window) DNDSetCursor(dmod dnd.DropMods) {
	dndc := DNDModCursor(dmod)
	oswin.TheApp.Cursor(w.OSWin).PushIfNot(dndc)
}

// DNDNotCursor sets the cursor to Not = can't accept a drop
func (w *Window) DNDNotCursor() {
	oswin.TheApp.Cursor(w.OSWin).PushIfNot(cursor.Not)
}

// DNDUpdateCursor updates the cursor based on the curent DND event mod if
// different from current (but no update if Not)
func (w *Window) DNDUpdateCursor(dmod dnd.DropMods) bool {
	dndc := DNDModCursor(dmod)
	curs := oswin.TheApp.Cursor(w.OSWin)
	if !curs.IsDrag() || curs.Current() == dndc {
		return false
	}
	curs.Push(dndc)
	return true
}

// DNDClearCursor clears any existing DND cursor that might have been set.
func (w *Window) DNDClearCursor() {
	curs := oswin.TheApp.Cursor(w.OSWin)
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

func (w *Window) ReportWinNodes() {
	nn := 0
	w.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		nn++
		return true
	})
	fmt.Printf("Window: %v has: %v nodes\n", w.Nm, nn)
}

// BenchmarkFullRender runs benchmark of 50 full re-renders (full restyling, layout,
// and everything), reporting targeted profile results and generating standard
// Go cpu.prof and mem.prof outputs.
func (w *Window) BenchmarkFullRender() {
	fmt.Println("Starting BenchmarkFullRender")
	w.ReportWinNodes()
	StartCPUMemProfile()
	StartTargProfile()
	ts := time.Now()
	n := 50
	for i := 0; i < n; i++ {
		w.Viewport.FullRender2DTree()
	}
	td := time.Now().Sub(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	EndTargProfile()
	EndCPUMemProfile()
}

// BenchmarkReRender runs benchmark of 50 re-render-only updates of display
// (just the raw rendering, no styling or layout), reporting targeted profile
// results and generating standard Go cpu.prof and mem.prof outputs.
func (w *Window) BenchmarkReRender() {
	fmt.Println("Starting BenchmarkReRender")
	w.ReportWinNodes()
	StartTargProfile()
	ts := time.Now()
	n := 50
	for i := 0; i < n; i++ {
		w.Viewport.Render2DTree()
	}
	td := time.Now().Sub(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	EndTargProfile()
}

//////////////////////////////////////////////////////////////////////////////////
//  WindowLists

// WindowList is a list of windows.
type WindowList []*Window

// Add adds a window to the list.
func (wl *WindowList) Add(w *Window) {
	WindowGlobalMu.Lock()
	*wl = append(*wl, w)
	WindowGlobalMu.Unlock()
}

// Delete removes a window from the list -- returns true if deleted.
func (wl *WindowList) Delete(w *Window) bool {
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
	sz := len(*wl)
	got := false
	for i := sz - 1; i >= 0; i-- {
		wi := (*wl)[i]
		if wi == w {
			copy((*wl)[i:], (*wl)[i+1:])
			(*wl)[sz-1] = nil
			(*wl) = (*wl)[:sz-1]
			sz = len(*wl)
			got = true
		}
	}
	return got
}

// FindName finds window with given name on list (case sensitive) -- returns
// window and true if found, nil, false otherwise
func (wl *WindowList) FindName(name string) (*Window, bool) {
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
	for _, wi := range *wl {
		if wi.Nm == name {
			return wi, true
		}
	}
	return nil, false
}

// Len() returns the length of the list, concurrent-safe
func (wl *WindowList) Len() int {
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
	return len(*wl)
}

// Win gets window at given index, concurrent-safe
func (wl *WindowList) Win(idx int) *Window {
	WindowGlobalMu.Lock()
	defer WindowGlobalMu.Unlock()
	if idx >= len(*wl) || idx < 0 {
		return nil
	}
	return (*wl)[idx]
}

// AllWindows is the list of all windows that have been created (dialogs, main
// windows, etc).
var AllWindows WindowList

// DialogWindows is the list of only dialog windows that have been created.
var DialogWindows WindowList

// MainWindows is the list of main windows (non-dialogs) that have been
// created.
var MainWindows WindowList

//////////////////////////////////////////////////////////////////////////////////
//  WindowGeom

var WinGeomPrefs = WindowGeomPrefs{}

// WindowGeom records the geometry settings used for a given window
type WindowGeom struct {
	WinName    string
	Screen     string
	LogicalDPI float32
	Size       image.Point
	Pos        image.Point
}

// WindowGeomPrefs records the window geometry by window name, screen name --
// looks up the info automatically for new windows and saves persistently
type WindowGeomPrefs map[string]map[string]WindowGeom

// WinGeomPrefsFileName is the name of the preferences file in GoGi prefs directory
var WinGeomPrefsFileName = "win_geom_prefs.json"

// WinGeomPrefsMu is read-write mutex that protects updating of WinGeomPrefs
var WinGeomPrefsMu sync.RWMutex

// Open Window Geom preferences from GoGi standard prefs directory
func (wg *WindowGeomPrefs) Open() error {
	WinGeomPrefsMu.Lock()
	defer WinGeomPrefsMu.Unlock()

	if wg == nil {
		*wg = make(WindowGeomPrefs, 1000)
	}
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName)
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		log.Println(err)
		return err
	}
	err = json.Unmarshal(b, wg)
	if err != nil {
		log.Println(err)
	}
	return err
}

// Save Window Geom Preferences to GoGi standard prefs directory
func (wg *WindowGeomPrefs) Save() error {
	WinGeomPrefsMu.Lock()
	defer WinGeomPrefsMu.Unlock()

	if wg == nil {
		return nil
	}
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName)
	b, err := json.MarshalIndent(wg, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(pnm, b, 0644)
	if err != nil {
		log.Println(err)
	}
	return err
}

// RecordPref records current state of window as preference
func (wg *WindowGeomPrefs) RecordPref(win *Window) {
	WinGeomPrefsMu.Lock()
	if wg == nil {
		*wg = make(WindowGeomPrefs, 100)
	}
	sc := win.OSWin.Screen()
	wgr := WindowGeom{WinName: win.Nm, Screen: sc.Name, LogicalDPI: win.LogicalDPI()}
	wgr.Pos = win.OSWin.Position()
	wgr.Size = win.OSWin.Size()
	if wgr.Size == image.ZP {
		WinGeomPrefsMu.Unlock()
		// fmt.Printf("Pref: NOT storing null size for win: %v scrn: %v\n", win.Nm, sc.Name)
		return
	}
	if (*wg)[win.Nm] == nil {
		(*wg)[win.Nm] = make(map[string]WindowGeom, 10)
	}
	(*wg)[win.Nm][sc.Name] = wgr
	WinGeomPrefsMu.Unlock()
	wg.Save()
}

// Pref returns an existing preference for given window name, or one adapted
// to given screen if only records are on a different screen -- if scrn is nil
// then default (first) screen is used from oswin.TheApp
func (wg *WindowGeomPrefs) Pref(winName string, scrn *oswin.Screen) *WindowGeom {
	WinGeomPrefsMu.RLock()
	defer WinGeomPrefsMu.RUnlock()

	if wg == nil {
		return nil
	}
	wps, ok := (*wg)[winName]
	if !ok {
		return nil
	}

	if scrn == nil {
		scrn = oswin.TheApp.Screen(0)
		// fmt.Printf("Pref: using scrn 0: %v\n", scrn.Name)
	}
	scsz := scrn.Geometry.Size()

	wp, ok := wps[scrn.Name]
	if ok {
		if scrn.LogicalDPI == wp.LogicalDPI {
			wp.Size.X = ints.MinInt(wp.Size.X, scsz.X)
			wp.Size.Y = ints.MinInt(wp.Size.Y, scsz.Y)
			return &wp
		} else {
			// fmt.Printf("rescaling scrn dpi: %v saved dpi: %v\n", scrn.LogicalDPI, wp.LogicalDPI)
			wp.Size.X = int(float32(wp.Size.X) * (scrn.LogicalDPI / wp.LogicalDPI))
			wp.Size.Y = int(float32(wp.Size.Y) * (scrn.LogicalDPI / wp.LogicalDPI))
			wp.Size.X = ints.MinInt(wp.Size.X, scsz.X)
			wp.Size.Y = ints.MinInt(wp.Size.Y, scsz.Y)
			return &wp
		}
	}

	if len(wps) == 0 { // shouldn't happen
		return nil
	}

	trgdpi := scrn.LogicalDPI
	// fmt.Printf("Pref: falling back on dpi conversion: %v\n", trgdpi)

	// try to find one with same logical dpi, else closest
	var closest *WindowGeom
	minDPId := float32(100000.0)
	for _, wp = range wps {
		if wp.LogicalDPI == trgdpi {
			return &wp
		}
		dpid := math32.Abs(wp.LogicalDPI - trgdpi)
		if dpid < minDPId {
			minDPId = dpid
			closest = &wp
		}
	}

	wp = *closest
	rescale := trgdpi / closest.LogicalDPI
	wp.Pos.X = int(float32(wp.Pos.X) * rescale)
	wp.Pos.Y = int(float32(wp.Pos.Y) * rescale)
	wp.Size.X = int(float32(wp.Size.X) * rescale)
	wp.Size.Y = int(float32(wp.Size.Y) * rescale)
	wp.Size.X = ints.MinInt(wp.Size.X, scsz.X)
	wp.Size.Y = ints.MinInt(wp.Size.Y, scsz.Y)
	// fmt.Printf("Pref: rescaled pos: %v size: %v\n", wp.Pos, wp.Size)
	return &wp
}

// DeleteAll deletes the file that saves the position and size of each window,
// by screen, and clear current in-memory cache.  You shouldn't need to use
// this but sometimes useful for testing.
func (wg *WindowGeomPrefs) DeleteAll() {
	WinGeomPrefsMu.RLock()
	defer WinGeomPrefsMu.RUnlock()

	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName)
	os.Remove(pnm)
	*wg = make(WindowGeomPrefs, 1000)
}
