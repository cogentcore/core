// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"image"
	"unicode/utf8"

	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
)

// Window is a double-buffered OS-specific hardware window.
//
// It provides basic GPU support functions, and is currently implemented
// via Vulkan on top of glfw cross-platform window mgmt toolkit (see driver/vkos).
// using the vgpu framework.
//
// The Window maintains its own vdraw.Drawer drawing system for rendering
// bitmap images and filled regions onto the window surface.
//
// The base full-window image should be drawn with a Scale call first,
// followed by any overlay images such as sprites or direct upload
// images already on the GPU (e.g., from 3D render frames)
//
// vgpu.MaxTexturesPerSet = 16 to work cross-platform, meaning that a maximum of 16
// images per descriptor set can be uploaded and be ready to use in one render pass.
// gi.Window uses multiple sets to get around this limitation.
type Window interface {
	events.Dequer

	// Name returns the name of the window -- name is used strictly for
	// internal tracking and finding of windows -- see Title for the displayed
	// title of the window.
	Name() string

	// SetName sets the name of the window.
	SetName(name string)

	// Title returns the current title of the window, which is displayed in the GUI.
	Title() string

	// SetTitle sets the current title of the window, which is displayed in the GUI.
	SetTitle(title string)

	// Size returns the current size of the window, in raw underlying dots / pixels.
	// This includes any high DPI factors that may not be used in OS window sizing
	//  (see WinSize for that size).
	Size() image.Point

	// WinSize returns the current size of the window, in OS-specific window manager
	// units that may not include any high DPI factors.
	WinSize() image.Point

	// Position returns the current left-top position of the window relative to
	// underlying screen, in OS-specific window manager coordinates.
	Position() image.Point

	// Insets returns the size of any insets on the window in raw device pixels (dots).
	// These insets can be caused by status bars, button overlays, or devices cutouts.
	Insets() styles.SideFloats

	// SetWinSize sets the size of the window, in OS-specific window manager
	// units that may not include any high DPI factors (DevPixRatio)
	// (i.e., the same units as returned in WinSize())
	SetWinSize(sz image.Point)

	// SetSize sets the size of the window, in actual pixel units
	// (i.e., the same units as returned by Size())
	// Divides by DevPixRatio before calling SetWinSize.
	SetSize(sz image.Point)

	// SetPos sets the position of the window, in OS window manager
	// coordinates, which may be different from Size() coordinates
	// that reflect high DPI
	SetPos(pos image.Point)

	// SetGeom sets the position and size in one call -- use this if doing
	// both because sequential calls to SetPos and SetSize might fail on some
	// platforms.  Size is in actual pixel units (i.e., same units as returned by Size()),
	// and Pos is in OS-specific window manager units (i.e., as returned in Pos())
	SetGeom(pos image.Point, sz image.Point)

	// Raise requests that the window be at the top of the stack of windows,
	// and receive focus.  If it is iconified, it will be de-iconified.  This
	// is the only supported mechanism for de-iconifying.
	Raise()

	// Minimize requests that the window be iconified, making it no longer
	// visible or active -- rendering should not occur for minimized windows.
	Minimize()

	// PhysicalDPI is the physical dots per inch of the window, for generating
	// true-to-physical-size output, for example -- see the gi/units package for
	// translating into various other units.
	PhysicalDPI() float32

	// LogicalDPI returns the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units --
	// physical DPI can be found in the Screen.
	LogicalDPI() float32

	// SetLogicalDPI sets the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units --
	// physical DPI can be found in the Screen.
	SetLogicalDPI(dpi float32)

	// Screen returns the screen for this window, with all the relevant
	// information about its properties.
	Screen() *Screen

	// MainMenu returns the OS-level main menu for this window -- this is
	// currently for MacOS only -- returns nil for others.
	MainMenu() MainMenu

	// Flags returns the bit flags for this window's properties set according
	// to WindowFlags bits.
	Flags() WindowFlags

	// Is returns whether the window has the given flag.
	Is(flag WindowFlags) bool

	// IsClosed returns true if this window has been closed (but some threads
	// may have not received the news yet)
	IsClosed() bool

	// IsVisible returns true if this window is not closed or minimized and
	// there are active screens
	IsVisible() bool

	// SetCloseReqFunc sets the function that is called whenever there is a
	// request to close the window (via a OS or a call to CloseReq() method).  That
	// function can then adjudicate whether and when to actually call Close.
	SetCloseReqFunc(fun func(win Window))

	// SetCloseCleanFunc sets the function that is called whenever window is
	// actually about to close (irrevocably) -- can do any necessary
	// last-minute cleanup here.
	SetCloseCleanFunc(fun func(win Window))

	// CloseReq is a close request, triggered either by OS or user call (e.g.,
	// via Close menu action) -- calls function previously-registered by
	// SetCloseReqFunc, which is then solely responsible for actually calling
	// Close.
	CloseReq()

	// CloseClean calls the function setup in SetCloseCleanFunc and does other
	// window cleanup -- called on way to closing.
	CloseClean()

	// Close requests that the window be closed. The behavior of the Window
	// after this, whether calling its methods or passing it as an argument,
	// is undefined.  See App.Quit methods to quit overall app.
	Close()

	// RunOnWin runs given function on the window's own separate goroutine
	RunOnWin(f func())

	// GoRunOnWin runs given function on the window's unique locked thread
	// and returns immediately.
	GoRunOnWin(f func())

	// SendEmptyEvent sends an empty, blank event to this window, which just has
	// the effect of pushing the system along during cases when the window
	// event loop needs to be "pinged" to get things moving along..
	SendEmptyEvent()

	// Handle returns the driver-specific handle for this window.
	// Currently, for all platforms, this is *glfw.Window, but that
	// cannot always be assumed.  Only provided for unforeseen emergency use --
	// please file an Issue for anything that should be added to Window
	// interface.
	Handle() any

	// OSHandle returns the OS-specific underlying window handle:
	// MacOS: NSWindow*, Windows:  HWND, LinuxX11: X11Window
	OSHandle() uintptr

	// Sets the mouse position to given values
	SetMousePos(x, y float64)

	// Sets the mouse cursor to be enabled (true by default) or disabled (false).
	// If disabled, setting raw to true will enable raw mouse movement
	// which can provide better control in a game environment (not avail on Mac).
	SetCursorEnabled(enabled, raw bool)

	// IsCursorEnabled returns true or false if cursor is enabled
	IsCursorEnabled() bool

	// Drawer returns the drawing system attached to this window surface.
	// This is typically used for high-performance rendering to the surface.
	// Wrap access in Lock() / Unlock() calls.
	Drawer() Drawer

	// Lock attempts to grab the overall window Mutex lock
	// and returns true if the window is still open and ready for business.
	// Otherwise, it might have been closed and any further use should be
	// abandoned.  Do this before any use of window resources
	// (eg. rendering to the Drawer).
	Lock() bool

	// Unlock unlocks the Mutex.  Must be called after Lock() is done.
	Unlock()

	// SetDestroyGPUResourcesFunc sets the given function
	// that will be called on the main thread just prior
	// to destroying the drawer and surface.
	SetDestroyGPUResourcesFunc(f func())

	// SetFPS sets the frames per second for Paint events that drive
	// updating of the window at regular intervals.  60 is default.
	// 0 disables -- no regular Paint events are sent.
	// This only takes effect prior to opening a new window.
	SetFPS(fps int)

	// SetTitleBarIsDark sets whether the color theme of the window
	// title bar is dark (as opposed to light).
	SetTitleBarIsDark(isDark bool)

	// EventMgr returns the events.Mgr for this window,
	// which manages all of the Event sending.
	EventMgr() *events.Mgr
}

////////////////////////////////////////////////////////////////////////////
// WindowOptions

// Qt options: http://doc.qt.io/qt-5/qt.html#WindowType-enum

// WindowFlags contains all the binary properties of a window -- by default
// with no other relevant flags a window is a main top-level window.
type WindowFlags int64 //enums:bitflag

const (
	// Dialog indicates that this is a temporary, pop-up window.
	Dialog WindowFlags = iota

	// Modal indicates that this dialog window blocks events going to other
	// windows until it is closed.
	Modal

	// Tool indicates that this is a floating tool window that has minimized
	// window decoration.
	Tool

	// Fullscreen indicates a window that occupies the entire screen.
	Fullscreen

	// Minimized indicates a window reduced to an icon, or otherwise no longer
	// visible or active.  Otherwise, the window should be assumed to be
	// visible.
	Minimized

	// Focused indicates that the window has the focus.
	Focused
)

// NewWindowOptions are optional arguments to NewWindow.
type NewWindowOptions struct {
	// Size specifies the dimensions of the new window, either in raw pixels
	// or std 96 dpi pixels depending on StdPixels. If Width or Height are
	// zero, a driver-dependent default will be used for each zero value
	// dimension
	Size image.Point

	// StdPixels means use standardized "pixel" units for the window size (96
	// per inch), not the actual underlying raw display dot pixels
	StdPixels bool

	// Pos specifies the position of the window, if non-zero -- always in
	// device-specific raw pixels
	Pos image.Point

	// Title specifies the window title.
	Title string

	// Flags can be set using WindowFlags to request different types of windows
	Flags WindowFlags
}

func (o *NewWindowOptions) SetDialog() {
	o.Flags.SetFlag(true, Dialog)
}

func (o *NewWindowOptions) SetModal() {
	o.Flags.SetFlag(true, Modal)
}

func (o *NewWindowOptions) SetTool() {
	o.Flags.SetFlag(true, Tool)
}

func (o *NewWindowOptions) SetFullscreen() {
	o.Flags.SetFlag(true, Fullscreen)
}

func WindowFlagsToBool(flags WindowFlags) (dialog, modal, tool, fullscreen bool) {
	dialog = flags.HasFlag(Dialog)
	modal = flags.HasFlag(Modal)
	tool = flags.HasFlag(Tool)
	fullscreen = flags.HasFlag(Fullscreen)
	return
}

// GetTitle returns a sanitized form of o.Title. In particular, its length will
// not exceed 4096, and it may be further truncated so that it is valid UTF-8
// and will not contain the NUL byte.
//
// o may be nil, in which case "" is returned.
func (o *NewWindowOptions) GetTitle() string {
	if o == nil {
		return ""
	}
	return sanitizeUTF8(o.Title, 4096)
}

func sanitizeUTF8(s string, n int) string {
	if n < len(s) {
		s = s[:n]
	}
	i := 0
	for i < len(s) {
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == 0 || (r == utf8.RuneError && n == 1) {
			break
		}
		i += n
	}
	return s[:i]
}

// Fixup fills in defaults and updates everything based on current screen and
// window context Specific hardware can fine-tune this as well, in driver code
func (o *NewWindowOptions) Fixup() {
	sc := TheApp.Screen(0)
	scsz := sc.Geometry.Size() // window coords size

	if o.Size.X <= 0 {
		o.StdPixels = false
		o.Size.X = int(0.8 * float32(scsz.X) * sc.DevicePixelRatio)
	}
	if o.Size.Y <= 0 {
		o.StdPixels = false
		o.Size.Y = int(0.8 * float32(scsz.Y) * sc.DevicePixelRatio)
	}

	o.Size, o.Pos = sc.ConstrainWinGeom(o.Size, o.Pos)
	if o.Pos.X == 0 && o.Pos.Y == 0 {
		wsz := sc.WinSizeFmPix(o.Size)
		dialog, modal, _, _ := WindowFlagsToBool(o.Flags)
		nw := TheApp.NWindows()
		if nw > 0 {
			lastw := TheApp.Window(nw - 1)
			lsz := lastw.WinSize()
			lp := lastw.Position()

			nwbig := wsz.X > lsz.X || wsz.Y > lsz.Y

			if modal || dialog || !nwbig { // place centered on top of current
				ctrx := lp.X + (lsz.X / 2)
				ctry := lp.Y + (lsz.Y / 2)
				o.Pos.X = ctrx - wsz.X/2
				o.Pos.Y = ctry - wsz.Y/2
			} else { // cascade to right
				o.Pos.X = lp.X + lsz.X // tile to right -- could depend on orientation
				o.Pos.Y = lp.Y + 72    // and move down a bit
			}
		} else { // center in screen
			o.Pos.X = scsz.X/2 - wsz.X/2
			o.Pos.Y = scsz.Y/2 - wsz.Y/2
		}
		o.Size, o.Pos = sc.ConstrainWinGeom(o.Size, o.Pos) // make sure ok
	}
}
