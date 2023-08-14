// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/goki/gi/colormap"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/gi/units"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"
	"github.com/goki/vgpu/vgpu"
	"github.com/goki/vgpu/vphong"
)

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

	// WinEventTrace reports a trace of window events to stdout
	// can be set in PrefsDebug from prefs gui
	// excludes mouse move events
	WinEventTrace = false

	// WinPublishTrace reports the stack trace leading up to win publish events
	// which are expensive -- wrap multiple updates in UpdateStart / End
	// to prevent
	// can be set in PrefsDebug from prefs gui
	WinPublishTrace = false

	// WinDrawTrace highlights the window regions that are drawn to update
	// the window, using filled colored rectangles
	WinDrawTrace = false

	// KeyEventTrace reports a trace of keyboard events to stdout
	// can be set in PrefsDebug from prefs gui
	KeyEventTrace = false

	// EventTrace reports a trace of event handing to stdout.
	// can be set in PrefsDebug from prefs gui
	EventTrace = false

	// WinNewCloseTime records last time a new window was opened or another
	// closed -- used to trigger updating of Window menus on each window.
	WinNewCloseTime time.Time

	// WindowGlobalMu is a mutex for any global state associated with windows
	WindowGlobalMu sync.Mutex

	// WindowOpenTimer is used for profiling the open time of windows
	// if doing profiling, it will report the time elapsed in msec
	// to point of establishing initial focus in the window.
	WindowOpenTimer time.Time
)

// These constants define use of Texture images in vdraw.Drawer
// for updating the window efficiently.  They are allocated in
// sets of 16:
// Set 0: 0 = main viewport, DirectUploads, Popups
// Set 1: 16 region update images
// Set 2: 16 sprite arrays with 128 allocated per size
const (
	// MaxDirectUploads are direct GPU texture transfer sources
	MaxDirectUploads = 7

	// MaxPopups is the maximum number of viewport regions
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

// Window provides an OS-specific window and all the associated event
// handling.  Widgets connect to event signals to receive relevant GUI events.
// There is a master Viewport that contains the full bitmap image of the
// window, onto which most widgets render.  For main windows (not dialogs or
// other popups), there is a master vertical layout under the Viewport
// (MasterVLay), whose first element is the MainMenu for the window (which can
// be empty, in which case it is not displayed).  On MacOS, this main menu
// updates the overall menubar, and also can show the local menu (on by default).
//
// Widgets should always use methods to access / set state, and generally should
// not do much directly with the window.  Almost everything here needs to be
// guarded by various mutexes.  Leaving everything accessible so expert outside
// access is still possible in a pinch, but again don't use it unless you know
// what you're doing (and it might change over time too..)
//
// Rendering logic:
//   - vdraw.Drawer manages all rendering to the window surface, provided via
//     the OSWin window, using vulkan stored images (16 max)
//   - Order is: Base Viewport2D (image 0), then direct uploads, popups, and sprites.
//   - DirectUps (e.g., gi3d.Scene) directly upload their own texture to a Draw image
//     (note: cannot upload directly to window as this prevents popups and overlays)
//   - Popups (which have their own Viewports)
//   - Sprites are managed as layered textures of the same size, to enable
//     unlimited number packed into a few descriptors for standard sizes.
type Window struct {
	NodeBase

	// displayed name of window, for window manager etc -- window object name is the internal handle and is used for tracking property info etc
	Title string `desc:"displayed name of window, for window manager etc -- window object name is the internal handle and is used for tracking property info etc"`

	// [view: -] the main data element represented by this window -- used for Recycle* methods for windows that represent a given data element -- prevents redundant windows
	Data any `json:"-" xml:"-" view:"-" desc:"the main data element represented by this window -- used for Recycle* methods for windows that represent a given data element -- prevents redundant windows"`

	// OS-specific window interface -- handles all the os-specific functions, including delivering events etc
	OSWin oswin.Window `json:"-" xml:"-" desc:"OS-specific window interface -- handles all the os-specific functions, including delivering events etc"`

	// event manager that handles dispersing events to nodes
	EventMgr EventMgr `json:"-" xml:"-" desc:"event manager that handles dispersing events to nodes"`

	// convenience pointer to window's master viewport child that handles the rendering
	Viewport *Viewport2D `json:"-" xml:"-" desc:"convenience pointer to window's master viewport child that handles the rendering"`

	// main vertical layout under Viewport -- first element is MainMenu (always -- leave empty to not render)
	MasterVLay *Layout `json:"-" xml:"-" desc:"main vertical layout under Viewport -- first element is MainMenu (always -- leave empty to not render)"`

	// main menu -- is first element of MasterVLay always -- leave empty to not render.  On MacOS, this drives screen main menu
	MainMenu *MenuBar `json:"-" xml:"-" desc:"main menu -- is first element of MasterVLay always -- leave empty to not render.  On MacOS, this drives screen main menu"`

	// sprites are named images that are rendered last overlaying everything else.
	Sprites Sprites `json:"-" xml:"-" desc:"sprites are named images that are rendered last overlaying everything else."`

	// name of sprite that is being dragged -- sprite event function is responsible for setting this.
	SpriteDragging string `json:"-" xml:"-" desc:"name of sprite that is being dragged -- sprite event function is responsible for setting this."`

	// [view: -] mutex that protects all updating / uploading of Textures
	UpMu sync.Mutex `json:"-" xml:"-" view:"-" desc:"mutex that protects all updating / uploading of Textures"`

	// currently active shortcuts for this window (shortcuts are always window-wide -- use widget key event processing for more local key functions)
	Shortcuts Shortcuts `json:"-" xml:"-" desc:"currently active shortcuts for this window (shortcuts are always window-wide -- use widget key event processing for more local key functions)"`

	// Current popup viewport that gets all events
	Popup ki.Ki `json:"-" xml:"-" desc:"Current popup viewport that gets all events"`

	// stack of popups
	PopupStack []ki.Ki `json:"-" xml:"-" desc:"stack of popups"`

	// this popup will be pushed at the end of the current event cycle -- use SetNextPopup
	NextPopup ki.Ki `json:"-" xml:"-" desc:"this popup will be pushed at the end of the current event cycle -- use SetNextPopup"`

	// node to focus on when next popup is activated -- use SetNextPopup
	PopupFocus ki.Ki `json:"-" xml:"-" desc:"node to focus on when next popup is activated -- use SetNextPopup"`

	// this popup will be popped at the end of the current event cycle -- use SetDelPopup
	DelPopup ki.Ki `json:"-" xml:"-" desc:"this popup will be popped at the end of the current event cycle -- use SetDelPopup"`

	// [view: -] read-write mutex that protects popup updating and access
	PopMu             sync.RWMutex `json:"-" xml:"-" view:"-" desc:"read-write mutex that protects popup updating and access"`
	lastWinMenuUpdate time.Time
	// below are internal vars used during the event loop
	delPop        bool
	skippedResize *window.Event
	lastEt        oswin.EventType

	SelectedWidget chan *WidgetBase

	// dir draws are direct upload regions -- direct uploaders upload their images directly to an image here
	DirDraws WindowDrawers `desc:"dir draws are direct upload regions -- direct uploaders upload their images directly to an image here"`
	PopDraws WindowDrawers // popup regions
	UpdtRegs WindowUpdates // misc vp update regions

	// the phongs for the window
	Phongs []*vphong.Phong ` json:"-" xml:"-" desc:"the phongs for the window"`

	// the render frames for the window
	Frames []*vgpu.RenderFrame ` json:"-" xml:"-" desc:"the render frames for the window"`
}

var TypeWindow = kit.Types.AddType(&Window{}, WindowProps)

var WindowProps = ki.Props{
	ki.EnumTypeFlag: TypeWinFlags,
}

// WinFlags extend NodeBase NodeFlags to hold Window state
type WinFlags int

var TypeWinFlags = kit.Enums.AddEnumExt(TypeNodeFlags, WinFlagsN, kit.BitFlag, nil)

const (
	// WinFlagHasGeomPrefs indicates if this window has WinGeomPrefs setting that
	// sized it -- affects whether other default geom should be applied.
	WinFlagHasGeomPrefs WinFlags = WinFlags(NodeFlagsN) + iota

	// WinFlagUpdating is atomic flag around global updating -- routines can check IsWinUpdating and bail
	WinFlagUpdating

	// WinFlagIsClosing is atomic flag indicating window is closing
	WinFlagIsClosing

	// WinFlagIsResizing is atomic flag indicating window is resizing
	WinFlagIsResizing

	// WinFlagGotPaint have we received our first paint event yet?
	// ignore other window events before this point
	WinFlagGotPaint

	// WinFlagGotFocus indicates that have we received OSWin focus
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

	WinFlagsN
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

// IsFocusActive returns true if window has focus active flag set
func (w *Window) IsFocusActive() bool {
	return w.HasFlag(int(WinFlagFocusActive))
}

// SetFocusActiveState sets focus active flag to given state
func (w *Window) SetFocusActiveState(active bool) {
	w.SetFlagState(active, int(WinFlagFocusActive))
}

// IsInSelectionMode returns true if window has selection mode set
func (w *Window) IsInSelectionMode() bool {
	return w.HasFlag(int(WinFlagSelectionMode))
}

// SetSelectionModeState sets selection mode to given state
func (w *Window) SetSelectionModeState(selmode bool) {
	w.SetFlagState(selmode, int(WinFlagSelectionMode))
}

/////////////////////////////////////////////////////////////////////////////
//        App wrappers for oswin (end-user doesn't need to import)

// SetAppName sets the application name -- defaults to GoGi if not otherwise set
// Name appears in the first app menu, and specifies the default application-specific
// preferences directory, etc
func SetAppName(name string) {
	oswin.TheApp.SetName(name)
}

// AppName returns the application name -- see SetAppName to set
func AppName() string {
	return oswin.TheApp.Name()
}

// SetAppAbout sets the 'about' info for the app -- appears as a menu option
// in the default app menu
func SetAppAbout(about string) {
	oswin.TheApp.SetAbout(about)
}

// SetQuitReqFunc sets the function that is called whenever there is a
// request to quit the app (via a OS or a call to QuitReq() method).  That
// function can then adjudicate whether and when to actually call Quit.
func SetQuitReqFunc(fun func()) {
	oswin.TheApp.SetQuitReqFunc(fun)
}

// SetQuitCleanFunc sets the function that is called whenever app is
// actually about to quit (irrevocably) -- can do any necessary
// last-minute cleanup here.
func SetQuitCleanFunc(fun func()) {
	oswin.TheApp.SetQuitCleanFunc(fun)
}

// Quit closes all windows and exits the program.
func Quit() {
	if !oswin.TheApp.IsQuitting() {
		oswin.TheApp.Quit()
	}
}

// PollEvents tells the main event loop to check for any gui events right now.
// Call this periodically from longer-running functions to ensure
// GUI responsiveness.
func PollEvents() {
	oswin.TheApp.PollEvents()
}

// OpenURL opens the given URL in the user's default browser.  On Linux
// this requires that xdg-utils package has been installed -- uses
// xdg-open command.
func OpenURL(url string) {
	oswin.TheApp.OpenURL(url)
}

/////////////////////////////////////////////////////////////////////////////
//                   New Windows and Init

// NewWindow creates a new window with given internal name handle, display
// name, and options.
func NewWindow(name, title string, opts *oswin.NewWindowOptions) *Window {
	Init() // overall gogi system initialization
	win := &Window{}
	win.InitName(win, name)
	win.EventMgr.Master = win
	win.Title = title
	win.SetOnlySelfUpdate() // has its own PublishImage update logic
	var err error
	win.OSWin, err = oswin.TheApp.NewWindow(opts)
	if err != nil {
		fmt.Printf("GoGi NewWindow error: %v \n", err)
		return nil
	}
	win.OSWin.SetName(title)
	win.OSWin.SetParent(win.This())
	win.NodeSig.Connect(win.This(), SignalWindowPublish)
	drw := win.OSWin.Drawer()
	drw.SetMaxTextures(vgpu.MaxTexturesPerSet * 3) // use 3 sets

	win.DirDraws.SetIdxRange(1, MaxDirectUploads)
	// win.DirDraws.FlipY = true // drawing is flipped in general here.
	win.PopDraws.SetIdxRange(win.DirDraws.MaxIdx, MaxPopups)
	win.UpdtRegs.SetIdxRange(RegionUpdateStart, MaxRegionUpdates)

	win.OSWin.SetDestroyGPUResourcesFunc(func() {
		for _, ph := range win.Phongs {
			ph.Destroy()
		}
		for _, fr := range win.Frames {
			fr.Destroy()
		}
	})
	return win
}

// NewMainWindow creates a new standard main window with given internal handle
// name, display name, and sizing, with default positioning, and initializes a
// viewport within it. The width and height are in standardized "pixel" units
// (96 per inch), not the actual underlying raw display dot pixels
func NewMainWindow(name, title string, width, height int) *Window {
	Init() // overall gogi system initialization, at latest possible moment
	opts := &oswin.NewWindowOptions{
		Title: title, Size: image.Point{width, height}, StdPixels: true,
	}
	wgp := WinGeomMgr.Pref(name, nil)
	if wgp != nil {
		WinGeomMgr.SettingStart()
		opts.Size = wgp.Size()
		opts.Pos = wgp.Pos()
		opts.StdPixels = false
		// fmt.Printf("got prefs for %v: size: %v pos: %v\n", name, opts.Size, opts.Pos)
		if _, found := AllWindows.FindName(name); found { // offset from existing
			opts.Pos.X += 20
			opts.Pos.Y += 20
		}
	}
	win := NewWindow(name, title, opts)
	WinGeomMgr.SettingEnd()
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
	vp.Fill = true
	vp.AddStyler(StylerParent(win), func() {
		vp.Style.BackgroundColor.SetColor(ColorScheme.Background)
		vp.Style.Color = ColorScheme.OnBackground // everything inherits this
	})

	win.AddChild(vp)
	win.Viewport = vp
	vp.Win = win
	win.ConfigVLay()
	WinNewCloseStamp()
	return win
}

// RecycleMainWindow looks for existing window with same Data --
// if found brings that to the front, returns true for bool.
// else (and if data is nil) calls NewDialogWin, and returns false.
func RecycleMainWindow(data any, name, title string, width, height int) (*Window, bool) {
	if data == nil {
		return NewMainWindow(name, title, width, height), false
	}
	ew, has := MainWindows.FindData(data)
	if has {
		if WinEventTrace {
			fmt.Printf("Win: %v getting recycled based on data match\n", ew.Nm)
		}
		ew.OSWin.Raise()
		return ew, true
	}
	nw := NewMainWindow(name, title, width, height)
	nw.Data = data
	return nw, false
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
	wgp := WinGeomMgr.Pref(name, nil)
	if wgp != nil {
		WinGeomMgr.SettingStart()
		opts.Size = wgp.Size()
		opts.Pos = wgp.Pos()
		opts.StdPixels = false
	}
	win := NewWindow(name, title, opts)
	WinGeomMgr.SettingEnd()
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

// RecycleDialogWin looks for existing window with same Data --
// if found brings that to the front, returns true for bool.
// else (and if data is nil) calls [NewDialogWin], and returns false.
func RecycleDialogWin(data any, name, title string, width, height int, modal bool) (*Window, bool) {
	if data == nil {
		return NewDialogWin(name, title, width, height, modal), false
	}
	ew, has := DialogWindows.FindData(data)
	if has {
		if WinEventTrace {
			fmt.Printf("Win: %v getting recycled based on data match\n", ew.Nm)
		}
		ew.OSWin.Raise()
		return ew, true
	}
	nw := NewDialogWin(name, title, width, height, modal)
	nw.Data = data
	return nw, false
}

// ConfigVLay creates and configures the vertical layout as first child of
// Viewport, and installs MainMenu as first element of layout.
func (w *Window) ConfigVLay() {
	vp := w.Viewport
	updt := vp.UpdateStart()
	defer vp.UpdateEnd(updt)
	if !vp.HasChildren() {
		vp.AddNewChild(TypeLayout, "main-vlay")
	}
	w.MasterVLay = vp.Child(0).Embed(TypeLayout).(*Layout)
	if !w.MasterVLay.HasChildren() {
		w.MasterVLay.AddNewChild(TypeMenuBar, "main-menu")
	}
	w.MasterVLay.Lay = LayoutVert
	w.MainMenu = w.MasterVLay.Child(0).(*MenuBar)
	w.MainMenu.MainMenu = true
	w.MainMenu.SetStretchMaxWidth()
}

// ConfigInsets updates the padding on the main layout of the window
// to the inset values provided by the OSWin window.
func (w *Window) ConfigInsets() {
	mainVlay, ok := w.Viewport.ChildByName("main-vlay", 0).(*Layout)
	if ok {
		insets := w.OSWin.Insets()
		mainVlay.AddStyler(StylerParent(w), func() {
			mainVlay.Style.Padding.Set(
				units.Dot(insets.Top),
				units.Dot(insets.Right),
				units.Dot(insets.Bottom),
				units.Dot(insets.Left),
			)
		})
	}

}

// AddMainMenu installs MainMenu as first element of main layout
// used for dialogs that don't always have a main menu -- returns
// menubar -- safe to call even if there is a menubar
func (w *Window) AddMainMenu() *MenuBar {
	vp := w.Viewport
	updt := vp.UpdateStart()
	defer vp.UpdateEnd(updt)
	if !vp.HasChildren() {
		vp.AddNewChild(TypeLayout, "main-vlay")
	}
	w.MasterVLay = vp.Child(0).Embed(TypeLayout).(*Layout)
	if !w.MasterVLay.HasChildren() {
		w.MainMenu = w.MasterVLay.AddNewChild(TypeMenuBar, "main-menu").(*MenuBar)
	} else {
		mmi := w.MasterVLay.ChildByName("main-menu", 0)
		if mmi != nil {
			mm := mmi.(*MenuBar)
			w.MainMenu = mm
			return mm
		}
	}
	w.MainMenu = w.MasterVLay.InsertNewChild(TypeMenuBar, 0, "main-menu").(*MenuBar)
	w.MainMenu.MainMenu = true
	w.MainMenu.SetStretchMaxWidth()
	return w.MainMenu
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
	cmw := w.MasterVLay.Child(1)
	if cmw != mw {
		w.MasterVLay.DeleteChildAtIndex(1, ki.DestroyKids)
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
	cmw := w.MasterVLay.Child(1)
	if ki.Type(cmw) != typ {
		w.MasterVLay.DeleteChildAtIndex(1, ki.DestroyKids)
		return w.MasterVLay.InsertNewChild(typ, 1, name)
	}
	return cmw
}

// SetMainFrame sets the main widget of this window as a Frame, with a default
// column-wise vertical layout and max stretch sizing, and returns that frame.
func (w *Window) SetMainFrame() *Frame {
	fr := w.SetMainWidgetType(TypeFrame, "main-frame").(*Frame)
	fr.Lay = LayoutVert
	fr.SetStretchMax()
	return fr
}

// MainFrame returns the main widget for this window as a Frame
// returns error if not there, or not a frame.
func (w *Window) MainFrame() (*Frame, error) {
	kw, err := w.MainWidget()
	if err != nil {
		return nil, err
	}
	mf, ok := kw.(*Frame)
	if ok {
		return mf, nil
	}
	return nil, fmt.Errorf("Main Widget is not a Frame for Window: %s", w.Nm)
}

// SetMainLayout sets the main widget of this window as a Layout, with a default
// column-wise vertical layout and max stretch sizing, and returns it.
func (w *Window) SetMainLayout() *Layout {
	fr := w.SetMainWidgetType(TypeLayout, "main-lay").(*Layout)
	fr.Lay = LayoutVert
	fr.SetStretchMax()
	return fr
}

// SetName sets name of this window and also the OSWin, and applies any window
// geometry settings associated with the new name if it is different from before
func (w *Window) SetName(name string) {
	curnm := w.Name()
	isdif := curnm != name
	w.NodeBase.SetName(name)
	if w.OSWin != nil {
		w.OSWin.SetName(name)
	}
	if isdif {
		for i, fw := range FocusWindows { // rename focus windows so we get focus later..
			if fw == curnm {
				FocusWindows[i] = name
			}
		}
	}
	if isdif && w.OSWin != nil {
		wgp := WinGeomMgr.Pref(name, w.OSWin.Screen())
		if wgp != nil {
			WinGeomMgr.SettingStart()
			if w.OSWin.Size() != wgp.Size() || w.OSWin.Position() != wgp.Pos() {
				if WinGeomTrace {
					log.Printf("WinGeomPrefs: SetName setting geom for window: %v pos: %v size: %v\n", w.Name(), wgp.Pos(), wgp.Size())
				}
				w.OSWin.SetGeom(wgp.Pos(), wgp.Size())
				oswin.TheApp.SendEmptyEvent()
			}
			WinGeomMgr.SettingEnd()
		}
	}
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
// MasterVLay -- returns error if not yet set.
func (w *Window) MainWidget() (ki.Ki, error) {
	return w.MasterVLay.ChildTry(1)
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
	sc := w.OSWin.Screen()
	if sc == nil {
		sc = oswin.TheApp.Screen(0)
	}
	pdpi := sc.PhysicalDPI
	// ldpi = pdpi * zoom * ldpi
	cldpinet := sc.LogicalDPI
	cldpi := cldpinet / oswin.ZoomFactor
	nldpinet := cldpinet + float32(6*steps)
	if nldpinet < 6 {
		nldpinet = 6
	}
	oswin.ZoomFactor = nldpinet / cldpi
	Prefs.ApplyDPI()
	fmt.Printf("Effective LogicalDPI now: %v  PhysicalDPI: %v  Eff LogicalDPIScale: %v  ZoomFactor: %v\n", nldpinet, pdpi, nldpinet/pdpi, oswin.ZoomFactor)
	w.FullReRender()
}

// WinViewport2D returns the viewport directly under this window that serves
// as the master viewport for the entire window.
func (w *Window) WinViewport2D() *Viewport2D {
	vpi := w.ChildByType(TypeViewport2D, ki.Embeds, 0)
	if vpi == nil { // shouldn't happen
		return nil
	}
	vp, _ := vpi.Embed(TypeViewport2D).(*Viewport2D)
	return vp
}

// SetWinSize requests that the window be resized to the given size
// in OS window manager specific coordinates, which may be different
// from the underlying pixel-level resolution of the window.
// This will trigger a resize event and be processed
// that way when it occurs.
func (w *Window) SetWinSize(sz image.Point) {
	w.OSWin.SetWinSize(sz)
}

// SetSize requests that the window be resized to the given size
// in underlying pixel coordinates, which means that the requested
// size is divided by the screen's DevicePixelRatio
func (w *Window) SetSize(sz image.Point) {
	w.OSWin.SetSize(sz)
}

// IsResizing means the window is actively being resized by user -- don't try
// to update otherwise
func (w *Window) IsResizing() bool {
	return w.HasFlag(int(WinFlagIsResizing))
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
func (w *Window) Resized(sz image.Point) {
	if !w.IsVisible() {
		return
	}
	curSz := w.Viewport.Geom.Size

	if curSz == sz {
		if WinEventTrace {
			fmt.Printf("Win: %v skipped same-size Resized: %v\n", w.Nm, curSz)
		}
		return
	}
	drw := w.OSWin.Drawer()
	if drw.Impl.MaxTextures != vgpu.MaxTexturesPerSet*3 { // this is essential after hibernate
		drw.SetMaxTextures(vgpu.MaxTexturesPerSet * 3) // use 3 sets
	}
	w.FocusInactivate()
	w.InactivateAllSprites()
	w.UpMu.Lock()
	if !w.IsVisible() {
		if WinEventTrace {
			fmt.Printf("Win: %v Resized already closed\n", w.Nm)
		}
		w.UpMu.Unlock()
		return
	}
	if WinEventTrace {
		fmt.Printf("Win: %v Resized from: %v to: %v\n", w.Nm, curSz, sz)
	}
	if curSz == (image.Point{}) { // first open
		StringsInsertFirstUnique(&FocusWindows, w.Nm, 10)
	}
	w.Viewport.Resize(sz)
	w.ConfigInsets()
	if WinGeomTrace {
		log.Printf("WinGeomPrefs: recording from Resize\n")
	}
	WinGeomMgr.RecordPref(w)
	w.UpMu.Unlock()
	w.FullReRender()
	// we need a second full re-render for fullscreen and snap resizes on Windows
	if oswin.TheApp.Platform() == oswin.Windows {
		w.FullReRender()
	}
}

// Raise requests that the window be at the top of the stack of windows,
// and receive focus.  If it is iconified, it will be de-iconified.  This
// is the only supported mechanism for de-iconifying.
func (w *Window) Raise() {
	w.OSWin.Raise()
}

// Minimize requests that the window be iconified, making it no longer
// visible or active -- rendering should not occur for minimized windows.
func (w *Window) Minimize() {
	w.OSWin.Minimize()
}

// Close closes the window -- this is not a request -- it means:
// definitely close it -- flags window as such -- check IsClosing()
func (w *Window) Close() {
	if w.IsClosing() {
		return
	}
	// this causes hangs etc: not good
	// w.UpMu.Lock() // allow other stuff to finish
	w.SetFlag(int(WinFlagIsClosing))
	// w.UpMu.Unlock()
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
	WindowGlobalMu.Lock()
	StringsDelete(&FocusWindows, w.Name())
	WindowGlobalMu.Unlock()
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
	w.SetDisabled() // marks as closed
	w.FocusInactivate()
	WindowGlobalMu.Lock()
	if len(FocusWindows) > 0 {
		pf := FocusWindows[0]
		WindowGlobalMu.Unlock()
		pfw, has := AllWindows.FindName(pf)
		if has {
			if WinEventTrace {
				fmt.Printf("Win: %v getting restored focus after: %v closed\n", pfw.Nm, w.Nm)
			}
			pfw.OSWin.Raise()
		} else {
			if WinEventTrace {
				fmt.Printf("Win: %v not found to restored focus: %v closed\n", pf, w.Nm)
			}
		}
	} else {
		WindowGlobalMu.Unlock()
	}
	// these are managed by the window itself
	w.Sprites.Reset()
	w.UpMu.Unlock()
}

// IsClosed reports if the window has been closed
func (w *Window) IsClosed() bool {
	if w.IsDisabled() || w.Viewport == nil {
		return true
	}
	return false
}

// SetCloseReqFunc sets the function that is called whenever there is a
// request to close the window (via a OS or a call to CloseReq() method).  That
// function can then adjudicate whether and when to actually call Close.
func (w *Window) SetCloseReqFunc(fun func(win *Window)) {
	w.OSWin.SetCloseReqFunc(func(owin oswin.Window) {
		fun(w)
	})
}

// SetCloseCleanFunc sets the function that is called whenever window is
// actually about to close (irrevocably) -- can do any necessary
// last-minute cleanup here.
func (w *Window) SetCloseCleanFunc(fun func(win *Window)) {
	w.OSWin.SetCloseCleanFunc(func(owin oswin.Window) {
		fun(w)
	})
}

// IsVisible is the main visibility check -- don't do any window updates if not visible!
func (w *Window) IsVisible() bool {
	if w == nil || w.This() == nil || w.OSWin == nil || w.IsClosed() || w.IsClosing() || !w.OSWin.IsVisible() {
		return false
	}
	return true
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
		oswin.InitScreenLogicalDPIFunc = Prefs.ApplyDPI // called when screens are initialized
		TheViewIFace.HiStyleInit()
		WinGeomMgr.NeedToReload() // gets time stamp associated with open, so it doesn't re-open
		WinGeomMgr.Open()
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
	w.SetFlag(int(WinFlagDoFullRender))
	w.EventLoop()
}

// GoStartEventLoop starts the event processing loop for this window in a new
// goroutine, and returns immediately.  Adds to WinWait waitgroup so a main
// thread can wait on that for all windows to close.
func (w *Window) GoStartEventLoop() {
	WinWait.Add(1)
	w.SetFlag(int(WinFlagDoFullRender), int(WinFlagGoLoop))
	go w.EventLoop()
}

// StopEventLoop tells the event loop to stop running when the next event arrives.
func (w *Window) StopEventLoop() {
	w.SetFlag(int(WinFlagStopEventLoop))
}

// SendCustomEvent sends a custom event with given data to this window -- widgets can connect
// to receive CustomEventType events to receive them.  Sometimes it is useful
// to send a custom event just to trigger a pass through the event loop, even
// if nobody is listening (e.g., if a popup is posted without a surrounding
// event, as in Complete.ShowCompletions
func (w *Window) SendCustomEvent(data any) {
	oswin.SendCustomEvent(w.OSWin, data)
}

/////////////////////////////////////////////////////////////////////////////
//                   Rendering

// SendShowEvent sends the WindowShowEvent to anyone listening -- only sent once..
func (w *Window) SendShowEvent() {
	if w.HasFlag(int(WinFlagSentShow)) {
		return
	}
	w.SetFlag(int(WinFlagSentShow))
	se := window.ShowEvent{}
	se.Action = window.Show
	se.Init()
	w.EventMgr.SendEventSignal(&se, Popups)
}

// SendWinFocusEvent sends the WindowFocusEvent to widgets
func (w *Window) SendWinFocusEvent(act window.Actions) {
	se := window.FocusEvent{}
	se.Action = act
	se.Init()
	w.EventMgr.SendEventSignal(&se, Popups)
}

// FullReRender performs a full re-render of the window -- each node renders
// into its viewport, aggregating into the main window viewport, which will
// drive an UploadAllViewports call after all the rendering is done, and
// signal the publishing of the window after that
func (w *Window) FullReRender() {
	if !w.IsVisible() {
		return
	}
	if WinEventTrace {
		fmt.Printf("Win: %v FullReRender (w.Viewport.SetNeedsFullRender)\n", w.Nm)
	}
	w.Viewport.SetNeedsFullRender()
	w.InitialFocus()
}

// InitialFocus establishes the initial focus for the window if no focus
// is set -- uses ActivateStartFocus or FocusNext as backup.
func (w *Window) InitialFocus() {
	w.EventMgr.InitialFocus()
	if prof.Profiling {
		now := time.Now()
		opent := now.Sub(WindowOpenTimer)
		fmt.Printf("Win: %v took: %v to open\n", w.Nm, opent)
	}
}

// UploadVpRegion uploads image for one viewport region on the screen, using
// vpBBox bounding box for the viewport, and winBBox bounding box for the
// window -- called after re-rendering specific nodes to update only the
// relevant part of the overall viewport image
func (w *Window) UploadVpRegion(vp *Viewport2D, vpBBox, winBBox image.Rectangle) {
	// fmt.Printf("win upload vpbox: %v  winbox: %v\n", vpBBox, winBBox)
	winrel := winBBox.Min.Sub(vpBBox.Min)
	if !w.IsVisible() {
		return
	}
	vpPixBB := vp.Pixels.Rect
	inBB := vpBBox.Intersect(vpPixBB)
	if inBB.Empty() {
		return
	}
	vpBBox = inBB
	winBBox = vpBBox.Add(winrel) // fix again
	w.UpMu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.SetWinUpdating()
	// pr := prof.Start("win.UploadVpRegion")

	idx, over := w.UpdtRegs.Add(winBBox, vp)
	if over {
		w.ResetUpdateRegionsImpl()
		if Render2DTrace || WinEventTrace {
			fmt.Printf("Win: %v region Vp %v, winbbox: %v reset updates\n", w.Path(), vp.Path(), winBBox)
		}
	} else {
		drw := w.OSWin.Drawer()
		vp.Pixels.Rect = vpBBox
		drw.SetGoImage(idx, 0, vp.Pixels, vgpu.NoFlipY)
		vp.Pixels.Rect = vpPixBB
		if Render2DTrace || WinEventTrace {
			fmt.Printf("Win: %v uploaded region Vp %v, winbbox: %v to index: %d\n", w.Path(), vp.Path(), winBBox, idx)
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
	w.UpMu.Unlock()
}

// UploadVp uploads entire viewport image for given viewport -- e.g., for
// popups etc updating separately
func (w *Window) UploadVp(vp *Viewport2D, offset image.Point) {
	vpr := vp.Pixels.Bounds()
	winBBox := vpr.Add(offset)
	if !w.IsVisible() {
		return
	}
	w.UpMu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.SetWinUpdating()
	updt := w.UpdateStart()
	idx := 0
	drw := w.OSWin.Drawer()
	if vp == w.Viewport {
		drw.SetGoImage(idx, 0, vp.Pixels, vgpu.NoFlipY)
	} else {
		// pr := prof.Start("win.UploadVp")
		gii, _ := KiToNode2D(vp.This())
		if gii != nil {
			idx, _ = w.PopDraws.Add(gii, winBBox)
			drw.SetGoImage(idx, 0, vp.Pixels, vgpu.NoFlipY)
		}
	}
	if Render2DTrace || WinEventTrace {
		fmt.Printf("Win: %v uploaded entire Vp %v, winbbox: %v to index: %d\n", w.Path(), vp.Path(), winBBox, idx)
	}
	w.ClearWinUpdating()
	w.UpMu.Unlock()
	w.UpdateEnd(updt) // drives publish
}

// UploadAllViewports does a complete upload of all active viewports, in the
// proper order, so as to completely refresh the window texture based on
// everything rendered
func (w *Window) UploadAllViewports() {
	if !w.IsVisible() {
		return
	}
	w.UpMu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.SetWinUpdating()
	// pr := prof.Start("win.UploadAllViewports")
	updt := w.UpdateStart()
	if Render2DTrace || WinEventTrace {
		fmt.Printf("Win: %v uploading full Vp, image bound: %v, updt: %v\n", w.Path(), w.Viewport.Pixels.Bounds(), updt)
	}
	w.PopMu.Lock()
	w.ResetUpdateRegionsImpl()
	w.PopMu.Unlock()

	// fmt.Printf("upload all views pop unlocked: %v\n", w.Nm)
	// pr.End()
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

// ResetUpdateRegions resets any existing window update regions and
// grabs the current state of the window viewport for subsequent rendering.
// This is protected by update mutex and suitable for random calls eg.,
// when a window is scrolling and positions are then invalid.
func (w *Window) ResetUpdateRegions() {
	w.UpMu.Lock() // block all updates while we publish
	w.ResetUpdateRegionsImpl()
	w.UpMu.Unlock()
}

// ResetUpdateRegionsImpl resets any existing window update regions and
// grabs the current state of the window and popup viewports for subsequent rendering.
func (w *Window) ResetUpdateRegionsImpl() {
	w.UpdtRegs.Reset()
	w.PopDraws.Reset()
	drw := w.OSWin.Drawer()
	drw.SetGoImage(0, 0, w.Viewport.Pixels, vgpu.NoFlipY)
	// then all the current popups
	// fmt.Printf("upload all views pop locked: %v\n", w.Nm)
	if w.PopupStack != nil {
		for _, pop := range w.PopupStack {
			gii, _ := KiToNode2D(pop)
			if gii != nil {
				vp := gii.AsViewport2D()
				r := vp.Geom.Bounds()
				idx, _ := w.PopDraws.Add(gii, vp.WinBBox)
				drw.SetGoImage(idx, 0, vp.Pixels, vgpu.NoFlipY)
				if Render2DTrace {
					fmt.Printf("Win: %v uploading popup stack Vp %v, win pos: %v, vp bounds: %v  idx: %d\n", w.Path(), vp.Path(), r.Min, vp.Pixels.Bounds(), idx)
				}
			}
		}
	}
	if w.Popup != nil {
		gii, _ := KiToNode2D(w.Popup)
		if gii != nil {
			vp := gii.AsViewport2D()
			r := vp.Geom.Bounds()
			idx, _ := w.PopDraws.Add(gii, vp.WinBBox)
			drw.SetGoImage(idx, 0, vp.Pixels, vgpu.NoFlipY)
			if Render2DTrace || WinEventTrace {
				fmt.Printf("Win: %v uploading top popup Vp %v, win pos: %v, vp bounds: %v  idx: %d\n", w.Path(), vp.Path(), r.Min, vp.Pixels.Bounds(), idx)
			}
		}
	}
}

// Publish does the final step of updating of the window based on the current
// texture (and overlay texture if active)
func (w *Window) Publish() {
	if !w.IsVisible() || w.OSWin.IsMinimized() {
		if WinEventTrace {
			fmt.Printf("skipping update on inactive / minimized window: %v\n", w.Nm)
		}
		return
	}
	w.UpMu.Lock()       // block all updates while we publish
	if !w.IsVisible() { // could have closed while we waited for lock
		if WinEventTrace {
			fmt.Printf("skipping update on inactive / minimized window: %v\n", w.Nm)
		}
		w.UpMu.Unlock()
		return
	}

	// note: this is key for finding redundant updates!
	if WinPublishTrace {
		fmt.Printf("\n\n###################################\n%v\n", string(debug.Stack()))
	}

	w.SetWinUpdating()
	if Update2DTrace || WinEventTrace {
		fmt.Printf("Win %v doing publish\n", w.Nm)
	}
	// pr := prof.Start("win.Publish")

	// note: vulkan complains about different threads for rendering but should be ok.
	// can't use RunOnWin method because it locks for main thread windows.
	// w.OSWin.RunOnWin(func() {})
	// and using RunOnMain makes the thing hella slow -- like opengl -- that was the issue there!
	// oswin.TheApp.RunOnMain(func() {

	if w.Sprites.Modified || w.Sprites.HasSizeChanged() {
		w.ConfigSprites()
		w.Sprites.Modified = false
	}

	drw := w.OSWin.Drawer()
	vpv := drw.GetImageVal(0).Texture
	if !vpv.IsActive() {
		if w.Viewport.Pixels == nil {
			if Update2DTrace {
				fmt.Printf("Win %s didn't have active image, viewport is nil\n", w.Nm)
			}
			w.ClearWinUpdating()
			w.UpMu.Unlock()
			return
		} else {
			if Update2DTrace {
				fmt.Printf("Win %s didn't have active image, setting to: %v\n", w.Nm, w.Viewport.Pixels.Bounds())
			}
			drw.SetGoImage(0, 0, w.Viewport.Pixels, vgpu.NoFlipY)
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
		var clrs [16]gist.Color
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

	// 	if Render2DTrace {
	// 		fmt.Printf("Win %v did publish\n", w.Nm)
	// 	}
	// pr.End()

	w.ClearWinUpdating()
	w.UpMu.Unlock()
}

// SignalWindowPublish is the signal receiver function that publishes the
// window updates when the window update signal (UpdateEnd) occurs
func SignalWindowPublish(winki, node ki.Ki, sig int64, data any) {
	win := winki.Embed(TypeWindow).(*Window)
	if WinEventTrace || Render2DTrace {
		fmt.Printf("Win: %v publishing image due to signal: %v from node: %v\n", win.Path(), ki.NodeSignals(sig), node.Path())
	}
	if !win.IsVisible() || win.IsWinUpdating() { // win.IsResizing() ||
		if WinEventTrace || Render2DTrace {
			fmt.Printf("not updating as invisible or already updating\n")
		}
		return
	}
	win.Publish()
}

// AddDirectUploader adds given node to those that have a DirectWinUpload method
// and directly render to the Drawer Texture via their own method.
// This is for gi3d.Scene for example.  Returns the index of the image to upload to.
func (w *Window) AddDirectUploader(node Node2D) int {
	w.UpMu.Lock()
	idx, _ := w.DirDraws.Add(node, image.Rectangle{})
	w.UpMu.Unlock()
	return idx
}

// DeleteDirectUploader removes given node to those that have a DirectWinUpload method.
func (w *Window) DeleteDirectUploader(node Node2D) {
	w.UpMu.Lock()
	w.DirDraws.Nodes.DeleteKey(node.AsGiNode())
	w.UpMu.Unlock()
}

/////////////////////////////////////////////////////////////////////////////
//                   Sprites

// SpriteByName returns a sprite by name -- false if not created yet
func (w *Window) SpriteByName(nm string) (*Sprite, bool) {
	w.UpMu.Lock()
	defer w.UpMu.Unlock()
	return w.Sprites.SpriteByName(nm)
}

// AddSprite adds an existing sprite to list of sprites, using the sprite.Name
// as the unique name key.
func (w *Window) AddSprite(sp *Sprite) {
	w.UpMu.Lock()
	defer w.UpMu.Unlock()
	w.Sprites.Add(sp)
	if sp.On {
		w.Sprites.Active++
	}
}

// ActivateSprite flags the sprite as active, and increments
// number of Active Sprites, so that it will actually be rendered.
// it is assumed that the image has not changed.
func (w *Window) ActivateSprite(nm string) {
	w.UpMu.Lock()
	defer w.UpMu.Unlock()

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
func (w *Window) InactivateSprite(nm string) {
	w.UpMu.Lock()
	defer w.UpMu.Unlock()

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
func (w *Window) InactivateAllSprites() {
	w.UpMu.Lock()
	defer w.UpMu.Unlock()

	for _, sp := range w.Sprites.Names.Order {
		if sp.Val.On {
			sp.Val.On = false
			w.Sprites.Active--
		}
	}
}

// DeleteSprite deletes given sprite, returns true if actually deleted.
// requires updating other sprites of same size -- use Inactivate if any chance of re-use.
func (w *Window) DeleteSprite(nm string) bool {
	w.UpMu.Lock()
	defer w.UpMu.Unlock()

	sp, ok := w.Sprites.SpriteByName(nm)
	if !ok {
		return false
	}
	w.Sprites.Delete(sp)
	w.Sprites.Active--
	return true
}

// SpriteEvent processes given event for any active sprites
func (w *Window) SelSpriteEvent(evi oswin.Event) {
	// w.UpMu.Lock()
	// defer w.UpMu.Unlock()

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
		if et == oswin.MouseDragEvent {
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
func (w *Window) ConfigSprites() {
	drw := w.OSWin.Drawer()
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
func (w *Window) DrawSprites() {
	drw := w.OSWin.Drawer()
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

/////////////////////////////////////////////////////////////////////////////
//                   MainMenu Updating

// MainMenuUpdated needs to be called whenever the main menu for this window
// is updated in terms of items added or removed.
func (w *Window) MainMenuUpdated() {
	if w == nil || w.MainMenu == nil || !w.IsVisible() {
		return
	}
	w.UpMu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.MainMenu.UpdateMainMenu(w) // main update menu call, in bars.go for MenuBar
	w.UpMu.Unlock()
}

// MainMenuUpdateActives needs to be called whenever items on the main menu
// for this window have their IsActive status updated.
func (w *Window) MainMenuUpdateActives() {
	if w == nil || w.MainMenu == nil || !w.IsVisible() {
		return
	}
	w.UpMu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	w.MainMenu.MainMenuUpdateActives(w) // also in bars.go for MenuBar
	w.UpMu.Unlock()
}

// MainMenuUpdateWindows updates a Window menu with a list of active menus.
func (w *Window) MainMenuUpdateWindows() {
	if w == nil || w.MainMenu == nil || !w.IsVisible() {
		return
	}
	w.UpMu.Lock()
	if !w.IsVisible() { // could have closed while we waited for lock
		w.UpMu.Unlock()
		return
	}
	WindowGlobalMu.Lock()
	wmeni := w.MainMenu.ChildByName("Window", 3)
	if wmeni == nil {
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

// PollEvents first tells the main event loop to check for any gui events now
// and then it runs the event processing loop for the Window as long
// as there are events to be processed, and then returns.
func (w *Window) PollEvents() {
	oswin.TheApp.PollEvents()
	for {
		evi, has := w.OSWin.PollEvent()
		if !has {
			break
		}
		w.ProcessEvent(evi)
	}
}

// EventLoop runs the event processing loop for the Window -- grabs oswin
// events for the window and dispatches them to receiving nodes, and manages
// other state etc (popups, etc).
func (w *Window) EventLoop() {
	for {
		if w.HasFlag(int(WinFlagStopEventLoop)) {
			w.ClearFlag(int(WinFlagStopEventLoop))
			break
		}
		evi := w.OSWin.NextEvent()
		if w.HasFlag(int(WinFlagStopEventLoop)) {
			w.ClearFlag(int(WinFlagStopEventLoop))
			break
		}
		w.ProcessEvent(evi)
	}
	if WinEventTrace {
		fmt.Printf("Win: %v out of event loop\n", w.Nm)
	}
	if w.HasFlag(int(WinFlagGoLoop)) {
		WinWait.Done()
	}
	// our last act must be self destruction!
	w.This().Destroy()
}

// ProcessEvent processes given oswin.Event
func (w *Window) ProcessEvent(evi oswin.Event) {
	et := evi.Type()
	// log.Printf("Got event: %v\n", et)
	w.delPop = false                     // if true, delete this popup after event loop
	if et > oswin.EventTypeN || et < 0 { // we don't handle other types of events here
		fmt.Printf("Win: %v got out-of-range event: %v\n", w.Nm, et)
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
	if FilterLaggyKeyEvents || et != oswin.KeyEvent { // don't filter key events
		if !w.FilterEvent(evi) {
			return
		}
	}
	w.EventMgr.LagLastSkipped = false
	w.lastEt = et

	if w.skippedResize != nil {
		w.Viewport.BBoxMu.RLock()
		vpsz := w.Viewport.Geom.Size
		w.Viewport.BBoxMu.RUnlock()
		if vpsz != w.OSWin.Size() {
			w.SetFlag(int(WinFlagIsResizing))
			w.Resized(w.OSWin.Size())
			w.skippedResize = nil
		}
	}

	if et != oswin.WindowResizeEvent && et != oswin.WindowPaintEvent {
		w.ClearFlag(int(WinFlagIsResizing))
	}

	w.EventMgr.MouseEvents(evi)

	if !w.HiPriorityEvents(evi) {
		return
	}

	////////////////////////////////////////////////////////////////////////////
	// Send Events to Widgets

	hasFocus := w.HasFlag(int(WinFlagGotFocus))
	if _, ok := evi.(*mouse.ScrollEvent); ok {
		if !hasFocus {
			w.EventMgr.Scrolling = nil // not valid
		}
		hasFocus = true // doesn't need focus!
	}
	if me, ok := evi.(*mouse.MoveEvent); ok {
		hasFocus = true // also doesn't need focus (there can be hover events while not focused)
		w.SetCursor(me.Event.Pos())
	}

	if (hasFocus || !evi.OnWinFocus()) && !evi.IsProcessed() {
		evToPopup := !w.CurPopupIsTooltip() // don't send events to tooltips!
		w.EventMgr.SendEventSignal(evi, evToPopup)
		if !w.delPop && et == oswin.MouseMoveEvent && !evi.IsProcessed() {
			didFocus := w.EventMgr.GenMouseFocusEvents(evi.(*mouse.MoveEvent), evToPopup)
			if didFocus && w.CurPopupIsTooltip() {
				w.delPop = true
			}
		}
	}

	////////////////////////////////////////////////////////////////////////////
	// Low priority windows events

	if !evi.IsProcessed() && et == oswin.KeyChordEvent {
		ke := evi.(*key.ChordEvent)
		kc := ke.Chord()
		if w.TriggerShortcut(kc) {
			evi.SetProcessed()
		}
	}

	if !evi.IsProcessed() {
		switch e := evi.(type) {
		case *key.ChordEvent:
			keyDelPop := w.KeyChordEventLowPri(e)
			if keyDelPop {
				w.delPop = true
			}
		}
	}

	w.EventMgr.MouseEventReset(evi)
	if evi.Type() == oswin.MouseEvent {
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
				if et != oswin.MouseMoveEvent {
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
				if fsz > 0 && et == oswin.KeyChordEvent {
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

// SetCursor sets the cursor based on the given mouse position
func (w *Window) SetCursor(pos image.Point) {
	maxLevel := 0
	maxLevelCursor := cursor.Arrow
	maxLevelWidget := &WidgetBase{}

	fun := func(k ki.Ki, level int, data any) bool {
		_, ni := KiToNode2D(k)
		if ni == nil {
			// could have nodes further down (eg with menu which is ki.Slice), so continue
			return ki.Continue
		}
		if !ni.PosInWinBBox(pos) {
			// however, if we are out of bbox, there is no way to get back in
			return ki.Break
		}
		if !ni.IsVisible() || level < maxLevel {
			// could have visible or higher level ones further down
			return ki.Continue
		}

		wb, ok := ni.This().Embed(TypeWidgetBase).(*WidgetBase)
		if !ok {
			// same logic as with Node2D
			return ki.Continue
		}
		maxLevel = level
		maxLevelCursor = wb.Style.Cursor
		maxLevelWidget = wb
		return ki.Continue
	}

	pop := w.CurPopup()
	if pop == nil {
		// if no popup, just do on window
		w.FuncDownMeFirst(0, nil, fun)
	} else {
		_, popni := KiToNode2D(pop)
		if popni == nil || !popni.PosInWinBBox(pos) || PopupIsTooltip(pop) {
			// if not in popup (or it is a tooltip), do on window
			w.FuncDownMeFirst(0, nil, fun)
		} else {
			// if in popup, do on popup
			popni.FuncDownMeFirst(0, nil, fun)
		}

	}
	oswin.TheApp.Cursor(w.OSWin).Set(maxLevelCursor)

	if w.IsInSelectionMode() {
		if maxLevelWidget != nil {
			w.SelectedWidget <- maxLevelWidget
		}
	}
}

// FilterEvent filters repeated laggy events -- key for responsive resize, scroll, etc
// returns false if event should not be processed further, and true if it should.
func (w *Window) FilterEvent(evi oswin.Event) bool {
	et := evi.Type()

	if w.HasFlag(int(WinFlagGotPaint)) && et == oswin.WindowPaintEvent && w.lastEt == oswin.WindowResizeEvent {
		if WinEventTrace {
			fmt.Printf("Win: %v skipping paint after resize\n", w.Nm)
		}
		w.Publish() // this is essential on mac for any paint event
		w.SetFlag(int(WinFlagGotPaint))
		return false // X11 always sends a paint after a resize -- we just use resize
	}

	if et != w.lastEt && w.lastEt != oswin.WindowResizeEvent {
		return true // non-repeat
	}

	if et == oswin.WindowResizeEvent {
		now := time.Now()
		lag := now.Sub(evi.Time())
		lagMs := int(lag / time.Millisecond)
		w.SetFlag(int(WinFlagIsResizing))
		we := evi.(*window.Event)
		// fmt.Printf("resize\n")
		if lagMs > EventSkipLagMSec {
			if WinEventTrace {
				fmt.Printf("Win: %v skipped et %v lag %v size: %v\n", w.Nm, et, lag, w.OSWin.Size())
			}
			w.EventMgr.LagLastSkipped = true
			w.skippedResize = we
			return false
		} else {
			we.SetProcessed()
			w.Resized(w.OSWin.Size())
			w.EventMgr.LagLastSkipped = false
			w.skippedResize = nil
			return false
		}
	}
	return w.EventMgr.FilterLaggyEvents(evi)
}

// HiProrityEvents processes High-priority events for Window.
// Window gets first crack at these events, and handles window-specific ones
// returns true if processing should continue and false if was handled
func (w *Window) HiPriorityEvents(evi oswin.Event) bool {
	switch e := evi.(type) {
	case *window.Event:
		switch e.Action {
		// case window.Resize: // note: already handled earlier in lag process
		case window.Close:
			// fmt.Printf("got close event for window %v \n", w.Nm)
			w.Closed()
			w.SetFlag(int(WinFlagStopEventLoop))
			return false
		case window.Minimize:
			// on mobile platforms, we need to set the size to 0 so that it detects a size difference
			// and lets the size event go through when we come back later
			if oswin.TheApp.Platform().IsMobile() {
				w.Viewport.Geom.Size = image.Point{}
			}
		case window.Paint:
			// fmt.Printf("got paint event for window %v \n", w.Nm)
			w.SetFlag(int(WinFlagGotPaint))
			if w.HasFlag(int(WinFlagDoFullRender)) {
				w.ClearFlag(int(WinFlagDoFullRender))
				// fmt.Printf("Doing full render at size: %v\n", w.Viewport.Geom.Size)
				if w.Viewport.Geom.Size != w.OSWin.Size() {
					w.Resized(w.OSWin.Size())
				} else {
					w.FullReRender() // note: this is currently needed for focus to actually
					// take effect in a popup, and also ensures dynamically sized elements are
					// properly sized.  It adds a bit of cost but really not that much and
					// probably worth it overall, even if we can fix the initial focus issue
					// w.InitialFocus()
				}
			}
			w.Publish()
		case window.Show:
			// note that this is sent delayed by vkos
			if WinEventTrace {
				fmt.Printf("Win: %v got show event\n", w.Nm)
			}
			if w.NeedWinMenuUpdate() {
				w.MainMenuUpdateWindows()
			}
			w.SendShowEvent() // happens AFTER full render
		case window.Move:
			e.SetProcessed()
			if w.HasFlag(int(WinFlagGotPaint)) { // moves before paint are not accurate on X11
				// fmt.Printf("win move: %v\n", w.OSWin.Position())
				if WinGeomTrace {
					log.Printf("WinGeomPrefs: recording from Move\n")
				}
				WinGeomMgr.RecordPref(w)
			}
		case window.Focus:
			StringsInsertFirstUnique(&FocusWindows, w.Nm, 10)
			if !w.HasFlag(int(WinFlagGotFocus)) {
				w.SetFlag(int(WinFlagGotFocus))
				w.SendWinFocusEvent(window.Focus)
				if WinEventTrace {
					fmt.Printf("Win: %v got focus\n", w.Nm)
				}
				if w.NeedWinMenuUpdate() {
					w.MainMenuUpdateWindows()
				}
			} else {
				if WinEventTrace {
					fmt.Printf("Win: %v got extra focus\n", w.Nm)
				}
			}
		case window.DeFocus:
			if WinEventTrace {
				fmt.Printf("Win: %v lost focus\n", w.Nm)
			}
			w.ClearFlag(int(WinFlagGotFocus))
			w.SendWinFocusEvent(window.DeFocus)
		case window.ScreenUpdate:
			w.Resized(w.OSWin.Size())
			// TODO: figure out how to restore this stuff without breaking window size on mobile

			// WinGeomMgr.AbortSave() // anything just prior to this is sus
			// if !oswin.TheApp.NoScreens() {
			// 	Prefs.UpdateAll()
			// 	WinGeomMgr.RestoreAll()
			// }
		}
		return false // don't do anything else!
	case *mouse.DragEvent:
		if w.EventMgr.DNDStage == DNDStarted {
			w.DNDMoveEvent(e)
		} else {
			w.SelSpriteEvent(evi)
			if !w.EventMgr.dragStarted {
				e.SetProcessed() // ignore
			}
		}
	case *mouse.Event:
		if w.EventMgr.DNDStage == DNDStarted && e.Action == mouse.Release {
			w.DNDDropEvent(e)
		}
		w.FocusActiveClick(e)
		w.SelSpriteEvent(evi)
		if w.NeedWinMenuUpdate() {
			w.MainMenuUpdateWindows()
		}
	case *mouse.MoveEvent:
		if bitflag.HasAllAtomic(&w.Flag, int(WinFlagGotPaint), int(WinFlagGotFocus)) {
			if w.HasFlag(int(WinFlagDoFullRender)) {
				w.ClearFlag(int(WinFlagDoFullRender))
				// if we are getting mouse input, and still haven't done this, do it..
				// fmt.Printf("Doing full render at size: %v\n", w.Viewport.Geom.Size)
				if w.Viewport.Geom.Size != w.OSWin.Size() {
					w.Resized(w.OSWin.Size())
				} else {
					w.FullReRender()
				}
				w.SendShowEvent() // happens AFTER full render
			}
			// if w.EventMgr.Focus == nil { // not using lock-protected b/c can conflict with popup
			w.EventMgr.ActivateStartFocus()
			// }
		}
	case *dnd.Event:
		if e.Action == dnd.External {
			w.EventMgr.DNDDropMod = e.Mod
		}
	case *key.ChordEvent:
		keyDelPop := w.KeyChordEventHiPri(e)
		if keyDelPop {
			w.delPop = true
		}
	}
	return true
}

/////////////////////////////////////////////////////////////////////////////
//                   Sending Events

// Most of event stuff is in events.go, controlled by EventMgr

func (w *Window) EventTopNode() ki.Ki {
	return w.This()
}

func (w *Window) FocusTopNode() ki.Ki {
	cpop := w.CurPopup()
	if cpop != nil {
		return cpop
	}
	return w.Viewport.This()
}

func (w *Window) EventTopUpdateStart() bool {
	return w.UpdateStart()
}

func (w *Window) EventTopUpdateEnd(updt bool) {
	w.UpdateEnd(updt)
}

// IsInScope returns true if the given object is in scope for receiving events.
// If popup is true, then only items on popup are in scope, otherwise
// items NOT on popup are in scope (if no popup, everything is in scope).
func (w *Window) IsInScope(k ki.Ki, popup bool) bool {
	cpop := w.CurPopup()
	if cpop == nil {
		return true
	}
	if k.This() == cpop {
		return popup
	}
	_, ni := KiToNode2D(k)
	if ni == nil {
		np := k.ParentByType(TypeNode2DBase, ki.Embeds)
		if np != nil {
			ni = np.Embed(TypeNode2DBase).(*Node2DBase)
		} else {
			return false
		}
	}
	mvp := ni.ViewportSafe()
	if mvp == nil {
		return false
	}
	if mvp.This() == cpop {
		return popup
	}
	return !popup
}

// AddShortcut adds given shortcut to given action.
func (w *Window) AddShortcut(chord key.Chord, act *Action) {
	if chord == "" {
		return
	}
	if w.Shortcuts == nil {
		w.Shortcuts = make(Shortcuts, 100)
	}
	sa, exists := w.Shortcuts[chord]
	if exists && sa != act && sa.Text != act.Text {
		if KeyEventTrace {
			log.Printf("gi.Window shortcut: %v already exists on action: %v -- will be overwritten with action: %v\n", chord, sa.Text, act.Text)
		}
	}
	w.Shortcuts[chord] = act
}

// DeleteShortcut deletes given shortcut
func (w *Window) DeleteShortcut(chord key.Chord, act *Action) {
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
	if sa.IsDisabled() {
		if KeyEventTrace {
			fmt.Printf("Shortcut chord: %v, action: %v -- is inactive, not fired\n", chord, sa.Text)
		}
		return false
	}

	if KeyEventTrace {
		fmt.Printf("Win: %v Shortcut chord: %v, action: %v triggered\n", w.Nm, chord, sa.Text)
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

// PopupIsTooltip returns true if the given popup item is a tooltip
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

// PopupIsCorrector returns true if the given popup item is a menu and a spell corrector
func PopupIsCorrector(pop ki.Ki) bool {
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
	if vp.IsCorrector() {
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

// DeleteTooltip deletes any tooltip popup (called when hover ends)
func (w *Window) DeleteTooltip() {
	w.PopMu.RLock()
	if w.CurPopupIsTooltip() {
		w.delPop = true
	}
	w.PopMu.RUnlock()
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
	// if we have a dialog open, close it if we didn't click in it
	if dlg, ok := pop.(*Dialog); ok {
		log.Println("pos", me.Pos(), "bbox", dlg.WinBBox)
		return !me.Pos().In(dlg.WinBBox)
	}
	if !PopupIsMenu(pop) {
		return false
	}
	if w.NextPopup != nil && PopupIsMenu(w.NextPopup) { // popping up another menu
		return false
	}
	if me.Button != mouse.Left && w.EventMgr.Dragging == nil { // probably menu activation in first place
		return false
	}
	return true
}

// PushPopup pushes current popup onto stack and set new popup.
func (w *Window) PushPopup(pop ki.Ki) {
	w.PopMu.Lock()
	w.ResetUpdateRegions()
	w.NextPopup = nil
	if w.PopupStack == nil {
		w.PopupStack = make([]ki.Ki, 0, 50)
	}
	ki.SetParent(pop, w.This()) // popup has parent as window -- draws directly in to assoc vp
	w.PopupStack = append(w.PopupStack, w.Popup)
	w.Popup = pop
	_, ni := KiToNode2D(pop)
	pfoc := w.PopupFocus
	w.PopupFocus = nil
	w.PopMu.Unlock()
	if ni != nil {
		ni.FullRender2DTree() // this locks viewport -- do it after unlocking popup
	}
	if pfoc != nil {
		w.EventMgr.PushFocus(pfoc)
	} else {
		w.EventMgr.PushFocus(pop)
	}
}

// DisconnectPopup disconnects given popup -- typically the current one.
func (w *Window) DisconnectPopup(pop ki.Ki) {
	w.PopDraws.Delete(pop.(Node))
	w.EventMgr.DisconnectAllEvents(pop, AllPris)
	ki.SetParent(pop, nil) // don't redraw the popup anymore
}

// ClosePopup close given popup -- must be the current one -- returns false if not.
func (w *Window) ClosePopup(pop ki.Ki) bool {
	if pop != w.CurPopup() {
		return false
	}
	w.PopMu.Lock()
	w.ResetUpdateRegions()
	if w.Popup == w.DelPopup {
		w.DelPopup = nil
	}
	w.UpMu.Lock()
	w.DisconnectPopup(pop)
	w.UpMu.Unlock()
	popped := w.PopPopup(pop)
	w.PopMu.Unlock()
	if popped {
		w.EventMgr.PopFocus()
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
	if KeyEventTrace {
		fmt.Printf("Window HiPri KeyInput: %v event: %v\n", w.Path(), e.String())
	}
	if e.IsProcessed() {
		return false
	}
	cs := e.Chord()
	kf := KeyFun(cs)
	cpop := w.CurPopup()
	switch kf {
	case KeyFunWinClose:
		w.CloseReq()
		e.SetProcessed()
	case KeyFunMenu:
		if w.MainMenu != nil {
			w.MainMenu.GrabFocus()
			e.SetProcessed()
		}
	case KeyFunAbort:
		if PopupIsMenu(cpop) || PopupIsTooltip(cpop) {
			delPop = true
			e.SetProcessed()
		} else if w.EventMgr.DNDStage > DNDNotStarted {
			w.ClearDragNDrop()
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
	if e.IsProcessed() {
		return false
	}
	w.EventMgr.ManagerKeyChordEvents(e)
	if e.IsProcessed() {
		return false
	}
	cs := e.Chord()
	kf := KeyFun(cs)
	delPop := false
	switch kf {
	case KeyFunWinSnapshot:
		dstr := time.Now().Format("Mon_Jan_2_15:04:05_MST_2006")
		fnm, _ := filepath.Abs("./GrabOf_" + w.Nm + "_" + dstr + ".png")
		SaveImage(fnm, w.Viewport.Pixels)
		fmt.Printf("Saved Window Image to: %s\n", fnm)
		e.SetProcessed()
	case KeyFunZoomIn:
		w.ZoomDPI(1)
		e.SetProcessed()
	case KeyFunZoomOut:
		w.ZoomDPI(-1)
		e.SetProcessed()
	case KeyFunRefresh:
		e.SetProcessed()
		fmt.Printf("Win: %v display refreshed\n", w.Nm)
		oswin.TheApp.GetScreens()
		Prefs.UpdateAll()
		WinGeomMgr.RestoreAll()
		// w.FocusInactivate()
		// w.FullReRender()
		// sz := w.OSWin.Size()
		// w.SetSize(sz)
	case KeyFunWinFocusNext:
		e.SetProcessed()
		AllWindows.FocusNext()
	}
	switch cs { // some other random special codes, during dev..
	case "Control+Alt+R":
		ProfileToggle()
		e.SetProcessed()
	case "Control+Alt+F":
		w.BenchmarkFullRender()
		e.SetProcessed()
	case "Control+Alt+H":
		w.BenchmarkReRender()
		e.SetProcessed()
	}
	// fmt.Printf("key chord: rune: %v Chord: %v\n", e.Rune, e.Chord())
	return delPop
}

/////////////////////////////////////////////////////////////////////////////
//                   Key Focus

// FocusActiveClick updates the FocusActive status based on mouse clicks in
// or out of the focused item
func (w *Window) FocusActiveClick(e *mouse.Event) {
	cfoc := w.EventMgr.CurFocus()
	if cfoc == nil || e.Button != mouse.Left || e.Action != mouse.Press {
		return
	}
	cpop := w.CurPopup()
	if cpop != nil { // no updating on popups
		return
	}
	nii, ni := KiToNode2D(cfoc)
	if ni != nil && ni.This() != nil {
		if ni.PosInWinBBox(e.Pos()) {
			if !w.HasFlag(int(WinFlagFocusActive)) {
				w.SetFlag(int(WinFlagFocusActive))
				nii.FocusChanged2D(FocusActive)
			}
		} else {
			if w.MainMenu != nil {
				if w.MainMenu.PosInWinBBox(e.Pos()) { // main menu is not inactivating!
					return
				}
			}
			if w.HasFlag(int(WinFlagFocusActive)) {
				w.ClearFlag(int(WinFlagFocusActive))
				nii.FocusChanged2D(FocusInactive)
			}
		}
	}
}

// FocusInactivate inactivates the current focus element
func (w *Window) FocusInactivate() {
	cfoc := w.EventMgr.CurFocus()
	if cfoc == nil || !w.HasFlag(int(WinFlagFocusActive)) {
		return
	}
	nii, ni := KiToNode2D(cfoc)
	if ni != nil && ni.This() != nil {
		w.ClearFlag(int(WinFlagFocusActive))
		nii.FocusChanged2D(FocusInactive)
	}
}

// IsWindowInFocus returns true if this window is the one currently in focus
func (w *Window) IsWindowInFocus() bool {
	fwin := oswin.TheApp.WindowInFocus()
	if w.OSWin == fwin {
		return true
	}
	return false
}

// WindowInFocus returns the window in focus according to oswin.
// There is a small chance it could be nil.
func WindowInFocus() *Window {
	fwin := oswin.TheApp.WindowInFocus()
	fw, _ := AllWindows.FindOSWin(fwin)
	return fw
}

/////////////////////////////////////////////////////////////////////////////
//                   DND: Drag-n-Drop

const DNDSpriteName = "gi.Window:DNDSprite"

// StartDragNDrop is called by a node to start a drag-n-drop operation on
// given source node, which is responsible for providing the data and Sprite
// representation of the node.
func (w *Window) StartDragNDrop(src ki.Ki, data mimedata.Mimes, sp *Sprite) {
	w.EventMgr.DNDStart(src, data)
	if _, sni := KiToNode2D(src); sni != nil { // 2d case
		if sw := sni.AsWidget(); sw != nil {
			sp.SetBottomPos(sw.LayState.Alloc.Pos.ToPoint())
		}
	}
	w.DeleteSprite(DNDSpriteName)
	sp.Name = DNDSpriteName
	sp.On = true
	w.AddSprite(sp)
	w.DNDSetCursor(dnd.DefaultModBits(w.EventMgr.LastModBits))
	w.UpdateSig()
}

// DNDMoveEvent handles drag-n-drop move events.
func (w *Window) DNDMoveEvent(e *mouse.DragEvent) {
	sp, ok := w.SpriteByName(DNDSpriteName)
	if ok {
		sp.SetBottomPos(e.Where)
	}
	de := w.EventMgr.SendDNDMoveEvent(e)
	w.DNDUpdateCursor(de.Mod)
	w.UpdateSig()
	e.SetProcessed()
}

// DNDDropEvent handles drag-n-drop drop event (action = release).
func (w *Window) DNDDropEvent(e *mouse.Event) {
	proc := w.EventMgr.SendDNDDropEvent(e)
	if !proc {
		w.ClearDragNDrop()
	}
}

// FinalizeDragNDrop is called by a node to finalize the drag-n-drop
// operation, after given action has been performed on the target -- allows
// target to cancel, by sending dnd.DropIgnore.
func (w *Window) FinalizeDragNDrop(action dnd.DropMods) {
	if w.EventMgr.DNDStage != DNDDropped {
		w.ClearDragNDrop()
		return
	}
	if w.EventMgr.DNDFinalEvent == nil { // shouldn't happen...
		w.ClearDragNDrop()
		return
	}
	de := w.EventMgr.DNDFinalEvent
	de.ClearProcessed()
	de.Mod = action
	if de.Source != nil {
		de.Action = dnd.DropFmSource
		w.EventMgr.SendSig(de.Source, w, de)
	}
	w.ClearDragNDrop()
}

// ClearDragNDrop clears any existing DND values.
func (w *Window) ClearDragNDrop() {
	w.EventMgr.ClearDND()
	w.DeleteSprite(DNDSpriteName)
	w.DNDClearCursor()
	w.UpdateSig()
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

// DNDUpdateCursor updates the cursor based on the current DND event mod if
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

// ReportWinNodes reports the number of nodes in this window
func (w *Window) ReportWinNodes() {
	nn := 0
	w.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d any) bool {
		nn++
		return ki.Continue
	})
	fmt.Printf("Win: %v has: %v nodes\n", w.Nm, nn)
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
